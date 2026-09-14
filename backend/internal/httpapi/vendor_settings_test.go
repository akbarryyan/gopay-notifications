package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVendorChangePasswordBerhasilDanBisaLoginDenganYangBaru(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"current_password":"` + testVendorPassword + `","new_password":"password-baru-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/me/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	// Password lama tidak boleh lagi berhasil.
	oldLoginRec := vendorLogin(t, h, "akbar", testVendorPassword)
	if oldLoginRec.Code == http.StatusOK {
		t.Fatal("login dengan password lama seharusnya gagal setelah diganti")
	}

	// Password baru harus berhasil.
	newLoginRec := vendorLogin(t, h, "akbar", "password-baru-123")
	if newLoginRec.Code != http.StatusOK {
		t.Fatalf("login dengan password baru = %d, mau 200 (body=%s)", newLoginRec.Code, newLoginRec.Body.String())
	}
}

func TestVendorChangePasswordSaatIniSalahDitolak(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"current_password":"salah-password","new_password":"password-baru-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/me/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "invalid_credentials" {
		t.Fatalf("error = %q, mau invalid_credentials", errBody.Error)
	}
}

func TestVendorChangePasswordTerlaluPendekDitolak(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"current_password":"` + testVendorPassword + `","new_password":"pendek"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/me/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestVendorChangePasswordButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/me/password",
		strings.NewReader(`{"current_password":"x","new_password":"password-baru-123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 tanpa cookie vendor_session", rec.Code)
	}
}
