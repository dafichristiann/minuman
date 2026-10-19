package handler

import (
	nethttp "net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/platform/httpx"
)

// uuidParam membaca parameter rute berbentuk UUID. Parameter cacat dijawab 404
// (bukan 422): bagi klien, resource itu memang tidak ada.
func uuidParam(w nethttp.ResponseWriter, r *nethttp.Request, name, field string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, name)))
	if err != nil || id == uuid.Nil {
		httpx.Error(w, r, apperr.New(apperr.CodeAPINotFound, nil).
			WithField(field, "tidak valid"))
		return uuid.Nil, false
	}
	return id, true
}

// chiURLParam membaca parameter rute apa adanya (mis. slug yang tidak berbentuk
// UUID).
func chiURLParam(r *nethttp.Request, name string) string {
	return strings.TrimSpace(chi.URLParam(r, name))
}

// intQuery membaca query integer dengan nilai default dan batas.
func intQuery(r *nethttp.Request, name string, def, min, max int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// boolQuery membaca query boolean ("1", "true", "yes").
func boolQuery(r *nethttp.Request, name string) bool {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get(name))) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}
