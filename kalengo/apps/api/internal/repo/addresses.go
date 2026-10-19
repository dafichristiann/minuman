package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/domain"
)

const addressColumns = `id, user_id, label, recipient, phone, province, city,
    district, postal_code, street, notes, area_id, is_default, created_at, updated_at`

func scanAddress(rows interface{ Scan(...any) error }) (domain.Address, error) {
	var a domain.Address
	err := rows.Scan(
		&a.ID, &a.UserID, &a.Label, &a.Recipient, &a.Phone, &a.Province, &a.City,
		&a.District, &a.PostalCode, &a.Street, &a.Notes, &a.AreaID, &a.IsDefault,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return domain.Address{}, mapErr(err)
	}
	return a, nil
}

// ListAddresses mengembalikan alamat aktif milik pengguna, default lebih dulu.
// Filter `user_id = $1` adalah pertahanan IDOR/BOLA (§13.1).
func (s *Store) ListAddresses(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+addressColumns+`
		FROM addresses
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY is_default DESC, created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Address, 0, 4)
	for rows.Next() {
		a, err := scanAddress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAddress mengambil satu alamat milik pengguna.
func (s *Store) GetAddress(ctx context.Context, userID, id uuid.UUID) (domain.Address, error) {
	return scanAddress(s.db.QueryRow(ctx, `
		SELECT `+addressColumns+`
		FROM addresses
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, id, userID))
}

// CountAddresses menghitung alamat aktif.
func (s *Store) CountAddresses(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM addresses
		WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&n)
	return n, err
}

// AddressInput adalah payload buat/ubah alamat (sudah divalidasi service).
type AddressInput struct {
	Label      string
	Recipient  string
	Phone      string
	Province   string
	City       string
	District   string
	PostalCode string
	Street     string
	Notes      *string
	AreaID     *string
	IsDefault  bool
}

// CreateAddress menyisipkan alamat. Alamat pertama otomatis menjadi default.
func (s *Store) CreateAddress(ctx context.Context, userID uuid.UUID, in AddressInput) (domain.Address, error) {
	var makeDefault bool
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) = 0 FROM addresses
		WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&makeDefault); err != nil {
		return domain.Address{}, err
	}
	makeDefault = makeDefault || in.IsDefault

	if makeDefault {
		if err := s.clearDefault(ctx, userID); err != nil {
			return domain.Address{}, err
		}
	}

	return scanAddress(s.db.QueryRow(ctx, `
		INSERT INTO addresses (user_id, label, recipient, phone, province, city,
		                      district, postal_code, street, notes, area_id, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING `+addressColumns,
		userID, in.Label, in.Recipient, in.Phone, in.Province, in.City,
		in.District, in.PostalCode, in.Street, in.Notes, in.AreaID, makeDefault,
	))
}

// UpdateAddress menimpa seluruh kolom alamat. Kepemilikan diverifikasi di WHERE.
func (s *Store) UpdateAddress(ctx context.Context, userID, id uuid.UUID, in AddressInput) (domain.Address, error) {
	if in.IsDefault {
		if err := s.clearDefault(ctx, userID); err != nil {
			return domain.Address{}, err
		}
	}
	return scanAddress(s.db.QueryRow(ctx, `
		UPDATE addresses
		SET label = $3, recipient = $4, phone = $5, province = $6, city = $7,
		    district = $8, postal_code = $9, street = $10, notes = $11, area_id = $12,
		    is_default = CASE WHEN $13 THEN true ELSE is_default END
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING `+addressColumns,
		id, userID, in.Label, in.Recipient, in.Phone, in.Province, in.City,
		in.District, in.PostalCode, in.Street, in.Notes, in.AreaID, in.IsDefault,
	))
}

// SetDefaultAddress menjadikan satu alamat sebagai default milik pengguna.
func (s *Store) SetDefaultAddress(ctx context.Context, userID, id uuid.UUID) (domain.Address, error) {
	// Pastikan alamat memang milik pengguna sebelum mengubah apa pun.
	if _, err := s.GetAddress(ctx, userID, id); err != nil {
		return domain.Address{}, err
	}
	if err := s.clearDefault(ctx, userID); err != nil {
		return domain.Address{}, err
	}
	return scanAddress(s.db.QueryRow(ctx, `
		UPDATE addresses
		SET is_default = true
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING `+addressColumns, id, userID))
}

// DeleteAddress melakukan soft delete. Bila alamat yang dihapus adalah default,
// alamat terlama berikutnya dinaikkan menjadi default agar checkout tetap bisa
// berjalan tanpa langkah tambahan dari pengguna.
func (s *Store) DeleteAddress(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE addresses
		SET deleted_at = now(), is_default = false
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Bila tidak ada default tersisa, promosikan yang terlama.
	_, err = s.db.Exec(ctx, `
		UPDATE addresses
		SET is_default = true
		WHERE id = (
		    SELECT id FROM addresses
		    WHERE user_id = $1 AND deleted_at IS NULL
		    ORDER BY created_at ASC
		    LIMIT 1
		)
		AND NOT EXISTS (
		    SELECT 1 FROM addresses
		    WHERE user_id = $1 AND deleted_at IS NULL AND is_default
		)`, userID)
	return err
}

func (s *Store) clearDefault(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE addresses
		SET is_default = false
		WHERE user_id = $1 AND is_default AND deleted_at IS NULL`, userID)
	return err
}
