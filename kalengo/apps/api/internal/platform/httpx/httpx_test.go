package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/platform/logger"
)

// errSentinel meniru error internal yang memuat detail sensitif; detail itu
// TIDAK boleh sampai ke klien.
var errSentinel = errors.New("gagal konek: kredensial database ditolak")

func errFmt(msg string) error { return fmt.Errorf("%s", msg) }

func TestJSONWritesStatusAndContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, http.StatusCreated, map[string]string{"ok": "ya"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, ingin 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ContentTypeJSON {
		t.Errorf("Content-Type = %q, ingin %q", ct, ContentTypeJSON)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, ingin \"nosniff\"", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body bukan JSON valid: %v", err)
	}
	if body["ok"] != "ya" {
		t.Errorf("body = %#v", body)
	}
}

func TestJSONDoesNotEscapeHTML(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, http.StatusOK, map[string]string{"nama": "Kopi & Susu <3"})

	if strings.Contains(rec.Body.String(), `\u0026`) {
		t.Errorf("karakter HTML tidak boleh di-escape: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Kopi & Susu <3") {
		t.Errorf("isi tidak sesuai: %s", rec.Body.String())
	}
}

func TestNoContentWrites204(t *testing.T) {
	rec := httptest.NewRecorder()

	NoContent(rec)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, ingin 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("204 tidak boleh punya body, dapat %q", rec.Body.String())
	}
}

func TestErrorWritesEnvelopeWithCodeAndRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	req = req.WithContext(logger.WithRequestID(req.Context(), "req-abcdefgh"))
	rec := httptest.NewRecorder()

	Error(rec, req, apperr.New(apperr.CodeCartPriceChanged, map[string]string{
		"produk": "Kopi Susu",
		"lama":   "Rp12.000",
		"baru":   "Rp14.000",
	}))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, ingin 409", rec.Code)
	}

	var env ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body bukan envelope valid: %v (%s)", err, rec.Body.String())
	}
	if env.Error.Code != apperr.CodeCartPriceChanged {
		t.Errorf("code = %q, ingin %q", env.Error.Code, apperr.CodeCartPriceChanged)
	}
	if env.Error.RequestID != "req-abcdefgh" {
		t.Errorf("request_id = %q, ingin diteruskan dari context", env.Error.RequestID)
	}
	if !strings.Contains(env.Error.Message, "Kopi Susu") {
		t.Errorf("pesan harus sudah dirender: %q", env.Error.Message)
	}
}

func TestErrorOnPlainErrorDoesNotLeakDetail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()

	Error(rec, req, errSentinel)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, ingin 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "kredensial database") {
		t.Errorf("detail internal bocor ke klien: %s", rec.Body.String())
	}

	var env ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body bukan envelope valid: %v", err)
	}
	if env.Error.Code != apperr.CodeAPIServerError {
		t.Errorf("code = %q, ingin %q", env.Error.Code, apperr.CodeAPIServerError)
	}
}

func TestErrorCodeUsesRegistryMessage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	rec := httptest.NewRecorder()

	ErrorCode(rec, req, apperr.CodeAPIRateLimit, map[string]string{"detik": "42"})

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, ingin 429", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "42") {
		t.Errorf("variabel pesan tidak dirender: %s", rec.Body.String())
	}
}

// ---------------------------------------------------------------
// DecodeJSON
// ---------------------------------------------------------------

type payload struct {
	Nama string `json:"nama"`
	Qty  int    `json:"qty"`
}

func TestDecodeJSONAcceptsValidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"nama":"Kopi","qty":2}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	var dst payload
	if err := DecodeJSON(rec, req, &dst, 1024); err != nil {
		t.Fatalf("DecodeJSON gagal: %v", err)
	}
	if dst.Nama != "Kopi" || dst.Qty != 2 {
		t.Errorf("hasil decode = %#v", dst)
	}
}

func TestDecodeJSONAcceptsContentTypeWithCharset(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"nama":"Teh","qty":1}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()

	var dst payload
	if err := DecodeJSON(rec, req, &dst, 1024); err != nil {
		t.Fatalf("DecodeJSON gagal untuk content-type dengan charset: %v", err)
	}
}

func TestDecodeJSONRejectsUnsupportedMediaType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`nama=Kopi`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &payload{}, 1024)
	if !apperr.Is(err, apperr.CodeAPIUnsupportedMedia) {
		t.Errorf("error = %v, ingin %s", err, apperr.CodeAPIUnsupportedMedia)
	}
}

// Field tak dikenal ditolak agar typo pada klien ketahuan lebih awal.
func TestDecodeJSONRejectsUnknownField(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"nama":"Kopi","kuantitas":2}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &payload{}, 1024)
	if !apperr.Is(err, apperr.CodeAPIValidation) {
		t.Fatalf("error = %v, ingin %s", err, apperr.CodeAPIValidation)
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error bukan *apperr.Error: %v", err)
	}
	if _, ok := appErr.Fields["kuantitas"]; !ok {
		t.Errorf("field bermasalah harus dilaporkan: %#v", appErr.Fields)
	}
}

func TestDecodeJSONRejectsWrongType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"nama":"Kopi","qty":"dua"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &payload{}, 1024)
	if !apperr.Is(err, apperr.CodeAPIValidation) {
		t.Errorf("error = %v, ingin %s", err, apperr.CodeAPIValidation)
	}
}

func TestDecodeJSONRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"nama":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &payload{}, 1024)
	if !apperr.Is(err, apperr.CodeAPIBadRequest) {
		t.Errorf("error = %v, ingin %s", err, apperr.CodeAPIBadRequest)
	}
}

func TestDecodeJSONRejectsEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &payload{}, 1024)
	if !apperr.Is(err, apperr.CodeAPIBadRequest) {
		t.Errorf("error = %v, ingin %s", err, apperr.CodeAPIBadRequest)
	}
}

// Body raksasa harus ditolak sebelum dihabiskan (§13.1).
func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	big := `{"nama":"` + strings.Repeat("a", 4096) + `","qty":1}`
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &payload{}, 256)
	if !apperr.Is(err, apperr.CodeSysUploadTooBig) {
		t.Errorf("error = %v, ingin %s", err, apperr.CodeSysUploadTooBig)
	}
}

func TestDecodeJSONRejectsTrailingData(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"nama":"Kopi","qty":1}{"nama":"Teh","qty":2}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &payload{}, 1024)
	if !apperr.Is(err, apperr.CodeAPIBadRequest) {
		t.Errorf("error = %v, ingin %s", err, apperr.CodeAPIBadRequest)
	}
}

func TestUnknownJSONFieldDetection(t *testing.T) {
	field, ok := UnknownJSONField(errFmt(`json: unknown field "kuantitas"`))
	if !ok || field != "kuantitas" {
		t.Errorf("UnknownJSONField = (%q, %v), ingin (\"kuantitas\", true)", field, ok)
	}
	if _, ok := UnknownJSONField(errFmt("json: cannot unmarshal string")); ok {
		t.Error("UnknownJSONField harus false untuk error lain")
	}
}
