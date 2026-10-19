package handler

import (
	nethttp "net/http"
	"time"

	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/service/catalog"
)

// Catalog menangani endpoint katalog publik (§12.2).
type Catalog struct {
	cfg *config.Config
	svc *catalog.Service
}

// NewCatalog membuat handler katalog.
func NewCatalog(cfg *config.Config, svc *catalog.Service) *Catalog {
	return &Catalog{cfg: cfg, svc: svc}
}

type refDTO struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func newRefDTO(ref *domain.Ref) *refDTO {
	if ref == nil {
		return nil
	}
	return &refDTO{ID: ref.ID.String(), Slug: ref.Slug, Name: ref.Name}
}

type productSummaryDTO struct {
	ID           string  `json:"id"`
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Flavor       *string `json:"flavor"`
	Brand        *refDTO `json:"brand"`
	Category     *refDTO `json:"category"`
	MascotSVGKey *string `json:"mascot_svg_key"`
	RequiresAge  bool    `json:"requires_age_verification"`
	PriceFrom    int64   `json:"price_from"`
	PriceTo      int64   `json:"price_to"`
	StockStatus  string  `json:"stock_status"`
	CreatedAt    string  `json:"created_at"`
}

func newProductSummaryDTO(p domain.ProductSummary) productSummaryDTO {
	return productSummaryDTO{
		ID:           p.ID.String(),
		Slug:         p.Slug,
		Name:         p.Name,
		Flavor:       p.Flavor,
		Brand:        newRefDTO(p.Brand),
		Category:     newRefDTO(p.Category),
		MascotSVGKey: p.MascotSVGKey,
		RequiresAge:  p.RequiresAgeCheck,
		PriceFrom:    p.PriceFrom,
		PriceTo:      p.PriceTo,
		StockStatus:  stockStatus(p.StockAvailable),
		CreatedAt:    p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// stockStatus hanya mengungkap "tersedia / habis", bukan angka stok.
func stockStatus(available int32) string {
	if available <= 0 {
		return "out_of_stock"
	}
	return "in_stock"
}

type variantDTO struct {
	ID             string `json:"id"`
	SKU            string `json:"sku"`
	VolumeML       int32  `json:"volume_ml"`
	PackSize       int32  `json:"pack_size"`
	Price          int64  `json:"price"`
	CompareAtPrice *int64 `json:"compare_at_price"`
	WeightGram     int32  `json:"weight_gram"`
	MaxPerOrder    int32  `json:"max_per_order"`
	StockStatus    string `json:"stock_status"`
	StockHint      string `json:"stock_hint"`
}

// newVariantDTO menerapkan §12.2: angka stok publik DIBULATKAN. Klien hanya
// menerima status dan teks, sehingga angka stok internal tidak bocor.
func newVariantDTO(v domain.Variant) variantDTO {
	status := "in_stock"
	hint := "Tersedia"

	available := v.Available()
	switch {
	case available <= 0:
		status = "out_of_stock"
		hint = "Habis"
	case available <= v.LowStockThreshold:
		status = "low_stock"
		hint = "Tersisa ≤10"
	}

	return variantDTO{
		ID:             v.ID.String(),
		SKU:            v.SKU,
		VolumeML:       v.VolumeML,
		PackSize:       v.PackSize,
		Price:          v.Price,
		CompareAtPrice: v.CompareAtPrice,
		WeightGram:     v.WeightGram,
		MaxPerOrder:    v.MaxPerOrder,
		StockStatus:    status,
		StockHint:      hint,
	}
}

type imageDTO struct {
	URL       string `json:"url"`
	Alt       string `json:"alt"`
	SortOrder int32  `json:"sort_order"`
}

type productDetailDTO struct {
	productSummaryDTO
	Description string         `json:"description"`
	Nutrition   map[string]any `json:"nutrition"`
	Images      []imageDTO     `json:"images"`
	Variants    []variantDTO   `json:"variants"`
	RatingAvg   float64        `json:"rating_avg"`
	RatingCount int32          `json:"rating_count"`
}

// List menangani GET /products.
func (h *Catalog) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	q := r.URL.Query()

	page, err := h.svc.List(r.Context(), catalog.ListInput{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Brand:    q.Get("brand"),
		MinPrice: q.Get("min_price"),
		MaxPrice: q.Get("max_price"),
		InStock:  boolQuery(r, "in_stock"),
		Sort:     q.Get("sort"),
		Cursor:   q.Get("cursor"),
		Limit:    intQuery(r, "limit", 0, 1, catalog.MaxLimit),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	items := make([]productSummaryDTO, 0, len(page.Items))
	for _, p := range page.Items {
		items = append(items, newProductSummaryDTO(p))
	}

	body := map[string]any{
		"items":    items,
		"has_more": page.HasMore,
	}
	if page.NextCursor != "" {
		body["next_cursor"] = page.NextCursor
	} else {
		body["next_cursor"] = nil
	}
	httpx.JSON(w, nethttp.StatusOK, body)
}

// Detail menangani GET /products/{slug}.
func (h *Catalog) Detail(w nethttp.ResponseWriter, r *nethttp.Request) {
	slug := chiURLParam(r, "product")
	product, err := h.svc.Detail(r.Context(), slug)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	images := make([]imageDTO, 0, len(product.Images))
	for _, img := range product.Images {
		images = append(images, imageDTO{URL: img.URL, Alt: img.Alt, SortOrder: img.SortOrder})
	}
	variants := make([]variantDTO, 0, len(product.Variants))
	for _, v := range product.Variants {
		variants = append(variants, newVariantDTO(v))
	}

	httpx.JSON(w, nethttp.StatusOK, productDetailDTO{
		productSummaryDTO: newProductSummaryDTO(product.ProductSummary),
		Description:       product.Description,
		Nutrition:         product.Nutrition,
		Images:            images,
		Variants:          variants,
		RatingAvg:         product.RatingAvg,
		RatingCount:       product.RatingCount,
	})
}

// Suggest menangani GET /products/suggest?q=.
func (h *Catalog) Suggest(w nethttp.ResponseWriter, r *nethttp.Request) {
	q := r.URL.Query().Get("q")
	items, err := h.svc.Suggest(r.Context(), q, intQuery(r, "limit", 0, 1, catalog.MaxSuggestion))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]refDTO, 0, len(items))
	for _, item := range items {
		out = append(out, refDTO{ID: item.ID.String(), Slug: item.Slug, Name: item.Name})
	}
	httpx.JSON(w, nethttp.StatusOK, map[string]any{"items": out})
}

// Categories menangani GET /categories.
func (h *Catalog) Categories(w nethttp.ResponseWriter, r *nethttp.Request) {
	items, err := h.svc.Categories(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]map[string]any, 0, len(items))
	for _, c := range items {
		item := map[string]any{
			"id":         c.ID.String(),
			"slug":       c.Slug,
			"name":       c.Name,
			"sort_order": c.SortOrder,
		}
		if c.ParentID != nil {
			item["parent_id"] = c.ParentID.String()
		} else {
			item["parent_id"] = nil
		}
		out = append(out, item)
	}
	httpx.JSON(w, nethttp.StatusOK, map[string]any{"items": out})
}

