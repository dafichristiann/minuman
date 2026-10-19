package handler

import (
	nethttp "net/http"
	"time"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/http/middleware"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/service/shop"
)

// Cart menangani keranjang, wishlist, dan langganan restock (§12.3).
type Cart struct {
	cfg *config.Config
	svc *shop.Service
}

// NewCart membuat handler keranjang.
func NewCart(cfg *config.Config, svc *shop.Service) *Cart {
	return &Cart{cfg: cfg, svc: svc}
}

type cartItemDTO struct {
	VariantID    string `json:"variant_id"`
	ProductID    string `json:"product_id"`
	ProductSlug  string `json:"product_slug"`
	ProductName  string `json:"product_name"`
	SKU          string `json:"sku"`
	VolumeML     int32  `json:"volume_ml"`
	PackSize     int32  `json:"pack_size"`
	Quantity     int32  `json:"quantity"`
	AddedPrice   int64  `json:"added_price"`
	UnitPrice    int64  `json:"unit_price"`
	LineTotal    int64  `json:"line_total"`
	PriceChanged bool   `json:"price_changed"`
	Available    int32  `json:"available"`
	IsAvailable  bool   `json:"is_available"`
	MaxPerOrder  int32  `json:"max_per_order"`
}

type cartIssueDTO struct {
	VariantID   string `json:"variant_id"`
	Code        string `json:"code"`
	ProductName string `json:"product_name"`
	AddedPrice  int64  `json:"added_price,omitempty"`
	UnitPrice   int64  `json:"unit_price,omitempty"`
	Available   int32  `json:"available"`
	Requested   int32  `json:"requested"`
}

type cartResponse struct {
	Items    []cartItemDTO  `json:"items"`
	Count    int32          `json:"count"`
	Subtotal int64          `json:"subtotal"`
	Currency string         `json:"currency"`
	Issues   []cartIssueDTO `json:"issues"`
}

func newCartResponse(view shop.View) cartResponse {
	items := make([]cartItemDTO, 0, len(view.Items))
	for _, it := range view.Items {
		items = append(items, cartItemDTO{
			VariantID:    it.VariantID.String(),
			ProductID:    it.ProductID.String(),
			ProductSlug:  it.ProductSlug,
			ProductName:  it.ProductName,
			SKU:          it.SKU,
			VolumeML:     it.VolumeML,
			PackSize:     it.PackSize,
			Quantity:     it.Quantity,
			AddedPrice:   it.AddedPrice,
			UnitPrice:    it.UnitPrice,
			LineTotal:    it.LineTotal(),
			PriceChanged: it.PriceChanged(),
			Available:    it.Available,
			IsAvailable:  it.IsAvailable,
			MaxPerOrder:  it.MaxPerOrder,
		})
	}

	issues := make([]cartIssueDTO, 0, len(view.Issues))
	for _, is := range view.Issues {
		issues = append(issues, cartIssueDTO{
			VariantID:   is.VariantID.String(),
			Code:        is.Code,
			ProductName: is.ProductName,
			AddedPrice:  is.AddedPrice,
			UnitPrice:   is.UnitPrice,
			Available:   is.Available,
			Requested:   is.Requested,
		})
	}

	return cartResponse{
		Items:    items,
		Count:    view.Count,
		Subtotal: view.Subtotal,
		Currency: "IDR",
		Issues:   issues,
	}
}

// Get menangani GET /cart.
func (h *Cart) Get(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	view, err := h.svc.Get(r.Context(), user.ID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newCartResponse(view))
}

// Clear menangani DELETE /cart.
//
// Membalas 204: klien yang meminta keranjang kosong sudah tahu hasilnya, dan
// §8.6 meminta toast `CART_CLEARED` dibuat di sisi klien (§12.3).
func (h *Cart) Clear(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	if err := h.svc.Clear(r.Context(), user.ID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

type setCartItemRequest struct {
	Quantity int32 `json:"quantity"`
}

// SetItem menangani PUT /cart/items/{variantId} (kuantitas absolut).
func (h *Cart) SetItem(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	variantID, ok := uuidParam(w, r, "variantId", "variant_id")
	if !ok {
		return
	}

	var req setCartItemRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	view, err := h.svc.SetItem(r.Context(), user.ID, variantID, req.Quantity)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newCartResponse(view))
}

// DeleteItem menangani DELETE /cart/items/{variantId}.
func (h *Cart) DeleteItem(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	variantID, ok := uuidParam(w, r, "variantId", "variant_id")
	if !ok {
		return
	}

	view, err := h.svc.RemoveItem(r.Context(), user.ID, variantID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newCartResponse(view))
}

type mergeItemRequest struct {
	VariantID string `json:"variant_id"`
	Quantity  int32  `json:"quantity"`
}

type mergeRequest struct {
	Items []mergeItemRequest `json:"items"`
}

type mergeSummary struct {
	Merged   int `json:"merged"`
	Adjusted int `json:"adjusted"`
	Removed  int `json:"removed"`
}

type mergeResponse struct {
	cartResponse
	Merge mergeSummary `json:"merge"`
}

// Merge menangani POST /cart/merge: menggabungkan keranjang tamu dari
// localStorage setelah login (§6.3).
//
// Responsnya adalah keranjang server ditambah ringkasan `merge`, sehingga klien
// bisa langsung memakai `items`/`count`/`subtotal` lalu menampilkan notifikasi
// "N item digabungkan" tanpa permintaan tambahan.
func (h *Cart) Merge(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}

	var req mergeRequest
	if err := httpx.DecodeJSON(w, r, &req, h.cfg.HTTP.MaxBodyBytes); err != nil {
		httpx.Error(w, r, err)
		return
	}

	items := make([]shop.MergeItem, 0, len(req.Items))
	for _, item := range req.Items {
		parsed, err := uuid.Parse(item.VariantID)
		if err != nil {
			// Baris cacat di keranjang tamu tidak boleh menggagalkan login:
			// klien tetap mendapat keranjang server yang valid.
			continue
		}
		items = append(items, shop.MergeItem{VariantID: parsed, Quantity: item.Quantity})
	}

	result, err := h.svc.Merge(r.Context(), user.ID, items)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, mergeResponse{
		cartResponse: newCartResponse(result.View),
		Merge: mergeSummary{
			Merged:   result.Merged,
			Adjusted: result.Adjusted,
			Removed:  result.Removed,
		},
	})
}

