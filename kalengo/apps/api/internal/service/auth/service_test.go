package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/domain"
	"github.com/kalengo/api/internal/repo"
	"github.com/kalengo/api/internal/security"
)

// ---------------------------------------------------------------------------
// perkakas uji
// ---------------------------------------------------------------------------

func testConfig() *config.Config {
	return &config.Config{
		Env: config.EnvDevelopment,
		App: config.AppConfig{Name: "kalengo"},
		Auth: config.AuthConfig{
			JWTSecret:         "rahasia-uji-yang-cukup-panjang-untuk-hs256",
			AccessTokenTTL:    15 * time.Minute,
			RefreshTokenTTL:   720 * time.Hour,
			CookieName:        "kalengo_at",
			RefreshCookieName: "kalengo_rt",
			CSRFCookieName:    "kalengo_csrf",
			CSRFHeaderName:    "X-CSRF-Token",
		},
		// Parameter argon2id diperkecil agar unit test cepat.
		Argon2: config.Argon2Config{MemoryKiB: 8 * 1024, Time: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32},
		RateLimit: config.RateLimitConfig{
			LoginFailPerWindow: 5,
			LoginFailWindow:    15 * time.Minute,
			LoginLockout:       15 * time.Minute,
		},
	}
}

func fastArgon() security.Argon2Params {
	return security.Argon2Params{Memory: 8 * 1024, Time: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// fakeUsers adalah Users in-memory dengan semantik yang sama seperti repo.Store
// untuk kolom yang dipakai layanan auth.
type fakeUsers struct {
	mu      sync.Mutex
	clock   *testClock
	byEmail map[string]domain.User
	byID    map[uuid.UUID]domain.User
}

func newFakeUsers(clock *testClock) *fakeUsers {
	return &fakeUsers{
		clock:   clock,
		byEmail: make(map[string]domain.User),
		byID:    make(map[uuid.UUID]domain.User),
	}
}

func (f *fakeUsers) CreateUser(_ context.Context, p repo.CreateUserParams) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.byEmail[p.Email]; exists {
		// Meniru UNIQUE(email) PostgreSQL; service memetakannya ke AUTH_EMAIL_TAKEN.
		return domain.User{}, &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}
	}

	now := f.clock.now()
	user := domain.User{
		ID:           uuid.New(),
		Email:        p.Email,
		PasswordHash: p.PasswordHash,
		FullName:     p.FullName,
		Phone:        p.Phone,
		Role:         "customer",
		BirthDate:    p.BirthDate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	f.byEmail[user.Email] = user
	f.byID[user.ID] = user
	return user, nil
}

func (f *fakeUsers) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.byEmail[email]
	if !ok || user.DeletedAt != nil {
		return domain.User{}, repo.ErrNotFound
	}
	return user, nil
}

func (f *fakeUsers) GetUserByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.byID[id]
	if !ok || user.DeletedAt != nil {
		return domain.User{}, repo.ErrNotFound
	}
	return user, nil
}

func (f *fakeUsers) MarkEmailVerified(_ context.Context, id uuid.UUID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.byID[id]
	if !ok {
		return 0, nil
	}
	if user.EmailVerifiedAt != nil {
		return 0, nil
	}
	now := f.clock.now()
	user.EmailVerifiedAt = &now
	f.byID[id] = user
	f.byEmail[user.Email] = user
	return 1, nil
}

func (f *fakeUsers) UpdateUserPassword(_ context.Context, id uuid.UUID, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.byID[id]
	if !ok {
		return repo.ErrNotFound
	}
	user.PasswordHash = hash
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	f.byID[id] = user
	f.byEmail[user.Email] = user
	return nil
}

func (f *fakeUsers) RecordLoginSuccess(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.byID[id]
	if !ok {
		return repo.ErrNotFound
	}
	now := f.clock.now()
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	user.LastLoginAt = &now
	f.byID[id] = user
	f.byEmail[user.Email] = user
	return nil
}

