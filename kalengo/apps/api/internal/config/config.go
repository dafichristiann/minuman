// Package config memuat seluruh konfigurasi aplikasi dari environment (§10.2).
//
// Aturan (§13.1): tidak ada rahasia di repo. Semua nilai sensitif datang dari
// environment / secret manager. Berkas .env hanya untuk pengembangan lokal.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Environment adalah mode berjalannya aplikasi.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// Config adalah akar konfigurasi.
type Config struct {
	Env  Environment
	App  AppConfig
	HTTP HTTPConfig
	DB   DBConfig
	// Redis dipakai untuk cache, rate limit, pub/sub SSE, dan antrean Asynq.
	Redis     RedisConfig
	Auth      AuthConfig
	Argon2    Argon2Config
	RateLimit RateLimitConfig
	Payment   PaymentConfig
	Shipping  ShippingConfig
	SMTP      SMTPConfig
	S3        S3Config
	Push      PushConfig
	Obs       ObsConfig
	Order     OrderConfig
	Build     BuildInfo
}

// AppConfig adalah identitas & lokasi aplikasi.
type AppConfig struct {
	Name       string
	BaseURL    string
	WebOrigin  []string // whitelist CORS, eksplisit (§13.1)
	LogLevel   slog.Level
	LogPretty  bool // hanya dev
	ShutdownIn time.Duration
}

// HTTPConfig mengatur server HTTP.
type HTTPConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	RequestTimeout    time.Duration
	MaxBodyBytes      int64
	// TrustProxyHeaders: aktifkan HANYA bila API berada di belakang reverse proxy
	// yang menulis X-Forwarded-For (Nginx/Caddy di §15). Bila tidak, header itu
	// bisa dipalsukan klien dan merusak rate limit berbasis IP.
	TrustProxyHeaders bool
}

