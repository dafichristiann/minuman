package handler

import (
	nethttp "net/http"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/http/middleware"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/service/auth"
)

// Auth menangani seluruh endpoint §12.1 yang tidak memerlukan sesi, plus
// logout & ganti password.
type Auth struct {
	cfg *config.Config
	svc *auth.Service
}

// NewAuth membuat handler autentikasi.
func NewAuth(cfg *config.Config, svc *auth.Service) *Auth {
	return &Auth{cfg: cfg, svc: svc}
}

type registerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	User      userDTO `json:"user"`
	CSRFToken string  `json:"csrf_token"`
}

// Register menangani POST /auth/register.
func (h *Auth) Register(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	result, err := h.svc.Register(r.Context(), auth.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FullName:  req.FullName,
		Phone:     req.Phone,
		BirthDate: req.BirthDate,
		UserAgent: r.UserAgent(),
		IP:        middleware.ClientIP(r, h.cfg.HTTP.TrustProxyHeaders),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	body := map[string]any{"user": newUserDTO(result.User)}
	if result.DevVerifyToken != "" {
		// Hanya di lingkungan non-produksi (lihat auth.Options.ExposeDevTokens).
		body["dev_verify_token"] = result.DevVerifyToken
	}
	httpx.JSON(w, nethttp.StatusCreated, body)
}

// Login menangani POST /auth/login.
func (h *Auth) Login(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	session, err := h.svc.Login(r.Context(), auth.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: r.UserAgent(),
		IP:        middleware.ClientIP(r, h.cfg.HTTP.TrustProxyHeaders),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	setSessionCookies(w, h.cfg, session)
	httpx.JSON(w, nethttp.StatusOK, sessionResponse{
		User:      newUserDTO(session.User),
		CSRFToken: session.CSRFToken,
	})
}

// Refresh menangani POST /auth/refresh. Body tidak dipakai: token dibaca dari cookie.
func (h *Auth) Refresh(w nethttp.ResponseWriter, r *nethttp.Request) {
	cookie, _ := r.Cookie(h.cfg.Auth.RefreshCookieName)
	token := ""
	if cookie != nil {
		token = cookie.Value
	}

	session, err := h.svc.Refresh(r.Context(), auth.RefreshInput{
		RefreshToken: token,
		UserAgent:    r.UserAgent(),
		IP:           middleware.ClientIP(r, h.cfg.HTTP.TrustProxyHeaders),
	})
	if err != nil {
		clearSessionCookies(w, h.cfg)
		httpx.Error(w, r, err)
		return
	}

	setSessionCookies(w, h.cfg, session)
	httpx.JSON(w, nethttp.StatusOK, sessionResponse{
		User:      newUserDTO(session.User),
		CSRFToken: session.CSRFToken,
	})
}

// Logout menangani POST /auth/logout.
func (h *Auth) Logout(w nethttp.ResponseWriter, r *nethttp.Request) {
	cookie, _ := r.Cookie(h.cfg.Auth.RefreshCookieName)
	token := ""
	if cookie != nil {
		token = cookie.Value
	}

	if err := h.svc.Logout(r.Context(), token); err != nil {
		httpx.Error(w, r, err)
		return
	}
	clearSessionCookies(w, h.cfg)
	httpx.NoContent(w)
}

// LogoutAll menangani POST /auth/logout-all.
func (h *Auth) LogoutAll(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	if err := h.svc.LogoutAll(r.Context(), user.ID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	clearSessionCookies(w, h.cfg)
	httpx.NoContent(w)
}

type tokenRequest struct {
	Token string `json:"token"`
}

// VerifyEmail menangani POST /auth/verify-email.
func (h *Auth) VerifyEmail(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req tokenRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.VerifyEmail(r.Context(), req.Token); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

type emailRequest struct {
	Email string `json:"email"`
}

// ResendVerification menangani POST /auth/verify-email/resend.
//
// Respons SELALU 202 agar endpoint tidak bisa dipakai memetakan email terdaftar.
func (h *Auth) ResendVerification(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req emailRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}
	token, err := h.svc.ResendVerification(r.Context(), req.Email)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	body := map[string]any{
		"message": "Kalau email itu terdaftar dan belum terverifikasi, tautan baru sudah dikirim.",
	}
	if token != "" && !h.cfg.IsProduction() {
		body["dev_verify_token"] = token
	}
	httpx.JSON(w, nethttp.StatusAccepted, body)
}

// ForgotPassword menangani POST /auth/forgot-password.
//
// Respons SELALU 202 dengan body sama, baik email terdaftar maupun tidak
// (§12.1: "Respons selalu sama").
func (h *Auth) ForgotPassword(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req emailRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}
	token, err := h.svc.ForgotPassword(r.Context(), req.Email)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	body := map[string]any{
		"message": "Kalau email itu terdaftar, tautan reset sudah dikirim.",
	}
	if token != "" && !h.cfg.IsProduction() {
		body["dev_reset_token"] = token
	}
	httpx.JSON(w, nethttp.StatusAccepted, body)
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPassword menangani POST /auth/reset-password.
func (h *Auth) ResetPassword(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req resetPasswordRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword menangani POST /me/password. Sesi lain dicabut; perangkat ini
// menerima sesi baru sehingga pengguna tidak terlempar keluar.
func (h *Auth) ChangePassword(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	var req changePasswordRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	session, err := h.svc.ChangePassword(r.Context(), auth.ChangePasswordInput{
		UserID:    user.ID,
		Current:   req.CurrentPassword,
		New:       req.NewPassword,
		UserAgent: r.UserAgent(),
		IP:        middleware.ClientIP(r, h.cfg.HTTP.TrustProxyHeaders),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	setSessionCookies(w, h.cfg, session)
	httpx.JSON(w, nethttp.StatusOK, sessionResponse{
		User:      newUserDTO(session.User),
		CSRFToken: session.CSRFToken,
	})
}
