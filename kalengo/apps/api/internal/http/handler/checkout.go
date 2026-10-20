package handler

import (
	nethttp "net/http"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/http/middleware"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/platform/shipping"
	ordersvc "github.com/kalengo/api/internal/service/order"
)

// Checkout menangani endpoint pendukung checkout: tarif ongkir & voucher (§12.4).
type Checkout struct {
	cfg *config.Config
	svc *ordersvc.Service
}

// NewCheckout membuat handler checkout.
func NewCheckout(cfg *config.Config, svc *ordersvc.Service) *Checkout {
	return &Checkout{cfg: cfg, svc: svc}
}

type shippingRatesRequest struct {
	AddressID string `json:"address_id"`
	Courier   string `json:"courier"`
}

type shippingRateDTO struct {
	Courier     string `json:"courier"`
	Service     string `json:"service"`
	Description string `json:"description"`
	Cost        int64  `json:"cost"`
	ETD         string `json:"etd"`
}

// ShippingRates menangani POST /shipping/rates.
func (h *Checkout) ShippingRates(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	var req shippingRatesRequest
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

	rates, err := h.svc.ShippingOptions(r.Context(), user.ID, addressID, req.Courier)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	items := make([]shippingRateDTO, 0, len(rates))
	for _, rt := range rates {
		items = append(items, shippingRateDTO{
			Courier:     rt.Courier,
			Service:     rt.Service,
			Description: rt.Description,
			Cost:        rt.Cost,
			ETD:         rt.ETD,
		})
	}
	httpx.JSON(w, nethttp.StatusOK, map[string]any{"items": items})
}

type voucherValidateRequest struct {
	Code string `json:"code"`
}

// ValidateVoucher menangani POST /vouchers/validate.
func (h *Checkout) ValidateVoucher(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	var req voucherValidateRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	info, err := h.svc.ValidateVoucher(r.Context(), user.ID, req.Code)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, info)
}

var _ shipping.Rate
