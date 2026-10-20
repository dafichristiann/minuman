# Penyimpangan & Keputusan

Setiap tempat di mana kode berbeda dari [`PRD.md`](PRD.md) — atau di mana PRD
memberi pilihan dan kami memilih — dicatat di sini. **PRD menang** sampai baris
di tabel ini disetujui.

Terakhir diperbarui: M1.

---

## 1. Keputusan atas pertanyaan terbuka PRD §20

PRD §20 sengaja membuka beberapa pertanyaan. Jawaban di bawah dipakai untuk M0
dan **boleh berubah** — bila berubah, hanya konfigurasi dan satu berkas ini yang
disentuh.

| # | Pertanyaan PRD | Keputusan | Dampak bila berubah |
|---|---|---|---|
| 2 | Jenis minuman | Non-alkohol, kecuali satu SKU uji ber-`requires_age_verification` | Hapus alur `AGE_GATE` |
| 3 | Payment gateway | **Midtrans Snap** (PRD sudah menyebut path `/webhooks/midtrans`) | Ganti `PAYMENT_PROVIDER`, tulis adapter baru di `internal/platform/payment` |
| 4 | Penyedia ongkir | **Biteship**; satu gudang | Ganti `SHIPPING_PROVIDER` + adapter `internal/platform/shipping` |
| 5 | Pasar & bahasa | Indonesia saja; IDR integer; Bahasa Indonesia | Bila multi-bahasa: tambah `i18n`, jangan hardcode teks di handler |
| 6 | Model bisnis | B2C satu toko | Bila multi-tenant: tambah `tenant_id` di semua tabel — **mahal**, tanyakan lebih dulu |

Semua nilai ini muncul sebagai variabel di [`.env.example`](../.env.example) dan
divalidasi saat boot oleh `internal/config/config.go`.

---

## 2. Penyimpangan skema (§11.2)

Migrasi `000001_init` menambahkan hal-hal berikut yang tidak eksplisit ditulis di
§11.2. Semuanya **aditif** — tidak ada kolom atau tabel PRD yang dihapus,
diganti nama, atau diubah tipenya.

### 2.1 Kolom audit yang diseragamkan (§11.1)

| Tabel | Ditambahkan | Alasan |
|---|---|---|
| `categories` | `created_at`, `updated_at` | §11.1 mewajibkan konvensi ini di semua tabel |
| `brands` | `created_at`, `updated_at` | idem |
| `product_images` | `created_at`, `updated_at` | idem |
| `vouchers` | `updated_at` | idem |
| `reviews` | `updated_at` | idem |

`updated_at` dijaga oleh trigger `trg_<tabel>_upd` (14 trigger seluruhnya),
bukan oleh aplikasi — supaya tidak bisa dilewati oleh jalur tulis mana pun.

### 2.2 Indeks foreign key

PRD §11.2 mendefinisikan FK tetapi tidak menyebut indeksnya. 48 indeks dibuat,
termasuk indeks untuk **setiap** FK. Tanpa indeks FK, penghapusan/penyuntingan
baris induk memicu sequential scan pada tabel anak.

### 2.3 `notifications` tanpa `updated_at`

Tabel `notifications` bervolume tinggi dan hanya ditulis sekali (append-only),
sehingga tidak diberi `updated_at` maupun trigger. Penandaan "sudah dibaca"
memakai kolom `read_at` — itu perubahan keadaan yang sah dan dicatat, bukan
penyuntingan baris.

### 2.4 Imutabilitas tabel append-only

`audit_logs`, `stock_movements`, dan `order_status_history` diberi trigger
`trg_audit_no_mod` (menolak `UPDATE`/`DELETE`) dan `REVOKE UPDATE, DELETE,
TRUNCATE` dari `kalengo_app` (§11.3, §13.1).

> Blok `REVOKE` dibungkus pengecekan `pg_roles` karena role `kalengo_app` tidak
> selalu ada — mis. di job CI yang memakai user `postgres` polos. Tanpa
> penjagaan itu, migrasi akan gagal di lingkungan tersebut.

### 2.5 `pg_stat_statements` diaktifkan

