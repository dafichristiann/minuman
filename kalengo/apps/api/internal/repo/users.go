package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kalengo/api/internal/domain"
)

// userColumns memakai `role::text` agar enum PostgreSQL dipindai sebagai string
// tanpa bergantung pada registrasi tipe kustom di pgx.
const userColumns = `id, email, password_hash, full_name, phone, role::text,
    email_verified_at, birth_date, failed_login_count, locked_until,
    last_login_at, deleted_at, created_at, updated_at`

func scanUser(row pgx.Row) (domain.User, error) {
	var (
		u  domain.User
		bd pgtype.Date
	)
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role,
		&u.EmailVerifiedAt, &bd, &u.FailedLoginCount, &u.LockedUntil,
		&u.LastLoginAt, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, mapErr(err)
	}
	if bd.Valid {
		t := bd.Time
		u.BirthDate = &t
	}
	return u, nil
}

// CreateUserParams adalah input pendaftaran.
type CreateUserParams struct {
	Email        string
	PasswordHash string
	FullName     string
	Phone        *string
	BirthDate    *time.Time
}

// CreateUser menyisipkan pengguna baru. Pelanggaran UNIQUE(email) diteruskan
// sebagai error pgx; service memetakannya ke AUTH_EMAIL_TAKEN.
func (s *Store) CreateUser(ctx context.Context, p CreateUserParams) (domain.User, error) {
	var bd pgtype.Date
	if p.BirthDate != nil {
		bd = pgtype.Date{Time: *p.BirthDate, Valid: true}
	}
	return scanUser(s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, phone, birth_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userColumns,
		p.Email, p.PasswordHash, p.FullName, p.Phone, bd,
	))
}

// GetUserByID mengambil pengguna aktif (belum dihapus).
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return scanUser(s.db.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`, id))
}

// GetUserByEmail mengambil pengguna aktif berdasarkan email (citext: case-insensitive).
func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(s.db.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`, email))
}

// MarkEmailVerified menandai email terverifikasi. 0 baris berarti sudah
// terverifikasi sebelumnya (idempoten).
func (s *Store) MarkEmailVerified(ctx context.Context, id uuid.UUID) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE users
		SET email_verified_at = now()
		WHERE id = $1 AND email_verified_at IS NULL AND deleted_at IS NULL`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// UpdateUserPassword mengganti hash password dan mereset penghitung lockout.
func (s *Store) UpdateUserPassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $2,
		    failed_login_count = 0,
		    locked_until = NULL
		WHERE id = $1 AND deleted_at IS NULL`, id, hash)
	return err
}

// RecordLoginSuccess mereset penghitung gagal dan mencatat waktu login.
func (s *Store) RecordLoginSuccess(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users
		SET failed_login_count = 0,
		    locked_until = NULL,
		    last_login_at = now()
		WHERE id = $1`, id)
	return err
}

// RecordLoginFailure menambah penghitung gagal dan mengunci akun bila ambang
// terlampaui (§13.1: lockout bertahap).
func (s *Store) RecordLoginFailure(ctx context.Context, id uuid.UUID, threshold int32, lockout time.Duration) (int32, *time.Time, error) {
	var (
		count  int32
		locked *time.Time
	)
	err := s.db.QueryRow(ctx, `
		UPDATE users
		SET failed_login_count = failed_login_count + 1,
		    locked_until = CASE
		        WHEN failed_login_count + 1 >= $2
		            THEN now() + make_interval(secs => $3)
		        ELSE locked_until
		    END
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING failed_login_count, locked_until`,
		id, threshold, lockout.Seconds(),
	).Scan(&count, &locked)
	if err != nil {
		return 0, nil, mapErr(err)
	}
	return count, locked, nil
}

// UpdateUserProfileParams memakai pointer untuk kolom yang boleh dikosongkan.
type UpdateUserProfileParams struct {
	FullName  string
	Phone     *string
	BirthDate *time.Time
}

// UpdateUserProfile menyimpan profil hasil merge di service.
func (s *Store) UpdateUserProfile(ctx context.Context, id uuid.UUID, p UpdateUserProfileParams) (domain.User, error) {
	var bd pgtype.Date
	if p.BirthDate != nil {
		bd = pgtype.Date{Time: *p.BirthDate, Valid: true}
	}
	return scanUser(s.db.QueryRow(ctx, `
		UPDATE users
		SET full_name = $2, phone = $3, birth_date = $4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+userColumns,
		id, p.FullName, p.Phone, bd,
	))
}

// AnonymizeUser menghapus data pribadi tanpa menghapus baris, agar riwayat
// pesanan dan audit tetap utuh (§6.7, UU PDP).
func (s *Store) AnonymizeUser(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE users
		SET email = concat('deleted+', id, '@invalid.kalengo'),
		    password_hash = 'deleted',
		    full_name = 'Pengguna Dihapus',
		    phone = NULL,
		    birth_date = NULL,
		    email_verified_at = NULL,
		    deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
