package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/domain/order"
)

const voucherColumns = `id, code, type::text, value, max_discount, min_spend,
    usage_limit, usage_count, per_user_limit, starts_at, ends_at, is_active`

func scanVoucher(row interface{ Scan(...any) error }) (order.Voucher, error) {
	var v order.Voucher
	err := row.Scan(&v.ID, &v.Code, &v.Type, &v.Value, &v.MaxDiscount, &v.MinSpend,
		&v.UsageLimit, &v.UsageCount, &v.PerUserMax, &v.StartsAt, &v.EndsAt, &v.IsActive)
	if err != nil {
		return order.Voucher{}, mapErr(err)
	}
	return v, nil
}

// GetVoucherByCode mencari voucher. `code` bertipe citext sehingga pencarian
// otomatis tidak membedakan huruf besar/kecil.
func (s *Store) GetVoucherByCode(ctx context.Context, code string) (order.Voucher, error) {
	return scanVoucher(s.db.QueryRow(ctx, `
		SELECT `+voucherColumns+`
		FROM vouchers
		WHERE code = $1`, code))
}

// ClaimVoucherUsage menambah usage_count bila kuota masih tersedia.
//
// Kondisi `usage_count < usage_limit` dievaluasi di bawah kunci baris, sehingga
// dua checkout paralel pada slot terakhir tidak mungkin sama-sama berhasil
// (§13.1 kuota atomik). false berarti kuota habis atau voucher tidak berlaku.
func (s *Store) ClaimVoucherUsage(ctx context.Context, voucherID uuid.UUID) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE vouchers
		SET usage_count = usage_count + 1
		WHERE id = $1
		  AND is_active
		  AND now() BETWEEN starts_at AND ends_at
		  AND (usage_limit IS NULL OR usage_count < usage_limit)`, voucherID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// CountVoucherRedemptions menghitung pemakaian voucher oleh satu pengguna.
//
// per_user_limit tidak bisa ditegakkan oleh UNIQUE(order_id) saja, jadi
// penghitungan ini yang menjadi gerbangnya. Dipanggil di dalam transaksi yang
// sama dengan ClaimVoucherUsage.
func (s *Store) CountVoucherRedemptions(ctx context.Context, voucherID, userID uuid.UUID) (int64, error) {
	var n int64
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM voucher_redemptions
		WHERE voucher_id = $1 AND user_id = $2`, voucherID, userID).Scan(&n)
	return n, err
}

// InsertVoucherRedemption mencatat pemakaian voucher pada pesanan.
func (s *Store) InsertVoucherRedemption(ctx context.Context, voucherID, userID, orderID uuid.UUID, amount int64) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO voucher_redemptions (voucher_id, user_id, order_id, amount)
		VALUES ($1, $2, $3, $4)`, voucherID, userID, orderID, amount)
	return err
}
