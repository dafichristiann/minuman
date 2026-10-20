// Package auth menangani pendaftaran, sesi, verifikasi email, dan reset
// password (§12.1). Seluruh kode error berasal dari registry apperr (§8.6).
package auth

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/repo"
	"github.com/kalengo/api/internal/security"
	"github.com/kalengo/api/internal/valid"
)

// Masa berlaku token sekali pakai.
const (
	verifyEmailTTL   = 24 * time.Hour
	resetPasswordTTL = 60 * time.Minute
)

// Users adalah irisan repo.Store yang dipakai layanan ini.
type Users interface {
	CreateUser(ctx context.Context, p repo.CreateUserParams) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	MarkEmailVerified(ctx context.Context, id uuid.UUID) (int64, error)
	UpdateUserPassword(ctx context.Context, id uuid.UUID, hash string) error
	RecordLoginSuccess(ctx context.Context, id uuid.UUID) error
	RecordLoginFailure(ctx context.Context, id uuid.UUID, threshold int32, lockout time.Duration) (int32, *time.Time, error)
}

// Sessions adalah irisan repo.Store untuk token sesi & token sekali pakai.
type Sessions interface {
	CreateRefreshToken(ctx context.Context, p repo.CreateRefreshTokenParams) (uuid.UUID, error)
	GetRefreshTokenByHash(ctx context.Context, hash []byte) (repo.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldID, newID uuid.UUID) (bool, error)
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	CreateOneTimeToken(ctx context.Context, p repo.CreateOneTimeTokenParams) error
	ConsumeOneTimeToken(ctx context.Context, purpose string, hash []byte) (uuid.UUID, error)
}

type atomicPasswordUpdater interface {
	UpdatePasswordAndRevokeAll(context.Context, uuid.UUID, string) error
}

type atomicPasswordResetter interface {
	ResetPassword(context.Context, []byte, string) error
}

// atomicEmailVerifier menandai email terverifikasi dalam transaksi yang sama
// dengan konsumsi token, sehingga tautan tidak hangus tanpa efek.
type atomicEmailVerifier interface {
	VerifyEmail(context.Context, []byte) error
}

// Mailer mengirim email transaksional. Implementasi nyata (SMTP + templat)
// masuk M3; pada M1 LogMailer mencatat kejadiannya agar alur tetap teruji tanpa
// mengirim email sungguhan.
type Mailer interface {
	SendVerifyEmail(ctx context.Context, to, name, token string) error
	SendResetPassword(ctx context.Context, to, name, token string) error
}

// LogMailer hanya mencatat bahwa email seharusnya dikirim. HANYA untuk
// pengembangan/staging — nilai token TIDAK dicatat ke log (§13.1).
type LogMailer struct{ Log *slog.Logger }

// SendVerifyEmail mencatat kejadian kirim email verifikasi.
func (m LogMailer) SendVerifyEmail(_ context.Context, to, _ string, token string) error {
	m.Log.Warn("mailer belum aktif (M3): email verifikasi tidak dikirim",
		"to", to, "purpose", repo.PurposeVerifyEmail, "token_length", len(token))
	return nil
}

// SendResetPassword mencatat kejadian kirim email reset.
func (m LogMailer) SendResetPassword(_ context.Context, to, _ string, token string) error {
	m.Log.Warn("mailer belum aktif (M3): email reset tidak dikirim",
		"to", to, "purpose", repo.PurposeResetPassword, "token_length", len(token))
	return nil
}

// Service adalah layanan autentikasi.
type Service struct {
	cfg      *config.Config
	users    Users
	sessions Sessions
	tokens   *security.TokenService
	mailer   Mailer
	argon    security.Argon2Params
	now      func() time.Time

	// exposeDevTokens hanya true saat pengembangan: body respons ikut memuat
	// token sekali pakai supaya alur registrasi bisa diuji tanpa SMTP (M3).
	exposeDevTokens bool
}

// Options mengonfigurasi Service.
type Options struct {
	Config   *config.Config
	Users    Users
	Sessions Sessions
	Mailer   Mailer
	// ExposeDevTokens hanya boleh true bila APP_ENV != production.
	ExposeDevTokens bool
}

// New membuat Service.
func New(opts Options) *Service {
	return &Service{
		cfg:      opts.Config,
		users:    opts.Users,
		sessions: opts.Sessions,
		tokens:   security.NewTokenService(opts.Config.Auth.JWTSecret, opts.Config.App.Name, opts.Config.Auth.AccessTokenTTL),
		mailer:   opts.Mailer,
		argon: security.Argon2Params{
			Memory:      opts.Config.Argon2.MemoryKiB,
			Time:        opts.Config.Argon2.Time,
			Parallelism: opts.Config.Argon2.Parallelism,
			SaltLength:  opts.Config.Argon2.SaltLength,
			KeyLength:   opts.Config.Argon2.KeyLength,
		},
		now:             time.Now,
		exposeDevTokens: opts.ExposeDevTokens,
	}
}

