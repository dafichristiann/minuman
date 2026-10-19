// Package domain memuat tipe data inti yang dipakai lintas lapisan.
//
// Aturan (§10.3): tipe di sini tidak tahu SQL, HTTP, maupun JSON. Bentuk JSON
// publik didefinisikan di handler, bukan di sini, supaya perubahan kontrak API
// tidak memaksa perubahan skema database.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// User adalah baris tabel users tanpa data yang tidak perlu dikirim ke klien.
//
// PasswordHash HANYA dipakai service auth; handler tidak boleh menyentuhnya.
type User struct {
	ID               uuid.UUID
	Email            string
	PasswordHash     string
	FullName         string
	Phone            *string
	Role             string
	EmailVerifiedAt  *time.Time
	BirthDate        *time.Time
	FailedLoginCount int32
	LockedUntil      *time.Time
	LastLoginAt      *time.Time
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// EmailVerified melaporkan status verifikasi email.
func (u User) EmailVerified() bool { return u.EmailVerifiedAt != nil }

// IsLocked melaporkan apakah akun sedang terkunci pada waktu `now`.
func (u User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && u.LockedUntil.After(now)
}

// Address adalah alamat pengiriman milik pengguna (§12.1, §11.2).
type Address struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Label      string
	Recipient  string
	Phone      string
	Province   string
	City       string
	District   string
	PostalCode string
	Street     string
	Notes      *string
	AreaID     *string
	IsDefault  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Field membedakan "tidak dikirim" dari "dikirim null" pada PATCH (§12.1).
// Sama seperti httpx.Optional, tetapi berada di lapisan domain agar service
// tidak perlu mengimpor paket HTTP.
type Field[T any] struct {
	Set   bool
	Value *T
}
