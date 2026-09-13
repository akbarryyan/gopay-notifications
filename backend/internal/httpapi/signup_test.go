package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func signupReq(t *testing.T, h http.Handler, businessName, email, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"business_name":"` + businessName + `","email":"` + email + `","username":"` + username + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSignupBerhasilMenerbitkanCookieDanAccount(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := signupReq(t, h, "Toko Baru", "baru@uji.test", "toko_baru", "password123")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if sessionCookieFrom(rec) == nil {
		t.Fatal("signup berhasil seharusnya langsung menerbitkan cookie admin_session (auto-login)")
	}

	loginRec := adminLogin(t, h, "toko_baru", "password123")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login setelah signup gagal: status = %d", loginRec.Code)
	}
}

func TestSignupEmailBentrokDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := signupReq(t, h, "Toko Lain", "acc_1@uji.test", "toko_lain", "password123")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "email_taken" {
		t.Fatalf("error = %q, mau email_taken", body.Error)
	}
}

func TestSignupUsernameBentrokDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := signupReq(t, h, "Toko Lain", "lain@uji.test", "admin", "password123")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "username_taken" {
		t.Fatalf("error = %q, mau username_taken", body.Error)
	}
}

func TestSignupPasswordPendekDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := signupReq(t, h, "Toko Baru", "baru2@uji.test", "toko_baru2", "pendek")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestSignupFieldKosongDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := signupReq(t, h, "", "baru3@uji.test", "toko_baru3", "password123")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestSignupDibatasiSetelahBanyakPercobaan(t *testing.T) {
	h := newAPIWithAdmin(t)

	var rec *httptest.ResponseRecorder
	for i := 0; i < 5; i++ {
		rec = signupReq(t, h, "Toko X", "x@uji.test", "toko_x", "pendek")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan ke-%d: status = %d, mau 400", i+1, rec.Code)
		}
	}

	rec = signupReq(t, h, "Toko X", "x2@uji.test", "toko_x2", "password123")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, mau 429", rec.Code)
	}
}

func TestSignupTidakButuhCredentialApaPun(t *testing.T) {
	h := newAPIWithAdmin(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signup",
		strings.NewReader(`{"business_name":"Toko Tanpa Auth","email":"tanpaauth@uji.test","username":"tanpa_auth","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 tanpa credential apa pun (body=%s)", rec.Code, rec.Body.String())
	}
}