`docker-compose.yml` memuat `shared_preload_libraries=pg_stat_statements`.
§13.3 mewajibkan analisis query, dan ekstensi itu hanya bisa dimuat saat server
dimulai. Di mesin tanpa ekstensi tersebut, `scripts/audit-queries.sql` bagian 1–2
akan gagal; bagian lain tetap jalan.

---

## 3. Registry kode error vs katalog notifikasi

Ini keputusan arsitektur paling penting di M0. Bacalah sebelum M6.

### 3.1 Dua himpunan yang berbeda

| | Katalog §8.6 | Registry `internal/apperr` |
|---|---|---|
| Isi | **±115** kode, termasuk notifikasi sukses | **56** kode |
| Dipakai oleh | frontend (`notifications.catalog.ts`) | backend: kode di `error.code` |
| Contoh | `CART_ADDED`, `AUTH_LOGIN_OK`, `PAY_SUCCESS` | `AUTH_LOGIN_FAIL`, `STOCK_OUT`, `API_RATE_LIMIT` |

Katalog §8.6 memuat banyak notifikasi yang **tidak pernah** datang dari server
sebagai error: `CART_ADDED` adalah toast yang dibuat klien setelah sukses;
`AUTH_VERIFY_OK` menyala dari tautan di email, bukan dari respons HTTP.

Menaruh semuanya di registry backend akan memaksa server memberi kode error
untuk peristiwa sukses — itu salah secara semantik.

### 3.2 Aturan yang mengikat keduanya

PRD §12 mensyaratkan: `error.code` **sama** dengan kode di katalog frontend.
Aturan itu tetap dipatuhi:

1. Setiap kode di `internal/apperr` **wajib** ada di katalog frontend dengan
   kode identik dan pesan identik.
2. Tidak setiap kode katalog berasal dari backend — yang tidak berasal
   ditandai `origin: "client"` di katalog.
3. `GET /api/v1/meta/error-codes` memublikasikan registry backend sebagai
   JSON, sehingga M6 bisa **memverifikasi sinkronisasi secara otomatis**
   (uji gagal bila ada kode backend yang tidak ada di katalog).

Endpoint #3 sudah hidup sejak M0. Uji verifikasi lintas bahasa ditulis di M6.

### 3.3 Kode tambahan di luar §8.6 — wajib direkonsiliasi di M6

20 kode berikut ditandai `// [TAMBAHAN]` di `internal/apperr/codes.go`. Kode ini
jelas dibutuhkan backend (mis. `ORD_NOT_FOUND`) tetapi tidak ada di §8.6.
**M6 wajib menambahkannya ke katalog frontend**, atau memetakannya ke kode §8.6
yang ada. Jangan menghapusnya tanpa alasan.