// Validate menangani POST /cart/validate (cek harga & stok ulang, §12.3).
func (h *Cart) Validate(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	view, err := h.svc.Validate(r.Context(), user.ID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, newCartResponse(view))
}

// ---------- wishlist ----------

type wishlistItemDTO struct {
	ProductID string  `json:"product_id"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Flavor    *string `json:"flavor"`
	PriceFrom int64   `json:"price_from"`
	InStock   bool    `json:"in_stock"`
	AddedAt   string  `json:"added_at"`
}

func newWishlistItemDTO(p domain.CardProduct) wishlistItemDTO {
	return wishlistItemDTO{
		ProductID: p.ID.String(),
		Slug:      p.Slug,
		Name:      p.Name,
		Flavor:    p.Flavor,
		PriceFrom: p.PriceFrom,
		InStock:   p.InStock,
		AddedAt:   p.AddedAt.UTC().Format(time.RFC3339),
	}
}

// ListWishlist menangani GET /wishlist.
func (h *Cart) ListWishlist(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeWishLoginReq, nil))
		return
	}
	items, err := h.svc.Wishlist(r.Context(), user.ID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]wishlistItemDTO, 0, len(items))
	for _, item := range items {
		out = append(out, newWishlistItemDTO(item))
	}
	httpx.JSON(w, nethttp.StatusOK, out)
}

// GetWishlist menangani GET /wishlist/{productId}: apakah produk sudah disimpan.
func (h *Cart) GetWishlist(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeWishLoginReq, nil))
		return
	}
	productID, ok := uuidParam(w, r, "productId", "product_id")
	if !ok {
		return
	}
	saved, err := h.svc.HasWishlist(r.Context(), user.ID, productID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, nethttp.StatusOK, map[string]any{
		"product_id": productID.String(),
		"saved":      saved,
	})
}

// AddWishlist menangani PUT /wishlist/{productId} (idempoten, §12.3).
//
// Membalas 204: toast `WISH_ADDED` dibuat klien (§8.6).
func (h *Cart) AddWishlist(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeWishLoginReq, nil))
		return
	}
	productID, ok := uuidParam(w, r, "productId", "product_id")
	if !ok {
		return
	}
	if err := h.svc.AddWishlist(r.Context(), user.ID, productID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// RemoveWishlist menangani DELETE /wishlist/{productId} (idempoten, §12.3).
func (h *Cart) RemoveWishlist(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeWishLoginReq, nil))
		return
	}
	productID, ok := uuidParam(w, r, "productId", "product_id")
	if !ok {
		return
	}
	if err := h.svc.RemoveWishlist(r.Context(), user.ID, productID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ---------- langganan restock ----------

// SubscribeStock menangani POST /stock-alerts/{variantId} (§12.3).
//
// Membalas 204 walau varian sudah tersedia: klien menampilkan CTA "tambah ke
// keranjang", bukan pesan sukses langganan (`STOCK_NOTIFY_ON`).
func (h *Cart) SubscribeStock(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	variantID, ok := uuidParam(w, r, "variantId", "variant_id")
	if !ok {
		return
	}
	if err := h.svc.SubscribeStock(r.Context(), user.ID, variantID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// UnsubscribeStock menangani DELETE /stock-alerts/{variantId}.
func (h *Cart) UnsubscribeStock(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.New(apperr.CodeAuthSessionExpired, nil))
		return
	}
	variantID, ok := uuidParam(w, r, "variantId", "variant_id")
	if !ok {
		return
	}
	if err := h.svc.UnsubscribeStock(r.Context(), user.ID, variantID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}
