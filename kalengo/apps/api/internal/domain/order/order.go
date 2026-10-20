package order

import (
	"time"

	"github.com/google/uuid"
)

// Order adalah satu baris `orders` yang sudah dipetakan ke tipe domain.
//
// Uang selalu integer rupiah (§13.4): tidak ada float di mana pun pada struct
// ini, termasuk saat di-scan dari bigint PostgreSQL.
type Order struct {
	ID              uuid.UUID
	OrderNo         string
	UserID          uuid.UUID
	Status          Status
	Subtotal        int64
	Discount        int64
	ShippingCost    int64
	Total           int64
	VoucherID       *uuid.UUID
	AddressSnapshot []byte
	Courier         *string
	CourierService  *string
	IdempotencyKey  string
	Fingerprint     *string
	ExpiresAt       time.Time
	PaidAt          *time.Time
	ShippedAt       *time.Time
	DeliveredAt     *time.Time
	CancelledAt     *time.Time
	CancelReason    *string
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Item adalah satu baris `order_items`. Harga dan nama dibekukan saat checkout
// sehingga perubahan katalog di kemudian hari tidak mengubah riwayat pesanan.
type Item struct {
	ID           uuid.UUID
	OrderID      uuid.UUID
	VariantID    uuid.UUID
	ProductName  string
	VariantLabel string
	SKU          string
	UnitPrice    int64
	Quantity     int32
	LineTotal    int64
}

// ItemInput adalah satu baris keranjang yang masuk ke pembuatan pesanan.
type ItemInput struct {
	VariantID    uuid.UUID
	ProductName  string
	VariantLabel string
	SKU          string
	UnitPrice    int64
	Quantity     int32
}

// LineTotal menghitung total satu baris.
func (i ItemInput) LineTotal() int64 { return i.UnitPrice * int64(i.Quantity) }

// PaymentStatus adalah nilai kolom payment_status (§11.2).
type PaymentStatus string

// Seluruh status pembayaran yang dikenal.
const (
	PaymentPending       PaymentStatus = "pending"
	PaymentPaid          PaymentStatus = "paid"
	PaymentFailed        PaymentStatus = "failed"
	PaymentExpired       PaymentStatus = "expired"
	PaymentRefunded      PaymentStatus = "refunded"
	PaymentPartialRefund PaymentStatus = "partial_refund"
)

// Payment adalah satu baris `payments`.
//
// Satu pesanan boleh punya beberapa percobaan pembayaran (pindah metode, retry),
// sehingga pembaca WAJIB memakai percobaan terbaru berdasarkan created_at.
type Payment struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	Provider    string
	ProviderRef *string
	Method      *string
	Amount      int64
	Status      PaymentStatus
	PaidAt      *time.Time
	RawResponse []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Voucher adalah satu baris `vouchers` yang sudah dipetakan.
type Voucher struct {
	ID          uuid.UUID
	Code        string
	Type        string // percent | fixed | free_shipping
	Value       int64
	MaxDiscount *int64
	MinSpend    int64
	UsageLimit  *int32
	UsageCount  int32
	PerUserMax  int32
	StartsAt    time.Time
	EndsAt      time.Time
	IsActive    bool
}

// Type voucher yang dikenal.
const (
	VoucherPercent      = "percent"
	VoucherFixed        = "fixed"
	VoucherFreeShipping = "free_shipping"
)
