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

func seedRawDeviceForAccount(t *testing.T, accountID, deviceID, name string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.CreateDevice(context.Background(), encKey(), accountID, deviceID, name, []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
}

// createVendorTestAccount membuat satu account customer lewat endpoint
// vendor sungguhan (bukan seed langsung ke store) -- dipakai berulang oleh
// test device/api-key di halaman detail account Vendor Dashboard, yang
// semuanya butuh accountID nyata dengan plan tertentu.
func createVendorTestAccount(t *testing.T, h http.Handler, cookie *http.Cookie, plan string) string {
	t.Helper()
	body := `{"business_name":"Toko Dev","email":"dev-` + plan + `@t.test","username":"toko_` + plan +
		`","plan":"` + plan + `","expires_at":"2027-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create account status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var created struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create account: %v", err)
	}
	return created.Account.ID
}

func TestVendorListDevices(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")

	seedRawDeviceForAccount(t, accountID, "dev_v1", "HP Vendor Test")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/accounts/"+accountID+"/devices", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Devices []struct {
			DeviceID string `json:"device_id"`
			Name     string `json:"name"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Devices) != 1 || body.Devices[0].DeviceID != "dev_v1" || body.Devices[0].Name != "HP Vendor Test" {
		t.Fatalf("devices = %+v, mau 1 baris dev_v1", body.Devices)
	}
	if strings.Contains(rec.Body.String(), `"secret"`) {
		t.Fatal("response tidak boleh memuat field secret sama sekali")
	}
}

func TestVendorListDevicesButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/accounts/acc_x/devices", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 tanpa cookie vendor_session", rec.Code)
	}
}

func TestVendorCreateDeviceButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	rec := vendorRequest(t, h, nil, http.MethodPost, "/api/v1/vendor/accounts/acc_x/devices", `{"name":"HP"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 tanpa cookie vendor_session", rec.Code)
	}
}

func TestVendorCreateDevice(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")

	rec := vendorRequest(t, h, cookie, http.MethodPost,
		"/api/v1/vendor/accounts/"+accountID+"/devices", `{"name":"HP Titipan Customer"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201 (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		DeviceID     string `json:"device_id"`
		DeviceSecret string `json:"device_secret"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DeviceID == "" || body.DeviceSecret == "" {
		t.Fatalf("body = %+v, mau device_id dan device_secret terisi", body)
	}

	list := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/accounts/"+accountID+"/devices", "")
	var listBody struct {
		Devices []struct {
			DeviceID string `json:"device_id"`
			Name     string `json:"name"`
		} `json:"devices"`
	}
	json.Unmarshal(list.Body.Bytes(), &listBody)
	if len(listBody.Devices) != 1 || listBody.Devices[0].DeviceID != body.DeviceID {
		t.Fatalf("devices = %+v, mau 1 baris %s", listBody.Devices, body.DeviceID)
	}
}

// TestVendorCreateDeviceKuotaPenuh memastikan kuota max_devices plan
// ditegakkan sungguhan di endpoint ini, bukan cuma UI -- plan seed "Starter"
// dibatasi 3 device (lihat seedDefaultPlans di auth_middleware_test.go).
func TestVendorCreateDeviceKuotaPenuh(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")

	for i := 0; i < 3; i++ {
		rec := vendorRequest(t, h, cookie, http.MethodPost,
			"/api/v1/vendor/accounts/"+accountID+"/devices", `{"name":"HP"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("device ke-%d status = %d (body=%s)", i+1, rec.Code, rec.Body.String())
		}
	}

	rec := vendorRequest(t, h, cookie, http.MethodPost,
		"/api/v1/vendor/accounts/"+accountID+"/devices", `{"name":"HP Kelima"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 device_limit_reached (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "device_limit_reached" {
		t.Fatalf("error = %q, mau device_limit_reached", errBody.Error)
	}
}

func TestVendorDeleteDevice(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")
	seedRawDeviceForAccount(t, accountID, "dev_hapus", "HP Dihapus")

	rec := vendorRequest(t, h, cookie, http.MethodDelete,
		"/api/v1/vendor/accounts/"+accountID+"/devices/dev_hapus", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	list := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/accounts/"+accountID+"/devices", "")
	var listBody struct {
		Devices []struct{} `json:"devices"`
	}
	json.Unmarshal(list.Body.Bytes(), &listBody)
	if len(listBody.Devices) != 0 {
		t.Fatalf("devices = %+v, mau kosong setelah dihapus", listBody.Devices)
	}
}

func TestVendorDeleteDeviceDenganRiwayatDitolak(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")
	seedRawDeviceForAccount(t, accountID, "dev_riwayat", "HP Lama")

	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	title := "Pembayaran QRIS statis diterima"
	if _, err := s.InsertEvent(context.Background(), store.Event{
		EventID: "evt_riwayat_vendor", AccountID: accountID, DeviceID: "dev_riwayat",
		Source: "gopay", PackageName: "com.gojek.gopaymerchant", Title: &title,
		PostedAt: fixedNow, ReceivedAt: fixedNow, RawPayload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("insert event: %v", err)
	}

	rec := vendorRequest(t, h, cookie, http.MethodDelete,
		"/api/v1/vendor/accounts/"+accountID+"/devices/dev_riwayat", "")
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

func TestVendorSetDeviceEnabled(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")
	seedRawDeviceForAccount(t, accountID, "dev_toggle", "HP Toggle")

	rec := vendorRequest(t, h, cookie, http.MethodPatch,
		"/api/v1/vendor/accounts/"+accountID+"/devices/dev_toggle", `{"enabled":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	list := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/accounts/"+accountID+"/devices", "")
	var listBody struct {
		Devices []struct {
			DeviceID string `json:"device_id"`
			Enabled  bool   `json:"enabled"`
		} `json:"devices"`
	}
	json.Unmarshal(list.Body.Bytes(), &listBody)
	if len(listBody.Devices) != 1 || listBody.Devices[0].Enabled {
		t.Fatalf("devices = %+v, mau dev_toggle enabled=false", listBody.Devices)
	}
}

func TestVendorSetDeviceEnabledDeviceTakDikenal(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	accountID := createVendorTestAccount(t, h, cookie, "Starter")

	rec := vendorRequest(t, h, cookie, http.MethodPatch,
		"/api/v1/vendor/accounts/"+accountID+"/devices/dev_tidak_ada", `{"enabled":false}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}
