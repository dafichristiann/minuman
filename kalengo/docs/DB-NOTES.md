# Catatan Database

Referensi: PRD §11 (skema), §11.1 (konvensi), §13.1 (hak akses), §13.4 (checklist
review skema). Semua penyimpangan dari §11.2 dicatat di
[`DEVIATIONS.md`](DEVIATIONS.md) §2.

---

## 1. Ringkasan skema v1

| | Jumlah |
|---|---|
| Tabel | 28 |
| Enum PostgreSQL | 6 (`user_role`, `order_status`, `payment_status`, `voucher_type`, `stock_reason`, `notif_channel`) |
| Indeks | 48 (termasuk indeks untuk **setiap** foreign key) |
| `CHECK` constraint | 42 |
| Trigger `updated_at` | 13 |
| Trigger imutabilitas | 1 (`audit_logs`, `stock_movements`, `order_status_history`) |

Tabel menurut peran:

| Kelompok | Tabel |
|---|---|
| Identitas | `users`, `refresh_tokens`, `one_time_tokens`, `addresses` |
| Katalog | `categories`, `brands`, `products`, `product_variants`, `product_images` |
| Persediaan | `stock_movements` *(append-only)*, `stock_alert_subscriptions` |
| Belanja | `carts`, `cart_items`, `wishlist_items` |
| Promo | `vouchers`, `voucher_redemptions` |
| Transaksi | `orders`, `order_items`, `order_status_history` *(append-only)*, `payments`, `payment_webhook_events`, `shipments` |
| Umpan balik | `reviews` |
| Notifikasi | `notifications`, `notification_preferences`, `push_subscriptions` |
| Preferensi | `user_ui_settings` |
| Audit | `audit_logs` *(append-only, ber-hash chain)* |

---

## 2. Checklist review skema §13.4

| Butir checklist | Status | Bukti |
|---|---|---|
| Setiap FK terindeks | ✅ | 48 indeks; setiap `REFERENCES` punya indeks pendamping di `000001_init.up.sql` |
| Tidak ada `text` bebas tanpa batas pada input publik | ✅ | `full_name` 2–100, `phone` 7–20, `reviews.body` ≤ 2000, dsb. |
| `CHECK` untuk angka (harga ≥ 0, stok ≥ 0, total konsisten) | ✅ | 42 constraint; lihat §3 |
| Soft delete konsisten + indeks parsial menghormatinya | ✅ | `deleted_at` + indeks parsial `WHERE deleted_at IS NULL` |
| Tidak ada `float` untuk uang | ✅ | 0 kolom `real`/`double precision`; uang = `bigint` rupiah |
| Snapshot data pesanan | ✅ | `order_items` menyimpan `product_name`, `variant_label`, `sku`, `unit_price`, `line_total`; `orders.address_snapshot` (jsonb) menyimpan alamat saat pesan |
| Rencana partisi/retensi untuk tabel besar | ⚠️ **belum** | Rencana ada di §5 di bawah; implementasi dijadwalkan M5 |
| Semua timestamp `timestamptz` (UTC) | ✅ | 58 kolom `timestamptz`; 0 `timestamp` tanpa zona; server `timezone=UTC` |
| Peran DB least-privilege terdokumentasi | ✅ | §4 |

Satu butir masih ⚠️. Jangan tandai selesai sebelum partisi benar-benar dibuat.

---

## 3. Invarian yang dijaga database

Invarian ini **tidak** boleh hanya hidup di kode aplikasi. Bila Anda menemukan
salah satunya bisa dilanggar lewat SQL langsung, itu bug skema.

### 3.1 Uang

- Semua kolom uang: `bigint`, satuan **rupiah bulat**. Tidak ada sen.
- `orders.total = subtotal - discount + shipping_cost` — dijaga `CHECK`.
- `orders.discount <= subtotal` — dijaga `CHECK`.
- `order_items.line_total = unit_price * quantity` — dijaga `CHECK`.
- `orders.address_snapshot` (jsonb) menyalin alamat **saat pesanan dibuat**.
  Riwayat pesanan tidak boleh berubah walau pengguna menyunting alamatnya.
- `compare_at_price IS NULL OR compare_at_price >= price` — harga coret tidak
  boleh lebih murah dari harga jual.
- `vouchers`: `CHECK (type <> 'percent' OR value BETWEEN 1 AND 100)` — persen
  tidak bisa 0 atau 500.

### 3.2 Persediaan

- `stock_on_hand >= 0`, `stock_reserved >= 0`, dan
  `CHECK (stock_reserved <= stock_on_hand)` pada `product_variants`.
  Reservasi tidak boleh melebihi stok fisik.
