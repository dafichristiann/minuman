// Package handler memuat handler HTTP.
//
// Aturan lapisan (§10.3): handler → service → repo. Handler tidak menyentuh SQL,
// service tidak tahu HTTP. Pada M0 hanya handler sistem (health & meta) yang ada;
// handler domain ditambahkan pada M1+.
package handler

import (
	nethttp "net/http"
	"sort"
	"time"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/platform/db"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/platform/logger"
	"github.com/kalengo/api/internal/platform/redisclient"
)

// System menangani endpoint operasional: /healthz, /readyz, /api/v1/meta.
//
// Endpoint ini SENGAJA di luar kontrak §12 dan tidak memakai envelope error,
// karena konsumennya adalah orkestrator (k8s/docker healthcheck) dan dashboard
// monitoring, bukan frontend.
type System struct {
	cfg       *config.Config
	db        *db.Pool
	redis     *redisclient.Client
	startedAt time.Time
}

// NewSystem membuat handler sistem.
func NewSystem(cfg *config.Config, database *db.Pool, rdb *redisclient.Client) *System {
	return &System{
		cfg:       cfg,
		db:        database,
		redis:     rdb,
		startedAt: time.Now(),
	}
}

type checkResult struct {
	Status  string `json:"status"` // "up" | "down"
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

type readyResponse struct {
	Status    string                 `json:"status"` // "ready" | "degraded"
	Version   string                 `json:"version"`
	UptimeSec int64                  `json:"uptime_seconds"`
	Checks    map[string]checkResult `json:"checks"`
}

// Live adalah liveness probe: proses hidup, tanpa menyentuh dependensi.
// Selalu 200 selama server bisa menjawab.
func (h *System) Live(w nethttp.ResponseWriter, r *nethttp.Request) {
	httpx.JSON(w, nethttp.StatusOK, map[string]any{
		"status":         "alive",
		"version":        h.cfg.Build.Version,
		"uptime_seconds": int64(time.Since(h.startedAt).Seconds()),
	})
}

// Ready adalah readiness probe: memastikan PostgreSQL dan Redis benar-benar
// bisa dipakai. 503 bila ada yang mati supaya load balancer berhenti mengirim
// trafik ke instance ini.
func (h *System) Ready(w nethttp.ResponseWriter, r *nethttp.Request) {
	checks := make(map[string]checkResult, 2)
	ready := true

	if h.db == nil {
		checks["postgres"] = checkResult{Status: "down", Error: "tidak dikonfigurasi"}
		ready = false
	} else {
		start := time.Now()
		if err := h.db.Health(r.Context()); err != nil {
			logger.FromContext(r.Context()).ErrorContext(r.Context(), "readiness: postgres tidak sehat", "error", err)
			checks["postgres"] = checkResult{Status: "down", Latency: time.Since(start).String(), Error: "tidak bisa dijangkau"}
			ready = false
		} else {
			checks["postgres"] = checkResult{Status: "up", Latency: time.Since(start).String()}
		}
	}

	if h.redis == nil {
		checks["redis"] = checkResult{Status: "down", Error: "tidak dikonfigurasi"}
		ready = false
	} else {
		start := time.Now()
		if err := h.redis.Health(r.Context()); err != nil {
			logger.FromContext(r.Context()).ErrorContext(r.Context(), "readiness: redis tidak sehat", "error", err)
			checks["redis"] = checkResult{Status: "down", Latency: time.Since(start).String(), Error: "tidak bisa dijangkau"}
			ready = false
		} else {
			checks["redis"] = checkResult{Status: "up", Latency: time.Since(start).String()}
		}
	}

	body := readyResponse{
		Status:    "degraded",
		Version:   h.cfg.Build.Version,
		UptimeSec: int64(time.Since(h.startedAt).Seconds()),
		Checks:    checks,
	}
	status := nethttp.StatusServiceUnavailable
	if ready {
		body.Status = "ready"
		status = nethttp.StatusOK
	} else if !h.cfg.IsProduction() && checks["postgres"].Status == "up" {
		status = nethttp.StatusOK
	}
	httpx.JSON(w, status, body)
}

// Meta mengembalikan informasi build & lingkungan untuk halaman /admin atau
// banner "versi baru tersedia" (§8.6 SYS_NEW_VERSION).
func (h *System) Meta(w nethttp.ResponseWriter, r *nethttp.Request) {
	httpx.JSON(w, nethttp.StatusOK, map[string]any{
		"app":               h.cfg.App.Name,
		"env":               string(h.cfg.Env),
		"version":           h.cfg.Build.Version,
		"commit":            h.cfg.Build.Commit,
		"uptime_seconds":    int64(time.Since(h.startedAt).Seconds()),
		"server_time":       time.Now().UTC().Format(time.RFC3339),
		"notif_modes":       []string{"normal", "calm", "silent"},
		"payment_provider":  h.cfg.Payment.Provider,
		"shipping_provider": h.cfg.Shipping.Provider,
	})
}

// ErrorCodes memublikasikan seluruh kode error backend agar katalog notifikasi
// frontend (§8.6) bisa diverifikasi sinkron saat build M6.
//
// Endpoint ini tidak rahasia (hanya daftar kode + pesan) dan berguna sebagai
// sumber tunggal kebenaran lintas bahasa.
func (h *System) ErrorCodes(w nethttp.ResponseWriter, r *nethttp.Request) {
	codes := apperr.Codes()
	items := make([]map[string]any, 0, len(codes))
	for _, code := range codes {
		def, ok := apperr.Lookup(code)
		if !ok {
			continue
		}
		items = append(items, map[string]any{
			"code":     def.Code,
			"status":   def.Status,
			"severity": string(def.Severity),
			"category": string(def.Category),
			"message":  def.Message,
			"persist":  def.Persist,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["code"].(string) < items[j]["code"].(string)
	})
	httpx.JSON(w, nethttp.StatusOK, map[string]any{
		"count": len(items),
		"items": items,
	})
}
