// Command worker menjalankan worker Asynq (§10.2): email transaksional,
// pengecekan kedaluwarsa pesanan, retry webhook, arsip notifikasi, dan
// verifikasi rantai audit harian.
//
// Pada M0 hanya kerangka: server Asynq hidup, satu task uji terdaftar, dan
// penutupan rapi. Handler task sesungguhnya ditambahkan pada M2–M4.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/platform/logger"
)

var (
	version = "dev"
	commit  = "unknown"
)

// TaskNoop adalah task uji untuk memastikan antrean benar-benar berjalan.
// Berguna saat M0: `make worker-ping` mengirim task ini dan menunggu hasilnya.
const TaskNoop = "system:noop"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "kalengo-worker gagal start: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Build.Version = version
	cfg.Build.Commit = commit

	log := logger.New(logger.Options{
		Level:   cfg.App.LogLevel,
		Pretty:  cfg.App.LogPretty,
		Service: "worker",
		Version: version,
	})
	log.Info("konfigurasi worker dimuat", "config", cfg.Redacted())

	redisOpt, err := asynqRedisOpt(cfg.Redis.URL)
	if err != nil {
		return err
	}

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskNoop, handleNoop(log))

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		// Antrean berprioritas: notifikasi & pembayaran lebih dulu (§8.7).
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
		StrictPriority: true,
		// Retry dengan backoff eksponensial; task gagal permanen masuk dead-letter
		// lewat asynq.Server. RetryAfterFunc ditambahkan pada M2.
		ShutdownTimeout: cfg.App.ShutdownIn,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Error("worker Asynq berhenti tak terduga", "error", err)
			stop()
		}
	}()
	log.Info("worker Asynq berjalan", "queues", []string{"critical", "default", "low"})

	<-ctx.Done()
	log.Info("sinyal berhenti diterima, menutup worker dengan rapi")
	srv.Shutdown()
	log.Info("worker berhenti dengan rapi")
	return nil
}

// handleNoop memastikan pipeline antrean sehat.
func handleNoop(log *slog.Logger) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.InfoContext(ctx, "task uji diterima",
			"type", t.Type(),
			"payload", string(t.Payload()),
			"received_at", time.Now().UTC().Format(time.RFC3339),
		)
		return nil
	}
}

// asynqRedisOpt membangun opsi koneksi Asynq dari REDIS_URL yang sama dengan API,
// sehingga hanya ada satu sumber konfigurasi Redis.
func asynqRedisOpt(rawURL string) (asynq.RedisConnOpt, error) {
	parsed, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("worker: REDIS_URL tidak valid: %w", err)
	}
	return asynq.RedisClientOpt{
		Addr:     parsed.Addr,
		Password: parsed.Password,
		DB:       parsed.DB,
		PoolSize: parsed.PoolSize,
	}, nil
}
