// Package httpx menyediakan pembantu HTTP: penulisan JSON, dekode body yang ketat,
// dan bentuk error seragam sesuai §12.
//
// Bentuk error (§12):
//
//	{ "error": { "code": "CART_PRICE_CHANGED", "message": "Harga berubah",
//	             "fields": {"qty":"maks 24"}, "request_id": "..." } }
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/platform/logger"
)

// ContentTypeJSON adalah tipe konten default seluruh API.
const ContentTypeJSON = "application/json; charset=utf-8"

// ErrorBody adalah isi objek "error".
type ErrorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// ErrorEnvelope membungkus ErrorBody, mengikuti gaya RFC 9457 yang disederhanakan.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// JSON menulis payload sebagai JSON dengan status tertentu.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	if payload == nil {
		return
	}
	enc := json.NewEncoder(w)
	// HTML escaping dimatikan karena output selalu JSON, bukan HTML (§13.1).
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		// Header sudah terkirim; tidak ada yang bisa dilakukan selain mencatat.
		slog.Error("gagal menulis respons JSON", "error", err)
	}
}

// NoContent menulis 204 tanpa body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error menulis error dalam bentuk envelope §12 dengan status yang tepat.
//
// Detail internal TIDAK pernah dikirim ke klien: hanya kode + pesan dari
// registry apperr. Error asal tetap tercatat di log.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	appErr := apperr.From(err)
	if appErr == nil {
		appErr = apperr.New(apperr.CodeAPIServerError, nil)
	}

	reqID, _ := logger.RequestIDFrom(r.Context())
	log := logger.FromContext(r.Context())

	level := slog.LevelWarn
	if appErr.Status >= 500 {
		level = slog.LevelError
	}
	log.Log(r.Context(), level, "permintaan gagal",
		"code", appErr.Code,
		"status", appErr.Status,
		"error", err,
	)

	JSON(w, appErr.Status, ErrorEnvelope{Error: ErrorBody{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Fields:    appErr.Fields,
		RequestID: reqID,
	}})
}

// ErrorCode menulis error berdasarkan kode registry, memakai variabel untuk pesan.
func ErrorCode(w http.ResponseWriter, r *http.Request, code string, vars map[string]string) {
	Error(w, r, apperr.New(code, vars))
}

// DecodeJSON membaca body JSON dengan batas ukuran dan penolakan field tak dikenal.
//
// `maxBytes` melindungi server dari body raksasa (§13.1: batas ukuran body).
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		return apperr.New(apperr.CodeAPIUnsupportedMedia, nil)
	}

	body := http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err, maxBytes)
	}
	// Pastikan tidak ada JSON tambahan setelah objek pertama.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperr.New(apperr.CodeAPIBadRequest, nil).WithField("body", "hanya satu objek JSON yang diizinkan")
	}
	return nil
}

func decodeError(err error, maxBytes int64) error {
	var maxErr *http.MaxBytesError
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	switch {
	case errors.As(err, &maxErr):
		return apperr.New(apperr.CodeSysUploadTooBig, map[string]string{
			"n": fmt.Sprintf("%d", maxBytes/(1<<20)),
		})
	case errors.As(err, &syntaxErr):
		return apperr.New(apperr.CodeAPIBadRequest, nil).WithField("body", "JSON tidak valid")
	case errors.As(err, &typeErr):
		return apperr.New(apperr.CodeAPIValidation, nil).WithField(typeErr.Field, "tipe nilai tidak sesuai")
	case errors.Is(err, io.EOF):
		return apperr.New(apperr.CodeAPIBadRequest, nil).WithField("body", "body kosong")
	}

	if field, ok := UnknownJSONField(err); ok {
		return apperr.New(apperr.CodeAPIValidation, nil).WithField(field, "field tidak dikenal")
	}
	return apperr.Wrap(err, apperr.CodeAPIBadRequest, nil)
}

// UnknownJSONField mengenali error decoder `DisallowUnknownFields`.
// encoding/json mengembalikan error ini sebagai error biasa (bukan tipe
// terdedikasi), jadi pencocokan dilakukan lewat pesannya.
func UnknownJSONField(err error) (string, bool) {
	const prefix = `json: unknown field `
	msg := err.Error()
	if !strings.HasPrefix(msg, prefix) {
		return "", false
	}
	field := strings.Trim(msg[len(prefix):], `"`)
	if field == "" {
		return "", false
	}
	return field, true
}
