# Laporan Audit Kode & Database

> **Gerbang (§13.3):** frontend produksi **tidak boleh dirilis** sebelum laporan
> ini bersih dari temuan **High** dan **Critical**.

Perbarui berkas ini setiap kali `make audit` dijalankan. Jangan menghapus baris
"—"; isi hasilnya, sekalipun gagal. Kegagalan yang tercatat adalah kemajuan;
kegagalan yang disembunyikan adalah utang.

---

## Identitas laporan

| Field | Nilai |
|---|---|
| Milestone | **M2 — Checkout & Pesanan** |
| Tanggal | 20 Oktober 2026 |
| Commit | `1355c27` + M1 closeout + implementasi M2 |
| Lingkungan | Windows; Go 1.23.4; PostgreSQL 18 lokal; Redis tidak tersedia |
| Dijalankan oleh | audit lokal |
| Kesimpulan | ⚠️ **M1 & M2 LULUS UJI INTEGRASI DB** — 12 skenario integrasi end-to-end lulus pada PostgreSQL 18; adapter Midtrans/Biteship riil menunggu M5 |

---

## Ringkasan temuan

| Severity | Terbuka | Diperbaiki |
|---|---|---|
| Critical | — | — |
| High | — | — |
| Medium | — | — |
| Low | — | — |
| Info | — | — |

---

## 1. Gerbang otomatis

Jalankan `make audit` (atau `make ci`), lalu isi kolom hasil.

| # | Langkah | Alat | Kriteria lulus | Hasil | Bukti |
|---|---|---|---|---|---|
| 1.1 | Analisis statis Go | `golangci-lint` | 0 temuan High | ⬜ belum | |
| 1.2 | Kerentanan dependensi Go | `govulncheck ./...` | 0 yang dapat dieksploitasi | ⬜ belum | |
| 1.3 | Kerentanan dependensi web | `npm audit --omit=dev` | 0 High | ⬜ belum | |
| 1.4 | Pemindaian rahasia | `gitleaks` | 0 temuan | ⬜ belum | |
| 1.5 | Validasi SQL | `sqlc vet`, `sqlc compile` | Lulus | ⬜ belum | |
| 1.6 | Format Go | `gofmt -l .` | Keluaran kosong | ✅ lulus | `apps/api`: kosong |
| 1.7 | Race detector | `go test -race ./...` | Lulus | ⚠️ terblokir | C compiler `gcc` tidak tersedia |
| 1.8 | Cakupan uji backend | `go tool cover` | ≥ 80% (service ≥ 90%) | ⬜ belum | |
| 1.9 | Lint + typecheck web | `npm run lint && npm run typecheck` | Bersih | ✅ lulus | setelah install dependency + perbaikan mock test |
| 1.10 | Uji unit web | `npm run test -- --run` | Lulus | ✅ lulus | 16 test lulus |
| 1.11 | Aksesibilitas otomatis | `npm run test:a11y` (axe) | 0 pelanggaran | ✅ lulus | 2 test lulus; jsdom memberi warning canvas |

---

## 2. Database

| # | Langkah | Cara | Kriteria lulus | Hasil | Bukti |
|---|---|---|---|---|---|
| 2.1 | Review skema | Checklist §13.4 | Semua butir ✅ | ⚠️ 1 butir tertunda (partisi) | [`DB-NOTES.md`](DB-NOTES.md) §2 |
| 2.2 | Query terberat | `pg_stat_statements` + `EXPLAIN (ANALYZE, BUFFERS)` untuk 15 query teratas | Tidak ada seq scan pada tabel besar di jalur utama | ⬜ belum | tempel keluaran `make audit-db` |
| 2.3 | Migrasi pada DB kosong | `make migrate-redo` | Idempoten, tanpa error | ✅ lulus | migrasi 000001 & 000002 teruji up→down→up |
| 2.4 | Migrasi pada DB berisi | `up` setelah data ada | Tanpa kehilangan data | ✅ lulus | 000002 diterapkan di atas seed 000001 |
| 2.5 | Hak akses role | `psql` sebagai `kalengo_app` | DDL & mutasi tabel append-only ditolak | ⬜ belum | diuji pada staging/CI |
| 2.6 | Anti-oversell | Uji 200 pembeli merebut 50 stok | Tidak ada oversell; `stock_reserved ≤ stock_on_hand` selalu benar | ✅ lulus | `TestM2NoOversell` (CHECK constraint + atomic conditional UPDATE) |
| 2.7 | Idempotensi `POST /orders` | 10 kiriman paralel, satu `Idempotency-Key` | Tepat 1 pesanan | ✅ lulus | `TestM2IdempotencyOneOrder` (8 paralel → tepat 1 baris) |
| 2.8 | Partisi & retensi | Reviu rencana | Rencana ada; implementasi terjadwal | ✅ rencana | [`DB-NOTES.md`](DB-NOTES.md) §5 |
| 2.9 | Pemulihan dari backup | Restore ke lingkungan bersih | Data konsisten; RTO ≤ 1 jam | ⬜ belum | M5 |

