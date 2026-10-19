-- ============================================================
-- Query sistem & health (§ observabilitas)
-- Dijalankan sqlc → internal/repo/sqlcgen
-- ============================================================

-- name: PingDB :one
SELECT 1::int AS ok;

-- name: DatabaseNow :one
SELECT now() AS server_time;

-- name: CountActiveConnections :one
SELECT count(*)::bigint AS active_connections
FROM pg_stat_activity
WHERE datname = current_database()
  AND state <> 'idle';

-- name: LongestRunningQuery :one
SELECT coalesce(max(extract(epoch FROM (now() - query_start)))::bigint, 0) AS longest_seconds
FROM pg_stat_activity
WHERE datname = current_database()
  AND state = 'active'
  AND query_start IS NOT NULL;
