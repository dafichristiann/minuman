// Package valid memuat validasi input yang dipakai lebih dari satu layanan.
//
// Semua fungsi mengembalikan *apperr.Error dengan kode API_VALIDATION dan pesan
// per-field, sehingga bentuk respons §12 konsisten di seluruh API.
package valid

import (
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/kalengo/api/internal/apperr"
)

func invalid(field, message string) *apperr.Error {
	return apperr.New(apperr.CodeAPIValidation, nil).WithField(field, message)
}

// Text memeriksa panjang teks (dihitung dalam rune, bukan byte) dan memangkas
// spasi di ujung.
func Text(field, value string, min, max int) (string, error) {
	trimmed := strings.TrimSpace(value)
	n := len([]rune(trimmed))
	if n < min {
		return "", invalid(field, "wajib diisi")
	}
	if n > max {
		return "", invalid(field, "maksimal "+itoa(max)+" karakter")
	}
	return trimmed, nil
}

// OptionalText sama seperti Text tetapi mengizinkan string kosong → nil.
func OptionalText(field, value string, max int) (*string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	if len([]rune(trimmed)) > max {
		return nil, invalid(field, "maksimal "+itoa(max)+" karakter")
	}
	return &trimmed, nil
}

// Email memvalidasi dan menormalkan alamat email (trim + lowercase).
//
// Normalisasi di aplikasi membuat perbandingan (mis. rate limit per akun)
// konsisten dengan kolom citext di database.
func Email(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("email", "wajib diisi")
	}
	if len(trimmed) > 254 {
		return "", invalid("email", "terlalu panjang")
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return "", invalid("email", "format email tidak valid")
	}
	at := strings.LastIndex(trimmed, "@")
	if at <= 0 || !strings.Contains(trimmed[at:], ".") {
		return "", invalid("email", "format email tidak valid")
	}
	return strings.ToLower(trimmed), nil
}

// FullName memvalidasi nama tampilan (2–100 karakter, §11.2).
func FullName(raw string) (string, error) {
	return Text("full_name", raw, 2, 100)
}

// Phone menormalkan nomor telepon Indonesia ke bentuk +62.
// Nilai kosong menghasilkan nil (kolom boleh NULL).
func Phone(raw string) (*string, error) {
	digits := make([]rune, 0, 16)
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9', r == '+':
			digits = append(digits, r)
		case r == ' ', r == '-', r == '(', r == ')':
			// pemisah diabaikan
		default:
			return nil, invalid("phone", "hanya angka yang diperbolehkan")
		}
	}
	s := string(digits)
	if s == "" {
		return nil, nil
	}
	switch {
	case strings.HasPrefix(s, "+62"):
	case strings.HasPrefix(s, "62"):
		s = "+" + s
	case strings.HasPrefix(s, "0"):
		s = "+62" + strings.TrimPrefix(s, "0")
	case strings.HasPrefix(s, "8"):
		s = "+62" + s
	default:
		return nil, invalid("phone", "nomor telepon tidak valid")
	}
	if len(s) < 10 || len(s) > 16 {
		return nil, invalid("phone", "nomor telepon tidak valid")
	}
	return &s, nil
}

// PhoneRequired sama seperti Phone tetapi menolak nilai kosong (alamat kirim).
func PhoneRequired(field, raw string) (string, error) {
	p, err := Phone(raw)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", invalid(field, "wajib diisi")
	}
	return *p, nil
}

// BirthDate memastikan tanggal lahir wajar: tidak di masa depan dan usia ≤ 120.
func BirthDate(raw string, now time.Time) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, invalid("birth_date", "format tanggal harus YYYY-MM-DD")
	}
	d = d.UTC()
	if d.After(now.UTC()) {
		return nil, invalid("birth_date", "tanggal lahir tidak boleh di masa depan")
	}
	if d.Before(now.UTC().AddDate(-120, 0, 0)) {
		return nil, invalid("birth_date", "tanggal lahir tidak wajar")
	}
	return &d, nil
}

// PostalCode memvalidasi kode pos Indonesia (5 digit).
func PostalCode(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) != 5 {
		return "", invalid("postal_code", "kode pos harus 5 digit")
	}
	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return "", invalid("postal_code", "kode pos harus 5 digit angka")
		}
	}
	return trimmed, nil
}

// EscapeLike melindungi pola LIKE dari wildcard yang diketik pengguna
// (`%`, `_`, dan backslash). Dipakai sebelum nilai masuk ke ILIKE $n || '%'.
func EscapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

func itoa(n int) string { return strconv.Itoa(n) }