- Setiap perubahan stok ditulis ke `stock_movements` (`delta <> 0`). Tabel ini
  append-only: `UPDATE`/`DELETE` ditolak trigger.
- **Aturan aplikasi:** jangan pernah `UPDATE product_variants.stock_on_hand`
  tanpa menulis baris `stock_movements` di transaksi yang sama. Uji audit di M2
  akan memeriksa konsistensi
  `stock_on_hand = SUM(delta)` per varian.

### 3.3 Promo

- `vouchers`: `CHECK (ends_at > starts_at)`.
- `CHECK (usage_limit IS NULL OR usage_count <= usage_limit)` — kuota tidak
  bisa terlampaui, bahkan oleh dua transaksi bersamaan (diperkuat `UPDATE ...
  WHERE usage_count < usage_limit` di service).
- `voucher_redemptions` memberi idempotensi satu pemakaian per pengguna.

### 3.4 Ulasan

- Hanya boleh dibuat setelah pesanan `delivered` (ditegakkan service + FK ke
  `order_items`).
- `rating BETWEEN 1 AND 5`.
- Satu ulasan per `order_item` (unique).

### 3.5 Audit

- `audit_logs` punya `prev_hash` + `row_hash` sehingga setiap baris terikat ke
  baris sebelumnya. `internal/audit.Verify` memeriksa seluruh rantai.
- `bigserial` (bukan UUID) dipakai khusus di `audit_logs`, `stock_movements`,
  dan `order_status_history` karena **urutan** adalah bagian dari makna: rantai
  hash dan riwayat status tidak boleh punya celah atau urutan acak.

---

## 4. Peran database (least privilege, §13.1)

Dibuat oleh `docker/postgres/initdb/01-roles.sql` saat volume pertama dibuat.

| Role | Login | Hak | Dipakai oleh |
|---|---|---|---|
| `kalengo_migrator` | ya | Pemilik skema. Boleh DDL, `TRUNCATE`, dan mengubah owner. | `migrate`, seed, `psql` admin |
| `kalengo_app` | ya | Hanya DML pada tabel aplikasi. `UPDATE`/`DELETE`/`TRUNCATE` **ditolak** di tiga tabel append-only. | API & worker (via `DATABASE_URL`) |
| `postgres` | ya | Superuser. **Jangan** dipakai aplikasi. | bootstrap, pemulihan bencana |

Aturan:

- `DATABASE_URL` (API) memakai `kalengo_app`. `MIGRATE_DATABASE_URL` memakai
  `kalengo_migrator`. Keduanya adalah variabel yang berbeda — jangan digabung.
- Bila aplikasi butuh hak DDL, itu tanda bug: perbaiki, jangan naikkan hak role.
- Rotasi kata sandi: ubah `KALENGO_APP_PASSWORD`/`KALENGO_MIGRATOR_PASSWORD` di
  `.env`, lalu `ALTER ROLE ... PASSWORD` di PostgreSQL. Tidak perlu deploy ulang
  bila kredensial dibaca dari secret manager.

Uji cepat bahwa hak akses bekerja:

```bash
psql "postgres://kalengo_app:app_dev_password@localhost:5432/kalengo?sslmode=disable" -c "DELETE FROM audit_logs;"
# ERROR: permission denied for table audit_logs
psql "postgres://kalengo_app:app_dev_password@localhost:5432/kalengo?sslmode=disable" -c "CREATE TABLE coba (id int);"
# ERROR: permission denied for schema public
```

---

## 5. Partisi & retensi — **rencana**, belum diimplementasikan

§13.4 mewajibkan rencana untuk tabel besar. Ini rencananya. Statusnya belum
dikerjakan; jadwal: **M5** (bersamaan dengan pengerasan performa).

### 5.1 Kapan mulai

Jangan partisi sebelum perlu — partisi menambah kompleksitas operasional.
Picu konversi bila **salah satu** terpenuhi:

| Tabel | Baris | Ukuran |
|---|---|---|
| `notifications` | > 20 juta | > 5 GB |
| `audit_logs` | > 30 juta | > 8 GB |
| `stock_movements` | > 20 juta | > 4 GB |

Pantau dengan `make audit-db` (bagian 6 mencetak ukuran tabel terbesar).

### 5.2 Skema partisi

`RANGE` per bulan pada `created_at` untuk ketiga tabel.

```sql
-- Contoh untuk notifications (dijalankan sebagai migrasi terpisah)
CREATE TABLE notifications_p (
  LIKE notifications INCLUDING ALL
) PARTITION BY RANGE (created_at);

CREATE TABLE notifications_2026_01
  PARTITION OF notifications_p
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
```

