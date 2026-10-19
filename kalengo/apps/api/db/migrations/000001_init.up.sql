-- ============================================================
-- KALENGO — migrasi 000001: skema awal v1
-- Sumber: §11.2 (DDL), §11.1 (konvensi), §11.3 (aturan integritas), §13.4
--
-- Dijalankan sebagai kalengo_migrator (pemilik skema, punya hak DDL).
-- Runtime (kalengo_app) TIDAK boleh menjalankan berkas ini.
--
-- Penyimpangan dari §11.2 yang disengaja (dicatat di docs/DEVIATIONS.md):
--   1. Tabel referensi (categories, brands, product_images, vouchers, reviews)
--      juga diberi created_at/updated_at + trigger, sesuai §11.1
--      ("Semua tabel punya created_at, updated_at (trigger)").
--   2. Indeks ditambahkan untuk SETIAP foreign key, sesuai §11.1
--      ("indeks untuk setiap FK dan kolom filter") — §11.2 hanya memuat sebagian.
--   3. notifications TIDAK diberi updated_at (tabel volume tinggi; hanya read_at
--      yang berubah). Rencana partisi & retensi ada di docs/DB-NOTES.md.
-- ============================================================

-- ===== EKSTENSI (§11.1) =====
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- pencarian toleran typo (§6.1)
CREATE EXTENSION IF NOT EXISTS citext;     -- email & kode voucher case-insensitive

-- ===== FUNGSI PEMBANTU =====
-- Trigger updated_at untuk semua tabel yang memilikinya (§11.1).
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ===== ENUMS (§11.2) =====
CREATE TYPE user_role AS ENUM ('customer', 'admin', 'staff');
CREATE TYPE order_status AS ENUM
  ('pending_payment', 'paid', 'processing', 'shipped', 'delivered', 'completed',
   'cancelled', 'expired', 'refund_requested', 'refunded');
CREATE TYPE payment_status AS ENUM
  ('pending', 'paid', 'failed', 'expired', 'refunded', 'partial_refund');
CREATE TYPE voucher_type AS ENUM ('percent', 'fixed', 'free_shipping');
CREATE TYPE stock_reason AS ENUM
  ('restock', 'sale', 'reservation', 'release', 'adjustment', 'return', 'damage');
CREATE TYPE notif_channel AS ENUM ('in_app', 'email', 'push');

