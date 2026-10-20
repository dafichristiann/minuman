# KALENGO

Toko daring minuman kaleng — React + Vite + TypeScript, Go, PostgreSQL 16, Redis.

> Sumber kebenaran produk & teknis ada di [`docs/PRD.md`](docs/PRD.md).
> Bila kode dan PRD berbeda, **PRD menang** sampai perbedaan itu dicatat di
> [`docs/DEVIATIONS.md`](docs/DEVIATIONS.md).

---

## Status milestone

| Milestone | Cakupan | Status |
|---|---|---|
| **M0** | Fondasi: monorepo, CI, docker-compose, konfigurasi, logger, skema DB v1, migrasi, kontrak OpenAPI, bangkai rute | **selesai (kode)** — baseline Go sudah diverifikasi; audit infra masih terbuka, lihat [`docs/M0-SETUP.md`](docs/M0-SETUP.md) |
| M1 | Autentikasi, akun, katalog, keranjang, wishlist, pengingat stok | **implementasi selesai** — integrasi PostgreSQL nyata lulus (10 uji integrasi); lihat [`docs/M1-NOTES.md`](docs/M1-NOTES.md) |
| M2 | Checkout, ongkir, voucher, pesanan, pembayaran, webhook | **implementasi selesai** — integrasi DB lulus (anti-oversell, idempotensi, webhook tanda tangan); provider tiruan, adapter jaringan riil di M5 |
| M3 | Notifikasi real-time (SSE), preferensi, push, UI settings | belum |
| M4 | Admin & audit | belum |
| M5 | Pengerasan keamanan, performa, aksesibilitas | belum |
| M6 | Moodboard (§4) → sistem desain → implementasi UI penuh | belum — gerbang §4 belum dijalankan, lihat [`docs/MOODBOARD.md`](docs/MOODBOARD.md) |

---

## Struktur repositori

```
kalengo/
├── apps/
│   ├── api/                     # Go (chi) — handler → service → repo
│   │   ├── cmd/{api,worker,worker-ping}/
│   │   ├── db/{migrations,queries,seed}/
│   │   ├── internal/
│   │   │   ├── apperr/          # registry kode error (sumber kebenaran §8.6)
│   │   │   ├── audit/           # hash chain audit log (§11.3)
│   │   │   ├── config/          # seluruh env, divalidasi saat boot
│   │   │   ├── domain/          # mesin status pesanan, aturan domain
│   │   │   ├── http/            # router, middleware, handler
│   │   │   └── platform/        # db, redis, logger, httpx, ratelimit
│   │   ├── openapi.yaml         # kontrak API (§12)
│   │   └── sqlc.yaml
│   └── web/                     # React + Vite + TS
├── docker/
│   ├── postgres/initdb/         # role least-privilege (§13.1)
│   └── web/nginx.conf
├── docs/
│   ├── PRD.md                   # PRD asli (jangan diedit)
│   ├── MOODBOARD.md             # gerbang §4 — wajib sebelum desain
│   ├── M0-SETUP.md              # cara menjalankan dari nol
│   ├── DEVIATIONS.md            # semua penyimpangan dari PRD
│   ├── DB-NOTES.md              # catatan skema, partisi, retensi
│   └── AUDIT-REPORT.md          # laporan audit §13.3
├── scripts/
│   ├── bootstrap.ps1            # padanan `make bootstrap` untuk Windows
│   └── k6/load-mixed.js         # uji beban §13.3
├── docker-compose.yml
├── Makefile
└── .env.example
```

---

## Mulai cepat

### Windows (PowerShell)

```powershell
cd kalengo
.\scripts\bootstrap.ps1        # infra + migrasi + seed + npm install
go run .\apps\api\cmd\api     # API   → http://localhost:8080/readyz
npm --prefix apps/web run dev # Web   → http://localhost:5173
```

### Linux / macOS / Git Bash

```bash
cd kalengo
make bootstrap
make up          # seluruh stack via Docker
```

### Seluruh stack lewat Docker

```bash
docker compose up -d
# API    http://localhost:8080/readyz
# Web    http://localhost:5173
# Mail   http://localhost:8025   (Mailpit)
# MinIO  http://localhost:9001   (kalengo / minio_dev_password)
```

**Baca [`docs/M0-SETUP.md`](docs/M0-SETUP.md) lebih dulu** — ada satu langkah
wajib (`go mod tidy`) karena `go.sum` sengaja tidak di-commit.

---

## Perintah penting

| Perintah | Arti |
|---|---|
| `make bootstrap` | Siapkan lingkungan dari nol |
| `make up` / `make down` | Nyalakan / matikan seluruh stack |
| `make logs` | Ikuti log api + worker |
| `make test-race` | Uji Go dengan race detector |
| `make migrate-up` / `migrate-down` | Jalankan / batalkan migrasi |
| `make migrate-redo` | Uji up → down → up (idempoten) |
| `make seed` | Isi data contoh |
| `make sqlc` / `make sqlc-vet` | Regenerasi / periksa kode SQL |
| `make audit` | Seluruh gerbang mutu §13.3 yang bisa diotomatiskan |
| `make audit-load` | Uji beban k6 |
| `make web-dev` / `make web-build` | Frontend dev / build produksi |

`make help` menampilkan daftar lengkap.

---

## Aturan yang tidak boleh dilanggar

1. **Harga selalu integer rupiah.** Tidak ada `float` untuk uang, di mana pun.
2. **Stok tidak pernah ditulis langsung.** Semua perubahan lewat `stock_movements`
   agar dapat diaudit.
3. **Setiap respons gagal memakai envelope** `{ "error": { code, message, fields?, request_id? } }`.
   `code` wajib cocok dengan katalog §8.6 dan registry `internal/apperr`.
4. **Setiap mutasi punya umpan balik sukses dan gagal.** Tidak ada POST/PATCH/DELETE
   yang diam-diam berhasil.
5. **Rahasia tidak pernah masuk repo.** `.env` diabaikan; gitleaks berjalan di CI.
6. **Token sesi tidak disimpan di `localStorage`.** Cookie `httpOnly` + CSRF.
7. **Desain menunggu moodboard.** Tidak ada pekerjaan visual sebelum §4
   ([`docs/MOODBOARD.md`](docs/MOODBOARD.md)) selesai dan disetujui.
8. **Gerbang CI merah = tidak merge.** Tidak ada pengecualian manual.

---

## Lisensi

Proprietary — © KALENGO. Seluruh hak dilindungi.