| Kode | Status HTTP | Mengapa dibutuhkan |
|---|---|---|
| `AUTH_REFRESH_REUSE` | 401 | Token refresh dipakai ulang = indikasi token dicuri (§13.1) |
| `AUTH_EMAIL_UNVERIFIED` | 403 | §8.6 punya `AUTH_VERIFY_*` tetapi tidak punya penolakan akses ini |
| `AUTH_CSRF_INVALID` | 403 | §13.1 mewajibkan proteksi CSRF; kegagalannya perlu kode |
| `CART_EMPTY` | 409 | Checkout keranjang kosong; §8.6 tidak punya kode ini |
| `VOUCHER_NOT_APPLICABLE` | 409 | Voucher valid tetapi tidak berlaku untuk keranjang ini |
| `CHK_MIN_ORDER` | 409 | Minimal belanja toko; berbeda dari `VOUCHER_MIN_SPEND` |
| `PAY_PROVIDER_ERROR` | 502 | Gateway tidak bisa dihubungi — pembeda penting saat insiden |
| `PAY_ALREADY_PAID` | 409 | Webhook ganda / pembayaran kedua |
| `ORD_NOT_FOUND` | 404 | Wajib untuk setiap resource |
| `ORD_INVALID_TRANSITION` | 409 | §11.4 mendefinisikan mesin status; pelanggarannya butuh kode |
| `ORD_REFUND_NOT_ALLOWED` | 409 | Refund di luar kebijakan |
| `REV_ALREADY_SUBMITTED` | 409 | §8.6 hanya punya `REV_NOT_ELIGIBLE` |
| `API_BAD_REQUEST` | 400 | Body tidak bisa di-parse — sebelum validasi field |
| `API_METHOD_NOT_ALLOWED` | 405 | chi menyediakan hook ini; wajib ada |
| `API_SERVICE_UNAVAILABLE` | 503 | Dependensi mati saat permintaan masuk |
| `API_UNSUPPORTED_MEDIA` | 415 | Content-Type bukan JSON |
| `API_TIMEOUT` | 504 | §1.4 punya anggaran waktu; pelanggarannya perlu kode |
| `API_IDEMPOTENCY_CONFLICT` | 409 | `Idempotency-Key` sama, body berbeda (§12) |
| `PAY_WEBHOOK_SIGNATURE_INVALID` | 401 | Tanda tangan webhook ditolak (§13.1 butir 3.7) |
| `PAY_AMOUNT_MISMATCH` | 409 | Nominal webhook tidak sama dengan total pesanan di database (§13.1 butir 3.7) |
| `ADM_AUDIT_CHAIN_BROKEN` | 500 | Verifikasi hash chain gagal (§11.3) — kritis |
| `ADM_STOCK_REASON_REQ` | 422 | §11.2/§8.6 mewajibkan alasan pada koreksi stok manual |

Yang **tidak** dimasukkan ke registry (keputusan sadar): `ADM_NEW_ORDER`,
`ADM_STOCK_LOW`, `ADM_WEBHOOK_FAIL`, `ADM_SUSPICIOUS`, `ADM_STATUS_CHANGED`,
`ADM_DELETE_CONFIRM`, `ADM_AUDIT_EXPORT`, `SYS_COPIED`, `SYS_UNSAVED_CHANGES`,
`NET_*`, `STOCK_LOW`, `STOCK_LAST`, `SEARCH_EMPTY`, `FILTER_APPLIED`.
Semuanya peristiwa notifikasi, bukan respons error HTTP. `ADM_WEBHOOK_FAIL` dan
`ADM_SUSPICIOUS` berasal dari worker lalu masuk Notification Center — bukan dari
respons API.

---

## 4. Penyimpangan lain

| Hal | PRD | Implementasi M0 | Alasan |
|---|---|---|---|
| Email dev | "mailhog" (§15) | image **Mailpit**, nama service tetap `mailhog` | MailHog tidak lagi dirawat; Mailpit kompatibel SMTP + UI |
| `sqlc` `strict_function_checks` | tidak disebut | dimatikan di M0 | menghalangi `sqlc generate` sebelum ada skema nyata; dinyalakan kembali di M1 |
| **Akses data M1** | §10.3 menyebut `sqlc generate` | **pgx/v5 tulis tangan** di `internal/repo`; `db/queries/*.sql` tetap jadi spesifikasi SQL kanonik | toolchain `sqlc` tidak tersedia di mesin pengembangan M1; lihat §6 |
| `apps/web` isi | React + Vite + TS penuh | **hanya** halaman diagnostik | §4 mewajibkan moodboard sebelum desain; lihat [`MOODBOARD.md`](MOODBOARD.md) |
| `go.sum` | tidak disebut | tidak di-commit | checksum harus dihasilkan toolchain Anda; lihat [`M0-SETUP.md`](M0-SETUP.md) §2 |
| Versi React | tidak disebut | **18.3** | 19 sudah tersedia; naik versi = keputusan M6 agar tidak mengubah perilaku saat fondasi belum stabil |
| `golangci-lint` | tidak dispesifikkan | **v2.1** (format config v2) | `.golangci.yml` memakai skema v2; CI memakai `golangci-lint-action@v8` |

---

## 6. Akses data M1: pgx tulis tangan, bukan keluaran `sqlc`

M1 ditulis tanpa `sqlc` karena binary-nya tidak tersedia di mesin ini. Semua
query M1 berada di `apps/api/internal/repo/*.go` sebagai SQL parametrik
(`$1`, `$2`, …) — **tidak ada** string SQL yang dirakit dari input pengguna.

