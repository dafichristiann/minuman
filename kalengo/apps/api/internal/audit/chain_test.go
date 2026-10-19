package audit

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func entry(action, entity string, after any) Entry {
	actor := "11111111-1111-1111-1111-111111111111"
	return Entry{
		OccurredAt: time.Date(2026, 10, 19, 9, 30, 0, 0, time.UTC),
		ActorID:    &actor,
		ActorRole:  "admin",
		IP:         "203.0.113.7",
		UserAgent:  "Mozilla/5.0",
		RequestID:  "req-12345678",
		Action:     action,
		Entity:     entity,
		EntityID:   "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Before:     map[string]any{"price": 12000},
		After:      after,
	}
}

// JSON kanonik harus stabil walau urutan kunci berbeda: tanpa ini, verifikasi
// rantai akan gagal palsu hanya karena urutan map berubah.
func TestCanonicalizeIsOrderIndependent(t *testing.T) {
	a := map[string]any{"b": 2, "a": 1, "c": map[string]any{"z": 1, "y": 2}}
	b := map[string]any{"c": map[string]any{"y": 2, "z": 1}, "a": 1, "b": 2}

	ca, err := Canonicalize(a)
	if err != nil {
		t.Fatalf("Canonicalize(a) gagal: %v", err)
	}
	cb, err := Canonicalize(b)
	if err != nil {
		t.Fatalf("Canonicalize(b) gagal: %v", err)
	}
	if !bytes.Equal(ca, cb) {
		t.Errorf("hasil kanonik berbeda:\n a=%s\n b=%s", ca, cb)
	}
	if string(ca) != `{"a":1,"b":2,"c":{"y":2,"z":1}}` {
		t.Errorf("bentuk kanonik tidak sesuai: %s", ca)
	}
}

func TestCanonicalizeHandlesAllScalarTypes(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, "null"},
		{true, "true"},
		{false, "false"},
		{"teks", `"teks"`},
		{int(7), "7"},
		{int64(-9), "-9"},
		{uint(3), "3"},
		{3.5, "3.5"},
		{[]any{1, "dua", nil}, `[1,"dua",null]`},
	}
	for _, tc := range cases {
		got, err := Canonicalize(tc.in)
		if err != nil {
			t.Errorf("Canonicalize(%#v) gagal: %v", tc.in, err)
			continue
		}
		if string(got) != tc.want {
			t.Errorf("Canonicalize(%#v) = %s, ingin %s", tc.in, got, tc.want)
		}
	}
}

func TestCanonicalizeRejectsUnsupportedType(t *testing.T) {
	if _, err := Canonicalize(make(chan int)); err == nil {
		t.Error("tipe yang tidak bisa jadi JSON harus ditolak")
	}
}

// Nilai dari PostgreSQL jsonb datang sebagai []byte; harus bisa dikanonikalisasi.
func TestCanonicalizeAcceptsJSONBytes(t *testing.T) {
	got, err := Canonicalize([]byte(`{"z":1,"a":2}`))
	if err != nil {
		t.Fatalf("Canonicalize([]byte) gagal: %v", err)
	}
	if string(got) != `{"a":2,"z":1}` {
		t.Errorf("hasil = %s, ingin urutan kunci terurut", got)
	}
}

func TestChainAndVerifyRoundTrip(t *testing.T) {
	rows := make([]Chained, 0, 3)

	var prev []byte
	for i, action := range []string{"product.price.update", "stock.adjust", "order.status.change"} {
		e := entry(action, "product", map[string]any{"price": 12000 + i})
		chained, err := e.Chain(prev)
		if err != nil {
			t.Fatalf("Chain ke-%d gagal: %v", i, err)
		}
		rows = append(rows, *chained)
		prev = chained.RowHash
	}

	if err := Verify(rows); err != nil {
		t.Fatalf("Verify gagal pada rantai yang sah: %v", err)
	}
}

