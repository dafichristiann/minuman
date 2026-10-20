# Catatan M1 — Backend Inti

Cakupan M1 (PRD §17): **auth, akun, alamat, katalog, pencarian, keranjang,
wishlist**. Gerbang kelulusan: *uji unit + integrasi lulus*.

Dokumen ini menjelaskan **apa yang dibangun**, **kontrak yang harus dipatuhi
klien**, dan **apa yang sengaja belum dikerjakan**.

---

## 1. Peta kode

| Lapisan | Berkas | Isi |
|---|---|---|
| Validasi | `internal/valid/valid.go` | email, nama, telepon (+62), tanggal lahir, kode pos, `EscapeLike` |
| Keamanan | `internal/security/password.go` | argon2id PHC, `DummyVerify` anti-enumerasi |
| Keamanan | `internal/security/token.go` | HS256 access token, token opaque + hash SHA-256, CSRF |
| Data | `internal/repo/*.go` | seluruh query M1 (parametrik, tanpa rakit string) |
| Domain | `internal/domain/{user,catalog,cart}.go` | tipe netral (tanpa SQL/HTTP/JSON) |
| Layanan | `internal/service/{auth,account,catalog,shop}` | aturan bisnis |
| HTTP | `internal/http/handler/{session,auth,me,address,catalog,cart}.go` | handler + DTO |
| HTTP | `internal/http/middleware/auth.go` | `RequireAuth`, `OptionalAuth`, `CSRF`, `RequireRole` |
| Rute | `internal/http/router.go` | pemasangan seluruh rute M1 |

Alur ketergantungan tetap satu arah: `handler → service → repo → Postgres`.
Tidak ada handler yang menyentuh `pgx` secara langsung.

---

## 2. Rute M1 yang sudah hidup

Semua di bawah `/api/v1`.

### Auth

| Method | Path | Batas laju | Catatan |
|---|---|---|---|
| POST | `/auth/register` | 5/jam/IP | tidak membuat sesi; email verifikasi dikirim |
| POST | `/auth/login` | 10/menit/IP | lockout bertahap via DB |
| POST | `/auth/refresh` | 5/jam/IP | rotasi + deteksi reuse |
| POST | `/auth/verify-email` | 5/jam/IP | token sekali pakai |
| POST | `/auth/verify-email/resend` | 5/jam/IP | respons selalu sama |
| POST | `/auth/forgot-password` | 5/jam/IP | **selalu** `202` |
| POST | `/auth/reset-password` | 5/jam/IP | mencabut semua sesi |
| POST | `/auth/logout` | — | butuh cookie refresh + CSRF |
| POST | `/auth/logout-all` | — | butuh access token + CSRF |

### Akun

`GET|PATCH|DELETE /me`, `POST /me/password`,
`GET|POST /me/addresses`, `PATCH|DELETE /me/addresses/{id}`,
`POST /me/addresses/{id}/default`.

### Katalog (publik, tanpa login)

`GET /products`, `GET /products/suggest`, `GET /products/{slug}`,
`GET /products/{slug}/reviews`, `GET /categories`, `GET /brands`.

> **Catatan rute chi:** detail produk dan ulasan memakai satu nama parameter
> (`{product}`) karena chi hanya mengizinkan satu nama parameter per posisi
> segmen. Handler detail memperlakukannya sebagai slug; handler ulasan
> mem-parse-nya sebagai UUID. Nama parameter di `openapi.yaml` tetap
> `{slug}`/`{id}` agar kontrak publik tidak berubah.

### Keranjang, wishlist, restock

`GET|DELETE /cart`, `PUT|DELETE /cart/items/{variantId}`,
`POST /cart/merge`, `POST /cart/validate`,
`GET /wishlist`, `GET|PUT|DELETE /wishlist/{productId}`,
`POST|DELETE /stock-alerts/{variantId}`.

Semuanya wajib login + CSRF.

---

## 3. Kontrak yang WAJIB dipatuhi klien

### 3.1 Satu permintaan refresh pada satu waktu (single-flight)

Deteksi reuse bersifat **ketat dan tanpa tenggang waktu**. Bila dua permintaan
`/auth/refresh` berjalan bersamaan dengan cookie yang sama, yang kalah balapan
dianggap pencurian: seluruh keluarga token dicabut dan pengguna terlempar ke
halaman login.

Klien **wajib** membungkus refresh dalam satu promise bersama:

```ts
let inflight: Promise<void> | null = null;
export function refreshSession() {
  inflight ??= doRefresh().finally(() => { inflight = null; });
  return inflight;
}
```

