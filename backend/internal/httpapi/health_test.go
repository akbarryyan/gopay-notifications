package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
)

var fixedNow = time.Unix(1789036200, 0)

func newTestAPI(t *testing.T) http.Handler {
	t.Helper()
	lic := licensecheck.License{Status: licensecheck.StatusActive}
	return httpapi.NewWithLicense(nil, nil, nil, nil, lic, func() time.Time { return fixedNow }).Handler()
}

func TestHealthReturnsOKAndServerTime(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	newTestAPI(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}

	var body struct {
		Status     string `json:"status"`
		ServerTime int64  `json:"server_time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, mau \"ok\"", body.Status)
	}
	if body.ServerTime != fixedNow.Unix() {
		t.Fatalf("server_time = %d, mau %d", body.ServerTime, fixedNow.Unix())
	}
}

func TestHealthRejectsPost(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)

	newTestAPI(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, mau 405", rec.Code)
	}
}
