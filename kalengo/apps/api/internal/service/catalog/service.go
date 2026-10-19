// Package catalog menyajikan katalog publik: daftar produk, detail, saran
// pencarian, kategori, merek, dan ulasan (§12.2).
package catalog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/repo"
	"github.com/kalengo/api/internal/valid"
)

// Batas paginasi.
const (
	DefaultLimit   = 20
	MaxLimit       = 60
	MaxSuggestion  = 10
	DefaultReviews = 10
	MaxReviews     = 50
)

// Store adalah irisan repo.Store yang dipakai layanan ini.
type Store interface {
	ListProducts(ctx context.Context, f repo.ProductFilter) ([]domain.ProductSummary, error)
	GetProductBySlug(ctx context.Context, slug string) (domain.Product, error)
	ListVariantsByProduct(ctx context.Context, productID uuid.UUID) ([]domain.Variant, error)
	ListProductImages(ctx context.Context, productID uuid.UUID) ([]domain.Image, error)
	ProductRating(ctx context.Context, productID uuid.UUID) (float64, int32, error)
	ListReviews(ctx context.Context, productID uuid.UUID, limit, offset int) ([]domain.Review, error)
	CountReviews(ctx context.Context, productID uuid.UUID) (int64, error)
	ListCategories(ctx context.Context) ([]domain.Category, error)
	ListBrands(ctx context.Context) ([]domain.Ref, error)
	SuggestProducts(ctx context.Context, prefix string, limit int) ([]domain.Ref, error)
}

// Service adalah layanan katalog.
type Service struct {
	store Store
}

// New membuat Service.
func New(store Store) *Service {
	return &Service{store: store}
}

// ListInput adalah filter mentah dari query string.
type ListInput struct {
	Query    string
	Category string
	Brand    string
	MinPrice string
	MaxPrice string
	InStock  bool
	Sort     string
	Cursor   string
	Limit    int
}

