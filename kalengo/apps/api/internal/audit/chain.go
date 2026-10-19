// Package audit mengimplementasikan rantai hash audit log (§13.2):
//
//	row_hash = SHA256(prev_hash || canonical_json(row))
//
// Tujuan: perubahan atau penghapusan baris lama terdeteksi. Job harian
// memverifikasi rantai; bila rusak, backend memicu ADM_SUSPICIOUS /
// ADM_AUDIT_CHAIN_BROKEN (§8.6 G).
//
// Definisi yang dipakai di sini (supaya implementasi dan verifikasi konsisten):
//   - "row" yang di-hash adalah objek kanonik berisi: occurred_at, actor_id,
//     actor_role, ip, user_agent, request_id, action, entity, entity_id,
//     before, after. Kolom id, prev_hash, dan row_hash TIDAK ikut di-hash.
//   - Baris pertama memakai prev_hash kosong (nil).
//   - JSON kanonik: kunci diurutkan, tanpa spasi, angka memakai representasi
//     json.Number apa adanya sehingga nilai tidak berubah karena float.
package audit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Entry adalah satu baris audit log yang siap ditulis/diverifikasi.
type Entry struct {
	OccurredAt time.Time
	ActorID    *string
	ActorRole  string
	IP         string
	UserAgent  string
	RequestID  string
	Action     string
	Entity     string
	EntityID   string
	// Before/After adalah JSON apa pun (biasanya map). Data sensitif WAJIB
	// sudah disamarkan sebelum masuk sini (§13.2).
	Before any
	After  any
}

// Chained adalah Entry beserta hasil rantai hash-nya.
type Chained struct {
	Entry
	PrevHash []byte
	RowHash  []byte
}

// Canonicalize menghasilkan JSON kanonik dari nilai apa pun.
func Canonicalize(v any) ([]byte, error) {
	normalized, err := normalize(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, normalized); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Payload membentuk objek yang di-hash untuk sebuah Entry.
func (e Entry) Payload() map[string]any {
	return map[string]any{
		"occurred_at": e.OccurredAt.UTC().Format(time.RFC3339Nano),
		"actor_id":    e.ActorID,
		"actor_role":  e.ActorRole,
		"ip":          e.IP,
		"user_agent":  e.UserAgent,
		"request_id":  e.RequestID,
		"action":      e.Action,
		"entity":      e.Entity,
		"entity_id":   e.EntityID,
		"before":      e.Before,
		"after":       e.After,
	}
}

// Chain menghitung row_hash dari payload Entry dengan prev_hash tertentu.
func (e Entry) Chain(prevHash []byte) (*Chained, error) {
	canonical, err := Canonicalize(e.Payload())
	if err != nil {
		return nil, fmt.Errorf("audit: gagal mengkanonikalisasi baris: %w", err)
	}
	rowHash := HashRow(prevHash, canonical)
	return &Chained{
		Entry:    e,
		PrevHash: append([]byte(nil), prevHash...),
		RowHash:  rowHash,
	}, nil
}

// HashRow menghitung SHA256(prev_hash || canonical_row).
func HashRow(prevHash, canonicalRow []byte) []byte {
	h := sha256.New()
	h.Write(prevHash)
	h.Write(canonicalRow)
	sum := h.Sum(nil)
	return sum
}

// Verify memeriksa rantai: prev_hash tiap baris harus sama dengan row_hash
// baris sebelumnya, dan row_hash harus cocok dengan isi barisnya.
func Verify(rows []Chained) error {
	var prev []byte
	for i, row := range rows {
		if !bytes.Equal(row.PrevHash, prev) {
			return fmt.Errorf("audit: rantai putus pada baris %d: prev_hash tidak cocok", i)
		}
		canonical, err := Canonicalize(row.Entry.Payload())
		if err != nil {
			return fmt.Errorf("audit: baris %d tidak bisa dikanonikalisasi: %w", i, err)
		}
		expected := HashRow(prev, canonical)
		if !bytes.Equal(expected, row.RowHash) {
			return fmt.Errorf("audit: row_hash baris %d tidak cocok (data mungkin diubah)", i)
		}
		prev = row.RowHash
	}
	return nil
}

// VerifyHex sama seperti Verify tetapi menerima hash heksadesimal, seperti
// yang tersimpan di kolom bytea PostgreSQL.
func VerifyHex(rows []Chained) error {
	for i := range rows {
		if rows[i].PrevHash != nil {
			decoded, err := hex.DecodeString(string(rows[i].PrevHash))
			if err != nil {
				return fmt.Errorf("audit: prev_hash baris %d bukan hex: %w", i, err)
			}
			rows[i].PrevHash = decoded
		}
		if rows[i].RowHash != nil {
			decoded, err := hex.DecodeString(string(rows[i].RowHash))
			if err != nil {
				return fmt.Errorf("audit: row_hash baris %d bukan hex: %w", i, err)
			}
			rows[i].RowHash = decoded
		}
	}
	return Verify(rows)
}

// Hex mengubah hash menjadi string heksadesimal (untuk log dan ekspor CSV).
func Hex(hash []byte) string { return hex.EncodeToString(hash) }

// ---------- JSON kanonik ----------

func normalize(v any) (any, error) {
	switch t := v.(type) {
	case nil, bool, string, json.Number,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v, nil
	case json.RawMessage:
		return decodeRawJSON(t)
	case []byte:
		// Nilai jsonb dari PostgreSQL sering tiba sebagai []byte. Tanpa cabang
		// ini, json.Marshal akan mengubahnya menjadi string base64 dan hash
		// menjadi tidak stabil.
		return decodeRawJSON(t)
	case map[string]any:
		// Rekursi WAJIB: nilai di dalam map bisa bertipe apa pun (mis. *string
		// dari kolom nullable). Tanpa rekursi, nilai itu lolos ke penulis JSON
		// kanonik dalam bentuk aslinya dan gagal dikenali.
		out := make(map[string]any, len(t))
		for k, item := range t {
			n, err := normalize(item)
			if err != nil {
				return nil, err
			}
			out[k] = n
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			n, err := normalize(item)
			if err != nil {
				return nil, err
			}
			out[i] = n
		}
		return out, nil
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("audit: tipe %T tidak bisa dijadikan JSON: %w", v, err)
	}
	return decodeRawJSON(raw)
}

// decodeRawJSON membaca JSON mentah menjadi nilai Go dengan angka sebagai
// json.Number, sehingga representasi angka tidak berubah karena float.
func decodeRawJSON(raw []byte) (any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("audit: JSON mentah tidak bisa dibaca: %w", err)
	}
	return out, nil
}

