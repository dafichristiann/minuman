package repo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kalengo/api/internal/domain/order"
)

// orderColumns dipakai bersama oleh semua SELECT pesanan.
const orderColumns = `id, order_no, user_id, status::text, subtotal, discount,
    shipping_cost, total, voucher_id, address_snapshot, courier, courier_service,
    idempotency_key, request_fingerprint, expires_at, paid_at, shipped_at,
    delivered_at, cancelled_at, cancel_reason, version, created_at, updated_at`

func scanOrder(row pgx.Row) (order.Order, error) {
	var o order.Order
	err := row.Scan(
		&o.ID, &o.OrderNo, &o.UserID, &o.Status, &o.Subtotal, &o.Discount,
		&o.ShippingCost, &o.Total, &o.VoucherID, &o.AddressSnapshot, &o.Courier,
		&o.CourierService, &o.IdempotencyKey, &o.Fingerprint, &o.ExpiresAt,
		&o.PaidAt, &o.ShippedAt, &o.DeliveredAt, &o.CancelledAt, &o.CancelReason,
		&o.Version, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return order.Order{}, mapErr(err)
	}
	return o, nil
}

// ErrInsufficientStock menandakan reservasi gagal karena stok tidak cukup.
//
// Dipisahkan dari error umum karena service harus menerjemahkannya ke
// STOCK_OUT beserta nama produk, bukan ke API_SERVER_ERROR.
type ErrInsufficientStock struct {
	VariantID uuid.UUID
	Requested int32
}

func (e *ErrInsufficientStock) Error() string {
	return fmt.Sprintf("repo: stok varian %s tidak cukup untuk %d unit", e.VariantID, e.Requested)
}

// CreateOrderParams adalah seluruh data yang dibutuhkan untuk menyimpan pesanan.
type CreateOrderParams struct {
	OrderID         uuid.UUID
	OrderNo         string
	UserID          uuid.UUID
	Subtotal        int64
	Discount        int64
	ShippingCost    int64
	Total           int64
	VoucherID       *uuid.UUID
	AddressSnapshot []byte
	Courier         *string
	CourierService  *string
	IdempotencyKey  string
	Fingerprint     string
	ExpiresAt       time.Time
	Items           []order.ItemInput
}

// GenerateOrderNo mengambil nomor urut baru dari sequence dan membentuk nomor
// pesanan `KLG-YYYYMMDD-NNNN`.
//
// Sequence PostgreSQL bebas balapan: dua checkout paralel tidak akan pernah
// memperoleh nomor yang sama, sehingga tidak perlu retry di aplikasi.
func (s *Store) GenerateOrderNo(ctx context.Context, at time.Time) (string, error) {
	var seq int64
	if err := s.db.QueryRow(ctx, `SELECT nextval('order_no_seq')`).Scan(&seq); err != nil {
		return "", err
	}
	return fmt.Sprintf("KLG-%s-%04d", at.UTC().Format("20060102"), seq), nil
}

// GetOrderByIdempotencyKey mencari pesanan yang sudah dibuat dengan kunci sama.
func (s *Store) GetOrderByIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) (order.Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE user_id = $1 AND idempotency_key = $2`, userID, key))
}

// GetOrderForUser mengambil pesanan milik pengguna. Filter `user_id` adalah
// pertahanan IDOR (§13.1): pesanan orang lain tampak "tidak ditemukan".
func (s *Store) GetOrderForUser(ctx context.Context, orderID, userID uuid.UUID) (order.Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE id = $1 AND user_id = $2`, orderID, userID))
}

// GetOrderByID mengambil pesanan tanpa filter pengguna (dipakai webhook).
func (s *Store) GetOrderByID(ctx context.Context, orderID uuid.UUID) (order.Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE id = $1`, orderID))
}

// GetOrderByNo mengambil pesanan berdasarkan nomor (dipakai webhook penyedia
// yang hanya mengenal nomor pesanan).
func (s *Store) GetOrderByNo(ctx context.Context, orderNo string) (order.Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE order_no = $1`, orderNo))
}

// OrderListFilter membatasi daftar pesanan.
type OrderListFilter struct {
	Status *order.Status
	Limit  int32
	// Cursor adalah created_at pesanan terakhir pada halaman sebelumnya.
	Cursor *time.Time
}

