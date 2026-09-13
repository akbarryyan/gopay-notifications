package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
)

const testVendorPassword = "password-vendor-test"

// newAPIWithVendor menyiapkan API dengan satu vendor admin (username
// "akbar") terdaftar.
func newAPIWithVendor(t *testing.T) http.Handler {
	t.Helper()
	s := newTestStore(t)
	if err := s.UpsertVendorAdmin(context.Background(), "akbar", testVendorPassword); err != nil {
		t.Fatalf("UpsertVendorAdmin: %v", err)
	}
	return httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(),
		func() time.Time { return fixedNow }).Handler()
}

func vendorLogin(t *testing.T, h http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	body := `{"username":"` + username + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	return rec
}

func vendorSessionCookieFrom(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "vendor_session" {
			return c
		}
	}
	return nil
}

func loginAsVendor(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	cookie := vendorSessionCookieFrom(vendorLogin(t, h, "akbar", testVendorPassword))
	if cookie == nil {
		t.Fatal("login vendor gagal menerbitkan cookie")
	}
	return cookie
}

func TestVendorLoginBerhasilMenerbitkanCookie(t *testing.T) {
	h := newAPIWithVendor(t)
	rec := vendorLogin(t, h, "akbar", testVendorPassword)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if vendorSessionCookieFrom(rec) == nil {
		t.Fatal("cookie vendor_session tidak ditemukan")
	}
}

func TestVendorSesiTidakBisaDipakaiSebagaiSesiCustomer(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)
	cookie.Name = "admin_session" // seandainya dipasang manual, harus tetap ditolak

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 -- sesi vendor tidak boleh valid sebagai sesi customer", rec.Code)
	}
}

func TestVendorCreateAccountDanReveal(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"business_name":"Toko Contoh","email":"toko@contoh.test","username":"toko_contoh","plan":"Business","expires_at":"2027-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201 (body=%s)", rec.Code, rec.Body.String())
	}
	var out struct {
		Success         bool   `json:"success"`
		InitialPassword string `json:"initial_password"`
		Account         struct {
			ID         string `json:"id"`
			MaxDevices int    `json:"max_devices"`
			Status     string `json:"status"`
		} `json:"account"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.InitialPassword == "" {
		t.Fatal("initial_password kosong")
	}
	if out.Account.MaxDevices != 10 {
		t.Fatalf("MaxDevices = %d, mau 10 (plan Business)", out.Account.MaxDevices)
	}
	if out.Account.Status != "active" {
		t.Fatalf("Status = %q, mau active", out.Account.Status)
	}
}

func TestVendorListAccounts(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"business_name":"Toko A","email":"a@t.test","username":"toko_a","plan":"Starter","expires_at":"2027-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(httptest.NewRecorder(), req)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/accounts", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	h.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", listRec.Code, listRec.Body.String())
	}
	var out struct {
		Accounts []struct {
			Username string `json:"username"`
		} `json:"accounts"`
	}
	json.Unmarshal(listRec.Body.Bytes(), &out)
	if len(out.Accounts) != 1 || out.Accounts[0].Username != "toko_a" {
		t.Fatalf("accounts = %+v, mau 1 baris toko_a", out.Accounts)
	}
}

func TestVendorRenewSuspendRevokeAccount(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	createBody := `{"business_name":"Toko B","email":"b@t.test","username":"toko_b","plan":"Starter","expires_at":"2026-01-01"}`
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
	id := created.Account.ID

	renewReq := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts/"+id+"/renew",
		strings.NewReader(`{"expires_at":"2028-01-01"}`))
	renewReq.Header.Set("Content-Type", "application/json")
	renewReq.AddCookie(cookie)
	renewRec := httptest.NewRecorder()
	h.ServeHTTP(renewRec, renewReq)
	if renewRec.Code != http.StatusOK {
		t.Fatalf("renew status = %d (body=%s)", renewRec.Code, renewRec.Body.String())
	}

	suspendReq := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts/"+id+"/suspend", nil)
	suspendReq.AddCookie(cookie)
	suspendRec := httptest.NewRecorder()
	h.ServeHTTP(suspendRec, suspendReq)
	if suspendRec.Code != http.StatusOK {
		t.Fatalf("suspend status = %d (body=%s)", suspendRec.Code, suspendRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/accounts/"+id, nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	var got struct {
		Account struct {
			Status    string `json:"status"`
			ExpiresAt string `json:"expires_at"`
		} `json:"account"`
	}
	json.Unmarshal(getRec.Body.Bytes(), &got)
	if got.Account.Status != "suspended" {
		t.Fatalf("Status = %q, mau suspended", got.Account.Status)
	}
	if got.Account.ExpiresAt != "2028-01-01" {
		t.Fatalf("ExpiresAt = %q, mau 2028-01-01", got.Account.ExpiresAt)
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts/"+id+"/revoke", nil)
	revokeReq.AddCookie(cookie)
	revokeRec := httptest.NewRecorder()
	h.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d (body=%s)", revokeRec.Code, revokeRec.Body.String())
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/audit-log", nil)
	auditReq.AddCookie(cookie)
	auditRec := httptest.NewRecorder()
	h.ServeHTTP(auditRec, auditReq)
	var audit struct {
		Entries []struct {
			Action string `json:"action"`
		} `json:"entries"`
	}
	json.Unmarshal(auditRec.Body.Bytes(), &audit)
	if len(audit.Entries) < 4 { // CREATED, RENEWED, SUSPENDED, REVOKED (+ akun lain di test lain kalau ada)
		t.Fatalf("audit entries = %+v, mau minimal 4", audit.Entries)
	}
}

func TestVendorAccountsButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vendor/accounts", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestVendorCreateAccountEmailBentrokMengembalikan409(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body1 := `{"business_name":"Toko A","email":"bentrok@uji.test","username":"toko_a","plan":"Starter","expires_at":"2027-01-01"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.AddCookie(cookie)
	h.ServeHTTP(httptest.NewRecorder(), req1)

	body2 := `{"business_name":"Toko B","email":"bentrok@uji.test","username":"toko_b","plan":"Starter","expires_at":"2027-01-01"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec2.Code, rec2.Body.String())
	}
	var out struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec2.Body.Bytes(), &out)
	if out.Error != "email_taken" {
		t.Fatalf("error = %q, mau email_taken", out.Error)
	}
}
