-- ============================================================
-- KALENGO — data contoh untuk PENGEMBANGAN saja (bukan migrasi)
--
--   make seed
--
-- Dijalankan sebagai kalengo_migrator (karena menulis ke tabel yang dimiliki
-- role itu). Idempoten: memakai UUID tetap + ON CONFLICT DO NOTHING, sehingga
-- aman dijalankan berulang.
--
-- TIDAK memuat pengguna: hash argon2id tidak boleh ditulis tangan. Pembuatan
-- akun admin/customer ada di `make seed-admin` (ditambahkan pada M1) supaya
-- parameter argon2id diambil dari konfigurasi runtime.
-- ============================================================

BEGIN;

-- ---------------------------------------------------------------
-- Kategori
-- ---------------------------------------------------------------
INSERT INTO categories (id, slug, name, sort_order) VALUES
  ('11111111-0000-4000-8000-000000000001', 'soda',        'Soda',         1),
  ('11111111-0000-4000-8000-000000000002', 'kopi-susu',   'Kopi Susu',    2),
  ('11111111-0000-4000-8000-000000000003', 'teh',         'Teh',          3),
  ('11111111-0000-4000-8000-000000000004', 'rendah-gula', 'Rendah Gula',  4),
  ('11111111-0000-4000-8000-000000000005', 'buah',        'Rasa Buah',    5)
ON CONFLICT (slug) DO NOTHING;

-- ---------------------------------------------------------------
-- Merek
-- ---------------------------------------------------------------
INSERT INTO brands (id, slug, name) VALUES
  ('22222222-0000-4000-8000-000000000001', 'kalengo-original', 'Kalengo Original'),
  ('22222222-0000-4000-8000-000000000002', 'kalengo-lite',     'Kalengo Lite'),
  ('22222222-0000-4000-8000-000000000003', 'nusantara-brew',   'Nusantara Brew')
ON CONFLICT (slug) DO NOTHING;

-- ---------------------------------------------------------------
-- Produk
-- Harga dalam RUPIAH BULAT (§11.1). mascot_svg_key menunjuk aset §3.6.
-- ---------------------------------------------------------------
INSERT INTO products
  (id, slug, name, description, brand_id, category_id, flavor, nutrition, mascot_svg_key, requires_age_verification)
VALUES
  ('33333333-0000-4000-8000-000000000001',
   'kalengo-soda-lemon-330',
   'Kalengo Soda Lemon',
   'Soda lemon dengan gelembung halus. Dingin lebih nikmat.',
   '22222222-0000-4000-8000-000000000001',
   '11111111-0000-4000-8000-000000000001',
   'Lemon',
   '{"energi_kcal":110,"gula_g":24,"natrium_mg":35,"porsi_ml":330}',
   'can-soda-lemon', false),

  ('33333333-0000-4000-8000-000000000002',
   'kalengo-kopi-susu-240',
   'Kalengo Kopi Susu',
   'Kopi susu gula aren dalam kaleng. Pahit lembut, manis pas.',
   '22222222-0000-4000-8000-000000000003',
   '11111111-0000-4000-8000-000000000002',
   'Kopi Susu Gula Aren',
   '{"energi_kcal":150,"gula_g":18,"natrium_mg":60,"kafein_mg":80,"porsi_ml":240}',
   'can-kopi-susu', false),

  ('33333333-0000-4000-8000-000000000003',
   'kalengo-teh-melati-350',
   'Kalengo Teh Melati',
   'Teh melati wangi, tanpa pemanis buatan.',
   '22222222-0000-4000-8000-000000000001',
   '11111111-0000-4000-8000-000000000003',
   'Melati',
   '{"energi_kcal":60,"gula_g":9,"natrium_mg":20,"porsi_ml":350}',
   'can-teh-melati', false),

  ('33333333-0000-4000-8000-000000000004',
   'kalengo-lite-soda-zero-330',
   'Kalengo Lite Soda Zero',
   'Nol gula, rasa jeruk nipis. Untuk yang jaga asupan.',
   '22222222-0000-4000-8000-000000000002',
   '11111111-0000-4000-8000-000000000004',
   'Jeruk Nipis',
   '{"energi_kcal":5,"gula_g":0,"natrium_mg":25,"porsi_ml":330}',
   'can-lite-zero', false),

  ('33333333-0000-4000-8000-000000000005',
   'kalengo-buah-mangga-330',
   'Kalengo Rasa Mangga',
   'Mangga harum manis, favorit anak-anak.',
   '22222222-0000-4000-8000-000000000001',
   '11111111-0000-4000-8000-000000000005',
   'Mangga',
   '{"energi_kcal":130,"gula_g":28,"natrium_mg":30,"porsi_ml":330}',
   'can-buah-mangga', false),

  ('33333333-0000-4000-8000-000000000006',
   'kalengo-soda-ginger-330',
   'Kalengo Soda Jahe',
   'Soda jahe hangat di tenggorokan. Produk khusus 21+.',
   '22222222-0000-4000-8000-000000000001',
   '11111111-0000-4000-8000-000000000001',
   'Jahe',
   '{"energi_kcal":115,"gula_g":22,"natrium_mg":40,"porsi_ml":330}',
   'can-soda-jahe', true)
