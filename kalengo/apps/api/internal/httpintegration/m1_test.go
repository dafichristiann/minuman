//go:build integration

// Package httpintegration menguji alur M1 lewat router produksi terhadap
// PostgreSQL sungguhan: migrasi + seed + request HTTP nyata.
//
// Cara menjalankan (PostgreSQL harus hidup dan sudah dimigrasi):
//
//	go test -tags integration -run TestM1 -v ./internal/httpintegration
//
// Variabel lingkungan:
//
//	KALENGO_TEST_DATABASE_URL  DSN database uji (wajib). Tanpa ini test SKIP,
//	                           sehingga `go test ./...` biasa tetap hijau.
//
// Catatan: Redis tidak wajib. Middleware rate limit bersifat fail-open saat
// Redis tidak terjangkau, dan /readyz menandai Redis "down" tanpa menggagalkan
// endpoint lain — itu perilaku yang diuji §13.3 "chaos kecil".
package httpintegration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/config"
	apiserver "github.com/kalengo/api/internal/http"
	"github.com/kalengo/api/internal/platform/db"
	"github.com/kalengo/api/internal/platform/payment"
	"github.com/kalengo/api/internal/platform/ratelimit"
	"github.com/kalengo/api/internal/platform/redisclient"
)

// testPaymentKey adalah kunci server tiruan yang dipakai router pada uji
// (lihat testConfig). Nilainya bukan rahasia apa pun.
const testPaymentKey = "server-key-uji-integrasi"

// ---------------------------------------------------------------------------
// perkakas
// ---------------------------------------------------------------------------

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("KALENGO_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("KALENGO_TEST_DATABASE_URL tidak diset; melewati uji integrasi")
	}
	return dsn
}

func testConfig() *config.Config {
	return &config.Config{
		Env: config.EnvStaging,
		App: config.AppConfig{Name: "kalengo", ShutdownIn: 5 * time.Second},
		Auth: config.AuthConfig{
			JWTSecret:         "rahasia-uji-integrasi-yang-cukup-panjang-hs256",
			AccessTokenTTL:    15 * time.Minute,
			RefreshTokenTTL:   720 * time.Hour,
			CookieName:        "kalengo_at",
			RefreshCookieName: "kalengo_rt",
			CSRFCookieName:    "kalengo_csrf",
			CSRFHeaderName:    "X-CSRF-Token",
			CookieSecure:      false,
		},
		// Parameter argon2id diperkecil: uji bukan tolok ukur kinerja hashing.
		Argon2: config.Argon2Config{MemoryKiB: 8 * 1024, Time: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32},
		// Penyedia M2 memakai kunci server tiruan ini.
		Payment: config.PaymentConfig{MidtransServerKey: testPaymentKey},
		HTTP: config.HTTPConfig{
			RequestTimeout: 25 * time.Second,
			MaxBodyBytes:   1 << 20,
		},
		RateLimit: config.RateLimitConfig{
			GeneralPerMinute:   10000,
			LoginPerMinute:     10000,
			RegisterPerHour:    10000,
			LoginFailPerWindow: 5,
			LoginFailWindow:    15 * time.Minute,
			LoginLockout:       15 * time.Minute,
		},
		Build: config.BuildInfo{Version: "test", Commit: "test"},
	}
}

// harness adalah satu API hidup dengan database sungguhan.
type harness struct {
	base    string
	client  *http.Client
	db      *db.Pool
	payment *payment.Fake
	cleanup func()
}

