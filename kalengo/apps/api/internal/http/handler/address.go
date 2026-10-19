package handler

import (
	nethttp "net/http"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/http/middleware"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/service/account"
)

// Address menangani buku alamat (§12.1).
type Address struct {
	cfg *config.Config
	svc *account.Service
}

// NewAddress membuat handler alamat.
func NewAddress(cfg *config.Config, svc *account.Service) *Address {
	return &Address{cfg: cfg, svc: svc}
}

// addressDTO adalah bentuk publik satu alamat.
type addressDTO struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Recipient  string  `json:"recipient"`
	Phone      string  `json:"phone"`
	Province   string  `json:"province"`
	City       string  `json:"city"`
	District   string  `json:"district"`
	PostalCode string  `json:"postal_code"`
	Street     string  `json:"street"`
	Notes      *string `json:"notes"`
	AreaID     *string `json:"area_id"`
	IsDefault  bool    `json:"is_default"`
}

func newAddressDTO(a domain.Address) addressDTO {
	return addressDTO{
		ID:         a.ID.String(),
		Label:      a.Label,
		Recipient:  a.Recipient,
		Phone:      a.Phone,
		Province:   a.Province,
		City:       a.City,
		District:   a.District,
		PostalCode: a.PostalCode,
		Street:     a.Street,
		Notes:      a.Notes,
		AreaID:     a.AreaID,
		IsDefault:  a.IsDefault,
	}
}

type addressRequest struct {
	Label      string `json:"label"`
	Recipient  string `json:"recipient"`
	Phone      string `json:"phone"`
	Province   string `json:"province"`
	City       string `json:"city"`
	District   string `json:"district"`
	PostalCode string `json:"postal_code"`
	Street     string `json:"street"`
	Notes      string `json:"notes"`
	AreaID     string `json:"area_id"`
	IsDefault  bool   `json:"is_default"`
}

func (req addressRequest) toInput() account.AddressInput {
	return account.AddressInput{
		Label:      req.Label,
		Recipient:  req.Recipient,
		Phone:      req.Phone,
		Province:   req.Province,
		City:       req.City,
		District:   req.District,
		PostalCode: req.PostalCode,
		Street:     req.Street,
		Notes:      req.Notes,
		AreaID:     req.AreaID,
		IsDefault:  req.IsDefault,
	}
}

// List menangani GET /addresses.
func (h *Address) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	items, err := h.svc.ListAddresses(r.Context(), user.ID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]addressDTO, 0, len(items))
	for _, a := range items {
		out = append(out, newAddressDTO(a))
	}
	httpx.JSON(w, nethttp.StatusOK, out)
}

// Create menangani POST /addresses.
func (h *Address) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	var req addressRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}
	created, err := h.svc.CreateAddress(r.Context(), user.ID, req.toInput())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusCreated, newAddressDTO(created))
}

// Update menangani PATCH /addresses/{id}.
func (h *Address) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	id, ok := uuidParam(w, r, "id", "id")
	if !ok {
		return
	}
	var req addressRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}
	updated, err := h.svc.UpdateAddress(r.Context(), user.ID, id, req.toInput())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newAddressDTO(updated))
}

// Delete menangani DELETE /addresses/{id}.
func (h *Address) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	id, ok := uuidParam(w, r, "id", "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteAddress(r.Context(), user.ID, id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// SetDefault menangani POST /addresses/{id}/default.
func (h *Address) SetDefault(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	id, ok := uuidParam(w, r, "id", "id")
	if !ok {
		return
	}
	updated, err := h.svc.SetDefaultAddress(r.Context(), user.ID, id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newAddressDTO(updated))
}
