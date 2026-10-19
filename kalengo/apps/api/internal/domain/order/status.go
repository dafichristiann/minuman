// Package order memuat domain pesanan, termasuk mesin status dari §11.4.
//
// Perubahan status pesanan HANYA boleh lewat satu fungsi service yang memvalidasi
// tabel transisi di sini dan menulis order_status_history (§11.3 poin 4).
package order

import (
	"fmt"
	"sort"
	"strings"
)

// Status adalah nilai kolom order_status (§11.2).
type Status string

// Seluruh status yang mungkin.
const (
	StatusPendingPayment  Status = "pending_payment"
	StatusPaid            Status = "paid"
	StatusProcessing      Status = "processing"
	StatusShipped         Status = "shipped"
	StatusDelivered       Status = "delivered"
	StatusCompleted       Status = "completed"
	StatusCancelled       Status = "cancelled"
	StatusExpired         Status = "expired"
	StatusRefundRequested Status = "refund_requested"
	StatusRefunded        Status = "refunded"
)

// AllStatuses berisi semua status, terurut seperti alur normal.
func AllStatuses() []Status {
	return []Status{
		StatusPendingPayment, StatusPaid, StatusProcessing, StatusShipped,
		StatusDelivered, StatusCompleted, StatusCancelled, StatusExpired,
		StatusRefundRequested, StatusRefunded,
	}
}

// transitions adalah tabel transisi legal (§11.4).
//
// Catatan untuk refund_requested: §11.4 menyebut "kembali ke status sebelumnya
// jika ditolak". Karena status sebelumnya bersifat dinamis, transisi keluar dari
// refund_requested divalidasi terpisah lewat RefundRejectionTarget.
var transitions = map[Status][]Status{
	StatusPendingPayment:  {StatusPaid, StatusCancelled, StatusExpired},
	StatusPaid:            {StatusProcessing, StatusRefundRequested, StatusCancelled},
	StatusProcessing:      {StatusShipped, StatusRefundRequested},
	StatusShipped:         {StatusDelivered},
	StatusDelivered:       {StatusCompleted, StatusRefundRequested},
	StatusRefundRequested: {StatusRefunded},
	StatusCompleted:       {},
	StatusCancelled:       {},
	StatusExpired:         {},
	StatusRefunded:        {},
}

// IsValid melaporkan apakah string ini status yang dikenal.
func (s Status) IsValid() bool {
	_, ok := transitions[s]
	return ok
}

// IsFinal melaporkan status akhir yang tidak bisa berubah lagi (§11.4).
func (s Status) IsFinal() bool {
	switch s {
	case StatusCompleted, StatusCancelled, StatusExpired, StatusRefunded:
		return true
	default:
		return false
	}
}

// IsPaidFamily melaporkan status yang berarti pembayaran sudah diterima.
func (s Status) IsPaidFamily() bool {
	switch s {
	case StatusPaid, StatusProcessing, StatusShipped, StatusDelivered, StatusCompleted:
		return true
	default:
		return false
	}
}

// AllowedTransitions mengembalikan daftar status tujuan yang legal, terurut.
func AllowedTransitions(from Status) []Status {
	out := append([]Status(nil), transitions[from]...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// CanTransition melaporkan apakah perpindahan from → to diizinkan.
func CanTransition(from, to Status) bool {
	if from == to {
		return false
	}
	if !from.IsValid() || !to.IsValid() {
		return false
	}
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// CanRejectRefund melaporkan apakah penolakan refund boleh mengembalikan pesanan
// ke `previous`. Hanya status yang sah untuk refund_requested yang diterima.
func CanRejectRefund(previous Status) bool {
	switch previous {
	case StatusPaid, StatusProcessing, StatusDelivered:
		return true
	default:
		return false
	}
}

// Parse mengubah string dari DB/HTTP menjadi Status yang tervalidasi.
func Parse(raw string) (Status, error) {
	s := Status(strings.ToLower(strings.TrimSpace(raw)))
	if !s.IsValid() {
		return "", fmt.Errorf("order: status %q tidak dikenal", raw)
	}
	return s, nil
}

// String memenuhi fmt.Stringer.
func (s Status) String() string { return string(s) }

// Refundable melaporkan apakah pesanan pada status ini boleh mengajukan refund.
func (s Status) Refundable() bool {
	switch s {
	case StatusPaid, StatusProcessing, StatusDelivered:
		return true
	default:
		return false
	}
}

// CancellableByCustomer melaporkan apakah pembeli boleh membatalkan sendiri.
// Setelah dikirim (shipped) pembatalan harus lewat admin + refund (§8.6 ORD_CANCEL_DENIED).
func (s Status) CancellableByCustomer() bool {
	return s == StatusPendingPayment || s == StatusPaid
}
