package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// accountState menjelaskan status akun uji: admin_status ("active",
// "suspended", "revoked") dan expires_at relatif terhadap fixedNow.
type accountState struct {
	adminStatus string
	expiresAt   time.Time
}

// newAPIWithAccountState menyiapkan API lengkap (device, admin, API key)
// dengan status account TERTENTU — dipakai untuk menguji requireActiveAccount
// di ketiga kategori route sekaligus tanpa menduplikasi setup tiga kali.
func newAPIWithAccountState(t *testing.T, st accountState) (h http.Handler, apiKey string) {
	t.Helper()

	s := newTestStore(t)
	ctx := context.Background()

	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_1", BusinessName: "Toko Uji", Email: "acc_1@uji.test",
		Username: "admin", PlaintextPassword: testAdminPassword,
		Plan: "Business", MaxDevices: 10, ExpiresAt: st.expiresAt,
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if st.adminStatus != "active" {
		if err := s.SetAccountAdminStatus(ctx, "acc_1", st.adminStatus); err != nil {
			t.Fatalf("set admin_status: %v", err)
		}
	}

	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	id, err := store.NewAPIKeyID()
	if err != nil {
		t.Fatalf("NewAPIKeyID: %v", err)
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret: %v", err)
	}
	if err := s.CreateAPIKey(ctx, "acc_1", id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	handler := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(),
		func() time.Time { return fixedNow }).Handler()
	return handler, rawKey
}

func TestRequireActiveAccountMenolakDeviceSaatKedaluwarsa(t *testing.T) {
	h, _ := newAPIWithAccountState(t, accountState{adminStatus: "active", expiresAt: fixedNow.Add(-time.Hour)})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, mau 402 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "account_expired" {
		t.Fatalf("error = %q, mau account_expired", got)
	}
}

func TestRequireActiveAccountMenolakAPIKeySaatDisuspend(t *testing.T) {
	h, apiKey := newAPIWithAccountState(t, accountState{adminStatus: "suspended", expiresAt: fixedNow.Add(365 * 24 * time.Hour)})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/inv_apa_saja", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, mau 402 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "account_suspended" {
		t.Fatalf("error = %q, mau account_suspended", got)
	}
}

func TestRequireActiveAccountMenolakAdminSaatDicabut(t *testing.T) {
	h, _ := newAPIWithAccountState(t, accountState{adminStatus: "revoked", expiresAt: fixedNow.Add(365 * 24 * time.Hour)})

	loginRec := adminLogin(t, h, "admin", testAdminPassword)
	cookie := sessionCookieFrom(loginRec)
	if cookie == nil {
		t.Fatal("login tetap harus berhasil walau akun tidak aktif")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, mau 402 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "account_revoked" {
		t.Fatalf("error = %q, mau account_revoked", got)
	}
}

func TestAdminLicenseEndpointTetapBisaDiaksesSaatTidakAktif(t *testing.T) {
	h, _ := newAPIWithAccountState(t, accountState{adminStatus: "active", expiresAt: fixedNow.Add(-time.Hour)})

	loginRec := adminLogin(t, h, "admin", testAdminPassword)
	cookie := sessionCookieFrom(loginRec)
	if cookie == nil {
		t.Fatal("login gagal")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/license", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 walau akun expired (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		License struct {
			Status string `json:"status"`
		} `json:"license"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.License.Status != "expired" {
		t.Fatalf("status = %q, mau expired", body.License.Status)
	}
}

func TestAdminLicenseEndpointButuhSesi(t *testing.T) {
	h, _ := newAPIWithAccountState(t, accountState{adminStatus: "active", expiresAt: fixedNow.Add(365 * 24 * time.Hour)})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/license", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestRequireActiveAccountMengizinkanSaatAktif(t *testing.T) {
	h, _ := newAPIWithAccountState(t, accountState{adminStatus: "active", expiresAt: fixedNow.Add(365 * 24 * time.Hour)})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}