// Tokens membuka TokenService agar middleware HTTP memakai kunci yang sama.
func (s *Service) Tokens() *security.TokenService { return s.tokens }

// SetClock mengganti sumber waktu (dipakai pengujian lockout & kedaluwarsa).
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// Session adalah hasil autentikasi yang siap dipasang sebagai cookie.
type Session struct {
	User             domain.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	CSRFToken        string
}

// RegisterInput adalah payload pendaftaran.
type RegisterInput struct {
	Email     string
	Password  string
	FullName  string
	Phone     string
	BirthDate string
	UserAgent string
	IP        string
}

// RegisterResult memuat pengguna baru dan (di dev) token verifikasi.
type RegisterResult struct {
	User           domain.User
	DevVerifyToken string
}

// Register membuat akun baru.
//
// Sesi TIDAK langsung dibuat: pengguna harus memverifikasi email lebih dulu,
// tetapi tetap boleh login dan berbelanja (pembatasan ada di checkout, M2).
func (s *Service) Register(ctx context.Context, in RegisterInput) (RegisterResult, error) {
	email, err := valid.Email(in.Email)
	if err != nil {
		return RegisterResult{}, err
	}
	fullName, err := valid.FullName(in.FullName)
	if err != nil {
		return RegisterResult{}, err
	}
	if err := security.ValidatePassword(in.Password); err != nil {
		return RegisterResult{}, passwordError(err)
	}
	phone, err := valid.Phone(in.Phone)
	if err != nil {
		return RegisterResult{}, err
	}
	birthDate, err := valid.BirthDate(in.BirthDate, s.now())
	if err != nil {
		return RegisterResult{}, err
	}

	hash, err := security.HashPassword(in.Password, s.argon)
	if err != nil {
		return RegisterResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	user, err := s.users.CreateUser(ctx, repo.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		FullName:     fullName,
		Phone:        phone,
		BirthDate:    birthDate,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return RegisterResult{}, apperr.New(apperr.CodeAuthEmailTaken, nil)
		}
		return RegisterResult{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	token, err := s.issueOneTimeToken(ctx, user, repo.PurposeVerifyEmail, verifyEmailTTL)
	if err != nil {
		return RegisterResult{}, err
	}
	s.sendVerifyEmail(ctx, user, token)

	res := RegisterResult{User: user}
	if s.exposeDevTokens {
		res.DevVerifyToken = token
	}
	return res, nil
}

// LoginInput adalah payload login.
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        string
}

// Login memverifikasi kredensial lalu menerbitkan sesi baru.
//
// Pesan gagal selalu generik dan waktu respons dijaga konstan dengan tetap
// menjalankan argon2id walau email tidak ditemukan (§13.1).
func (s *Service) Login(ctx context.Context, in LoginInput) (Session, error) {
	email, err := valid.Email(in.Email)
	if err != nil {
		security.DummyVerify(in.Password, s.argon)
		return Session{}, apperr.New(apperr.CodeAuthLoginFail, nil)
	}

	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			security.DummyVerify(in.Password, s.argon)
			return Session{}, apperr.New(apperr.CodeAuthLoginFail, nil)
		}
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	now := s.now().UTC()
	if user.IsLocked(now) {
		return Session{}, lockedError(*user.LockedUntil, now)
	}

	ok, err := security.VerifyPassword(user.PasswordHash, in.Password)
	if err != nil {
		// Hash rusak: login tidak boleh lolos, tetapi detailnya tidak dibocorkan.
		slog.ErrorContext(ctx, "hash password tidak bisa diverifikasi", "error", err, "user_id", user.ID)
		return Session{}, apperr.New(apperr.CodeAuthLoginFail, nil)
	}
	if !ok {
		count, lockedUntil, ferr := s.users.RecordLoginFailure(
			ctx, user.ID,
			int32(s.cfg.RateLimit.LoginFailPerWindow),
			s.cfg.RateLimit.LoginLockout,
		)
		if ferr != nil {
			return Session{}, apperr.Wrap(ferr, apperr.CodeAPIServerError, nil)
		}
		if lockedUntil != nil && lockedUntil.After(now) {
			return Session{}, lockedError(*lockedUntil, now)
		}
		slog.InfoContext(ctx, "login gagal", "user_id", user.ID, "attempt", count)
		return Session{}, apperr.New(apperr.CodeAuthLoginFail, nil)
	}

	if err := s.users.RecordLoginSuccess(ctx, user.ID); err != nil {
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	user.FailedLoginCount = 0
	user.LockedUntil = nil

	return s.issueSession(ctx, user, in.UserAgent, in.IP, uuid.New())
}

// RefreshInput adalah payload rotasi sesi.
type RefreshInput struct {
	RefreshToken string
	UserAgent    string
	IP           string
}

// Refresh merotasi refresh token (§13.1).
//
// Token yang sudah dicabut namun dipakai lagi dianggap pencurian: seluruh
// keluarga token dicabut dan klien menerima AUTH_REFRESH_REUSE.
func (s *Service) Refresh(ctx context.Context, in RefreshInput) (Session, error) {
	if in.RefreshToken == "" {
		return Session{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
	}
	hash, err := security.HashOpaqueTokenString(in.RefreshToken)
	if err != nil {
		return Session{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
	}

	stored, err := s.sessions.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return Session{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
		}
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	if stored.RevokedAt != nil {
		// Hanya token yang sudah DIGANTIKAN (replaced_by terisi) yang menandakan
		// reuse. Token yang dicabut tanpa pengganti berasal dari logout / ganti
		// password, dan itu sekadar sesi yang berakhir — bukan pencurian.
		if stored.ReplacedBy == nil {
			return Session{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
		}
		if rerr := s.sessions.RevokeFamily(ctx, stored.FamilyID); rerr != nil {
			slog.ErrorContext(ctx, "gagal mencabut keluarga token saat deteksi reuse",
				"error", rerr, "family_id", stored.FamilyID)
		}
		slog.WarnContext(ctx, "refresh token dipakai ulang", "user_id", stored.UserID, "family_id", stored.FamilyID)
		return Session{}, apperr.New(apperr.CodeAuthRefreshReuse, nil)
	}
	if !s.now().UTC().Before(stored.ExpiresAt) {
		return Session{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
	}

	user, err := s.users.GetUserByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return Session{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
		}
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	return s.rotateSession(ctx, user, in.UserAgent, in.IP, stored.ID, stored.FamilyID)
}

// Logout mencabut seluruh keluarga token milik refresh token yang dikirim.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash, err := security.HashOpaqueTokenString(refreshToken)
	if err != nil {
		return nil // token cacat: tidak ada yang bisa dicabut
	}
	stored, err := s.sessions.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil
		}
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.sessions.RevokeFamily(ctx, stored.FamilyID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// LogoutAll mencabut seluruh sesi pengguna (§12.1).
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.sessions.RevokeAllForUser(ctx, userID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// VerifyEmail memakai token sekali pakai untuk menandai email terverifikasi.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	hash, err := security.HashOpaqueTokenString(token)
	if err != nil {
		return apperr.New(apperr.CodeAuthVerifyExpired, nil)
	}
	if atomic, ok := s.sessions.(atomicEmailVerifier); ok {
		if err := atomic.VerifyEmail(ctx, hash); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return apperr.New(apperr.CodeAuthVerifyExpired, nil)
			}
			return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
		}
		return nil
	}
	userID, err := s.sessions.ConsumeOneTimeToken(ctx, repo.PurposeVerifyEmail, hash)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return apperr.New(apperr.CodeAuthVerifyExpired, nil)
		}
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if _, err := s.users.MarkEmailVerified(ctx, userID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// ResendVerification membuat token verifikasi baru bila email belum terverifikasi.
//
// Pemanggil selalu membalas dengan pesan yang sama agar endpoint ini tidak bisa
// dipakai menebak email mana yang terdaftar.
func (s *Service) ResendVerification(ctx context.Context, email string) (string, error) {
	normalized, err := valid.Email(email)
	if err != nil {
		return "", nil
	}
	user, err := s.users.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", nil
		}
		return "", apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if user.EmailVerified() {
		return "", nil
	}
	token, err := s.issueOneTimeToken(ctx, user, repo.PurposeVerifyEmail, verifyEmailTTL)
	if err != nil {
		return "", err
	}
	s.sendVerifyEmail(ctx, user, token)
	return token, nil
}

// ForgotPassword membuat token reset. Pemanggil selalu membalas 202.
func (s *Service) ForgotPassword(ctx context.Context, email string) (string, error) {
	normalized, err := valid.Email(email)
	if err != nil {
		return "", nil
	}
	user, err := s.users.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", nil
		}
		return "", apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	token, err := s.issueOneTimeToken(ctx, user, repo.PurposeResetPassword, resetPasswordTTL)
	if err != nil {
		return "", err
	}
	if s.mailer != nil {
		if merr := s.mailer.SendResetPassword(ctx, user.Email, user.FullName, token); merr != nil {
			slog.ErrorContext(ctx, "gagal mengirim email reset", "error", merr, "user_id", user.ID)
		}
	}
	return token, nil
}

// ResetPassword menetapkan password baru dan mencabut seluruh sesi lama.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := security.ValidatePassword(newPassword); err != nil {
		return passwordError(err)
	}
	hash, err := security.HashOpaqueTokenString(token)
	if err != nil {
		return apperr.New(apperr.CodeAuthVerifyExpired, nil)
	}
	newHash, err := security.HashPassword(newPassword, s.argon)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if atomic, ok := s.sessions.(atomicPasswordResetter); ok {
		if err := atomic.ResetPassword(ctx, hash, newHash); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return apperr.New(apperr.CodeAuthVerifyExpired, nil)
			}
			return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
		}
		return nil
	}
	userID, err := s.sessions.ConsumeOneTimeToken(ctx, repo.PurposeResetPassword, hash)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return apperr.New(apperr.CodeAuthVerifyExpired, nil)
		}
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.users.UpdateUserPassword(ctx, userID, newHash); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.sessions.RevokeAllForUser(ctx, userID); err != nil {
		return apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return nil
}

