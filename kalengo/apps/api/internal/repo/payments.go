package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/domain/order"
)

const paymentColumns = `id, order_id, provider, provider_ref, method, amount,
    status::text, paid_at, raw_response, created_at, updated_at`

func scanPayment(row interface{ Scan(...any) error }) (order.Payment, error) {
	var p order.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Provider, &p.ProviderRef, &p.Method,
		&p.Amount, &p.Status, &p.PaidAt, &p.RawResponse, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return order.Payment{}, mapErr(err)
	}
	return p, nil
}

// CreatePaymentParams adalah input pencatatan percobaan pembayaran.
type CreatePaymentParams struct {
	OrderID     uuid.UUID
	Provider    string
	ProviderRef *string
	Method      *string
	Amount      int64
	Status      order.PaymentStatus
}

// CreatePayment mencatat percobaan pembayaran baru.
//
// Satu pesanan boleh punya beberapa baris pembayaran (pindah metode atau
// percobaan ulang), sehingga pembaca selalu memakai baris terbaru.
func (s *Store) CreatePayment(ctx context.Context, p CreatePaymentParams) (order.Payment, error) {
	status := p.Status
	if status == "" {
		status = order.PaymentPending
	}
	return scanPayment(s.db.QueryRow(ctx, `
		INSERT INTO payments (order_id, provider, provider_ref, method, amount, status)
		VALUES ($1, $2, $3, $4, $5, $6::payment_status)
		RETURNING `+paymentColumns,
		p.OrderID, p.Provider, p.ProviderRef, p.Method, p.Amount, string(status),
	))
}

// GetLatestPaymentForOrder mengambil percobaan pembayaran terakhir.
func (s *Store) GetLatestPaymentForOrder(ctx context.Context, orderID uuid.UUID) (order.Payment, error) {
	return scanPayment(s.db.QueryRow(ctx, `
		SELECT `+paymentColumns+`
		FROM payments
		WHERE order_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1`, orderID))
}

// UpdateLatestPaymentStatus mengubah status seluruh pembayaran tertunda pada
// satu pesanan, sehingga daftar pesanan tidak menampilkan "pending" yang basi.
func (s *Store) UpdateLatestPaymentStatus(ctx context.Context, orderID uuid.UUID, status order.PaymentStatus, raw []byte) error {
	_, err := s.db.Exec(ctx, `
		UPDATE payments
		SET status = $2::payment_status,
		    paid_at = CASE WHEN $2 = 'paid' THEN now() ELSE paid_at END,
		    raw_response = COALESCE($3, raw_response)
		WHERE order_id = $1 AND status = 'pending'`, orderID, string(status), raw)
	return err
}

// RecordWebhookEvent menyimpan event webhook secara idempoten.
//
// Mengembalikan false bila kombinasi (provider, event_id) sudah pernah diterima:
// pemanggil WAJIB berhenti dan membalas sukses tanpa mengubah state apa pun.
func (s *Store) RecordWebhookEvent(ctx context.Context, provider, eventID string, signatureValid bool, payload []byte) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO payment_webhook_events (provider, event_id, signature_valid, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (provider, event_id) DO NOTHING`, provider, eventID, signatureValid, payload)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// MarkWebhookProcessed menandai event selesai diproses, beserta galatnya bila ada.
func (s *Store) MarkWebhookProcessed(ctx context.Context, provider, eventID string, procErr *string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE payment_webhook_events
		SET processed_at = now(), error = $3, attempts = attempts + 1
		WHERE provider = $1 AND event_id = $2`, provider, eventID, procErr)
	return err
}

// UpsertShipment menyimpan status pengiriman untuk sebuah pesanan.
func (s *Store) UpsertShipment(ctx context.Context, orderID uuid.UUID, courier, trackingNo, status string, events []byte) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO shipments (order_id, courier, tracking_no, status, events)
		VALUES ($1, $2, $3, $4, COALESCE($5, '[]'::jsonb))
		ON CONFLICT (order_id) DO UPDATE
		SET courier     = COALESCE(EXCLUDED.courier, shipments.courier),
		    tracking_no = COALESCE(EXCLUDED.tracking_no, shipments.tracking_no),
		    status      = EXCLUDED.status,
		    events      = COALESCE($5, shipments.events)`,
		orderID, courier, trackingNo, status, events)
	return err
}