### 3.2 Cookie & CSRF

| Cookie | HttpOnly | Path | Isi |
|---|---|---|---|
| `kalengo_at` | ya | `/` | access token |
| `kalengo_rt` | ya | `/api/v1/auth` | refresh token |
| `kalengo_csrf` | **tidak** | `/` | token CSRF |

Setiap permintaan **mutasi** (`POST`, `PUT`, `PATCH`, `DELETE`) ke area
terautentikasi harus menyertakan header `X-CSRF-Token` yang nilainya sama
dengan cookie `kalengo_csrf` (pola double-submit). Metode aman (`GET`, `HEAD`,
`OPTIONS`) tidak diperiksa.

`kalengo_rt` sengaja dibatasi ke `/api/v1/auth` sehingga token refresh tidak
ikut terkirim ke endpoint lain.

### 3.3 Status respons yang membawa makna

| Kode | Arti |
|---|---|
| `204` | berhasil, tanpa body (`DELETE /cart`, `PUT /wishlist/{id}`, dll) — toast dibuat klien, mis. `WISH_ADDED` |
| `202` | diterima, hasil tidak diungkap (`/auth/forgot-password`) |
| `200` | body JSON ada (keranjang selalu dikembalikan penuh setelah perubahan) |

### 3.4 Kode error = kode notifikasi

`error.code` pada respons §12 **sama persis** dengan kode di katalog §8.6.
Klien memetakan kode → notifikasi; klien tidak perlu membaca `error.message`.

Detail penting:

- `CART_QTY_MAX` membawa `{n}` (batas efektif = `min(max_per_order, stok)`) dan
  `{produk}` di dalam `error.message`; klien memformat ulang dengan teksnya
  sendiri.
- Daftar masalah keranjang ada di `issues[]` (bukan `error`), masing-masing
  berisi `code`, `variant_id`, `added_price`, `unit_price`, `available`,
  `requested`.
- `AUTH_REFRESH_REUSE` **bukan** sesi kedaluwarsa biasa: klien harus
  menampilkan peringatan keamanan dan mengarahkan ke login.

### 3.5 Token verifikasi di lingkungan pengembangan

Selama `APP_ENV != production`, respons `register`, `verify-email/resend`, dan
`forgot-password` memuat token sekali pakai (`dev_verify_token` /
`dev_reset_token`) supaya alur email bisa diuji tanpa SMTP. **Fitur ini mati di
produksi** dan akan digantikan mailer sungguhan di M3.

### 3.6 Kursor paginasi

`GET /products` memakai `?cursor=` yang isinya adalah base64url dari
`{"s","k","i"}`. Aturan yang perlu diketahui klien:

- kursor **tidak** boleh dirakit sendiri — hanya boleh dipakai ulang apa adanya;
- kursor terikat pada `sort`; mengirim kursor `price_asc` bersama
  `sort=newest` menghasilkan `API_VALIDATION`;
- `has_more` + `next_cursor` adalah satu-satunya cara tahu masih ada halaman.

### 3.7 Stok publik dibulatkan

PRD §12.2 melarang membocorkan angka stok persis. Detail produk mengembalikan
`stock_status` (`in_stock` \| `low_stock` \| `out_of_stock`) dan `stock_hint`
(mis. `"Tersisa ≤10"`), bukan kuantitas. Angka persis hanya muncul di panel
admin (M4) dan di validasi keranjang (`available`, dipakai untuk membatasi
kuantitas).

---

## 4. Aturan bisnis yang sudah dikunci di M1

| Aturan | Perilaku |
|---|---|
| Password | minimal 10 karakter, argon2id (`ARGON2_*`), tidak boleh sama dengan password lama saat ganti |
| Registrasi | membuat akun belum terverifikasi; **belum** ada sesi |
| Verifikasi email | tidak menghalangi login/belanja; gerbangnya dipasang di checkout (M2) |
| Login gagal | dihitung di database; setelah `LOGIN_FAIL_PER_WINDOW` akun terkunci `LOGIN_LOCKOUT` |
| Rotasi refresh | token lama dicabut + ditautkan ke pengganti (`replaced_by`) |
| Reuse | hanya bila token punya `replaced_by`; token yang dicabut karena logout/ganti password menjawab `AUTH_SESSION_EXPIRED` |
| Ganti/reset password | mencabut **semua** sesi; perangkat pemanggil `/me/password` langsung dapat sesi baru |
| Keranjang | harga & stok selalu dari server; `PUT` menetapkan kuantitas absolut (1–99) |
| `POST /cart/merge` | melewati item tak tersedia, **memotong** kelebihan kuantitas (bukan menolak), maksimal 100 baris |
| Wishlist | idempoten; produk tak dikenal → `API_NOT_FOUND` |
| Restock | langganan hanya dicatat bila stok habis; bila tersedia, `204` tanpa langganan |

