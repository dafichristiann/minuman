// Package apperr adalah satu-satunya cara backend membuat error yang dikirim
// ke klien (§12). `error.code` WAJIB sama dengan kode di katalog notifikasi
// frontend (§8.6) supaya pesan di UI seragam dan mudah diubah.
//
// Aturan: handler TIDAK membuat pesan error sendiri. Semua pesan berasal dari
// registry di codes.go, dirender dengan variabel.
package apperr

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

// Severity mencocokkan kolom Severity di §8.6.
type Severity string

const (
	SeveritySuccess  Severity = "success"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Category mengelompokkan kode seperti huruf A–G di §8.6.
type Category string

const (
	CategoryAuth     Category = "auth"
	CategoryAccount  Category = "account"
	CategoryCatalog  Category = "catalog"
	CategoryVoucher  Category = "voucher"
	CategoryCheckout Category = "checkout"
	CategoryPayment  Category = "payment"
	CategoryOrder    Category = "order"
	CategorySystem   Category = "system"
	CategoryAdmin    Category = "admin"
)

// Definition adalah metadata satu kode error.
type Definition struct {
	Code     string
	Status   int
	Severity Severity
	Category Category
	// Message memakai placeholder {nama}. Lihat Render.
	Message string
	// Persist menandakan notifikasi ini juga masuk Notification Center (§8.2).
	Persist bool
}

// Error adalah error terstruktur yang siap diserialisasi ke klien.
type Error struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Fields   map[string]string `json:"fields,omitempty"`
	Status   int               `json:"-"`
	Severity Severity          `json:"-"`
	Category Category          `json:"-"`
	Persist  bool              `json:"-"`
	Meta     map[string]any    `json:"-"`
	cause    error
}

// Error memenuhi interface error. Formatnya berguna di log; klien menerima
// bentuk JSON dari httpx, bukan string ini.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s (%d): %s: %v", e.Code, e.Status, e.Message, e.cause)
	}
	return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message)
}

// Unwrap mendukung errors.Is/As ke error asal.
func (e *Error) Unwrap() error { return e.cause }

// LogValue membuat error tampil rapi di log terstruktur.
func (e *Error) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("code", e.Code),
		slog.Int("status", e.Status),
		slog.String("severity", string(e.Severity)),
		slog.String("message", e.Message),
	)
}

// WithField menambahkan satu pesan validasi per field (dipakai §8.6 API_VALIDATION).
func (e *Error) WithField(field, message string) *Error {
	if e.Fields == nil {
		e.Fields = make(map[string]string, 1)
	}
	e.Fields[field] = message
	return e
}

// WithFields menggabungkan beberapa pesan field sekaligus.
func (e *Error) WithFields(fields map[string]string) *Error {
	if len(fields) == 0 {
		return e
	}
	if e.Fields == nil {
		e.Fields = make(map[string]string, len(fields))
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// WithMeta menyimpan data non-publik (mis. variant_id) yang dipakai service.
func (e *Error) WithMeta(key string, value any) *Error {
	if e.Meta == nil {
		e.Meta = make(map[string]any, 1)
	}
	e.Meta[key] = value
	return e
}

// New membuat *Error dari kode terdaftar. `vars` dipakai untuk merender pesan.
// Kode yang tidak terdaftar TIDAK pernah dipakai: hasilnya API_SERVER_ERROR,
// dan kejadian itu dicatat sebagai error agar tertangkap di test.
func New(code string, vars map[string]string) *Error {
	def, ok := registry[code]
	if !ok {
		slog.Error("kode error tidak terdaftar di registry apperr", "code", code)
		def = registry[CodeAPIServerError]
		code = CodeAPIServerError
	}
	return &Error{
		Code:     code,
		Message:  Render(def.Message, vars),
		Status:   def.Status,
		Severity: def.Severity,
		Category: def.Category,
		Persist:  def.Persist,
	}
}

// Wrap membungkus error asal dengan kode terdaftar; error asal tetap bisa
// ditelusuri lewat errors.Is/errors.As dan muncul di log.
func Wrap(err error, code string, vars map[string]string) *Error {
	if err == nil {
		return nil
	}
	e := New(code, vars)
	e.cause = err
	return e
}

// From mengubah error apa pun menjadi *Error. Error yang sudah *Error
// dikembalikan apa adanya; sisanya jadi API_SERVER_ERROR tanpa membocorkan
// detail internal ke klien.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	wrapped := New(CodeAPIServerError, nil)
	wrapped.cause = err
	return wrapped
}

// Status mengembalikan HTTP status yang tepat untuk error apa pun.
func Status(err error) int {
	if err == nil {
		return 200
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return 500
}

// Is melaporkan apakah error (atau rantainya) memakai kode tertentu.
func Is(err error, code string) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Code == code
}

// Render mengganti placeholder {nama} pada template dengan nilai dari vars.
// Placeholder tanpa nilai dibiarkan apa adanya agar kekurangan variabel terlihat
// saat review, bukan menghasilkan pesan kosong.
func Render(template string, vars map[string]string) string {
	if template == "" || len(vars) == 0 {
		return template
	}
	out := template
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// Lookup mengambil definisi kode; ok=false bila tidak terdaftar.
func Lookup(code string) (Definition, bool) {
	def, ok := registry[code]
	return def, ok
}

// Codes mengembalikan semua kode terdaftar, terurut. Dipakai oleh test dan
// oleh endpoint internal /api/v1/meta/error-codes (sinkronisasi dengan katalog FE).
func Codes() []string {
	out := make([]string, 0, len(registry))
	for code := range registry {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// register dipakai codes.go untuk mengisi registry saat init.
func register(defs ...Definition) {
	for _, d := range defs {
		if _, dup := registry[d.Code]; dup {
			panic("apperr: kode terdaftar dua kali: " + d.Code)
		}
		if d.Status < 400 || d.Status > 599 {
			panic("apperr: status tidak valid untuk " + d.Code)
		}
		if strings.TrimSpace(d.Message) == "" {
			panic("apperr: pesan kosong untuk " + d.Code)
		}
		registry[d.Code] = d
	}
}

var registry = make(map[string]Definition)
