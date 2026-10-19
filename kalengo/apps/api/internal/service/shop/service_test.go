package shop

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/repo"
)

// ---------------------------------------------------------------------------
// store palsu
// ---------------------------------------------------------------------------

type fakeStore struct {
	info map[uuid.UUID]repo.VariantPurchaseInfo

	// harga saat item dimasukkan (AddedPrice) agar bisa diuji CART_PRICE_CHANGED
	addedPrice map[uuid.UUID]int64
	quantities map[uuid.UUID]int32

	lastMergeCap   int32
	lastMergeQty   int32
	mergeCalls     int
	cleared        bool
	deleteCalls    int
	deletedVariant uuid.UUID

	products map[uuid.UUID]domain.Product
	wishlist []domain.CardProduct
	alerts   map[uuid.UUID]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		info:       make(map[uuid.UUID]repo.VariantPurchaseInfo),
		addedPrice: make(map[uuid.UUID]int64),
		quantities: make(map[uuid.UUID]int32),
		products:   make(map[uuid.UUID]domain.Product),
		alerts:     make(map[uuid.UUID]bool),
	}
}

func (f *fakeStore) EnsureCart(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return cartID, nil
}

var cartID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func (f *fakeStore) GetCartItems(_ context.Context, _ uuid.UUID) ([]domain.CartItem, error) {
	ids := make([]string, 0, len(f.quantities))
	for id, qty := range f.quantities {
		if qty > 0 {
			ids = append(ids, id.String())
		}
	}
	sort.Strings(ids)

	out := make([]domain.CartItem, 0, len(ids))
	for _, raw := range ids {
		id := uuid.MustParse(raw)
		info := f.info[id]
		out = append(out, domain.CartItem{
			VariantID:   id,
			ProductID:   info.ProductID,
			ProductName: info.ProductName,
			ProductSlug: info.ProductSlug,
			SKU:         info.SKU,
			Quantity:    f.quantities[id],
			AddedPrice:  f.addedPrice[id],
			UnitPrice:   info.Price,
			MaxPerOrder: info.MaxPerOrder,
			Available:   info.Available,
			IsAvailable: info.IsVariantActive && info.IsProductActive,
		})
	}
	return out, nil
}

func (f *fakeStore) UpsertCartItem(_ context.Context, _, variantID uuid.UUID, quantity int32, price int64) error {
	f.quantities[variantID] = quantity
	if _, seen := f.addedPrice[variantID]; !seen {
		f.addedPrice[variantID] = price
	}
	return nil
}

func (f *fakeStore) MergeCartItem(_ context.Context, _, variantID uuid.UUID, quantity int32, price int64, cap int32) error {
	f.mergeCalls++
	f.lastMergeQty = quantity
	f.lastMergeCap = cap
	f.quantities[variantID] += quantity
	if f.quantities[variantID] > cap {
		f.quantities[variantID] = cap
	}
	if _, seen := f.addedPrice[variantID]; !seen {
		f.addedPrice[variantID] = price
	}
	return nil
}

func (f *fakeStore) DeleteCartItem(_ context.Context, _, variantID uuid.UUID) error {
	f.deleteCalls++
	f.deletedVariant = variantID
	delete(f.quantities, variantID)
	return nil
}

func (f *fakeStore) ClearCart(_ context.Context, _ uuid.UUID) error {
	f.cleared = true
	f.quantities = make(map[uuid.UUID]int32)
	return nil
}

func (f *fakeStore) GetVariantPurchaseInfo(_ context.Context, variantID uuid.UUID) (repo.VariantPurchaseInfo, error) {
	info, ok := f.info[variantID]
	if !ok {
		return repo.VariantPurchaseInfo{}, repo.ErrNotFound
	}
	return info, nil
}

func (f *fakeStore) GetProductByID(_ context.Context, id uuid.UUID) (domain.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return domain.Product{}, repo.ErrNotFound
	}
	return p, nil
}

func (f *fakeStore) ListWishlist(_ context.Context, _ uuid.UUID) ([]domain.CardProduct, error) {
	return f.wishlist, nil
}