Konversi tanpa henti layanan:

1. Buat tabel berpartisi (`notifications_p`) dengan struktur identik.
2. Buat trigger yang menulis ke **kedua** tabel selama migrasi.
3. Salin data lama per bulan, dalam batch, di luar jam sibuk.
4. `BEGIN; ALTER TABLE notifications RENAME TO notifications_old;
   ALTER TABLE notifications_p RENAME TO notifications; COMMIT;` — singkat.
5. Periksa jumlah baris sama, lalu lepas trigger dan hapus tabel lama.

**Catatan untuk `audit_logs`:** rantai hash terurut menurut `id`. Partisi
`RANGE` tidak merusak urutan itu, tetapi `internal/audit.Verify` harus membaca
partisi terurut menaik. Verifikasi lintas bulan tetap mungkin selama `id`
monoton.

### 5.3 Pembuatan partisi otomatis

Pilih salah satu:

- `pg_partman` + `pg_cron` — paling sedikit kode, tapi menambah dependensi.
- Job Asynq bulanan (`cmd/worker`) yang menjalankan DDL `CREATE TABLE ...
  PARTITION OF ...` untuk 3 bulan ke depan — tanpa dependensi baru, dan
  kegagalannya terlihat di dashboard worker.

**Rekomendasi: opsi kedua.** Sudah ada worker, dan DDL-nya trivially
idempoten (`CREATE TABLE IF NOT EXISTS ... PARTITION OF`).

### 5.4 Retensi

| Tabel | Simpan di database | Setelah itu |
|---|---|---|
| `notifications` | 90 hari | Hapus partisi (`DROP TABLE`) — nilai bisnisnya habis |
| `stock_movements` | 24 bulan | Arsip ke MinIO (Parquet) lalu hapus partisi |
| `audit_logs` | 60 bulan | Arsip ke MinIO dengan checksum + simpan manifest hash chain |

Prosedur arsip (bukan `DELETE`):

```sql
-- 1. Ekspor partisi ke Parquet (dijalankan di luar transaksi aplikasi)
COPY (SELECT * FROM audit_logs_2024_01) TO PROGRAM 'aws s3 cp - s3://kalengo-archive/audit/2024-01.parquet' WITH (FORMAT csv, HEADER);
-- 2. Setelah unduhan diverifikasi (jumlah baris + hash rentang), lepas partisi
ALTER TABLE audit_logs DETACH PARTITION audit_logs_2024_01;
DROP TABLE audit_logs_2024_01;
```

**Jangan** memakai `DELETE` massal pada tabel berpartisi — itu membuang manfaat
partisi dan menggandakan beban WAL.

### 5.5 Yang harus tetap bekerja setelah partisi

- [ ] `internal/audit.Verify` lulus pada rentang lintas bulan
- [ ] `notifications` tetap bisa di-query per `user_id` (`read_at IS NULL`)
      tanpa membaca semua partisi — indeks harus dibuat di setiap partisi
- [ ] `audit-db` tetap melaporkan ukuran per partisi, bukan hanya induknya
- [ ] Migrasi `up`/`down` masih idempoten (uji `make migrate-redo`)

---

## 6. Konvensi migrasi

- Nama berkas: `NNNNNN_nama_singkat.{up,down}.sql` (seq, bukan timestamp).
- Satu migrasi = satu perubahan yang bisa dijelaskan dalam satu kalimat.
- `down` wajib ada dan wajib benar-benar membalik `up`. Uji dengan
  `make migrate-redo` sebelum commit.
- **Jangan pernah mengedit migrasi yang sudah dirilis.** Buat migrasi baru.
- Perubahan yang mengunci tabel besar harus dipisah: `ALTER TABLE ... ADD COLUMN`
  (nullable, tanpa default volatile) dulu, isi data, baru pasang constraint
  `NOT VALID` lalu `VALIDATE CONSTRAINT`.
- Setiap migrasi baru harus memperbarui `db/queries/` dan menjalankan
  `make sqlc` bila query tersentuh.

---

## 7. Memeriksa sendiri

```bash
make psql            # masuk sebagai migrator
make psql-app        # masuk sebagai kalengo_app (uji hak akses)

# ringkasan tabel & ukuran
docker compose exec postgres psql -U kalengo_migrator -d kalengo -c '\dt+'

# constraint pada satu tabel
docker compose exec postgres psql -U kalengo_migrator -d kalengo -c '\d+ orders'

# analisis query (§13.3)
make audit-db
```

Setelah beban k6 dijalankan, tempelkan keluaran `make audit-db` ke
[`AUDIT-REPORT.md`](AUDIT-REPORT.md).
