package catalog

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/repo"
)

// ---------------------------------------------------------------------------
// store palsu
// ---------------------------------------------------------------------------

type fakeStore struct {
	products []domain.ProductSummary

	lastFilter   repo.ProductFilter
	listCalls    int
	suggestCalls int
	lastPrefix   string

	detail    domain.Product
	detailErr error

	variants []domain.Variant
	images   []domain.Image
	rating   float64
	ratings  int32

	reviews []domain.Review
	total   int64
}

func (f *fakeStore) ListProducts(_ context.Context, filter repo.ProductFilter) ([]domain.ProductSummary, error) {
	f.listCalls++
	f.lastFilter = filter
	limit := filter.Limit
	if limit > len(f.products) {
		limit = len(f.products)
	}
	return f.products[:limit], nil
}

func (f *fakeStore) GetProductBySlug(_ context.Context, _ string) (domain.Product, error) {
	if f.detailErr != nil {
		return domain.Product{}, f.detailErr
	}
	return f.detail, nil
}

func (f *fakeStore) ListVariantsByProduct(_ context.Context, _ uuid.UUID) ([]domain.Variant, error) {
	return f.variants, nil
}

func (f *fakeStore) ListProductImages(_ context.Context, _ uuid.UUID) ([]domain.Image, error) {
	return f.images, nil
}

func (f *fakeStore) ProductRating(_ context.Context, _ uuid.UUID) (float64, int32, error) {
	return f.rating, f.ratings, nil
}

func (f *fakeStore) ListReviews(_ context.Context, _ uuid.UUID, limit, offset int) ([]domain.Review, error) {
	if offset >= len(f.reviews) {
		return nil, nil
	}
	end := offset + limit
	if end > len(f.reviews) {
		end = len(f.reviews)
	}
	return f.reviews[offset:end], nil
}

func (f *fakeStore) CountReviews(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.total, nil
}

func (f *fakeStore) ListCategories(_ context.Context) ([]domain.Category, error) {
	return []domain.Category{{ID: uuid.New(), Slug: "kopi", Name: "Kopi"}}, nil
}

func (f *fakeStore) ListBrands(_ context.Context) ([]domain.Ref, error) {
	return []domain.Ref{{ID: uuid.New(), Slug: "kalengo", Name: "Kalengo"}}, nil
}

func (f *fakeStore) SuggestProducts(_ context.Context, prefix string, _ int) ([]domain.Ref, error) {
	f.suggestCalls++
	f.lastPrefix = prefix
	return []domain.Ref{{ID: uuid.New(), Slug: "kopi-susu", Name: "Kopi Susu"}}, nil
}

func wantCode(t *testing.T, err error, code string) *apperr.Error {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, ingin kode %s", code)
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error bukan *apperr.Error: %v", err)
	}
	if appErr.Code != code {
		t.Fatalf("kode = %s, ingin %s", appErr.Code, code)
	}
	return appErr
}