// ChangePasswordInput adalah payload ganti password dari halaman akun.
type ChangePasswordInput struct {
	UserID    uuid.UUID
	Current   string
	New       string
	UserAgent string
	IP        string
}

// ChangePassword memverifikasi password lama, menyimpan yang baru, mencabut
// SEMUA sesi, lalu menerbitkan sesi baru untuk perangkat pemanggil (§12.1:
// "ganti password (cabut sesi lain)").
func (s *Service) ChangePassword(ctx context.Context, in ChangePasswordInput) (Session, error) {
	user, err := s.users.GetUserByID(ctx, in.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return Session{}, apperr.New(apperr.CodeAuthSessionExpired, nil)
		}
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	ok, err := security.VerifyPassword(user.PasswordHash, in.Current)
	if err != nil || !ok {
		return Session{}, apperr.New(apperr.CodeAuthLoginFail, nil)
	}
	if err := security.ValidatePassword(in.New); err != nil {
		return Session{}, passwordError(err)
	}
	if in.Current == in.New {
		return Session{}, apperr.New(apperr.CodeAuthPasswordWeak, nil).
			WithField("new_password", "password baru harus berbeda dari yang lama")
	}

	newHash, err := security.HashPassword(in.New, s.argon)
	if err != nil {
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if atomic, ok := s.users.(atomicPasswordUpdater); ok {
		if err := atomic.UpdatePasswordAndRevokeAll(ctx, user.ID, newHash); err != nil {
			return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
		}
	} else if err := s.users.UpdateUserPassword(ctx, user.ID, newHash); err != nil {
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	} else if err := s.sessions.RevokeAllForUser(ctx, user.ID); err != nil {
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	// Perangkat ini langsung mendapat sesi baru supaya pengguna tidak terlempar.
	return s.issueSession(ctx, user, in.UserAgent, in.IP, uuid.New())
}

// issuedSession adalah hasil internal penerbitan sesi. refreshID dipakai untuk
// menautkan token lama ke token pengganti saat rotasi; nilainya tidak pernah
// dikirim ke klien.
type issuedSession struct {
	session   Session
	refreshID uuid.UUID
}

// issueSession menerbitkan sesi baru (login, ganti password).
func (s *Service) issueSession(ctx context.Context, user domain.User, userAgent, ip string, familyID uuid.UUID) (Session, error) {
	issued, err := s.issueSessionInternal(ctx, user, userAgent, ip, familyID)
	if err != nil {
		return Session{}, err
	}
	return issued.session, nil
}

// rotateSession menerbitkan sesi baru lalu mencabut token lama (§13.1).
//
// Bila token lama ternyata sudah dicabut pihak lain, token yang baru dibuat
// dibuang dan SELURUH keluarga dicabut: itu tanda refresh token dicuri.
func (s *Service) rotateSession(ctx context.Context, user domain.User, userAgent, ip string, oldID, familyID uuid.UUID) (Session, error) {
	issued, err := s.issueSessionInternal(ctx, user, userAgent, ip, familyID)
	if err != nil {
		return Session{}, err
	}

	rotated, err := s.sessions.RotateRefreshToken(ctx, oldID, issued.refreshID)
	if err != nil {
		return Session{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if !rotated {
		if derr := s.sessions.DeleteRefreshToken(ctx, issued.refreshID); derr != nil {
			slog.ErrorContext(ctx, "gagal membuang token pengganti", "error", derr)
		}
		if rerr := s.sessions.RevokeFamily(ctx, familyID); rerr != nil {
			slog.ErrorContext(ctx, "gagal mencabut keluarga token", "error", rerr, "family_id", familyID)
		}
		slog.WarnContext(ctx, "rotasi refresh token kalah balapan", "user_id", user.ID, "family_id", familyID)
		return Session{}, apperr.New(apperr.CodeAuthRefreshReuse, nil)
	}
	return issued.session, nil
}

// issueSessionInternal membuat access token, refresh token, dan token CSRF.
//
// `familyID` yang sama dipakai saat rotasi sehingga deteksi reuse tetap bekerja;
// login baru memakai keluarga baru.
func (s *Service) issueSessionInternal(ctx context.Context, user domain.User, userAgent, ip string, familyID uuid.UUID) (issuedSession, error) {
	access, accessExp, err := s.tokens.IssueAccess(user.ID, user.Role)
	if err != nil {
		return issuedSession{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	refreshPlain, refreshHash, err := security.NewOpaqueToken()
	if err != nil {
		return issuedSession{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	refreshExp := s.now().UTC().Add(s.cfg.Auth.RefreshTokenTTL)

	newID, err := s.sessions.CreateRefreshToken(ctx, repo.CreateRefreshTokenParams{
		UserID:    user.ID,
		FamilyID:  familyID,
		TokenHash: refreshHash,
		UserAgent: userAgent,
		IP:        ip,
		ExpiresAt: refreshExp,
	})
	if err != nil {
		return issuedSession{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	csrf, err := security.NewCSRFToken()
	if err != nil {
		return issuedSession{}, apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}

	return issuedSession{
		refreshID: newID,
		session: Session{
			User:             user,
			AccessToken:      access,
			AccessExpiresAt:  accessExp,
			RefreshToken:     refreshPlain,
			RefreshExpiresAt: refreshExp,
			CSRFToken:        csrf,
		},
	}, nil
}

// issueOneTimeToken membuat token sekali pakai dan mengembalikan bentuk
// plaintext-nya (hanya versi hash yang disimpan).
func (s *Service) issueOneTimeToken(ctx context.Context, user domain.User, purpose string, ttl time.Duration) (string, error) {
	plain, hash, err := security.NewOpaqueToken()
	if err != nil {
		return "", apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	if err := s.sessions.CreateOneTimeToken(ctx, repo.CreateOneTimeTokenParams{
		UserID:    user.ID,
		Purpose:   purpose,
		TokenHash: hash,
		ExpiresAt: s.now().UTC().Add(ttl),
	}); err != nil {
		return "", apperr.Wrap(err, apperr.CodeAPIServerError, nil)
	}
	return plain, nil
}

// sendVerifyEmail mengirim email verifikasi bila mailer tersedia. Kegagalan
// pengiriman TIDAK menggagalkan pendaftaran: akun sudah ada dan pengguna bisa
// meminta kirim ulang.
func (s *Service) sendVerifyEmail(ctx context.Context, user domain.User, token string) {
	if s.mailer == nil {
		return
	}
	if err := s.mailer.SendVerifyEmail(ctx, user.Email, user.FullName, token); err != nil {
		slog.ErrorContext(ctx, "gagal mengirim email verifikasi", "error", err, "user_id", user.ID)
	}
}

func lockedError(until, now time.Time) *apperr.Error {
	minutes := int(until.Sub(now).Minutes()) + 1
	if minutes < 1 {
		minutes = 1
	}
	return apperr.New(apperr.CodeAuthLoginLocked, map[string]string{"menit": strconv.Itoa(minutes)})
}

func passwordError(err error) *apperr.Error {
	switch {
	case errors.Is(err, security.ErrPasswordTooShort):
		return apperr.New(apperr.CodeAuthPasswordWeak, nil).
			WithField("password", "minimal "+strconv.Itoa(security.MinPasswordLength)+" karakter")
	case errors.Is(err, security.ErrPasswordTooLong):
		return apperr.New(apperr.CodeAuthPasswordWeak, nil).
			WithField("password", "maksimal "+strconv.Itoa(security.MaxPasswordBytes)+" byte")
	default:
		return apperr.New(apperr.CodeAuthPasswordWeak, nil)
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
