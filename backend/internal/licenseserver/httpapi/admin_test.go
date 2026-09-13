package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testVendorPassword = "password-vendor-test"

func vendorLogin(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	body := `{"username":"vendor","password":"` + testVendorPassword + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login gagal: %d %s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "vendor_session" {
			return c
		}
	}
	t.Fatal("cookie vendor_session tidak ditemukan")
	return nil
}

func TestAdminEndpointTanpaSesiDitolak(t *testing.T) {
	api, _ := newTestAPI(t)
	h := api.Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/customers", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestCreateAndListCustomer(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	createTestVendorAdmin(t, s, "vendor", testVendorPassword)
	cookie := vendorLogin(t, h)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/customers", strings.NewReader(`{"name":"Toko Baru"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201 (body=%s)", rec.Code, rec.Body.String())
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/customers", nil)
	listReq.AddCookie(cookie)
	h.ServeHTTP(listRec, listReq)

	var body struct {
		Customers []struct {
			Name string `json:"name"`
		} `json:"customers"`
	}
	json.Unmarshal(listRec.Body.Bytes(), &body)
	if len(body.Customers) != 1 || body.Customers[0].Name != "Toko Baru" {
		t.Fatalf("customers = %+v", body.Customers)
	}
}

func TestCreateLicenseMenampilkanKeyMentahSekali(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	createTestVendorAdmin(t, s, "vendor", testVendorPassword)
	cookie := vendorLogin(t, h)

	custID, _ := createCustomerViaAPI(t, h, cookie, "Toko Baru")

	rec := httptest.NewRecorder()
	body := `{"plan":"Business","expires_at":"2027-09-13"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/customers/"+custID+"/licenses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201 (body=%s)", rec.Code, rec.Body.String())
	}

	var out struct {
		License struct {
			Key        string `json:"key"`
			MaxDevices int    `json:"max_devices"`
			Status     string `json:"status"`
		} `json:"license"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(out.License.Key, "PB-BUSINESS-") {
		t.Fatalf("key = %q, mau berawalan PB-BUSINESS-", out.License.Key)
	}
	if out.License.MaxDevices != 10 {
		t.Fatalf("max_devices = %d, mau 10 (preset Business)", out.License.MaxDevices)
	}
	if out.License.Status != "active" {
		t.Fatalf("status = %q, mau active", out.License.Status)
	}
}

func TestCreateLicensePlanTidakDikenalDitolak(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	createTestVendorAdmin(t, s, "vendor", testVendorPassword)
	cookie := vendorLogin(t, h)
	custID, _ := createCustomerViaAPI(t, h, cookie, "Toko Baru")

	rec := httptest.NewRecorder()
	body := `{"plan":"TidakAda","expires_at":"2027-09-13"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/customers/"+custID+"/licenses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", rec.Code)
	}
}

func TestRenewSuspendRevokeLicense(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	createTestVendorAdmin(t, s, "vendor", testVendorPassword)
	cookie := vendorLogin(t, h)
	custID, _ := createCustomerViaAPI(t, h, cookie, "Toko Baru")
	licID := createLicenseViaAPI(t, h, cookie, custID, "Business", "2027-09-13")

	// Renew
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/licenses/"+licID+"/renew",
		strings.NewReader(`{"expires_at":"2028-09-13"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("renew status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	// Suspend
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/licenses/"+licID+"/suspend", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspend status = %d, mau 200", rec.Code)
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/licenses/"+licID, nil)
	getReq.AddCookie(cookie)
	h.ServeHTTP(getRec, getReq)
	var getBody struct {
		License struct {
			Status    string `json:"status"`
			ExpiresAt string `json:"expires_at"`
		} `json:"license"`
	}
	json.Unmarshal(getRec.Body.Bytes(), &getBody)
	if getBody.License.Status != "suspended" {
		t.Fatalf("status = %q, mau suspended", getBody.License.Status)
	}
	if getBody.License.ExpiresAt != "2028-09-13" {
		t.Fatalf("expires_at = %q, mau 2028-09-13 (dari renew sebelumnya)", getBody.License.ExpiresAt)
	}

	// Revoke
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/licenses/"+licID+"/revoke", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, mau 200", rec.Code)
	}
}

func TestResetInstallationViaAdminAPI(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	createTestVendorAdmin(t, s, "vendor", testVendorPassword)
	cookie := vendorLogin(t, h)
	custID, _ := createCustomerViaAPI(t, h, cookie, "Toko Baru")
	rawKey := createLicenseViaAPIWithKey(t, h, cookie, custID, "Business", "2027-09-13")

	actRec := doActivate(t, h, rawKey, "production")
	var actBody struct {
		InstallationID string `json:"installation_id"`
	}
	json.Unmarshal(actRec.Body.Bytes(), &actBody)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/installations/"+actBody.InstallationID+"/reset", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestListAuditLogMencatatAksiVendor(t *testing.T) {
	api, s := newTestAPI(t)
	h := api.Handler()
	createTestVendorAdmin(t, s, "vendor", testVendorPassword)
	cookie := vendorLogin(t, h)
	createCustomerViaAPI(t, h, cookie, "Toko Baru")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-log", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}

	var body struct {
		Entries []struct {
			Action string `json:"action"`
		} `json:"entries"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	found := false
	for _, e := range body.Entries {
		if e.Action == "CUSTOMER_CREATED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("audit log tidak memuat CUSTOMER_CREATED: %+v", body.Entries)
	}
}

// --- helper -----------------------------------------------------------

func createCustomerViaAPI(t *testing.T, h http.Handler, cookie *http.Cookie, name string) (id string, rec *httptest.ResponseRecorder) {
	t.Helper()
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/customers", strings.NewReader(`{"name":"`+name+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create customer gagal: %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Customer struct {
			ID string `json:"id"`
		} `json:"customer"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Customer.ID, rec
}

// createLicenseViaAPIWithKey adalah createLicenseViaAPI yang juga
// mengembalikan license key MENTAH — cuma ada di response create, tidak
// bisa diambil lagi setelahnya (sama seperti API key/webhook secret di
// backend customer).
func createLicenseViaAPIWithKey(t *testing.T, h http.Handler, cookie *http.Cookie, customerID, plan, expiresAt string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	body := `{"plan":"` + plan + `","expires_at":"` + expiresAt + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/customers/"+customerID+"/licenses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create license gagal: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		License struct {
			Key string `json:"key"`
		} `json:"license"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	return out.License.Key
}

func createLicenseViaAPI(t *testing.T, h http.Handler, cookie *http.Cookie, customerID, plan, expiresAt string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	body := `{"plan":"` + plan + `","expires_at":"` + expiresAt + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/customers/"+customerID+"/licenses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create license gagal: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		License struct {
			ID string `json:"id"`
		} `json:"license"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	return out.License.ID
}
