package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// newAPIWithLicense menyiapkan API lengkap (device, admin, API key) dengan
// lisensi TERTENTU — dipakai untuk menguji requireLicense di ketiga
// kategori route sekaligus tanpa menduplikasi setup tiga kali.
func newAPIWithLicense(t *testing.T, lic licensecheck.License) (h http.Handler, apiKey string) {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL belum diset. Jalankan: make db-up migrate")
	}

	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)

	if _, err := s.Pool().Exec(ctx,
		"TRUNCATE notification_events, event_reviews, invoices, api_keys, webhook_deliveries, webhook_endpoints, devices, admin_users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.UpsertAdmin(ctx, "admin", testAdminPassword); err != nil {
		t.Fatalf("UpsertAdmin: %v", err)
	}
	id, err := store.NewAPIKeyID()
	if err != nil {
		t.Fatalf("NewAPIKeyID: %v", err)
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret: %v", err)
	}
	if err := s.CreateAPIKey(ctx, id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	handler := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), lic,
		func() time.Time { return fixedNow }).Handler()
	return handler, rawKey
}

func TestRequireLicenseMenolakDeviceSaatTidakAktif(t *testing.T) {
	h, _ := newAPIWithLicense(t, licensecheck.License{Status: licensecheck.StatusExpired})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, mau 402 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "license_expired" {
		t.Fatalf("error = %q, mau license_expired", got)
	}
}

func TestRequireLicenseMenolakAPIKeySaatTidakAktif(t *testing.T) {
	h, apiKey := newAPIWithLicense(t, licensecheck.License{Status: licensecheck.StatusMissing})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/inv_apa_saja", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, mau 402 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "license_missing" {
		t.Fatalf("error = %q, mau license_missing", got)
	}
}

func TestRequireLicenseMenolakAdminSaatTidakAktif(t *testing.T) {
	h, _ := newAPIWithLicense(t, licensecheck.License{Status: licensecheck.StatusInvalid})

	loginRec := adminLogin(t, h, "admin", testAdminPassword)
	cookie := sessionCookieFrom(loginRec)
	if cookie == nil {
		t.Fatal("login tetap harus berhasil walau lisensi tidak aktif")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, mau 402 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "license_invalid" {
		t.Fatalf("error = %q, mau license_invalid", got)
	}
}

func TestAdminLicenseEndpointTetapBisaDiaksesSaatTidakAktif(t *testing.T) {
	h, _ := newAPIWithLicense(t, licensecheck.License{Status: licensecheck.StatusExpired, Reason: "lisensi sudah kedaluwarsa"})

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
		t.Fatalf("status = %d, mau 200 walau lisensi expired (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		License struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
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
	h, _ := newAPIWithLicense(t, licensecheck.License{Status: licensecheck.StatusActive})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/license", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestRequireLicenseMengizinkanSaatAktif(t *testing.T) {
	h, _ := newAPIWithLicense(t, licensecheck.License{Status: licensecheck.StatusActive})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}
