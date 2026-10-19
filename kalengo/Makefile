# ============================================================
# KALENGO — Makefile (§10.3, §13.3, §15)
#
#   make help          daftar perintah
#   make bootstrap     siapkan lingkungan (infra + migrasi + dependensi)
#   make audit         jalankan seluruh pemeriksaan §13.3 yang bisa diotomatiskan
#
# Pemakaian di Windows: jalankan lewat Git Bash atau WSL.
# Alternatif tanpa make: scripts/bootstrap.ps1 (PowerShell).
# ============================================================

SHELL := /bin/sh

API_DIR   := apps/api
WEB_DIR   := apps/web
MIGRATIONS := $(API_DIR)/db/migrations

# Koneksi lokal: app = least-privilege, migrator = pemilik skema (§13.1)
DATABASE_URL          ?= postgres://kalengo_app:app_dev_password@localhost:5432/kalengo?sslmode=disable
MIGRATE_DATABASE_URL  ?= postgres://kalengo_migrator:migrator_dev_password@localhost:5432/kalengo?sslmode=disable

# Alat eksternal (bisa di-override: make sqlc SQLC=/path/to/sqlc)
GO        ?= go
SQLC      ?= sqlc
MIGRATE   ?= migrate
GOLANGCI  ?= golangci-lint
GITLEAKS  ?= gitleaks
GOVULNCHECK ?= govulncheck
GOSEC     ?= gosec
K6        ?= k6
NPM       ?= npm

VERSION   ?= dev
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.DEFAULT_GOAL := help
.PHONY: help bootstrap infra infra-down up down logs tidy fmt fmt-check lint test test-race cover \
        sqlc sqlc-vet build run worker worker-ping \
        migrate-up migrate-down migrate-redo migrate-version migrate-new migrate-image \
        seed psql psql-app \
        security secrets vuln audit audit-db audit-concurrency audit-idempotency audit-load \
        web-install web-dev web-lint web-test web-build web-budget web-audit ci clean

# ---------------------------------------------------------------
help: ## Tampilkan daftar perintah
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------
# Lingkungan
# ---------------------------------------------------------------
bootstrap: infra migrate-up tidy web-install ## Siapkan lingkungan pengembangan dari nol
	@echo ">> bootstrap selesai. Jalankan: make up"

infra: ## Nyalakan hanya infrastruktur (postgres, redis, mailhog, minio)
	docker compose up -d postgres redis mailhog minio minio-init

infra-down: ## Matikan infrastruktur
	docker compose down

up: ## Nyalakan seluruh stack
	docker compose up -d

down: ## Matikan seluruh stack
	docker compose down

logs: ## Ikuti log api + worker
	docker compose logs -f api worker

# ---------------------------------------------------------------
# Go: kualitas kode
# ---------------------------------------------------------------
tidy: ## go mod tidy (WAJIB dijalankan sekali agar go.sum terbentuk)
	cd $(API_DIR) && $(GO) mod tidy

fmt: ## Rapikan format Go
	cd $(API_DIR) && $(GO) fmt ./...

fmt-check: ## Pastikan format Go sudah rapi (dipakai CI)
	@cd $(API_DIR) && test -z "$$(gofmt -l .)" || (echo ">> file belum diformat:"; gofmt -l .; exit 1)

lint: ## golangci-lint (§13.3)
	cd $(API_DIR) && $(GOLANGCI) run --config ../../.golangci.yml ./...

test: ## Uji unit Go
	cd $(API_DIR) && $(GO) test ./...

test-race: ## Uji Go dengan race detector (§13.3)
	cd $(API_DIR) && $(GO) test -race ./...

cover: ## Uji + laporan cakupan (target §1.4: ≥80%, service ≥90%)
	cd $(API_DIR) && $(GO) test -covermode=atomic -coverprofile=coverage.out ./... \
	  && $(GO) tool cover -func=coverage.out | tail -n 20

sqlc: ## Regenerasi kode sqlc
	cd $(API_DIR) && $(SQLC) generate

sqlc-vet: ## sqlc vet (§13.3)
	cd $(API_DIR) && $(SQLC) vet

build: ## Bangun biner api & worker
	cd $(API_DIR) && $(GO) build -trimpath \
	  -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" \
	  -o bin/api ./cmd/api \
	&& $(GO) build -trimpath \
	  -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" \
	  -o bin/worker ./cmd/worker

run: ## Jalankan API lokal (butuh infra + .env)
	cd $(API_DIR) && $(GO) run ./cmd/api

worker: ## Jalankan worker lokal
	cd $(API_DIR) && $(GO) run ./cmd/worker

worker-ping: ## Kirim task uji ke antrean Asynq
	cd $(API_DIR) && $(GO) run ./cmd/worker-ping

# ---------------------------------------------------------------
# Database & migrasi
# ---------------------------------------------------------------
migrate-up: ## Jalankan migrasi (role migrator)
	$(MIGRATE) -path $(MIGRATIONS) -database "$(MIGRATE_DATABASE_URL)" up