// Brands menangani GET /brands.
func (h *Catalog) Brands(w nethttp.ResponseWriter, r *nethttp.Request) {
	items, err := h.svc.Brands(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]refDTO, 0, len(items))
	for _, b := range items {
		out = append(out, refDTO{ID: b.ID.String(), Slug: b.Slug, Name: b.Name})
	}
	httpx.JSON(w, nethttp.StatusOK, map[string]any{"items": out})
}

// Reviews menangani GET /products/{id}/reviews.
func (h *Catalog) Reviews(w nethttp.ResponseWriter, r *nethttp.Request) {
	productID, ok := uuidParam(w, r, "product", "id")
	if !ok {
		return
	}

	page, err := h.svc.Reviews(r.Context(), productID,
		intQuery(r, "page", 1, 1, 10_000),
		intQuery(r, "limit", 0, 1, catalog.MaxReviews),
	)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	items := make([]map[string]any, 0, len(page.Items))
	for _, rv := range page.Items {
		items = append(items, map[string]any{
			"id":         rv.ID.String(),
			"rating":     rv.Rating,
			"body":       rv.Body,
			"author":     rv.Author,
			"created_at": rv.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	httpx.JSON(w, nethttp.StatusOK, map[string]any{
		"items":  items,
		"total":  page.Total,
		"limit":  page.Limit,
		"offset": page.Offset,
	})
}