func (f *fakeUsers) RecordLoginFailure(_ context.Context, id uuid.UUID, threshold int32, lockout time.Duration) (int32, *time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	user, ok := f.byID[id]
	if !ok {
		return 0, nil, repo.ErrNotFound
	}
	user.FailedLoginCount++
	if user.FailedLoginCount >= threshold {
		until := f.clock.now().Add(lockout)
		user.LockedUntil = &until
	}
	f.byID[id] = user
	f.byEmail[user.Email] = user
	return user.FailedLoginCount, user.LockedUntil, nil
}

// fakeToken meniru satu baris refresh_tokens.
type fakeToken struct {
	id         uuid.UUID
	userID     uuid.UUID
	familyID   uuid.UUID
	expiresAt  time.Time
	revokedAt  *time.Time
	replacedBy *uuid.UUID
}

type fakeOTT struct {
	userID    uuid.UUID
	purpose   string
	expiresAt time.Time
	used      bool
}

// fakeSessions meniru repo.Store untuk refresh_tokens & one_time_tokens.
type fakeSessions struct {
	mu      sync.Mutex
	clock   *testClock
	byHash  map[string]*fakeToken
	byID    map[uuid.UUID]*fakeToken
	ottHash map[string]*fakeOTT
}

func newFakeSessions(clock *testClock) *fakeSessions {
	return &fakeSessions{
		clock:   clock,
		byHash:  make(map[string]*fakeToken),
		byID:    make(map[uuid.UUID]*fakeToken),
		ottHash: make(map[string]*fakeOTT),
	}
}

func (f *fakeSessions) CreateRefreshToken(_ context.Context, p repo.CreateRefreshTokenParams) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	tok := &fakeToken{
		id:        uuid.New(),
		userID:    p.UserID,
		familyID:  p.FamilyID,
		expiresAt: p.ExpiresAt,
	}
	f.byHash[string(p.TokenHash)] = tok
	f.byID[tok.id] = tok
	return tok.id, nil
}

func (f *fakeSessions) GetRefreshTokenByHash(_ context.Context, hash []byte) (repo.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	tok, ok := f.byHash[string(hash)]
	if !ok {
		return repo.RefreshToken{}, repo.ErrNotFound
	}
	return repo.RefreshToken{
		ID:         tok.id,
		UserID:     tok.userID,
		FamilyID:   tok.familyID,
		ExpiresAt:  tok.expiresAt,
		RevokedAt:  tok.revokedAt,
		ReplacedBy: tok.replacedBy,
	}, nil
}

func (f *fakeSessions) RotateRefreshToken(_ context.Context, oldID, newID uuid.UUID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	old, ok := f.byID[oldID]
	if !ok {
		return false, nil
	}
	if old.revokedAt != nil {
		return false, nil
	}
	now := f.clock.now()
	replacement := newID
	old.revokedAt = &now
	old.replacedBy = &replacement
	return true, nil
}

func (f *fakeSessions) DeleteRefreshToken(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if tok, ok := f.byID[id]; ok {
		delete(f.byID, id)
		for h, candidate := range f.byHash {
			if candidate == tok {
				delete(f.byHash, h)
			}
		}
	}
	return nil
}

func (f *fakeSessions) RevokeFamily(_ context.Context, familyID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := f.clock.now()
	for _, tok := range f.byID {
		if tok.familyID == familyID && tok.revokedAt == nil {
			tok.revokedAt = &now
		}
	}
	return nil
}

func (f *fakeSessions) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := f.clock.now()
	for _, tok := range f.byID {
		if tok.userID == userID && tok.revokedAt == nil {
			tok.revokedAt = &now
		}
	}
	return nil
}