func newHarness(t *testing.T, dsn string) *harness {
	t.Helper()
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := testConfig()
	cfg.DB.URL = dsn

	pool, err := db.New(ctx, db.Options{
		DSN:          dsn,
		MaxConns:     4,
		MinConns:     0,
		AppName:      "kalengo-test",
		PingAttempts: 3,
		PingBackoff:  200 * time.Millisecond,
	}, log)
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}

	// Redis sengaja menunjuk port mati: menguji jalur fail-open rate limiter.
	rdb, err := redisclient.New(ctx, redisclient.Options{
		URL:          "redis://127.0.0.1:6399/0",
		DialTimeout:  200 * time.Millisecond,
		PingAttempts: 1,
		PingBackoff:  50 * time.Millisecond,
	})
	if err != nil {
		rdb = nil // tidak terpakai bila gagal; handler sistem sudah menoleransi nil
	}

	var limiter *ratelimit.Limiter
	if rdb != nil {
		limiter = ratelimit.New(rdb.Client, "kalengo:test")
	}

	// Penyedia palsu yang sama dengan yang dipakai router, agar uji bisa
	// menandatangani webhook dengan kunci yang benar.
	// Router membuat penyedianya sendiri; kunci server harus sama agar tanda
	// tangan yang dibuat uji diterima.
	router := apiserver.NewRouter(apiserver.Deps{
		Config:  cfg,
		Log:     log,
		DB:      pool,
		Redis:   rdb,
		Limiter: limiter,
	})

	srv := httptest.NewServer(router)
	jar, _ := cookiejar.New(nil)

	h := &harness{
		base:    srv.URL,
		client:  &http.Client{Jar: jar, Timeout: 30 * time.Second},
		db:      pool,
		payment: &payment.Fake{ServerKey: testPaymentKey},
	}
	h.cleanup = func() {
		srv.Close()
		pool.Close()
		if rdb != nil {
			_ = rdb.Close()
		}
	}
	return h
}

// do mengirim permintaan dan mengembalikan status + body mentah.
func (h *harness) do(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()
	return h.doWithHeader(t, method, path, body, nil)
}

// doWithHeader mengirim permintaan dengan header tambahan.
func (h *harness) doWithHeader(t *testing.T, method, path string, body any, headers map[string]string) (int, []byte) {
	t.Helper()

	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, h.base+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// Sertakan CSRF dari cookie (pola double-submit §13.1).
	for _, c := range h.client.Jar.Cookies(req.URL) {
		if c.Name == "kalengo_csrf" {
			req.Header.Set("X-CSRF-Token", c.Value)
		}
	}

	res, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("baca body: %v", err)
	}
	return res.StatusCode, raw
}

func decode(t *testing.T, raw []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("unmarshal %s: %v", string(raw), err)
	}
}

// uniqueEmail menghasilkan email unik agar test tidak bergantung urutan.
func uniqueEmail() string {
	return fmt.Sprintf("m1-%s@test.local", uuid.NewString())
}

// cleanupUser menghapus pengguna uji beserta data turunannya.
func (h *harness) cleanupUser(t *testing.T, email string) {
	t.Helper()
	ctx := context.Background()
	// ON DELETE CASCADE belum tentu ada; hapus anak lebih dulu.
	_, err := h.db.Exec(ctx, `
		DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, email)
	if err != nil {
		t.Logf("cleanup refresh_tokens: %v", err)
	}
	_, err = h.db.Exec(ctx, `
		DELETE FROM one_time_tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, email)
	if err != nil {
		t.Logf("cleanup one_time_tokens: %v", err)
	}
	if _, err := h.db.Exec(ctx, `DELETE FROM users WHERE email = $1`, email); err != nil {
		t.Logf("cleanup users: %v", err)
	}
}

// ---------------------------------------------------------------------------
// uji
// ---------------------------------------------------------------------------

