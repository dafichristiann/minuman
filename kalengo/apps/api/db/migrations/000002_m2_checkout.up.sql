-- ============================================================
-- KALENGO — migrasi 000002: fondasi checkout M2
--
-- Aditif saja: tidak ada kolom/tabel §11.2 yang dihapus atau diubah tipenya.
-- Semua butir di bawah dicatat di docs/DEVIATIONS.md §7.
-- ============================================================

-- 1. Nomor pesanan.
-- §11.2 mewajibkan orders.order_no UNIQUE tetapi tidak menyediakan pembangkit.
-- Sequence membuat penomoran bebas balapan: dua checkout paralel tidak akan
-- pernah menerima nomor yang sama, tanpa perlu retry di aplikasi.
CREATE SEQUENCE IF NOT EXISTS order_no_seq START 1;

COMMENT ON SEQUENCE order_no_seq IS
  'Pembangkit nomor pesanan; format KLG-YYYYMMDD-<nextval> (lihat repo.GenerateOrderNo)';

-- 2. Sidik jari permintaan untuk idempotensi.
-- UNIQUE (user_id, idempotency_key) sudah menjamin satu pesanan per kunci, tetapi
-- tidak bisa membedakan "kunci sama, isi sama" dari "kunci sama, isi BEDA".
-- Tanpa kolom ini, permintaan kedua yang berbeda isi akan diam-diam dijawab
-- dengan pesanan lama — perilaku yang membingungkan dan sulit diaudit.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS request_fingerprint text;

COMMENT ON COLUMN orders.request_fingerprint IS
  'SHA-256 kanonik dari body permintaan checkout; mismatch pada kunci yang sama = API_IDEMPOTENCY_CONFLICT';

-- 3. Dokumentasi konvensi tanda delta pada ledger stok (§13.3 audit).
-- Angka ini dipakai untuk membuktikan stok tidak pernah hilang atau berganda.
COMMENT ON COLUMN stock_movements.delta IS
  'reservation/release menggerakkan stock_reserved (+/-); sale menggerakkan on_hand dan reserved (-); restock/adjustment/return menggerakkan on_hand (+)';
