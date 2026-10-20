package handler

import (
	"io"
	nethttp "net/http"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/platform/httpx"
	ordersvc "github.com/kalengo/api/internal/service/order"
)

// Webhook menangani notifikasi dari penyedia pembayaran & pengiriman (§12.5).
//
// Endpoint ini SENGAJA berada di luar autentikasi & CSRF: pemanggilnya adalah
// penyedia, bukan pengguna. Keamanannya bertumpu pada verifikasi tanda tangan
// di lapisan service, bukan pada sesi.
type Webhook struct {
	svc *ordersvc.Service
}

// NewWebhook membuat handler webhook.
func NewWebhook(svc *ordersvc.Service) *Webhook {
	return &Webhook{svc: svc}
}

// readRaw membaca body mentah.
//
// Body TIDAK di-decode lewat DecodeJSON: payload penyedia memuat puluhan field
// yang tidak kita kenal, dan tanda tangan dihitung atas byte aslinya.
func readRaw(w nethttp.ResponseWriter, r *nethttp.Request, limit int64) ([]byte, bool) {
	if limit <= 0 {
		limit = 1 << 20
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		httpx.Error(w, r, apperr.New(apperr.CodeAPIBadRequest, nil))
		return nil, false
	}
	if int64(len(raw)) > limit {
		httpx.Error(w, r, apperr.New(apperr.CodeSysUploadTooBig, nil))
		return nil, false
	}
	if len(raw) == 0 {
		httpx.Error(w, r, apperr.New(apperr.CodeAPIBadRequest, nil))
		return nil, false
	}
	return raw, true
}

// Midtrans menangani POST /webhooks/midtrans.
func (h *Webhook) Midtrans(w nethttp.ResponseWriter, r *nethttp.Request) {
	raw, ok := readRaw(w, r, 1<<20)
	if !ok {
		return
	}

	result, err := h.svc.HandlePaymentWebhook(r.Context(), raw)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	// Replay maupun event yang tidak mengubah status tetap dijawab 200 supaya
	// penyedia berhenti mengirim ulang.
	httpx.JSON(w, nethttp.StatusOK, map[string]any{
		"received":  true,
		"duplicate": result.Duplicate,
		"applied":   result.Applied,
		"status":    result.Status,
	})
}

// Shipping menangani POST /webhooks/shipping.
func (h *Webhook) Shipping(w nethttp.ResponseWriter, r *nethttp.Request) {
	raw, ok := readRaw(w, r, 1<<20)
	if !ok {
		return
	}

	result, err := h.svc.HandleShippingWebhook(r.Context(), raw)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, map[string]any{
		"received":  true,
		"duplicate": result.Duplicate,
		"status":    result.Status,
	})
}