// DBConfig menunjuk ke PostgreSQL (§10.2, §13.1 least-privilege).
type DBConfig struct {
	// URL memakai role aplikasi: TANPA hak DDL.
	URL string
	// MigrateURL memakai role pemilik skema. Hanya dipakai CLI migrasi, bukan runtime.
	MigrateURL      string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

// RedisConfig mengatur klien Redis.
type RedisConfig struct {
	URL         string
	PoolSize    int
	DialTimeout time.Duration
}

// AuthConfig mengatur token & cookie (§10.2, §13.1).
type AuthConfig struct {
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	CookieDomain      string
	CookieSecure      bool
	CookieName        string // access token
	RefreshCookieName string
	CSRFCookieName    string
	CSRFHeaderName    string
}

// Argon2Config mencatat parameter argon2id (§13.1). Menyimpan bersamanya penting
// agar hash lama tetap bisa diverifikasi setelah parameter dinaikkan.
type Argon2Config struct {
	MemoryKiB   uint32
	Time        uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// RateLimitConfig adalah batas default (§12.8).
type RateLimitConfig struct {
	GeneralPerMinute   int
	LoginPerMinute     int
	RegisterPerHour    int
	VoucherPerMinute   int
	OrderPerMinute     int
	LoginFailPerWindow int
	LoginFailWindow    time.Duration
	LoginLockout       time.Duration
}

// PaymentConfig untuk Midtrans Snap (§6.5).
type PaymentConfig struct {
	Provider          string
	MidtransEnv       string
	MidtransServerKey string
	MidtransClientKey string
	RequireSignature  bool
}

// ShippingConfig untuk Biteship (§6.4).
type ShippingConfig struct {
	Provider        string
	BiteshipAPIKey  string
	BiteshipBaseURL string
	OriginAreaID    string
	RatesCacheTTL   time.Duration
}

// SMTPConfig untuk email transaksional (§8.2).
type SMTPConfig struct {
	Host    string
	Port    int
	User    string
	Pass    string
	From    string
	UseTLS  bool
	Timeout time.Duration
}

// S3Config untuk object storage media produk.
type S3Config struct {
	Endpoint     string
	Region       string
	Bucket       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
}

// PushConfig untuk Web Push VAPID (§8.7).
type PushConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
}

// ObsConfig untuk observabilitas (§7).
type ObsConfig struct {
	SentryDSN      string
	OTLPEndpoint   string
	MetricsEnabled bool
}

// OrderConfig mengatur siklus hidup pesanan (§6.4).
type OrderConfig struct {
	ReservationMinutes  time.Duration
	ExpirySweepInterval time.Duration
}

// BuildInfo diisi lewat -ldflags pada saat build.
type BuildInfo struct {
	Version string
	Commit  string
}

// Load membaca environment, memuat .env bila ada (non-produksi), lalu memvalidasi.
func Load() (*Config, error) {
	// .env tidak pernah dibaca di produksi: konfigurasi datang dari orchestrator.
	env := Environment(strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))))
	if env == "" {
		env = EnvDevelopment
	}
	if env != EnvProduction {
		if _, statErr := os.Stat(".env"); statErr == nil {
			// Best-effort: variabel environment yang sudah ada TIDAK ditimpa.
			_ = godotenv.Load(".env")
		}
	}

	var errs []error

	cfg := &Config{Env: env}
	cfg.Build = BuildInfo{Version: str("APP_VERSION", "dev"), Commit: str("APP_COMMIT", "unknown")}

	level, err := parseLevel(str("LOG_LEVEL", "info"))
	if err != nil {
		errs = append(errs, err)
	}
	cfg.App = AppConfig{
		Name:       str("APP_NAME", "kalengo"),
		BaseURL:    str("BASE_URL", "http://localhost:8080"),
		WebOrigin:  splitAndTrim(str("WEB_ORIGIN", "http://localhost:5173")),
		LogLevel:   level,
		LogPretty:  env == EnvDevelopment,
		ShutdownIn: duration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}

	cfg.HTTP = HTTPConfig{
		Addr:              str("HTTP_ADDR", ":8080"),
		ReadTimeout:       duration("HTTP_READ_TIMEOUT", 10*time.Second),
		ReadHeaderTimeout: duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		WriteTimeout:      duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       duration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		RequestTimeout:    duration("HTTP_REQUEST_TIMEOUT", 25*time.Second),
		MaxBodyBytes:      int64(integer("HTTP_MAX_BODY_BYTES", 1<<20)), // 1 MB (§13.1)
		TrustProxyHeaders: boolean("HTTP_TRUST_PROXY_HEADERS", false),
	}

	cfg.DB = DBConfig{
		URL:             str("DATABASE_URL", ""),
		MigrateURL:      str("MIGRATE_DATABASE_URL", ""),
		MaxConns:        int32(integer("DB_MAX_CONNS", 20)),
		MinConns:        int32(integer("DB_MIN_CONNS", 2)),
		MaxConnLifetime: duration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		MaxConnIdleTime: duration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		ConnectTimeout:  duration("DB_CONNECT_TIMEOUT", 5*time.Second),
	}

	cfg.Redis = RedisConfig{
		URL:         str("REDIS_URL", "redis://localhost:6379/0"),
		PoolSize:    integer("REDIS_POOL_SIZE", 20),
		DialTimeout: duration("REDIS_DIAL_TIMEOUT", 3*time.Second),
	}

	cfg.Auth = AuthConfig{
		JWTSecret:         str("JWT_SECRET", ""),
		AccessTokenTTL:    duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:   duration("REFRESH_TOKEN_TTL", 720*time.Hour),
		CookieDomain:      str("COOKIE_DOMAIN", ""),
		CookieSecure:      boolean("COOKIE_SECURE", env != EnvDevelopment),
		CookieName:        str("ACCESS_COOKIE_NAME", "kalengo_at"),
		RefreshCookieName: str("REFRESH_COOKIE_NAME", "kalengo_rt"),
		CSRFCookieName:    str("CSRF_COOKIE_NAME", "kalengo_csrf"),
		CSRFHeaderName:    str("CSRF_HEADER_NAME", "X-CSRF-Token"),
	}

	cfg.Argon2 = Argon2Config{
		MemoryKiB:   uint32(integer("ARGON2_MEMORY_KIB", 64*1024)),
		Time:        uint32(integer("ARGON2_TIME", 3)),
		Parallelism: uint8(integer("ARGON2_PARALLELISM", 2)),
		SaltLength:  uint32(integer("ARGON2_SALT_LENGTH", 16)),
		KeyLength:   uint32(integer("ARGON2_KEY_LENGTH", 32)),
	}

	cfg.RateLimit = RateLimitConfig{
		GeneralPerMinute:   integer("RATE_LIMIT_GENERAL_PER_MIN", 120),
		LoginPerMinute:     integer("RATE_LIMIT_LOGIN_PER_MIN", 10),
		RegisterPerHour:    integer("RATE_LIMIT_REGISTER_PER_HOUR", 5),
		VoucherPerMinute:   integer("RATE_LIMIT_VOUCHER_PER_MIN", 20),
		OrderPerMinute:     integer("RATE_LIMIT_ORDER_PER_MIN", 10),
		LoginFailPerWindow: integer("LOGIN_FAIL_THRESHOLD", 5),
		LoginFailWindow:    duration("LOGIN_FAIL_WINDOW", 15*time.Minute),
		LoginLockout:       duration("LOGIN_LOCKOUT", 15*time.Minute),
	}

	cfg.Payment = PaymentConfig{
		Provider:          str("PAYMENT_PROVIDER", "midtrans"),
		MidtransEnv:       str("MIDTRANS_ENV", "sandbox"),
		MidtransServerKey: str("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey: str("MIDTRANS_CLIENT_KEY", ""),
		RequireSignature:  boolean("MIDTRANS_WEBHOOK_SIGNATURE_REQUIRED", true),
	}

	cfg.Shipping = ShippingConfig{
		Provider:        str("SHIPPING_PROVIDER", "biteship"),
		BiteshipAPIKey:  str("BITESHIP_API_KEY", ""),
		BiteshipBaseURL: str("BITESHIP_BASE_URL", "https://api.biteship.com"),
		OriginAreaID:    str("BITESHIP_ORIGIN_AREA_ID", ""),
		RatesCacheTTL:   duration("BITESHIP_RATES_CACHE_TTL", 5*time.Minute), // §6.4 cache 5 menit
	}

	cfg.SMTP = SMTPConfig{
		Host:    str("SMTP_HOST", "localhost"),
		Port:    integer("SMTP_PORT", 1025),
		User:    str("SMTP_USER", ""),
		Pass:    str("SMTP_PASS", ""),
		From:    str("SMTP_FROM", "KALENGO <no-reply@kalengo.test>"),
		UseTLS:  boolean("SMTP_TLS", false),
		Timeout: duration("SMTP_TIMEOUT", 10*time.Second),
	}

	cfg.S3 = S3Config{
		Endpoint:     str("S3_ENDPOINT", ""),
		Region:       str("S3_REGION", "us-east-1"),
		Bucket:       str("S3_BUCKET", "kalengo-media"),
		AccessKey:    str("S3_ACCESS_KEY", ""),
		SecretKey:    str("S3_SECRET_KEY", ""),
		UsePathStyle: boolean("S3_USE_PATH_STYLE", true),
	}

	cfg.Push = PushConfig{
		VAPIDPublicKey:  str("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey: str("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:    str("VAPID_SUBJECT", "mailto:admin@kalengo.test"),
	}

	cfg.Obs = ObsConfig{
		SentryDSN:      str("SENTRY_DSN", ""),
		OTLPEndpoint:   str("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		MetricsEnabled: boolean("METRICS_ENABLED", true),
	}

	cfg.Order = OrderConfig{
		// §6.4: stok direservasi 15 menit
		ReservationMinutes:  minutes("ORDER_RESERVATION_MINUTES", 15*time.Minute),
		ExpirySweepInterval: duration("ORDER_EXPIRY_SWEEP_INTERVAL", time.Minute),
	}

	errs = append(errs, cfg.validate()...)
	if joined := errors.Join(errs...); joined != nil {
		return nil, fmt.Errorf("konfigurasi tidak valid: %w", joined)
	}
	return cfg, nil
}

func (c *Config) validate() []error {
	var errs []error
	req := func(name, value string, minLen int) {
		if value == "" {
			errs = append(errs, fmt.Errorf("%s wajib diisi", name))
			return
		}
		if minLen > 0 && len(value) < minLen {
			errs = append(errs, fmt.Errorf("%s minimal %d karakter (saat ini %d)", name, minLen, len(value)))
		}
	}

	req("DATABASE_URL", c.DB.URL, 0)
	if c.DB.URL != "" && !c.DB.URLIsSafeForRuntime() {
		// Peringatan keras: role runtime tidak boleh bisa DDL (§13.1).
		errs = append(errs, errors.New("DATABASE_URL tampak memakai role migrator/superuser; runtime wajib memakai role least-privilege (lihat docs/DB-NOTES.md)"))
	}
	if c.DB.MaxConns < 1 {
		errs = append(errs, errors.New("DB_MAX_CONNS minimal 1"))
	}
	if c.DB.MinConns > c.DB.MaxConns {
		errs = append(errs, errors.New("DB_MIN_CONNS tidak boleh lebih besar dari DB_MAX_CONNS"))
	}

	if c.IsProduction() {
		req("JWT_SECRET", c.Auth.JWTSecret, 32)
		if !c.Auth.CookieSecure {
			errs = append(errs, errors.New("COOKIE_SECURE wajib true di produksi"))
		}
		if len(c.App.WebOrigin) == 0 {
			errs = append(errs, errors.New("WEB_ORIGIN wajib diisi di produksi"))
		}
	} else if c.Auth.JWTSecret == "" {
		// Nilai dev yang jelas-jelas bukan rahasia; tetap dicatat di log startup.
		c.Auth.JWTSecret = "dev_only_secret_change_me_min_32_bytes_long!!"
	}

	if _, err := url.Parse(c.App.BaseURL); err != nil {
		errs = append(errs, fmt.Errorf("BASE_URL tidak valid: %w", err))
	}
	for _, o := range c.App.WebOrigin {
		if _, err := url.Parse(o); err != nil {
			errs = append(errs, fmt.Errorf("WEB_ORIGIN %q tidak valid: %w", o, err))
		}
	}
	if c.Argon2.Parallelism == 0 || c.Argon2.Time == 0 || c.Argon2.MemoryKiB == 0 {
		errs = append(errs, errors.New("parameter ARGON2_* tidak boleh nol"))
	}
	if c.Order.ReservationMinutes <= 0 {
		errs = append(errs, errors.New("ORDER_RESERVATION_MINUTES harus > 0"))
	}
	if c.Payment.Provider == "midtrans" && c.IsProduction() {
		req("MIDTRANS_SERVER_KEY", c.Payment.MidtransServerKey, 0)
		req("MIDTRANS_CLIENT_KEY", c.Payment.MidtransClientKey, 0)
	}
	if c.Shipping.Provider == "biteship" && c.IsProduction() {
		req("BITESHIP_API_KEY", c.Shipping.BiteshipAPIKey, 0)
		req("BITESHIP_ORIGIN_AREA_ID", c.Shipping.OriginAreaID, 0)
	}
	return errs
}

// IsProduction melaporkan apakah aplikasi berjalan di produksi.
func (c *Config) IsProduction() bool { return c.Env == EnvProduction }

// IsDevelopment melaporkan apakah aplikasi berjalan di mesin pengembang.
func (c *Config) IsDevelopment() bool { return c.Env == EnvDevelopment }

// URLIsSafeForRuntime memastikan DATABASE_URL tidak memakai role superuser/migrator.
func (c *DBConfig) URLIsSafeForRuntime() bool {
	u, err := url.Parse(c.URL)
	if err != nil {
		return true // biarkan validator URL lain yang melaporkan
	}
	switch u.User.Username() {
	case "postgres", "kalengo_migrator", "supabase_admin", "root":
		return false
	default:
		return true
	}
}

// Redacted mengembalikan salinan konfigurasi yang aman dicatat ke log (§13.1).
func (c *Config) Redacted() map[string]any {
	return map[string]any{
		"env":               c.Env,
		"app_name":          c.App.Name,
		"http_addr":         c.HTTP.Addr,
		"log_level":         c.App.LogLevel.String(),
		"db_url_role":       dbRole(c.DB.URL),
		"redis_url":         maskURL(c.Redis.URL),
		"web_origin":        c.App.WebOrigin,
		"cookie_secure":     c.Auth.CookieSecure,
		"jwt_secret_set":    c.Auth.JWTSecret != "",
		"payment_provider":  c.Payment.Provider,
		"shipping_provider": c.Shipping.Provider,
		"midtrans_env":      c.Payment.MidtransEnv,
		"version":           c.Build.Version,
		"commit":            c.Build.Commit,
	}
}

func dbRole(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return "unknown"
	}
	return u.User.Username()
}

func maskURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.Redacted()
}

// ---------- pembantu pembacaan environment ----------

func str(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return def
}

func integer(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		slog.Warn("nilai environment bukan angka, memakai nilai default", "key", key, "value", raw, "default", def)
		return def
	}
	return v
}

func boolean(key string, def bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		slog.Warn("nilai environment bukan boolean, memakai nilai default", "key", key, "value", raw, "default", def)
		return def
	}
	return v
}

// Duration menerima format Go ("15m", "720h") dan bentuk angka polos (detik).
func duration(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		slog.Warn("nilai environment bukan durasi, memakai nilai default", "key", key, "value", raw, "default", def.String())
		return def
	}
	return v
}

// minutes sama seperti duration, tetapi angka polos berarti MENIT
// (dipakai untuk ORDER_RESERVATION_MINUTES sesuai §6.4).
func minutes(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	if mins, err := strconv.Atoi(raw); err == nil {
		return time.Duration(mins) * time.Minute
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		slog.Warn("nilai environment bukan durasi/menit, memakai nilai default", "key", key, "value", raw, "default", def.String())
		return def
	}
	return v
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL %q tidak dikenal (debug|info|warn|error)", raw)
	}
}
