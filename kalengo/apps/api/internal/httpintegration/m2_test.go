//go:build integration

package httpintegration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// perkakas M2
// ---------------------------------------------------------------------------

// seedVariant mengambil satu varian berstok dari seed.
func (h *harness) seedVariant(t *testing.T, minStock int) (variantID uuid.UUID, price int64, name string) {
	t.Helper()
	err := h.db.QueryRow(context.Background(), `
		SELECT v.id, v.price, p.name
		FROM product_variants v
		JOIN products p ON p.id = v.product_id
		WHERE v.is_active AND p.is_active AND p.deleted_at IS NULL
		  AND v.stock_on_hand - v.stock_reserved >= $1
		ORDER BY v.stock_on_hand DESC
		LIMIT 1`, minStock).Scan(&variantID, &price, &name)
	if err != nil {
		t.Fatalf("ambil varian seed: %v", err)
	}
	return variantID, price, name
}

// addToCart menaruh satu varian di keranjang pengguna yang sedang login.
func (h *harness) addToCart(t *testing.T, variantID uuid.UUID, qty int) {
	t.Helper()
	status, raw := h.do(t, http.MethodPut, "/api/v1/cart/items/"+variantID.String(), map[string]any{
		"quantity": qty,
	})
	if status != http.StatusOK {
		t.Fatalf("tambah keranjang = %d. body=%s", status, raw)
	}
}

// createAddress membuat alamat pengiriman dan mengembalikan id-nya.
func (h *harness) createAddress(t *testing.T) uuid.UUID {
	t.Helper()
	status, raw := h.do(t, http.MethodPost, "/api/v1/me/addresses", map[string]any{
		"label":       "Rumah",
		"recipient":   "Budi Uji",
		"phone":       "081234567890",
		"province":    "DKI Jakarta",
		"city":        "Jakarta Selatan",
		"district":    "Kebayoran Baru",
		"postal_code": "12110",
		"street":      "Jl. Uji Integrasi No. 1",
		"area_id":     "ID-JKT-01",
		"is_default":  true,
	})
	if status != http.StatusCreated {
		t.Fatalf("buat alamat = %d. body=%s", status, raw)
	}
	var addr struct {
		ID string `json:"id"`
	}
	decode(t, raw, &addr)
	id, err := uuid.Parse(addr.ID)
	if err != nil {
		t.Fatalf("alamat id tidak valid: %v", err)
	}
	return id
}

// registerAndLogin membuat pengguna baru yang sudah login.
func (h *harness) registerAndLogin(t *testing.T, password string) (email string, userID uuid.UUID) {
	t.Helper()
	email = uniqueEmail()

	status, raw := h.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":     email,
		"password":  password,
		"full_name": "Budi Checkout",
	})
	if status != http.StatusCreated {
		t.Fatalf("register = %d. body=%s", status, raw)
	}
	var reg struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	decode(t, raw, &reg)
	userID, err := uuid.Parse(reg.User.ID)
	if err != nil {
		t.Fatalf("user id tidak valid: %v", err)
	}

	status, raw = h.do(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("login = %d. body=%s", status, raw)
	}
	return email, userID
}

// stockOf membaca pasangan stok dari database.
func (h *harness) stockOf(t *testing.T, variantID uuid.UUID) (onHand, reserved int32) {
	t.Helper()
	if err := h.db.QueryRow(context.Background(),
		`SELECT stock_on_hand, stock_reserved FROM product_variants WHERE id = $1`, variantID,
	).Scan(&onHand, &reserved); err != nil {
		t.Fatalf("baca stok: %v", err)
	}
	return onHand, reserved
}

type createOrderResp struct {
	ID       string `json:"id"`
	OrderNo  string `json:"order_no"`
	Status   string `json:"status"`
	Subtotal int64  `json:"subtotal"`
	Discount int64  `json:"discount"`
	Shipping int64  `json:"shipping_cost"`
	Total    int64  `json:"total"`
	Items    []struct {
		VariantID string `json:"variant_id"`
		Quantity  int32  `json:"quantity"`
		UnitPrice int64  `json:"unit_price"`
	} `json:"items"`
}

