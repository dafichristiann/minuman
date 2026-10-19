package httpx

import (
	"encoding/json"
	"testing"
)

type patchFixture struct {
	Name  Optional[string] `json:"name"`
	Phone Optional[string] `json:"phone"`
}

func TestOptionalAbsent(t *testing.T) {
	var f patchFixture
	if err := json.Unmarshal([]byte(`{}`), &f); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if f.Phone.Set {
		t.Fatal("field yang tidak dikirim dianggap Set")
	}
	if f.Phone.Value != nil {
		t.Fatal("field yang tidak dikirim punya nilai")
	}
}

func TestOptionalExplicitNull(t *testing.T) {
	var f patchFixture
	if err := json.Unmarshal([]byte(`{"phone":null}`), &f); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !f.Phone.Set {
		t.Fatal("null tidak menandai Set")
	}
	if !f.Phone.IsNull() {
		t.Fatal("IsNull() = false untuk null")
	}
}

func TestOptionalValue(t *testing.T) {
	var f patchFixture
	if err := json.Unmarshal([]byte(`{"phone":"+6281234567890"}`), &f); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !f.Phone.Set {
		t.Fatal("nilai tidak menandai Set")
	}
	if f.Phone.Value == nil || *f.Phone.Value != "+6281234567890" {
		t.Fatalf("Value = %v", f.Phone.Value)
	}
	if f.Name.Set {
		t.Fatal("name ikut terisi padahal tidak dikirim")
	}
}

func TestOptionalRejectsWrongType(t *testing.T) {
	var f patchFixture
	if err := json.Unmarshal([]byte(`{"name":123}`), &f); err == nil {
		t.Fatal("tipe salah diterima")
	}
}

func TestOptionalMarshal(t *testing.T) {
	value := "budi"
	out, err := json.Marshal(Optional[string]{Set: true, Value: &value})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(out) != `"budi"` {
		t.Fatalf("Marshal() = %s", out)
	}

	out, err = json.Marshal(Optional[string]{Set: true})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(out) != "null" {
		t.Fatalf("Marshal() = %s, ingin null", out)
	}
}
