# syntax=docker/dockerfile:1.7
# ============================================================
# KALENGO API & WORKER (Go) — §15
#   dev      : untuk docker compose (mount source, hot-ish reload)
#   build    : kompilasi biner statis
#   runtime  : distroless non-root (satu image, dua entrypoint)
#   migrate  : image berisi biner golang-migrate + berkas migrasi
# ============================================================

# ---------- dev ----------
FROM golang:1.23-alpine AS dev
ENV CGO_ENABLED=0 \
    GOFLAGS=-mod=mod \
    GOPATH=/go
RUN apk add --no-cache git ca-certificates tzdata make bash
WORKDIR /src
EXPOSE 8080
CMD ["go", "run", "./cmd/api"]

# ---------- deps ----------
FROM golang:1.23-alpine AS deps
ENV CGO_ENABLED=0
RUN apk add --no-cache git ca-certificates
WORKDIR /src
# go.sum belum ada sebelum `go mod tidy` pertama: glob dibiarkan opsional.
COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download

# ---------- build ----------
FROM deps AS build
COPY apps/api/ ./
ARG VERSION=dev
ARG COMMIT=unknown
RUN go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/api ./cmd/api \
 && go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/worker ./cmd/worker

# ---------- runtime (distroless, non-root) ----------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api"]

# ---------- migrate (job sekali jalan; butuh hak DDL) ----------
FROM golang:1.23-alpine AS migrate
ENV CGO_ENABLED=0
RUN apk add --no-cache git ca-certificates
WORKDIR /src
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1
COPY apps/api/db/migrations/ /migrations/
COPY apps/api/scripts/migrate-entrypoint.sh /migrate-entrypoint.sh
RUN chmod +x /migrate-entrypoint.sh
ENTRYPOINT ["/migrate-entrypoint.sh"]