// TestM1AuthFlow menutup gap M1: register → verifikasi email → login → /me.
func TestM1AuthFlow(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	email := uniqueEmail()
	defer h.cleanupUser(t, email)

	const password = "kaleng-enak-2026"

	// --- 1. register ---
	// Catatan: handler memakai decoder "field tidak dikenal ditolak", dan
	// accept_terms belum diterima registerRequest. Kontrak OpenAPI masih
	// mencantumkannya; selisih ini dicatat di docs/AUDIT-REPORT.md.
	status, raw := h.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":     email,
		"password":  password,
		"full_name": "Budi Uji Integrasi",
	})
	if status != http.StatusCreated {
		t.Fatalf("register status = %d, ingin 201. body=%s", status, raw)
	}

	var reg struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
		DevVerifyToken string `json:"dev_verify_token"`
	}
	decode(t, raw, &reg)
	if reg.User.Email != email {
		t.Fatalf("email user = %q, ingin %q", reg.User.Email, email)
	}
	if reg.User.ID == "" {
		t.Fatal("user.id kosong pada respons register")
	}
	if reg.DevVerifyToken == "" {
		t.Fatal("dev_verify_token kosong; seharusnya ada di luar produksi")
	}

	// Baris benar-benar ada di PostgreSQL.
	var dbCount int
	if err := h.db.QueryRow(context.Background(),
		`SELECT count(*) FROM users WHERE email = $1 AND email_verified_at IS NULL`, email).Scan(&dbCount); err != nil {
		t.Fatalf("query users: %v", err)
	}
	if dbCount != 1 {
		t.Fatalf("user di DB = %d, ingin 1 dan belum terverifikasi", dbCount)
	}

	// --- 2. verifikasi email (atomik: token terpakai + user terverifikasi) ---
	status, raw = h.do(t, http.MethodPost, "/api/v1/auth/verify-email", map[string]any{
		"token": reg.DevVerifyToken,
	})
	if status != http.StatusNoContent {
		t.Fatalf("verify-email status = %d, ingin 204. body=%s", status, raw)
	}
	if err := h.db.QueryRow(context.Background(),
		`SELECT count(*) FROM users WHERE email = $1 AND email_verified_at IS NOT NULL`, email).Scan(&dbCount); err != nil {
		t.Fatalf("query verified: %v", err)
	}
	if dbCount != 1 {
		t.Fatal("email belum terverifikasi di DB setelah verify-email")
	}

	// Token sekali pakai: percobaan kedua harus gagal.
	status, _ = h.do(t, http.MethodPost, "/api/v1/auth/verify-email", map[string]any{
		"token": reg.DevVerifyToken,
	})
	if status == http.StatusNoContent {
		t.Fatal("token verifikasi bisa dipakai dua kali")
	}

	// --- 3. login ---
	status, raw = h.do(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("login status = %d, ingin 200. body=%s", status, raw)
	}
	var login struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
		CSRFToken string `json:"csrf_token"`
	}
	decode(t, raw, &login)
	if login.User.Email != email {
		t.Fatalf("login user.email = %q, ingin %q", login.User.Email, email)
	}
	if login.CSRFToken == "" {
		t.Fatal("csrf_token kosong pada respons login")
	}

	// Atribut cookie diverifikasi dari header Set-Cookie, bukan dari cookiejar:
	// net/http/cookiejar tidak menyimpan atribut HttpOnly/Path/SameSite.
	h.assertCookieAttributes(t, email, password)

	// --- 4. /me dengan cookie nyata ---
	status, raw = h.do(t, http.MethodGet, "/api/v1/me", nil)
	if status != http.StatusOK {
		t.Fatalf("/me status = %d, ingin 200. body=%s", status, raw)
	}
	var me struct {
		Email         string `json:"email"`
		FullName      string `json:"full_name"`
		EmailVerified bool   `json:"email_verified"`
	}
	decode(t, raw, &me)
	if me.Email != email {
		t.Fatalf("/me email = %q, ingin %q", me.Email, email)
	}
	if me.FullName != "Budi Uji Integrasi" {
		t.Fatalf("/me full_name = %q", me.FullName)
	}
	if !me.EmailVerified {
		t.Fatal("/me email_verified = false setelah verifikasi")
	}
}

// TestM1UnauthenticatedIsRejected memastikan area privat menolak tanpa cookie.
func TestM1UnauthenticatedIsRejected(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	for _, path := range []string{"/api/v1/me", "/api/v1/cart", "/api/v1/wishlist"} {
		status, raw := h.do(t, http.MethodGet, path, nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("%s tanpa sesi = %d, ingin 401. body=%s", path, status, raw)
		}
	}
}