func writeCanonical(buf *bytes.Buffer, v any) error {
	switch t := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if t {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		encoded, err := json.Marshal(t)
		if err != nil {
			return err
		}
		buf.Write(encoded)
	case json.Number:
		if _, err := strconv.ParseFloat(t.String(), 64); err != nil {
			return fmt.Errorf("audit: angka tidak valid %q", t.String())
		}
		buf.WriteString(t.String())
	case json.RawMessage:
		decoded, err := decodeRawJSON(t)
		if err != nil {
			return err
		}
		return writeCanonical(buf, decoded)
	case []byte:
		decoded, err := decodeRawJSON(t)
		if err != nil {
			return err
		}
		return writeCanonical(buf, decoded)
	case int:
		buf.WriteString(strconv.FormatInt(int64(t), 10))
	case int8, int16, int32, int64:
		buf.WriteString(strconv.FormatInt(toInt64(t), 10))
	case uint, uint8, uint16, uint32, uint64:
		buf.WriteString(strconv.FormatUint(toUint64(t), 10))
	case float32:
		buf.WriteString(strconv.FormatFloat(float64(t), 'g', -1, 32))
	case float64:
		buf.WriteString(strconv.FormatFloat(t, 'g', -1, 64))
	case []any:
		buf.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodedKey, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(encodedKey)
			buf.WriteByte(':')
			if err := writeCanonical(buf, t[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		// Jalur cadangan untuk nilai yang belum dinormalisasi. Seharusnya tidak
		// pernah tercapai (normalize sudah rekursif), tetapi lebih baik
		// dikonversi lewat JSON daripada membuat verifikasi rantai gagal.
		normalized, err := normalize(v)
		if err != nil {
			return err
		}
		if normalized == nil {
			buf.WriteString("null")
			return nil
		}
		return writeCanonical(buf, normalized)
	}
	return nil
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int8:
		return int64(t)
	case int16:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch t := v.(type) {
	case uint:
		return uint64(t)
	case uint8:
		return uint64(t)
	case uint16:
		return uint64(t)
	case uint32:
		return uint64(t)
	case uint64:
		return t
	default:
		return 0
	}
}
