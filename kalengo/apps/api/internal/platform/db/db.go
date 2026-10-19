// Package db menyiapkan koneksi PostgreSQL via pgx/v5 (§10.2).
//
// Catatan penting (§13.1): DSN yang dipakai runtime memakai role least-privilege
// (kalengo_app) yang TIDAK punya hak DDL. Migrasi memakai DSN terpisah.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Options mengatur connection pool.
type Options struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
	// AppName dipakai pada application_name agar mudah dilacak di pg_stat_activity.
	AppName string
	// PingAttempts adalah jumlah percobaan saat startup (DB sering belum siap).
	PingAttempts int
	PingBackoff  time.Duration
}

// Pool membungkus pgxpool.Pool dengan pembantu transaksi dan health check.
type Pool struct {
	*pgxpool.Pool
	log *slog.Logger
}

// New membuat pool dan memastikan koneksi benar-benar hidup.
func New(ctx context.Context, opts Options, log *slog.Logger) (*Pool, error) {
	if opts.DSN == "" {
		return nil, errors.New("db: DSN kosong")
	}
	if opts.MaxConns <= 0 {
		opts.MaxConns = 20
	}
	if opts.MinConns < 0 {
		opts.MinConns = 0
	}
	if opts.MinConns > opts.MaxConns {
		opts.MinConns = opts.MaxConns
	}
	if opts.ConnectTimeout <= 0 {
		opts.ConnectTimeout = 5 * time.Second
	}
	if opts.PingAttempts <= 0 {
		opts.PingAttempts = 10
	}
	if opts.PingBackoff <= 0 {
		opts.PingBackoff = time.Second
	}

	poolCfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: DSN tidak valid: %w", err)
	}

	poolCfg.MaxConns = opts.MaxConns
	poolCfg.MinConns = opts.MinConns
	poolCfg.MaxConnLifetime = opts.MaxConnLifetime
	poolCfg.MaxConnIdleTime = opts.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	poolCfg.ConnConfig.RuntimeParams["application_name"] = opts.AppName
	// Semua timestamp UTC (§11.1)
	poolCfg.ConnConfig.RuntimeParams["timezone"] = "UTC"
	// Timezone klien PostgreSQL: pastikan interval/serialisasi waktu konsisten.
	poolCfg.ConnConfig.RuntimeParams["DateStyle"] = "ISO, MDY"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("db: gagal membuat pool: %w", err)
	}

	p := &Pool{Pool: pool, log: log}
	if err := p.waitReady(ctx, opts); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

// waitReady mencoba Ping beberapa kali agar container API tidak gagal hanya
// karena PostgreSQL belum selesai booting.
func (p *Pool) waitReady(ctx context.Context, opts Options) error {
	var lastErr error
	for attempt := 1; attempt <= opts.PingAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, opts.ConnectTimeout)
		lastErr = p.Ping(pingCtx)
		cancel()
		if lastErr == nil {
			if attempt > 1 {
				p.log.Info("koneksi database siap", "attempt", attempt)
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		p.log.Warn("database belum siap, mencoba lagi",
			"attempt", attempt, "of", opts.PingAttempts, "backoff", opts.PingBackoff.String(), "error", lastErr)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.PingBackoff):
		}
	}
	return fmt.Errorf("db: tidak bisa terhubung setelah %d percobaan: %w", opts.PingAttempts, lastErr)
}

// Health dipakai endpoint /readyz (§ observabilitas).
func (p *Pool) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.Ping(ctx)
}

// InTx menjalankan fn di dalam satu transaksi dan menangani commit/rollback.
// Semua mutasi multi-tabel (reservasi stok, pembuatan pesanan) WAJIB lewat sini.
func (p *Pool) InTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return p.inTx(ctx, pgx.TxOptions{}, fn)
}

// InTxSerializable menjalankan fn dengan isolasi SERIALIZABLE untuk kasus
// yang sangat sensitif terhadap race (mis. klaim kuota voucher).
func (p *Pool) InTxSerializable(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return p.inTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable}, fn)
}

func (p *Pool) inTx(ctx context.Context, txOpts pgx.TxOptions, fn func(ctx context.Context, tx pgx.Tx) error) (err error) {
	tx, err := p.BeginTx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("db: gagal membuka transaksi: %w", err)
	}

	defer func() {
		if rec := recover(); rec != nil {
			// Rollback darurat; panic diteruskan ke middleware recover.
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(rec)
		}
		if err != nil {
			if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				p.log.Error("rollback transaksi gagal", "error", rbErr)
			}
			return
		}
		if cErr := tx.Commit(ctx); cErr != nil {
			err = fmt.Errorf("db: commit gagal: %w", cErr)
		}
	}()

	err = fn(ctx, tx)
	return err
}

// Stats melaporkan statistik pool untuk metrik Prometheus.
func (p *Pool) Stats() PoolStats {
	s := p.Pool.Stat()
	return PoolStats{
		TotalConns:           s.TotalConns(),
		IdleConns:            s.IdleConns(),
		AcquiredConns:        s.AcquiredConns(),
		MaxConns:             s.MaxConns(),
		AcquireCount:         s.AcquireCount(),
		EmptyAcquireCount:    s.EmptyAcquireCount(),
		CanceledAcquireCount: s.CanceledAcquireCount(),
	}
}

// PoolStats adalah cuplikan statistik pool.
type PoolStats struct {
	TotalConns           int32
	IdleConns            int32
	AcquiredConns        int32
	MaxConns             int32
	AcquireCount         int64
	EmptyAcquireCount    int64
	CanceledAcquireCount int64
}

// IsUniqueViolation melaporkan apakah error berasal dari constraint unik
// (mis. idempotency key pesanan, dedupe notifikasi).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsCheckViolation melaporkan pelanggaran CHECK constraint
// (mis. stock_reserved > stock_on_hand).
func IsCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

// IsForeignKeyViolation melaporkan pelanggaran foreign key.
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// IsSerializationFailure melaporkan kegagalan serialisasi (perlu retry).
func IsSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "40P01")
}
