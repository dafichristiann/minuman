#!/bin/sh
# ============================================================
# Entrypoint image `migrate` (§15: "migrasi dijalankan sebelum rilis").
#
# Pemakaian:
#   docker run --rm -e MIGRATE_DATABASE_URL=... kalengo-migrate up
#   docker run --rm -e MIGRATE_DATABASE_URL=... kalengo-migrate down 1
#   docker run --rm -e MIGRATE_DATABASE_URL=... kalengo-migrate version
#
# WAJIB memakai role kalengo_migrator (punya hak DDL). Runtime API memakai
# kalengo_app yang tidak punya hak DDL (§13.1).
# ============================================================
set -eu

if [ -z "${MIGRATE_DATABASE_URL:-}" ]; then
  echo "migrate: MIGRATE_DATABASE_URL wajib diisi (role kalengo_migrator)" >&2
  exit 1
fi

command="${1:-up}"
shift || true

case "$command" in
  up | down | version | force | drop | goto) ;;
  *)
    echo "migrate: perintah '$command' tidak didukung oleh entrypoint ini" >&2
    exit 2
    ;;
esac

echo "migrate: menjalankan '$command' (path=/migrations)"
exec migrate -path /migrations -database "$MIGRATE_DATABASE_URL" "$command" "$@"
