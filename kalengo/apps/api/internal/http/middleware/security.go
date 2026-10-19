package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/cors"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/platform/logger"
	"github.com/kalengo/api/internal/platform/ratelimit"
)

// SecurityHeaders memasang header keamanan wajib (§13.1).
//
// API hanya mengembalikan JSON, jadi CSP bisa sangat ketat: `default-src 'none'`.
// CSP untuk HTML (React SPA) dipasang oleh Nginx — lihat docker/web/nginx.conf.
func SecurityHeaders(isProduction bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
			h.Set("Cross-Origin-Resource-Policy", "same-site")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			// HSTS hanya bermakna di HTTPS; jangan dipasang di dev agar
			// localhost tidak "terkunci" di browser pengembang.
			if isProduction {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS memasang whitelist domain eksplisit (§13.1). Tidak ada wildcard `*`
// karena autentikasi memakai cookie (AllowCredentials).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept", "Authorization", "Content-Type",
			HeaderRequestID, "X-CSRF-Token", "Idempotency-Key",
		},
		ExposedHeaders:   []string{HeaderRequestID, "Retry-After"},
		AllowCredentials: true,
		MaxAge:           600,
		Debug:            false,
	})
}

// KeyFunc menentukan kunci pembatas laju untuk sebuah permintaan.
type KeyFunc func(r *http.Request) string

// KeyByIP membatasi berdasarkan IP klien.
func KeyByIP(trustProxy bool) KeyFunc {
	return func(r *http.Request) string {
		return "ip:" + ClientIP(r, trustProxy)
	}
}

// KeyByUser membatasi berdasarkan pengguna; jatuh ke IP bila belum login.
// ID pengguna diambil dari context (diisi middleware autentikasi pada M1).
func KeyByUser(userIDFrom func(*http.Request) string, trustProxy bool) KeyFunc {
	return func(r *http.Request) string {
		if uid := userIDFrom(r); uid != "" {
			return "user:" + uid
		}
		return "ip:" + ClientIP(r, trustProxy)
	}
}

// RateLimit membatasi laju permintaan memakai sliding window Redis (§12.8).
//
// Bila Redis mati, permintaan DIBIARKAN lewat (fail-open) tetapi dicatat sebagai
// error: ketersediaan toko lebih penting daripada penegakan batas saat gangguan.
func RateLimit(limiter *ratelimit.Limiter, key KeyFunc, limit int, window time.Duration, skipSSE bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			// Streaming SSE berumur panjang: jangan dihitung sebagai trafik biasa.
			if skipSSE && r.Header.Get("Accept") == "text/event-stream" {
				next.ServeHTTP(w, r)
				return
			}

			result, err := limiter.Allow(r.Context(), key(r), limit, window)
			if err != nil {
				log := logger.FromContext(r.Context())
				log.ErrorContext(r.Context(), "rate limiter gagal, permintaan diteruskan (fail-open)", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

			if !result.Allowed {
				seconds := int(result.RetryAfter.Seconds())
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				httpx.Error(w, r, apperr.New(apperr.CodeAPIRateLimit, map[string]string{
					"detik": strconv.Itoa(seconds),
				}))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
