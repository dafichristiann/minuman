// Package security memuat primitif keamanan yang dipakai lintas layanan:
// hashing password (argon2id), token sesi, dan token CSRF (§13.1).
//
// Paket ini TIDAK boleh mengimpor database, HTTP, atau konfigurasi aplikasi;
// tujuannya agar primitifnya mudah diuji dan tidak bisa dipakai salah lapisan.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// MinPasswordLength sesuai §13.1: minimum 10 karakter, tanpa aturan komposisi.
const MinPasswordLength = 10

// MaxPasswordBytes adalah batas atas yang wajar. Bukan "pembatasan karakter"
// seperti di §13.1, melainkan perlindungan DoS: argon2id memproses seluruh
// panjang input, sehingga body 1 MB tidak boleh berubah menjadi CPU 1 MB.
const MaxPasswordBytes = 1024

// Errors yang bisa dibedakan pemanggil.
var (
	ErrPasswordTooShort = errors.New("security: password terlalu pendek")
	ErrPasswordTooLong  = errors.New("security: password terlalu panjang")
	ErrHashMalformed    = errors.New("security: hash argon2id tidak dikenali")
)

// Argon2Params mencerminkan kolom parameter di config.Argon2Config.
// Parameter disimpan DI DALAM hash sehingga menaikkan biaya tidak memutus
// verifikasi hash lama (§13.1).
type Argon2Params struct {
	Memory      uint32 // KiB
	Time        uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params adalah nilai aman bila pemanggil tidak mengonfigurasi.
var DefaultArgon2Params = Argon2Params{
	Memory:      64 * 1024,
	Time:        3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

var phcEncoding = base64.RawStdEncoding

// ValidatePassword menerapkan kebijakan password §13.1 tanpa aturan komposisi.
func ValidatePassword(password string) error {
	if len([]rune(password)) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	// Password yang hanya berisi spasi hampir selalu hasil salah ketik.
	if strings.TrimSpace(password) == "" {
		return ErrPasswordTooShort
	}
	return nil
}

// HashPassword menghasilkan string PHC:
//
//	$argon2id$v=19$m=65536,t=3,p=2$<salt-b64>$<key-b64>
func HashPassword(password string, p Argon2Params) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	p = normalizeParams(p)

	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security: gagal membuat salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Parallelism, p.KeyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Parallelism,
		phcEncoding.EncodeToString(salt), phcEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword memeriksa password terhadap hash PHC. Waktu eksekusi selalu
// sebanding dengan parameter hash, bukan dengan benar/salahnya password, dan
// perbandingan memakai subtle.ConstantTimeCompare.
func VerifyPassword(encoded, password string) (bool, error) {
	params, salt, want, err := parsePHC(encoded)
	if err != nil {
		return false, err
	}

	got := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Parallelism, params.KeyLength)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// DummyVerify melakukan hashing dengan biaya yang sama seperti verifikasi
// sungguhan. Dipakai saat email tidak ditemukan supaya penyerang tidak bisa
// membedakan "email tidak ada" dari "password salah" lewat waktu respons
// (§13.1: pesan generik + waktu respons konstan).
func DummyVerify(password string, p Argon2Params) {
	p = normalizeParams(p)
	_, _ = VerifyPassword(onceDummyHash(p), password)
}

var (
	dummyOnce sync.Once
	dummyHash string
)

// onceDummyHash dihitung sekali per proses (bukan per request) agar tidak
// menambah ~50 ms ke setiap login yang gagal.
func onceDummyHash(p Argon2Params) string {
	dummyOnce.Do(func() {
		h, err := HashPassword("kalengo-dummy-password-not-a-secret", p)
		if err != nil {
			// Tidak mungkin terjadi: konstanta di atas valid.
			panic(err)
		}
		dummyHash = h
	})
	return dummyHash
}

func normalizeParams(p Argon2Params) Argon2Params {
	if p.Memory == 0 {
		p.Memory = DefaultArgon2Params.Memory
	}
	if p.Time == 0 {
		p.Time = DefaultArgon2Params.Time
	}
	if p.Parallelism == 0 {
		p.Parallelism = DefaultArgon2Params.Parallelism
	}
	if p.SaltLength == 0 {
		p.SaltLength = DefaultArgon2Params.SaltLength
	}
	if p.KeyLength == 0 {
		p.KeyLength = DefaultArgon2Params.KeyLength
	}
	return p
}

// parsePHC membongkar format PHC secara ketat. Input yang aneh tidak boleh
// membuat alokasi besar: semua nilai dibatasi.
func parsePHC(encoded string) (Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, key]
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}

	var p Argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Parallelism); err != nil {
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}
	if p.Memory == 0 || p.Memory > 1<<21 { // maks 2 GiB
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}
	if p.Time == 0 || p.Time > 32 {
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}
	if p.Parallelism == 0 || p.Parallelism > 64 {
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}

	salt, err := phcEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}
	key, err := phcEncoding.DecodeString(parts[5])
	if err != nil || len(key) < 16 || len(key) > 128 {
		return Argon2Params{}, nil, nil, ErrHashMalformed
	}

	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(key))
	return p, salt, key, nil
}
