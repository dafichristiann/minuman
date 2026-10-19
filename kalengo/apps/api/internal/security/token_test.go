package security

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testTokens() *TokenService {
	return NewTokenService("rahasia-uji-yang-cukup-panjang-untuk-hs256", "kalengo", 15*time.Minute)
}

func TestIssueAndParseAccess(t *testing.T) {
	ts := testTokens()
	userID := uuid.New()

	raw, expires, err := ts.IssueAccess(userID, "customer")
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}
	if !expires.After(time.Now()) {
		t.Fatalf("ExpiresAt = %v, harus di masa depan", expires)
	}

	claims, err := ts.ParseAccess(raw)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, ingin %v", claims.UserID, userID)
	}
	if claims.Role != "customer" {
		t.Errorf("Role = %q, ingin customer", claims.Role)
	}
}

func TestParseAccessRejectsExpired(t *testing.T) {
	ts := testTokens()
	raw, _, err := ts.IssueAccess(uuid.New(), "customer")
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}

	ts.SetClock(func() time.Time { return time.Now().Add(16 * time.Minute) })
	if _, err := ts.ParseAccess(raw); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("ParseAccess() error = %v, ingin ErrTokenExpired", err)
	}
}

func TestParseAccessRejectsTamperedSignature(t *testing.T) {
	ts := testTokens()
	raw, _, err := ts.IssueAccess(uuid.New(), "customer")
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}

	// Ubah satu karakter di bagian tanda tangan.
	tampered := raw[:len(raw)-1] + "A"
	if tampered == raw {
		tampered = raw[:len(raw)-1] + "B"
	}
	if _, err := ts.ParseAccess(tampered); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("ParseAccess(tampered) error = %v, ingin ErrTokenInvalid", err)
	}
}

func TestParseAccessRejectsAlgNone(t *testing.T) {
	ts := testTokens()
	// Header palsu dengan alg "none" tidak boleh diterima karena header
	// dibandingkan persis dengan yang kita terbitkan.
	forged := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ4In0."
	if _, err := ts.ParseAccess(forged); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("ParseAccess(alg none) error = %v, ingin ErrTokenInvalid", err)
	}
}

func TestParseAccessRejectsOtherIssuer(t *testing.T) {
	issuerA := NewTokenService("rahasia-uji-yang-cukup-panjang-untuk-hs256", "kalengo", time.Minute)
	issuerB := NewTokenService("rahasia-uji-yang-cukup-panjang-untuk-hs256", "penyusup", time.Minute)

	raw, _, err := issuerB.IssueAccess(uuid.New(), "customer")
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}
	if _, err := issuerA.ParseAccess(raw); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("ParseAccess() error = %v, ingin ErrTokenInvalid", err)
	}
}

func TestParseAccessRejectsEmpty(t *testing.T) {
	if _, err := testTokens().ParseAccess(""); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("ParseAccess(\"\") error = %v, ingin ErrTokenInvalid", err)
	}
}

func TestOpaqueTokenRoundTrip(t *testing.T) {
	plain, hash, err := NewOpaqueToken()
	if err != nil {
		t.Fatalf("NewOpaqueToken() error = %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("panjang hash = %d, ingin 32", len(hash))
	}

	again, err := HashOpaqueTokenString(plain)
	if err != nil {
		t.Fatalf("HashOpaqueTokenString() error = %v", err)
	}
	if string(again) != string(hash) {
		t.Fatal("hash dari token yang sama berbeda")
	}
}

func TestHashOpaqueTokenStringRejectsWrongShape(t *testing.T) {
	cases := []string{"", "bukan-token", "c2FsdHQ", "!!!", "AAAA"}
	for _, in := range cases {
		if _, err := HashOpaqueTokenString(in); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("HashOpaqueTokenString(%q) error = %v, ingin ErrTokenInvalid", in, err)
		}
	}
}

func TestValidCSRF(t *testing.T) {
	token, err := NewCSRFToken()
	if err != nil {
		t.Fatalf("NewCSRFToken() error = %v", err)
	}
	if !ValidCSRF(token, token) {
		t.Fatal("token CSRF yang sama ditolak")
	}
	if ValidCSRF(token, "") {
		t.Fatal("header kosong diterima")
	}
	if ValidCSRF("", token) {
		t.Fatal("cookie kosong diterima")
	}
	if ValidCSRF(token, token+"x") {
		t.Fatal("token berbeda panjang diterima")
	}
}
