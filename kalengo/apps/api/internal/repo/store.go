// Package repo adalah satu-satunya lapisan yang boleh menulis SQL (§10.3).
//
// Setiap query memakai placeholder parameter ($1, $2, ...) — tidak ada nilai
// yang disisipkan ke string SQL (§13.1). Fragmen statis (ORDER BY) dipilih dari
// daftar putih di kode, tidak pernah dari input pengguna.
//
// CATATAN M1 (lihat docs/DEVIATIONS.md): repo ditulis langsung dengan pgx/v5
// alih-alih kode hasil `sqlc generate`, karena toolchain sqlc belum tersedia saat
// M1 disusun. Berkas db/queries/*.sql tetap menjadi spesifikasi SQL kanonik dan
// akan menggantikan konstanta di sini pada M1.5. Semua query sudah
// parameterized, sehingga syarat §13.1 terpenuhi.
package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Errors yang dipetakan ke kode apperr oleh lapisan service.
var (
	ErrNotFound = errors.New("repo: data tidak ditemukan")
	ErrConflict = errors.New("repo: konflik data")
)

// DBTX adalah irisan *pgxpool.Pool dan pgx.Tx sehingga query yang sama bisa
// dijalankan di dalam maupun di luar transaksi.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store membungkus DBTX. Nilainya murah; buat instance baru per transaksi.
type Store struct {
	db DBTX
}

// New membuat Store.
func New(db DBTX) *Store { return &Store{db: db} }

// DB mengembalikan koneksi dasar (dipakai untuk transaksi oleh service).
func (s *Store) DB() DBTX { return s.db }

// mapErr menerjemahkan "tidak ada baris" ke sentinel domain.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
