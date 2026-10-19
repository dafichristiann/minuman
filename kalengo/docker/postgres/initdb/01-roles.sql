-- ============================================================
-- KALENGO — bootstrap role & database (§13.1 "Peran DB least-privilege")
-- Dijalankan otomatis oleh image postgres saat volume data masih kosong.
--
-- Dua role, sesuai §13.1:
--   kalengo_migrator : PEMILIK skema, punya hak DDL. Dipakai Makefile / CI saja.
--   kalengo_app      : runtime API & worker. Hanya DML. TIDAK punya hak DDL.
--
-- CATATAN PENTING soal sintaks:
--   psql TIDAK melakukan interpolasi variabel (`:'nama'`) di dalam string
--   dollar-quoted `$$ ... $$`. Karena itu blok DO tidak dipakai di sini;
--   kondisional ditangani dengan `SELECT ... WHERE NOT EXISTS` + `\gexec`.
--
-- Verifikasi manual setelah container sehat:
--   docker exec -it kalengo-postgres psql -U kalengo_app -d kalengo -c "CREATE TABLE x(i int);"
--   -> harus GAGAL dengan "permission denied for schema public"
-- ============================================================

\set ON_ERROR_STOP on

\getenv db_name           KALENGO_DB
\getenv app_user          KALENGO_APP_USER
\getenv app_password      KALENGO_APP_PASSWORD
\getenv migrator_user     KALENGO_MIGRATOR_USER
\getenv migrator_password KALENGO_MIGRATOR_PASSWORD

\echo '[initdb] menyiapkan role kalengo_migrator & kalengo_app'

-- ---------- 1. Role ----------
-- Buat bila belum ada. `\gexec` menjalankan teks SQL yang dihasilkan SELECT.
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'migrator_user', :'migrator_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'migrator_user')
\gexec

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'app_user', :'app_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user')
\gexec

-- Samakan kata sandi bila role sudah ada (mis. volume lama).
ALTER ROLE :"migrator_user" WITH LOGIN PASSWORD :'migrator_password';
ALTER ROLE :"app_user"      WITH LOGIN PASSWORD :'app_password';

-- Batasi resource: role aplikasi tidak boleh menghabiskan koneksi server.
ALTER ROLE :"app_user" CONNECTION LIMIT 60;
ALTER ROLE :"app_user" SET statement_timeout = '15s';
ALTER ROLE :"app_user" SET idle_in_transaction_session_timeout = '30s';
ALTER ROLE :"app_user" SET search_path = public;

-- ---------- 2. Database ----------
SELECT format(
         'CREATE DATABASE %I OWNER %I ENCODING ''UTF8'' TEMPLATE template0',
         :'db_name', :'migrator_user'
       )
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'db_name')
\gexec

-- ---------- 3. Hak di level database ----------
REVOKE ALL ON DATABASE :"db_name" FROM PUBLIC;
GRANT CONNECT, TEMPORARY ON DATABASE :"db_name" TO :"app_user";
GRANT ALL ON DATABASE :"db_name" TO :"migrator_user";

-- ---------- 4. Hak di dalam database kalengo ----------
\connect :db_name

-- Schema public dimiliki migrator; PUBLIC tidak boleh membuat objek (PG15+ default).
ALTER SCHEMA public OWNER TO :"migrator_user";
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO :"app_user";
GRANT ALL ON SCHEMA public TO :"migrator_user";

-- Objek BARU yang dibuat migrator otomatis memberi hak DML ke role aplikasi.
-- (karena itu migrasi HARUS dijalankan sebagai kalengo_migrator)
ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_user" IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO :"app_user";
ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_user" IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO :"app_user";
ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_user" IN SCHEMA public
  GRANT EXECUTE ON FUNCTIONS TO :"app_user";

-- ---------- 5. Ekstensi pra-syarat (§11.1) ----------
-- pgcrypto/pg_trgm/citext adalah "trusted extension", jadi pemilik DB boleh
-- membuatnya. pg_stat_statements TIDAK trusted, sehingga dibuat di sini oleh
-- superuser (skrip initdb berjalan sebagai POSTGRES_USER).
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Hak eksekusi fungsi ekstensi untuk role aplikasi (mis. gen_random_uuid()).
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO :"app_user";

\echo '[initdb] role kalengo_migrator & kalengo_app siap; database kalengo dibuat oleh kalengo_migrator.'