func (f *fakeSessions) CreateOneTimeToken(_ context.Context, p repo.CreateOneTimeTokenParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.ottHash {
		if existing.userID == p.UserID && existing.purpose == p.Purpose && !existing.used {
			existing.used = true
		}
	}
	f.ottHash[string(p.TokenHash)] = &fakeOTT{
		userID:    p.UserID,
		purpose:   p.Purpose,
		expiresAt: p.ExpiresAt,
	}
	return nil
}

func (f *fakeSessions) ConsumeOneTimeToken(_ context.Context, purpose string, hash []byte) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	row, ok := f.ottHash[string(hash)]
	if !ok || row.purpose != purpose || row.used || !row.expiresAt.After(f.clock.now()) {
		return uuid.Nil, repo.ErrNotFound
	}
	row.used = true
	return row.userID, nil
}

func newTestService(t *testing.T) (*Service, *fakeUsers, *fakeSessions, *testClock) {
	t.Helper()

	clock := &testClock{t: time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)}
	users := newFakeUsers(clock)
	sessions := newFakeSessions(clock)

	svc := New(Options{
		Config:          testConfig(),
		Users:           users,
		Sessions:        sessions,
		ExposeDevTokens: true,
	})
	svc.SetClock(clock.now)
	return svc, users, sessions, clock
}

func wantCode(t *testing.T, err error, code string) *apperr.Error {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, ingin kode %s", code)
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error bukan *apperr.Error: %v", err)
	}
	if appErr.Code != code {
		t.Fatalf("kode = %s, ingin %s (pesan: %s)", appErr.Code, code, appErr.Message)
	}
	return appErr
}

