package repo

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kalengo/api/internal/domain"
)

// productListCTE adalah kerangka daftar produk (§6.1).
//
// Bentuk kueri sengaja dipisah dari bagian ORDER BY / cursor. Dua penanda
// /*CURSOR*/ dan /*ORDER*/ digantikan fragmen STATIS dari daftar putih di
// productListOrder — tidak pernah dari input pengguna (§13.1).
//
// Parameter:
//
//	$1 category_id  $2 brand_id   $3 q          $4 min_price  $5 max_price
//	$6 in_stock     $7 limit      $8 cursor_key $9 cursor_id
const productListCTE = `
WITH agg AS (
    SELECT p.id, p.slug, p.name, p.flavor, p.mascot_svg_key,
           p.requires_age_verification, p.created_at,
           b.id::text AS brand_id, b.slug AS brand_slug, b.name AS brand_name,
           c.id::text AS category_id, c.slug AS category_slug, c.name AS category_name,
           MIN(v.price) AS price_from,
           MAX(v.price) AS price_to,
           COALESCE(SUM(v.stock_on_hand - v.stock_reserved), 0)::int AS stock_available,
           (SELECT COALESCE(SUM(oi.quantity), 0)::bigint
              FROM order_items oi
              JOIN orders o ON o.id = oi.order_id
              JOIN product_variants pv ON pv.id = oi.variant_id
             WHERE pv.product_id = p.id
               AND o.status IN ('paid','processing','shipped','delivered','completed')
           ) AS units_sold
      FROM products p
      LEFT JOIN product_variants v ON v.product_id = p.id AND v.is_active
      LEFT JOIN brands b     ON b.id = p.brand_id
      LEFT JOIN categories c ON c.id = p.category_id
     WHERE p.deleted_at IS NULL
       AND p.is_active
       AND ($1::uuid IS NULL OR p.category_id = $1)
       AND ($2::uuid IS NULL OR p.brand_id = $2)
       AND (
           $3::text IS NULL
           OR p.name ILIKE '%' || $3 || '%'
           OR p.flavor ILIKE '%' || $3 || '%'
           OR p.search @@ websearch_to_tsquery('simple', $3)
       )
     GROUP BY p.id, b.id, c.id
    HAVING MIN(v.price) IS NOT NULL
)
SELECT id, slug, name, flavor, mascot_svg_key, requires_age_verification, created_at,
       brand_id, brand_slug, brand_name,
       category_id, category_slug, category_name,
       price_from, price_to, stock_available, units_sold
  FROM agg
 WHERE ($4::bigint IS NULL OR price_from >= $4)
   AND ($5::bigint IS NULL OR price_from <= $5)
   AND ($6::boolean IS NOT TRUE OR stock_available > 0)
   /*CURSOR*/
 ORDER BY /*ORDER*/
 LIMIT $7`

// productListOrder memilih fragmen keyset + urutan untuk tiap mode sort.
// Nilai `sort` sudah divalidasi domain.ValidSort sebelum mencapai sini.
func productListOrder(sort string) (cursor string, order string) {
	switch sort {
	case domain.SortPriceAsc:
		return "AND ($8::bigint IS NULL OR (price_from, id) > ($8::bigint, $9::uuid))",
			"price_from ASC, id ASC"
	case domain.SortPriceDesc:
		return "AND ($8::bigint IS NULL OR (price_from, id) < ($8::bigint, $9::uuid))",
			"price_from DESC, id DESC"
	case domain.SortName:
		return "AND ($8::text IS NULL OR (name, id) > ($8::text, $9::uuid))",
			"name ASC, id ASC"
	case domain.SortPopular:
		return "AND ($8::bigint IS NULL OR (units_sold, id) < ($8::bigint, $9::uuid))",
			"units_sold DESC, id DESC"
	default: // domain.SortNewest
		return "AND ($8::timestamptz IS NULL OR (created_at, id) < ($8::timestamptz, $9::uuid))",
			"created_at DESC, id DESC"
	}
}

// ProductFilter adalah filter daftar produk (§6.1) yang sudah divalidasi.
type ProductFilter struct {
	Query       *string
	CategoryID  *uuid.UUID
	BrandID     *uuid.UUID
	MinPrice    *int64
	MaxPrice    *int64
	InStockOnly bool
	Sort        string
	Limit       int
	CursorKey   any
	CursorID    *uuid.UUID
}