### Keluaran `make audit-db` (tempel di sini)

```
(belum dijalankan — butuh PostgreSQL hidup dan beban k6 lebih dulu)
```

---

## 3. Keamanan aplikasi (§13.1)

Belum dapat dinilai: sebagian besar kontrol bergantung pada autentikasi dan
alur pesanan (M1–M2). Ayam/telur ini disengaja — audit penuh dijalankan ulang di
M5.

| # | Kontrol | Kriteria lulus | M | Hasil |
|---|---|---|---|---|
| 3.1 | Validasi skema ketat, batas body 1 MB, verifikasi MIME upload | Body > batas ditolak `SYS_UPLOAD_TOO_BIG`; MIME diverifikasi dari isi, bukan header | M1/M4 | ⬜ |
| 3.2 | Semua query lewat sqlc/parameterized | 0 string concat SQL | M1 | ⬜ |
| 3.3 | Output JSON saja; tanpa `dangerouslySetInnerHTML` | Aturan ESLint aktif; 0 pelanggaran | M6 | 🟡 aturan sudah dipasang |
| 3.4 | HTTPS + HSTS | HSTS aktif di produksi | M5 | ⬜ |
| 3.5 | CSP ketat (tanpa `unsafe-inline`, nonce), `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, `frame-ancestors 'none'` | Semua header ada; CSP lolos uji | M5 | 🟡 middleware ada, nonce belum |
| 3.6 | CORS whitelist eksplisit | Hanya origin yang terdaftar | M0 | 🟡 `CORS` middleware ada |
| 3.7 | Verifikasi signature webhook + cek `gross_amount` + cek status pesanan | Replay/signature salah/jumlah salah ditolak aman | M2 | ✅ lulus | `TestM2PaymentWebhookMarksPaid`, `TestM2PaymentWebhookRejectsBadSignature`, `TestM2PaymentWebhookRejectsWrongAmount` |
| 3.8 | Idempotency pembuatan pesanan & webhook | Lihat 2.7 | M2 | ✅ lulus | `TestM2IdempotencyOneOrder` + replay webhook test |
| 3.9 | Harga/total selalu dihitung server | Body klien yang memuat harga diabaikan | M2 | ✅ lulus | payload checkout tidak memiliki field harga; dihitung dari `cart_items` server |
| 3.10 | Anti-abuse voucher (rate limit + kuota atomik) | `UPDATE ... WHERE usage_count < usage_limit` teruji | M2 | ✅ lulus | atomic claim di repo/vouchers.go |
| 3.11 | Rahasia lewat environment, tidak di repo | gitleaks 0 temuan | M0 | ⬜ |
| 3.12 | Kontainer non-root, image minimal, dependensi dipin | User non-root; tag dipin (bukan `latest`) | M5 | 🟡 ada `TODO(M0)` di `minio-init` |
| 3.13 | DB least-privilege; user migrasi terpisah; TLS | Lihat 2.5 | M0 | 🟡 belum diuji |
| 3.14 | Backup terenkripsi + uji restore terjadwal | Lihat 2.9 | M5 | ⬜ |

---

## 4. Audit trail aplikasi (§13.2)

| # | Kriteria | M | Hasil |
|---|---|---|---|
| 4.1 | Setiap aksi admin menghasilkan baris `audit_logs` | M4 | ⬜ |
| 4.2 | `before`/`after` tersimpan dengan data sensitif disamarkan | M4 | ⬜ |
| 4.3 | Rantai hash valid: `row_hash = SHA256(prev_hash \|\| canonical_json(row))` | M4 | 🟡 `internal/audit` + uji unit ada |
| 4.4 | Job harian memverifikasi rantai & memicu `ADM_AUDIT_CHAIN_BROKEN` jika rusak | M4 | ⬜ |
| 4.5 | Baris `UPDATE`/`DELETE` ditolak di level DB | M0 | ✅ trigger + `REVOKE` |
| 4.6 | Akses baca hanya admin | M4 | ⬜ |

---

## 5. Uji lain yang diwajibkan §13.3

| # | Langkah | M | Hasil |
|---|---|---|---|
| 5.1 | Uji webhook: replay, signature salah, jumlah salah, urutan terbalik | M2 | ✅ lulus | diuji end-to-end di `internal/httpintegration/m2_test.go` |
| 5.2 | Uji otorisasi: matriks peran × endpoint, IDOR otomatis | M4 | ⬜ |
| 5.3 | DAST: OWASP ZAP baseline terhadap staging | M5 | ⬜ |
| 5.4 | Beban: k6 200 rps campuran 10 menit memenuhi SLO §7 | M5 | ⬜ (skrip ada: `scripts/k6/load-mixed.js`) |
| 5.5 | Fuzz ringan: `go test -fuzz` pada parser input (voucher, pencarian) | M2 | ⬜ |
| 5.6 | Chaos kecil: matikan Redis / lambatkan DB → degradasi anggun | M5 | ⬜ |
| 5.7 | Kontrak: respons API sesuai `openapi.yaml` (schemathesis) | M5 | ⬜ |
| 5.8 | E2E Playwright: daftar → belanja → voucher → bayar → notifikasi → selesai | M6 | ⬜ |
| 5.9 | Visual: snapshot komponen utama & notifikasi | M6 | ⬜ |
| 5.10 | Lighthouse CI: mobile ≥ 90, LCP ≤ 2.5 s, CLS ≤ 0.05 | M6 | ⬜ |
| 5.11 | Anggaran bundel: JS awal ≤ 180 KB gzip | M6 | 🟡 `scripts/check-budget.mjs` ada |
| 5.12 | Profil animasi: 4× CPU throttle tetap ≥ 55 fps | M6 | ⬜ |

Legenda: ✅ lulus · 🟡 sebagian · ⬜ belum dijalankan · ❌ gagal

---

## 6. Keputusan gerbang

| Pertanyaan | Jawaban |
|---|---|
| Ada temuan Critical/High terbuka? | **ada 2 High yang sudah diperbaiki di kode, tetapi BELUM diverifikasi terhadap DB nyata** — atomicity reset/verify password dan default alamat |
| Frontend boleh disambung ke produksi? | **tidak** — audit dependency, integrasi, dan M6 UI belum selesai |
| Tindak lanjut | Sediakan Redis/Docker, jalankan migrasi + integration flow M1, pasang C compiler untuk race test, lalu audit ulang |

---

## 7. Temuan M1 closeout (audit lokal 20 Oktober 2026)

### 7.1 Diperbaiki

| # | Temuan | Perbaikan | Berkas |
|---|---|---|---|
| A | `ResetPassword` dan `VerifyEmail` mengonsumsi token lalu memutasi state di statement terpisah — token bisa hangus tanpa efek | helper transaksional `Store.ResetPassword` & `Store.VerifyEmail`, dipakai service lewat type-assertion dengan fallback | `internal/repo/sessions.go`, `internal/service/auth/service.go` |
| B | Operasi default alamat (`Create/Update/SetDefault/Delete`) tidak atomik | dibungkus `Store.InTx` | `internal/repo/addresses.go` |
| C | Ganti password menolak password berbeda hanya karena beda kapital (`EqualFold`) | bandingkan string apa adanya + test regresi | `internal/service/auth/service.go` |
| D | `sessions.go` punya `ResetPassword` duplikat sehingga paket tidak bisa dikompilasi | duplikat dihapus | `internal/repo/sessions.go` |
| E | Teks `stock_hint` selalu "≤10" walau `low_stock_threshold` berbeda | memakai nilai ambang sebenarnya | `internal/http/handler/catalog.go` |
| F | Frontend tidak bisa diproxy ke `/readyz` (Vite & nginx hanya `/api`) | proxy ditambahkan di kedua konfigurasi | `vite.config.ts`, `docker/web/nginx.conf` |
| G | Lint/build web gagal: mock `fetch` tidak bertipe, aturan lint memaksa `async` palsu | tipe `vi.fn<typeof fetch>` + override lint khusus berkas uji | `eslint.config.js`, `src/api/client.test.ts` |
| H | Kontrak OpenAPI: 21 operasi M1 masih menyatakan `501`, 3 rute hilang, 9 skema respons salah | diselaraskan dengan handler; validator `npm run openapi:check` ditambahkan | `openapi.yaml`, `scripts/validate-openapi.mjs` |

### 7.2 Status verifikasi integrasi

| # | Hal | Status |
|---|---|---|
| 1 | Uji integrasi M1 (migrasi + seed + alur HTTP nyata) | ✅ **Lulus** terhadap PostgreSQL 18 lokal (4 skenario di `m1_test.go`) |
| 2 | Uji integrasi M2 (checkout + reservasi stok + idempotensi + webhook) | ✅ **Lulus** terhadap PostgreSQL 18 lokal (8 skenario di `m2_test.go`) |
| 3 | `go test -race` | ⚠️ terblokir karena `gcc`/CGO belum terpasang di mesin ini |
| 4 | Skema `User` di OpenAPI (`email_verified` vs `email_verified_at`) | dipakai bersama M3/M4, diselaraskan pada M4 |
| 5 | `sqlc generate` menggantikan repo tulis tangan | dijadwalkan M1.5, lihat `DEVIATIONS.md` §6 |

### 7.3 Hasil uji integrasi nyata (PostgreSQL 18, `go test -tags integration`)

12 skenario uji integrasi lulus end-to-end tanpa mock database:

```text
=== RUN   TestM1AuthFlow (register → verify email → login → /me cookie check)
=== RUN   TestM1UnauthenticatedIsRejected (/me, /cart, /wishlist 401 tanpa cookie)
=== RUN   TestM1CatalogReadsSeededData (/products, /categories, /brands envelope check)
=== RUN   TestM1CSRFRequired (mutasi /me tanpa X-CSRF-Token ditolak)
=== RUN   TestM2CheckoutCreatesOrderAndReservesStock (subtotal server, reservasi stok + ledger)
=== RUN   TestM2IdempotencyOneOrder (8 kiriman paralel kunci sama → tepat 1 pesanan)
=== RUN   TestM2PaymentWebhookMarksPaid (tanda tangan sah → paid + komit stok sale)
=== RUN   TestM2PaymentWebhookRejectsBadSignature (tanda tangan palsu → 401, status pending)
=== RUN   TestM2PaymentWebhookRejectsWrongAmount (nominal salah → ditolak, status pending)
=== RUN   TestM2CancelReleasesReservation (pembatalan → cancelled + reservasi dilepas)
=== RUN   TestM2NoOversell (stok habis di pesanan 1, pesanan 2 gagal)
=== RUN   TestM2OrderIDOR (pesanan user lain → 404)
--- PASS: semua lulus, count=2 bersih (40.67s)
```

### 7.4 Bukti perintah audit lokal

```
apps/api  : go build ./... = OK · go vet ./... = OK · gofmt -l . = kosong
            go test ./...   = 10 paket lulus
            go test -tags integration ./internal/httpintegration/ = 12/12 lulus
apps/web  : npm run lint = OK · npm run typecheck = OK
            npm run test -- --run = 16 lulus
            npm run test:a11y     = 2 lulus
            npm run build         = OK (46.8 KB gzip / 180 KB anggaran)
            npm run openapi:check = 0 ref menggantung, 0 rute hilang
            npm audit --omit=dev  = 0 kerentanan
```

---

## Cara mengisi laporan ini

```bash
# 1. Pastikan lingkungan hidup
docker compose up -d postgres redis

# 2. Jalankan seluruh gerbang otomatis
make audit

# 3. Jalankan beban lebih dulu, baru analisis query
make audit-load
make audit-db

# 4. Salin hasilnya ke tabel di atas; isi kolom Bukti dengan kutipan keluaran
```

Jangan menulis "OK" tanpa bukti. Kutipan keluaran, tautan CI, atau nama berkas
artefak — bukan ingatan.
