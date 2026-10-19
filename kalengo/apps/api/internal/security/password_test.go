package security

import (
	"errors"
	"strings"
	"testing"
)

// fastParams menurunkan biaya argon2id agar unit test tetap cepat. Parameter
// produksi diuji hanya lewat validasi bentuk, bukan lewat waktu komputasi.
func fastParams() Argon2Params {
	return Argon2Params{Memory: 8 * 1024, Time: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("kaleng-enak-2026", fastParams())
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("hash tidak berformat PHC argon2id: %q", hash)
	}

	ok, err := VerifyPassword(hash, "kaleng-enak-2026")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatal("password benar ditolak")
	}

	ok, err = VerifyPassword(hash, "kaleng-enak-2027")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if ok {
		t.Fatal("password salah diterima")
	}
}

func TestHashPasswordUsesFreshSalt(t *testing.T) {
	a, err := HashPassword("kaleng-enak-2026", fastParams())
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	b, err := HashPassword("kaleng-enak-2026", fastParams())
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if a == b {
		t.Fatal("dua hash identik: salt tidak acak")
	}
}

func TestHashPasswordEmbedsParams(t *testing.T) {
	params := Argon2Params{Memory: 16 * 1024, Time: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	hash, err := HashPassword("kaleng-enak-2026", params)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.Contains(hash, "m=16384,t=2,p=1") {
		t.Fatalf("parameter tidak tersimpan di hash: %q", hash)
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"terlalu pendek", "pendek", ErrPasswordTooShort},
		{"pas batas", "0123456789", nil},
		{"hanya spasi", strings.Repeat(" ", 12), ErrPasswordTooShort},
		{"terlalu panjang", strings.Repeat("a", MaxPasswordBytes+1), ErrPasswordTooLong},
		{"unicode dihitung rune", "kaleng🥤enak", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ValidatePassword() = %v, ingin %v", err, tc.wantErr)
			}
		})
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	cases := []string{
		"",
		"bukan-hash",
		"$argon2id$v=19$m=65536,t=3,p=2$hanya-salt",
		"$argon2i$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0c2E$a2V5a2V5a2V5a2V5a2V5aw",
		"$argon2id$v=18$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0c2E$a2V5a2V5a2V5a2V5a2V5aw",
	}
	for _, encoded := range cases {
		if _, err := VerifyPassword(encoded, "apa-saja-yang-panjang"); !errors.Is(err, ErrHashMalformed) {
			t.Errorf("VerifyPassword(%q) error = %v, ingin ErrHashMalformed", encoded, err)
		}
	}
}

func TestDummyVerifyNeverPanics(t *testing.T) {
	// DummyVerify dipanggil pada jalur "email tidak ditemukan", sehingga tidak
	// boleh pernah membuat proses gagal.
	for i := 0; i < 3; i++ {
		DummyVerify("apa-saja-yang-panjang", fastParams())
	}
}
