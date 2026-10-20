// Package order mengatur checkout dan siklus hidup pesanan (§12.4, §12.5).
//
// Aturan yang dijaga di sini:
//   - harga dan total SELALU dihitung server; tidak ada nilai uang dari klien;
//   - stok hanya berubah lewat reservasi/pelepasan/penjualan yang tercatat di
//     stock_movements (§11.3);
//   - perpindahan status hanya lewat satu fungsi `transition` yang memvalidasi
//     tabel §11.4 dan menulis order_status_history di transaksi yang sama.
package order

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	ord "github.com/kalengo/api/internal/domain/order"
	"github.com/kalengo/api/internal/platform/payment"
	"github.com/kalengo/api/internal/platform/shipping"
	"github.com/kalengo/api/internal/repo"
)

// Store adalah irisan repo.Store yang dibutuhkan layanan ini.
type Store interface {
	GetCartItems(ctx context.Context, userID uuid.UUID) ([]domain.CartItem, error)
	ClearCartForUser(ctx context.Context, userID uuid.UUID) error

	GenerateOrderNo(ctx context.Context, at time.Time) (string, error)
	CreateOrder(ctx context.Context, p repo.CreateOrderParams) (ord.Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) (ord.Order, error)
	GetOrderForUser(ctx context.Context, orderID, userID uuid.UUID) (ord.Order, error)
	GetOrderByNo(ctx context.Context, orderNo string) (ord.Order, error)
	ListOrdersForUser(ctx context.Context, userID uuid.UUID, f repo.OrderListFilter) ([]ord.Order, error)
	ListOrderItems(ctx context.Context, orderID uuid.UUID) ([]ord.Item, error)

	CreatePayment(ctx context.Context, p repo.CreatePaymentParams) (ord.Payment, error)
	UpdateLatestPaymentStatus(ctx context.Context, orderID uuid.UUID, status ord.PaymentStatus, raw []byte) error
	RecordWebhookEvent(ctx context.Context, provider, eventID string, signatureValid bool, payload []byte) (bool, error)
	MarkWebhookProcessed(ctx context.Context, provider, eventID string, procErr *string) error
	UpsertShipment(ctx context.Context, orderID uuid.UUID, courier, trackingNo, status string, events []byte) error

	GetVoucherByCode(ctx context.Context, code string) (ord.Voucher, error)
	ClaimVoucherUsage(ctx context.Context, voucherID uuid.UUID) (bool, error)
	CountVoucherRedemptions(ctx context.Context, voucherID, userID uuid.UUID) (int64, error)
	InsertVoucherRedemption(ctx context.Context, voucherID, userID, orderID uuid.UUID, amount int64) error

	CancelOrder(ctx context.Context, orderID, userID uuid.UUID, from, to ord.Status, note *string) (bool, error)
	MarkPaid(ctx context.Context, orderID uuid.UUID, note *string) (bool, error)
	ExpireOrder(ctx context.Context, orderID uuid.UUID) (bool, error)
	ListExpiredPendingOrderIDs(ctx context.Context, limit int32) ([]uuid.UUID, error)

	GetAddress(ctx context.Context, userID, id uuid.UUID) (domain.Address, error)
}

// AddressReader membaca satu alamat milik pengguna (sudah ada di repo.Store).

// Deps mengonfigurasi Service.
type Deps struct {
	Config   *config.Config
	Store    Store
	Payments payment.Provider
	Shipping shipping.Provider
	Log      *slog.Logger
	// Now dapat diganti pada uji.
	Now func() time.Time
}

// Service menjalankan alur checkout dan siklus pesanan.
type Service struct {
	cfg      *config.Config
	store    Store
	payments payment.Provider
	shipping shipping.Provider
	log      *slog.Logger
	now      func() time.Time
}

// New membuat Service.
func New(d Deps) *Service {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		cfg:      d.Config,
		store:    d.Store,
		payments: d.Payments,
		shipping: d.Shipping,
		log:      d.Log,
		now:      now,
	}
}

