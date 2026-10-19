// Command worker-ping mengirim satu task uji ke antrean Asynq.
//
// Gunanya pada M0: membuktikan pipeline antrean (Redis → worker) benar-benar
// hidup tanpa perlu menunggu fitur M2. Dipakai oleh `make worker-ping`.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/kalengo/api/internal/config"
)

// taskNoop harus sama dengan TaskNoop di cmd/worker.
const taskNoop = "system:noop"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "worker-ping gagal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	redisOpt, err := asynqRedisOpt(cfg.Redis.URL)
	if err != nil {
		return err
	}

	client := asynq.NewClient(redisOpt)
	defer func() {
		_ = client.Close()
	}()

	payload, err := json.Marshal(map[string]any{
		"pesan":      "ping dari worker-ping",
		"dikirim_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("gagal menyusun payload: %w", err)
	}

	info, err := client.Enqueue(
		asynq.NewTask(taskNoop, payload),
		asynq.Queue("critical"),
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Second),
	)
	if err != nil {
		return fmt.Errorf("gagal memasukkan task: %w", err)
	}

	fmt.Printf("task terkirim: id=%s queue=%s\n", info.ID, info.Queue)
	fmt.Println("cek log worker: docker compose logs -f worker")
	return nil
}

// asynqRedisOpt membangun opsi koneksi Asynq dari REDIS_URL yang sama dengan API.
func asynqRedisOpt(rawURL string) (asynq.RedisConnOpt, error) {
	parsed, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL tidak valid: %w", err)
	}
	return asynq.RedisClientOpt{
		Addr:     parsed.Addr,
		Password: parsed.Password,
		DB:       parsed.DB,
		PoolSize: parsed.PoolSize,
	}, nil
}