// ListProducts mengambil satu halaman produk (keyset pagination).
func (s *Store) ListProducts(ctx context.Context, f ProductFilter) ([]domain.ProductSummary, error) {
	cursorFrag, orderFrag := productListOrder(f.Sort)
	query := strings.Replace(productListCTE, "/*CURSOR*/", cursorFrag, 1)
	query = strings.Replace(query, "/*ORDER*/", orderFrag, 1)

	rows, err := s.db.Query(ctx, query,
		f.CategoryID, f.BrandID, f.Query, f.MinPrice, f.MaxPrice,
		f.InStockOnly, f.Limit, f.CursorKey, f.CursorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ProductSummary, 0, f.Limit)
	for rows.Next() {
		item, err := scanProductSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanProductSummary(row pgx.Row) (domain.ProductSummary, error) {
	var (
		p                                      domain.ProductSummary
		brandID, brandSlug, brandName          *string
		categoryID, categorySlug, categoryName *string
	)
	err := row.Scan(
		&p.ID, &p.Slug, &p.Name, &p.Flavor, &p.MascotSVGKey, &p.RequiresAgeCheck, &p.CreatedAt,
		&brandID, &brandSlug, &brandName,
		&categoryID, &categorySlug, &categoryName,
		&p.PriceFrom, &p.PriceTo, &p.StockAvailable, &p.UnitsSold,
	)
	if err != nil {
		return domain.ProductSummary{}, mapErr(err)
	}
	p.Brand = buildRef(brandID, brandSlug, brandName)
	p.Category = buildRef(categoryID, categorySlug, categoryName)
	return p, nil
}

func buildRef(id, slug, name *string) *domain.Ref {
	if id == nil || slug == nil || name == nil {
		return nil
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return nil
	}
	return &domain.Ref{ID: parsed, Slug: *slug, Name: *name}
}

// GetProductBySlug mengambil produk aktif beserta brand & kategorinya.
func (s *Store) GetProductBySlug(ctx context.Context, slug string) (domain.Product, error) {
	var (
		p                                      domain.Product
		nutrition                              []byte
		brandID, brandSlug, brandName          *string
		categoryID, categorySlug, categoryName *string
	)
	err := s.db.QueryRow(ctx, `
		SELECT p.id, p.slug, p.name, p.flavor, p.mascot_svg_key,
		       p.requires_age_verification, p.created_at,
		       p.description, p.nutrition,
		       b.id::text, b.slug, b.name,
		       c.id::text, c.slug, c.name
		FROM products p
		LEFT JOIN brands b     ON b.id = p.brand_id
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE p.slug = $1 AND p.deleted_at IS NULL AND p.is_active`, slug,
	).Scan(
		&p.ID, &p.Slug, &p.Name, &p.Flavor, &p.MascotSVGKey, &p.RequiresAgeCheck, &p.CreatedAt,
		&p.Description, &nutrition,
		&brandID, &brandSlug, &brandName,
		&categoryID, &categorySlug, &categoryName,
	)
	if err != nil {
		return domain.Product{}, mapErr(err)
	}
	p.Brand = buildRef(brandID, brandSlug, brandName)
	p.Category = buildRef(categoryID, categorySlug, categoryName)
	p.Nutrition = decodeJSONMap(nutrition)
	return p, nil
}

// GetProductByID adalah varian untuk admin & keranjang.
func (s *Store) GetProductByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	var (
		p         domain.Product
		nutrition []byte
	)
	err := s.db.QueryRow(ctx, `
		SELECT p.id, p.slug, p.name, p.flavor, p.mascot_svg_key,
		       p.requires_age_verification, p.created_at,
		       p.description, p.nutrition
		FROM products p
		WHERE p.id = $1 AND p.deleted_at IS NULL`, id,
	).Scan(
		&p.ID, &p.Slug, &p.Name, &p.Flavor, &p.MascotSVGKey, &p.RequiresAgeCheck, &p.CreatedAt,
		&p.Description, &nutrition,
	)
	if err != nil {
		return domain.Product{}, mapErr(err)
	}
	p.Nutrition = decodeJSONMap(nutrition)
	return p, nil
}

// ListVariantsByProduct mengambil varian aktif sebuah produk.
func (s *Store) ListVariantsByProduct(ctx context.Context, productID uuid.UUID) ([]domain.Variant, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, sku, volume_ml, pack_size, price, compare_at_price, weight_gram,
		       stock_on_hand, stock_reserved, low_stock_threshold, max_per_order, is_active
		FROM product_variants
		WHERE product_id = $1 AND is_active
		ORDER BY volume_ml, pack_size`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Variant, 0, 8)
	for rows.Next() {
		var v domain.Variant
		if err := rows.Scan(
			&v.ID, &v.SKU, &v.VolumeML, &v.PackSize, &v.Price, &v.CompareAtPrice, &v.WeightGram,
			&v.StockOnHand, &v.StockReserved, &v.LowStockThreshold, &v.MaxPerOrder, &v.IsActive,
		); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListProductImages mengambil galeri produk berurutan.
func (s *Store) ListProductImages(ctx context.Context, productID uuid.UUID) ([]domain.Image, error) {
	rows, err := s.db.Query(ctx, `
		SELECT url, alt, sort_order
		FROM product_images
		WHERE product_id = $1
		ORDER BY sort_order, created_at`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Image, 0, 4)
	for rows.Next() {
		var img domain.Image
		if err := rows.Scan(&img.URL, &img.Alt, &img.SortOrder); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

// ProductRating menghitung rata-rata & jumlah ulasan tampil.
func (s *Store) ProductRating(ctx context.Context, productID uuid.UUID) (float64, int32, error) {
	var (
		avg   float64
		count int32
	)
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(rating), 0)::float8, COUNT(*)::int
		FROM reviews
		WHERE product_id = $1 AND is_visible`, productID).Scan(&avg, &count)
	return avg, count, err
}