// List mengembalikan satu halaman produk beserta kursor berikutnya.
func (s *Service) List(ctx context.Context, in ListInput) (domain.ProductPage, error) {
	sort := strings.TrimSpace(in.Sort)
	if sort == "" {
		sort = domain.SortNewest
	}
	if !domain.ValidSort(sort) {
		return domain.ProductPage{}, apperr.New(apperr.CodeAPIValidation, nil).
			WithField("sort", "pilihan: newest, price_asc, price_desc, name, popular")
	}

	limit := in.Limit
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit < 1 || limit > MaxLimit {
		return domain.ProductPage{}, apperr.New(apperr.CodeAPIValidation, nil).
			WithField("limit", "1–"+strconv.Itoa(MaxLimit))
	}

	categoryID, err := optionalUUID("category", in.Category)
	if err != nil {
		return domain.ProductPage{}, err
	}
	brandID, err := optionalUUID("brand", in.Brand)
	if err != nil {
		return domain.ProductPage{}, err
	}
	minPrice, err := optionalPrice("min_price", in.MinPrice)
	if err != nil {
		return domain.ProductPage{}, err
	}
	maxPrice, err := optionalPrice("max_price", in.MaxPrice)
	if err != nil {
		return domain.ProductPage{}, err
	}
	if minPrice != nil && maxPrice != nil && *minPrice > *maxPrice {
		return domain.ProductPage{}, apperr.New(apperr.CodeAPIValidation, nil).
			WithField("min_price", "tidak boleh lebih besar dari max_price")
	}

	filter := repo.ProductFilter{
		CategoryID:  categoryID,
		BrandID:     brandID,
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		InStockOnly: in.InStock,
		Sort:        sort,
		Limit:       limit + 1, // satu baris ekstra untuk mendeteksi halaman berikutnya
	}

	query := strings.TrimSpace(in.Query)
	if query != "" {
		if len([]rune(query)) > 100 {
			return domain.ProductPage{}, apperr.New(apperr.CodeAPIValidation, nil).
				WithField("q", "maksimal 100 karakter")
		}
		filter.Query = &query
	}

	if in.Cursor != "" {
		cur, err := decodeCursor(in.Cursor)
		if err != nil {
			return domain.ProductPage{}, err
		}
		if cur.Sort != sort {
			return domain.ProductPage{}, apperr.New(apperr.CodeAPIValidation, nil).
				WithField("cursor", "kursor berasal dari urutan yang berbeda")
		}
		key, id, err := cur.apply(sort)
		if err != nil {
			return domain.ProductPage{}, err
		}
		filter.CursorKey = key
		filter.CursorID = &id
	}

	items, err := s.store.ListProducts(ctx, filter)
	if err != nil {
		return domain.ProductPage{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	page := domain.ProductPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		page.HasMore = true
		page.NextCursor = encodeCursor(sort, page.Items[limit-1])
	}
	return page, nil
}

// Detail mengembalikan produk lengkap dengan varian, galeri, dan rating.
func (s *Service) Detail(ctx context.Context, slug string) (domain.Product, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return domain.Product{}, apperr.New(apperr.CodeAPINotFound, nil)
	}

	product, err := s.store.GetProductBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return domain.Product{}, apperr.New(apperr.CodeAPINotFound, nil)
		}
		return domain.Product{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	variants, err := s.store.ListVariantsByProduct(ctx, product.ID)
	if err != nil {
		return domain.Product{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	images, err := s.store.ListProductImages(ctx, product.ID)
	if err != nil {
		return domain.Product{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	ratingAvg, ratingCount, err := s.store.ProductRating(ctx, product.ID)
	if err != nil {
		return domain.Product{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	product.Variants = variants
	product.Images = images
	product.RatingAvg = ratingAvg
	product.RatingCount = ratingCount

	// Ringkasan daftar dipakai juga di detail; isi harga & stok dari varian
	// supaya konsisten walau produk baru saja berubah.
	product.PriceFrom, product.PriceTo, product.StockAvailable = summarizeVariants(variants)
	return product, nil
}

// Suggest memberi saran pencarian (dipanggil setelah debounce 250 ms di klien).
func (s *Service) Suggest(ctx context.Context, q string, limit int) ([]domain.Ref, error) {
	prefix := strings.TrimSpace(q)
	if len([]rune(prefix)) < 2 {
		return []domain.Ref{}, nil
	}
	if len([]rune(prefix)) > 60 {
		prefix = string([]rune(prefix)[:60])
	}
	if limit <= 0 {
		limit = MaxSuggestion
	}
	if limit > MaxSuggestion {
		limit = MaxSuggestion
	}
	items, err := s.store.SuggestProducts(ctx, valid.EscapeLike(prefix), limit)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return items, nil
}

// Categories mengembalikan kategori aktif.
func (s *Service) Categories(ctx context.Context) ([]domain.Category, error) {
	items, err := s.store.ListCategories(ctx)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return items, nil
}

// Brands mengembalikan seluruh merek.
func (s *Service) Brands(ctx context.Context) ([]domain.Ref, error) {
	items, err := s.store.ListBrands(ctx)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return items, nil
}

// Reviews mengembalikan ulasan tampil dengan paginasi offset.
func (s *Service) Reviews(ctx context.Context, productID uuid.UUID, page, limit int) (domain.ReviewPage, error) {
	if limit <= 0 {
		limit = DefaultReviews
	}
	if limit > MaxReviews {
		limit = MaxReviews
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	total, err := s.store.CountReviews(ctx, productID)
	if err != nil {
		return domain.ReviewPage{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	items, err := s.store.ListReviews(ctx, productID, limit, offset)
	if err != nil {
		return domain.ReviewPage{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return domain.ReviewPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

// ---------- kursor ----------

// cursorPayload adalah bentuk internal kursor yang dikodekan base64url.
//
// Kursor TIDAK ditandatangani: isinya hanya posisi urutan, bukan data sensitif,
// dan setiap nilai divalidasi ulang sebelum dipakai (§13.1).
type cursorPayload struct {
	Sort string `json:"s"`
	Key  string `json:"k"`
	ID   string `json:"i"`
}

func encodeCursor(sort string, item domain.ProductSummary) string {
	key := ""
	switch sort {
	case domain.SortPriceAsc, domain.SortPriceDesc:
		key = strconv.FormatInt(item.PriceFrom, 10)
	case domain.SortPopular:
		key = strconv.FormatInt(item.UnitsSold, 10)
	case domain.SortName:
		key = item.Name
	default:
		key = item.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	raw, _ := json.Marshal(cursorPayload{Sort: sort, Key: key, ID: item.ID.String()})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(raw string) (cursorPayload, error) {
	invalid := apperr.New(apperr.CodeAPIValidation, nil).WithField("cursor", "kursor tidak valid")

	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cursorPayload{}, invalid
	}
	var c cursorPayload
	if err := json.Unmarshal(data, &c); err != nil {
		return cursorPayload{}, invalid
	}
	if !domain.ValidSort(c.Sort) || c.Key == "" || c.ID == "" {
		return cursorPayload{}, invalid
	}
	return c, nil
}

// apply mengubah kursor menjadi nilai bertipe yang diminta query.
func (c cursorPayload) apply(sort string) (any, uuid.UUID, error) {
	invalid := apperr.New(apperr.CodeAPIValidation, nil).WithField("cursor", "kursor tidak valid")

	id, err := uuid.Parse(c.ID)
	if err != nil {
		return nil, uuid.Nil, invalid
	}

	switch sort {
	case domain.SortPriceAsc, domain.SortPriceDesc, domain.SortPopular:
		n, err := strconv.ParseInt(c.Key, 10, 64)
		if err != nil {
			return nil, uuid.Nil, invalid
		}
		return n, id, nil
	case domain.SortName:
		if len(c.Key) > 200 {
			return nil, uuid.Nil, invalid
		}
		return c.Key, id, nil
	default:
		t, err := time.Parse(time.RFC3339Nano, c.Key)
		if err != nil {
			return nil, uuid.Nil, invalid
		}
		return t.UTC(), id, nil
	}
}

// ---------- pembantu ----------

func optionalUUID(field, raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperr.New(apperr.CodeAPIValidation, nil).WithField(field, "UUID tidak valid")
	}
	return &parsed, nil
}

func optionalPrice(field, raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return nil, apperr.New(apperr.CodeAPIValidation, nil).WithField(field, "harga harus angka ≥ 0")
	}
	return &n, nil
}

func summarizeVariants(variants []domain.Variant) (from, to int64, stock int32) {
	if len(variants) == 0 {
		return 0, 0, 0
	}
	from = variants[0].Price
	to = variants[0].Price
	for _, v := range variants {
		if v.Price < from {
			from = v.Price
		}
		if v.Price > to {
			to = v.Price
		}
		if v.Available() > 0 {
			stock += v.Available()
		}
	}
	return from, to, stock
}