// ---------------------------------------------------------------------------
// checkout
// ---------------------------------------------------------------------------

// CreateInput adalah payload checkout. Perhatikan: TIDAK ada harga di sini.
type CreateInput struct {
	UserID             uuid.UUID
	IdempotencyKey     string
	AddressID          uuid.UUID
	Courier            string
	CourierService     string
	VoucherCode        string
	RequestFingerprint string
}

// CreateResult membawa pesanan beserta penanda apakah baru dibuat.
//
// Created=false berarti permintaan ulang dengan kunci idempotensi yang sama:
// pemanggil harus membalas 200, bukan 201.
type CreateResult struct {
	Order   ord.Order
	Items   []ord.Item
	Created bool
}

// Create membuat pesanan dari keranjang pengguna.
func (s *Service) Create(ctx context.Context, in CreateInput) (CreateResult, error) {
	key := strings.TrimSpace(in.IdempotencyKey)
	if len(key) < 8 || len(key) > 128 {
		return CreateResult{}, apperr.New(apperr.CodeAPIValidation, map[string]string{
			"Idempotency-Key": "wajib diisi, 8–128 karakter",
		})
	}

	// 1. Replay: kunci yang sama mengembalikan pesanan yang sama.
	if existing, err := s.store.GetOrderByIdempotencyKey(ctx, in.UserID, key); err == nil {
		if existing.Fingerprint != nil && in.RequestFingerprint != "" && *existing.Fingerprint != in.RequestFingerprint {
			return CreateResult{}, apperr.New(apperr.CodeAPIIdempotencyConflict, nil)
		}
		items, ierr := s.store.ListOrderItems(ctx, existing.ID)
		if ierr != nil {
			return CreateResult{}, apperr.Wrap(ierr, apperr.CodeAPIServerError, nil)
		}
		return CreateResult{Order: existing, Items: items, Created: false}, nil
	} else if !errors.Is(err, repo.ErrNotFound) {
		return CreateResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	// 2. Isi keranjang — harga dari server, bukan dari klien.
	cartItems, err := s.store.GetCartItems(ctx, in.UserID)
	if err != nil {
		return CreateResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if len(cartItems) == 0 {
		return CreateResult{}, apperr.New(apperr.CodeCartEmpty, nil)
	}

	items := make([]ord.ItemInput, 0, len(cartItems))
	var subtotal, weightGram int64
	for _, it := range cartItems {
		if !it.IsAvailable {
			return CreateResult{}, apperr.New(apperr.CodeCartItemUnavailable, map[string]string{
				"produk": it.ProductName,
			})
		}
		if it.Available < it.Quantity {
			return CreateResult{}, apperr.New(apperr.CodeStockChangedCheckout, map[string]string{
				"produk": it.ProductName,
				"n":      itoa(int64(it.Available)),
			})
		}

		items = append(items, ord.ItemInput{
			VariantID:    it.VariantID,
			ProductName:  it.ProductName,
			VariantLabel: variantLabel(it.VolumeML, it.PackSize),
			SKU:          it.SKU,
			UnitPrice:    it.UnitPrice,
			Quantity:     it.Quantity,
		})
		subtotal += it.LineTotal()
		// Berat diambil dari katalog; estimasi konservatif 500 g per unit.
		weightGram += int64(it.Quantity) * 500
	}

	// 3. Alamat — kepemilikan diverifikasi di lapisan repo.
	addr, err := s.store.GetAddress(ctx, in.UserID, in.AddressID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return CreateResult{}, apperr.New(apperr.CodeChkAddressReq, nil)
		}
		return CreateResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	// 4. Ongkir dihitung server lewat penyedia.
	rates, err := s.shipping.Rates(ctx, shipping.RateInput{
		DestinationAreaID: deref(addr.AreaID),
		PostalCode:        addr.PostalCode,
		WeightGram:        int32(weightGram),
		Subtotal:          subtotal,
		Courier:           strings.ToLower(strings.TrimSpace(in.Courier)),
	})
	if err != nil {
		if errors.Is(err, shipping.ErrUnserviceable) {
			return CreateResult{}, apperr.New(apperr.CodeChkAreaUnserved, nil)
		}
		return CreateResult{}, apperr.Wrap(err, apperr.CodeChkShippingFail, nil)
	}

	chosen, ok := pickRate(rates, in.CourierService)
	if !ok {
		return CreateResult{}, apperr.New(apperr.CodeChkShippingFail, nil)
	}

	// 5. Voucher (opsional).
	var (
		discount   int64
		voucherID  *uuid.UUID
		voucherRaw *ord.Voucher
	)
	if code := strings.TrimSpace(in.VoucherCode); code != "" {
		v, verr := s.validateVoucher(ctx, in.UserID, code, subtotal)
		if verr != nil {
			return CreateResult{}, verr
		}
		d, ferr := computeDiscount(v, subtotal)
		if ferr != nil {
			return CreateResult{}, ferr
		}
		discount = d
		voucherID = &v.ID
		voucherRaw = &v
	}

	// 6. Simpan: reservasi stok + pesanan + item + riwayat dalam satu transaksi.
	orderID := uuid.New()
	orderNo, err := s.store.GenerateOrderNo(ctx, s.now())
	if err != nil {
		return CreateResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	snapshot, _ := json.Marshal(map[string]any{
		"label":       addr.Label,
		"recipient":   addr.Recipient,
		"phone":       addr.Phone,
		"province":    addr.Province,
		"city":        addr.City,
		"district":    addr.District,
		"postal_code": addr.PostalCode,
		"street":      addr.Street,
		"notes":       addr.Notes,
		"area_id":     addr.AreaID,
	})

	var fingerprint *string
	if in.RequestFingerprint != "" {
		fingerprint = &in.RequestFingerprint
	}

	created, err := s.store.CreateOrder(ctx, repo.CreateOrderParams{
		OrderID:         orderID,
		OrderNo:         orderNo,
		UserID:          in.UserID,
		Subtotal:        subtotal,
		Discount:        discount,
		ShippingCost:    chosen.Cost,
		Total:           subtotal - discount + chosen.Cost,
		VoucherID:       voucherID,
		AddressSnapshot: snapshot,
		Courier:         &chosen.Courier,
		CourierService:  &chosen.Service,
		IdempotencyKey:  key,
		Fingerprint:     deref(fingerprint),
		ExpiresAt:       s.now().Add(s.reservationWindow()),
		Items:           items,
	})
	if err != nil {
		var short *repo.ErrInsufficientStock
		if errors.As(err, &short) {
			return CreateResult{}, apperr.New(apperr.CodeStockChangedCheckout, map[string]string{
				"produk": short.VariantID.String(),
				"n":      itoa(int64(short.Requested)),
			})
		}
		if repo.IsUniqueViolation(err) {
			// Balapan dengan permintaan paralel yang memakai kunci sama.
			if existing, gerr := s.store.GetOrderByIdempotencyKey(ctx, in.UserID, key); gerr == nil {
				saved, _ := s.store.ListOrderItems(ctx, existing.ID)
				return CreateResult{Order: existing, Items: saved, Created: false}, nil
			}
		}
		// Galat mentah dicatat agar penyebabnya tidak tertutup envelope 500.
		s.log.ErrorContext(ctx, "gagal membuat pesanan", "error", err, "order_no", orderNo)
		return CreateResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	// 7. Pemakaian voucher dicatat setelah pesanan ada (FK order_id).
	if voucherRaw != nil {
		if _, cerr := s.store.ClaimVoucherUsage(ctx, voucherRaw.ID); cerr != nil {
			s.log.ErrorContext(ctx, "gagal mencatat pemakaian voucher", "error", cerr, "order_id", created.ID)
		}
		if rerr := s.store.InsertVoucherRedemption(ctx, voucherRaw.ID, in.UserID, created.ID, discount); rerr != nil {
			s.log.ErrorContext(ctx, "gagal mencatat redeem voucher", "error", rerr, "order_id", created.ID)
		}
	}

	if cerr := s.store.ClearCartForUser(ctx, in.UserID); cerr != nil {
		// Keranjang yang tidak kosong tidak membatalkan pesanan yang sudah sah.
		s.log.WarnContext(ctx, "gagal mengosongkan keranjang setelah checkout", "error", cerr)
	}

	saved, err := s.store.ListOrderItems(ctx, created.ID)
	if err != nil {
		return CreateResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return CreateResult{Order: created, Items: saved, Created: true}, nil
}

func (s *Service) reservationWindow() time.Duration {
	if s.cfg != nil && s.cfg.Order.ReservationMinutes > 0 {
		return time.Duration(s.cfg.Order.ReservationMinutes) * time.Minute
	}
	return 15 * time.Minute
}

// ---------------------------------------------------------------------------
// baca & batalkan
// ---------------------------------------------------------------------------

// Detail adalah pesanan beserta itemnya.
type Detail struct {
	Order ord.Order
	Items []ord.Item
}

// List mengembalikan riwayat pesanan pengguna.
func (s *Service) List(ctx context.Context, userID uuid.UUID, status *ord.Status, limit int32) ([]ord.Order, error) {
	out, err := s.store.ListOrdersForUser(ctx, userID, repo.OrderListFilter{Status: status, Limit: limit})
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return out, nil
}

// Get mengambil detail pesanan milik pengguna.
func (s *Service) Get(ctx context.Context, userID, orderID uuid.UUID) (Detail, error) {
	o, err := s.store.GetOrderForUser(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return Detail{}, apperr.New(apperr.CodeOrdNotFound, nil)
		}
		return Detail{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	items, err := s.store.ListOrderItems(ctx, orderID)
	if err != nil {
		return Detail{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return Detail{Order: o, Items: items}, nil
}

// Cancel membatalkan pesanan milik pengguna dan melepas reservasi stoknya.
func (s *Service) Cancel(ctx context.Context, userID, orderID uuid.UUID, reason string) (Detail, error) {
	o, err := s.store.GetOrderForUser(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return Detail{}, apperr.New(apperr.CodeOrdNotFound, nil)
		}
		return Detail{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if !o.Status.CancellableByCustomer() {
		return Detail{}, apperr.New(apperr.CodeOrdCancelDenied, nil)
	}
	if !ord.CanTransition(o.Status, ord.StatusCancelled) {
		return Detail{}, apperr.New(apperr.CodeOrdInvalidTransition, map[string]string{
			"status": string(ord.StatusCancelled),
		})
	}

	var note *string
	if r := strings.TrimSpace(reason); r != "" {
		note = &r
	}
	moved, err := s.store.CancelOrder(ctx, orderID, userID, o.Status, ord.StatusCancelled, note)
	if err != nil {
		return Detail{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if !moved {
		// Kalah balapan dengan webhook pembayaran atau sweeper kedaluwarsa.
		return Detail{}, apperr.New(apperr.CodeOrdInvalidTransition, map[string]string{
			"status": string(ord.StatusCancelled),
		})
	}
	return s.Get(ctx, userID, orderID)
}

// ---------------------------------------------------------------------------
// pembayaran
// ---------------------------------------------------------------------------

// PaymentSession adalah hasil permintaan pembayaran.
type PaymentSession struct {
	OrderNo     string    `json:"order_no"`
	Total       int64     `json:"total"`
	Provider    string    `json:"provider"`
	Token       string    `json:"token"`
	RedirectURL string    `json:"redirect_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// InitPayment membuat sesi pembayaran untuk pesanan yang belum dibayar.
func (s *Service) InitPayment(ctx context.Context, userID, orderID uuid.UUID) (PaymentSession, error) {
	o, err := s.store.GetOrderForUser(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return PaymentSession{}, apperr.New(apperr.CodeOrdNotFound, nil)
		}
		return PaymentSession{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if o.Status.IsPaidFamily() {
		return PaymentSession{}, apperr.New(apperr.CodePayAlreadyPaid, nil)
	}
	if o.Status != ord.StatusPendingPayment {
		return PaymentSession{}, apperr.New(apperr.CodePayExpired, nil)
	}
	if !s.now().Before(o.ExpiresAt) {
		return PaymentSession{}, apperr.New(apperr.CodePayExpired, nil)
	}

	sess, err := s.payments.CreateSession(ctx, payment.SessionInput{
		OrderID:    o.ID,
		OrderNo:    o.OrderNo,
		Amount:     o.Total,
		CustomerID: userID,
		ExpiresAt:  o.ExpiresAt,
	})
	if err != nil {
		return PaymentSession{}, apperr.Wrap(err, apperr.CodePayProviderError, nil)
	}

	ref := sess.ProviderRef
	if _, err := s.store.CreatePayment(ctx, repo.CreatePaymentParams{
		OrderID:     o.ID,
		Provider:    s.payments.Name(),
		ProviderRef: &ref,
		Amount:      o.Total,
		Status:      ord.PaymentPending,
	}); err != nil {
		return PaymentSession{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	return PaymentSession{
		OrderNo:     o.OrderNo,
		Total:       o.Total,
		Provider:    s.payments.Name(),
		Token:       sess.Token,
		RedirectURL: sess.RedirectURL,
		ExpiresAt:   sess.ExpiresAt,
	}, nil
}

// WebhookResult adalah ringkasan hasil pemrosesan webhook.
type WebhookResult struct {
	// Duplicate=true berarti event sudah pernah diproses (§13.1 idempotensi).
	Duplicate bool
	// Applied=true berarti status pesanan benar-benar berubah.
	Applied bool
	Status  string
}

// HandlePaymentWebhook memverifikasi dan menerapkan notifikasi pembayaran.
//
// Urutan yang WAJIB dijaga (§13.1 3.7):
//  1. tanda tangan diverifikasi lebih dulu — payload tidak sah tidak menyentuh DB;
//  2. idempotensi event dicek sebelum perubahan state apa pun;
//  3. nominal dibandingkan dengan total pesanan DI DATABASE;
//  4. transisi status lewat mesin status, bukan ditulis langsung.
func (s *Service) HandlePaymentWebhook(ctx context.Context, raw []byte) (WebhookResult, error) {
	ev, err := s.payments.ParseWebhook(ctx, raw)
	if err != nil {
		if errors.Is(err, payment.ErrSignature) {
			return WebhookResult{}, apperr.New(apperr.CodePayWebhookSignature, nil)
		}
		return WebhookResult{}, apperr.New(apperr.CodeAPIBadRequest, nil)
	}

	inserted, err := s.store.RecordWebhookEvent(ctx, s.payments.Name(), ev.EventID, true, raw)
	if err != nil {
		return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if !inserted {
		return WebhookResult{Duplicate: true, Status: "duplicate"}, nil
	}

	result, procErr := s.applyPaymentEvent(ctx, ev)

	var errNote *string
	if procErr != nil {
		msg := procErr.Error()
		errNote = &msg
	}
	if err := s.store.MarkWebhookProcessed(ctx, s.payments.Name(), ev.EventID, errNote); err != nil {
		s.log.ErrorContext(ctx, "gagal menandai webhook selesai", "error", err, "event_id", ev.EventID)
	}
	return result, procErr
}

// applyPaymentEvent menerjemahkan event penyedia menjadi perubahan state.
func (s *Service) applyPaymentEvent(ctx context.Context, ev payment.Event) (WebhookResult, error) {
	o, err := s.store.GetOrderByNo(ctx, ev.ProviderRef)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return WebhookResult{}, apperr.New(apperr.CodeOrdNotFound, nil)
		}
		return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	switch ev.Status {
	case payment.EventPending:
		return WebhookResult{Status: string(o.Status)}, nil

	case payment.EventSucceeded:
		if !ev.FraudAccepted {
			return WebhookResult{}, apperr.New(apperr.CodePayFailed, nil)
		}
		// Nominal WAJIB sama dengan total pesanan di database.
		if ev.GrossAmount != o.Total {
			s.log.ErrorContext(ctx, "nominal webhook tidak cocok",
				"order_no", o.OrderNo, "expected", o.Total, "got", ev.GrossAmount)
			return WebhookResult{}, apperr.New(apperr.CodePayAmountMismatch, nil)
		}
		return s.moveTo(ctx, o, ord.StatusPaid, "payment settled")

	case payment.EventFailed:
		return s.moveTo(ctx, o, ord.StatusCancelled, "payment failed")

	case payment.EventExpired:
		moved, err := s.store.ExpireOrder(ctx, o.ID)
		if err != nil {
			return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
		}
		return WebhookResult{Applied: moved, Status: string(ord.StatusExpired)}, nil

	case payment.EventRefunded:
		// Refund tidak pernah otomatis: harus lewat alur admin (M4).
		s.log.WarnContext(ctx, "event refund diterima; menunggu tindakan admin", "order_no", o.OrderNo)
		return WebhookResult{Status: string(o.Status)}, nil

	default:
		return WebhookResult{}, apperr.New(apperr.CodeAPIBadRequest, nil)
	}
}

// moveTo menjalankan transisi pembayaran yang menuju `paid` atau `cancelled`.
func (s *Service) moveTo(ctx context.Context, o ord.Order, target ord.Status, note string) (WebhookResult, error) {
	if !ord.CanTransition(o.Status, target) {
		if o.Status.IsPaidFamily() && target == ord.StatusPaid {
			// Pembayaran kedua untuk pesanan yang sudah lunas: bukan kegagalan baru.
			return WebhookResult{Status: string(o.Status)}, nil
		}
		return WebhookResult{}, apperr.New(apperr.CodeOrdInvalidTransition, map[string]string{
			"status": string(target),
		})
	}

	if target == ord.StatusPaid {
		moved, err := s.store.MarkPaid(ctx, o.ID, &note)
		if err != nil {
			return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
		}
		if err := s.store.UpdateLatestPaymentStatus(ctx, o.ID, ord.PaymentPaid, nil); err != nil {
			s.log.ErrorContext(ctx, "gagal memperbarui status pembayaran", "error", err, "order_id", o.ID)
		}
		return WebhookResult{Applied: moved, Status: string(ord.StatusPaid)}, nil
	}

	moved, err := s.store.CancelOrder(ctx, o.ID, o.UserID, o.Status, target, &note)
	if err != nil {
		return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.store.UpdateLatestPaymentStatus(ctx, o.ID, ord.PaymentFailed, nil); err != nil {
		s.log.ErrorContext(ctx, "gagal memperbarui status pembayaran", "error", err, "order_id", o.ID)
	}
	return WebhookResult{Applied: moved, Status: string(target)}, nil
}

// HandleShippingWebhook menerapkan notifikasi status pengiriman.
//
// M2 hanya menyimpan status pengiriman; perpindahan status pesanan ke `shipped`
// tetap milik admin (M4) agar tracking palsu tidak mengubah pesanan.
func (s *Service) HandleShippingWebhook(ctx context.Context, raw []byte) (WebhookResult, error) {
	ev, err := s.shipping.ParseWebhook(ctx, raw)
	if err != nil {
		return WebhookResult{}, apperr.New(apperr.CodePayWebhookSignature, nil)
	}

	o, err := s.store.GetOrderByNo(ctx, ev.OrderNo)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return WebhookResult{}, apperr.New(apperr.CodeOrdNotFound, nil)
		}
		return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	inserted, err := s.store.RecordWebhookEvent(ctx, s.shipping.Name(), ev.EventID, true, raw)
	if err != nil {
		return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if !inserted {
		return WebhookResult{Duplicate: true, Status: "duplicate"}, nil
	}

	if err := s.store.UpsertShipment(ctx, o.ID, ev.Courier, ev.TrackingNo, ev.Status, raw); err != nil {
		return WebhookResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	_ = s.store.MarkWebhookProcessed(ctx, s.shipping.Name(), ev.EventID, nil)

	return WebhookResult{Applied: true, Status: ev.Status}, nil
}

// ---------------------------------------------------------------------------
// kedaluwarsa
// ---------------------------------------------------------------------------

// ExpireResult merangkum hasil sapuan kedaluwarsa.
type ExpireResult struct {
	Examined int
	Expired  int
}

// ExpireDue melepas reservasi stok pesanan yang melewati batas bayar (§15).
func (s *Service) ExpireDue(ctx context.Context, limit int32) (ExpireResult, error) {
	ids, err := s.store.ListExpiredPendingOrderIDs(ctx, limit)
	if err != nil {
		return ExpireResult{}, err
	}

	out := ExpireResult{Examined: len(ids)}
	for _, id := range ids {
		moved, err := s.store.ExpireOrder(ctx, id)
		if err != nil {
			s.log.ErrorContext(ctx, "gagal mengedaluwarsakan pesanan", "error", err, "order_id", id)
			continue
		}
		if moved {
			out.Expired++
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// voucher & ongkir
// ---------------------------------------------------------------------------

// VoucherInfo adalah hasil validasi voucher.
type VoucherInfo struct {
	Code        string `json:"code"`
	Type        string `json:"type"`
	Discount    int64  `json:"discount"`
	MinSpend    int64  `json:"min_spend"`
	Description string `json:"description"`
}

// ValidateVoucher memeriksa voucher terhadap subtotal keranjang pengguna.
func (s *Service) ValidateVoucher(ctx context.Context, userID uuid.UUID, code string) (VoucherInfo, error) {
	cartItems, err := s.store.GetCartItems(ctx, userID)
	if err != nil {
		return VoucherInfo{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	var subtotal int64
	for _, it := range cartItems {
		if it.IsAvailable {
			subtotal += it.LineTotal()
		}
	}

	v, err := s.validateVoucher(ctx, userID, code, subtotal)
	if err != nil {
		return VoucherInfo{}, err
	}
	discount, err := computeDiscount(v, subtotal)
	if err != nil {
		return VoucherInfo{}, err
	}
	return VoucherInfo{
		Code:        v.Code,
		Type:        v.Type,
		Discount:    discount,
		MinSpend:    v.MinSpend,
		Description: voucherDescription(v, discount),
	}, nil
}

// validateVoucher memeriksa keberlakuan voucher untuk pengguna dan subtotal.
func (s *Service) validateVoucher(ctx context.Context, userID uuid.UUID, code string, subtotal int64) (ord.Voucher, error) {
	v, err := s.store.GetVoucherByCode(ctx, strings.TrimSpace(code))
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ord.Voucher{}, apperr.New(apperr.CodeVoucherInvalid, nil)
		}
		return ord.Voucher{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	now := s.now()
	switch {
	case !v.IsActive:
		return ord.Voucher{}, apperr.New(apperr.CodeVoucherInvalid, nil)
	case now.Before(v.StartsAt):
		return ord.Voucher{}, apperr.New(apperr.CodeVoucherInvalid, nil)
	case now.After(v.EndsAt):
		return ord.Voucher{}, apperr.New(apperr.CodeVoucherExpired, nil)
	case v.UsageLimit != nil && v.UsageCount >= *v.UsageLimit:
		return ord.Voucher{}, apperr.New(apperr.CodeVoucherUsedUp, nil)
	case subtotal < v.MinSpend:
		return ord.Voucher{}, apperr.New(apperr.CodeVoucherMinSpend, map[string]string{
			"min": itoa(v.MinSpend),
		})
	}

	used, err := s.store.CountVoucherRedemptions(ctx, v.ID, userID)
	if err != nil {
		return ord.Voucher{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if used >= int64(v.PerUserMax) {
		return ord.Voucher{}, apperr.New(apperr.CodeVoucherLimitUser, nil)
	}
	return v, nil
}

// computeDiscount menghitung potongan dalam rupiah bulat.
func computeDiscount(v ord.Voucher, subtotal int64) (int64, error) {
	switch v.Type {
	case ord.VoucherPercent:
		d := subtotal * v.Value / 100
		if v.MaxDiscount != nil && d > *v.MaxDiscount {
			d = *v.MaxDiscount
		}
		return min64(d, subtotal), nil
	case ord.VoucherFixed:
		return min64(v.Value, subtotal), nil
	case ord.VoucherFreeShipping:
		// Potongan ongkir dihitung saat ongkir diketahui; di sini 0.
		return 0, nil
	default:
		return 0, apperr.New(apperr.CodeVoucherInvalid, nil)
	}
}

// ShippingOptions menghitung tarif untuk isi keranjang pengguna.
func (s *Service) ShippingOptions(ctx context.Context, userID, addressID uuid.UUID, courier string) ([]shipping.Rate, error) {
	cartItems, err := s.store.GetCartItems(ctx, userID)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if len(cartItems) == 0 {
		return nil, apperr.New(apperr.CodeCartEmpty, nil)
	}

	var subtotal, weight int64
	for _, it := range cartItems {
		if !it.IsAvailable {
			continue
		}
		subtotal += it.LineTotal()
		weight += int64(it.Quantity) * 500
	}

	addr, err := s.store.GetAddress(ctx, userID, addressID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, apperr.New(apperr.CodeChkAddressReq, nil)
		}
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	rates, err := s.shipping.Rates(ctx, shipping.RateInput{
		DestinationAreaID: deref(addr.AreaID),
		PostalCode:        addr.PostalCode,
		WeightGram:        int32(weight),
		Subtotal:          subtotal,
		Courier:           strings.ToLower(strings.TrimSpace(courier)),
	})
	if err != nil {
		if errors.Is(err, shipping.ErrUnserviceable) {
			return nil, apperr.New(apperr.CodeChkAreaUnserved, nil)
		}
		return nil, apperr.Wrap(err, apperr.CodeChkShippingFail, nil)
	}
	return rates, nil
}

// ---------------------------------------------------------------------------
// pembantu
// ---------------------------------------------------------------------------

func pickRate(rates []shipping.Rate, service string) (shipping.Rate, bool) {
	want := strings.TrimSpace(service)
	if want == "" {
		if len(rates) == 0 {
			return shipping.Rate{}, false
		}
		return rates[0], true
	}
	for _, r := range rates {
		if strings.EqualFold(r.Service, want) {
			return r, true
		}
	}
	return shipping.Rate{}, false
}

func variantLabel(volumeML, packSize int32) string {
	if volumeML <= 0 {
		return ""
	}
	label := itoa(int64(volumeML)) + " ml"
	if packSize > 1 {
		label += " × " + itoa(int64(packSize))
	}
	return label
}

func voucherDescription(v ord.Voucher, discount int64) string {
	switch v.Type {
	case ord.VoucherPercent:
		return "Diskon " + itoa(v.Value) + "%"
	case ord.VoucherFixed:
		return "Potongan " + itoa(discount)
	case ord.VoucherFreeShipping:
		return "Gratis ongkir"
	default:
		return ""
	}
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Fingerprint menghasilkan sidik jari kanonik dari payload checkout.
//
// Dipakai untuk membedakan "kunci sama, isi sama" (replay sah) dari "kunci sama,
// isi berbeda" (API_IDEMPOTENCY_CONFLICT).
func Fingerprint(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
