package valid

import (
	"strings"
	"testing"
	"time"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"normal", "budi@contoh.com", "budi@contoh.com", false},
		{"huruf besar & spasi", "  Budi@Contoh.COM  ", "budi@contoh.com", false},
		{"tanpa domain", "budi@contoh", "", true},
		{"tanpa at", "budi.contoh.com", "", true},
		{"kosong", "   ", "", true},
		{"terlalu panjang", "a@" + strings.Repeat("a", 260) + ".com", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Email(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Email(%q) tidak menghasilkan error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Email(%q) error = %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("Email(%q) = %q, ingin %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestPhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantNil bool
		wantErr bool
	}{
		{"lokal dengan pemisah", "0812-3456-7890", "+6281234567890", false, false},
		{"tanpa nol", "8123456789", "+628123456789", false, false},
		{"kode negara tanpa plus", "6281234567890", "+6281234567890", false, false},
		{"sudah plus", "+6281234567890", "+6281234567890", false, false},
		{"kosong", "", "", true, false},
		{"huruf", "0812abc", "", true, true},
		{"terlalu pendek", "0812", "", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Phone(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Phone(%q) tidak menghasilkan error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Phone(%q) error = %v", tc.input, err)
			}
			if tc.wantNil {
				if got != nil {
					t.Fatalf("Phone(%q) = %v, ingin nil", tc.input, *got)
				}
				return
			}
			if got == nil || *got != tc.want {
				t.Fatalf("Phone(%q) = %v, ingin %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestPhoneRequired(t *testing.T) {
	if _, err := PhoneRequired("phone", "   "); err == nil {
		t.Fatal("PhoneRequired menerima string kosong")
	}
	got, err := PhoneRequired("phone", "081234567890")
	if err != nil {
		t.Fatalf("PhoneRequired() error = %v", err)
	}
	if got != "+6281234567890" {
		t.Fatalf("PhoneRequired() = %q", got)
	}
}

func TestText(t *testing.T) {
	got, err := Text("label", "  Rumah  ", 1, 10)
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if got != "Rumah" {
		t.Fatalf("Text() = %q, ingin Rumah", got)
	}
	if _, err := Text("label", "   ", 1, 10); err == nil {
		t.Fatal("Text() menerima string kosong")
	}
	if _, err := Text("label", "terlalu panjang", 1, 5); err == nil {
		t.Fatal("Text() menerima string terlalu panjang")
	}
}

func TestOptionalText(t *testing.T) {
	got, err := OptionalText("notes", "  ", 10)
	if err != nil {
		t.Fatalf("OptionalText() error = %v", err)
	}
	if got != nil {
		t.Fatalf("OptionalText() = %v, ingin nil", *got)
	}
}

func TestPostalCode(t *testing.T) {
	if _, err := PostalCode("12345"); err != nil {
		t.Fatalf("PostalCode() error = %v", err)
	}
	for _, in := range []string{"1234", "123456", "1234a"} {
		if _, err := PostalCode(in); err == nil {
			t.Errorf("PostalCode(%q) tidak menghasilkan error", in)
		}
	}
}

func TestBirthDate(t *testing.T) {
	now := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)

	got, err := BirthDate("  ", now)
	if err != nil {
		t.Fatalf("BirthDate(\"\") error = %v", err)
	}
	if got != nil {
		t.Fatalf("BirthDate(\"\") = %v, ingin nil", *got)
	}

	if _, err := BirthDate("2000-01-15", now); err != nil {
		t.Fatalf("BirthDate(valid) error = %v", err)
	}
	if _, err := BirthDate("2030-01-01", now); err == nil {
		t.Fatal("BirthDate menerima tanggal masa depan")
	}
	if _, err := BirthDate("1800-01-01", now); err == nil {
		t.Fatal("BirthDate menerima tanggal tidak wajar")
	}
	if _, err := BirthDate("15-01-2000", now); err == nil {
		t.Fatal("BirthDate menerima format salah")
	}
}

func TestEscapeLike(t *testing.T) {
	got := EscapeLike(`100%_kopi\susu`)
	want := `100\%\_kopi\\susu`
	if got != want {
		t.Fatalf("EscapeLike() = %q, ingin %q", got, want)
	}
}
