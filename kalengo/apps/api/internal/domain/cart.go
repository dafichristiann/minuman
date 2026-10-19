package domain

import (
	"time"

	"github.com/google/uuid"
)

// CartItem adalah satu baris keranjang beserta data produk terkini.
//
// `AddedPrice` adalah harga saat item dimasukkan; `UnitPrice` adalah harga
// server saat ini. Selisihnya memicu CART_PRICE_CHANGED (§8.6).
type CartItem struct {
	VariantID   uuid.UUID
	ProductID   uuid.UUID
	ProductSlug string
	ProductName string
	SKU         string
	VolumeML    int32
	PackSize    int32
	Quantity    int32
	AddedPrice  int64
	UnitPrice   int64
	MaxPerOrder int32
	Available   int32
	IsAvailable bool
}

// PriceChanged melaporkan apakah harga berubah sejak item ditambahkan.
func (i CartItem) PriceChanged() bool { return i.UnitPrice != i.AddedPrice }

// LineTotal menghitung subtotal baris memakai harga server (sumber kebenaran).
func (i CartItem) LineTotal() int64 { return i.UnitPrice * int64(i.Quantity) }

// Cart adalah keranjang pengguna.
type Cart struct {
	Items     []CartItem
	UpdatedAt time.Time
}

// Count menjumlahkan kuantitas seluruh item.
func (c Cart) Count() int32 {
	var n int32
	for _, it := range c.Items {
		n += it.Quantity
	}
	return n
}

// Subtotal menjumlahkan nilai seluruh item dengan harga server.
func (c Cart) Subtotal() int64 {
	var total int64
	for _, it := range c.Items {
		total += it.LineTotal()
	}
	return total
}

// CardProduct adalah produk yang disimpan di wishlist.
type CardProduct struct {
	ID        uuid.UUID
	Slug      string
	Name      string
	Flavor    *string
	PriceFrom int64
	InStock   bool
	AddedAt   time.Time
}