func registerFixture(t *testing.T, svc *Service, email string) RegisterResult {
	t.Helper()

	res, err := svc.Register(context.Background(), RegisterInput{
		Email:    email,
		Password: "kaleng-enak-2026",
		FullName: "Budi Santoso",
		Phone:    "081234567890",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return res
}

// ---------------------------------------------------------------------------
// registrasi
// ---------------------------------------------------------------------------

func TestRegisterCreatesUnverifiedUser(t *testing.T) {
	svc, _, sessions, _ := newTestService(t)

	res := registerFixture(t, svc, "  Budi@Contoh.COM ")

	if res.User.Email != "budi@contoh.com" {
		t.Errorf("Email = %q, ingin dinormalkan menjadi budi@contoh.com", res.User.Email)
	}
	if res.User.EmailVerified() {
		t.Error("pengguna baru sudah terverifikasi")
	}
	if res.User.Role != "customer" {
		t.Errorf("Role = %q, ingin customer", res.User.Role)
	}
	if res.User.Phone == nil || *res.User.Phone != "+6281234567890" {
		t.Errorf("Phone = %v, ingin +6281234567890", res.User.Phone)
	}
	if res.DevVerifyToken == "" {
		t.Error("token verifikasi dev tidak dikembalikan")
	}
	if len(sessions.ottHash) != 1 {
		t.Errorf("jumlah token sekali pakai = %d, ingin 1", len(sessions.ottHash))
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	registerFixture(t, svc, "budi@contoh.com")
	_, err := svc.Register(context.Background(), RegisterInput{
		Email:    "BUDI@contoh.com",
		Password: "kaleng-enak-2026",
		FullName: "Budi Lain",
	})

	appErr := wantCode(t, err, apperr.CodeAuthEmailTaken)
	if len(appErr.Fields) != 0 {
		t.Errorf("Fields = %v, ingin kosong (jangan bocorkan field mana yang bentrok)", appErr.Fields)
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	_, err := svc.Register(context.Background(), RegisterInput{
		Email:    "budi@contoh.com",
		Password: "pendek",
		FullName: "Budi Santoso",
	})

	appErr := wantCode(t, err, apperr.CodeAuthPasswordWeak)
	if appErr.Fields["password"] == "" {
		t.Errorf("Fields = %v, ingin penjelasan pada field password", appErr.Fields)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	cases := []struct {
		name  string
		input RegisterInput
		field string
	}{
		{
			name:  "email tanpa domain",
			input: RegisterInput{Email: "budi@contoh", Password: "kaleng-enak-2026", FullName: "Budi Santoso"},
			field: "email",
		},
		{
			name:  "nama terlalu pendek",
			input: RegisterInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026", FullName: "B"},
			field: "full_name",
		},
		{
			name:  "nomor telepon tidak valid",
			input: RegisterInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026", FullName: "Budi Santoso", Phone: "12345"},
			field: "phone",
		},
		{
			name:  "tanggal lahir di masa depan",
			input: RegisterInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026", FullName: "Budi Santoso", BirthDate: "2030-01-01"},
			field: "birth_date",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), tc.input)
			appErr := wantCode(t, err, apperr.CodeAPIValidation)
			if appErr.Fields[tc.field] == "" {
				t.Errorf("Fields = %v, ingin penjelasan pada field %s", appErr.Fields, tc.field)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// login & lockout
// ---------------------------------------------------------------------------

func TestLoginUnknownEmailUsesGenericError(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	_, err := svc.Login(context.Background(), LoginInput{
		Email:    "tidak-ada@contoh.com",
		Password: "kaleng-enak-2026",
	})

	appErr := wantCode(t, err, apperr.CodeAuthLoginFail)
	if len(appErr.Fields) != 0 {
		t.Errorf("Fields = %v, ingin kosong", appErr.Fields)
	}
}

func TestLoginWrongPasswordLocksAccount(t *testing.T) {
	svc, _, _, clock := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	in := LoginInput{Email: "budi@contoh.com", Password: "password-salah-sekali"}

	// Percobaan 1–4 masih AUTH_LOGIN_FAIL; percobaan ke-5 mengunci akun.
	for i := 1; i <= 4; i++ {
		_, err := svc.Login(context.Background(), in)
		wantCode(t, err, apperr.CodeAuthLoginFail)
	}
	_, err := svc.Login(context.Background(), in)
	appErr := wantCode(t, err, apperr.CodeAuthLoginLocked)
	if !strings.Contains(appErr.Message, "menit") {
		t.Errorf("Message = %q, ingin menyebut sisa waktu dalam menit", appErr.Message)
	}

	// Password BENAR pun tetap ditolak selama lockout berjalan.
	_, err = svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	wantCode(t, err, apperr.CodeAuthLoginLocked)

	// Setelah masa lockout lewat, login kembali normal.
	clock.advance(16 * time.Minute)
	if _, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"}); err != nil {
		t.Fatalf("login setelah lockout error = %v", err)
	}
}

func TestLoginIssuesUsableSession(t *testing.T) {
	svc, users, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	session, err := svc.Login(context.Background(), LoginInput{
		Email:     "budi@contoh.com",
		Password:  "kaleng-enak-2026",
		UserAgent: "uji/1.0",
		IP:        "203.0.113.10",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if session.AccessToken == "" || session.RefreshToken == "" || session.CSRFToken == "" {
		t.Fatal("sesi tidak lengkap (access/refresh/csrf kosong)")
	}
	if !session.AccessExpiresAt.After(time.Now().Add(14 * time.Minute)) {
		t.Errorf("AccessExpiresAt = %v, terlalu dekat", session.AccessExpiresAt)
	}

	claims, err := svc.Tokens().ParseAccess(session.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if claims.UserID != session.User.ID {
		t.Errorf("klaim sub = %v, ingin %v", claims.UserID, session.User.ID)
	}

	// RecordLoginSuccess harus mereset penghitung gagal.
	stored, _ := users.GetUserByID(context.Background(), session.User.ID)
	if stored.FailedLoginCount != 0 || stored.LockedUntil != nil {
		t.Errorf("setelah login sukses: count=%d locked=%v, ingin 0/nil", stored.FailedLoginCount, stored.LockedUntil)
	}
}

// ---------------------------------------------------------------------------
// refresh & deteksi reuse
// ---------------------------------------------------------------------------

func TestRefreshRotatesToken(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	first, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	second, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: first.RefreshToken})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh token tidak berputar")
	}
	if second.User.ID != first.User.ID {
		t.Errorf("UserID berubah: %v → %v", first.User.ID, second.User.ID)
	}

	// Token baru tetap bisa dipakai.
	if _, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: second.RefreshToken}); err != nil {
		t.Fatalf("Refresh(token kedua) error = %v", err)
	}
}

func TestRefreshDetectsReuseAndKillsFamily(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	first, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	second, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: first.RefreshToken})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Token lama dipakai lagi → pencurian.
	_, err = svc.Refresh(context.Background(), RefreshInput{RefreshToken: first.RefreshToken})
	wantCode(t, err, apperr.CodeAuthRefreshReuse)

	// Token terbaru ikut mati karena seluruh keluarga dicabut. Perangkat yang
	// tidak bersalah cukup melihat sesi berakhir — hanya permintaan yang
	// membawa token hasil rotasi yang dilaporkan sebagai reuse.
	_, err = svc.Refresh(context.Background(), RefreshInput{RefreshToken: second.RefreshToken})
	wantCode(t, err, apperr.CodeAuthSessionExpired)
}

func TestRefreshRejectsUnknownAndLoggedOutTokens(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	_, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: "token-yang-tidak-pernah-ada"})
	wantCode(t, err, apperr.CodeAuthSessionExpired)

	session, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := svc.Logout(context.Background(), session.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	// Logout bukan reuse: sesi hanya berakhir.
	_, err = svc.Refresh(context.Background(), RefreshInput{RefreshToken: session.RefreshToken})
	wantCode(t, err, apperr.CodeAuthSessionExpired)
}

func TestRefreshRejectsExpiredToken(t *testing.T) {
	svc, _, _, clock := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	session, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	clock.advance(721 * time.Hour)
	_, err = svc.Refresh(context.Background(), RefreshInput{RefreshToken: session.RefreshToken})
	wantCode(t, err, apperr.CodeAuthSessionExpired)
}

// ---------------------------------------------------------------------------
// verifikasi email & reset password
// ---------------------------------------------------------------------------

func TestVerifyEmailConsumesTokenOnce(t *testing.T) {
	svc, users, _, _ := newTestService(t)
	res := registerFixture(t, svc, "budi@contoh.com")

	if err := svc.VerifyEmail(context.Background(), res.DevVerifyToken); err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}
	stored, _ := users.GetUserByID(context.Background(), res.User.ID)
	if !stored.EmailVerified() {
		t.Error("email belum ditandai terverifikasi")
	}

	if err := svc.VerifyEmail(context.Background(), res.DevVerifyToken); err == nil {
		t.Fatal("token yang sama diterima dua kali")
	} else {
		wantCode(t, err, apperr.CodeAuthVerifyExpired)
	}
}

