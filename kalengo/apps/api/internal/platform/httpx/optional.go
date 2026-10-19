package httpx

import "encoding/json"

// Optional membedakan tiga keadaan pada body PATCH (§12.1):
//
//	field tidak dikirim        -> Set=false, Value=nil
//	field dikirim sebagai null -> Set=true,  Value=nil
//	field dikirim dengan nilai -> Set=true,  Value=&v
//
// Tanpa pembeda ini, `PATCH /me` tidak bisa mengosongkan nomor telepon:
// `*string` bernilai nil dipakai untuk "tidak diubah" sekaligus "dikosongkan".
//
// Catatan: UnmarshalJSON hanya dipanggil encoding/json bila field benar-benar
// ada di body, sehingga Set dapat dipercaya.
type Optional[T any] struct {
	Set   bool
	Value *T
}

// UnmarshalJSON menandai field sebagai hadir lalu menyimpan nilainya.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// MarshalJSON mengembalikan null bila field tidak diset atau diset null.
// Dipakai hanya bila tipe ini ikut diserialisasi kembali (mis. cache/debug).
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if o.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.Value)
}

// IsNull melaporkan bahwa field dikirim eksplisit sebagai null.
func (o Optional[T]) IsNull() bool { return o.Set && o.Value == nil }
