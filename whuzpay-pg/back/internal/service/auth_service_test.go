package service

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/akbarryyan/pg-aggregator-back/internal/domain/merchant"
	"github.com/google/uuid"
)

// ---- fakeAuthMerchantRepo -----------------------------------------------

type fakeAuthMerchantRepo struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*merchant.Merchant
}

func newFakeAuthMerchantRepo() *fakeAuthMerchantRepo {
	return &fakeAuthMerchantRepo{byID: map[uuid.UUID]*merchant.Merchant{}}
}

func (f *fakeAuthMerchantRepo) GetByID(ctx context.Context, id uuid.UUID) (*merchant.Merchant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.byID[id]
	if !ok {
		return nil, merchant.ErrMerchantNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeAuthMerchantRepo) GetByEmail(ctx context.Context, email string) (*merchant.Merchant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.byID {
		if strings.EqualFold(m.Email, email) {
			cp := *m
			return &cp, nil
		}
	}
	return nil, merchant.ErrMerchantNotFound
}

func (f *fakeAuthMerchantRepo) Create(ctx context.Context, req *merchant.CreateMerchantRequest) (*merchant.Merchant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	m := &merchant.Merchant{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		BusinessName: req.BusinessName,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	f.byID[m.ID] = m
	return m, nil
}

func (f *fakeAuthMerchantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
	return nil
}

// ---- fakeAuthMerchantUserRepo -------------------------------------------

type fakeAuthMerchantUserRepo struct {
	mu       sync.Mutex
	byID     map[uuid.UUID]*merchant.User
	failNext bool
}

func newFakeAuthMerchantUserRepo() *fakeAuthMerchantUserRepo {
	return &fakeAuthMerchantUserRepo{byID: map[uuid.UUID]*merchant.User{}}
}

func (f *fakeAuthMerchantUserRepo) GetByEmail(ctx context.Context, email string) (*merchant.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if strings.EqualFold(u.Email, email) {
			cp := *u
			return &cp, nil
		}
	}
	return nil, merchant.ErrMerchantUserNotFound
}

func (f *fakeAuthMerchantUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*merchant.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, merchant.ErrMerchantUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeAuthMerchantUserRepo) UpdateLastLoginAt(ctx context.Context, id uuid.UUID, at time.Time) error {
	return nil
}

func (f *fakeAuthMerchantUserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) error {
	return nil
}

func (f *fakeAuthMerchantUserRepo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return nil
}

func (f *fakeAuthMerchantUserRepo) Create(ctx context.Context, u *merchant.User) (*merchant.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return nil, errors.New("simulated insert failure")
	}
	now := time.Now().UTC()
	cp := *u
	cp.ID = uuid.New()
	cp.CreatedAt = now
	cp.UpdatedAt = now
	f.byID[cp.ID] = &cp
	return &cp, nil
}

// ---- tests ---------------------------------------------------------------

func newTestAuthServiceForRegister() (*AuthService, *fakeAuthMerchantRepo, *fakeAuthMerchantUserRepo) {
	merchantRepo := newFakeAuthMerchantRepo()
	userRepo := newFakeAuthMerchantUserRepo()
	svc := NewAuthService(nil, "test-secret").WithMerchantAuth(userRepo, merchantRepo)
	return svc, merchantRepo, userRepo
}

func validRegisterRequest() *merchant.RegisterRequest {
	return &merchant.RegisterRequest{
		Name:         "Budi Santoso",
		BusinessName: "Toko Budi Jaya",
		Email:        "budi@tokobudi.id",
		Phone:        "08123456789",
		Password:     "supersecret1",
	}
}

func TestAuthService_RegisterMerchant_HappyPath(t *testing.T) {
	svc, merchantRepo, userRepo := newTestAuthServiceForRegister()

	resp, err := svc.RegisterMerchant(context.Background(), validRegisterRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected auto-login token to be issued")
	}
	if resp.User.BusinessName != "Toko Budi Jaya" {
		t.Errorf("expected business name to match, got %q", resp.User.BusinessName)
	}
	if len(merchantRepo.byID) != 1 {
		t.Fatalf("expected 1 merchant created, got %d", len(merchantRepo.byID))
	}
	if len(userRepo.byID) != 1 {
		t.Fatalf("expected 1 owner user created, got %d", len(userRepo.byID))
	}
	for _, u := range userRepo.byID {
		if u.Role != "owner" {
			t.Errorf("expected role owner, got %q", u.Role)
		}
		if !u.IsActive {
			t.Errorf("expected user to be active")
		}
		if u.MerchantID != resp.User.MerchantID {
			t.Errorf("expected user merchant_id %s to match created merchant %s", u.MerchantID, resp.User.MerchantID)
		}
	}
}

func TestAuthService_RegisterMerchant_DuplicateMerchantEmail(t *testing.T) {
	svc, merchantRepo, _ := newTestAuthServiceForRegister()
	merchantRepo.byID[uuid.New()] = &merchant.Merchant{ID: uuid.New(), Email: "budi@tokobudi.id"}

	_, err := svc.RegisterMerchant(context.Background(), validRegisterRequest())
	if err != merchant.ErrMerchantAlreadyExists {
		t.Fatalf("expected ErrMerchantAlreadyExists, got %v", err)
	}
}

func TestAuthService_RegisterMerchant_DuplicateUserEmail(t *testing.T) {
	svc, _, userRepo := newTestAuthServiceForRegister()
	userRepo.byID[uuid.New()] = &merchant.User{ID: uuid.New(), Email: "budi@tokobudi.id"}

	_, err := svc.RegisterMerchant(context.Background(), validRegisterRequest())
	if err != merchant.ErrMerchantAlreadyExists {
		t.Fatalf("expected ErrMerchantAlreadyExists, got %v", err)
	}
}

