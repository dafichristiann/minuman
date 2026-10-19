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

// Me menangani GET/PATCH/DELETE /me (§12.1).
type Me struct {
	cfg *config.Config
	svc *account.Service
}

// NewMe membuat handler profil.
func NewMe(cfg *config.Config, svc *account.Service) *Me {
	return &Me{cfg: cfg, svc: svc}
}

// Get menangani GET /me.
func (h *Me) Get(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	profile, err := h.svc.Me(r.Context(), user.ID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newUserDTO(profile))
}

type patchMeRequest struct {
	FullName  httpx.Optional[string] `json:"full_name"`
	Phone     httpx.Optional[string] `json:"phone"`
	BirthDate httpx.Optional[string] `json:"birth_date"`
}

// Patch menangani PATCH /me.
//
// Body memakai httpx.Optional agar `"phone": null` berarti MENGOSONGKAN,
// sedangkan field yang tidak dikirim berarti "jangan diubah".
func (h *Me) Patch(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	var req patchMeRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	updated, err := h.svc.UpdateProfile(r.Context(), user.ID, account.ProfilePatch{
		FullName:  fieldFrom(req.FullName),
		Phone:     fieldFrom(req.Phone),
		BirthDate: fieldFrom(req.BirthDate),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newUserDTO(updated))
}

type deleteMeRequest struct {
	Confirmation string `json:"confirmation"`
}

// Delete menangani DELETE /me (anonimisasi, bukan penghapusan baris).
func (h *Me) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	var req deleteMeRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.DeleteAccount(r.Context(), user.ID, req.Confirmation); err != nil {
		httpx.Error(w, r, err)
		return
	}
	clearSessionCookies(w, h.cfg)
	httpx.NoContent(w)
}

// fieldFrom menerjemahkan httpx.Optional (lapisan HTTP) ke domain.Field
// (lapisan domain) tanpa membocorkan tipe HTTP ke service.
func fieldFrom[T any](o httpx.Optional[T]) domain.Field[T] {
	return domain.Field[T]{Set: o.Set, Value: o.Value}
}