Yang harus dilakukan saat `sqlc` tersedia (target: M1.5, sebelum M2 menulis
transaksi pesanan):

1. Isi `db/queries/*.sql` mengikuti query yang sudah terbukti di `internal/repo`.
2. Jalankan `sqlc generate`; hasilnya menggantikan berkas repo satu per satu.
3. Jaga `repo.Store` sebagai tipe tunggal — service tidak boleh berubah.
4. Hapus berkas repo manual yang sudah tergantikan, lalu jalankan
   `go test ./...` (uji service memakai store palsu, jadi harus tetap lulus).

Alasan mencatat ini secara eksplisit: kode akses data tulis tangan adalah tempat
paling mungkin munculnya SQL yang tidak terparameterisasi. Setiap penambahan
query wajib melewati `internal/repo` dan memakai placeholder bernomor.

### 6.1 Dua penyesuaian perilaku yang menyertai M1

| Hal | Perilaku | Alasan |
|---|---|---|
| Deteksi reuse refresh token | hanya bila token punya `replaced_by` | token yang dicabut karena logout/ganti password bukan pencurian; klien harus melihat `AUTH_SESSION_EXPIRED`, bukan `AUTH_REFRESH_REUSE` |
| Stok di katalog publik | `stock_status` + `stock_hint` ("Tersisa ≤10"), bukan angka | §12.2 melarang membocorkan kuantitas persis ke publik |

---

## 7. Keputusan & Implementasi M2 (Checkout, Pesanan, Pembayaran)

| # | Hal | Keputusan | Alasan |
|---|---|---|---|
| 7.1 | Penyedia pembayaran M2 | **`payment.Fake`** (dalam proses, tanpa jaringan) | Meniru tanda tangan `SHA512(order+status+cost+key)` Midtrans Snap tanpa ketergantungan API pihak ketiga saat pengujian lokal/CI; adapter jaringan riil masuk M5 |
| 7.2 | Penyedia pengiriman M2 | **`shipping.Fake`** (dalam proses) | Tarif deterministik per kg & per area; adapter Biteship riil masuk M5 |
| 7.3 | Idempotensi `POST /orders` | Kunci unik `(user_id, idempotency_key)` + kolom `request_fingerprint` | Permintaan kedua dengan kunci & isi sama dijawab `200` (pesanan yang sudah ada); bila body berbeda menghasilkan `API_IDEMPOTENCY_CONFLICT` (409) |
| 7.4 | Konvensi tanda delta ledger stok | `reservation`: `+qty`; `release`: `-qty`; `sale`: `-qty` | Konsisten dengan ledger ganda: `stock_reserved = sum(reservation + release)`, `stock_on_hand = sum(restock + adjustment + sale)` |
| 7.5 | Pembangkit `order_no` | Sequence PostgreSQL `order_no_seq` (migrasi 000002) | Format `KLG-YYYYMMDD-NNNN`; bebas balapan tanpa retry di aplikasi |
| 7.6 | Webhook Midtrans idempotensi | `payment_webhook_events` dengan `UNIQUE (provider, event_id)` | Event ganda dijawab `200` tanpa mutasi ulang state |
| 7.7 | Urutan eksekusi checkout TX | Insert `orders` → reserve stok + movement → insert items → commit | Foreign key `stock_movements.order_id → orders(id)` mewajibkan baris pesanan ada lebih dulu di transaksi yang sama |

---

## 8. Cara menambah baris di dokumen ini

Saat Anda menyimpang dari PRD:

1. Tambahkan baris ke tabel yang sesuai — jangan tulis prosa panjang.
2. Sebutkan **apa** yang berbeda, **mengapa**, dan **apa dampaknya**.
3. Bila keputusan itu memengaruhi kontrak API, perbarui juga `openapi.yaml`.
4. Bila memengaruhi kode error, perbarui `internal/apperr/codes.go` **dan**
   tabel §3.3 di atas.

Penyimpangan yang tidak dicatat akan muncul kembali sebagai bug pada M5.
