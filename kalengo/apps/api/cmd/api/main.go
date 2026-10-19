// Command api adalah biner HTTP KALENGO (§10.2, §10.3).
//
// Tanggung jawab M0: memuat konfigurasi, menyiapkan logger, database, Redis,
// rate limiter, lalu menjalankan server HTTP dengan graceful shutdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kalengo/api/internal/config"
	apiserver "github.com/kalengo/api/internal/http"
	"github.com/kalengo/api/internal/platform/db"
	"github.com/kalengo/api/internal/platform/logger"
	"github.com/kalengo/api/internal/platform/ratelimit"
	"github.com/kalengo/api/internal/platform/redisclient"
)

// Diisi lewat -ldflags pada saat build (lihat apps/api/Dockerfile).
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if err := run(); err != nil {
		// Logger belum tentu siap saat konfigurasi gagal; tulis ke stderr.
		fmt.Fprintf(os.Stderr, "kalengo-api gagal start: %v\n", err)
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
		Service: "api",
		Version: version,
	})
	log.Info("konfigurasi dimuat", "config", cfg.Redacted())

	// Root context: dibatalkan saat SIGINT/SIGTERM diterima.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.New(ctx, db.Options{
		DSN:             cfg.DB.URL,
		MaxConns:        cfg.DB.MaxConns,
		MinConns:        cfg.DB.MinConns,
		MaxConnLifetime: cfg.DB.MaxConnLifetime,
		MaxConnIdleTime: cfg.DB.MaxConnIdleTime,
		ConnectTimeout:  cfg.DB.ConnectTimeout,
		AppName:         cfg.App.Name + "-api",
		PingAttempts:    15,
		PingBackoff:     2 * time.Second,
	}, log)
	if err != nil {
		return err
	}
	defer pool.Close()

	rdb, err := redisclient.New(ctx, redisclient.Options{
		URL:          cfg.Redis.URL,
		PoolSize:     cfg.Redis.PoolSize,
		DialTimeout:  cfg.Redis.DialTimeout,
		PingAttempts: 15,
		PingBackoff:  2 * time.Second,
	})
	if err != nil {
		return err
	}
	defer func() {
		if cerr := rdb.Close(); cerr != nil {
			log.Warn("gagal menutup koneksi Redis", "error", cerr)
		}
	}()

	limiter := ratelimit.New(rdb.Client, cfg.App.Name+":rl")

	router := apiserver.NewRouter(apiserver.Deps{
		Config:  cfg,
		Log:     log,
		DB:      pool,
		Redis:   rdb,
		Limiter: limiter,
	})

	srv := &nethttp.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		// WriteTimeout berlaku juga untuk SSE. Handler SSE (§8.7) WAJIB memakai
		// http.NewResponseController(w).SetWriteDeadline(time.Time{}) agar
		// streaming panjang tidak diputus.
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
		ErrorLog:     nil,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server HTTP mendengarkan", "addr", cfg.HTTP.Addr, "env", string(cfg.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server HTTP berhenti tak terduga: %w", err)
	case <-ctx.Done():
		log.Info("sinyal berhenti diterima, menutup server dengan rapi",
			"timeout", cfg.App.ShutdownIn.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownIn)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("gagal menutup server dengan rapi, memaksa berhenti", "error", err)
		return err
	}
	log.Info("server berhenti dengan rapi")
	return nil
}
