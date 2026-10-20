package handler

import (
	"errors"
	nethttp "net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain/order"
	ord "github.com/kalengo/api/internal/domain/order"
	"github.com/kalengo/api/internal/http/middleware"
	"github.com/kalengo/api/internal/platform/httpx"
	ordersvc "github.com/kalengo/api/internal/service/order"
)

// Order menangani endpoint pesanan (§12.4).
type Order struct {
	cfg *config.Config
	svc *ordersvc.Service
}

// NewOrder membuat handler pesanan.
func NewOrder(cfg *config.Config, svc *ordersvc.Service) *Order {
	return &Order{cfg: cfg, svc: svc}
}

type orderItemDTO struct {
	VariantID    string `json:"variant_id"`
	ProductName  string `json:"product_name"`
	VariantLabel string `json:"variant_label"`
	SKU          string `json:"sku"`
	UnitPrice    int64  `json:"unit_price"`
	Quantity     int32  `json:"quantity"`
	LineTotal    int64  `json:"line_total"`
}

func newOrderItemDTOs(items []ord.Item) []orderItemDTO {
	out := make([]orderItemDTO, 0, len(items))
	for _, it := range items {
		out = append(out, orderItemDTO{
			VariantID:    it.VariantID.String(),
			ProductName:  it.ProductName,
			VariantLabel: it.VariantLabel,
			SKU:          it.SKU,
			UnitPrice:    it.UnitPrice,
			Quantity:     it.Quantity,
			LineTotal:    it.LineTotal,
		})
	}
	return out
}

type orderDTO struct {
	ID           string         `json:"id"`
	OrderNo      string         `json:"order_no"`
	Status       string         `json:"status"`
	Subtotal     int64          `json:"subtotal"`
	Discount     int64          `json:"discount"`
	ShippingCost int64          `json:"shipping_cost"`
	Total        int64          `json:"total"`
	Courier      *string        `json:"courier"`
	CourierSvc   *string        `json:"courier_service"`
	ExpiresAt    string         `json:"expires_at"`
	PaidAt       *string        `json:"paid_at"`
	CancelledAt  *string        `json:"cancelled_at"`
	CancelReason *string        `json:"cancel_reason"`
	CreatedAt    string         `json:"created_at"`
	Items        []orderItemDTO `json:"items"`
}

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.UTC().Format(time.RFC3339)
	return &v
}

func newOrderDTO(o ord.Order, items []ord.Item) orderDTO {
	return orderDTO{
		ID:           o.ID.String(),
		OrderNo:      o.OrderNo,
		Status:       string(o.Status),
		Subtotal:     o.Subtotal,
		Discount:     o.Discount,
		ShippingCost: o.ShippingCost,
		Total:        o.Total,
		Courier:      o.Courier,
		CourierSvc:   o.CourierService,
		ExpiresAt:    o.ExpiresAt.UTC().Format(time.RFC3339),
		PaidAt:       rfc3339Ptr(o.PaidAt),
		CancelledAt:  rfc3339Ptr(o.CancelledAt),
		CancelReason: o.CancelReason,
		CreatedAt:    o.CreatedAt.UTC().Format(time.RFC3339),
		Items:        newOrderItemDTOs(items),
	}
}

type createOrderRequest struct {
	AddressID      string `json:"address_id"`
	Courier        string `json:"courier"`
	CourierService string `json:"courier_service"`
	VoucherCode    string `json:"voucher_code,omitempty"`
	Notes          string `json:"notes,omitempty"`
	AcceptTerms    bool   `json:"accept_terms,omitempty"`
}

// Create menangani POST /orders.
func (h *Order) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	var req createOrderRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	addressID, err := uuid.Parse(req.AddressID)
	if err != nil {
		httpx.Error(w, r, apperr.New(apperr.CodeAPIValidation, map[string]string{
			"address_id": "wajib berupa UUID alamat milikmu",
		}))
		return
	}

	result, err := h.svc.Create(r.Context(), ordersvc.CreateInput{
		UserID:             user.ID,
		IdempotencyKey:     r.Header.Get("Idempotency-Key"),
		AddressID:          addressID,
		Courier:            req.Courier,
		CourierService:     req.CourierService,
		VoucherCode:        req.VoucherCode,
		RequestFingerprint: ordersvc.Fingerprint(req),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	status := nethttp.StatusCreated
	if !result.Created {
		// Replay kunci idempotensi: pesanan lama, bukan yang baru dibuat.
		status = nethttp.StatusOK
	}
	httpx.JSON(w, status, newOrderDTO(result.Order, result.Items))
}

// List menangani GET /orders.
func (h *Order) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	q := r.URL.Query()
	var status *order.Status
	if raw := q.Get("status"); raw != "" {
		s, err := ord.Parse(raw)
		if err != nil {
			httpx.Error(w, r, apperr.New(apperr.CodeAPIValidation, map[string]string{
				"status": "status tidak dikenal",
			}))
			return
		}
		status = &s
	}

	limit := int32(20)
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 50 {
			limit = int32(n)
		}
	}

	orders, err := h.svc.List(r.Context(), user.ID, status, limit)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	items := make([]orderDTO, 0, len(orders))
	for _, o := range orders {
		items = append(items, newOrderDTO(o, nil))
	}
	httpx.JSON(w, nethttp.StatusOK, map[string]any{"items": items})
}

// Detail menangani GET /orders/{id}.
func (h *Order) Detail(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, apperr.New(apperr.CodeOrdNotFound, nil))
		return
	}

	detail, err := h.svc.Get(r.Context(), user.ID, orderID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newOrderDTO(detail.Order, detail.Items))
}

type cancelOrderRequest struct {
	Reason string `json:"reason"`
}

// Cancel menangani POST /orders/{id}/cancel.
func (h *Order) Cancel(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, apperr.New(apperr.CodeOrdNotFound, nil))
		return
	}

	// Body opsional: pembatalan tanpa alasan tetap sah.
	var req cancelOrderRequest
	if r.ContentLength > 0 {
		if derr := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); derr != nil && !errors.Is(derr, errNoBody) {
			httpx.Error(w, r, derr)
			return
		}
	}

	detail, err := h.svc.Cancel(r.Context(), user.ID, orderID, req.Reason)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newOrderDTO(detail.Order, detail.Items))
}

// Pay menangani POST /orders/{id}/pay.
func (h *Order) Pay(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, apperr.New(apperr.CodeOrdNotFound, nil))
		return
	}

	session, err := h.svc.InitPayment(r.Context(), user.ID, orderID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, session)
}

// errNoBody menandai permintaan tanpa body pada endpoint yang body-nya opsional.
var errNoBody = errors.New("handler: body kosong")

var _ = sort.Strings