// TestM1CatalogReadsSeededData membaca katalog dari seed nyata.
func TestM1CatalogReadsSeededData(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	status, raw := h.do(t, http.MethodGet, "/api/v1/products?limit=5", nil)
	if status != http.StatusOK {
		t.Fatalf("/products status = %d, ingin 200. body=%s", status, raw)
	}
	var list struct {
		Items []struct {
			Slug        string `json:"slug"`
			Name        string `json:"name"`
			PriceFrom   int64  `json:"price_from"`
			StockStatus string `json:"stock_status"`
		} `json:"items"`
		HasMore bool `json:"has_more"`
	}
	decode(t, raw, &list)
	if len(list.Items) == 0 {
		t.Fatal("/products kosong; seed belum dijalankan?")
	}
	for _, p := range list.Items {
		if p.Slug == "" || p.Name == "" {
			t.Fatalf("produk tanpa slug/nama: %+v", p)
		}
		if p.PriceFrom <= 0 {
			t.Fatalf("price_from bukan integer rupiah positif: %+v", p)
		}
	}

	// Kategori & merek harus memakai envelope { items: [...] }.
	for _, path := range []string{"/api/v1/categories", "/api/v1/brands"} {
		status, raw := h.do(t, http.MethodGet, path, nil)
		if status != http.StatusOK {
			t.Fatalf("%s status = %d. body=%s", path, status, raw)
		}
		var envelope struct {
			Items []json.RawMessage `json:"items"`
		}
		decode(t, raw, &envelope)
		if envelope.Items == nil {
			t.Fatalf("%s tidak memakai envelope {items}", path)
		}
	}
}

// TestM1CSRFRequired memastikan mutasi tanpa header CSRF ditolak.
func TestM1CSRFRequired(t *testing.T) {
	h := newHarness(t, testDSN(t))
	defer h.cleanup()

	email := uniqueEmail()
	defer h.cleanupUser(t, email)

	const password = "kaleng-enak-2026"
	if status, raw := h.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": email, "password": password, "full_name": "Budi CSRF",
	}); status != http.StatusCreated {
		t.Fatalf("register = %d. body=%s", status, raw)
	}
	if status, raw := h.do(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email": email, "password": password,
	}); status != http.StatusOK {
		t.Fatalf("login = %d. body=%s", status, raw)
	}

	// Kirim PATCH tanpa header CSRF.
	req, err := http.NewRequest(http.MethodPatch, h.base+"/api/v1/me",
		bytes.NewReader([]byte(`{"full_name":"Nama Baru"}`)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("patch me: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		t.Fatal("PATCH /me tanpa header X-CSRF-Token berhasil; proteksi CSRF bocor")
	}
}

// loginCapture mengirim login dengan klien tanpa jar dan mengembalikan
// header Set-Cookie mentah untuk diperiksa atributnya.
func (h *harness) loginCapture(t *testing.T, email, password string) []string {
	t.Helper()

	payload, err := json.Marshal(map[string]any{"email": email, "password": password})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, h.base+"/api/v1/auth/login", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("login capture status = %d", res.StatusCode)
	}
	return res.Header.Values("Set-Cookie")
}

// assertCookieAttributes memastikan §13.1 dipatuhi: token sesi tidak dapat
// dibaca JavaScript, sedangkan token CSRF memang harus bisa (double-submit).
func (h *harness) assertCookieAttributes(t *testing.T, email, password string) {
	t.Helper()

	cookies := h.loginCapture(t, email, password)
	seen := map[string]bool{}

	for _, raw := range cookies {
		parts := strings.Split(raw, ";")
		name := strings.TrimSpace(strings.SplitN(parts[0], "=", 2)[0])
		lower := strings.ToLower(raw)
		seen[name] = true

		switch name {
		case "kalengo_at", "kalengo_rt":
			if !strings.Contains(lower, "httponly") {
				t.Fatalf("cookie %s harus HttpOnly: %s", name, raw)
			}
			if c, err := http.ParseSetCookie(raw); err == nil && c.Secure {
				// Secure wajib di produksi; di server test (HTTP) boleh false.
				t.Logf("cookie %s Secure=true", name)
			}
		case "kalengo_csrf":
			if strings.Contains(lower, "httponly") {
				t.Fatalf("cookie CSRF tidak boleh HttpOnly (klien harus membacanya): %s", raw)
			}
		}
	}

	for _, want := range []string{"kalengo_at", "kalengo_rt", "kalengo_csrf"} {
		if !seen[want] {
			t.Fatalf("cookie %s tidak dipasang saat login", want)
		}
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url %q: %v", raw, err)
	}
	return u
}
