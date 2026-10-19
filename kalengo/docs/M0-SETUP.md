# M0 — Panduan Setup

Dokumen ini adalah jalan keluar dari kondisi "kode ada, tapi belum pernah
dijalankan". M0 ditulis di mesin yang **tidak** memiliki Go, Docker, maupun
`psql`, sehingga belum ada satu pun baris yang dikompilasi atau dimigrasikan.
Langkah 3 di bawah wajib dijalankan sekali sebelum apa pun bisa dipercaya.

---

## 1. Prasyarat

| Perangkat | Versi minimum | Catatan |
|---|---|---|
| Go | 1.23 | `go.mod` menargetkan `go 1.23` |
| Node.js | 20.11 LTS (disarankan 22) | Vite 5 + Vitest 2 |
| npm | 10 | ikut Node |
| Docker Desktop | 4.30+ | Compose v2 wajib (skrip memakai `docker compose`) |
| `golang-migrate` | 4.18 | atau pakai container (lihat 4b) |
| `sqlc` | 1.27 | hanya bila mengubah query |
| `golangci-lint` | 1.62 | CI memakai versi ini |
| `make` | opsional | Windows: pakai `scripts/bootstrap.ps1` atau Git Bash |

Pasang alat Go sekali saja:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

---

## 2. Satu hal yang harus dipahami lebih dulu: `go.sum` belum ada

`apps/api/go.sum` **sengaja tidak di-commit** karena checksum-nya harus
dihasilkan oleh toolchain Anda sendiri, bukan disalin dari mesin lain.

Akibatnya:

- `go build`, `go test`, dan `golangci-lint` akan **gagal** sampai
  `go mod tidy` dijalankan sekali.
- File `go.mod` sudah memuat seluruh dependensi yang dipakai; `go mod tidy`
  hanya mengisi `go.sum` dan membuang dependensi yang tidak terpakai.
- Setelah `go.sum` terbentuk, **commit file itu**. Sejak saat itu CI
  reproducible dan langkah ini tidak perlu diulang.

```bash
cd apps/api
go mod tidy          # WAJIB, sekali
```

Container `api` dan `worker` di `docker-compose.yml` sudah menjalankan
`go mod tidy` sebelum `go run`, jadi jalur Docker tidak terpengaruh.

### Format

Seluruh file Go ditulis tangan, sehingga perataan struct mungkin belum
gofmt-kanonik. Jalankan sekali:

```bash
cd apps/api
go fmt ./...
```

CI menolak kode yang belum diformat (`gofmt -l .` harus kosong).

---

## 3. Bootstrap otomatis

### Windows

```powershell
cd kalengo
.\scripts\bootstrap.ps1
```

Skrip: memeriksa prasyarat → membuat `.env` → menyalakan infra → menunggu
PostgreSQL → `go mod tidy` → `go fmt` → migrasi → seed → `npm install`.

Mulai ulang dari nol (data hilang):

```powershell
.\scripts\bootstrap.ps1 -Reset
```

### Linux / macOS / Git Bash

```bash
cd kalengo
make bootstrap
```

---

## 4. Bootstrap manual (bila skrip gagal)

### 4a. Infrastruktur

```bash
cp .env.example .env
docker compose up -d postgres redis mailhog minio minio-init
docker compose ps          # tunggu semua "healthy"
```

Role database dibuat otomatis oleh `docker/postgres/initdb/01-roles.sql` saat
volume PostgreSQL **pertama kali** dibuat:

| Role | Hak | Dipakai oleh |
|---|---|---|
| `kalengo_migrator` | pemilik skema, boleh DDL | migrasi, seed, `psql` admin |
| `kalengo_app` | DML saja; `UPDATE`/`DELETE` ditolak di tabel append-only | API & worker |

> Bila volume sudah pernah dibuat sebelum `01-roles.sql` ada, role tidak akan
> dibuat ulang. Jalankan `docker compose down -v` lalu `up` lagi.

### 4b. Migrasi

Dengan `migrate` lokal:

```bash
cd apps/api
migrate -path db/migrations \
  -database "postgres://kalengo_migrator:migrator_dev_password@localhost:5432/kalengo?sslmode=disable" up
```

Tanpa `migrate` lokal (lewat container):

```bash
docker run --rm --network kalengo_default \
  -v "$PWD/apps/api/db/migrations:/migrations:ro" \
  migrate/migrate:v4.18.1 \
  -path=/migrations \
  -database "postgres://kalengo_migrator:migrator_dev_password@postgres:5432/kalengo?sslmode=disable" up
```

Uji idempotensi (wajib sebelum milestone ditutup):

```bash
make migrate-redo
```

### 4c. Seed

```bash
make seed
# atau, dari Windows tanpa make:
Get-Content apps/api/db/seed/dev_seed.sql -Raw |
  docker compose exec -T postgres psql -U kalengo_migrator -d kalengo -v ON_ERROR_STOP=1
```

### 4d. Backend & frontend

```bash
# Terminal 1 — API
cd apps/api && go run ./cmd/api

# Terminal 2 — worker
cd apps/api && go run ./cmd/worker

# Terminal 3 — web
cd apps/web && npm install && npm run dev
```

---

## 5. Daftar verifikasi M0

Jalankan berurutan. Semua harus lulus sebelum M0 dinyatakan selesai.

### Backend

```bash
cd apps/api
go mod tidy
gofmt -l .                      # harus kosong
go vet ./...
go build ./...
go test -race ./...
golangci-lint run --config ../../.golangci.yml ./...
```