ON CONFLICT (slug) DO NOTHING;

-- ---------------------------------------------------------------
-- Varian
-- Termasuk satu varian sengaja HABIS dan satu STOK MENIPIS agar UI dan
-- notifikasi STOCK_OUT / ADM_STOCK_LOW bisa diuji tanpa manipulasi manual.
-- ---------------------------------------------------------------
INSERT INTO product_variants
  (id, product_id, sku, volume_ml, pack_size, price, compare_at_price, weight_gram,
   stock_on_hand, stock_reserved, low_stock_threshold, max_per_order)
VALUES
  -- Soda Lemon
  ('44444444-0000-4000-8000-000000000001', '33333333-0000-4000-8000-000000000001', 'KLG-SLM-330-1',  330, 1,  6500, NULL,   370, 240, 0, 12, 24),
  ('44444444-0000-4000-8000-000000000002', '33333333-0000-4000-8000-000000000001', 'KLG-SLM-330-6',  330, 6, 36000, 39000, 2250,  90, 0, 10, 8),
  ('44444444-0000-4000-8000-000000000003', '33333333-0000-4000-8000-000000000001', 'KLG-SLM-330-24', 330, 24, 138000, 156000, 9000, 30, 0, 6, 4),

  -- Kopi Susu
  ('44444444-0000-4000-8000-000000000004', '33333333-0000-4000-8000-000000000002', 'KLG-KPS-240-1',  240, 1,  8000, NULL,   280, 180, 0, 12, 24),
  ('44444444-0000-4000-8000-000000000005', '33333333-0000-4000-8000-000000000002', 'KLG-KPS-240-6',  240, 6, 45000, 48000, 1700,  60, 0, 10, 8),
  -- stok menipis: 6 tersedia dari ambang 10 → memicu ADM_STOCK_LOW
  ('44444444-0000-4000-8000-000000000006', '33333333-0000-4000-8000-000000000002', 'KLG-KPS-240-12', 240, 12, 88000, NULL,  3400,   6, 0, 10, 4),

  -- Teh Melati
  ('44444444-0000-4000-8000-000000000007', '33333333-0000-4000-8000-000000000003', 'KLG-THM-350-1',  350, 1,  6000, NULL,   380, 200, 0, 12, 24),
  ('44444444-0000-4000-8000-000000000008', '33333333-0000-4000-8000-000000000003', 'KLG-THM-350-6',  350, 6, 33000, NULL,  2300,  75, 0, 10, 8),

  -- Lite Soda Zero
  ('44444444-0000-4000-8000-000000000009', '33333333-0000-4000-8000-000000000004', 'KLG-LTZ-330-1',  330, 1,  7000, NULL,   360, 150, 0, 12, 24),
  ('44444444-0000-4000-8000-00000000000a', '33333333-0000-4000-8000-000000000004', 'KLG-LTZ-330-6',  330, 6, 39000, 42000, 2200,  48, 0, 10, 8),

  -- Rasa Mangga
  ('44444444-0000-4000-8000-00000000000b', '33333333-0000-4000-8000-000000000005', 'KLG-MGG-330-1',  330, 1,  6500, NULL,   370, 220, 0, 12, 24),
  -- sengaja HABIS agar alur STOCK_OUT / "kabari saya" bisa diuji
  ('44444444-0000-4000-8000-00000000000c', '33333333-0000-4000-8000-000000000005', 'KLG-MGG-330-6',  330, 6, 36000, NULL,  2250,   0, 0, 10, 8),

  -- Soda Jahe (21+)
  ('44444444-0000-4000-8000-00000000000d', '33333333-0000-4000-8000-000000000006', 'KLG-SJG-330-1',  330, 1,  7500, NULL,   370, 120, 0, 12, 12)
