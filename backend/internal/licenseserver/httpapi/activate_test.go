package httpapi_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type statePayload struct {
	LicenseID      string `json:"license_id"`
	InstallationID string `json:"installation_id"`
	Customer       string `json:"customer"`
	Plan           string `json:"plan"`
	AdminStatus    string `json:"admin_status"`
	MaxDevices     int    `json:"max_devices"`
}

// decodeStatePayload membaca baris pertama license_state (base64 JSON)
// TANPA verifikasi signature — cukup untuk memeriksa isi response di test
// httpapi ini; kebenaran kriptografinya sendiri diuji tuntas di
// internal/licensecheck.
func decodeStatePayload(t *testing.T, licenseState string) statePayload {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(licenseState), "\n")
	if len(lines) < 2 {
		t.Fatalf("license_state tidak berbentuk 2 baris: %q", licenseState)
	}
	raw, err := base64.StdEncoding.DecodeString(lines[0])
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	var p statePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return p
}

func doActivate(t *testing.T, h http.Handler, licenseKey, environment string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"license_key":"` + licenseKey + `","environment":"` + environment + `","product_version":"1.0.0"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/license/activate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	return rec
}

func TestActivateBerhasil(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	rawKey, licID := createTestLicenseWithKey(t, s)

	rec := doActivate(t, h, rawKey, "production")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		InstallationID string `json:"installation_id"`
		LicenseState   string `json:"license_state"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.InstallationID == "" {
		t.Fatal("installation_id kosong")
	}

	payload := decodeStatePayload(t, body.LicenseState)
	if payload.LicenseID != licID || payload.InstallationID != body.InstallationID {
		t.Fatalf("payload tidak sesuai: %+v", payload)
	}
	if payload.AdminStatus != "active" || payload.Plan != "Business" || payload.MaxDevices != 10 {
		t.Fatalf("payload tidak sesuai: %+v", payload)
	}
}

func TestActivateKeySalahDitolak(t *testing.T) {
	api, _ := newTestAPI(t)
	h := api.Handler()

	rec := doActivate(t, h, "PB-SALAH-0000-0000-0000", "production")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestActivateKuotaPenuhMenghasilkan409(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	rawKey, _ := createTestLicenseWithKey(t, s)

	if rec := doActivate(t, h, rawKey, "production"); rec.Code != http.StatusOK {
		t.Fatalf("aktivasi pertama gagal: %d %s", rec.Code, rec.Body.String())
	}
	rec := doActivate(t, h, rawKey, "production")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (kuota production = 1)", rec.Code)
	}
}

func TestActivateEnvironmentTidakDikenalDitolak(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	rawKey, _ := createTestLicenseWithKey(t, s)

	rec := doActivate(t, h, rawKey, "staging")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", rec.Code)
	}
}

func doValidate(t *testing.T, h http.Handler, licenseKey, installationID string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"installation_id":"` + installationID + `","product_version":"1.0.0"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/license/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+licenseKey)
	h.ServeHTTP(rec, req)
	return rec
}

func TestValidateBerhasil(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	rawKey, _ := createTestLicenseWithKey(t, s)

	actRec := doActivate(t, h, rawKey, "production")
	var actBody struct {
		InstallationID string `json:"installation_id"`
	}
	json.Unmarshal(actRec.Body.Bytes(), &actBody)

	rec := doValidate(t, h, rawKey, actBody.InstallationID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestValidateInstallationSudahDiresetMenghasilkan404(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	rawKey, _ := createTestLicenseWithKey(t, s)

	actRec := doActivate(t, h, rawKey, "production")
	var actBody struct {
		InstallationID string `json:"installation_id"`
	}
	json.Unmarshal(actRec.Body.Bytes(), &actBody)

	if err := s.ResetInstallation(t.Context(), actBody.InstallationID); err != nil {
		t.Fatalf("ResetInstallation: %v", err)
	}

	rec := doValidate(t, h, rawKey, actBody.InstallationID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 setelah installation direset", rec.Code)
	}
}

func TestValidateKeySalahDitolak(t *testing.T) {
	api, _ := newTestAPI(t)
	h := api.Handler()

	rec := doValidate(t, h, "PB-SALAH-0000-0000-0000", "inst_apa_saja")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestValidateInstallationTidakCocokLicenseLainDitolak(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	rawKeyA, _ := createTestLicenseWithKey(t, s)
	rawKeyB, _ := createTestLicenseWithKey(t, s)

	actRec := doActivate(t, h, rawKeyA, "production")
	var actBody struct {
		InstallationID string `json:"installation_id"`
	}
	json.Unmarshal(actRec.Body.Bytes(), &actBody)

	// installation_id milik license A, tapi pakai key license B.
	rec := doValidate(t, h, rawKeyB, actBody.InstallationID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 (installation bukan milik license ini)", rec.Code)
	}
}