- [ ] `gofmt -l .` tidak mencetak apa pun
- [ ] `go build ./...` sukses (tiga biner: `api`, `worker`, `worker-ping`)
- [ ] `go test -race ./...` hijau
- [ ] `golangci-lint` tanpa temuan blocker

### Database

```bash
make migrate-redo
make seed
make audit-db
```

- [ ] `migrate-redo` selesai tanpa error (up → down → up → down → up)
- [ ] `seed` selesai tanpa error, dan aman dijalankan dua kali
- [ ] `audit-queries.sql` berjalan tanpa error (hasil boleh kosong)
- [ ] Tabel `audit_logs`, `stock_movements`, `order_status_history` menolak
      `UPDATE`/`DELETE` sebagai `kalengo_app`

Uji hak akses:

```bash
psql "postgres://kalengo_app:app_dev_password@localhost:5432/kalengo?sslmode=disable" \
  -c "DELETE FROM audit_logs;"     # harus DITOLAK
psql "postgres://kalengo_app:app_dev_password@localhost:5432/kalengo?sslmode=disable" \
  -c "CREATE TABLE coba (id int);" # harus DITOLAK
```

### API hidup

```bash
curl -s localhost:8080/healthz | jq
curl -s localhost:8080/readyz  | jq
curl -s localhost:8080/api/v1/meta | jq
curl -s localhost:8080/api/v1/meta/error-codes | jq '.count'
curl -s localhost:8080/api/v1/tidak-ada | jq '.error.code'   # API_NOT_FOUND
```

- [ ] `/healthz` → 200 `{"status":"alive"}`
- [ ] `/readyz` → 200 `{"status":"ready"}`; matikan Redis → 503 `degraded`
- [ ] `/api/v1/meta` → memuat `app`, `env`, `version`, `payment_provider`
- [ ] `/api/v1/meta/error-codes` → `count` > 0
- [ ] rute tak dikenal → `API_NOT_FOUND`
- [ ] respons error memuat `request_id` yang sama dengan header `X-Request-Id`
- [ ] `POST /api/v1/meta` → `API_METHOD_NOT_ALLOWED`

### Worker

```bash
make worker-ping
docker compose logs --tail=20 worker   # atau terminal `go run ./cmd/worker`
```

- [ ] `worker-ping` mencetak `task terkirim: id=... queue=critical`
- [ ] log worker menampilkan task `system:noop` diproses

### Frontend

```bash
cd apps/web
npm install
npm run lint
npm run typecheck
npm run test -- --run
npm run test:a11y
npm run build
node scripts/check-budget.mjs
```

- [ ] lint & typecheck bersih
- [ ] uji unit hijau
- [ ] uji axe tanpa pelanggaran
- [ ] build sukses; anggaran performa terpenuhi
- [ ] `http://localhost:5173` menampilkan "API terjangkau" dan tabel dependensi

### Keamanan

```bash
make secrets     # gitleaks: 0 temuan
make vuln        # govulncheck
make security    # gosec
```

---

## 6. Hal yang diketahui belum lengkap di M0

Ini bukan bug yang tersembunyi — semuanya disengaja dan tercatat:

| Hal | Alasan | Ditutup di |
|---|---|---|
| Frontend hanya halaman diagnostik | PRD §4 mewajibkan moodboard sebelum desain | M6 |
| Sebagian besar rute API → 501 | Kontrak §12 sudah ditulis, implementasi bertahap | M1–M4 |
| Belum ada `service/` dan `repo/` | Baru dipakai saat handler domain pertama ada | M1 |
| Belum ada kode sqlc tergenerasi | Butuh skema nyata + `sqlc generate` | M1 |
| `audit-concurrency` / `audit-idempotency` | Menunjuk paket `internal/service/...` yang belum ada | M2 |
| Notifikasi frontend | Menunggu API notifikasi | M3/M6 |
| Angka p95 di k6 | Endpoint domain masih 501 | M2 |

---

## 7. Pemecahan masalah

| Gejala | Sebab | Solusi |
|---|---|---|
| `missing go.sum entry` | `go mod tidy` belum dijalankan | `cd apps/api && go mod tidy` |
| `role "kalengo_app" does not exist` | Volume dibuat sebelum `initdb` ada | `docker compose down -v && docker compose up -d` |
| `readyz` → `postgres: down` | Container belum sehat / kredensial salah | `docker compose ps`, cek `DATABASE_URL` |
| `readyz` → `redis: down` | Password Redis tidak ikut di URL | `REDIS_URL` harus `redis://:redis_dev_password@redis:6379/0` |
| `migrate: Dirty database version` | Migrasi terhenti di tengah | Perbaiki penyebabnya, lalu `migrate force <versi-sebelumnya>` |
| Vite tidak bisa memanggil API | Proxy target salah | Dev lokal: `VITE_PROXY_TARGET=http://localhost:8080`; Docker: `http://api:8080` |
| Lint gagal: `file is not included in any tsconfig` | File baru di luar `src/` | Tambahkan ke `tsconfig.node.json` |
| `npm ci` gagal di CI | `package-lock.json` belum di-commit | Jalankan `npm install` lalu commit lockfile |

---

## 8. Setelah M0

M1 memulai backend sesuai urutan §17:

1. Tabel & query autentikasi (sudah ada di skema v1)
2. `internal/repo` (sqlc) + `internal/service/auth`
3. Handler `/auth/*`, `/me`, `/addresses` + middleware `RequireAuth`
4. Katalog: `/products`, `/categories`, `/brands` dengan filter & paginasi
5. Keranjang, wishlist, pengingat stok
6. Uji: unit service ≥ 90% cakupan, integrasi repo terhadap PostgreSQL nyata

Jangan menyentuh frontend sampai M6.
