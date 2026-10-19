package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/repo"
	"github.com/kalengo/api/internal/security"
)

// AuthUser adalah identitas pemanggil yang sudah diverifikasi.
type AuthUser struct {
	ID            uuid.UUID
	Role          string
	Email         string
	EmailVerified bool
}

type authUserKey struct{}

// UserFrom mengambil identitas pemanggil dari context.
func UserFrom(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(authUserKey{}).(AuthUser)
	return u, ok
}

// UserIDFrom mengambil id pemanggil atau string kosong (dipakai KeyByUser).
func UserIDFrom(r *http.Request) string {
	if u, ok := UserFrom(r.Context()); ok {
		return u.ID.String()
	}
	return ""
}

// UserLoader mengambil pengguna dari database. Middleware memuat ulang setiap
// request supaya perubahan peran, penghapusan akun, dan pencabutan sesi
// langsung berlaku — JWT berumur 15 menit tidak boleh jadi sumber otoritas.
type UserLoader interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
}

// RequireAuth menolak permintaan tanpa access token yang valid.
func RequireAuth(cfg *config.Config, tokens *security.TokenService, users UserLoader) func(http.Handler) http.Handler {
	return authenticate(cfg, tokens, users, false)
}

// OptionalAuth meneruskan permintaan tanpa identitas bila token tidak ada.
// Token yang ADA tetapi tidak valid tetap ditolak: klien harus tahu sesinya
// berakhir, bukan diam-diam dianggap tamu.
func OptionalAuth(cfg *config.Config, tokens *security.TokenService, users UserLoader) func(http.Handler) http.Handler {
	return authenticate(cfg, tokens, users, true)
}

func authenticate(cfg *config.Config, tokens *security.TokenService, users UserLoader, optional bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cfg.Auth.CookieName)
			if err != nil || cookie.Value == "" {
				if optional {
					next.ServeHTTP(w, r)
					return
				}
				unauthorized(w, r)
				return
			}

			claims, err := tokens.ParseAccess(cookie.Value)
			if err != nil {
				// Kedaluwarsa maupun tidak valid sama-sama berarti "sesi
				// berakhir" bagi klien: panggil /auth/refresh.
				unauthorized(w, r)
				return
			}

			user, err := users.GetUserByID(r.Context(), claims.UserID)
			if err != nil {
				if errors.Is(err, repo.ErrNotFound) {
					unauthorized(w, r)
					return
				}
				unavailable(w, r, err)
				return
			}

			ctx := context.WithValue(r.Context(), authUserKey{}, AuthUser{
				ID:            user.ID,
				Role:          user.Role,
				Email:         user.Email,
				EmailVerified: user.EmailVerified(),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole membatasi rute ke peran tertentu (§12.7). Dipasang SETELAH
// RequireAuth. Penolakan memakai API_FORBIDDEN, bukan 404, karena rute admin
// memang diketahui publik.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFrom(r.Context())
			if !ok {
				unauthorized(w, r)
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				httpx.Error(w, r, apperr.New(apperr.CodeAPIForbidden, nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireVerifiedEmail menolak aksi yang menuntut email terverifikasi.
// Dipakai pada checkout (M2); akun belum terverifikasi tetap boleh berbelanja.
func RequireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok {
			unauthorized(w, r)
			return
		}
		if !user.EmailVerified {
			httpx.Error(w, r, apperr.New(apperr.CodeAuthEmailUnverified, nil))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CSRF menerapkan pola double-submit cookie (§13.1).
//
// Cookie CSRF sengaja TIDAK HttpOnly (JavaScript harus membacanya), sehingga
// nilainya bisa dicocokkan dengan header X-CSRF-Token. Penyerang lintas situs
// bisa mengirim cookie, tetapi tidak bisa membaca nilainya untuk mengisi header.
func CSRF(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(cfg.Auth.CSRFCookieName)
			if err != nil {
				httpx.Error(w, r, apperr.New(apperr.CodeAuthCSRFInvalid, nil))
				return
			}
			header := r.Header.Get(cfg.Auth.CSRFHeaderName)
			if !security.ValidCSRF(cookie.Value, header) {
				httpx.Error(w, r, apperr.New(apperr.CodeAuthCSRFInvalid, nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// unauthorized menjawab permintaan tanpa sesi valid dengan AUTH_SESSION_EXPIRED
// (pesan registry sudah menyebut bahwa keranjang tetap aman).
func unauthorized(w http.ResponseWriter, r *http.Request) {
	httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
}

// unavailable dipakai saat pemuatan pengguna gagal karena masalah server,
// bukan karena sesinya tidak valid.
func unavailable(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Error(w, r, apperr.Wrap(err, apperr.CodeAPIServerError, nil))
}