migrate-down: ## Batalkan SEMUA migrasi (hapus data!)
	$(MIGRATE) -path $(MIGRATIONS) -database "$(MIGRATE_DATABASE_URL)" down

migrate-redo: ## Uji up → down → up pada DB kosong (§13.3)
	@echo ">> down"
	$(MIGRATE) -path $(MIGRATIONS) -database "$(MIGRATE_DATABASE_URL)" down || true
	@echo ">> up"
	$(MIGRATE) -path $(MIGRATIONS) -database "$(MIGRATE_DATABASE_URL)" up
	@echo ">> down"
	$(MIGRATE) -path $(MIGRATIONS) -database "$(MIGRATE_DATABASE_URL)" down
	@echo ">> up"
	$(MIGRATE) -path $(MIGRATIONS) -database "$(MIGRATE_DATABASE_URL)" up
	@echo ">> migrasi idempoten"

migrate-version: ## Versi migrasi saat ini
	$(MIGRATE) -path $(MIGRATIONS) -database "$(MIGRATE_DATABASE_URL)" version

migrate-new: ## Buat migrasi baru: make migrate-new NAME=tambah_tabel_x
	@test -n "$(NAME)" || (echo ">> NAME wajib diisi"; exit 1)
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS) -seq $(NAME)

migrate-image: ## Bangun image migrasi (dipakai CI/CD)
	docker build -f $(API_DIR)/Dockerfile --target migrate -t kalengo-migrate:$(VERSION) .

seed: ## Isi data contoh untuk pengembangan
	psql "$(MIGRATE_DATABASE_URL)" -v ON_ERROR_STOP=1 -f $(API_DIR)/db/seed/dev_seed.sql

psql: ## Masuk psql sebagai migrator
	psql "$(MIGRATE_DATABASE_URL)"

psql-app: ## Masuk psql sebagai role aplikasi (uji bahwa DDL ditolak)
	psql "$(DATABASE_URL)"

# ---------------------------------------------------------------
# Keamanan (§13.3)
# ---------------------------------------------------------------
security: secrets vuln ## Pemindaian rahasia + kerentanan dependensi
	cd $(API_DIR) && $(GOSEC) -quiet -exclude-generated ./...

secrets: ## gitleaks (§13.3: 0 temuan)
	$(GITLEAKS) detect --config .gitleaks.toml --no-banner --redact

vuln: ## govulncheck (§13.3)
	cd $(API_DIR) && $(GOVULNCHECK) ./...

audit: fmt-check lint test-race security sqlc-vet migrate-redo web-lint web-budget web-audit ## Audit otomatis §13.3
	@echo ">> audit otomatis selesai. Lengkapi tabel manual di docs/AUDIT-REPORT.md"

audit-db: ## Analisis query terberat (§13.3): pg_stat_statements + EXPLAIN
	psql "$(MIGRATE_DATABASE_URL)" -v ON_ERROR_STOP=1 -f $(API_DIR)/scripts/audit-queries.sql

audit-concurrency: ## Uji konkurensi stok: 200 pembeli, 50 stok (§13.3)
	cd $(API_DIR) && $(GO) test -run TestStockConcurrency -count=1 -v ./internal/service/...

audit-idempotency: ## Uji idempotensi POST /orders (§13.3)
	cd $(API_DIR) && $(GO) test -run TestOrderIdempotency -count=1 -v ./internal/service/...

audit-load: ## Uji beban k6 (§13.3)
	$(K6) run scripts/k6/load-mixed.js

# ---------------------------------------------------------------
# Frontend
# ---------------------------------------------------------------
web-install: ## Pasang dependensi frontend
	cd $(WEB_DIR) && $(NPM) install

web-dev: ## Jalankan Vite dev server
	cd $(WEB_DIR) && $(NPM) run dev

web-lint: ## Lint + typecheck frontend
	cd $(WEB_DIR) && $(NPM) run lint && $(NPM) run typecheck

web-test: ## Uji unit frontend
	cd $(WEB_DIR) && $(NPM) run test -- --run

web-build: ## Build produksi frontend
	cd $(WEB_DIR) && $(NPM) run build

web-budget: web-build ## Periksa anggaran ukuran bundel (§1.4)
	cd $(WEB_DIR) && node scripts/check-budget.mjs

web-audit: ## npm audit + uji a11y (§13.3)
	cd $(WEB_DIR) && $(NPM) audit --omit=dev --audit-level=high
	cd $(WEB_DIR) && $(NPM) run test:a11y

# ---------------------------------------------------------------
ci: fmt-check lint test-race security sqlc-vet web-lint web-test web-budget web-audit ## Rangkaian yang dijalankan CI

clean: ## Bersihkan artefak build
	rm -rf $(API_DIR)/bin $(API_DIR)/coverage.out $(WEB_DIR)/dist
