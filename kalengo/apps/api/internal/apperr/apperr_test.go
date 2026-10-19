package apperr

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestNewRendersMessageWithVars(t *testing.T) {
	err := New(CodeStockChangedCheckout, map[string]string{"produk": "Kopi Susu", "n": "3"})

	if err.Code != CodeStockChangedCheckout {
		t.Errorf("Code = %q, ingin %q", err.Code, CodeStockChangedCheckout)
	}
	want := "Stok Kopi Susu tinggal 3. Sesuaikan jumlahnya."
	if err.Message != want {
		t.Errorf("Message = %q, ingin %q", err.Message, want)
	}
	if err.Status != 409 {
		t.Errorf("Status = %d, ingin 409", err.Status)
	}
	if err.Severity != SeverityError {
		t.Errorf("Severity = %q, ingin %q", err.Severity, SeverityError)
	}
	if err.Category != CategoryCatalog {
		t.Errorf("Category = %q, ingin %q", err.Category, CategoryCatalog)
	}
}

// Placeholder tanpa nilai dibiarkan apa adanya supaya kekurangan variabel
// terlihat saat review, bukan menghasilkan pesan kosong yang membingungkan.
func TestRenderLeavesMissingPlaceholders(t *testing.T) {
	err := New(CodeStockChangedCheckout, map[string]string{"produk": "Teh Manis"})

	if !strings.Contains(err.Message, "{n}") {
		t.Errorf("placeholder tanpa nilai harus dibiarkan: %q", err.Message)
	}
}

func TestRenderWithNoVars(t *testing.T) {
	got := Render("Halo {nama}", nil)
	if got != "Halo {nama}" {
		t.Errorf("Render tanpa vars = %q", got)
	}
}

// Kode yang tidak terdaftar tidak boleh bocor sebagai kode aneh ke klien.
func TestNewFallsBackToServerErrorForUnknownCode(t *testing.T) {
	err := New("KODE_YANG_TIDAK_ADA", nil)

	if err.Code != CodeAPIServerError {
		t.Errorf("Code = %q, ingin %q", err.Code, CodeAPIServerError)
	}
	if err.Status != 500 {
		t.Errorf("Status = %d, ingin 500", err.Status)
	}
}

func TestWrapKeepsCause(t *testing.T) {
	cause := errors.New("koneksi putus")
	err := Wrap(cause, CodeAPIServerError, nil)

	if !errors.Is(err, cause) {
		t.Error("Wrap harus mempertahankan error asal untuk errors.Is")
	}
	if err.Error() == "" {
		t.Error("Error() tidak boleh kosong")
	}
}

func TestWrapNilReturnsNil(t *testing.T) {
	if got := Wrap(nil, CodeAPIServerError, nil); got != nil {
		t.Errorf("Wrap(nil) = %v, ingin nil", got)
	}
}

func TestFromMapsUnknownErrorToServerError(t *testing.T) {
	appErr := From(errors.New("boom"))

	if appErr.Code != CodeAPIServerError {
		t.Errorf("Code = %q, ingin %q", appErr.Code, CodeAPIServerError)
	}
	if appErr.Message == "" {
		t.Error("Message tidak boleh kosong")
	}
}

func TestFromPassesThroughKnownError(t *testing.T) {
	original := New(CodeVoucherExpired, nil)
	got := From(original)

	if got != original {
		t.Error("From harus mengembalikan *Error yang sama")
	}
}

func TestFromWrappedChain(t *testing.T) {
	inner := New(CodeCartEmpty, nil)
	outer := fmt.Errorf("handler gagal: %w", inner)

	got := From(outer)
	if got.Code != CodeCartEmpty {
		t.Errorf("Code = %q, ingin %q (errors.As harus menembus wrap)", got.Code, CodeCartEmpty)
	}
}

func TestIsAndStatus(t *testing.T) {
	err := New(CodeAPIRateLimit, map[string]string{"detik": "12"})

	if !Is(err, CodeAPIRateLimit) {
		t.Error("Is harus true untuk kode yang sama")
	}
	if Is(err, CodeAPIServerError) {
		t.Error("Is tidak boleh true untuk kode lain")
	}
	if got := Status(err); got != 429 {
		t.Errorf("Status = %d, ingin 429", got)
	}
	if got := Status(nil); got != 200 {
		t.Errorf("Status(nil) = %d, ingin 200", got)
	}
	if got := Status(errors.New("x")); got != 500 {
		t.Errorf("Status(error biasa) = %d, ingin 500", got)
	}
}

func TestWithFieldsAccumulates(t *testing.T) {
	err := New(CodeAPIValidation, nil).
		WithField("email", "Format email tidak valid").
		WithFields(map[string]string{"phone": "Terlalu panjang"})

	if len(err.Fields) != 2 {
		t.Fatalf("Fields = %#v, ingin 2 entri", err.Fields)
	}
	if err.Fields["email"] == "" || err.Fields["phone"] == "" {
		t.Errorf("Fields tidak lengkap: %#v", err.Fields)
	}
}

func TestWithMeta(t *testing.T) {
	err := New(CodeStockOut, nil).WithMeta("variant_id", "abc")
	if err.Meta["variant_id"] != "abc" {
		t.Errorf("Meta tidak tersimpan: %#v", err.Meta)
	}
}

