// Package shipping menyediakan antarmuka penyedia pengiriman (§12.4).
//
// M2 memakai implementasi palsu tanpa jaringan. Adapter Biteship ditambahkan
// pada M5 dengan kontrak yang sama.
package shipping

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

// Errors yang dikembalikan penyedia.
var (
	// ErrUnserviceable berarti tujuan tidak dapat dilayani.
	ErrUnserviceable = errors.New("shipping: tujuan tidak dilayani")
	// ErrSignature menandakan webhook pengiriman tidak dapat dipercaya.
	ErrSignature = errors.New("shipping: tanda tangan webhook tidak valid")
)

// RateInput adalah permintaan tarif pengiriman.
type RateInput struct {
	DestinationAreaID string
	PostalCode        string
	WeightGram        int32
	Subtotal          int64
	Courier           string
}

// Rate adalah satu pilihan pengiriman beserta tarifnya (rupiah bulat).
type Rate struct {
	Courier     string
	Service     string
	Description string
	Cost        int64
	ETD         string
}

// ShipmentEvent adalah webhook status pengiriman yang sudah diverifikasi.
type ShipmentEvent struct {
	EventID    string
	OrderNo    string
	Status     string
	TrackingNo string
	Courier    string
}

// Provider adalah antarmuka penyedia pengiriman.
type Provider interface {
	Name() string
	Rates(ctx context.Context, in RateInput) ([]Rate, error)
	ParseWebhook(ctx context.Context, raw []byte) (ShipmentEvent, error)
}

// Fake adalah penyedia pengiriman tiruan dengan tarif deterministik.
//
// Tarif dihitung dari berat dan tujuan: cukup realistis untuk menguji alur
// checkout, dan sepenuhnya dapat diprediksi pada uji.
type Fake struct {
	// FreeShippingMin adalah ambang gratis ongkir. 0 berarti tidak ada.
	FreeShippingMin int64
}

// Name memenuhi Provider.
func (f *Fake) Name() string { return "biteship-fake" }

// couriers adalah daftar layanan yang selalu tersedia.
var couriers = []struct {
	courier string
	service string
	desc    string
	perKg   int64
	base    int64
	etd     string
}{
	{"jne", "REG", "Reguler", 9000, 8000, "2-3 hari"},
	{"jne", "YES", "Yakin Esok Sampai", 18000, 15000, "1 hari"},
	{"sicepat", "BEST", "Best Reguler", 8500, 7500, "2-4 hari"},
}

// Rates menghitung tarif untuk setiap layanan.
//
// Tujuan tanpa area dan tanpa kode pos dianggap tidak dapat dilayani: lebih baik
// menolak di depan daripada membuat pesanan yang tidak bisa dikirim.
func (f *Fake) Rates(_ context.Context, in RateInput) ([]Rate, error) {
	if in.DestinationAreaID == "" && in.PostalCode == "" {
		return nil, ErrUnserviceable
	}
	if in.WeightGram <= 0 {
		return nil, ErrUnserviceable
	}

	// Berat dibulatkan ke atas per kilogram, minimal 1 kg.
	kg := int64((in.WeightGram + 999) / 1000)
	if kg < 1 {
		kg = 1
	}

	out := make([]Rate, 0, len(couriers))
	for _, c := range couriers {
		if in.Courier != "" && in.Courier != c.courier {
			continue
		}
		cost := c.base + c.perKg*kg
		if f.FreeShippingMin > 0 && in.Subtotal >= f.FreeShippingMin {
			cost = 0
		}
		out = append(out, Rate{
			Courier:     c.courier,
			Service:     c.service,
			Description: c.desc,
			Cost:        cost,
			ETD:         c.etd,
		})
	}
	if len(out) == 0 {
		return nil, ErrUnserviceable
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cost != out[j].Cost {
			return out[i].Cost < out[j].Cost
		}
		return out[i].Service < out[j].Service
	})
	return out, nil
}

// webhookPayload adalah bentuk payload webhook pengiriman tiruan.
type webhookPayload struct {
	OrderNo    string `json:"order_id"`
	Status     string `json:"status"`
	TrackingNo string `json:"tracking_no"`
	Courier    string `json:"courier"`
	Signature  string `json:"signature"`
}

// ParseWebhook memverifikasi tanda tangan lalu menormalkan payload.
func (f *Fake) ParseWebhook(_ context.Context, raw []byte) (ShipmentEvent, error) {
	var p webhookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return ShipmentEvent{}, ErrSignature
	}
	if p.OrderNo == "" || p.Status == "" {
		return ShipmentEvent{}, ErrSignature
	}

	want := f.Sign(p.OrderNo, p.Status, p.TrackingNo)
	if subtle.ConstantTimeCompare([]byte(want), []byte(p.Signature)) != 1 {
		return ShipmentEvent{}, ErrSignature
	}

	return ShipmentEvent{
		EventID:    p.OrderNo + ":" + p.Status,
		OrderNo:    p.OrderNo,
		Status:     p.Status,
		TrackingNo: p.TrackingNo,
		Courier:    p.Courier,
	}, nil
}

// Sign membuat tanda tangan webhook tiruan.
func (f *Fake) Sign(orderNo, status, trackingNo string) string {
	sum := sha256.Sum256([]byte(orderNo + "|" + status + "|" + trackingNo))
	return hex.EncodeToString(sum[:])
}

var _ Provider = (*Fake)(nil)