// postRaw mengirim body mentah dengan header tambahan (dipakai webhook).
func (h *harness) postRaw(t *testing.T, path string, body []byte, headers map[string]string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, h.base+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

// ---------------------------------------------------------------------------
// uji
// ---------------------------------------------------------------------------

// TestM2CheckoutCreatesOrderAndReservesStock menguji alur checkout lengkap.
func TestM2CheckoutCreatesOrderAndReservesStock(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"
	email, userID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, email)
	defer h.cleanupOrders(t, userID)

	addrID := h.createAddress(t)
	variantID, price, _ := h.seedVariant(t, 10)
	onHandBefore, reservedBefore := h.stockOf(t, variantID)

	h.addToCart(t, variantID, 2)

	status, raw := h.do(t, http.MethodPost, "/api/v1/shipping/rates", map[string]any{
		"address_id": addrID.String(),
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
	chosen := rates.Items[0]

	status, raw = h.doWithHeader(t, http.MethodPost, "/api/v1/orders", map[string]any{
		"address_id":      addrID.String(),
		"courier":         chosen.Courier,
		"courier_service": chosen.Service,
	}, map[string]string{"Idempotency-Key": "checkout-" + uuid.NewString()})
	if status != http.StatusCreated {
		t.Fatalf("buat pesanan = %d, ingin 201. body=%s", status, raw)
	}

	var created createOrderResp
	decode(t, raw, &created)

	// Uang dihitung server dan konsisten.
	if created.Subtotal != price*2 {
		t.Fatalf("subtotal = %d, ingin %d", created.Subtotal, price*2)
	}
	if created.Total != created.Subtotal-created.Discount+created.Shipping {
		t.Fatalf("total %d tidak konsisten dengan subtotal/discount/ongkir", created.Total)
	}
	if created.Status != "pending_payment" {
		t.Fatalf("status awal = %q, ingin pending_payment", created.Status)
	}
	if created.OrderNo == "" {
		t.Fatal("order_no kosong")
	}

	// Stok: reservasi bertambah, stok fisik TIDAK berubah.
	onHandAfter, reservedAfter := h.stockOf(t, variantID)
	if onHandAfter != onHandBefore {
		t.Fatalf("stock_on_hand berubah saat reservasi: %d → %d", onHandBefore, onHandAfter)
	}
	if reservedAfter != reservedBefore+2 {
		t.Fatalf("stock_reserved = %d, ingin %d", reservedAfter, reservedBefore+2)
	}

	// Ledger mencatat reservasi.
	var moved int
	if err := h.db.QueryRow(context.Background(), `
		SELECT count(*) FROM stock_movements
		WHERE order_id = $1 AND reason = 'reservation' AND delta = 2`,
		created.ID).Scan(&moved); err != nil {
		t.Fatalf("baca ledger: %v", err)
	}
	if moved != 1 {
		t.Fatalf("baris stock_movements reservation = %d, ingin 1", moved)
	}

	// Keranjang dikosongkan setelah checkout.
	status, raw = h.do(t, http.MethodGet, "/api/v1/cart", nil)
	if status != http.StatusOK {
		t.Fatalf("baca keranjang = %d", status)
	}
	var cart struct {
		Items []json.RawMessage `json:"items"`
	}
	decode(t, raw, &cart)
	if len(cart.Items) != 0 {
		t.Fatalf("keranjang tidak kosong setelah checkout: %d item", len(cart.Items))
	}
}

// TestM2IdempotencyOneOrder membuat pesanan dari kunci sama berkali-kali.
func TestM2IdempotencyOneOrder(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"
	email, userID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, email)
	defer h.cleanupOrders(t, userID)

	addrID := h.createAddress(t)
	variantID, _, _ := h.seedVariant(t, 20)
	h.addToCart(t, variantID, 1)

	key := "idem-" + uuid.NewString()
	body := map[string]any{"address_id": addrID.String()}

	const parallel = 8
	var wg sync.WaitGroup
	statuses := make([]int, parallel)

	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			st, _ := h.doWithHeader(t, http.MethodPost, "/api/v1/orders", body,
				map[string]string{"Idempotency-Key": key})
			statuses[idx] = st
		}(i)
	}
	wg.Wait()

	for i, st := range statuses {
		if st != http.StatusCreated && st != http.StatusOK {
			t.Fatalf("kiriman %d = %d, ingin 201 atau 200", i, st)
		}
	}

	var count int
	if err := h.db.QueryRow(context.Background(),
		`SELECT count(*) FROM orders WHERE user_id = $1 AND idempotency_key = $2`,
		userID, key).Scan(&count); err != nil {
		t.Fatalf("hitung pesanan: %v", err)
	}
	if count != 1 {
		t.Fatalf("pesanan dengan kunci sama = %d, ingin tepat 1", count)
	}
}

