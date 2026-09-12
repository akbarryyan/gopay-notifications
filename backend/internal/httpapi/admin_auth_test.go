package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

const testAdminPassword = "password-admin-test"

// newAPIWithAdmin menyiapkan API dengan satu akun admin terdaftar.
func newAPIWithAdmin(t *testing.T) http.Handler {
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
	if err := s.UpsertAdmin(ctx, "admin", testAdminPassword); err != nil {
		t.Fatalf("UpsertAdmin: %v", err)
	}

	return httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), func() time.Time { return fixedNow }).Handler()
}

func adminLogin(t *testing.T, h http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	body := `{"username":"` + username + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	return rec
}

// sessionCookieFrom mengambil cookie sesi dari response login agar dapat
// dipasang kembali di request berikutnya, meniru browser sungguhan.
func sessionCookieFrom(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "admin_session" {
			return c
		}
	}
	return nil
}

func TestAdminLoginBerhasilMenerbitkanCookie(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := adminLogin(t, h, "admin", testAdminPassword)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	cookie := sessionCookieFrom(rec)
	if cookie == nil {
		t.Fatal("cookie admin_session tidak ditemukan di response")
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie sesi wajib HttpOnly")
	}
}

func TestAdminLoginPasswordSalahDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := adminLogin(t, h, "admin", "password-salah")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if sessionCookieFrom(rec) != nil {
		t.Fatal("tidak boleh ada cookie sesi untuk login yang gagal")
	}
}

func TestAdminLoginUsernameTakDikenalDisamakanDenganPasswordSalah(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := adminLogin(t, h, "bukan_admin", "apa saja")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "invalid_credentials" {
		t.Fatalf("error = %q, mau invalid_credentials — username tak dikenal tidak boleh dibedakan dari password salah", body.Error)
	}
}

func TestAdminLoginDibatasiSetelahBanyakPercobaanGagal(t *testing.T) {
	h := newAPIWithAdmin(t)

	var rec *httptest.ResponseRecorder
	for i := 0; i < 5; i++ {
		rec = adminLogin(t, h, "admin", "password-salah")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("percobaan ke-%d: status = %d, mau 401", i+1, rec.Code)
		}
	}

	// Percobaan ke-6, bahkan dengan password yang BENAR, harus tetap ditolak
	// karena sudah melewati batas — throttle bekerja per IP, bukan per hasil.
	rec = adminLogin(t, h, "admin", testAdminPassword)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, mau 429", rec.Code)
	}
}

func TestAdminAksesEndpointTerlindungTanpaSesiDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestAdminAksesEndpointTerlindungDenganSesiValid(t *testing.T) {
	h := newAPIWithAdmin(t)

	loginRec := adminLogin(t, h, "admin", testAdminPassword)
	cookie := sessionCookieFrom(loginRec)
	if cookie == nil {
		t.Fatal("login seharusnya menerbitkan cookie")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestAdminAksesDenganCookieDiubahDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	loginRec := adminLogin(t, h, "admin", testAdminPassword)
	cookie := sessionCookieFrom(loginRec)
	if cookie == nil {
		t.Fatal("login seharusnya menerbitkan cookie")
	}
	cookie.Value = cookie.Value[:len(cookie.Value)-1] + "x"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestAdminLogoutMenghapusCookie(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/logout", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
	cookie := sessionCookieFrom(rec)
	if cookie == nil {
		t.Fatal("logout seharusnya tetap mengirim cookie (untuk menghapusnya di browser)")
	}
	if cookie.MaxAge >= 0 {
		t.Fatalf("MaxAge = %d, mau negatif supaya browser menghapus cookie", cookie.MaxAge)
	}
}
