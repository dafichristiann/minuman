-- ============================================================
-- KALENGO — audit query (§13.3)
--
-- Jalankan setelah beban k6 dijalankan, sebagai kalengo_migrator:
--   make audit-db
-- atau:
--   psql "$MIGRATE_DATABASE_URL" -v ON_ERROR_STOP=1 -f apps/api/scripts/audit-queries.sql
--
-- Yang diperiksa:
--   1. Query terberat menurut total waktu (kandidat indeks baru)
--   2. Query dengan rata-rata terlama (kandidat tuning)
--   3. Tabel paling banyak sequential scan (indeks kurang / tidak terpakai)
--   4. Indeks yang tidak pernah dipakai (kandidat hapus, hemat write)
--   5. Kesiapan indeks untuk jalur kritis (pesanan, notifikasi, stok)
--
-- Hasil wajib ditempelkan ke docs/AUDIT-REPORT.md (§13.3).
-- ============================================================

\echo '=== 1. Query terberat (total waktu) ==='
SELECT
  round(total_exec_time::numeric, 2)                        AS total_ms,
  calls,
  round(mean_exec_time::numeric, 2)                         AS mean_ms,
  round(100.0 * total_exec_time / NULLIF(sum(total_exec_time) OVER (), 0), 1) AS pct_total,
  rows,
  left(regexp_replace(query, '\s+', ' ', 'g'), 160)         AS query
FROM pg_stat_statements
WHERE query NOT LIKE '%pg_stat_statements%'
  AND dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
ORDER BY total_exec_time DESC
LIMIT 20;

\echo ''
\echo '=== 2. Rata-rata terlama (minimal 50 panggilan) ==='
SELECT
  round(mean_exec_time::numeric, 2) AS mean_ms,
  calls,
  round(max_exec_time::numeric, 2)  AS max_ms,
  left(regexp_replace(query, '\s+', ' ', 'g'), 160) AS query
FROM pg_stat_statements
WHERE calls >= 50
  AND query NOT LIKE '%pg_stat_statements%'
ORDER BY mean_exec_time DESC
LIMIT 20;

\echo ''
\echo '=== 3. Tabel dengan sequential scan terbanyak ==='
SELECT
  relname                                  AS tabel,
  seq_scan,
  seq_tup_read,
  idx_scan,
  coalesce(round(idx_scan::numeric / NULLIF(seq_scan, 0), 2), 0) AS rasio_idx_seq,
  n_live_tup                               AS baris_hidup
FROM pg_stat_user_tables
WHERE n_live_tup > 0
ORDER BY seq_scan DESC
LIMIT 20;

\echo ''
\echo '=== 4. Indeks yang belum pernah dipakai (kandidat evaluasi) ==='
SELECT
  relname   AS tabel,
  indexrelname AS indeks,
  idx_scan,
  pg_size_pretty(pg_relation_size(indexrelid)) AS ukuran
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND indexrelname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC
LIMIT 20;

\echo ''
\echo '=== 5. Jalur kritis: query yang harus tetap cepat (§1.4 p95 <= 300 ms) ==='
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT id, slug, name
FROM products
WHERE deleted_at IS NULL
  AND is_active
ORDER BY created_at DESC, id DESC
LIMIT 24;

EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT id, status, expires_at
FROM orders
WHERE status = 'pending_payment'
  AND expires_at < now() + interval '1 minute';

EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT id, code, title
FROM notifications
WHERE user_id = gen_random_uuid()
  AND read_at IS NULL
ORDER BY created_at DESC
LIMIT 20;

EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT p.slug, p.name
FROM products p
WHERE p.deleted_at IS NULL
  AND p.is_active
  AND p.name ILIKE 'kopi%'
ORDER BY length(p.name), p.name
LIMIT 8;

\echo ''
\echo '=== 6. Ukuran tabel terbesar (rencana partisi §13.4) ==='
SELECT
  relname AS tabel,
  pg_size_pretty(pg_total_relation_size(relid)) AS total,
  pg_size_pretty(pg_relation_size(relid))       AS tabel_saja,
  pg_size_pretty(pg_total_relation_size(relid) - pg_relation_size(relid)) AS indeks
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC
LIMIT 15;

\echo ''
\echo '=== 7. Kesiapan koneksi (pool API vs batas role) ==='
SELECT
  rolname      AS role,
  rolconnlimit AS batas_koneksi
FROM pg_roles
WHERE rolname IN ('kalengo_app', 'kalengo_migrator');

SELECT
  state,
  count(*) AS jumlah
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY state
ORDER BY jumlah DESC;