// TestM2PaymentWebhookMarksPaid menguji webhook tanda tangan sah.
func TestM2PaymentWebhookMarksPaid(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"
	email, userID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, email)
	defer h.cleanupOrders(t, userID)

	addrID := h.createAddress(t)
	variantID, _, _ := h.seedVariant(t, 10)
	onHandBefore, reservedBefore := h.stockOf(t, variantID)
	h.addToCart(t, variantID, 3)

	created := h.checkout(t, addrID, "checkout-"+uuid.NewString())

	// Sesi pembayaran.
	status, raw := h.do(t, http.MethodPost, "/api/v1/orders/"+created.ID+"/pay", nil)
	if status != http.StatusOK {
		t.Fatalf("init pembayaran = %d. body=%s", status, raw)
	}

	// Webhook tiruan bertanda tangan sah.
	payload := h.payment.BuildWebhook(created.OrderNo, "settlement", fmt.Sprintf("%d.00", created.Total))
	status, raw = h.postRaw(t, "/api/v1/webhooks/midtrans", payload, nil)
	if status != http.StatusOK {
		t.Fatalf("webhook = %d. body=%s", status, raw)
	}
	var wh struct {
		Applied bool `json:"applied"`
	}
	decode(t, raw, &wh)
	if !wh.Applied {
		t.Fatalf("webhook tidak mengubah pesanan: %s", raw)
	}

	// Pesanan lunas.
	status, raw = h.do(t, http.MethodGet, "/api/v1/orders/"+created.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("detail pesanan = %d. body=%s", status, raw)
	}
	var after createOrderResp
	decode(t, raw, &after)
	if after.Status != "paid" {
		t.Fatalf("status = %q, ingin paid", after.Status)
	}

	// Stok fisik berkurang 3 dan reservasi dilepas sebesar 3.
	//
	// Assert dibuat RELATIF terhadap nilai sebelum checkout: varian seed dipakai
	// bersama oleh beberapa uji, sehingga nilai absolut akan rapuh.
	onHandAfter, reservedAfter := h.stockOf(t, variantID)
	if onHandAfter != onHandBefore-3 {
		t.Fatalf("stock_on_hand = %d, ingin %d", onHandAfter, onHandBefore-3)
	}
	if reservedAfter != reservedBefore {
		t.Fatalf("stock_reserved = %d, ingin kembali ke %d setelah lunas", reservedAfter, reservedBefore)
	}

	// Replay webhook yang sama tidak mengubah apa pun.
	status, raw = h.postRaw(t, "/api/v1/webhooks/midtrans", payload, nil)
	if status != http.StatusOK {
		t.Fatalf("replay webhook = %d. body=%s", status, raw)
	}
	var dup struct {
		Duplicate bool `json:"duplicate"`
	}
	decode(t, raw, &dup)
	if !dup.Duplicate {
		t.Fatal("replay webhook tidak dikenali sebagai duplikat")
	}
	onHandAgain, _ := h.stockOf(t, variantID)
	if onHandAgain != onHandAfter {
		t.Fatalf("stok berubah pada replay: %d → %d", onHandAfter, onHandAgain)
	}
}

// TestM2PaymentWebhookRejectsBadSignature memastikan payload palsu ditolak.
func TestM2PaymentWebhookRejectsBadSignature(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"
	email, userID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, email)
	defer h.cleanupOrders(t, userID)

	addrID := h.createAddress(t)
	variantID, _, _ := h.seedVariant(t, 10)
	h.addToCart(t, variantID, 1)
	created := h.checkout(t, addrID, "sig-"+uuid.NewString())

	bad, _ := json.Marshal(map[string]any{
		"order_id":           created.OrderNo,
		"status_code":        "200",
		"gross_amount":       fmt.Sprintf("%d.00", created.Total),
		"transaction_status": "settlement",
		"fraud_status":       "accept",
		"signature_key":      "tanda-tangan-palsu",
	})

	status, raw := h.postRaw(t, "/api/v1/webhooks/midtrans", bad, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("signature palsu = %d, ingin 401. body=%s", status, raw)
	}

	// Pesanan tidak boleh berubah.
	status, raw = h.do(t, http.MethodGet, "/api/v1/orders/"+created.ID, nil)
	var after createOrderResp
	decode(t, raw, &after)
	if after.Status != "pending_payment" {
		t.Fatalf("status berubah oleh payload palsu: %q", after.Status)
	}
}

// TestM2PaymentWebhookRejectsWrongAmount memastikan nominal salah ditolak.
func TestM2PaymentWebhookRejectsWrongAmount(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"
	email, userID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, email)
	defer h.cleanupOrders(t, userID)

	addrID := h.createAddress(t)
	variantID, _, _ := h.seedVariant(t, 10)
	h.addToCart(t, variantID, 1)
	created := h.checkout(t, addrID, "amt-"+uuid.NewString())

	// Tanda tangan sah, tetapi nominal lebih kecil dari total pesanan.
	payload := h.payment.BuildWebhook(created.OrderNo, "settlement", "1.00")
	status, raw := h.postRaw(t, "/api/v1/webhooks/midtrans", payload, nil)
	if status == http.StatusOK {
		t.Fatalf("nominal salah diterima: %s", raw)
	}

	status, raw = h.do(t, http.MethodGet, "/api/v1/orders/"+created.ID, nil)
	var after createOrderResp
	decode(t, raw, &after)
	if after.Status != "pending_payment" {
		t.Fatalf("status berubah oleh nominal salah: %q", after.Status)
	}
}