// ListOrdersForUser mengembalikan pesanan terbaru lebih dulu.
func (s *Store) ListOrdersForUser(ctx context.Context, userID uuid.UUID, f OrderListFilter) ([]order.Order, error) {
	limit := f.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := s.db.Query(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE user_id = $1
		  AND ($2::text IS NULL OR status = $2::order_status)
		  AND ($3::timestamptz IS NULL OR created_at < $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4`, userID, statusOrNil(f.Status), f.Cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]order.Order, 0, limit)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func statusOrNil(s *order.Status) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// ListOrderItems mengambil seluruh item pesanan, terurut stabil.
func (s *Store) ListOrderItems(ctx context.Context, orderID uuid.UUID) ([]order.Item, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, order_id, variant_id, product_name, variant_label, sku,
		       unit_price, quantity, line_total
		FROM order_items
		WHERE order_id = $1
		ORDER BY product_name, sku`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]order.Item, 0, 4)
	for rows.Next() {
		var it order.Item
		if err := rows.Scan(&it.ID, &it.OrderID, &it.VariantID, &it.ProductName,
			&it.VariantLabel, &it.SKU, &it.UnitPrice, &it.Quantity, &it.LineTotal); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// insertOrder menyimpan pesanan beserta item dan riwayat status awal.
func (s *Store) insertOrder(ctx context.Context, p CreateOrderParams) (order.Order, error) {
	o, err := scanOrder(s.db.QueryRow(ctx, `
		INSERT INTO orders (id, order_no, user_id, status, subtotal, discount,
		                    shipping_cost, total, voucher_id, address_snapshot,
		                    courier, courier_service, idempotency_key,
		                    request_fingerprint, expires_at)
		VALUES ($1, $2, $3, 'pending_payment', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING `+orderColumns,
		p.OrderID, p.OrderNo, p.UserID, p.Subtotal, p.Discount, p.ShippingCost,
		p.Total, p.VoucherID, p.AddressSnapshot, p.Courier, p.CourierService,
		p.IdempotencyKey, p.Fingerprint, p.ExpiresAt,
	))
	if err != nil {
		return order.Order{}, err
	}

	for _, it := range p.Items {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO order_items (order_id, variant_id, product_name, variant_label,
			                         sku, unit_price, quantity, line_total)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			p.OrderID, it.VariantID, it.ProductName, it.VariantLabel, it.SKU,
			it.UnitPrice, it.Quantity, it.LineTotal()); err != nil {
			return order.Order{}, err
		}
	}

	if err := s.insertStatusHistory(ctx, p.OrderID, nil, order.StatusPendingPayment, &p.UserID, nil); err != nil {
		return order.Order{}, err
	}
	return o, nil
}

// CreateOrder membuat pesanan secara atomik: pesanan, item, riwayat status,
// reservasi stok, dan (opsional) pemakaian voucher terjadi dalam satu transaksi.
//
// URUTAN PENTING: baris pesanan disisipkan LEBIH DULU, baru reservasi stok.
// stock_movements.order_id memakai foreign key ke orders (fk_sm_order), sehingga
// mencatat ledgers stok sebelum pesanan ada akan ditolak database. Karena
// semuanya berada dalam satu transaksi, kegagalan stok tetap membatalkan
// pesanan — tidak ada pesanan yatim maupun reservasi yatim.
func (s *Store) CreateOrder(ctx context.Context, p CreateOrderParams) (order.Order, error) {
	var out order.Order
	err := s.InTx(ctx, func(tx *Store) error {
		o, err := tx.insertOrder(ctx, p)
		if err != nil {
			return err
		}
		out = o

		// Urutkan varian agar dua checkout dengan isi berbeda tidak deadlock.
		items := append([]order.ItemInput(nil), p.Items...)
		sort.Slice(items, func(i, j int) bool {
			return items[i].VariantID.String() < items[j].VariantID.String()
		})

		for _, it := range items {
			if err := tx.reserveStock(ctx, it.VariantID, it.Quantity); err != nil {
				return err
			}
			if err := tx.insertStockMovement(ctx, StockMovementParams{
				VariantID: it.VariantID,
				Delta:     it.Quantity,
				Reason:    "reservation",
				OrderID:   &p.OrderID,
				Note:      strPtr("checkout"),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

// reserveStock menaikkan stock_reserved bila masih cukup.
//
// 0 baris berarti stok tidak cukup — bukan error SQL. Kondisi pada WHERE yang
// membuat operasi ini balapan-aman: baris dikunci saat UPDATE, sehingga pesaing
// melihat nilai terbaru.
func (s *Store) reserveStock(ctx context.Context, variantID uuid.UUID, qty int32) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE product_variants
		SET stock_reserved = stock_reserved + $1,
		    version        = version + 1
		WHERE id = $2
		  AND is_active
		  AND stock_on_hand - stock_reserved >= $1`, qty, variantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return &ErrInsufficientStock{VariantID: variantID, Requested: qty}
	}
	return nil
}

// releaseStock mengembalikan reservasi ke stok tersedia.
func (s *Store) releaseStock(ctx context.Context, variantID uuid.UUID, qty int32) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE product_variants
		SET stock_reserved = stock_reserved - $1,
		    version        = version + 1
		WHERE id = $2 AND stock_reserved >= $1`, qty, variantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// commitStockSale mengurangi stok fisik sekaligus melepas reservasi.
//
// Kedua kolom WAJIB berubah dalam satu pernyataan: CHECK (stock_reserved <=
// stock_on_hand) akan menolak bila hanya salah satunya yang berubah.
func (s *Store) commitStockSale(ctx context.Context, variantID uuid.UUID, qty int32) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE product_variants
		SET stock_on_hand  = stock_on_hand - $1,
		    stock_reserved = stock_reserved - $1,
		    version        = version + 1
		WHERE id = $2
		  AND stock_on_hand >= $1
		  AND stock_reserved >= $1`, qty, variantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// StockMovementParams adalah satu baris ledger stok.
type StockMovementParams struct {
	VariantID uuid.UUID
	Delta     int32
	Reason    string
	OrderID   *uuid.UUID
	ActorID   *uuid.UUID
	Note      *string
}

// insertStockMovement mencatat pergerakan stok. Tabel bersifat append-only:
// koreksi dilakukan dengan baris baru, bukan UPDATE.
func (s *Store) insertStockMovement(ctx context.Context, m StockMovementParams) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO stock_movements (variant_id, delta, reason, order_id, actor_id, note)
		VALUES ($1, $2, $3::stock_reason, $4, $5, $6)`,
		m.VariantID, m.Delta, m.Reason, m.OrderID, m.ActorID, m.Note)
	return err
}

// insertStatusHistory mencatat transisi status.
func (s *Store) insertStatusHistory(ctx context.Context, orderID uuid.UUID, from *order.Status, to order.Status, actorID *uuid.UUID, note *string) error {
	var fromRaw *string
	if from != nil {
		v := string(*from)
		fromRaw = &v
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO order_status_history (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, $2::order_status, $3::order_status, $4, $5)`,
		orderID, fromRaw, string(to), actorID, note)
	return err
}

// StatusStamps adalah kolom waktu yang diisi saat status tertentu dicapai.
type StatusStamps struct {
	PaidAt      bool
	ShippedAt   bool
	DeliveredAt bool
	CancelledAt bool
	CancelCause *string
}

// allowedStamp hanya memetakan status ke kolom waktunya; nilai lain tidak diisi.
func allowedStamp(to order.Status, s StatusStamps) StatusStamps {
	out := StatusStamps{}
	switch to {
	case order.StatusPaid:
		out.PaidAt = true
	case order.StatusShipped:
		out.ShippedAt = true
	case order.StatusDelivered, order.StatusCompleted:
		out.DeliveredAt = s.DeliveredAt
	case order.StatusCancelled:
		out.CancelledAt = true
		out.CancelCause = s.CancelCause
	case order.StatusExpired:
		out.CancelledAt = true
	}
	return out
}

// UpdateOrderStatus memindahkan status pesanan secara optimistis.
//
// Klausa `AND status = $from` adalah inti perlindungan balapan: sweeper
// kedaluwarsa dan webhook pembayaran boleh datang bersamaan, tetapi hanya satu
// yang berhasil. false berarti pesanan sudah berpindah oleh pihak lain.
func (s *Store) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, from, to order.Status, actorID *uuid.UUID, note *string, stamps StatusStamps) (bool, error) {
	st := allowedStamp(to, stamps)

	tag, err := s.db.Exec(ctx, `
		UPDATE orders
		SET status        = $3::order_status,
		    paid_at       = CASE WHEN $4 THEN now() ELSE paid_at END,
		    shipped_at    = CASE WHEN $5 THEN now() ELSE shipped_at END,
		    delivered_at  = CASE WHEN $6 THEN now() ELSE delivered_at END,
		    cancelled_at  = CASE WHEN $7 THEN now() ELSE cancelled_at END,
		    cancel_reason = COALESCE($8, cancel_reason),
		    version       = version + 1
		WHERE id = $1 AND status = $2::order_status`,
		orderID, string(from), string(to), st.PaidAt, st.ShippedAt, st.DeliveredAt, st.CancelledAt, st.CancelCause)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	if err := s.insertStatusHistory(ctx, orderID, &from, to, actorID, note); err != nil {
		return false, err
	}
	return true, nil
}

// CancelOrder membatalkan pesanan dan melepas seluruh reservasi stoknya dalam
// satu transaksi.
//
// Bila transisi status gagal (pesanan sudah dibayar/kedaluwarsa), stok TIDAK
// dilepas: mencegah stok hilang dua kali.
func (s *Store) CancelOrder(ctx context.Context, orderID, userID uuid.UUID, from order.Status, to order.Status, note *string) (bool, error) {
	moved := false
	err := s.InTx(ctx, func(tx *Store) error {
		items, err := tx.ListOrderItems(ctx, orderID)
		if err != nil {
			return err
		}

		ok, err := tx.UpdateOrderStatus(ctx, orderID, from, to, &userID, note, StatusStamps{CancelledAt: true, CancelCause: note})
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		moved = true

		sorted := append([]order.Item(nil), items...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].VariantID.String() < sorted[j].VariantID.String()
		})
		for _, it := range sorted {
			if err := tx.releaseStock(ctx, it.VariantID, it.Quantity); err != nil {
				return err
			}
			if err := tx.insertStockMovement(ctx, StockMovementParams{
				VariantID: it.VariantID,
				Delta:     -it.Quantity,
				Reason:    "release",
				OrderID:   &orderID,
				Note:      strPtr("cancelled"),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return moved, err
}

// MarkPaid memindahkan pesanan ke `paid` dan mencatat penjualan stok.
func (s *Store) MarkPaid(ctx context.Context, orderID uuid.UUID, note *string) (bool, error) {
	moved := false
	err := s.InTx(ctx, func(tx *Store) error {
		items, err := tx.ListOrderItems(ctx, orderID)
		if err != nil {
			return err
		}

		ok, err := tx.UpdateOrderStatus(ctx, orderID, order.StatusPendingPayment, order.StatusPaid, nil, note, StatusStamps{PaidAt: true})
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		moved = true

		sorted := append([]order.Item(nil), items...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].VariantID.String() < sorted[j].VariantID.String()
		})
		for _, it := range sorted {
			if err := tx.commitStockSale(ctx, it.VariantID, it.Quantity); err != nil {
				return err
			}
			if err := tx.insertStockMovement(ctx, StockMovementParams{
				VariantID: it.VariantID,
				Delta:     -it.Quantity,
				Reason:    "sale",
				OrderID:   &orderID,
				Note:      strPtr("payment confirmed"),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return moved, err
}

// ListExpiredPendingOrderIDs mengambil pesanan yang reservasinya habis.
//
// FOR UPDATE SKIP LOCKED membuat beberapa worker bisa berjalan bersamaan tanpa
// saling menunggu dan tanpa memproses pesanan yang sama dua kali.
func (s *Store) ListExpiredPendingOrderIDs(ctx context.Context, limit int32) ([]uuid.UUID, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, `
		SELECT id FROM orders
		WHERE status = 'pending_payment' AND expires_at < now()
		ORDER BY expires_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ExpireOrder memindahkan pesanan kedaluwarsa dan melepas stoknya.
func (s *Store) ExpireOrder(ctx context.Context, orderID uuid.UUID) (bool, error) {
	moved := false
	err := s.InTx(ctx, func(tx *Store) error {
		items, err := tx.ListOrderItems(ctx, orderID)
		if err != nil {
			return err
		}

		ok, err := tx.UpdateOrderStatus(ctx, orderID, order.StatusPendingPayment, order.StatusExpired, nil, strPtr("reservation expired"), StatusStamps{})
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		moved = true

		sorted := append([]order.Item(nil), items...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].VariantID.String() < sorted[j].VariantID.String()
		})
		for _, it := range sorted {
			if err := tx.releaseStock(ctx, it.VariantID, it.Quantity); err != nil {
				return err
			}
			if err := tx.insertStockMovement(ctx, StockMovementParams{
				VariantID: it.VariantID,
				Delta:     -it.Quantity,
				Reason:    "release",
				OrderID:   &orderID,
				Note:      strPtr("expired"),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return moved, err
}

// ClearCartForUser mengosongkan keranjang milik pengguna setelah checkout
// berhasil. Berbeda dari ClearCart(cartID) di cart.go yang bekerja per id
// keranjang; versi ini dipanggil checkout yang hanya tahu user.
//
// Berjalan di dalam transaksi pembuatan pesanan, sehingga keranjang hanya
// kosong bila pesanan benar-benar tersimpan.
func (s *Store) ClearCartForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM cart_items
		WHERE cart_id IN (SELECT id FROM carts WHERE user_id = $1)`, userID)
	return err
}

// IsUniqueViolation melaporkan pelanggaran UNIQUE (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsCheckViolation melaporkan pelanggaran CHECK (SQLSTATE 23514).
func IsCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

func strPtr(s string) *string { return &s }
