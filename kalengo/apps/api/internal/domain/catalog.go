package domain

import (
	"time"

	"github.com/google/uuid"
)

// Ref adalah referensi ringan untuk brand/kategori.
type Ref struct {
	ID   uuid.UUID
	Slug string
	Name string
}

// Image adalah satu gambar galeri produk.
type Image struct {
	URL       string
	Alt       string
	SortOrder int32
}

// Variant adalah varian yang bisa dibeli (ukuran kaleng × isi paket).
type Variant struct {
	ID                uuid.UUID
	SKU               string
	VolumeML          int32
	PackSize          int32
	Price             int64
	CompareAtPrice    *int64
	WeightGram        int32
	StockOnHand       int32
	StockReserved     int32
	LowStockThreshold int32
	MaxPerOrder       int32
	IsActive          bool
}

// Available adalah stok yang benar-benar bisa dijual (bukan stok fisik).
func (v Variant) Available() int32 { return v.StockOnHand - v.StockReserved }

// ProductSummary adalah bentuk baris daftar produk (§12.2 GET /products).
type ProductSummary struct {
	ID               uuid.UUID
	Slug             string
	Name             string
	Flavor           *string
	Brand            *Ref
	Category         *Ref
	MascotSVGKey     *string
	RequiresAgeCheck bool
	PriceFrom        int64
	PriceTo          int64
	StockAvailable   int32
	UnitsSold        int64
	CreatedAt        time.Time
}

// Product adalah detail lengkap (§12.2 GET /products/{slug}).
type Product struct {
	ProductSummary
	Description string
	Nutrition   map[string]any
	Images      []Image
	Variants    []Variant
	RatingAvg   float64
	RatingCount int32
}

// Category adalah node kategori (boleh bertingkat lewat ParentID).
type Category struct {
	ID        uuid.UUID
	Slug      string
	Name      string
	ParentID  *uuid.UUID
	SortOrder int32
}

// Review adalah ulasan yang sudah tayang.
type Review struct {
	ID        uuid.UUID
	Rating    int16
	Body      *string
	Author    string
	CreatedAt time.Time
}

// ReviewPage adalah hasil paginasi ulasan.
type ReviewPage struct {
	Items  []Review
	Total  int64
	Limit  int
	Offset int
}

// ProductPage adalah hasil paginasi kursor.
type ProductPage struct {
	Items      []ProductSummary
	NextCursor string
	HasMore    bool
}

// Sort mode daftar produk (§6.1).
const (
	SortNewest    = "newest"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortName      = "name"
	SortPopular   = "popular"
)

// ValidSort melaporkan apakah nilai sort dikenal.
func ValidSort(s string) bool {
	switch s {
	case SortNewest, SortPriceAsc, SortPriceDesc, SortName, SortPopular:
		return true
	default:
		return false
	}
}