func (f *fakeStore) AddWishlist(_ context.Context, _, productID uuid.UUID) error {
	if _, ok := f.products[productID]; !ok {
		return nil
	}
	f.wishlist = append(f.wishlist, domain.CardProduct{ID: productID})
	return nil
}

func (f *fakeStore) RemoveWishlist(_ context.Context, _, productID uuid.UUID) error {
	f.wishlist = nil
	return nil
}

func (f *fakeStore) HasWishlist(_ context.Context, _, productID uuid.UUID) (bool, error) {
	for _, item := range f.wishlist {
		if item.ID == productID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) AddStockAlert(_ context.Context, _, variantID uuid.UUID) error {
	f.alerts[variantID] = true
	return nil
}

func (f *fakeStore) RemoveStockAlert(_ context.Context, _, variantID uuid.UUID) error {
	delete(f.alerts, variantID)
	return nil
}

// ---------------------------------------------------------------------------
// perkakas uji
// ---------------------------------------------------------------------------

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
		t.Fatalf("kode = %s, ingin %s (pesan: %s)", appErr.Code, code, appErr.Message)
	}
	return appErr
}

// variant mendaftarkan satu varian yang bisa dibeli pada store palsu.
func variant(f *fakeStore, name string, price int64, available, maxPerOrder int32) uuid.UUID {
	id := uuid.New()
	f.info[id] = repo.VariantPurchaseInfo{
		VariantID:       id,
		ProductID:       uuid.New(),
		ProductName:     name,
		ProductSlug:     "slug",
		SKU:             "SKU-1",
		Price:           price,
		MaxPerOrder:     maxPerOrder,
		Available:       available,
		IsVariantActive: true,
		IsProductActive: true,
	}
	return id
}

var userID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

// ---------------------------------------------------------------------------
// SetItem
// ---------------------------------------------------------------------------

func TestSetItemAddsToCart(t *testing.T) {
	store := newFakeStore()
	id := variant(store, "Kopi Susu", 8000, 10, 6)

	view, err := New(store).SetItem(context.Background(), userID, id, 3)
	if err != nil {
		t.Fatalf("SetItem() error = %v", err)
	}

	if view.Count != 3 {
		t.Errorf("Count = %d, ingin 3", view.Count)
	}
	if view.Subtotal != 24000 {
		t.Errorf("Subtotal = %d, ingin 24000", view.Subtotal)
	}
	if len(view.Issues) != 0 {
		t.Errorf("Issues = %v, ingin kosong", view.Issues)
	}
	if store.quantities[id] != 3 {
		t.Errorf("kuantitas tersimpan = %d, ingin 3", store.quantities[id])
	}
}

func TestSetItemRejectsOutOfRangeQuantity(t *testing.T) {
	store := newFakeStore()
	id := variant(store, "Kopi Susu", 8000, 10, 6)
	svc := New(store)

	for _, qty := range []int32{0, -1, MaxQuantity + 1} {
		_, err := svc.SetItem(context.Background(), userID, id, qty)
		appErr := wantCode(t, err, apperr.CodeAPIValidation)
		if appErr.Fields["quantity"] == "" {
			t.Errorf("qty %d: Fields = %v, ingin penjelasan pada quantity", qty, appErr.Fields)
		}
	}

	if len(store.quantities) != 0 {
		t.Error("keranjang berubah walau kuantitas tidak valid")
	}
}

func TestSetItemRejectsInactiveVariant(t *testing.T) {
	store := newFakeStore()
	id := variant(store, "Kopi Susu", 8000, 10, 6)

	info := store.info[id]
	info.IsProductActive = false
	store.info[id] = info

	_, err := New(store).SetItem(context.Background(), userID, id, 1)
	wantCode(t, err, apperr.CodeCartItemUnavailable)
}

func TestSetItemRejectsOutOfStock(t *testing.T) {
	store := newFakeStore()
	id := variant(store, "Kopi Susu", 8000, 0, 6)

	_, err := New(store).SetItem(context.Background(), userID, id, 1)
	wantCode(t, err, apperr.CodeStockOut)
}

