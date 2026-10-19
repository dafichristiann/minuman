// Package middleware berisi middleware HTTP bersama (§10.2):
// request ID, recover, timeout, CORS, secure headers, rate limit.
//
// Urutan pemasangan penting dan ditetapkan di internal/http/router.go.
package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/platform/logger"
)

// HeaderRequestID adalah header yang membawa request ID ke/dari klien.
const HeaderRequestID = "X-Request-Id"

// ---------- request info (pola rute untuk log) ----------

type requestInfoKey struct{}

// requestInfo diisi oleh RoutePattern dan dibaca AccessLog. Dipakai objek
// bermutasi karena chi membuat *http.Request baru saat context berubah,
// sehingga nilai context tidak bisa "naik" kembali ke middleware luar.
type requestInfo struct {
	route string
}

// RequestInfoFrom mengambil informasi permintaan dari context.
func RequestInfoFrom(ctx context.Context) (string, bool) {
	info, ok := ctx.Value(requestInfoKey{}).(*requestInfo)
	if !ok || info == nil || info.route == "" {
		return "", false
	}
	return info.route, true
}

// RequestID memberi setiap permintaan identitas unik yang ikut tercatat di
// log, di respons error, dan (nanti) di audit_logs.request_id (§13.2).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(HeaderRequestID))
		// Terima hanya ID yang tampak wajar; selain itu buat sendiri agar klien
		// tidak bisa menyuntikkan nilai aneh ke log.
		if !isSafeRequestID(id) {
			id = uuid.NewString()
		}
		w.Header().Set(HeaderRequestID, id)
		ctx := logger.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isSafeRequestID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

// RoutePattern menyimpan pola rute chi (mis. "/api/v1/orders/{id}") sehingga log
// dan audit tidak memuat ID asli. Pola baru tersedia setelah chi mencocokkan
// rute, jadi middleware ini harus dipasang pada router (bukan sebelum router).
func RoutePattern(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		info, _ := ctx.Value(requestInfoKey{}).(*requestInfo)

		if rctx := chi.RouteContext(ctx); rctx != nil {
			if pattern := rctx.RoutePattern(); pattern != "" {
				if info != nil {
					info.route = pattern
				}
				ctx = logger.WithRoute(ctx, pattern)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// responseRecorder mencatat status dan ukuran respons untuk log akses.
type responseRecorder struct {
	http.ResponseWriter
	status  int
	bytes   int64
	written bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.written {
		return
	}
	r.status = status
	r.written = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.status = http.StatusOK
		r.written = true
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

// Flush mendukung SSE (§8.7): tanpa ini, streaming tidak akan terkirim.
func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap memungkinkan http.ResponseController menembus ke writer asli.
func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// AccessLog mencatat setiap permintaan sebagai log terstruktur.
// URL mentah TIDAK dicatat (bisa memuat token di query); yang dicatat pola rute.
func AccessLog(log *slog.Logger, trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			info := &requestInfo{}
			ctx := context.WithValue(r.Context(), requestInfoKey{}, info)
			ctx = logger.WithLogger(ctx, log)
			r = r.WithContext(ctx)

			next.ServeHTTP(rec, r)

			route := info.route
			if route == "" {
				route = "unmatched"
			}

			level := slog.LevelInfo
			switch {
			case rec.status >= 500:
				level = slog.LevelError
			case rec.status >= 400:
				level = slog.LevelWarn
			}

			log.LogAttrs(r.Context(), level, "http_request",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", rec.status),
				slog.Int64("bytes", rec.bytes),
				slog.Duration("duration", time.Since(start)),
				slog.String("ip", ClientIP(r, trustProxy)),
				slog.String("user_agent", truncate(r.UserAgent(), 200)),
			)
		})
	}
}

// Recoverer menangkap panic agar satu permintaan buruk tidak mematikan server.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// http.ErrAbortHandler adalah sinyal internal net/http untuk
					// membatalkan respons; jangan dianggap kegagalan tak terduga.
					if rec == http.ErrAbortHandler {
						panic(rec)
					}
					log.ErrorContext(r.Context(), "panic tertangkap",
						"panic", rec,
						"stack", string(debug.Stack()),
					)
					httpx.Error(w, r, apperr.New(apperr.CodeAPIServerError, nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBody menolak body yang lebih besar dari batas sebelum handler membacanya.
func MaxBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit > 0 && r.ContentLength > limit {
				httpx.Error(w, r, apperr.New(apperr.CodeSysUploadTooBig, map[string]string{
					"n": strconv.FormatInt(limit/(1<<20), 10),
				}))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// NoStore mencegah cache pada respons yang memuat data pribadi
// (profil, keranjang, notifikasi).
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		next.ServeHTTP(w, r)
	})
}

// ClientIP mengembalikan alamat IP klien. Header X-Forwarded-For hanya dipakai
// bila `trustProxy` true, karena header itu bisa dipalsukan klien.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Elemen paling kiri adalah klien asli menurut proxy terdekat.
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return normalizeIP(ip)
			}
		}
		if xrip := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xrip != "" {
			return normalizeIP(xrip)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return normalizeIP(r.RemoteAddr)
	}
	return normalizeIP(host)
}

func normalizeIP(raw string) string {
	if ip := net.ParseIP(raw); ip != nil {
		return ip.String()
	}
	return "unknown"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
