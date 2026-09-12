package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func adminGet(t *testing.T, h http.Handler, cookie *http.Cookie, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	return rec
}

func loginAsAdmin(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	cookie := sessionCookieFrom(adminLogin(t, h, "admin", testAdminPassword))
	if cookie == nil {
		t.Fatal("login gagal menerbitkan cookie")
	}
	return cookie
}

func TestAdminOverviewKosongMasukAkal(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminGet(t, h, cookie, "/api/v1/admin/overview")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Devices struct {
			Total int `json:"total"`
		} `json:"devices"`
		Events struct {
			Today    int     `json:"today"`
			LatestAt *string `json:"latest_at"`
		} `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Devices.Total != 0 {
		t.Fatalf("Devices.Total = %d, mau 0", body.Devices.Total)
	}
	if body.Events.LatestAt != nil {
		t.Fatal("LatestAt seharusnya null sebelum ada event")
	}
}

func TestAdminOverviewMenghitungDeviceDanEventSungguhan(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	// Ditulis lewat koneksi store terpisah ke database yang SAMA — hasilnya
	// harus terlihat oleh handler admin tanpa perlu lewat jalur HTTP device.
	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.CreateDevice(context.Background(), encKey(), "dev_seed", "HP Seed", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	title := "Transfer masuk"
	if _, err := s.InsertEvent(context.Background(), store.Event{
		EventID:     "evt_00000000000000000000000000000001",
		DeviceID:    "dev_seed",
		Source:      "gopay",
		PackageName: "com.gojek.gopaymerchant",
		Title:       &title,
		PostedAt:    time.Now(),
		ReceivedAt:  time.Now(),
		RawPayload:  []byte(`{}`),
	}); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	rec := adminGet(t, h, cookie, "/api/v1/admin/overview")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Devices struct {
			Total int `json:"total"`
		} `json:"devices"`
		Events struct {
			Today int `json:"today"`
		} `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Devices.Total != 1 {
		t.Fatalf("Devices.Total = %d, mau 1", body.Devices.Total)
	}
	if body.Events.Today != 1 {
		t.Fatalf("Events.Today = %d, mau 1", body.Events.Today)
	}
}

func TestAdminOverviewMemerlukanSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}
