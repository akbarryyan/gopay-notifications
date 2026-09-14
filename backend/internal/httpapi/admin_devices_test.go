package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	if err := s.CreateDevice(context.Background(), encKey(), "acc_1", deviceID, name, []byte("s")); err != nil {
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

func adminPost(t *testing.T, h http.Handler, cookie *http.Cookie, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminCreateDeviceSwalayanBerhasil(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminPost(t, h, cookie, "/api/v1/admin/devices", `{"name":"HP Toko Baru"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	var created struct {
		Success      bool   `json:"success"`
		DeviceID     string `json:"device_id"`
		DeviceSecret string `json:"device_secret"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.DeviceID == "" || created.DeviceSecret == "" {
		t.Fatalf("device_id/device_secret kosong: %+v", created)
	}

	list := adminGet(t, h, cookie, "/api/v1/admin/devices")
	var body struct {
		Devices []struct {
			DeviceID string `json:"device_id"`
			Name     string `json:"name"`
		} `json:"devices"`
	}
	json.Unmarshal(list.Body.Bytes(), &body)
	if len(body.Devices) != 1 || body.Devices[0].DeviceID != created.DeviceID || body.Devices[0].Name != "HP Toko Baru" {
		t.Fatalf("devices = %+v, mau 1 baris %q", body.Devices, created.DeviceID)
	}
	// secret tidak pernah ikut di GET list, cuma sekali di respons create ini.
	if strings.Contains(list.Body.String(), created.DeviceSecret) {
		t.Fatal("device_secret tidak boleh muncul lagi di GET /admin/devices")
	}
}

func TestAdminCreateDeviceNamaKosongDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminPost(t, h, cookie, "/api/v1/admin/devices", `{"name":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

// TestAdminCreateDeviceKuotaHabisDitolak: newAPIWithAdmin membuat acc_1
// dengan MaxDevices=10 -- isi sampai penuh lewat endpoint yang sama
// (bukan seedRawDevice, supaya sekalian membuktikan device ke-11 memang
// ditolak oleh endpoint, bukan cuma dihitung salah).
func TestAdminCreateDeviceKuotaHabisDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	for i := range 10 {
		rec := adminPost(t, h, cookie, "/api/v1/admin/devices", fmt.Sprintf(`{"name":"HP %d"}`, i))
		if rec.Code != http.StatusCreated {
			t.Fatalf("device ke-%d: status = %d (body=%s)", i, rec.Code, rec.Body.String())
		}
	}

	rec := adminPost(t, h, cookie, "/api/v1/admin/devices", `{"name":"HP ke-11"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("device ke-11: status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "device_limit_reached" {
		t.Fatalf("error = %q, mau device_limit_reached", errBody.Error)
	}
}

func adminDelete(t *testing.T, h http.Handler, cookie *http.Cookie, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminDeleteDeviceTanpaRiwayatBerhasil(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	created := adminPost(t, h, cookie, "/api/v1/admin/devices", `{"name":"HP Baru"}`)
	var body struct {
		DeviceID string `json:"device_id"`
	}
	json.Unmarshal(created.Body.Bytes(), &body)

	rec := adminDelete(t, h, cookie, "/api/v1/admin/devices/"+body.DeviceID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	list := adminGet(t, h, cookie, "/api/v1/admin/devices")
	var listBody struct {
		Devices []struct{} `json:"devices"`
	}
	json.Unmarshal(list.Body.Bytes(), &listBody)
	if len(listBody.Devices) != 0 {
		t.Fatalf("devices = %+v, mau kosong setelah dihapus", listBody.Devices)
	}
}

func TestAdminDeleteDeviceDenganRiwayatDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)
	// seedRawDevice memakai account_id "acc_1", sama dengan account bawaan
	// newAPIWithAdmin -- sampleEvent() (paket store_test) tidak dipakai di
	// sini, jadi event disisipkan langsung lewat store dengan device_id
	// yang sama.
	seedRawDevice(t, "dev_riwayat", "HP Lama")

	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	title := "Pembayaran QRIS statis diterima"
	if _, err := s.InsertEvent(context.Background(), store.Event{
		EventID: "evt_riwayat", AccountID: "acc_1", DeviceID: "dev_riwayat",
		Source: "gopay", PackageName: "com.gojek.gopaymerchant", Title: &title,
		PostedAt: fixedNow, ReceivedAt: fixedNow, RawPayload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("insert event: %v", err)
	}

	rec := adminDelete(t, h, cookie, "/api/v1/admin/devices/dev_riwayat")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "device_has_events" {
		t.Fatalf("error = %q, mau device_has_events", errBody.Error)
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
