// Package ratelimit mengimplementasikan pembatas laju sliding-window di Redis,
// sesuai §12.8. Dipakai middleware HTTP dan (nanti) pengecekan login/lockout.
//
// Algoritma: sorted set per kunci; anggota lama dibuang berdasarkan skor waktu.
// Atomik lewat satu skrip Lua sehingga aman untuk banyak instance API.
package ratelimit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowScript mengembalikan {allowed, remaining, retryAfterMs}.
const slidingWindowScript = `
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit  = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)

if count < limit then
  redis.call('ZADD', key, now, member)
  redis.call('PEXPIRE', key, window)
  return {1, limit - count - 1, 0}
end

local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
local retry = 0
if oldest[2] ~= nil then
  retry = window - (now - tonumber(oldest[2]))
  if retry < 0 then retry = 0 end
end
return {0, 0, retry}
`

// Result adalah hasil satu pengecekan.
type Result struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
	Limit      int64
}

// Limiter membatasi laju berdasarkan kunci arbitrer (IP, user ID, dll).
type Limiter struct {
	rdb  redis.UniversalClient
	keyp string // prefix kunci, mis. "kalengo:rl"
}

// New membuat Limiter. Prefix membantu pemisahan lingkungan pada satu Redis.
func New(rdb redis.UniversalClient, prefix string) *Limiter {
	if prefix == "" {
		prefix = "kalengo:rl"
	}
	return &Limiter{rdb: rdb, keyp: prefix}
}

// Allow memeriksa apakah aksi untuk `scope` masih diizinkan.
//
// scope contoh: "ip:203.0.113.7", "user:9c1f...", "login:acct:user@mail.com".
func (l *Limiter) Allow(ctx context.Context, scope string, limit int, window time.Duration) (Result, error) {
	if limit <= 0 {
		return Result{Allowed: true, Limit: 0}, nil
	}
	if window <= 0 {
		return Result{}, errors.New("ratelimit: window harus > 0")
	}

	key := fmt.Sprintf("%s:%s:%d", l.keyp, scope, window.Milliseconds())
	member := fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomHex(6))

	res, err := l.rdb.Eval(ctx, slidingWindowScript, []string{key},
		time.Now().UnixMilli(),
		window.Milliseconds(),
		limit,
		member,
	).Slice()
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit: skrip gagal: %w", err)
	}
	if len(res) != 3 {
		return Result{}, fmt.Errorf("ratelimit: respons skrip tidak terduga (len=%d)", len(res))
	}

	allowed, _ := res[0].(int64)
	remaining, _ := res[1].(int64)
	retryMs, _ := res[2].(int64)

	return Result{
		Allowed:    allowed == 1,
		Remaining:  remaining,
		RetryAfter: time.Duration(retryMs) * time.Millisecond,
		Limit:      int64(limit),
	}, nil
}

// Reset menghapus semua penghitung untuk sebuah scope. Dipakai setelah login
// berhasil (agar kegagalan sebelumnya tidak menahan pengguna sah) atau oleh admin.
func (l *Limiter) Reset(ctx context.Context, scope string) error {
	pattern := fmt.Sprintf("%s:%s:*", l.keyp, scope)
	var cursor uint64
	for {
		keys, next, err := l.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("ratelimit: scan gagal: %w", err)
		}
		if len(keys) > 0 {
			if err := l.rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("ratelimit: hapus kunci gagal: %w", err)
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// Tidak fatal: cukup untuk membuat anggota unik.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