func TestLogValueContainsCode(t *testing.T) {
	err := New(CodePayExpired, nil)
	val := err.LogValue()

	found := false
	for _, attr := range val.Group() {
		if attr.Key == "code" && attr.Value.String() == CodePayExpired {
			found = true
		}
	}
	if !found {
		t.Error("LogValue harus memuat kode")
	}
}

// ---------------------------------------------------------------
// Integritas registry: menjaga §8.6 tetap sinkron dan mencegah kode "hantu".
// ---------------------------------------------------------------
func TestRegistryIsConsistent(t *testing.T) {
	codes := Codes()
	if len(codes) < 40 {
		t.Fatalf("registry hanya berisi %d kode; terlalu sedikit untuk katalog §8.6", len(codes))
	}

	for _, code := range codes {
		def, ok := Lookup(code)
		if !ok {
			t.Errorf("Lookup(%q) gagal padahal ada di Codes()", code)
			continue
		}
		if def.Code != code {
			t.Errorf("kunci registry %q tidak sama dengan Definition.Code %q", code, def.Code)
		}
		if def.Status < 400 || def.Status > 599 {
			t.Errorf("%s: status %d di luar 4xx/5xx", code, def.Status)
		}
		if strings.TrimSpace(def.Message) == "" {
			t.Errorf("%s: pesan kosong", code)
		}
		switch def.Severity {
		case SeveritySuccess, SeverityInfo, SeverityWarning, SeverityError, SeverityCritical:
		default:
			t.Errorf("%s: severity %q tidak dikenal", code, def.Severity)
		}
		switch def.Category {
		case CategoryAuth, CategoryAccount, CategoryCatalog, CategoryVoucher,
			CategoryCheckout, CategoryPayment, CategoryOrder, CategorySystem, CategoryAdmin:
		default:
			t.Errorf("%s: kategori %q tidak dikenal", code, def.Category)
		}
	}
}

func TestCodesAreSortedAndUnique(t *testing.T) {
	codes := Codes()
	if !sort.StringsAreSorted(codes) {
		t.Error("Codes() harus terurut agar diff katalog stabil")
	}
	seen := make(map[string]bool, len(codes))
	for _, c := range codes {
		if seen[c] {
			t.Errorf("kode duplikat: %s", c)
		}
		seen[c] = true
	}
}

// Kode yang dipakai §12 sebagai contoh harus benar-benar ada.
func TestRequiredCodesExist(t *testing.T) {
	required := []string{
		CodeCartPriceChanged, CodeStockChangedCheckout, CodeAPIRateLimit,
		CodeAPIServerError, CodeAPIValidation, CodeAPIForbidden, CodeAPINotFound,
		CodeAuthLoginFail, CodeAuthSessionExpired, CodeVoucherInvalid,
		CodeChkAddressReq, CodePayExpired, CodeOrdCancelDenied,
		CodeRevNotEligible, CodeAgeGate, CodeAdmStockReasonReq,
	}
	for _, code := range required {
		if _, ok := Lookup(code); !ok {
			t.Errorf("kode wajib %q tidak terdaftar", code)
		}
	}
}

// Login gagal harus selalu generik agar tidak ada user enumeration (§13.1).
func TestLoginFailMessageIsGeneric(t *testing.T) {
	def, ok := Lookup(CodeAuthLoginFail)
	if !ok {
		t.Fatal("AUTH_LOGIN_FAIL harus terdaftar")
	}
	lower := strings.ToLower(def.Message)
	for _, forbidden := range []string{"email tidak", "password salah", "belum terdaftar", "tidak ada"} {
		if strings.Contains(lower, forbidden) {
			t.Errorf("pesan AUTH_LOGIN_FAIL membocorkan informasi: %q", def.Message)
		}
	}
}

// Kode keamanan/pembayaran wajib masuk Notification Center (§8.2).
func TestCriticalCodesArePersisted(t *testing.T) {
	persisted := []string{
		CodeAuthLoginLocked, CodeAuthSessionExpired, CodeAuthRefreshReuse,
		CodeAuthEmailUnverified, CodePayFailed, CodePayExpired,
		CodeAPIServerError, CodeAdmAuditChainBroken, CodeAgeGate,
		CodeStockChangedCheckout, CodeAPIForbidden,
	}
	for _, code := range persisted {
		def, ok := Lookup(code)
		if !ok {
			t.Errorf("%s tidak terdaftar", code)
			continue
		}
		if !def.Persist {
			t.Errorf("%s harus Persist=true agar tersimpan di Notification Center", code)
		}
	}
}

// Notifikasi gangguan sesaat TIDAK boleh masuk Notification Center,
// supaya riwayat pengguna tidak dipenuhi kebisingan.
func TestTransientCodesAreNotPersisted(t *testing.T) {
	transient := []string{
		CodeAPIRateLimit, CodeAPIServiceUnavailable, CodeChkShippingFail,
		CodeAPITimeout, CodeChkDuplicateBlocked, CodePayProviderError,
	}
	for _, code := range transient {
		def, ok := Lookup(code)
		if !ok {
			t.Errorf("%s tidak terdaftar", code)
			continue
		}
		if def.Persist {
			t.Errorf("%s sebaiknya Persist=false karena bersifat sementara", code)
		}
	}
}
