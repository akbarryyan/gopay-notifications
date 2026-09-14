package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVendorOverviewButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/overview", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 tanpa cookie vendor_session", rec.Code)
	}
}

func TestVendorOverviewMeringkasAccountDanDaily(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"business_name":"Toko A","email":"a@t.test","username":"toko_a","plan":"Starter","expires_at":"2027-01-01"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	h.ServeHTTP(httptest.NewRecorder(), createReq)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/overview", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	var out struct {
		Overview struct {
			Accounts struct {
				Total int `json:"total"`
			} `json:"accounts"`
			Daily []struct {
				Date string `json:"date"`
			} `json:"daily"`
			RecentAccounts []struct {
				Username string `json:"username"`
			} `json:"recent_accounts"`
		} `json:"overview"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, rec.Body.String())
	}
	if out.Overview.Accounts.Total != 1 {
		t.Errorf("Accounts.Total = %d, mau 1", out.Overview.Accounts.Total)
	}
	if len(out.Overview.Daily) != 14 {
		t.Errorf("len(Daily) = %d, mau 14", len(out.Overview.Daily))
	}
	if len(out.Overview.RecentAccounts) != 1 || out.Overview.RecentAccounts[0].Username != "toko_a" {
		t.Fatalf("RecentAccounts = %+v, mau 1 baris toko_a", out.Overview.RecentAccounts)
	}
}

func TestVendorOverviewRecentAccountsDibatasiLimaBaris(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	for i := range 7 {
		body := fmt.Sprintf(
			`{"business_name":"Toko %d","email":"toko%d@t.test","username":"toko_%d","plan":"Starter","expires_at":"2027-01-01"}`,
			i, i, i)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/overview", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var out struct {
		Overview struct {
			RecentAccounts []struct {
				Username string `json:"username"`
			} `json:"recent_accounts"`
		} `json:"overview"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, rec.Body.String())
	}
	if len(out.Overview.RecentAccounts) != 5 {
		t.Fatalf("len(RecentAccounts) = %d, mau 5 (dibatasi, walau 7 account dibuat)", len(out.Overview.RecentAccounts))
	}
}
