-- ============================================================
-- KALENGO — migrasi 000001 DOWN: membatalkan skema awal v1
--
-- Dijalankan sebagai kalengo_migrator.
-- Wajib lulus uji `up → down → up` pada DB kosong DAN DB berisi (§13.3),
-- jadi seluruh pernyataan memakai IF EXISTS.
--
-- PERINGATAN: berkas ini MENGHAPUS SEMUA DATA. Jangan pernah dijalankan di
-- staging/production tanpa snapshot terverifikasi.
-- ============================================================

-- ===== AUDIT LOG =====
DROP TRIGGER IF EXISTS trg_audit_no_mod ON audit_logs;
DROP TABLE IF EXISTS audit_logs;

-- ===== NOTIFIKASI =====
DROP TABLE IF EXISTS user_ui_settings;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;

-- ===== ULASAN =====
DROP TABLE IF EXISTS reviews;

-- ===== PEMBAYARAN & PENGIRIMAN =====
DROP TABLE IF EXISTS shipments;
DROP TABLE IF EXISTS payment_webhook_events;
DROP TABLE IF EXISTS payments;

-- ===== PESANAN =====
DROP TABLE IF EXISTS voucher_redemptions;
DROP TABLE IF EXISTS order_status_history;
DROP TABLE IF EXISTS order_items;

-- FK dari stock_movements ke orders harus dilepas sebelum orders dihapus.
ALTER TABLE IF EXISTS stock_movements DROP CONSTRAINT IF EXISTS fk_sm_order;
DROP TABLE IF EXISTS orders;

-- ===== VOUCHER =====
DROP TABLE IF EXISTS vouchers;

-- ===== KERANJANG & WISHLIST =====
DROP TABLE IF EXISTS stock_alert_subscriptions;
DROP TABLE IF EXISTS wishlist_items;
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS carts;

-- ===== KATALOG =====
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS brands;
DROP TABLE IF EXISTS categories;

-- ===== USERS & AUTH =====
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS one_time_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

-- ===== FUNGSI =====
DROP FUNCTION IF EXISTS audit_immutable();
DROP FUNCTION IF EXISTS set_updated_at();

-- ===== ENUMS =====
DROP TYPE IF EXISTS notif_channel;
DROP TYPE IF EXISTS stock_reason;
DROP TYPE IF EXISTS voucher_type;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS user_role;

-- ===== EKSTENSI =====
-- pg_stat_statements SENGAJA tidak dihapus: dibuat oleh initdb (docker/postgres/initdb),
-- bukan oleh migrasi ini, dan dipakai lintas database untuk audit query (§13.3).
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS pgcrypto;
