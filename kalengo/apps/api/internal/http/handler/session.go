package handler

import (
	nethttp "net/http"
	"time"

	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/service/auth"
)

// Cookie §13.1: HttpOnly + Secure + SameSite=Lax.
//
// Refresh token dibatasi pada path /api/v1/auth sehingga tidak ikut terkirim
// pada setiap permintaan biasa — memperkecil paparan bila ada XSS di halaman
// lain pada domain yang sama.
const refreshCookiePath = "/api/v1/auth"

func setSessionCookies(w nethttp.ResponseWriter, cfg *config.Config, s auth.Session) {
	sameSite := nethttp.SameSiteLaxMode

	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:     cfg.Auth.CookieName,
		Value:    s.AccessToken,
		Path:     "/",
		Domain:   cfg.Auth.CookieDomain,
		Expires:  s.AccessExpiresAt,
		MaxAge:   secondsUntil(s.AccessExpiresAt),
		HttpOnly: true,
		Secure:   cfg.Auth.CookieSecure,
		SameSite: sameSite,
	})

	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:     cfg.Auth.RefreshCookieName,
		Value:    s.RefreshToken,
		Path:     refreshCookiePath,
		Domain:   cfg.Auth.CookieDomain,
		Expires:  s.RefreshExpiresAt,
		MaxAge:   secondsUntil(s.RefreshExpiresAt),
		HttpOnly: true,
		Secure:   cfg.Auth.CookieSecure,
		SameSite: sameSite,
	})

	// Token CSRF HARUS bisa dibaca JavaScript (double-submit), jadi HttpOnly=false.
	// Isinya bukan rahasia sesi: hanya pembanding untuk header X-CSRF-Token.
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:     cfg.Auth.CSRFCookieName,
		Value:    s.CSRFToken,
		Path:     "/",
		Domain:   cfg.Auth.CookieDomain,
		Expires:  s.RefreshExpiresAt,
		MaxAge:   secondsUntil(s.RefreshExpiresAt),
		HttpOnly: false,
		Secure:   cfg.Auth.CookieSecure,
		SameSite: sameSite,
	})
}

func clearSessionCookies(w nethttp.ResponseWriter, cfg *config.Config) {
	past := time.Unix(0, 0).UTC()
	for _, c := range []struct {
		name     string
		path     string
		httpOnly bool
	}{
		{cfg.Auth.CookieName, "/", true},
		{cfg.Auth.RefreshCookieName, refreshCookiePath, true},
		{cfg.Auth.CSRFCookieName, "/", false},
	} {
		nethttp.SetCookie(w, &nethttp.Cookie{
			Name:     c.name,
			Value:    "",
			Path:     c.path,
			Domain:   cfg.Auth.CookieDomain,
			Expires:  past,
			MaxAge:   -1,
			HttpOnly: c.httpOnly,
			Secure:   cfg.Auth.CookieSecure,
			SameSite: nethttp.SameSiteLaxMode,
		})
	}
}

func secondsUntil(t time.Time) int {
	n := int(time.Until(t).Seconds())
	if n < 1 {
		return 1
	}
	return n
}

// userDTO adalah bentuk publik profil (§12.1). Password hash, penghitung
// gagal login, dan kolom internal lain TIDAK pernah ikut.
type userDTO struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	FullName      string  `json:"full_name"`
	Phone         *string `json:"phone"`
	Role          string  `json:"role"`
	EmailVerified bool    `json:"email_verified"`
	BirthDate     *string `json:"birth_date"`
	CreatedAt     string  `json:"created_at"`
}

func newUserDTO(u domain.User) userDTO {
	var birth *string
	if u.BirthDate != nil {
		formatted := u.BirthDate.Format("2006-01-02")
		birth = &formatted
	}
	return userDTO{
		ID:            u.ID.String(),
		Email:         u.Email,
		FullName:      u.FullName,
		Phone:         u.Phone,
		Role:          u.Role,
		EmailVerified: u.EmailVerified(),
		BirthDate:     birth,
		CreatedAt:     u.CreatedAt.UTC().Format(time.RFC3339),
	}
}
