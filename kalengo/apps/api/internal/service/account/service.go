// Package account menangani profil pengguna dan buku alamat (§12.1).
package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/repo"
	"github.com/kalengo/api/internal/valid"
)

// DeleteConfirmation adalah kata yang harus diketik untuk menghapus akun
// (§8.6 ACC_DELETE_CONFIRM).
const DeleteConfirmation = "HAPUS"

// Users adalah irisan repo.Store yang dibutuhkan.
type Users interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	UpdateUserProfile(ctx context.Context, id uuid.UUID, p repo.UpdateUserProfileParams) (domain.User, error)
	AnonymizeUser(ctx context.Context, id uuid.UUID) error
}

// Addresses adalah irisan repo.Store untuk alamat.
type Addresses interface {
	ListAddresses(ctx context.Context, userID uuid.UUID) ([]domain.Address, error)
	CreateAddress(ctx context.Context, userID uuid.UUID, in repo.AddressInput) (domain.Address, error)
	UpdateAddress(ctx context.Context, userID, id uuid.UUID, in repo.AddressInput) (domain.Address, error)
	SetDefaultAddress(ctx context.Context, userID, id uuid.UUID) (domain.Address, error)
	DeleteAddress(ctx context.Context, userID, id uuid.UUID) error
}

// Sessions dipakai saat penghapusan akun: semua sesi harus mati.
type Sessions interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// Service adalah layanan akun.
type Service struct {
	users     Users
	addresses Addresses
	sessions  Sessions
}

// New membuat Service.
func New(users Users, addresses Addresses, sessions Sessions) *Service {
	return &Service{users: users, addresses: addresses, sessions: sessions}
}

// Me mengembalikan profil pengguna yang sedang login.
//
// Bila barisnya sudah tidak ada (mis. akun dihapus di perangkat lain), sesi
// dianggap berakhir — bukan 404 — supaya frontend membersihkan state login.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return domain.User{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
		}
		return domain.User{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return user, nil
}

// ProfilePatch memakai domain.Field agar `null` berbeda dari "tidak dikirim".
type ProfilePatch struct {
	FullName  domain.Field[string]
	Phone     domain.Field[string]
	BirthDate domain.Field[string]
}

// UpdateProfile menggabungkan patch dengan nilai lama lalu menyimpannya.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, patch ProfilePatch) (domain.User, error) {
	current, err := s.Me(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}

	fullName := current.FullName
	if patch.FullName.Set && patch.FullName.Value != nil {
		fullName, err = valid.FullName(*patch.FullName.Value)
		if err != nil {
			return domain.User{}, err
		}
	}

	phone := current.Phone
	if patch.Phone.Set {
		raw := ""
		if patch.Phone.Value != nil {
			raw = *patch.Phone.Value
		}
		phone, err = valid.Phone(raw)
		if err != nil {
			return domain.User{}, err
		}
	}

	birthDate := current.BirthDate
	if patch.BirthDate.Set {
		raw := ""
		if patch.BirthDate.Value != nil {
			raw = *patch.BirthDate.Value
		}
		birthDate, err = valid.BirthDate(raw, time.Now().UTC())
		if err != nil {
			return domain.User{}, err
		}
	}

	updated, err := s.users.UpdateUserProfile(ctx, userID, repo.UpdateUserProfileParams{
		FullName:  fullName,
		Phone:     phone,
		BirthDate: birthDate,
	})
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return domain.User{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
		}
		return domain.User{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return updated, nil
}

