//go:build integration

package httpintegration

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// checkout membuat pesanan dan mengembalikan hasilnya.
//
// Ongkir dipilih otomatis: layanan termurah, supaya uji tidak bergantung pada
// urutan layanan yang dikembalikan penyedia.
func (h *harness) checkout(t *testing.T, addressID uuid.UUID, idempotencyKey string) createOrderResp {
	t.Helper()

	status, raw := h.checkoutRaw(t, addressID, idempotencyKey)
	if status != http.StatusCreated {
		t.Fatalf("buat pesanan = %d, ingin 201. body=%s", status, raw)
	}
	var out createOrderResp
	decode(t, raw, &out)
	return out
}

// checkoutStatus hanya mengembalikan status HTTP pembuatan pesanan.
func (h *harness) checkoutStatus(t *testing.T, addressID uuid.UUID, idempotencyKey string) int {
	t.Helper()
	status, _ := h.checkoutRaw(t, addressID, idempotencyKey)
	return status
}

// checkoutRaw menjalankan POST /orders dengan ongkir termurah.
func (h *harness) checkoutRaw(t *testing.T, addressID uuid.UUID, idempotencyKey string) (int, []byte) {
	t.Helper()

	status, raw := h.do(t, http.MethodPost, "/api/v1/shipping/rates", map[string]any{
		"address_id": addressID.String(),
	})
	if status != http.StatusOK {
		t.Fatalf("tarif ongkir = %d. body=%s", status, raw)
	}

	var rates struct {
		Items []struct {
			Courier string `json:"courier"`
			Service string `json:"service"`
			Cost    int64  `json:"cost"`
		} `json:"items"`
	}
	decode(t, raw, &rates)
	if len(rates.Items) == 0 {
		t.Fatal("tarif ongkir kosong")
	}

	cheapest := rates.Items[0]
	for _, r := range rates.Items {
		if r.Cost < cheapest.Cost {
			cheapest = r
		}
	}

	return h.doWithHeader(t, http.MethodPost, "/api/v1/orders", map[string]any{
		"address_id":      addressID.String(),
		"courier":         cheapest.Courier,
		"courier_service": cheapest.Service,
	}, map[string]string{"Idempotency-Key": idempotencyKey})
}

// cleanupOrders menghapus pesanan pengguna beserta turunannya.
//
// Urutan penting: baris anak dulu, karena voucher_redemptions dan order_items
// memakai FK. stock_movements memakai ON DELETE SET NULL sehingga aman setelahnya.
func (h *harness) cleanupOrders(t *testing.T, userID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	steps := []struct {
		name string
		sql  string
	}{
		{"voucher_redemptions", `DELETE FROM voucher_redemptions WHERE user_id = $1`},
		{"order_items", `DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1)`},
		{"order_status_history", `DELETE FROM order_status_history WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1)`},
		{"payments", `DELETE FROM payments WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1)`},
		{"shipments", `DELETE FROM shipments WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1)`},
		{"orders", `DELETE FROM orders WHERE user_id = $1`},
	}

	for _, s := range steps {
		if _, err := h.db.Exec(ctx, s.sql, userID); err != nil {
			t.Logf("cleanup %s: %v", s.name, err)
		}
	}

	// Ledger stok bersifat append-only: baris uji dibersihkan lewat koneksi
	// superuser dan setelah pesanan hilang (order_id menjadi NULL).
	if _, err := h.db.Exec(ctx, `
		DELETE FROM stock_movements
		WHERE order_id IS NULL AND note IN ('checkout', 'cancelled', 'expired', 'payment confirmed')
		  AND created_at > now() - interval '1 hour'`); err != nil {
		t.Logf("cleanup stock_movements: %v", err)
	}
}
