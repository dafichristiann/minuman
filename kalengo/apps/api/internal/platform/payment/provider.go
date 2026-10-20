// Package payment menyediakan antarmuka penyedia pembayaran (§12.4, §12.5).
//
// M2 hanya memakai implementasi palsu (Fake) yang berjalan di dalam proses,
// tanpa panggilan jaringan. Adapter Midtrans sungguhan ditambahkan pada M5
// dengan kontrak yang sama, sehingga layanan pesanan tidak perlu berubah.
package payment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Errors yang dikembalikan penyedia.
var (
	// ErrSignature menandakan payload webhook tidak dapat dipercaya.
	ErrSignature = errors.New("payment: tanda tangan webhook tidak valid")
	// ErrMalformed menandakan payload tidak bisa dibaca sebagai event.
	ErrMalformed = errors.New("payment: payload webhook tidak dikenali")
)

// SessionInput adalah permintaan pembuatan sesi pembayaran.
type SessionInput struct {
	OrderID    uuid.UUID
	OrderNo    string
	Amount     int64 // rupiah bulat
	CustomerID uuid.UUID
	ExpiresAt  time.Time
}

// Session adalah hasil pembuatan sesi pembayaran.
//
// RedirectURL boleh kosong pada implementasi palsu; klien hanya membutuhkannya
// untuk berpindah ke halaman penyedia.
type Session struct {
	ProviderRef string
	Token       string
	RedirectURL string
	ExpiresAt   time.Time
}

// EventStatus adalah status pembayaran yang dinormalkan dari penyedia.
type EventStatus string

// Status event yang dikenal.
const (
	EventPending   EventStatus = "pending"
	EventSucceeded EventStatus = "succeeded"
	EventFailed    EventStatus = "failed"
	EventExpired   EventStatus = "expired"
	EventRefunded  EventStatus = "refunded"
)

// Event adalah webhook yang sudah diverifikasi dan dinormalkan.
type Event struct {
	// EventID adalah kunci idempotensi. Event yang sama dikirim dua kali harus
	// menghasilkan EventID yang sama sehingga bisa ditolak di tingkat database.
	EventID string
	// ProviderRef menunjuk payments.provider_ref.
	ProviderRef string
	Status      EventStatus
	// GrossAmount adalah jumlah rupiah yang diklaim penyedia. Pemanggil WAJIB
	// membandingkannya dengan orders.total — nilai ini tidak boleh dipercaya.
	GrossAmount int64
	// FraudAccepted false berarti penyedia menandai transaksi mencurigakan.
	FraudAccepted bool
}

// Provider adalah antarmuka penyedia pembayaran.
type Provider interface {
	Name() string
	// CreateSession membuat sesi pembayaran untuk sebuah pesanan.
	CreateSession(ctx context.Context, in SessionInput) (Session, error)
	// ParseWebhook memverifikasi tanda tangan lalu menormalkan payload.
	// Bila tanda tangan tidak valid, kembalikan ErrSignature dan JANGAN sentuh
	// state apa pun.
	ParseWebhook(ctx context.Context, raw []byte) (Event, error)
}
