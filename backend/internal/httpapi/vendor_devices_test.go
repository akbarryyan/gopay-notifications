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

func TestVendorListDevices(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	createBody := `{"business_name":"Toko Dev","email":"dev@t.test","username":"toko_dev","plan":"Starter","expires_at":"2027-01-01"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, createReq)
	var created struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	json.Unmarshal(createRec.Body.Bytes(), &created)
	accountID := created.Account.ID

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
