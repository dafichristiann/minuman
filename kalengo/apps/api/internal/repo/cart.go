package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/domain"
)

// EnsureCart mengembalikan id keranjang pengguna, membuatnya bila belum ada.
//
// Trik `ON CONFLICT ... DO UPDATE` dipakai agar RETURNING selalu menghasilkan
// baris, termasuk saat baris sudah ada (DO NOTHING tidak mengembalikan apa pun).
func (s *Store) EnsureCart(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO carts (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
		RETURNING id`, userID).Scan(&id)
	if err != nil {
		return uuid.Nil, mapErr(err)
	}
	return id, nil
}

// GetCartItems mengambil isi keranjang beserta harga server terkini.
//
// Cacat produk (nonaktif / dihapus) tetap dikembalikan agar UI bisa
// menampilkan CART_ITEM_UNAVAILABLE alih-alih menghilangkannya diam-diam.
func (s *Store) GetCartItems(ctx context.Context, userID uuid.UUID) ([]domain.CartItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT ci.variant_id, p.id, p.slug, p.name, v.sku, v.volume_ml, v.pack_size,
		       ci.quantity, ci.added_price, v.price, v.max_per_order,
		       (v.stock_on_hand - v.stock_reserved)::int,
		       (v.is_active AND p.is_active AND p.deleted_at IS NULL)
		FROM carts c
		JOIN cart_items ci ON ci.cart_id = c.id
		JOIN product_variants v ON v.id = ci.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE c.user_id = $1
		ORDER BY p.name, v.volume_ml, v.pack_size`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.CartItem, 0, 8)
	for rows.Next() {
		var it domain.CartItem
		if err := rows.Scan(
			&it.VariantID, &it.ProductID, &it.ProductSlug, &it.ProductName, &it.SKU,
			&it.VolumeML, &it.PackSize, &it.Quantity, &it.AddedPrice, &it.UnitPrice,
			&it.MaxPerOrder, &it.Available, &it.IsAvailable,
		); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CartUpdatedAt mengembalikan waktu perubahan terakhir keranjang.
func (s *Store) CartUpdatedAt(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM cart_items ci
		JOIN carts c ON c.id = ci.cart_id
		WHERE c.user_id = $1`, userID).Scan(&count)
	return count, err
}

// UpsertCartItem menetapkan kuantitas sebuah varian (nilai absolut).
// added_price hanya diisi saat baris pertama kali dibuat, sehingga perubahan
// harga di server tetap terdeteksi sebagai CART_PRICE_CHANGED.
func (s *Store) UpsertCartItem(ctx context.Context, cartID, variantID uuid.UUID, quantity int32, price int64) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO cart_items (cart_id, variant_id, quantity, added_price)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = EXCLUDED.quantity`,
		cartID, variantID, quantity, price)
	return err
}

// MergeCartItem menambahkan kuantitas ke baris yang ada (dipakai /cart/merge),
// dibatasi `cap` agar tidak melewati max_per_order.
func (s *Store) MergeCartItem(ctx context.Context, cartID, variantID uuid.UUID, quantity int32, price int64, cap int32) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO cart_items (cart_id, variant_id, quantity, added_price)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (cart_id, variant_id) DO UPDATE
		SET quantity = LEAST(cart_items.quantity + EXCLUDED.quantity, $5::int)`,
		cartID, variantID, quantity, price, cap)
	return err
}

// DeleteCartItem menghapus satu varian dari keranjang.
func (s *Store) DeleteCartItem(ctx context.Context, cartID, variantID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM cart_items WHERE cart_id = $1 AND variant_id = $2`, cartID, variantID)
	return err
}

// ClearCart mengosongkan keranjang.
func (s *Store) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	return err
}

// ---------- wishlist ----------

// ListWishlist mengembalikan produk yang disimpan pengguna (§6.8).
func (s *Store) ListWishlist(ctx context.Context, userID uuid.UUID) ([]domain.CardProduct, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.id, p.slug, p.name, p.flavor,
		       COALESCE(MIN(v.price), 0)::bigint AS price_from,
		       COALESCE(SUM(v.stock_on_hand - v.stock_reserved), 0) > 0 AS in_stock,
		       w.created_at
		FROM wishlist_items w
		JOIN products p ON p.id = w.product_id
		LEFT JOIN product_variants v ON v.product_id = p.id AND v.is_active
		WHERE w.user_id = $1 AND p.deleted_at IS NULL
		GROUP BY p.id, w.created_at
		ORDER BY w.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.CardProduct, 0, 8)
	for rows.Next() {
		var it domain.CardProduct
		if err := rows.Scan(&it.ID, &it.Slug, &it.Name, &it.Flavor, &it.PriceFrom, &it.InStock, &it.AddedAt); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// AddWishlist menyimpan produk ke wishlist (idempoten).
func (s *Store) AddWishlist(ctx context.Context, userID, productID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO wishlist_items (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, product_id) DO NOTHING`, userID, productID)
	return err
}

// RemoveWishlist menghapus produk dari wishlist. 0 baris bukan error agar
// operasinya idempoten dari sisi klien.
func (s *Store) RemoveWishlist(ctx context.Context, userID, productID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM wishlist_items WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

// HasWishlist melaporkan apakah produk ada di wishlist pengguna.
func (s *Store) HasWishlist(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM wishlist_items WHERE user_id = $1 AND product_id = $2)`,
		userID, productID).Scan(&exists)
	return exists, err
}

// ---------- "kabari saya saat restock" ----------

// AddStockAlert mendaftarkan langganan notifikasi restock (§8.6 STOCK_BACK).
func (s *Store) AddStockAlert(ctx context.Context, userID, variantID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO stock_alert_subscriptions (user_id, variant_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, variant_id) DO UPDATE SET notified_at = NULL`,
		userID, variantID)
	return err
}

// RemoveStockAlert membatalkan langganan restock.
func (s *Store) RemoveStockAlert(ctx context.Context, userID, variantID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM stock_alert_subscriptions WHERE user_id = $1 AND variant_id = $2`,
		userID, variantID)
	return err
}
