package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestVendorWebhookDeliveriesButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/webhook-deliveries", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 tanpa cookie vendor_session", rec.Code)
	}
}

// seedWebhookDelivery membuat endpoint + satu delivery untuk satu account,
// langsung lewat store (alur aslinya dipicu worker webhook, yang tidak
// dijalankan di test ini).
func seedWebhookDelivery(t *testing.T, accountID, endpointName, status string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	endpointID, err := store.NewWebhookID()
	if err != nil {
		t.Fatalf("NewWebhookID: %v", err)
	}
	_, secret, err := store.GenerateWebhookSecret()
	if err != nil {
		t.Fatalf("GenerateWebhookSecret: %v", err)
	}
	if err := s.CreateWebhookEndpoint(ctx, webhookSecretKey(), accountID, endpointID, endpointName,
		"https://"+endpointName+".example.test/hook", []string{store.WebhookEventInvoicePaid}, secret); err != nil {
		t.Fatalf("CreateWebhookEndpoint: %v", err)
	}

	if _, err := s.Pool().Exec(ctx,
		`INSERT INTO webhook_deliveries
		   (id, account_id, endpoint_id, event, invoice_id, payload, status, attempt,
		    http_status, duration_ms, created_at)
		 VALUES ($1, $2, $3, $4, NULL, '{}', $5, 1, 500, 1200, $6)`,
		"whd_"+endpointID, accountID, endpointID, store.WebhookEventInvoicePaid, status, time.Now()); err != nil {
		t.Fatalf("insert delivery: %v", err)
	}
}

func TestVendorWebhookDeliveriesLintasAccount(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	ids := make([]string, 0, 2)
	for _, biz := range []struct{ name, email, username string }{
		{"Toko Webhook A", "wha@t.test", "toko_wh_a"},
		{"Toko Webhook B", "whb@t.test", "toko_wh_b"},
	} {
		body := `{"business_name":"` + biz.name + `","email":"` + biz.email +
			`","username":"` + biz.username + `","plan":"Starter","expires_at":"2027-01-01"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var created struct {
			Account struct {
				ID string `json:"id"`
			} `json:"account"`
		}
		json.Unmarshal(rec.Body.Bytes(), &created)
		ids = append(ids, created.Account.ID)
	}

	seedWebhookDelivery(t, ids[0], "hook-a", store.WebhookDeliveryFailed)
	seedWebhookDelivery(t, ids[1], "hook-b", store.WebhookDeliveryDelivered)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/webhook-deliveries", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Deliveries []struct {
			BusinessName string `json:"business_name"`
			EndpointName string `json:"endpoint_name"`
			Status       string `json:"status"`
		} `json:"deliveries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Deliveries) != 2 {
		t.Fatalf("deliveries = %+v, mau 2 (lintas account)", body.Deliveries)
	}
	if strings.Contains(rec.Body.String(), `"payload"`) {
		t.Fatal("payload sengaja tidak diambil untuk daftar lintas-account")
	}
}

func TestVendorWebhookDeliveriesFilterStatus(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"business_name":"Toko Filter","email":"f@t.test","username":"toko_filter","plan":"Starter","expires_at":"2027-01-01"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
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

	seedWebhookDelivery(t, created.Account.ID, "hook-gagal", store.WebhookDeliveryFailed)
	seedWebhookDelivery(t, created.Account.ID, "hook-sukses", store.WebhookDeliveryDelivered)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/webhook-deliveries?status=FAILED", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var out struct {
		Deliveries []struct {
			EndpointName string `json:"endpoint_name"`
			Status       string `json:"status"`
		} `json:"deliveries"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Deliveries) != 1 || out.Deliveries[0].Status != store.WebhookDeliveryFailed {
		t.Fatalf("deliveries = %+v, mau 1 baris FAILED", out.Deliveries)
	}
}

func TestVendorWebhookDeliveriesStatusTidakDikenalDitolak(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/webhook-deliveries?status=NGAWUR", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}
