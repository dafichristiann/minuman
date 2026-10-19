-- ============================================================
-- Query autentikasi & akun (§12.1) — dipakai mulai M1.
--
-- Aturan §13.1: SETIAP query yang mengembalikan data milik pengguna WAJIB
-- memfilter `user_id = $1` untuk mencegah IDOR/BOLA. Jangan hapus filter itu.
-- ============================================================

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1
  AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name, phone, birth_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: MarkEmailVerified :execrows
UPDATE users
SET email_verified_at = now()
WHERE id = $1
  AND email_verified_at IS NULL;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2,
    failed_login_count = 0,
    locked_until = NULL
WHERE id = $1;

-- name: RecordLoginSuccess :exec
UPDATE users
SET failed_login_count = 0,
    locked_until = NULL,
    last_login_at = now()
WHERE id = $1;

-- name: RecordLoginFailure :one
-- Menambah penghitung gagal dan mengunci akun bila melewati ambang (§13.1).
UPDATE users
SET failed_login_count = failed_login_count + 1,
    locked_until = CASE
      WHEN failed_login_count + 1 >= sqlc.arg('threshold')::int
        THEN now() + sqlc.arg('lockout')::interval
      ELSE locked_until
    END
WHERE id = $1
RETURNING failed_login_count, locked_until;

-- name: UpdateUserProfile :exec
UPDATE users
SET full_name  = $2,
    phone      = $3,
    birth_date = $4
WHERE id = $1
  AND deleted_at IS NULL;

-- name: AnonymizeUser :exec
-- Hapus akun sesuai UU PDP (§6.7): data pribadi diganti, baris tetap ada agar
-- riwayat pesanan & audit tidak rusak.
UPDATE users
SET email              = concat('deleted+', id, '@invalid.kalengo'),
    password_hash      = 'deleted',
    full_name          = 'Pengguna Dihapus',
    phone              = NULL,
    birth_date         = NULL,
    email_verified_at  = NULL,
    deleted_at         = now()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListUsersForAdmin :many
SELECT id, email, full_name, role, email_verified_at, last_login_at, created_at, deleted_at
FROM users
WHERE (sqlc.narg('role')::user_role IS NULL OR role = sqlc.narg('role')::user_role)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