func TestVerifyDetectsTamperedRow(t *testing.T) {
	var prev []byte
	e1, err := entry("product.price.update", "product", map[string]any{"price": 12000}).Chain(prev)
	if err != nil {
		t.Fatalf("Chain gagal: %v", err)
	}
	e2, err := entry("product.price.update", "product", map[string]any{"price": 9000}).Chain(e1.RowHash)
	if err != nil {
		t.Fatalf("Chain gagal: %v", err)
	}

	rows := []Chained{*e1, *e2}
	if err := Verify(rows); err != nil {
		t.Fatalf("rantai awal harus sah: %v", err)
	}

	// Penyerang mengubah isi baris tanpa memperbarui hash.
	rows[1].After = map[string]any{"price": 1}

	err = Verify(rows)
	if err == nil {
		t.Fatal("Verify harus mendeteksi baris yang diubah")
	}
	if !strings.Contains(err.Error(), "row_hash") {
		t.Errorf("pesan error kurang informatif: %v", err)
	}
}

func TestVerifyDetectsDeletedRow(t *testing.T) {
	var prev []byte
	var rows []Chained
	for i := 0; i < 3; i++ {
		chained, err := entry("stock.adjust", "variant", map[string]any{"delta": i}).Chain(prev)
		if err != nil {
			t.Fatalf("Chain gagal: %v", err)
		}
		rows = append(rows, *chained)
		prev = chained.RowHash
	}

	// Menghapus baris tengah memutus rantai.
	truncated := []Chained{rows[0], rows[2]}
	err := Verify(truncated)
	if err == nil {
		t.Fatal("Verify harus mendeteksi baris yang hilang")
	}
	if !strings.Contains(err.Error(), "rantai putus") {
		t.Errorf("pesan error kurang informatif: %v", err)
	}
}

func TestVerifyEmptyChainIsValid(t *testing.T) {
	if err := Verify(nil); err != nil {
		t.Errorf("rantai kosong harus sah, dapat %v", err)
	}
}

func TestVerifyHexDecodesHexHashes(t *testing.T) {
	var prev []byte
	e1, err := entry("product.create", "product", nil).Chain(prev)
	if err != nil {
		t.Fatalf("Chain gagal: %v", err)
	}
	e2, err := entry("product.update", "product", nil).Chain(e1.RowHash)
	if err != nil {
		t.Fatalf("Chain gagal: %v", err)
	}

	// Simulasi nilai yang diekspor sebagai heksadesimal (mis. untuk CSV audit):
	// baris pertama tanpa prev_hash, baris kedua memakai hash baris pertama.
	rows := []Chained{
		{
			Entry:    e1.Entry,
			PrevHash: nil,
			RowHash:  []byte(hex.EncodeToString(e1.RowHash)),
		},
		{
			Entry:    e2.Entry,
			PrevHash: []byte(hex.EncodeToString(e1.RowHash)),
			RowHash:  []byte(hex.EncodeToString(e2.RowHash)),
		},
	}

	if err := VerifyHex(rows); err != nil {
		t.Fatalf("VerifyHex gagal: %v", err)
	}
}

func TestVerifyHexRejectsInvalidHex(t *testing.T) {
	rows := []Chained{{RowHash: []byte("bukan-hex!")}}
	if err := VerifyHex(rows); err == nil {
		t.Error("VerifyHex harus menolak hash yang bukan heksadesimal")
	}
}

func TestHashRowIsDeterministicAndChains(t *testing.T) {
	row := []byte(`{"a":1}`)

	first := HashRow(nil, row)
	second := HashRow(nil, row)
	if !bytes.Equal(first, second) {
		t.Error("HashRow harus deterministik")
	}
	if len(first) != 32 {
		t.Errorf("panjang SHA-256 = %d byte, ingin 32", len(first))
	}

	// prev_hash berbeda harus menghasilkan row_hash berbeda.
	other := HashRow([]byte("sebelumnya"), row)
	if bytes.Equal(first, other) {
		t.Error("prev_hash harus memengaruhi hasil hash")
	}
}

func TestHexHelper(t *testing.T) {
	got := Hex([]byte{0x00, 0x0f, 0xff})
	if got != "000fff" {
		t.Errorf("Hex = %q, ingin \"000fff\"", got)
	}
}

// Payload tidak boleh memuat kolom id/prev_hash/row_hash, sesuai definisi paket ini.
func TestPayloadExcludesHashesAndID(t *testing.T) {
	payload := entry("x.y", "z", nil).Payload()

	for _, forbidden := range []string{"id", "prev_hash", "row_hash"} {
		if _, exists := payload[forbidden]; exists {
			t.Errorf("payload tidak boleh memuat %q", forbidden)
		}
	}
	if payload["action"] != "x.y" {
		t.Errorf("payload.action = %v, ingin \"x.y\"", payload["action"])
	}
}