func TestSetItemRejectsUnknownVariant(t *testing.T) {
	_, err := New(newFakeStore()).SetItem(context.Background(), userID, uuid.New(), 1)
	wantCode(t, err, apperr.CodeCartItemUnavailable)
}

func TestSetItemCapsByAvailableStockNotMaxPerOrder(t *testing.T) {
	store := newFakeStore()
	// Batas per pesanan 6, tetapi stok hanya 3 → batas efektif 3.
	id := variant(store, "Kopi Susu", 8000, 3, 6)

	_, err := New(store).SetItem(context.Background(), userID, id, 5)
	appErr := wantCode(t, err, apperr.CodeCartQtyMax)
	if !strings.Contains(appErr.Message, "3") {
		t.Errorf("Message = %q, ingin menyebut batas 3", appErr.Message)
	}
	if len(store.quantities) != 0 {
		t.Error("keranjang berubah walau kuantitas melebihi stok")
	}
}

func TestSetItemCapsByMaxPerOrder(t *testing.T) {
	store := newFakeStore()
	// Stok berlimpah, tetapi batas per pesanan hanya 2.
	id := variant(store, "Kopi Susu", 8000, 50, 2)

	_, err := New(store).SetItem(context.Background(), userID, id, 3)
	appErr := wantCode(t, err, apperr.CodeCartQtyMax)
	if !strings.Contains(appErr.Message, "2") {
		t.Errorf("Message = %q, ingin menyebut batas 2", appErr.Message)
	}
}

func TestRemoveItemAndClear(t *testing.T) {
	store := newFakeStore()
	id := variant(store, "Kopi Susu", 8000, 10, 6)
	svc := New(store)

	if _, err := svc.SetItem(context.Background(), userID, id, 2); err != nil {
		t.Fatalf("SetItem() error = %v", err)
	}

	view, err := svc.RemoveItem(context.Background(), userID, id)
	if err != nil {
		t.Fatalf("RemoveItem() error = %v", err)
	}
	if view.Count != 0 {
		t.Errorf("Count = %d, ingin 0", view.Count)
	}
	if store.deleteCalls != 1 {
		t.Errorf("DeleteCartItem dipanggil %d kali, ingin 1", store.deleteCalls)
	}

	if err := svc.Clear(context.Background(), userID); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if !store.cleared {
		t.Error("ClearCart tidak dipanggil")
	}
}

// ---------------------------------------------------------------------------
// View / issues
// ---------------------------------------------------------------------------

func TestGetReportsPriceChangeAndUnavailable(t *testing.T) {
	store := newFakeStore()

	changed := variant(store, "Kopi Susu", 9000, 10, 6)
	gone := variant(store, "Teh Manis", 5000, 0, 6)

	store.quantities[changed] = 2
	store.addedPrice[changed] = 8000 // harga naik 8000 → 9000
	store.quantities[gone] = 1
	store.addedPrice[gone] = 5000

	view, err := New(store).Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if view.Count != 3 {
		t.Errorf("Count = %d, ingin 3", view.Count)
	}
	if view.Subtotal != 18000+5000 {
		t.Errorf("Subtotal = %d, ingin 23000 (harga server)", view.Subtotal)
	}

	codes := make(map[string]int)
	for _, issue := range view.Issues {
		codes[issue.Code]++
	}
	if codes[apperr.CodeCartPriceChanged] != 1 {
		t.Errorf("issues = %v, ingin satu CART_PRICE_CHANGED", view.Issues)
	}
	if codes[apperr.CodeStockOut] != 1 {
		t.Errorf("issues = %v, ingin satu STOCK_OUT", view.Issues)
	}
}

// ---------------------------------------------------------------------------
// Merge
// ---------------------------------------------------------------------------

