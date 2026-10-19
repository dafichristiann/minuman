// Package shop menangani keranjang, wishlist, dan langganan notifikasi restock
// (§12.3). Harga dan stok SELALU diambil dari server (§6.3).
package shop

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/repo"
)

// Batas kuantitas per baris keranjang (CHECK di skema: 1–99).
const (
	MinQuantity = 1
	MaxQuantity = 99
)

// Store adalah irisan repo.Store yang dibutuhkan.
type Store interface {
	EnsureCart(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	GetCartItems(ctx context.Context, userID uuid.UUID) ([]domain.CartItem, error)
	UpsertCartItem(ctx context.Context, cartID, variantID uuid.UUID, quantity int32, price int64) error
	MergeCartItem(ctx context.Context, cartID, variantID uuid.UUID, quantity int32, price int64, cap int32) error
	DeleteCartItem(ctx context.Context, cartID, variantID uuid.UUID) error
	ClearCart(ctx context.Context, cartID uuid.UUID) error
	GetVariantPurchaseInfo(ctx context.Context, variantID uuid.UUID) (repo.VariantPurchaseInfo, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (domain.Product, error)

	ListWishlist(ctx context.Context, userID uuid.UUID) ([]domain.CardProduct, error)
	AddWishlist(ctx context.Context, userID, productID uuid.UUID) error
	RemoveWishlist(ctx context.Context, userID, productID uuid.UUID) error
	HasWishlist(ctx context.Context, userID, productID uuid.UUID) (bool, error)

	AddStockAlert(ctx context.Context, userID, variantID uuid.UUID) error
	RemoveStockAlert(ctx context.Context, userID, variantID uuid.UUID) error
}

// Service adalah layanan keranjang & wishlist.
type Service struct {
	store Store
}

// New membuat Service.
func New(store Store) *Service { return &Service{store: store} }

// Issue adalah satu masalah pada keranjang. `Code` memakai kode §8.6 sehingga
// frontend bisa menampilkan notifikasi yang tepat tanpa menerjemahkan teks.
type Issue struct {
	VariantID   uuid.UUID
	Code        string
	ProductName string
	AddedPrice  int64
	UnitPrice   int64
	Available   int32
	Requested   int32
}

// View adalah tampilan keranjang siap kirim ke klien.
type View struct {
	Items    []domain.CartItem
	Count    int32
	Subtotal int64
	Issues   []Issue
}

// Get mengembalikan isi keranjang beserta masalah yang perlu ditampilkan.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (View, error) {
	items, err := s.store.GetCartItems(ctx, userID)
	if err != nil {
		return View{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return buildView(items), nil
}

// SetItem menetapkan kuantitas absolut satu varian.
func (s *Service) SetItem(ctx context.Context, userID, variantID uuid.UUID, quantity int32) (View, error) {
	if err := validateQuantity(quantity); err != nil {
		return View{}, err
	}

	info, err := s.store.GetVariantPurchaseInfo(ctx, variantID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return View{}, apperr.New(apperr.CodeCartItemUnavailable, nil).
				WithField("variant_id", "varian tidak ditemukan")
		}
		return View{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if !info.IsVariantActive || !info.IsProductActive {
		return View{}, apperr.New(apperr.CodeCartItemUnavailable, map[string]string{"produk": info.ProductName})
	}
	if info.Available <= 0 {
		return View{}, apperr.New(apperr.CodeStockOut, map[string]string{"produk": info.ProductName})
	}
	if limit := effectiveLimit(info); quantity > limit {
		return View{}, apperr.New(apperr.CodeCartQtyMax, map[string]string{
			"n":      strconv.Itoa(int(limit)),
			"produk": info.ProductName,
		})
	}

	cartID, err := s.store.EnsureCart(ctx, userID)
	if err != nil {
		return View{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.store.UpsertCartItem(ctx, cartID, variantID, quantity, info.Price); err != nil {
		return View{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return s.Get(ctx, userID)
}

// RemoveItem menghapus satu varian dari keranjang.
func (s *Service) RemoveItem(ctx context.Context, userID, variantID uuid.UUID) (View, error) {
	cartID, err := s.store.EnsureCart(ctx, userID)
	if err != nil {
		return View{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.store.DeleteCartItem(ctx, cartID, variantID); err != nil {
		return View{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return s.Get(ctx, userID)
}

// Clear mengosongkan keranjang.
func (s *Service) Clear(ctx context.Context, userID uuid.UUID) error {
	cartID, err := s.store.EnsureCart(ctx, userID)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.store.ClearCart(ctx, cartID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// MergeItem adalah satu baris keranjang tamu dari localStorage.
type MergeItem struct {
	VariantID uuid.UUID
	Quantity  int32
}

// MergeResult melaporkan apa yang terjadi saat penggabungan keranjang tamu.
type MergeResult struct {
	Merged   int
	Adjusted int
	Removed  int
	View     View
}

// Merge menggabungkan keranjang tamu (§6.3).
//
// Item yang tidak tersedia DILEWATI (tidak menggagalkan seluruh proses), dan
// kuantitas yang melebihi batas dipotong — bukan ditolak — karena pengguna
// tidak melihat kesalahan apa pun saat checkout tamu.
func (s *Service) Merge(ctx context.Context, userID uuid.UUID, items []MergeItem) (MergeResult, error) {
	if len(items) == 0 {
		view, err := s.Get(ctx, userID)
		if err != nil {
			return MergeResult{}, err
		}
		return MergeResult{View: view}, nil
	}
	if len(items) > 100 {
		return MergeResult{}, apperr.New(apperr.CodeAPIValidation, nil).
			WithField("items", "maksimal 100 baris")
	}

	cartID, err := s.store.EnsureCart(ctx, userID)
	if err != nil {
		return MergeResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	var result MergeResult
	for _, item := range items {
		if item.VariantID == uuid.Nil || item.Quantity < MinQuantity || item.Quantity > MaxQuantity {
			result.Removed++
			continue
		}

		info, err := s.store.GetVariantPurchaseInfo(ctx, item.VariantID)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				result.Removed++
				continue
			}
			return MergeResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
		}
		if !info.IsVariantActive || !info.IsProductActive || info.Available <= 0 {
			result.Removed++
			continue
		}

		qty := item.Quantity
		if limit := effectiveLimit(info); qty > limit {
			qty = limit
			result.Adjusted++
		}
		if err := s.store.MergeCartItem(ctx, cartID, item.VariantID, qty, info.Price, effectiveLimit(info)); err != nil {
			return MergeResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
		}
		result.Merged++
	}

	view, err := s.Get(ctx, userID)
	if err != nil {
		return MergeResult{}, err
	}
	result.View = view
	return result, nil
}

// Validate memeriksa ulang harga & stok keranjang (§12.3).
//
// Hasilnya bukan error: klien memakai daftar `issues` untuk menampilkan
// notifikasi yang relevan, sementara keranjang tetap bisa dilihat.
func (s *Service) Validate(ctx context.Context, userID uuid.UUID) (View, error) {
	return s.Get(ctx, userID)
}

func buildView(items []domain.CartItem) View {
	view := View{Items: items, Issues: make([]Issue, 0, 2)}
	for _, it := range items {
		view.Count += it.Quantity
		view.Subtotal += it.LineTotal()

		switch {
		case !it.IsAvailable:
			view.Issues = append(view.Issues, Issue{
				VariantID:   it.VariantID,
				Code:        apperr.CodeCartItemUnavailable,
				ProductName: it.ProductName,
				UnitPrice:   it.UnitPrice,
				Available:   it.Available,
				Requested:   it.Quantity,
			})
		case it.Available <= 0:
			view.Issues = append(view.Issues, Issue{
				VariantID:   it.VariantID,
				Code:        apperr.CodeStockOut,
				ProductName: it.ProductName,
				UnitPrice:   it.UnitPrice,
				Available:   0,
				Requested:   it.Quantity,
			})
		case it.PriceChanged():
			view.Issues = append(view.Issues, Issue{
				VariantID:   it.VariantID,
				Code:        apperr.CodeCartPriceChanged,
				ProductName: it.ProductName,
				AddedPrice:  it.AddedPrice,
				UnitPrice:   it.UnitPrice,
				Available:   it.Available,
				Requested:   it.Quantity,
			})
		}
	}
	return view
}

// effectiveLimit adalah batas kuantitas yang benar-benar bisa dipenuhi:
// batas per pesanan DAN stok tersedia.
func effectiveLimit(info repo.VariantPurchaseInfo) int32 {
	limit := info.MaxPerOrder
	if limit <= 0 || info.Available < limit {
		limit = info.Available
	}
	if limit < 1 {
		limit = 1
	}
	if limit > MaxQuantity {
		limit = MaxQuantity
	}
	return limit
}

func validateQuantity(q int32) error {
	if q < MinQuantity || q > MaxQuantity {
		return apperr.New(apperr.CodeAPIValidation, nil).
			WithField("quantity", "antara "+strconv.Itoa(MinQuantity)+" dan "+strconv.Itoa(MaxQuantity))
	}
	return nil
}

// ---------- wishlist ----------

// Wishlist mengembalikan produk yang disimpan pengguna.
func (s *Service) Wishlist(ctx context.Context, userID uuid.UUID) ([]domain.CardProduct, error) {
	items, err := s.store.ListWishlist(ctx, userID)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return items, nil
}

// HasWishlist melaporkan apakah produk sudah disimpan pengguna (GET /wishlist/{productId}).
func (s *Service) HasWishlist(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	saved, err := s.store.HasWishlist(ctx, userID, productID)
	if err != nil {
		return false, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return saved, nil
}

// AddWishlist menyimpan produk (idempoten).
func (s *Service) AddWishlist(ctx context.Context, userID, productID uuid.UUID) error {
	product, err := s.store.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return apperr.New(apperr.CodeAPINotFound, nil).WithField("product_id", "produk tidak ditemukan")
		}
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.store.AddWishlist(ctx, userID, product.ID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// RemoveWishlist menghapus produk dari wishlist (idempoten).
func (s *Service) RemoveWishlist(ctx context.Context, userID, productID uuid.UUID) error {
	if err := s.store.RemoveWishlist(ctx, userID, productID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// ---------- langganan restock ----------

// SubscribeStock mendaftarkan "kabari saya saat tersedia" (§8.6 STOCK_BACK).
func (s *Service) SubscribeStock(ctx context.Context, userID, variantID uuid.UUID) error {
	info, err := s.store.GetVariantPurchaseInfo(ctx, variantID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return apperr.New(apperr.CodeAPINotFound, nil).WithField("variant_id", "varian tidak ditemukan")
		}
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if info.Available > 0 {
		// Sudah tersedia: tidak perlu langganan. Bukan error — klien cukup
		// menampilkan CTA "tambah ke keranjang" seperti biasa.
		return nil
	}
	if err := s.store.AddStockAlert(ctx, userID, variantID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// UnsubscribeStock membatalkan langganan restock.
func (s *Service) UnsubscribeStock(ctx context.Context, userID, variantID uuid.UUID) error {
	if err := s.store.RemoveStockAlert(ctx, userID, variantID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}
