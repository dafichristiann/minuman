// Package logger menyiapkan log terstruktur (slog) sesuai §10.2:
// JSON + request ID + user ID, tanpa PII sensitif.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ctxKey int

const (
	loggerKey ctxKey = iota
	requestIDKey
	userIDKey
	routeKey
)

// Options mengatur pembuatan logger.
type Options struct {
	// Level minimum yang dicatat.
	Level slog.Level
	// Pretty memakai handler teks (enak dibaca di terminal). Produksi selalu JSON.
	Pretty bool
	// Service menandai asal log: "api" atau "worker".
	Service string
	// Version adalah versi build yang dilaporkan pada setiap baris.
	Version string
	// Output tujuan log; nil berarti os.Stdout.
	Output io.Writer
}

// New membuat logger akar. Selalu dibungkus contextHandler agar request_id dan
// user_id otomatis ikut pada setiap baris bila tersedia di context.
func New(opts Options) *slog.Logger {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   opts.Level <= slog.LevelDebug,
		ReplaceAttr: redact,
	}

	var base slog.Handler
	if opts.Pretty {
		handlerOpts.ReplaceAttr = chain(redact, tintAttr)
		base = slog.NewTextHandler(out, handlerOpts)
	} else {
		base = slog.NewJSONHandler(out, handlerOpts)
	}

	attrs := []any{slog.String("service", opts.Service)}
	if opts.Version != "" {
		attrs = append(attrs, slog.String("version", opts.Version))
	}

	return slog.New(contextHandler{Handler: base}).With(attrs...)
}

// contextHandler menyuntikkan nilai dari context ke setiap rekaman log.
type contextHandler struct{ slog.Handler }

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if v, ok := RequestIDFrom(ctx); ok {
		r.AddAttrs(slog.String("request_id", v))
	}
	if v, ok := UserIDFrom(ctx); ok {
		r.AddAttrs(slog.String("user_id", v))
	}
	if v, ok := RouteFrom(ctx); ok {
		r.AddAttrs(slog.String("route", v))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name)}
}

// WithLogger menyimpan logger ke context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext mengambil logger dari context; jatuh ke slog.Default bila tidak ada.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// WithRequestID menyimpan request ID ke context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFrom membaca request ID dari context.
func RequestIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey).(string)
	return v, ok && v != ""
}

// WithUserID menyimpan ID pengguna terautentikasi ke context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFrom membaca ID pengguna dari context.
func UserIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok && v != ""
}

// WithRoute menyimpan pola rute (bukan URL mentah, agar tidak ada PII di label) ke context.
func WithRoute(ctx context.Context, route string) context.Context {
	return context.WithValue(ctx, routeKey, route)
}

// RouteFrom membaca pola rute dari context.
func RouteFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(routeKey).(string)
	return v, ok && v != ""
}

// sensitiveKeys adalah kunci yang isinya TIDAK boleh masuk log (§13.1).
var sensitiveKeys = map[string]struct{}{
	"password": {}, "passwd": {}, "password_hash": {}, "passwordhash": {},
	"token": {}, "access_token": {}, "refresh_token": {}, "id_token": {},
	"authorization": {}, "cookie": {}, "set-cookie": {}, "csrf": {}, "csrf_token": {},
	"jwt": {}, "secret": {}, "client_secret": {}, "api_key": {}, "apikey": {},
	"server_key": {}, "private_key": {}, "signature": {}, "signature_key": {},
	"card_number": {}, "cvv": {}, "pin": {}, "otp": {},
	"nik": {}, "npwp": {}, "p256dh": {}, "auth": {},
}

// redact menimpa nilai atribut sensitif dengan "[REDACTED]".
func redact(_ []string, a slog.Attr) slog.Attr {
	if _, bad := sensitiveKeys[strings.ToLower(a.Key)]; bad {
		return slog.String(a.Key, "[REDACTED]")
	}
	// Email & nomor telepon disamarkan sebagian agar tetap bisa ditelusuri
	// tanpa membocorkan identitas penuh.
	switch strings.ToLower(a.Key) {
	case "email":
		return slog.String(a.Key, MaskEmail(a.Value.String()))
	case "phone", "phone_number", "recipient_phone":
		return slog.String(a.Key, MaskPhone(a.Value.String()))
	}
	return a
}

// MaskEmail menyamarkan email: "dinda@mail.com" -> "di***@mail.com".
func MaskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "[invalid-email]"
	}
	local, domain := email[:at], email[at:]
	if len(local) <= 2 {
		return "***" + domain
	}
	return local[:2] + "***" + domain
}

// MaskPhone menyamarkan nomor telepon: "081234567890" -> "0812****890".
func MaskPhone(phone string) string {
	if len(phone) <= 6 {
		return "***"
	}
	return phone[:4] + "****" + phone[len(phone)-3:]
}

func chain(fns ...func([]string, slog.Attr) slog.Attr) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		for _, fn := range fns {
			a = fn(groups, a)
		}
		return a
	}
}

// tintAttr memberi warna sederhana pada level di mode pretty.
func tintAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key != slog.LevelKey {
		return a
	}
	level, ok := a.Value.Any().(slog.Level)
	if !ok {
		return a
	}
	var color string
	switch {
	case level < slog.LevelInfo:
		color = "\x1b[36m" // cyan
	case level < slog.LevelWarn:
		color = "\x1b[32m" // hijau
	case level < slog.LevelError:
		color = "\x1b[33m" // kuning
	default:
		color = "\x1b[31m" // merah
	}
	return slog.String(slog.LevelKey, color+level.String()+"\x1b[0m")
}
