package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/akbarryyan/pg-aggregator-back/internal/domain/admin"
	"github.com/akbarryyan/pg-aggregator-back/internal/domain/merchant"
	"github.com/akbarryyan/pg-aggregator-back/internal/gopayonboard"
	"github.com/akbarryyan/pg-aggregator-back/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	adminTokenIssuer      = "pg-aggregator"
	adminTokenAudience    = "admin"
	merchantTokenAudience = "merchant"
	adminTokenTTL         = 24 * time.Hour
	merchantTokenTTL      = 24 * time.Hour
)

type AdminClaims struct {
	AdminID uuid.UUID `json:"admin_id"`
	Email   string    `json:"email"`
	Role    string    `json:"role"`
	jwt.RegisteredClaims
}

type MerchantClaims struct {
	UserID     uuid.UUID `json:"user_id"`
	MerchantID uuid.UUID `json:"merchant_id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	jwt.RegisteredClaims
}

// authMerchantRepository and authMerchantUserRepository capture exactly the
// repository methods AuthService depends on, so tests can substitute
// in-memory fakes instead of a live Postgres connection — same pattern as
// PaymentService's interfaces in interfaces.go. Concrete *repository.X types
// already satisfy these implicitly; main.go needs no changes.
type authMerchantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*merchant.Merchant, error)
	GetByEmail(ctx context.Context, email string) (*merchant.Merchant, error)
	Create(ctx context.Context, req *merchant.CreateMerchantRequest) (*merchant.Merchant, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type authMerchantUserRepository interface {
	GetByEmail(ctx context.Context, email string) (*merchant.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*merchant.User, error)
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID, at time.Time) error
	UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	Create(ctx context.Context, u *merchant.User) (*merchant.User, error)
}

// authAdminRepository captures exactly the repository methods AuthService
// depends on for admin auth — same fake-ability pattern as
// authMerchantRepository/authMerchantUserRepository above.
// *repository.AdminRepository already satisfies this implicitly.
type authAdminRepository interface {
	GetByEmail(ctx context.Context, email string) (*admin.Admin, error)
	GetByID(ctx context.Context, id uuid.UUID) (*admin.Admin, error)
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID, at time.Time) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	ExistsByEmailExceptID(ctx context.Context, email string, id uuid.UUID) (bool, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) (*admin.Admin, error)
}

type AuthService struct {
	adminRepo            authAdminRepository
	merchantUserRepo     authMerchantUserRepository
	merchantRepo         authMerchantRepository
	gopayCredentialsRepo gopayCredentialsRepository
	gopayBaseURL         string
	gopayPublicURL       string
	jwtSecret            []byte
}

func NewAuthService(adminRepo authAdminRepository, jwtSecret string) *AuthService {
	return &AuthService{
		adminRepo: adminRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthService) WithMerchantAuth(
	merchantUserRepo authMerchantUserRepository,
	merchantRepo authMerchantRepository,
) *AuthService {
	s.merchantUserRepo = merchantUserRepo
	s.merchantRepo = merchantRepo
	return s
}

// WithGopayOnboarding mengaktifkan cascade onboarding otomatis (lihat
// tryConnectGopay) -- gopayBaseURL dan gopayPublicURL BEDA secara
// sengaja: gopayBaseURL boleh loopback (mis. http://127.0.0.1:8080 di
// produksi, lebih cepat, satu mesin dengan gopay-notifications),
// gopayPublicURL WAJIB selalu URL publik (https://whuzpay.com) karena
// dipakai membangun backend_url di payload QR pairing yang harus bisa
// diakses HP lewat internet, bukan loopback VPS.
func (s *AuthService) WithGopayOnboarding(gopayBaseURL, gopayPublicURL string, repo gopayCredentialsRepository) *AuthService {
	s.gopayBaseURL = gopayBaseURL
	s.gopayPublicURL = gopayPublicURL
	s.gopayCredentialsRepo = repo
	return s
}

func (s *AuthService) LoginAdmin(ctx context.Context, req *admin.LoginRequest) (*admin.LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	a, err := s.adminRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == admin.ErrAdminNotFound {
			return nil, admin.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to load admin: %w", err)
	}

	if !a.IsActive {
		return nil, admin.ErrAdminInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(req.Password)); err != nil {
		return nil, admin.ErrInvalidCredentials
	}

	now := time.Now().UTC()
	expiresAt := now.Add(adminTokenTTL)

	claims := AdminClaims{
		AdminID: a.ID,
		Email:   a.Email,
		Role:    "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    adminTokenIssuer,
			Audience:  []string{adminTokenAudience},
			Subject:   a.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	if err := s.adminRepo.UpdateLastLoginAt(ctx, a.ID, now); err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}
	a.LastLoginAt = &now
	a.UpdatedAt = now

	return &admin.LoginResponse{
		Token:     signed,
		TokenType: "Bearer",
		ExpiresIn: int64(adminTokenTTL.Seconds()),
		Admin:     admin.ToAdminResponse(a),
	}, nil
}

func (s *AuthService) ParseAdminToken(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, admin.ErrInvalidToken
		}
		return s.jwtSecret, nil
	}, jwt.WithAudience(adminTokenAudience), jwt.WithIssuer(adminTokenIssuer))
	if err != nil {
		return nil, admin.ErrInvalidToken
	}

	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, admin.ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) GetAdminByID(ctx context.Context, id uuid.UUID) (*admin.Admin, error) {
	return s.adminRepo.GetByID(ctx, id)
}

func (s *AuthService) UpdateProfile(ctx context.Context, adminID uuid.UUID, req *admin.UpdateProfileRequest) (*admin.Admin, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if name == "" {
		return nil, admin.ErrNameRequired
	}
	if email == "" {
		return nil, admin.ErrEmailRequired
	}

	current, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if !current.IsActive {
		return nil, admin.ErrAdminInactive
	}

	taken, err := s.adminRepo.ExistsByEmailExceptID(ctx, email, adminID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, admin.ErrEmailAlreadyUsed
	}

	updated, err := s.adminRepo.UpdateProfile(ctx, adminID, name, email)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// VerifyAdminPassword checks the admin's current password without changing it.
func (s *AuthService) VerifyAdminPassword(ctx context.Context, adminID uuid.UUID, password string) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return admin.ErrInvalidCredentials
	}

	a, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}
	if !a.IsActive {
		return admin.ErrAdminInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)); err != nil {
		return admin.ErrInvalidCredentials
	}
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, adminID uuid.UUID, req *admin.ChangePasswordRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	a, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}

	if !a.IsActive {
		return admin.ErrAdminInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return admin.ErrCurrentPasswordInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.adminRepo.UpdatePasswordHash(ctx, adminID, string(hash)); err != nil {
		return fmt.Errorf("failed to save new password: %w", err)
	}

	return nil
}

// --- Merchant dashboard auth ---

func (s *AuthService) LoginMerchant(ctx context.Context, req *merchant.UserLoginRequest) (*merchant.UserLoginResponse, error) {
	if s.merchantUserRepo == nil {
		return nil, fmt.Errorf("merchant auth not configured")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	u, err := s.merchantUserRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == merchant.ErrMerchantUserNotFound {
			return nil, merchant.ErrMerchantInvalidCredentials
		}
		return nil, err
	}
	if !u.IsActive {
		return nil, merchant.ErrMerchantUserInactive
	}

	// Ensure parent merchant is active
	if s.merchantRepo != nil {
		m, err := s.merchantRepo.GetByID(ctx, u.MerchantID)
		if err != nil {
			return nil, err
		}
		if !m.IsActive {
			return nil, merchant.ErrMerchantUserInactive
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, merchant.ErrMerchantInvalidCredentials
	}

	now := time.Now().UTC()
	signed, err := s.signMerchantToken(u, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign merchant token: %w", err)
	}

	_ = s.merchantUserRepo.UpdateLastLoginAt(ctx, u.ID, now)
	u.LastLoginAt = &now

	respUser := merchant.ToUserResponse(u)
	if s.merchantRepo != nil {
		if m, err := s.merchantRepo.GetByID(ctx, u.MerchantID); err == nil {
			respUser.BusinessName = m.BusinessName
			respUser.WebhookURL = m.WebhookURL
		}
	}

	return &merchant.UserLoginResponse{
		Token:     signed,
		TokenType: "Bearer",
		ExpiresIn: int64(merchantTokenTTL.Seconds()),
		User:      respUser,
	}, nil
}

func (s *AuthService) RegisterMerchant(ctx context.Context, req *merchant.RegisterRequest) (*merchant.RegisterMerchantResponse, error) {
	if s.merchantRepo == nil || s.merchantUserRepo == nil {
		return nil, fmt.Errorf("merchant auth not configured")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	businessName := strings.TrimSpace(req.BusinessName)
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	if _, err := s.merchantRepo.GetByEmail(ctx, email); err == nil {
		return nil, merchant.ErrMerchantAlreadyExists
	} else if err != merchant.ErrMerchantNotFound {
		return nil, err
	}
	if _, err := s.merchantUserRepo.GetByEmail(ctx, email); err == nil {
		return nil, merchant.ErrMerchantAlreadyExists
	} else if err != merchant.ErrMerchantUserNotFound {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	createdMerchant, err := s.merchantRepo.Create(ctx, &merchant.CreateMerchantRequest{
		Name:         name,
		Email:        email,
		Phone:        phone,
		BusinessName: businessName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create merchant: %w", err)
	}

	createdUser, err := s.merchantUserRepo.Create(ctx, &merchant.User{
		MerchantID:   createdMerchant.ID,
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         "owner",
		IsActive:     true,
	})
	if err != nil {
		if delErr := s.merchantRepo.Delete(ctx, createdMerchant.ID); delErr != nil {
			logger.ErrorfCtx(ctx, "failed to roll back merchant %s after registration failure: %v", createdMerchant.ID, delErr)
		}
		return nil, fmt.Errorf("failed to create merchant owner account: %w", err)
	}

	now := time.Now().UTC()
	token, err := s.signMerchantToken(createdUser, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign merchant token: %w", err)
	}
	_ = s.merchantUserRepo.UpdateLastLoginAt(ctx, createdUser.ID, now)

	userResp := merchant.ToUserResponse(createdUser)
	userResp.BusinessName = businessName
	resp := &merchant.RegisterMerchantResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: int64(merchantTokenTTL.Seconds()),
		User:      userResp,
	}

	// Best-effort dari sini -- akun whuzpay-pg di atas SUDAH final. Tidak
	// ada apa pun di bawah ini yang boleh mengubah resp jadi error. Lihat
	// spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.3.
	s.tryConnectGopay(ctx, createdMerchant.ID, businessName, email, req.Password, req.QRISImageBase64, resp)

	return resp, nil
}

func (s *AuthService) signMerchantToken(u *merchant.User, now time.Time) (string, error) {
	expiresAt := now.Add(merchantTokenTTL)
	claims := MerchantClaims{
		UserID:     u.ID,
		MerchantID: u.MerchantID,
		Email:      u.Email,
		Role:       u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    adminTokenIssuer,
			Audience:  []string{merchantTokenAudience},
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// tryConnectGopay menjalankan cascade onboarding gopay-notifications
// (spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.1-§4.3).
// Best-effort murni -- setiap kegagalan dicatat di resp.GopayMessage,
// tidak pernah dikembalikan sebagai error ke pemanggil.
func (s *AuthService) tryConnectGopay(
	ctx context.Context,
	merchantID uuid.UUID,
	businessName, email, password, qrisImageBase64 string,
	resp *merchant.RegisterMerchantResponse,
) {
	if s.gopayCredentialsRepo == nil || s.gopayBaseURL == "" {
		return
	}

	client, err := gopayonboard.NewClient(s.gopayBaseURL, s.gopayPublicURL)
	if err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: buat client gagal untuk merchant %s: %v", merchantID, err)
		resp.GopayMessage = "Sedang ada gangguan menyambungkan otomatis, hubungkan manual lewat Settings."
		return
	}

	baseUsername := gopayonboard.SanitizeUsername(email)
	username := baseUsername
	var signErr error
	for attempt := 0; attempt < 4; attempt++ {
		candidate := baseUsername
		if attempt > 0 {
			candidate = fmt.Sprintf("%s%d", baseUsername, attempt)
		}
		signErr = client.SignUp(ctx, businessName, email, candidate, password)
		if signErr == nil {
			username = candidate
			break
		}
		if !errors.Is(signErr, gopayonboard.ErrUsernameTaken) {
			break // ErrEmailTaken atau error lain -- retry username tidak akan menolong
		}
	}
	if signErr != nil {
		if errors.Is(signErr, gopayonboard.ErrEmailTaken) {
			resp.GopayMessage = "Akun whuzpay-pg berhasil dibuat. Email ini sudah terdaftar di gopay-notifications -- hubungkan manual lewat Settings."
		} else {
			logger.ErrorfCtx(ctx, "gopay onboarding: signup gagal untuk merchant %s: %v", merchantID, signErr)
			resp.GopayMessage = "Akun whuzpay-pg berhasil dibuat. Sedang ada gangguan menyambungkan otomatis, hubungkan manual lewat Settings."
		}
		return
	}
	resp.GopayUsername = username
	resp.GopayConnected = true

	var apiKey, webhookSecret *string
	if key, err := client.CreateAPIKey(ctx, "whuzpay-pg"); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: create api key gagal untuk merchant %s: %v", merchantID, err)
		resp.GopayMessage = "Akun gopay-notifications tersambung, tapi API key gagal dibuat otomatis -- buat manual lewat Settings."
	} else {
		apiKey = &key
	}

	if secret, err := client.CreateWebhook(ctx, "whuzpay-pg",
		"https://pg.whuzpay.com/api/v1/provider-webhooks/gopay",
		[]string{"invoice.paid", "invoice.expired"}); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: create webhook gagal untuk merchant %s: %v", merchantID, err)
		if resp.GopayMessage == "" {
			resp.GopayMessage = "Akun gopay-notifications tersambung, tapi webhook gagal dibuat otomatis -- buat manual lewat Settings."
		}
	} else {
		webhookSecret = &secret
	}

	if err := s.gopayCredentialsRepo.Upsert(ctx, merchantID, apiKey, webhookSecret, &username); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: simpan kredensial gagal untuk merchant %s: %v", merchantID, err)
	}

	if deviceID, deviceSecret, err := client.CreateDevice(ctx, businessName+" - Bridge"); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: create device gagal untuk merchant %s: %v", merchantID, err)
	} else {
		resp.GopayDevice = &merchant.GopayDeviceInfo{
			DeviceID:     deviceID,
			DeviceSecret: deviceSecret,
			BackendURL:   client.PublicBackendURL(),
		}
	}

	if qrisImageBase64 != "" {
		if err := client.UploadQRISImage(ctx, qrisImageBase64); err != nil {
			logger.ErrorfCtx(ctx, "gopay onboarding: upload qris gagal untuk merchant %s: %v", merchantID, err)
		} else if err := s.gopayCredentialsRepo.MarkQRISConfigured(ctx, merchantID); err != nil {
			logger.ErrorfCtx(ctx, "gopay onboarding: tandai qris gagal untuk merchant %s: %v", merchantID, err)
		}
	}
}

func (s *AuthService) ParseMerchantToken(tokenString string) (*MerchantClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MerchantClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, merchant.ErrMerchantInvalidCredentials
		}
		return s.jwtSecret, nil
	}, jwt.WithAudience(merchantTokenAudience), jwt.WithIssuer(adminTokenIssuer))
	if err != nil {
		return nil, merchant.ErrMerchantInvalidCredentials
	}
	claims, ok := token.Claims.(*MerchantClaims)
	if !ok || !token.Valid {
		return nil, merchant.ErrMerchantInvalidCredentials
	}
	return claims, nil
}

func (s *AuthService) GetMerchantUserByID(ctx context.Context, id uuid.UUID) (*merchant.User, error) {
	if s.merchantUserRepo == nil {
		return nil, merchant.ErrMerchantUserNotFound
	}
	return s.merchantUserRepo.GetByID(ctx, id)
}

func (s *AuthService) GetMerchantUserProfile(ctx context.Context, userID uuid.UUID) (*merchant.UserResponse, error) {
	u, err := s.GetMerchantUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := merchant.ToUserResponse(u)
	if s.merchantRepo != nil {
		if m, err := s.merchantRepo.GetByID(ctx, u.MerchantID); err == nil {
			resp.BusinessName = m.BusinessName
			resp.WebhookURL = m.WebhookURL
		}
	}
	return resp, nil
}

func (s *AuthService) UpdateMerchantUserProfile(ctx context.Context, userID uuid.UUID, req *merchant.UserUpdateProfileRequest) (*merchant.UserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	u, err := s.GetMerchantUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, merchant.ErrMerchantUserInactive
	}
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if err := s.merchantUserRepo.UpdateProfile(ctx, userID, name, email); err != nil {
		return nil, err
	}
	return s.GetMerchantUserProfile(ctx, userID)
}

func (s *AuthService) ChangeMerchantPassword(ctx context.Context, userID uuid.UUID, req *merchant.UserChangePasswordRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	u, err := s.GetMerchantUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !u.IsActive {
		return merchant.ErrMerchantUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return merchant.ErrMerchantCurrentPasswordInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.merchantUserRepo.UpdatePasswordHash(ctx, userID, string(hash))
}

func (s *AuthService) VerifyMerchantPassword(ctx context.Context, userID uuid.UUID, password string) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return merchant.ErrMerchantInvalidCredentials
	}
	u, err := s.GetMerchantUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !u.IsActive {
		return merchant.ErrMerchantUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return merchant.ErrMerchantInvalidCredentials
	}
	return nil
}