func sampleProducts(n int) []domain.ProductSummary {
	base := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	out := make([]domain.ProductSummary, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.ProductSummary{
			ID:             uuid.New(),
			Slug:           "produk-" + string(rune('a'+i)),
			Name:           "Produk " + string(rune('A'+i)),
			PriceFrom:      int64(5000 + i*1000),
			PriceTo:        int64(5000 + i*1000),
			UnitsSold:      int64(100 - i),
			CreatedAt:      base.Add(time.Duration(i) * time.Minute),
			StockAvailable: 10,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// kursor
// ---------------------------------------------------------------------------

func TestCursorRoundTripAllSorts(t *testing.T) {
	item := domain.ProductSummary{
		ID:        uuid.New(),
		Name:      "Kopi Susu Gula Aren",
		PriceFrom: 7500,
		UnitsSold: 42,
		CreatedAt: time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC),
	}

	sorts := []string{
		domain.SortNewest,
		domain.SortPriceAsc,
		domain.SortPriceDesc,
		domain.SortName,
		domain.SortPopular,
	}

	for _, sort := range sorts {
		t.Run(sort, func(t *testing.T) {
			encoded := encodeCursor(sort, item)

			decoded, err := decodeCursor(encoded)
			if err != nil {
				t.Fatalf("decodeCursor() error = %v", err)
			}
			if decoded.Sort != sort {
				t.Fatalf("Sort = %q, ingin %q", decoded.Sort, sort)
			}
			if decoded.ID != item.ID.String() {
				t.Fatalf("ID = %q, ingin %q", decoded.ID, item.ID)
			}

			key, id, err := decoded.apply(sort)
			if err != nil {
				t.Fatalf("apply() error = %v", err)
			}
			if id != item.ID {
				t.Fatalf("id = %v, ingin %v", id, item.ID)
			}

			switch sort {
			case domain.SortPriceAsc, domain.SortPriceDesc:
				if got, ok := key.(int64); !ok || got != item.PriceFrom {
					t.Fatalf("key = %v, ingin %d", key, item.PriceFrom)
				}
			case domain.SortPopular:
				if got, ok := key.(int64); !ok || got != item.UnitsSold {
					t.Fatalf("key = %v, ingin %d", key, item.UnitsSold)
				}
			case domain.SortName:
				if got, ok := key.(string); !ok || got != item.Name {
					t.Fatalf("key = %v, ingin %q", key, item.Name)
				}
			default:
				got, ok := key.(time.Time)
				if !ok || !got.Equal(item.CreatedAt) {
					t.Fatalf("key = %v, ingin %v", key, item.CreatedAt)
				}
			}
		})
	}
}

// encodeCursorRaw mengodekan payload JSON apa adanya, untuk menguji validasi
// isi kursor (bukan hanya validasi base64).
func encodeCursorRaw(payload string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	cases := []string{
		"",
		"bukan-base64!!",
		"e30", // {}
		encodeCursorRaw(`{"s":"acak","k":"x","i":"y"}`),
		encodeCursorRaw(`{"s":"newest","k":"","i":"y"}`),
		encodeCursorRaw(`{"s":"newest","k":"x","i":""}`),
	}
	for _, in := range cases {
		if _, err := decodeCursor(in); !apperr.Is(err, apperr.CodeAPIValidation) {
			t.Errorf("decodeCursor(%q) error = %v, ingin API_VALIDATION", in, err)
		}
	}
}

func TestCursorApplyRejectsUnparseableKey(t *testing.T) {
	cases := []cursorPayload{
		{Sort: domain.SortPriceAsc, Key: "mahal", ID: uuid.NewString()},
		{Sort: domain.SortNewest, Key: "kemarin", ID: uuid.NewString()},
		{Sort: domain.SortNewest, Key: "2026-10-20T09:00:00Z", ID: "bukan-uuid"},
	}
	for _, c := range cases {
		if _, _, err := c.apply(c.Sort); !apperr.Is(err, apperr.CodeAPIValidation) {
			t.Errorf("apply(%+v) error = %v, ingin API_VALIDATION", c, err)
		}
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestListDefaultsAndPagination(t *testing.T) {
	store := &fakeStore{products: sampleProducts(25)}
	svc := New(store)

	page, err := svc.List(context.Background(), ListInput{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if store.lastFilter.Sort != domain.SortNewest {
		t.Errorf("Sort = %q, ingin newest (default)", store.lastFilter.Sort)
	}
	if store.lastFilter.Limit != DefaultLimit+1 {
		t.Errorf("Limit = %d, ingin %d (limit+1 untuk deteksi halaman)", store.lastFilter.Limit, DefaultLimit+1)
	}
	if len(page.Items) != DefaultLimit {
		t.Errorf("jumlah item = %d, ingin %d", len(page.Items), DefaultLimit)
	}
	if !page.HasMore {
		t.Error("HasMore = false, ingin true")
	}
	if page.NextCursor == "" {
		t.Fatal("NextCursor kosong padahal masih ada halaman")
	}

	// Kursor lanjutan harus bisa dipakai.
	if _, err := svc.List(context.Background(), ListInput{Cursor: page.NextCursor}); err != nil {
		t.Fatalf("List(halaman 2) error = %v", err)
	}
	if store.lastFilter.CursorID == nil {
		t.Error("CursorID tidak diteruskan ke query")
	}
}

func TestListLastPageHasNoCursor(t *testing.T) {
	svc := New(&fakeStore{products: sampleProducts(3)})

	page, err := svc.List(context.Background(), ListInput{Limit: 5})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.HasMore {
		t.Error("HasMore = true pada halaman terakhir")
	}
	if page.NextCursor != "" {
		t.Errorf("NextCursor = %q, ingin kosong", page.NextCursor)
	}
}

func TestListRejectsBadInput(t *testing.T) {
	store := &fakeStore{products: sampleProducts(2)}
	svc := New(store)

	tests := []struct {
		name  string
		input ListInput
		field string
	}{
		{"sort tidak dikenal", ListInput{Sort: "termurah"}, "sort"},
		{"limit nol", ListInput{Limit: -1}, "limit"},
		{"limit terlalu besar", ListInput{Limit: MaxLimit + 1}, "limit"},
		{"kategori bukan uuid", ListInput{Category: "kopi"}, "category"},
		{"merek bukan uuid", ListInput{Brand: "kalengo"}, "brand"},
		{"harga bukan angka", ListInput{MinPrice: "murah"}, "min_price"},
		{"harga negatif", ListInput{MaxPrice: "-1"}, "max_price"},
		{"rentang harga terbalik", ListInput{MinPrice: "20000", MaxPrice: "10000"}, "min_price"},
		{"query terlalu panjang", ListInput{Query: string(make([]rune, 101))}, "q"},
		{"kursor acak", ListInput{Cursor: "acak"}, "cursor"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.List(context.Background(), tc.input)
			appErr := wantCode(t, err, apperr.CodeAPIValidation)
			if appErr.Fields[tc.field] == "" {
				t.Errorf("Fields = %v, ingin penjelasan pada %s", appErr.Fields, tc.field)
			}
		})
	}

	if store.listCalls != 0 {
		t.Error("query dieksekusi walau input tidak valid")
	}
}

func TestListRejectsCursorFromOtherSort(t *testing.T) {
	store := &fakeStore{products: sampleProducts(3)}
	svc := New(store)

	cursor := encodeCursor(domain.SortPriceAsc, sampleProducts(1)[0])
	_, err := svc.List(context.Background(), ListInput{Sort: domain.SortNewest, Cursor: cursor})

	appErr := wantCode(t, err, apperr.CodeAPIValidation)
	if appErr.Fields["cursor"] == "" {
		t.Errorf("Fields = %v, ingin penjelasan pada cursor", appErr.Fields)
	}
	if store.listCalls != 0 {
		t.Error("query dieksekusi walau kursor tidak cocok")
	}
}

// ---------------------------------------------------------------------------
// Detail, Suggest, Reviews
// ---------------------------------------------------------------------------

func TestDetailSummarizesVariants(t *testing.T) {
	productID := uuid.New()
	store := &fakeStore{
		detail: domain.Product{ProductSummary: domain.ProductSummary{ID: productID, Slug: "kopi-susu"}},
		variants: []domain.Variant{
			{ID: uuid.New(), Price: 8000, StockOnHand: 5, StockReserved: 0, IsActive: true},
			{ID: uuid.New(), Price: 6000, StockOnHand: 4, StockReserved: 4, IsActive: true}, // habis
			{ID: uuid.New(), Price: 12000, StockOnHand: 3, StockReserved: 1, IsActive: true},
		},
		rating:  4.6,
		ratings: 12,
	}

	product, err := New(store).Detail(context.Background(), " kopi-susu ")
	if err != nil {
		t.Fatalf("Detail() error = %v", err)
	}

	if product.PriceFrom != 6000 || product.PriceTo != 12000 {
		t.Errorf("rentang harga = %d–%d, ingin 6000–12000", product.PriceFrom, product.PriceTo)
	}
	if product.StockAvailable != 7 {
		t.Errorf("StockAvailable = %d, ingin 7 (varian habis tidak dihitung)", product.StockAvailable)
	}
	if product.RatingAvg != 4.6 || product.RatingCount != 12 {
		t.Errorf("rating = %v/%d, ingin 4.6/12", product.RatingAvg, product.RatingCount)
	}
}

func TestDetailNotFound(t *testing.T) {
	svc := New(&fakeStore{detailErr: repo.ErrNotFound})

	_, err := svc.Detail(context.Background(), "tidak-ada")
	wantCode(t, err, apperr.CodeAPINotFound)
}

func TestSuggestNeedsTwoCharacters(t *testing.T) {
	store := &fakeStore{}
	svc := New(store)

	items, err := svc.Suggest(context.Background(), " k ", 5)
	if err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("items = %v, ingin kosong", items)
	}
	if store.suggestCalls != 0 {
		t.Error("query saran dijalankan untuk 1 karakter")
	}
}

func TestSuggestEscapesWildcards(t *testing.T) {
	store := &fakeStore{}
	svc := New(store)

	if _, err := svc.Suggest(context.Background(), "100%", 99); err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}
	if store.lastPrefix != `100\%` {
		t.Errorf("prefix = %q, ingin %q", store.lastPrefix, `100\%`)
	}
}

func TestReviewsPagination(t *testing.T) {
	reviewList := make([]domain.Review, 60)
	for i := range reviewList {
		reviewList[i] = domain.Review{ID: uuid.New(), Rating: 5}
	}
	store := &fakeStore{reviews: reviewList, total: 60}
	svc := New(store)

	// page < 1 diperlakukan sebagai halaman 1; limit dibatasi MaxReviews.
	page, err := svc.Reviews(context.Background(), uuid.New(), 0, 999)
	if err != nil {
		t.Fatalf("Reviews() error = %v", err)
	}
	if page.Limit != MaxReviews {
		t.Errorf("Limit = %d, ingin %d", page.Limit, MaxReviews)
	}
	if page.Offset != 0 {
		t.Errorf("Offset = %d, ingin 0", page.Offset)
	}
	if len(page.Items) != MaxReviews {
		t.Errorf("jumlah item = %d, ingin %d", len(page.Items), MaxReviews)
	}
	if page.Total != 60 {
		t.Errorf("Total = %d, ingin 60", page.Total)
	}

	// Halaman 3 harus mulai pada offset (3-1) × limit.
	page, err = svc.Reviews(context.Background(), uuid.New(), 3, 10)
	if err != nil {
		t.Fatalf("Reviews(halaman 3) error = %v", err)
	}
	if page.Offset != 20 {
		t.Errorf("Offset = %d, ingin 20", page.Offset)
	}
	if len(page.Items) != 10 {
		t.Errorf("jumlah item = %d, ingin 10", len(page.Items))
	}
}