func TestResendVerificationIsSilentForUnknownOrVerified(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	token, err := svc.ResendVerification(context.Background(), "tidak-ada@contoh.com")
	if err != nil {
		t.Fatalf("ResendVerification() error = %v", err)
	}
	if token != "" {
		t.Error("token dikembalikan untuk email yang tidak terdaftar")
	}

	res := registerFixture(t, svc, "budi@contoh.com")
	if err := svc.VerifyEmail(context.Background(), res.DevVerifyToken); err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}

	token, err = svc.ResendVerification(context.Background(), "budi@contoh.com")
	if err != nil {
		t.Fatalf("ResendVerification(verified) error = %v", err)
	}
	if token != "" {
		t.Error("token dibuat untuk email yang sudah terverifikasi")
	}
}

func TestForgotPasswordDoesNotEnumerate(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	known, err := svc.ForgotPassword(context.Background(), "budi@contoh.com")
	if err != nil {
		t.Fatalf("ForgotPassword(known) error = %v", err)
	}
	if known == "" {
		t.Fatal("token reset tidak dibuat untuk email terdaftar")
	}

	unknown, err := svc.ForgotPassword(context.Background(), "tidak-ada@contoh.com")
	if err != nil {
		t.Fatalf("ForgotPassword(unknown) error = %v", err)
	}
	if unknown != "" {
		t.Error("token dibuat untuk email yang tidak terdaftar")
	}
}