// DeleteAccount menganonimisasi akun (§6.7) setelah konfirmasi literal.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, confirmation string) error {
	if strings.TrimSpace(confirmation) != DeleteConfirmation {
		return apperr.New(apperr.CodeAccDeleteConfirm, nil).
			WithField("confirmation", "ketik "+DeleteConfirmation+" untuk melanjutkan")
	}
	if err := s.users.AnonymizeUser(ctx, userID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return apperr.New(apperr.CodeAuthSessionExpired, nil)
		}
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.sessions.RevokeAllForUser(ctx, userID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// ---------- buku alamat ----------

// AddressInput adalah payload alamat dari klien (belum divalidasi).
type AddressInput struct {
	Label      string
	Recipient  string
	Phone      string
	Province   string
	City       string
	District   string
	PostalCode string
	Street     string
	Notes      string
	AreaID     string
	IsDefault  bool
}

// ListAddresses mengembalikan alamat aktif milik pengguna.
func (s *Service) ListAddresses(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	items, err := s.addresses.ListAddresses(ctx, userID)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return items, nil
}

// CreateAddress memvalidasi lalu menyimpan alamat baru.
func (s *Service) CreateAddress(ctx context.Context, userID uuid.UUID, in AddressInput) (domain.Address, error) {
	normalized, err := validateAddress(in)
	if err != nil {
		return domain.Address{}, err
	}
	created, err := s.addresses.CreateAddress(ctx, userID, normalized)
	if err != nil {
		return domain.Address{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return created, nil
}

// UpdateAddress menimpa alamat milik pengguna.
func (s *Service) UpdateAddress(ctx context.Context, userID, id uuid.UUID, in AddressInput) (domain.Address, error) {
	normalized, err := validateAddress(in)
	if err != nil {
		return domain.Address{}, err
	}
	updated, err := s.addresses.UpdateAddress(ctx, userID, id, normalized)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return domain.Address{}, apperr.New(apperr.CodeAPINotFound, nil).
				WithField("id", "alamat tidak ditemukan")
		}
		return domain.Address{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return updated, nil
}

// SetDefaultAddress menjadikan satu alamat sebagai default.
func (s *Service) SetDefaultAddress(ctx context.Context, userID, id uuid.UUID) (domain.Address, error) {
	updated, err := s.addresses.SetDefaultAddress(ctx, userID, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return domain.Address{}, apperr.New(apperr.CodeAPINotFound, nil).
				WithField("id", "alamat tidak ditemukan")
		}
		return domain.Address{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return updated, nil
}

// DeleteAddress menghapus (soft delete) satu alamat.
func (s *Service) DeleteAddress(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.addresses.DeleteAddress(ctx, userID, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return apperr.New(apperr.CodeAPINotFound, nil).WithField("id", "alamat tidak ditemukan")
		}
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

func validateAddress(in AddressInput) (repo.AddressInput, error) {
	label, err := valid.Text("label", in.Label, 1, 40)
	if err != nil {
		return repo.AddressInput{}, err
	}
	recipient, err := valid.Text("recipient", in.Recipient, 2, 100)
	if err != nil {
		return repo.AddressInput{}, err
	}
	phone, err := valid.PhoneRequired("phone", in.Phone)
	if err != nil {
		return repo.AddressInput{}, err
	}
	province, err := valid.Text("province", in.Province, 2, 100)
	if err != nil {
		return repo.AddressInput{}, err
	}
	city, err := valid.Text("city", in.City, 2, 100)
	if err != nil {
		return repo.AddressInput{}, err
	}
	district, err := valid.Text("district", in.District, 2, 100)
	if err != nil {
		return repo.AddressInput{}, err
	}
	postal, err := valid.PostalCode(in.PostalCode)
	if err != nil {
		return repo.AddressInput{}, err
	}
	street, err := valid.Text("street", in.Street, 5, 200)
	if err != nil {
		return repo.AddressInput{}, err
	}
	notes, err := valid.OptionalText("notes", in.Notes, 300)
	if err != nil {
		return repo.AddressInput{}, err
	}
	areaID, err := valid.OptionalText("area_id", in.AreaID, 64)
	if err != nil {
		return repo.AddressInput{}, err
	}

	return repo.AddressInput{
		Label:      label,
		Recipient:  recipient,
		Phone:      phone,
		Province:   province,
		City:       city,
		District:   district,
		PostalCode: postal,
		Street:     street,
		Notes:      notes,
		AreaID:     areaID,
		IsDefault:  in.IsDefault,
	}, nil
}
