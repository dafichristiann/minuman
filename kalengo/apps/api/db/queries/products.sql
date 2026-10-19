-- ============================================================
-- Query katalog (§12.2) — dipakai mulai M1.
--
-- Catatan stok: angka stok publik DIBULATKAN di service (§12.2: "tersisa ≤10"),
-- bukan di query, agar kebijakan bisnis mudah diubah tanpa mengubah SQL.
-- ============================================================

-- name: GetProductBySlug :one
SELECT p.*
FROM products p
WHERE p.slug = $1
  AND p.deleted_at IS NULL
  AND p.is_active;

-- name: GetProductByID :one
SELECT p.*
FROM products p
WHERE p.id = $1
  AND p.deleted_at IS NULL;

-- name: ListProducts :many
-- Paginasi cursor (§6.1): urut (created_at, id) menurun; kursor dari item terakhir.
SELECT p.id, p.slug, p.name, p.flavor, p.brand_id, p.category_id,
       p.mascot_svg_key, p.requires_age_verification, p.created_at
FROM products p
WHERE p.deleted_at IS NULL
  AND p.is_active
  AND (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id')::uuid)
  AND (sqlc.narg('brand_id')::uuid IS NULL OR p.brand_id = sqlc.narg('brand_id')::uuid)
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR (p.created_at, p.id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY p.created_at DESC, p.id DESC
LIMIT $1;

-- name: SearchProducts :many
-- Pencarian gabungan full-text + trigram untuk toleransi typo (§6.1).
SELECT p.id, p.slug, p.name, p.flavor,
       greatest(
         ts_rank(p.search, websearch_to_tsquery('simple', sqlc.arg('q')::text)),
         similarity(p.name, sqlc.arg('q')::text)
       ) AS rank_score
FROM products p
WHERE p.deleted_at IS NULL
  AND p.is_active
  AND (
    p.search @@ websearch_to_tsquery('simple', sqlc.arg('q')::text)
    OR p.name % sqlc.arg('q')::text
  )
ORDER BY rank_score DESC, p.name
LIMIT $1;

-- name: SuggestProducts :many
-- Saran pencarian otomatis (debounce 250 ms di klien, §6.1).
SELECT p.slug, p.name
FROM products p
WHERE p.deleted_at IS NULL
  AND p.is_active
  AND p.name ILIKE sqlc.arg('prefix')::text || '%'
ORDER BY length(p.name), p.name
LIMIT $1;

-- name: ListVariantsByProduct :many
SELECT *
FROM product_variants
WHERE product_id = $1
  AND is_active
ORDER BY volume_ml, pack_size;

-- name: GetVariantByID :one
SELECT *
FROM product_variants
WHERE id = $1;

-- name: ReserveStock :execrows
-- Inti anti-oversell (§11.3 poin 1). 0 baris = stok kurang.
-- Pemanggil WAJIB mengurutkan variant_id agar tidak deadlock.
UPDATE product_variants
SET stock_reserved = stock_reserved + sqlc.arg('qty')::int,
    version        = version + 1
WHERE id = sqlc.arg('variant_id')::uuid
  AND is_active
  AND stock_on_hand - stock_reserved >= sqlc.arg('qty')::int;

-- name: ReleaseStock :execrows
UPDATE product_variants
SET stock_reserved = stock_reserved - sqlc.arg('qty')::int,
    version        = version + 1
WHERE id = sqlc.arg('variant_id')::uuid
  AND stock_reserved >= sqlc.arg('qty')::int;

-- name: CommitStockSale :execrows
-- Saat pembayaran sukses: stok fisik berkurang DAN reservasi dilepas (§11.3 poin 2).
UPDATE product_variants
SET stock_on_hand  = stock_on_hand - sqlc.arg('qty')::int,
    stock_reserved = stock_reserved - sqlc.arg('qty')::int,
    version        = version + 1
WHERE id = sqlc.arg('variant_id')::uuid
  AND stock_on_hand >= sqlc.arg('qty')::int
  AND stock_reserved >= sqlc.arg('qty')::int;

-- name: AdjustStock :execrows
-- Perubahan stok manual admin; wajib disertai alasan + catatan (§12.7).
UPDATE product_variants
SET stock_on_hand = stock_on_hand + sqlc.arg('delta')::int,
    version       = version + 1
WHERE id = sqlc.arg('variant_id')::uuid
  AND stock_on_hand + sqlc.arg('delta')::int >= 0;

-- name: InsertStockMovement :one
INSERT INTO stock_movements (variant_id, delta, reason, order_id, actor_id, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at;

-- name: ListLowStockVariants :many
-- Peringatan stok menipis (§8.6 ADM_STOCK_LOW).
SELECT v.id, v.sku, v.product_id, v.stock_on_hand, v.stock_reserved,
       v.low_stock_threshold, p.name AS product_name
FROM product_variants v
JOIN products p ON p.id = v.product_id
WHERE v.is_active
  AND p.deleted_at IS NULL
  AND v.stock_on_hand - v.stock_reserved <= v.low_stock_threshold
ORDER BY (v.stock_on_hand - v.stock_reserved) ASC
LIMIT $1;