func TestAuthService_RegisterMerchant_RollsBackMerchantWhenUserCreateFails(t *testing.T) {
	svc, merchantRepo, userRepo := newTestAuthServiceForRegister()
	userRepo.failNext = true

	_, err := svc.RegisterMerchant(context.Background(), validRegisterRequest())
	if err == nil {
		t.Fatal("expected error when owner user creation fails")
	}
	if len(merchantRepo.byID) != 0 {
		t.Fatalf("expected merchant to be rolled back, got %d remaining", len(merchantRepo.byID))
	}
}

// ---- cascade onboarding gopay-notifications ------------------------------
// Lihat docs/superpowers/specs/2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.1-§4.3.

func TestRegisterMerchant_CascadeSuksesPenuh(t *testing.T) {
	var uploadedQRIS bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/signup":
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi"})
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.URL.Path == "/api/v1/admin/api-keys":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"key":"sk_abc"}`))
		case r.URL.Path == "/api/v1/admin/webhooks":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"secret":"whsec_abc"}`))
		case r.URL.Path == "/api/v1/admin/devices":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"device_id":"dev_abc","device_secret":"c2VjcmV0"}`))
		case r.URL.Path == "/api/v1/admin/account/qris-image":
			uploadedQRIS = true
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	svc, _, _ := newTestAuthServiceForRegister()
	svc.WithGopayOnboarding(srv.URL, srv.URL, newFakeGopayCredsRepo())

	pngB64 := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	req := validRegisterRequest()
	req.Email = "budi@toko.com"
	req.QRISImageBase64 = pngB64

	resp, err := svc.RegisterMerchant(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterMerchant: %v", err)
	}
	if resp.Token == "" {
		t.Error("Token kosong -- auto-login seharusnya tetap terbit walau cascade gopay dijalankan")
	}
	if !resp.GopayConnected {
		t.Error("GopayConnected = false, mau true")
	}
	if resp.GopayUsername != "budi" {
		t.Errorf("GopayUsername = %q, mau budi", resp.GopayUsername)
	}
	if resp.GopayDevice == nil || resp.GopayDevice.DeviceID != "dev_abc" {
		t.Errorf("GopayDevice = %+v", resp.GopayDevice)
	}
	if resp.GopayDevice.BackendURL != srv.URL+"/api/v1" {
		t.Errorf("BackendURL = %q", resp.GopayDevice.BackendURL)
	}
	if !uploadedQRIS {
		t.Error("QRIS tidak pernah diupload")
	}
}

func TestRegisterMerchant_EmailTakenDiGopay_TetapBerhasilDenganFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"success":false,"error":"email_taken","message":"sudah dipakai"}`))
	}))
	defer srv.Close()

	svc, _, _ := newTestAuthServiceForRegister()
	svc.WithGopayOnboarding(srv.URL, srv.URL, newFakeGopayCredsRepo())

	req := validRegisterRequest()
	req.Email = "budi@toko.com"
	resp, err := svc.RegisterMerchant(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterMerchant harus tetap sukses, dapat: %v", err)
	}
	if resp.Token == "" {
		t.Error("akun whuzpay-pg wajib tetap dibuat + auto-login walau gopay gagal")
	}
	if resp.GopayConnected {
		t.Error("GopayConnected = true, mau false")
	}
	if resp.GopayMessage == "" {
		t.Error("GopayMessage kosong, mau ada penjelasan fallback manual")
	}
}

func TestRegisterMerchant_GopayTidakBisaDihubungi_TetapBerhasil(t *testing.T) {
	svc, _, _ := newTestAuthServiceForRegister()
	// Sengaja arahkan ke port yang tidak ada listener-nya sama sekali.
	svc.WithGopayOnboarding("http://127.0.0.1:1", "http://127.0.0.1:1", newFakeGopayCredsRepo())

	req := validRegisterRequest()
	req.Email = "budi2@toko.com"
	resp, err := svc.RegisterMerchant(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterMerchant harus tetap sukses, dapat: %v", err)
	}
	if resp.Token == "" || resp.GopayConnected {
		t.Errorf("resp = %+v", resp)
	}
}

func TestRegisterMerchant_GagalSebagian_ApiKeyTersimpanWebhookKosong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/signup":
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi"})
			_, _ = w.Write([]byte(`{"success":true}`))
		case "/api/v1/admin/api-keys":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"key":"sk_abc"}`))
		case "/api/v1/admin/webhooks":
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/v1/admin/devices":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	svc, _, _ := newTestAuthServiceForRegister()
	credsRepo := newFakeGopayCredsRepo()
	svc.WithGopayOnboarding(srv.URL, srv.URL, credsRepo)

	req := validRegisterRequest()
	req.Email = "budi3@toko.com"
	resp, err := svc.RegisterMerchant(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterMerchant: %v", err)
	}
	if !resp.GopayConnected {
		t.Error("signup sukses, GopayConnected harus true walau langkah sesudahnya gagal sebagian")
	}
	if resp.GopayDevice != nil {
		t.Error("device gagal dibuat, GopayDevice harus nil")
	}

	// Kredensial API key yang berhasil harus tetap tersimpan meski webhook gagal.
	found := false
	for _, c := range credsRepo.creds {
		if c.APIKey != nil && *c.APIKey == "sk_abc" {
			found = true
			if c.WebhookSecret != nil {
				t.Error("webhook gagal dibuat, WebhookSecret seharusnya tetap nil")
			}
		}
	}
	if !found {
		t.Error("API key yang berhasil didapat tidak tersimpan")
	}
}
