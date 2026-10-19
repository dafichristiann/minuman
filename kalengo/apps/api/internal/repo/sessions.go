package repo

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// RefreshToken adalah satu baris refresh_tokens (termasuk yang sudah dicabut,
// karena deteksi reuse justru butuh melihat baris yang dicabut).
type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	FamilyID   uuid.UUID
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID
}

// CreateRefreshTokenParams adalah input penyimpanan sesi.
type CreateRefreshTokenParams struct {
	UserID    uuid.UUID
	FamilyID  uuid.UUID
	TokenHash []byte
	UserAgent string
	IP        string
	ExpiresAt time.Time
}

// CreateRefreshToken menyimpan hash token. Token asli tidak pernah masuk DB.
func (s *Store) CreateRefreshToken(ctx context.Context, p CreateRefreshTokenParams) (uuid.UUID, error) {
	var id uuid.UUID
	var ip *netip.Addr
	if addr, err := netip.ParseAddr(p.IP); err == nil {
		ip = &addr
	}
	err := s.db.QueryRow(ctx, `
		INSERT INTO refresh_tokens (user_id, family_id, token_hash, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		p.UserID, p.FamilyID, p.TokenHash, nullIfEmpty(p.UserAgent), ip, p.ExpiresAt,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, mapErr(err)
	}
	return id, nil
}

// GetRefreshTokenByHash mencari token berdasarkan hash SHA-256.
//
// Baris yang sudah dicabut TETAP dikembalikan: service membutuhkannya untuk
// membedakan "token tidak dikenal" dari "token dipakai ulang" (§13.1).
func (s *Store) GetRefreshTokenByHash(ctx context.Context, hash []byte) (RefreshToken, error) {
	var t RefreshToken
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, family_id, expires_at, revoked_at, replaced_by
		FROM refresh_tokens
		WHERE token_hash = $1`, hash,
	).Scan(&t.ID, &t.UserID, &t.FamilyID, &t.ExpiresAt, &t.RevokedAt, &t.ReplacedBy)
	if err != nil {
		return RefreshToken{}, mapErr(err)
	}
	return t, nil
}

// RotateRefreshToken mencabut token lama dan menautkannya ke token pengganti.
//
// Mengembalikan false bila token sudah dicabut pihak lain lebih dulu — yaitu
// tanda reuse. Pemanggil WAJIB mencabut seluruh keluarga token dalam kasus itu.
func (s *Store) RotateRefreshToken(ctx context.Context, oldID, newID uuid.UUID) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now(), replaced_by = $2
		WHERE id = $1 AND revoked_at IS NULL`, oldID, newID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// DeleteRefreshToken menghapus token pengganti yang gagal ditautkan (rollback
// lunak setelah deteksi reuse), supaya tidak ada sesi "hantu".
func (s *Store) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE id = $1`, id)
	return err
}

// RevokeFamily mencabut semua token dalam satu keluarga (logout, reuse, ganti
// password).
func (s *Store) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE family_id = $1 AND revoked_at IS NULL`, familyID)
	return err
}

// RevokeAllForUser mencabut seluruh sesi pengguna (logout-all, reset password).
func (s *Store) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

// ActiveRefreshFamilies menghitung sesi aktif (dipakai /me untuk info sesi).
func (s *Store) ActiveRefreshFamilies(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := s.db.QueryRow(ctx, `
		SELECT count(DISTINCT family_id)
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()`, userID).Scan(&n)
	return n, err
}

// Purpose token sekali pakai (§11.2).
const (
	PurposeVerifyEmail   = "verify_email"
	PurposeResetPassword = "reset_password"
)

// CreateOneTimeTokenParams adalah input pembuatan token sekali pakai.
type CreateOneTimeTokenParams struct {
	UserID    uuid.UUID
	Purpose   string
	TokenHash []byte
	ExpiresAt time.Time
}

// CreateOneTimeToken menyimpan token verifikasi/reset. Token lama dengan
// tujuan yang sama dicabut lebih dulu agar hanya tautan terbaru yang berlaku.
func (s *Store) CreateOneTimeToken(ctx context.Context, p CreateOneTimeTokenParams) error {
	if _, err := s.db.Exec(ctx, `
		UPDATE one_time_tokens
		SET used_at = now()
		WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL`, p.UserID, p.Purpose); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO one_time_tokens (user_id, purpose, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`, p.UserID, p.Purpose, p.TokenHash, p.ExpiresAt)
	return err
}

// ConsumeOneTimeToken menandai token terpakai secara atomik dan mengembalikan
// pemiliknya. Satu token hanya bisa dipakai sekali, bahkan saat request paralel.
func (s *Store) ConsumeOneTimeToken(ctx context.Context, purpose string, hash []byte) (uuid.UUID, error) {
	var userID uuid.UUID
	err := s.db.QueryRow(ctx, `
		UPDATE one_time_tokens
		SET used_at = now()
		WHERE token_hash = $1
		  AND purpose = $2
		  AND used_at IS NULL
		  AND expires_at > now()
		RETURNING user_id`, hash, purpose).Scan(&userID)
	if err != nil {
		return uuid.Nil, mapErr(err)
	}
	return userID, nil
}

// DeleteExpiredOneTimeTokens dipanggil job harian (§15).
func (s *Store) DeleteExpiredOneTimeTokens(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM one_time_tokens
		WHERE expires_at < $1 AND (used_at IS NOT NULL OR expires_at < $1)`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
