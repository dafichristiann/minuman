// Package redisclient menyiapkan klien Redis: cache, rate limit, pub/sub SSE,
// dan backend antrean Asynq (§10.2).
package redisclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Options mengatur klien Redis.
type Options struct {
	URL         string
	PoolSize    int
	DialTimeout time.Duration
	// PingAttempts: Redis sering belum siap saat container API start.
	PingAttempts int
	PingBackoff  time.Duration
}

// Client membungkus redis.Client.
type Client struct {
	*redis.Client
}

// New memparse URL (mis. redis://:pass@host:6379/0) lalu memastikan koneksi hidup.
func New(ctx context.Context, opts Options) (*Client, error) {
	if opts.URL == "" {
		return nil, errors.New("redis: URL kosong")
	}
	if opts.PoolSize <= 0 {
		opts.PoolSize = 20
	}
	if opts.DialTimeout <= 0 {
		opts.DialTimeout = 3 * time.Second
	}
	if opts.PingAttempts <= 0 {
		opts.PingAttempts = 10
	}
	if opts.PingBackoff <= 0 {
		opts.PingBackoff = time.Second
	}

	parsed, err := redis.ParseURL(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("redis: URL tidak valid: %w", err)
	}
	parsed.PoolSize = opts.PoolSize
	parsed.DialTimeout = opts.DialTimeout
	parsed.ReadTimeout = 3 * time.Second
	parsed.WriteTimeout = 3 * time.Second
	parsed.ContextTimeoutEnabled = true

	client := redis.NewClient(parsed)
	c := &Client{Client: client}

	var lastErr error
	for attempt := 1; attempt <= opts.PingAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, opts.DialTimeout)
		lastErr = client.Ping(pingCtx).Err()
		cancel()
		if lastErr == nil {
			return c, nil
		}
		if ctx.Err() != nil {
			_ = client.Close()
			return nil, ctx.Err()
		}
		select {
		case <-ctx.Done():
			_ = client.Close()
			return nil, ctx.Err()
		case <-time.After(opts.PingBackoff):
		}
	}
	_ = client.Close()
	return nil, fmt.Errorf("redis: tidak bisa terhubung setelah %d percobaan: %w", opts.PingAttempts, lastErr)
}

// Health dipakai endpoint /readyz.
func (c *Client) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.Ping(ctx).Err()
}
