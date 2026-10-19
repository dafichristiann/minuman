package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// setBaseEnv menyiapkan environment minimum yang valid untuk pengembangan.
func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://kalengo_app:secret@localhost:5432/kalengo?sslmode=disable")
	t.Setenv("MIGRATE_DATABASE_URL", "postgres://kalengo_migrator:secret@localhost:5432/kalengo?sslmode=disable")
	t.Setenv("JWT_SECRET", "dev_only_secret_change_me_min_32_bytes_long!!")
	t.Setenv("WEB_ORIGIN", "http://localhost:5173")
}

func TestLoadUsesDefaults(t *testing.T) {
	setBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() tidak boleh gagal: %v", err)
	}

	if cfg.Env != EnvDevelopment {
		t.Errorf("Env = %q, ingin %q", cfg.Env, EnvDevelopment)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, ingin \":8080\"", cfg.HTTP.Addr)
	}
	if cfg.HTTP.MaxBodyBytes != 1<<20 {
		t.Errorf("HTTP.MaxBodyBytes = %d, ingin %d (1 MB, §13.1)", cfg.HTTP.MaxBodyBytes, 1<<20)
	}
	if cfg.Order.ReservationMinutes != 15*time.Minute {
		t.Errorf("Order.ReservationMinutes = %s, ingin 15m (§6.4)", cfg.Order.ReservationMinutes)
	}
	if cfg.RateLimit.LoginPerMinute != 10 {
		t.Errorf("RateLimit.LoginPerMinute = %d, ingin 10 (§12.8)", cfg.RateLimit.LoginPerMinute)
	}
	if cfg.Payment.Provider != "midtrans" {
		t.Errorf("Payment.Provider = %q, ingin \"midtrans\"", cfg.Payment.Provider)
	}
	if cfg.Shipping.Provider != "biteship" {
		t.Errorf("Shipping.Provider = %q, ingin \"biteship\"", cfg.Shipping.Provider)
	}
	if cfg.Shipping.RatesCacheTTL != 5*time.Minute {
		t.Errorf("Shipping.RatesCacheTTL = %s, ingin 5m (§6.4)", cfg.Shipping.RatesCacheTTL)
	}
	if !cfg.IsDevelopment() || cfg.IsProduction() {
		t.Error("IsDevelopment/IsProduction tidak konsisten dengan Env=development")
	}
	if cfg.HTTP.TrustProxyHeaders {
		t.Error("TrustProxyHeaders harus default false agar X-Forwarded-For tidak bisa dipalsukan")
	}
}

func TestLoadRejectsMigratorRoleForRuntime(t *testing.T) {
	setBaseEnv(t)
	// Role migrator TIDAK boleh dipakai runtime (§13.1).
	t.Setenv("DATABASE_URL", "postgres://kalengo_migrator:secret@localhost:5432/kalengo?sslmode=disable")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() harus gagal bila DATABASE_URL memakai role migrator")
	}
	if !strings.Contains(err.Error(), "least-privilege") {
		t.Errorf("pesan error kurang jelas: %v", err)
	}
}

func TestLoadRejectsSuperuserRole(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DATABASE_URL", "postgres://postgres:secret@localhost:5432/kalengo?sslmode=disable")

	if _, err := Load(); err == nil {
		t.Fatal("Load() harus menolak role superuser untuk runtime")
	}
}

func TestProductionRequiresSecureCookieAndStrongSecret(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("JWT_SECRET", "terlalu-pendek")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() harus gagal di produksi tanpa COOKIE_SECURE=true dan JWT_SECRET >= 32 byte")
	}
	msg := err.Error()
	if !strings.Contains(msg, "COOKIE_SECURE") {
		t.Errorf("error harus menyebut COOKIE_SECURE: %v", err)
	}
	if !strings.Contains(msg, "JWT_SECRET") {
		t.Errorf("error harus menyebut JWT_SECRET: %v", err)
	}
}

func TestProductionRequiresPaymentCredentials(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 40))
	// MIDTRANS_* dan BITESHIP_* sengaja dibiarkan kosong.

	_, err := Load()
	if err == nil {
		t.Fatal("Load() harus gagal bila kredensial Midtrans/Biteship kosong di produksi")
	}
	if !strings.Contains(err.Error(), "MIDTRANS_SERVER_KEY") {
		t.Errorf("error harus menyebut MIDTRANS_SERVER_KEY: %v", err)
	}
	if !strings.Contains(err.Error(), "BITESHIP_API_KEY") {
		t.Errorf("error harus menyebut BITESHIP_API_KEY: %v", err)
	}
}

func TestLoadRejectsUnknownLogLevel(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("LOG_LEVEL", "verbose")

	if _, err := Load(); err == nil {
		t.Fatal("Load() harus menolak LOG_LEVEL yang tidak dikenal")
	}
}

func TestDurationsAndMinutesParsing(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("ACCESS_TOKEN_TTL", "20m")
	t.Setenv("ORDER_RESERVATION_MINUTES", "7")
	t.Setenv("HTTP_MAX_BODY_BYTES", "2048")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() gagal: %v", err)
	}
	if cfg.Auth.AccessTokenTTL != 20*time.Minute {
		t.Errorf("AccessTokenTTL = %s, ingin 20m", cfg.Auth.AccessTokenTTL)
	}
	// ORDER_RESERVATION_MINUTES memakai satuan MENIT, bukan detik.
	if cfg.Order.ReservationMinutes != 7*time.Minute {
		t.Errorf("ReservationMinutes = %s, ingin 7m (angka polos = menit)", cfg.Order.ReservationMinutes)
	}
	if cfg.HTTP.MaxBodyBytes != 2048 {
		t.Errorf("MaxBodyBytes = %d, ingin 2048", cfg.HTTP.MaxBodyBytes)
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"":      slog.LevelInfo,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for raw, want := range cases {
		got, err := parseLevel(raw)
		if err != nil {
			t.Errorf("parseLevel(%q) error: %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("parseLevel(%q) = %v, ingin %v", raw, got, want)
		}
	}
	if _, err := parseLevel("nope"); err == nil {
		t.Error("parseLevel(\"nope\") harus error")
	}
}

func TestRedactedHidesSecrets(t *testing.T) {
	setBaseEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() gagal: %v", err)
	}

	red := cfg.Redacted()
	if _, exists := red["jwt_secret"]; exists {
		t.Error("Redacted() tidak boleh memuat jwt_secret")
	}
	if set, ok := red["jwt_secret_set"].(bool); !ok || !set {
		t.Error("Redacted() harus melaporkan jwt_secret_set=true")
	}
	if role, _ := red["db_url_role"].(string); role != "kalengo_app" {
		t.Errorf("db_url_role = %q, ingin \"kalengo_app\"", role)
	}
	// URL Redis harus disamarkan password-nya.
	if redisURL, _ := red["redis_url"].(string); strings.Contains(redisURL, "redis_dev_password") {
		t.Errorf("redis_url masih memuat password: %s", redisURL)
	}
}

func TestSplitAndTrim(t *testing.T) {
	got := splitAndTrim(" http://a.test , http://b.test ,, ")
	if len(got) != 2 || got[0] != "http://a.test" || got[1] != "http://b.test" {
		t.Errorf("splitAndTrim salah: %#v", got)
	}
}