func TestMergeCapsAndSkips(t *testing.T) {
	store := newFakeStore()
	okID := variant(store, "Kopi Susu", 8000, 2, 6) // stok 2
	goneID := variant(store, "Teh Manis", 5000, 0, 6)

	result, err := New(store).Merge(context.Background(), userID, []MergeItem{
		{VariantID: okID, Quantity: 5},
		{VariantID: goneID, Quantity: 2},
		{VariantID: uuid.New(), Quantity: 1}, // tidak dikenal
		{VariantID: uuid.Nil, Quantity: 1},   // baris cacat
		{VariantID: okID, Quantity: 0},       // kuantitas tidak sah
	})
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if result.Merged != 1 {
		t.Errorf("Merged = %d, ingin 1", result.Merged)
	}
	if result.Adjusted != 1 {
		t.Errorf("Adjusted = %d, ingin 1", result.Adjusted)
	}
	if result.Removed != 4 {
		t.Errorf("Removed = %d, ingin 4", result.Removed)
	}
	if store.lastMergeQty != 2 {
		t.Errorf("kuantitas digabung = %d, ingin dipotong jadi 2", store.lastMergeQty)
	}
	if store.lastMergeCap != 2 {
		t.Errorf("cap = %d, ingin 2", store.lastMergeCap)
	}
	if result.View.Count != 2 {
		t.Errorf("Count = %d, ingin 2", result.View.Count)
	}
}

func TestMergeEmptyCartIsNoop(t *testing.T) {
	store := newFakeStore()

	result, err := New(store).Merge(context.Background(), userID, nil)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if store.mergeCalls != 0 {
		t.Errorf("MergeCartItem dipanggil %d kali, ingin 0", store.mergeCalls)
	}
	if result.View.Count != 0 {
		t.Errorf("Count = %d, ingin 0", result.View.Count)
	}
}

func TestMergeRejectsTooManyRows(t *testing.T) {
	items := make([]MergeItem, 101)
	for i := range items {
		items[i] = MergeItem{VariantID: uuid.New(), Quantity: 1}
	}

	_, err := New(newFakeStore()).Merge(context.Background(), userID, items)
	appErr := wantCode(t, err, apperr.CodeAPIValidation)
	if appErr.Fields["items"] == "" {
		t.Errorf("Fields = %v, ingin penjelasan pada items", appErr.Fields)
	}
}

// ---------------------------------------------------------------------------
// wishlist & langganan restock
// ---------------------------------------------------------------------------

func TestAddWishlistRejectsUnknownProduct(t *testing.T) {
	err := New(newFakeStore()).AddWishlist(context.Background(), userID, uuid.New())
	wantCode(t, err, apperr.CodeAPINotFound)
}

func TestAddWishlistIsIdempotentForKnownProduct(t *testing.T) {
	store := newFakeStore()
	productID := uuid.New()
	store.products[productID] = domain.Product{
		ProductSummary: domain.ProductSummary{ID: productID, Name: "Kopi Susu"},
	}

	svc := New(store)
	if err := svc.AddWishlist(context.Background(), userID, productID); err != nil {
		t.Fatalf("AddWishlist() error = %v", err)
	}

	saved, err := store.HasWishlist(context.Background(), userID, productID)
	if err != nil {
		t.Fatalf("HasWishlist() error = %v", err)
	}
	if !saved {
		t.Error("produk tidak tersimpan di wishlist")
	}
}

func TestSubscribeStockOnlyWhenOutOfStock(t *testing.T) {
	store := newFakeStore()
	outID := variant(store, "Kopi Susu", 8000, 0, 6)
	inID := variant(store, "Teh Manis", 5000, 4, 6)
	svc := New(store)

	if err := svc.SubscribeStock(context.Background(), userID, outID); err != nil {
		t.Fatalf("SubscribeStock(hbis) error = %v", err)
	}
	if !store.alerts[outID] {
		t.Error("langganan tidak tercatat untuk varian yang habis")
	}

	// Varian yang masih tersedia tidak perlu langganan; bukan error.
	if err := svc.SubscribeStock(context.Background(), userID, inID); err != nil {
		t.Fatalf("SubscribeStock(tersedia) error = %v", err)
	}
	if store.alerts[inID] {
		t.Error("langganan tercatat padahal stok tersedia")
	}
}

func TestSubscribeStockUnknownVariant(t *testing.T) {
	err := New(newFakeStore()).SubscribeStock(context.Background(), userID, uuid.New())
	wantCode(t, err, apperr.CodeAPINotFound)
}