-- ============================================================
-- USERS & AUTH
-- ============================================================
CREATE TABLE users (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email              citext NOT NULL UNIQUE,
  password_hash      text   NOT NULL,
  full_name          text   NOT NULL CHECK (char_length(full_name) BETWEEN 2 AND 100),
  phone              text   CHECK (phone IS NULL OR char_length(phone) BETWEEN 7 AND 20),
  role               user_role NOT NULL DEFAULT 'customer',
  email_verified_at  timestamptz,
  birth_date         date,
  failed_login_count int NOT NULL DEFAULT 0 CHECK (failed_login_count >= 0),
  locked_until       timestamptz,
  last_login_at      timestamptz,
  deleted_at         timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_users_upd BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Sesi refresh token: disimpan sebagai HASH, dengan family untuk deteksi reuse (§13.1).
CREATE TABLE refresh_tokens (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  family_id   uuid NOT NULL,
  token_hash  bytea NOT NULL UNIQUE,
  user_agent  text,
  ip          inet,
  expires_at  timestamptz NOT NULL,
  revoked_at  timestamptz,
  replaced_by uuid REFERENCES refresh_tokens(id) ON DELETE SET NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_rt_user   ON refresh_tokens(user_id);
CREATE INDEX idx_rt_family ON refresh_tokens(family_id);
CREATE INDEX idx_rt_replaced_by ON refresh_tokens(replaced_by);
-- Pembersihan harian token kedaluwarsa (§15)
CREATE INDEX idx_rt_expires ON refresh_tokens(expires_at);

-- Token sekali pakai: verifikasi email & reset password.
CREATE TABLE one_time_tokens (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose    text NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
  token_hash bytea NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  used_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_ott_user    ON one_time_tokens(user_id);
CREATE INDEX idx_ott_expires ON one_time_tokens(expires_at) WHERE used_at IS NULL;

CREATE TABLE addresses (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  label       text NOT NULL,
  recipient   text NOT NULL,
  phone       text NOT NULL,
  province    text NOT NULL,
  city        text NOT NULL,
  district    text NOT NULL,
  postal_code text NOT NULL,
  street      text NOT NULL,
  notes       text,
  area_id     text,                      -- id area Biteship (§6.4)
  is_default  boolean NOT NULL DEFAULT false,
  deleted_at  timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_addr_user ON addresses(user_id) WHERE deleted_at IS NULL;
-- Hanya satu alamat default per pengguna (§11.2)
CREATE UNIQUE INDEX uq_addr_default ON addresses(user_id)
  WHERE is_default AND deleted_at IS NULL;
CREATE TRIGGER trg_addresses_upd BEFORE UPDATE ON addresses
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- KATALOG
-- ============================================================
CREATE TABLE categories (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug       text NOT NULL UNIQUE,
  name       text NOT NULL,
  parent_id  uuid REFERENCES categories(id) ON DELETE SET NULL,
  sort_order int NOT NULL DEFAULT 0,
  is_active  boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_categories_parent ON categories(parent_id);
CREATE TRIGGER trg_categories_upd BEFORE UPDATE ON categories
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE brands (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug       text NOT NULL UNIQUE,
  name       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_brands_upd BEFORE UPDATE ON brands
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE products (
  id                        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug                      text NOT NULL UNIQUE,
  name                      text NOT NULL,
  description               text NOT NULL DEFAULT '',
  brand_id                  uuid REFERENCES brands(id) ON DELETE SET NULL,
  category_id               uuid REFERENCES categories(id) ON DELETE SET NULL,
  flavor                    text,
  nutrition                 jsonb NOT NULL DEFAULT '{}',
  mascot_svg_key            text,          -- kunci aset ilustrasi kaleng (§3.6)
  requires_age_verification boolean NOT NULL DEFAULT false,
  is_active                 boolean NOT NULL DEFAULT true,
  search                    tsvector GENERATED ALWAYS AS (
                              to_tsvector('simple',
                                coalesce(name, '') || ' ' ||
                                coalesce(flavor, '') || ' ' ||
                                coalesce(description, ''))
                            ) STORED,
  deleted_at                timestamptz,
  created_at                timestamptz NOT NULL DEFAULT now(),
  updated_at                timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_products_search   ON products USING gin (search);
CREATE INDEX idx_products_trgm     ON products USING gin (name gin_trgm_ops);
CREATE INDEX idx_products_cat      ON products(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_brand    ON products(brand_id)    WHERE deleted_at IS NULL;
CREATE INDEX idx_products_active   ON products(is_active, created_at DESC) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_products_upd BEFORE UPDATE ON products
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product_variants (
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id          uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  sku                 text NOT NULL UNIQUE,
  volume_ml           int  NOT NULL CHECK (volume_ml > 0),
  pack_size           int  NOT NULL DEFAULT 1 CHECK (pack_size > 0),
  price               bigint NOT NULL CHECK (price >= 0),
  compare_at_price    bigint CHECK (compare_at_price IS NULL OR compare_at_price >= price),
  weight_gram         int  NOT NULL CHECK (weight_gram > 0),
  stock_on_hand       int  NOT NULL DEFAULT 0 CHECK (stock_on_hand >= 0),
  stock_reserved      int  NOT NULL DEFAULT 0 CHECK (stock_reserved >= 0),
  low_stock_threshold int  NOT NULL DEFAULT 10 CHECK (low_stock_threshold >= 0),
  max_per_order       int  NOT NULL DEFAULT 24 CHECK (max_per_order > 0),
  is_active           boolean NOT NULL DEFAULT true,
  version             int  NOT NULL DEFAULT 1,   -- optimistic locking
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  -- Invarian inti anti-oversell (§11.3)
  CHECK (stock_reserved <= stock_on_hand)
);
CREATE INDEX idx_variants_product ON product_variants(product_id);
-- Untuk job "peringatan stok menipis" (§8.6 ADM_STOCK_LOW)
CREATE INDEX idx_variants_low_stock ON product_variants(product_id)
  WHERE is_active AND stock_on_hand - stock_reserved <= 10;
CREATE TRIGGER trg_variants_upd BEFORE UPDATE ON product_variants
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product_images (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  url        text NOT NULL,
  alt        text NOT NULL,
  sort_order int NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_images_product ON product_images(product_id, sort_order);
CREATE TRIGGER trg_images_upd BEFORE UPDATE ON product_images
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Buku besar stok: APPEND-ONLY. Tidak ada updated_at, tidak ada soft delete.
CREATE TABLE stock_movements (
  id         bigserial PRIMARY KEY,
  variant_id uuid NOT NULL REFERENCES product_variants(id) ON DELETE RESTRICT,
  delta      int  NOT NULL CHECK (delta <> 0),
  reason     stock_reason NOT NULL,
  order_id   uuid,                        -- FK ditambahkan setelah tabel orders
  actor_id   uuid REFERENCES users(id) ON DELETE SET NULL,
  note       text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_stock_mv_variant ON stock_movements(variant_id, created_at DESC);
CREATE INDEX idx_stock_mv_order   ON stock_movements(order_id) WHERE order_id IS NOT NULL;
CREATE INDEX idx_stock_mv_actor   ON stock_movements(actor_id) WHERE actor_id IS NOT NULL;

-- ============================================================
-- KERANJANG & WISHLIST
-- ============================================================
CREATE TABLE carts (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  updated_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_carts_upd BEFORE UPDATE ON carts
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cart_items (
  cart_id     uuid NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
  variant_id  uuid NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
  quantity    int  NOT NULL CHECK (quantity BETWEEN 1 AND 99),
  added_price bigint NOT NULL CHECK (added_price >= 0), -- deteksi CART_PRICE_CHANGED
  PRIMARY KEY (cart_id, variant_id)
);
CREATE INDEX idx_cart_items_variant ON cart_items(variant_id);

CREATE TABLE wishlist_items (
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, product_id)
);
CREATE INDEX idx_wishlist_product ON wishlist_items(product_id);

-- "Kabari saya saat restock" (§8.6 STOCK_NOTIFY_ON / STOCK_BACK)
CREATE TABLE stock_alert_subscriptions (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  variant_id  uuid NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
  notified_at timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, variant_id)
);
CREATE INDEX idx_stock_alert_pending ON stock_alert_subscriptions(variant_id)
  WHERE notified_at IS NULL;

-- ============================================================
-- VOUCHER
-- ============================================================
CREATE TABLE vouchers (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code          citext NOT NULL UNIQUE,
  type          voucher_type NOT NULL,
  value         bigint NOT NULL CHECK (value > 0),  -- persen (1-100) atau rupiah
  max_discount  bigint CHECK (max_discount IS NULL OR max_discount > 0),
  min_spend     bigint NOT NULL DEFAULT 0 CHECK (min_spend >= 0),
  usage_limit   int CHECK (usage_limit IS NULL OR usage_limit > 0),
  usage_count   int NOT NULL DEFAULT 0 CHECK (usage_count >= 0),
  per_user_limit int NOT NULL DEFAULT 1 CHECK (per_user_limit > 0),
  starts_at     timestamptz NOT NULL,
  ends_at       timestamptz NOT NULL,
  is_active     boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CHECK (ends_at > starts_at),
  CHECK (type <> 'percent' OR value BETWEEN 1 AND 100),
  CHECK (usage_limit IS NULL OR usage_count <= usage_limit)
);
-- Klaim kuota atomik: UPDATE ... WHERE usage_count < usage_limit (§13.1)
CREATE INDEX idx_vouchers_active ON vouchers(starts_at, ends_at) WHERE is_active;
CREATE TRIGGER trg_vouchers_upd BEFORE UPDATE ON vouchers
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- PESANAN
-- ============================================================
CREATE TABLE orders (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_no        text NOT NULL UNIQUE,          -- KLG-20261019-XXXX
  user_id         uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status          order_status NOT NULL DEFAULT 'pending_payment',
  subtotal        bigint NOT NULL CHECK (subtotal >= 0),
  discount        bigint NOT NULL DEFAULT 0 CHECK (discount >= 0),
  shipping_cost   bigint NOT NULL DEFAULT 0 CHECK (shipping_cost >= 0),
  total           bigint NOT NULL CHECK (total >= 0),
  voucher_id      uuid REFERENCES vouchers(id) ON DELETE SET NULL,
  address_snapshot jsonb NOT NULL,               -- salinan alamat saat pesan (§13.4)
  courier         text,
  courier_service text,
  idempotency_key text NOT NULL,
  expires_at      timestamptz NOT NULL,          -- batas bayar = akhir reservasi stok
  paid_at         timestamptz,
  shipped_at      timestamptz,
  delivered_at    timestamptz,
  cancelled_at    timestamptz,
  cancel_reason   text,
  version         int NOT NULL DEFAULT 1,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, idempotency_key),
  CHECK (total = subtotal - discount + shipping_cost),
  CHECK (discount <= subtotal)
);
CREATE INDEX idx_orders_user        ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_status_exp  ON orders(status, expires_at);
CREATE INDEX idx_orders_voucher     ON orders(voucher_id) WHERE voucher_id IS NOT NULL;
-- Job expire berjalan tiap menit (§15); indeks parsial mempercepat sapuan.
CREATE INDEX idx_orders_expiring ON orders(expires_at)
  WHERE status = 'pending_payment';
CREATE TRIGGER trg_orders_upd BEFORE UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE stock_movements
  ADD CONSTRAINT fk_sm_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE SET NULL;

-- Item pesanan menyimpan SNAPSHOT nama/harga agar riwayat tidak berubah (§13.4).
CREATE TABLE order_items (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id      uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  variant_id    uuid NOT NULL REFERENCES product_variants(id) ON DELETE RESTRICT,
  product_name  text NOT NULL,
  variant_label text NOT NULL,
  sku           text NOT NULL,
  unit_price    bigint NOT NULL CHECK (unit_price >= 0),
  quantity      int NOT NULL CHECK (quantity > 0),
  line_total    bigint NOT NULL CHECK (line_total = unit_price * quantity)
);
CREATE INDEX idx_oi_order   ON order_items(order_id);
CREATE INDEX idx_oi_variant ON order_items(variant_id);

CREATE TABLE order_status_history (
  id          bigserial PRIMARY KEY,
  order_id    uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  from_status order_status,
  to_status   order_status NOT NULL,
  actor_id    uuid REFERENCES users(id) ON DELETE SET NULL,
  note        text,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_osh_order ON order_status_history(order_id, created_at DESC);
CREATE INDEX idx_osh_actor ON order_status_history(actor_id) WHERE actor_id IS NOT NULL;

CREATE TABLE voucher_redemptions (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  voucher_id uuid NOT NULL REFERENCES vouchers(id) ON DELETE RESTRICT,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  order_id   uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  amount     bigint NOT NULL CHECK (amount > 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (order_id)
);
CREATE INDEX idx_vred_voucher ON voucher_redemptions(voucher_id);
CREATE INDEX idx_vred_user    ON voucher_redemptions(user_id);

-- ============================================================
-- PEMBAYARAN & PENGIRIMAN
-- ============================================================
CREATE TABLE payments (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id     uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  provider     text NOT NULL,
  provider_ref text UNIQUE,
  method       text,
  amount       bigint NOT NULL CHECK (amount >= 0),
  status       payment_status NOT NULL DEFAULT 'pending',
  paid_at      timestamptz,
  raw_response jsonb,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_pay_order  ON payments(order_id);
CREATE INDEX idx_pay_status ON payments(status, created_at DESC);
CREATE TRIGGER trg_payments_upd BEFORE UPDATE ON payments
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Idempotensi webhook + jejak kegagalan (§11.3 poin 5, §8.6 ADM_WEBHOOK_FAIL)
CREATE TABLE payment_webhook_events (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider        text NOT NULL,
  event_id        text NOT NULL,
  signature_valid boolean NOT NULL,
  payload         jsonb NOT NULL,
  processed_at    timestamptz,
  error           text,
  attempts        int NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  created_at      timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider, event_id)
);
CREATE INDEX idx_wh_unprocessed ON payment_webhook_events(created_at)
  WHERE processed_at IS NULL;

CREATE TABLE shipments (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id    uuid NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
  courier     text NOT NULL,
  tracking_no text,
  status      text NOT NULL DEFAULT 'created',
  events      jsonb NOT NULL DEFAULT '[]',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_shipments_upd BEFORE UPDATE ON shipments
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- ULASAN (hanya dari pembeli dengan pesanan completed, §6.8)
-- ============================================================
CREATE TABLE reviews (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  product_id    uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  order_item_id uuid NOT NULL UNIQUE REFERENCES order_items(id) ON DELETE CASCADE,
  rating        smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
  body          text CHECK (body IS NULL OR char_length(body) <= 2000),
  is_visible    boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_reviews_product ON reviews(product_id, created_at DESC) WHERE is_visible;
CREATE INDEX idx_reviews_user    ON reviews(user_id);
CREATE TRIGGER trg_reviews_upd BEFORE UPDATE ON reviews
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- NOTIFIKASI (§8)
-- ============================================================
-- `code` HARUS cocok dengan katalog §8.6 (dan registry apperr di backend).
CREATE TABLE notifications (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code       text NOT NULL,
  severity   text NOT NULL CHECK (severity IN ('success', 'info', 'warning', 'error', 'critical')),
  title      text NOT NULL,
  body       text,
  data       jsonb NOT NULL DEFAULT '{}',
  dedupe_key text,
  read_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_notif_user_unread ON notifications(user_id, created_at DESC)
  WHERE read_at IS NULL;
CREATE INDEX idx_notif_user_all ON notifications(user_id, created_at DESC);
-- Deduplikasi notifikasi yang sama (§8.3)
CREATE UNIQUE INDEX uq_notif_dedupe ON notifications(user_id, dedupe_key)
  WHERE dedupe_key IS NOT NULL;
-- Retensi 90 hari (§8.2, §15)
CREATE INDEX idx_notif_retention ON notifications(created_at);

CREATE TABLE notification_preferences (
  user_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  channel  notif_channel NOT NULL,
  category text NOT NULL,
  enabled  boolean NOT NULL DEFAULT true,
  PRIMARY KEY (user_id, channel, category)
);
-- 'security' dan 'order' pada channel email/in_app tidak boleh dimatikan;
-- aturan ini DIPAKSA di service (lihat docs/NOTIFICATIONS.md), bukan di DB,
-- supaya tetap bisa diaudit dan diuji.

CREATE TABLE push_subscriptions (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  endpoint   text NOT NULL UNIQUE,
  p256dh     text NOT NULL,
  auth       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_push_user ON push_subscriptions(user_id);

CREATE TABLE user_ui_settings (
  user_id       uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  notif_mode    text NOT NULL DEFAULT 'normal'
                  CHECK (notif_mode IN ('normal', 'calm', 'silent')),
  reduce_motion boolean NOT NULL DEFAULT false,
  sound_enabled boolean NOT NULL DEFAULT false
);

-- ============================================================
-- AUDIT LOG (append-only, berantai hash — §13.2)
-- ============================================================
CREATE TABLE audit_logs (
  id          bigserial PRIMARY KEY,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  actor_id    uuid,
  actor_role  text,
  ip          inet,
  user_agent  text,
  request_id  text,
  action      text NOT NULL,
  entity      text NOT NULL,
  entity_id   text,
  before      jsonb,
  after       jsonb,
  prev_hash   bytea,
  row_hash    bytea
);
CREATE INDEX idx_audit_entity ON audit_logs(entity, entity_id, occurred_at DESC);
CREATE INDEX idx_audit_actor  ON audit_logs(actor_id, occurred_at DESC);
CREATE INDEX idx_audit_action ON audit_logs(action, occurred_at DESC);
-- Verifikasi rantai harian membaca berurutan; indeks ini membantu rentang waktu.
CREATE INDEX idx_audit_time   ON audit_logs(occurred_at);

-- Larang UPDATE/DELETE pada audit_logs (lapisan 1: trigger).
CREATE OR REPLACE FUNCTION audit_immutable() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs bersifat append-only: % tidak diizinkan', TG_OP
    USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_audit_no_mod BEFORE UPDATE OR DELETE ON audit_logs
  FOR EACH ROW EXECUTE FUNCTION audit_immutable();

-- Lapisan 2: cabut hak UPDATE/DELETE dari role aplikasi (bila role-nya ada).
-- Di lingkungan yang tidak memakai role terpisah (mis. testcontainer), blok ini
-- dilewati dengan aman; trigger di atas tetap melindungi.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kalengo_app') THEN
    REVOKE UPDATE, DELETE, TRUNCATE ON audit_logs FROM kalengo_app;
    REVOKE UPDATE, DELETE, TRUNCATE ON stock_movements FROM kalengo_app;
    REVOKE UPDATE, DELETE, TRUNCATE ON order_status_history FROM kalengo_app;
    -- Penghapusan audit lama (>1 tahun) hanya lewat job dengan role khusus.
    COMMENT ON TABLE audit_logs IS
      'Append-only. Retensi minimal 1 tahun (§13.2). Purge hanya lewat role khusus, bukan kalengo_app.';
  END IF;
END
$$;

-- ============================================================
-- KOMENTAR SKEMA (memudahkan review §13.4)
-- ============================================================
COMMENT ON COLUMN product_variants.version IS
  'Optimistic locking; dinaikkan setiap perubahan stok/harga.';
COMMENT ON COLUMN orders.address_snapshot IS
  'Snapshot alamat saat pesanan dibuat; riwayat tidak boleh berubah walau alamat diedit.';
COMMENT ON COLUMN order_items.product_name IS
  'Snapshot nama produk & harga saat pembelian (§13.4).';
COMMENT ON COLUMN vouchers.usage_count IS
  'Dinaikkan dengan UPDATE ... WHERE usage_count < usage_limit (atomik, §13.1).';
COMMENT ON COLUMN notifications.dedupe_key IS
  'Kunci deduplikasi notifikasi (§8.3); unik per pengguna bila tidak NULL.';
