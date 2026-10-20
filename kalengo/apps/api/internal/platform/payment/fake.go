package payment

import (
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Fake adalah penyedia pembayaran tiruan yang meniru skema tanda tangan
// Midtrans tanpa melakukan panggilan jaringan.
//
// Skema tanda tangan Midtrans (§12.5):
//
//	sha512(order_id + status_code + gross_amount + server_key)
//
// Nilai `gross_amount` di-hash PERSIS seperti yang diterima (biasanya
// "125000.00"), bukan hasil parsing lalu diformat ulang. Perbedaan sekecil itu
// akan membuat tanda tangan sah ditolak.
type Fake struct {
	// ServerKey adalah kunci rahasia yang dipakai bersama dengan penyedia.
	ServerKey string
	// Now dapat diganti pada uji untuk mengendalikan waktu kedaluwarsa sesi.
	Now func() time.Time
}

// Name memenuhi Provider.
func (f *Fake) Name() string { return "midtrans-fake" }

func (f *Fake) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

// CreateSession membuat sesi pembayaran deterministik dari nomor pesanan.
//
// Tidak ada jaringan: token dibangkitkan dari hash sehingga hasilnya stabil dan
// dapat diperiksa pada uji.
func (f *Fake) CreateSession(_ context.Context, in SessionInput) (Session, error) {
	if in.OrderNo == "" {
		return Session{}, ErrMalformed
	}
	sum := sha512.Sum512([]byte(in.OrderNo + "|" + strconv.FormatInt(in.Amount, 10) + "|" + f.ServerKey))
	token := hex.EncodeToString(sum[:16])

	return Session{
		ProviderRef: in.OrderNo,
		Token:       token,
		RedirectURL: "",
		ExpiresAt:   in.ExpiresAt,
	}, nil
}

// webhookPayload adalah bentuk payload yang diterima dari penyedia.
//
// Field angka dibaca sebagai string karena penyedia mengirim desimal
// ("125000.00") dan uang tidak boleh melewati float (§13.4).
type webhookPayload struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
}

// ParseWebhook memverifikasi tanda tangan lalu menormalkan payload.
func (f *Fake) ParseWebhook(_ context.Context, raw []byte) (Event, error) {
	var p webhookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Event{}, ErrMalformed
	}
	if p.OrderID == "" || p.SignatureKey == "" || p.TransactionStatus == "" {
		return Event{}, ErrMalformed
	}
	if f.ServerKey == "" {
		// Fail closed: tanpa kunci, tanda tangan tidak bisa diverifikasi.
		return Event{}, ErrSignature
	}

	// Tanda tangan atas nilai mentah, sesuai urutan Midtrans.
	want := sha512.Sum512([]byte(p.OrderID + p.StatusCode + p.GrossAmount + f.ServerKey))
	wantHex := hex.EncodeToString(want[:])

	// Perbandingan waktu konstan agar tidak bocor lewat pengukuran waktu.
	if subtle.ConstantTimeCompare([]byte(wantHex), []byte(strings.ToLower(p.SignatureKey))) != 1 {
		return Event{}, ErrSignature
	}

	amount, err := ParseRupiah(p.GrossAmount)
	if err != nil {
		return Event{}, ErrMalformed
	}

	status, ok := mapStatus(p.TransactionStatus, p.FraudStatus)
	if !ok {
		return Event{}, ErrMalformed
	}

	return Event{
		EventID:       p.OrderID + ":" + p.TransactionStatus + ":" + p.StatusCode + ":" + p.FraudStatus,
		ProviderRef:   p.OrderID,
		Status:        status,
		GrossAmount:   amount,
		FraudAccepted: p.FraudStatus == "" || p.FraudStatus == "accept",
	}, nil
}

// mapStatus menerjemahkan status penyedia ke status ternormalisasi.
func mapStatus(transaction, fraud string) (EventStatus, bool) {
	switch transaction {
	case "pending":
		return EventPending, true
	case "capture":
		// capture dengan fraud ditolak bukan pembayaran sah.
		if fraud == "accept" {
			return EventSucceeded, true
		}
		return EventFailed, true
	case "settlement":
		return EventSucceeded, true
	case "deny", "cancel", "failure":
		return EventFailed, true
	case "expire":
		return EventExpired, true
	case "refund", "partial_refund":
		return EventRefunded, true
	default:
		return "", false
	}
}

// ParseRupiah mengubah nominal penyedia menjadi integer rupiah.
//
// Menerima "125000" dan "125000.00". Pecahan selain nol ditolak: rupiah tidak
// punya subunit yang dipakai pada sistem ini, sehingga nilai seperti "1234.56"
// menandakan kesalahan penyedia, bukan nilai yang pantas dibulatkan.
func ParseRupiah(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("payment: nominal kosong")
	}

	whole, frac, hasFrac := strings.Cut(s, ".")
	if hasFrac {
		frac = strings.TrimRight(frac, "0")
		if frac != "" {
			return 0, fmt.Errorf("payment: nominal %q bukan rupiah bulat", raw)
		}
	}
	whole = strings.TrimPrefix(whole, "+")

	v, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("payment: nominal %q tidak valid: %w", raw, err)
	}
	if v < 0 {
		return 0, fmt.Errorf("payment: nominal negatif %q", raw)
	}
	return v, nil
}

// Sign membuat tanda tangan untuk payload tiruan. Dipakai uji dan seed dev.
func (f *Fake) Sign(orderID, statusCode, grossAmount string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + f.ServerKey))
	return hex.EncodeToString(sum[:])
}

// BuildWebhook merakit payload webhook bertanda tangan sah. Membantu uji
// integrasi meniru penyedia tanpa menyalin logika tanda tangan.
func (f *Fake) BuildWebhook(orderNo, transactionStatus, grossAmount string) []byte {
	const statusCode = "200"
	body := map[string]any{
		"order_id":           orderNo,
		"status_code":        statusCode,
		"gross_amount":       grossAmount,
		"transaction_status": transactionStatus,
		"fraud_status":       "accept",
		"signature_key":      f.Sign(orderNo, statusCode, grossAmount),
	}
	raw, _ := json.Marshal(body)
	return raw
}

// ProviderRefForOrderNo memudahkan pemanggil menautkan pesanan ke pembayaran.
func ProviderRefForOrderNo(orderNo string) *string { return &orderNo }

var _ Provider = (*Fake)(nil)

// Ensure uuid tetap terpakai meski tanda tangan di atas tidak memerlukannya.
var _ = uuid.Nil
