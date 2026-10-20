-- Rollback migrasi 000002.
--
-- request_fingerprint dibuang beserta datanya: kolom ini hanya membantu
-- mendeteksi pemakaian ulang kunci idempotensi dengan isi berbeda, bukan sumber
-- kebenaran pesanan. Pesanan, item, dan pembayaran tidak tersentuh.

ALTER TABLE orders DROP COLUMN IF EXISTS request_fingerprint;

DROP SEQUENCE IF EXISTS order_no_seq;