// TestM2CancelReleasesReservation memastikan pembatalan mengembalikan stok.
func TestM2CancelReleasesReservation(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"
	email, userID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, email)
	defer h.cleanupOrders(t, userID)

	addrID := h.createAddress(t)
	variantID, _, _ := h.seedVariant(t, 10)
	_, reservedBefore := h.stockOf(t, variantID)
	h.addToCart(t, variantID, 2)
	created := h.checkout(t, addrID, "cancel-"+uuid.NewString())

	status, raw := h.do(t, http.MethodPost, "/api/v1/orders/"+created.ID+"/cancel", map[string]any{
		"reason": "berubah pikiran",
	})
	if status != http.StatusOK {
		t.Fatalf("batal = %d. body=%s", status, raw)
	}
	var after createOrderResp
	decode(t, raw, &after)
	if after.Status != "cancelled" {
		t.Fatalf("status = %q, ingin cancelled", after.Status)
	}

	_, reservedAfter := h.stockOf(t, variantID)
	if reservedAfter != reservedBefore {
		t.Fatalf("reservasi tidak dilepas: %d → %d", reservedBefore, reservedAfter)
	}
}

// TestM2NoOversell memastikan stok tidak bisa terjual melebihi persediaan.
func TestM2NoOversell(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"
	email, userID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, email)
	defer h.cleanupOrders(t, userID)

	addrID := h.createAddress(t)

	// Varian dengan stok kecil, tetapi masih bisa dipesan sekaligus:
	// max_per_order membatasi kuantitas per pesanan (CART_QTY_MAX), jadi stok
	// harus berada dalam batas itu agar pesanan pertama sah.
	var (
		variantID uuid.UUID
		available int32
		maxPer    int32
	)
	if err := h.db.QueryRow(context.Background(), `
		SELECT v.id, (v.stock_on_hand - v.stock_reserved)::int, v.max_per_order
		FROM product_variants v
		JOIN products p ON p.id = v.product_id
		WHERE v.is_active AND p.is_active AND p.deleted_at IS NULL
		  AND v.stock_on_hand - v.stock_reserved > 0
		  AND v.stock_on_hand - v.stock_reserved <= v.max_per_order
		ORDER BY v.stock_on_hand ASC
		LIMIT 1`).Scan(&variantID, &available, &maxPer); err != nil {
		t.Skipf("tidak ada varian berstok kecil yang bisa dipesan sekaligus: %v", err)
	}

	h.addToCart(t, variantID, int(available))

	// Pesanan pertama menghabiskan stok; pesanan kedua harus ditolak.
	first := h.checkoutStatus(t, addrID, "over-1-"+uuid.NewString())
	if first != http.StatusCreated {
		t.Fatalf("pesanan pertama = %d, ingin 201", first)
	}

	// Keranjang sudah kosong setelah checkout, isi lagi untuk percobaan kedua.
	if status, raw := h.do(t, http.MethodPut, "/api/v1/cart/items/"+variantID.String(),
		map[string]any{"quantity": 1}); status != http.StatusOK {
		t.Fatalf("isi ulang keranjang = %d. body=%s", status, raw)
	}

	second := h.checkoutStatus(t, addrID, "over-2-"+uuid.NewString())
	if second == http.StatusCreated {
		t.Fatal("oversell: pesanan kedua berhasil padahal stok sudah direservasi")
	}

	// Invariant database tetap benar.
	onHand, reserved := h.stockOf(t, variantID)
	if reserved > onHand {
		t.Fatalf("stock_reserved %d melebihi stock_on_hand %d", reserved, onHand)
	}
}

// TestM2OrderIDOR memastikan pesanan orang lain tidak bisa dibaca.
func TestM2OrderIDOR(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	const password = "kaleng-enak-2026"

	emailA, userA := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, emailA)
	defer h.cleanupOrders(t, userA)

	addrID := h.createAddress(t)
	variantID, _, _ := h.seedVariant(t, 10)
	h.addToCart(t, variantID, 1)
	created := h.checkout(t, addrID, "idor-"+uuid.NewString())

	// Pengguna kedua memakai sesi baru.
	otherEmail, otherID := h.registerAndLogin(t, password)
	defer h.cleanupUser(t, otherEmail)
	defer h.cleanupOrders(t, otherID)

	status, raw := h.do(t, http.MethodGet, "/api/v1/orders/"+created.ID, nil)
	if status != http.StatusNotFound {
		t.Fatalf("pesanan orang lain = %d, ingin 404. body=%s", status, raw)
	}
}
