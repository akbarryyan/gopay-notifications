package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func seedRawDevice(t *testing.T, deviceID, name string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.CreateDevice(context.Background(), encKey(), deviceID, name, []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
}

func TestAdminDevicesTidakMemuatSecret(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	seedRawDevice(t, "dev_a", "HP A")

	rec := adminGet(t, h, cookie, "/api/v1/admin/devices")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"secret"`) {
		t.Fatal("response tidak boleh memuat field secret sama sekali")
	}
}

func TestAdminDevicesStatusPendingSebelumHeartbeat(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	seedRawDevice(t, "dev_a", "HP A")

	rec := adminGet(t, h, cookie, "/api/v1/admin/devices")

	var body struct {
		Devices []struct {
			DeviceID string `json:"device_id"`
			Status   string `json:"status"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Devices) != 1 {
		t.Fatalf("len = %d, mau 1", len(body.Devices))
	}
	if body.Devices[0].Status != "PENDING" {
		t.Fatalf("status = %q, mau PENDING", body.Devices[0].Status)
	}
}

func TestAdminSetDeviceEnabledMenonaktifkanDevice(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	seedRawDevice(t, "dev_a", "HP A")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/devices/dev_a",
		strings.NewReader(`{"enabled":false}`))
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	list := adminGet(t, h, cookie, "/api/v1/admin/devices")
	var body struct {
		Devices []struct {
			Enabled bool   `json:"enabled"`
			Status  string `json:"status"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Devices[0].Enabled {
		t.Fatal("device seharusnya dinonaktifkan")
	}
	if body.Devices[0].Status != "DISABLED" {
		t.Fatalf("status = %q, mau DISABLED", body.Devices[0].Status)
	}
}

func TestAdminSetDeviceEnabledDeviceTakDikenal(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/devices/tidak_ada",
		strings.NewReader(`{"enabled":false}`))
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}

func TestAdminDevicesMemerlukanSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/devices", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestAdminEventsMemerlukanSesiBukanBasicAuth(t *testing.T) {
	// Rute admin terpisah dari GET /api/v1/events yang dilindungi Caddy —
	// tanpa cookie sesi, permintaan ini harus ditolak di level aplikasi.
	h := newAPIWithAdmin(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/events", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestAdminEventsDenganSesiMengembalikanArray(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminGet(t, h, cookie, "/api/v1/admin/events")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []any `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Events == nil {
		t.Fatal("events = null, mau array kosong")
	}
}
