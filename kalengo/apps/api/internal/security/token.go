package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Errors token. Pemanggil WAJIB membedakan kedaluwarsa dari tidak valid:
// keduanya menghasilkan UX berbeda (AUTH_SESSION_EXPIRED vs AUTH_REFRESH_REUSE).
var (
	ErrTokenInvalid = errors.New("security: token tidak valid")
	ErrTokenExpired = errors.New("security: token kedaluwarsa")
)

// Claims adalah isi access token yang sudah diverifikasi.
type Claims struct {
	UserID    uuid.UUID
	Role      string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// TokenService membuat dan memverifikasi JWT akses (HS256).
//
// JWT dipakai HANYA untuk access token berumur pendek (15 menit) yang disimpan
// di cookie HttpOnly. Otorisasi tetap divalidasi ulang ke database pada setiap
// request (lihat middleware.RequireAuth), sehingga pencabutan sesi, perubahan
// peran, dan penghapusan akun langsung berlaku.
type TokenService struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
	now       func() time.Time
}

// NewTokenService membuat TokenService. `secret` minimal 32 byte (divalidasi
// di config), `issuer` mis. "kalengo".
func NewTokenService(secret, issuer string, accessTTL time.Duration) *TokenService {
	if issuer == "" {
		issuer = "kalengo"
	}
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	return &TokenService{
		secret:    []byte(secret),
		issuer:    issuer,
		accessTTL: accessTTL,
		now:       time.Now,
	}
}

// SetClock mengganti sumber waktu (dipakai test).
func (t *TokenService) SetClock(now func() time.Time) { t.now = now }

// AccessTTL mengembalikan masa berlaku access token.
func (t *TokenService) AccessTTL() time.Duration { return t.accessTTL }

// joseHeader selalu sama; `alg` tidak pernah dibaca dari token sehingga
// serangan "alg: none" tidak mungkin terjadi.
var joseHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

type accessPayload struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Iss  string `json:"iss"`
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
}

// IssueAccess membuat access token untuk pengguna.
func (t *TokenService) IssueAccess(userID uuid.UUID, role string) (string, time.Time, error) {
	if userID == uuid.Nil {
		return "", time.Time{}, fmt.Errorf("security: user id kosong")
	}
	issued := t.now().UTC()
	expires := issued.Add(t.accessTTL)

	payload, err := json.Marshal(accessPayload{
		Sub:  userID.String(),
		Role: role,
		Iss:  t.issuer,
		Iat:  issued.Unix(),
		Exp:  expires.Unix(),
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("security: gagal menyusun claims: %w", err)
	}

	signingInput := joseHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(t.sign(signingInput)), expires, nil
}

// ParseAccess memverifikasi tanda tangan, issuer, dan masa berlaku.
func (t *TokenService) ParseAccess(raw string) (Claims, error) {
	if raw == "" {
		return Claims{}, ErrTokenInvalid
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Claims{}, ErrTokenInvalid
	}
	// Header harus persis header yang kita terbitkan (alg terpaku ke HS256).
	if parts[0] != joseHeader {
		return Claims{}, ErrTokenInvalid
	}

	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}
	wantSig := t.sign(parts[0] + "." + parts[1])
	if !hmac.Equal(gotSig, wantSig) {
		return Claims{}, ErrTokenInvalid
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}
	var p accessPayload
	dec := json.NewDecoder(strings.NewReader(string(rawPayload)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Claims{}, ErrTokenInvalid
	}

	if p.Iss != t.issuer {
		return Claims{}, ErrTokenInvalid
	}
	userID, err := uuid.Parse(p.Sub)
	if err != nil || userID == uuid.Nil {
		return Claims{}, ErrTokenInvalid
	}

	expires := time.Unix(p.Exp, 0).UTC()
	if !t.now().UTC().Before(expires) {
		return Claims{}, ErrTokenExpired
	}

	return Claims{
		UserID:    userID,
		Role:      p.Role,
		IssuedAt:  time.Unix(p.Iat, 0).UTC(),
		ExpiresAt: expires,
	}, nil
}

func (t *TokenService) sign(input string) []byte {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

// ---------- token opaque (refresh & sekali pakai) ----------

// opaqueBytes = 32 byte = 256 bit entropi.
const opaqueBytes = 32

// NewOpaqueToken menghasilkan token acak yang dikirim ke klien dan hash
// SHA-256 yang disimpan di database. Token asli TIDAK PERNAH disimpan (§13.1).
//
// Token dienkode base64url tanpa padding agar aman dipakai di cookie/URL.
func NewOpaqueToken() (plaintext string, hash []byte, err error) {
	buf := make([]byte, opaqueBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, fmt.Errorf("security: gagal membuat token acak: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), HashOpaqueToken(buf), nil
}

// HashOpaqueToken menghitung SHA-256 dari byte mentah token. Dipakai baik saat
// menyimpan maupun saat mencari token yang datang dari klien.
//
// SHA-256 (bukan argon2) cukup di sini karena token punya 256 bit entropi:
// tidak ada ruang untuk brute force seperti pada password manusia.
func HashOpaqueToken(raw []byte) []byte {
	sum := sha256.Sum256(raw)
	return sum[:]
}

// HashOpaqueTokenString mendekode token base64url dari klien lalu menghitung
// hash-nya. Token berbentuk lain dianggap tidak valid.
func HashOpaqueTokenString(plaintext string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(plaintext)
	if err != nil || len(raw) != opaqueBytes {
		return nil, ErrTokenInvalid
	}
	return HashOpaqueToken(raw), nil
}

// ---------- CSRF (double-submit cookie) ----------

// CSRFHeader adalah header yang wajib diisi nilai cookie CSRF.
const CSRFHeader = "X-CSRF-Token"

// NewCSRFToken membuat token CSRF acak 256 bit.
func NewCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("security: gagal membuat token CSRF: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// ValidCSRF membandingkan cookie dan header secara constant-time.
// Nilai kosong selalu ditolak.
func ValidCSRF(cookie, header string) bool {
	if cookie == "" || header == "" || len(cookie) != len(header) {
		return false
	}
	return hmac.Equal([]byte(cookie), []byte(header))
}