ON CONFLICT (sku) DO NOTHING;

-- ---------------------------------------------------------------
-- Gambar produk (URL placeholder ke MinIO dev; diisi aset nyata pada M6)
-- ---------------------------------------------------------------
INSERT INTO product_images (id, product_id, url, alt, sort_order) VALUES
  ('55555555-0000-4000-8000-000000000001', '33333333-0000-4000-8000-000000000001', 'http://localhost:9000/kalengo-media/public/soda-lemon-1.webp', 'Kaleng Kalengo Soda Lemon tampak depan', 0),
  ('55555555-0000-4000-8000-000000000002', '33333333-0000-4000-8000-000000000002', 'http://localhost:9000/kalengo-media/public/kopi-susu-1.webp', 'Kaleng Kalengo Kopi Susu tampak depan', 0),
  ('55555555-0000-4000-8000-000000000003', '33333333-0000-4000-8000-000000000003', 'http://localhost:9000/kalengo-media/public/teh-melati-1.webp', 'Kaleng Kalengo Teh Melati tampak depan', 0),
  ('55555555-0000-4000-8000-000000000004', '33333333-0000-4000-8000-000000000004', 'http://localhost:9000/kalengo-media/public/lite-zero-1.webp', 'Kaleng Kalengo Lite Soda Zero tampak depan', 0),
  ('55555555-0000-4000-8000-000000000005', '33333333-0000-4000-8000-000000000005', 'http://localhost:9000/kalengo-media/public/buah-mangga-1.webp', 'Kaleng Kalengo Rasa Mangga tampak depan', 0),
  ('55555555-0000-4000-8000-000000000006', '33333333-0000-4000-8000-000000000006', 'http://localhost:9000/kalengo-media/public/soda-jahe-1.webp', 'Kaleng Kalengo Soda Jahe tampak depan', 0)
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------
-- Voucher contoh
-- ---------------------------------------------------------------
INSERT INTO vouchers
  (id, code, type, value, max_discount, min_spend, usage_limit, per_user_limit, starts_at, ends_at, is_active)
VALUES
  ('66666666-0000-4000-8000-000000000001', 'NGOPI10',   'percent',       10, 15000, 50000,  1000, 1, now() - interval '1 day', now() + interval '60 days', true),
  ('66666666-0000-4000-8000-000000000002', 'HEMAT5000', 'fixed',       5000,  NULL, 40000,  500,  1, now() - interval '1 day', now() + interval '30 days', true),
  ('66666666-0000-4000-8000-000000000003', 'GRATISONGKIR', 'free_shipping', 1, NULL, 100000, 200, 1, now() - interval '1 day', now() + interval '30 days', true),
  -- sengaja kedaluwarsa untuk menguji VOUCHER_EXPIRED
  ('66666666-0000-4000-8000-000000000004', 'KEDALUWARSA', 'fixed',      10000, NULL, 0, NULL, 1, now() - interval '30 days', now() - interval '1 day', true)
ON CONFLICT (code) DO NOTHING;

COMMIT;

\echo '[seed] data contoh siap: 5 kategori, 3 merek, 6 produk, 14 varian, 6 gambar, 4 voucher'
\echo '[seed] catatan: akun pengguna dibuat lewat make seed-admin (M1)'
