package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type adminInvoicesResponse struct {
	Invoices []struct {
		ID          string `json:"id"`
		ExternalRef string `json:"external_ref"`
		Status      string `json:"status"`
	} `json:"invoices"`
}

func TestAdminInvoicesMenampilkanYangSungguhanAda(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	// Ditulis lewat koneksi store terpisah, pola yang sama dengan
	// TestAdminOverviewMenghitungDeviceDanEventSungguhan — handler admin
	// harus melihatnya tanpa jalur HTTP invoice sama sekali.
	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if _, _, err := s.CreateInvoice(context.Background(), time.Now(), "acc_1", "ORDER-1", 50000); err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	rec := adminGet(t, h, cookie, "/api/v1/admin/invoices")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body adminInvoicesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Invoices) != 1 || body.Invoices[0].ExternalRef != "ORDER-1" {
		t.Fatalf("Invoices = %+v, mau 1 baris ORDER-1", body.Invoices)
	}
}

func TestAdminInvoicesFilterStatus(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if _, _, err := s.CreateInvoice(context.Background(), time.Now(), "acc_1", "ORDER-1", 50000); err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	rec := adminGet(t, h, cookie, "/api/v1/admin/invoices?status=PAID")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body adminInvoicesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Invoices) != 0 {
		t.Fatalf("Invoices dengan status=PAID = %+v, mau kosong (invoice yang ada masih PENDING)", body.Invoices)
	}
}

func TestAdminInvoicesStatusTakDikenalDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminGet(t, h, cookie, "/api/v1/admin/invoices?status=BUKAN_STATUS")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", rec.Code)
	}
}

func TestAdminInvoicesMemerlukanSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}