func TestResetPasswordRevokesSessions(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	session, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	token, err := svc.ForgotPassword(context.Background(), "budi@contoh.com")
	if err != nil {
		t.Fatalf("ForgotPassword() error = %v", err)
	}
	if err := svc.ResetPassword(context.Background(), token, "kaleng-baru-2027"); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	// Sesi lama tidak boleh bertahan.
	_, err = svc.Refresh(context.Background(), RefreshInput{RefreshToken: session.RefreshToken})
	wantCode(t, err, apperr.CodeAuthSessionExpired)

	// Password lama gagal, password baru berhasil.
	_, err = svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	wantCode(t, err, apperr.CodeAuthLoginFail)

	if _, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-baru-2027"}); err != nil {
		t.Fatalf("login dengan password baru error = %v", err)
	}
}

func TestResetPasswordRejectsReusedToken(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	token, err := svc.ForgotPassword(context.Background(), "budi@contoh.com")
	if err != nil {
		t.Fatalf("ForgotPassword() error = %v", err)
	}
	if err := svc.ResetPassword(context.Background(), token, "kaleng-baru-2027"); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	err = svc.ResetPassword(context.Background(), token, "kaleng-baru-2028")
	wantCode(t, err, apperr.CodeAuthVerifyExpired)
}

// ---------------------------------------------------------------------------
// ganti password
// ---------------------------------------------------------------------------

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	laptop, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026", UserAgent: "laptop"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	phone, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026", UserAgent: "ponsel"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fresh, err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:    laptop.User.ID,
		Current:   "kaleng-enak-2026",
		New:       "kaleng-baru-2027",
		UserAgent: "laptop",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	// Perangkat pemanggil mendapat sesi baru yang sah.
	if _, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: fresh.RefreshToken}); err != nil {
		t.Fatalf("Refresh(sesi baru) error = %v", err)
	}

	// Perangkat lain terlempar.
	_, err = svc.Refresh(context.Background(), RefreshInput{RefreshToken: phone.RefreshToken})
	wantCode(t, err, apperr.CodeAuthSessionExpired)
}

func TestChangePasswordRejectsWrongCurrent(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	_, err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:  uuid.New(),
		Current: "apa-saja-yang-panjang",
		New:     "kaleng-baru-2027",
	})
	wantCode(t, err, apperr.CodeAuthSessionExpired)

	session, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	_, err = svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:  session.User.ID,
		Current: "password-yang-salah",
		New:     "kaleng-baru-2027",
	})
	wantCode(t, err, apperr.CodeAuthLoginFail)

	_, err = svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:  session.User.ID,
		Current: "kaleng-enak-2026",
		New:     "kaleng-enak-2026",
	})
	wantCode(t, err, apperr.CodeAuthPasswordWeak)
}

// Password bersifat case-sensitive: variasi huruf besar/kecil adalah password
// BERBEDA dan tidak boleh ditolak sebagai "sama dengan yang lama".
func TestChangePasswordAllowsCaseVariant(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	registerFixture(t, svc, "budi@contoh.com")

	session, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "kaleng-enak-2026"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if _, err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:  session.User.ID,
		Current: "kaleng-enak-2026",
		New:     "Kaleng-Enak-2026",
	}); err != nil {
		t.Fatalf("ChangePassword(variasi kapital) error = %v, ingin lulus", err)
	}

	if _, err := svc.Login(context.Background(), LoginInput{Email: "budi@contoh.com", Password: "Kaleng-Enak-2026"}); err != nil {
		t.Fatalf("Login(password baru) error = %v", err)
	}
}