---

## 5. Uji

```powershell
cd apps\api
go mod tidy          # wajib sekali; go.sum belum di-commit
gofmt -l .           # harus kosong
go build ./...
go vet ./...
go test ./...
```

Uji M1 yang ditambahkan:

| Berkas | Fokus |
|---|---|
| `internal/security/password_test.go` | hash/verifikasi, salt unik, PHC rusak ditolak, batas panjang |
| `internal/security/token_test.go` | terbit/parse access token, kedaluwarsa, tanda tangan diubah, `alg:none`, issuer lain, token opaque, CSRF |
| `internal/platform/httpx/optional_test.go` | PATCH bedakan "tidak dikirim" vs `null` |
| `internal/valid/valid_test.go` | normalisasi telepon, email, tanggal lahir, `EscapeLike` |
| `internal/service/auth/service_test.go` | registrasi ganda, lockout, rotasi, **reuse membunuh keluarga**, no-enumeration, reset & ganti password |
| `internal/service/catalog/service_test.go` | bolak-balik kursor untuk 5 sort, kursor salah sort, batas halaman, ringkasan varian |
| `internal/service/shop/service_test.go` | batas kuantitas efektif, item tidak aktif, penggabungan yang memotong, langganan restock |

Uji layanan memakai **store palsu in-memory**, jadi tidak butuh PostgreSQL —
cocok untuk CI. Uji integrasi terhadap Postgres (migrasi + seed + alur HTTP
nyata) menjadi bagian penutup M1.

---

## 6. Yang sengaja belum dikerjakan

| Hal | Milestone | Catatan |
|---|---|---|
| Pengiriman email sungguhan | M3 | sekarang `LogMailer` (tidak mencatat token ke log) |
| `GET /me/ui-settings`, preferensi notifikasi, push | M3 | rute sudah disiapkan di `openapi.yaml` |
| Checkout, pesanan, ongkir, voucher, Midtrans | M2 | `internal/platform/payment` masih kosong |
| Panel admin, audit log API, verifikasi hash chain | M4 | skema & trigger sudah ada sejak M0 |
| `sqlc generate` | M1.5 | lihat [`DEVIATIONS.md`](DEVIATIONS.md) §6 |
| Panel notifikasi (frontend) | M3/M6 | katalog ±115 kode |

---

## 7. Daftar periksa penutup M1

- [x] Seluruh rute §12.1–§12.3 terpasang di router
- [x] Setiap query memakai placeholder bernomor (`$1`, `$2`, …)
- [x] `gofmt -l .` kosong
- [x] `go build ./...` lulus
- [x] `go vet ./...` lulus
- [x] `go test ./...` lulus (10 paket, 147 fungsi uji)
- [ ] Uji integrasi terhadap PostgreSQL nyata (migrasi + seed + alur HTTP); tertunda karena Redis dan Docker belum tersedia di mesin audit
- [ ] `openapi.yaml` diperbarui untuk respons yang berubah (204/array)
- [ ] `go.sum` di-commit dan cache modul CI dinyalakan kembali

---

## 8. Bug M0 yang baru tertangkap saat M1

Toolchain Go baru tersedia di akhir M1, jadi seluruh kode M0 baru benar-benar
dikompilasi sekarang. Dua hal ketahuan:

| Berkas | Masalah | Perbaikan |
|---|---|---|
| `internal/audit/chain.go` | `normalize` meloloskan `map[string]any` tanpa menelusuri isinya. Nilai `*string` (mis. `actor_id`) sampai ke penulis JSON kanonik dan memicu *"tipe tidak didukung"* — artinya **setiap baris audit dengan `actor_id` akan gagal di-hash** | `normalize` sekarang rekursif untuk map & slice; penulis JSON punya jalur cadangan yang menormalkan nilai sebelum menulis |
| `internal/http/middleware` (M0) | `SYS_TIMEOUT`/`SYS_NETWORK_OFFLINE` tidak ada di registry backend | sudah diganti `API_TIMEOUT`/`NET_OFFLINE` sebelum M1 |

Keduanya ditemukan karena uji dijalankan, bukan karena membaca ulang kode —
argumen kuat untuk menjalankan `go test ./...` di setiap milestone, bukan hanya
di akhir.