// ListReviews mengambil ulasan tampil dengan paginasi offset.
func (s *Store) ListReviews(ctx context.Context, productID uuid.UUID, limit, offset int) ([]domain.Review, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.id, r.rating, r.body, r.created_at, u.full_name
		FROM reviews r
		JOIN users u ON u.id = r.user_id
		WHERE r.product_id = $1 AND r.is_visible
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $2 OFFSET $3`, productID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Review, 0, limit)
	for rows.Next() {
		var r domain.Review
		var fullName string
		if err := rows.Scan(&r.ID, &r.Rating, &r.Body, &r.CreatedAt, &fullName); err != nil {
			return nil, mapErr(err)
		}
		r.Author = maskName(fullName)
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountReviews menghitung ulasan tampil (untuk metadata paginasi).
func (s *Store) CountReviews(ctx context.Context, productID uuid.UUID) (int64, error) {
	var n int64
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM reviews WHERE product_id = $1 AND is_visible`, productID).Scan(&n)
	return n, err
}

// ListCategories mengembalikan kategori aktif terurut.
func (s *Store) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, slug, name, parent_id, sort_order
		FROM categories
		WHERE is_active
		ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Category, 0, 16)
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.ParentID, &c.SortOrder); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListBrands mengembalikan seluruh merek terurut.
func (s *Store) ListBrands(ctx context.Context) ([]domain.Ref, error) {
	rows, err := s.db.Query(ctx, `SELECT id, slug, name FROM brands ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Ref, 0, 16)
	for rows.Next() {
		var r domain.Ref
		if err := rows.Scan(&r.ID, &r.Slug, &r.Name); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SuggestProducts memberi saran pencarian berbasis awalan nama (§12.2).
// `prefix` sudah di-escape service agar wildcard LIKE tidak bocor.
func (s *Store) SuggestProducts(ctx context.Context, prefix string, limit int) ([]domain.Ref, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, slug, name
		FROM products
		WHERE deleted_at IS NULL AND is_active AND name ILIKE $1 || '%'
		ORDER BY length(name), name
		LIMIT $2`, prefix, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Ref, 0, limit)
	for rows.Next() {
		var r domain.Ref
		if err := rows.Scan(&r.ID, &r.Slug, &r.Name); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// VariantPurchaseInfo adalah data varian yang dibutuhkan aturan keranjang
// (§6.3): harga server, batas per pesanan, dan ketersediaan.
type VariantPurchaseInfo struct {
	VariantID       uuid.UUID
	ProductID       uuid.UUID
	ProductName     string
	ProductSlug     string
	SKU             string
	Price           int64
	MaxPerOrder     int32
	Available       int32
	IsVariantActive bool
	IsProductActive bool
}

// GetVariantPurchaseInfo mengambil data pembelian satu varian.
func (s *Store) GetVariantPurchaseInfo(ctx context.Context, variantID uuid.UUID) (VariantPurchaseInfo, error) {
	var v VariantPurchaseInfo
	err := s.db.QueryRow(ctx, `
		SELECT v.id, p.id, p.name, p.slug, v.sku, v.price, v.max_per_order,
		       (v.stock_on_hand - v.stock_reserved)::int,
		       v.is_active,
		       (p.is_active AND p.deleted_at IS NULL)
		FROM product_variants v
		JOIN products p ON p.id = v.product_id
		WHERE v.id = $1`, variantID,
	).Scan(
		&v.VariantID, &v.ProductID, &v.ProductName, &v.ProductSlug, &v.SKU, &v.Price, &v.MaxPerOrder,
		&v.Available, &v.IsVariantActive, &v.IsProductActive,
	)
	if err != nil {
		return VariantPurchaseInfo{}, mapErr(err)
	}
	return v, nil
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

// maskName menampilkan nama ulasan secara minimal: kata pertama + inisial
// (mis. "Rina S."). Ulasan publik tidak perlu nama lengkap (§7 privasi).
func maskName(full string) string {
	fields := strings.Fields(full)
	if len(fields) == 0 {
		return "Pengguna Kalengo"
	}
	if len(fields) == 1 {
		return fields[0]
	}
	initial := []rune(fields[1])
	return fields[0] + " " + strings.ToUpper(string(initial[0])) + "."
}
