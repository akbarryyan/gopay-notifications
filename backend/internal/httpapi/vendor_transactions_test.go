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

func TestVendorTransactionsButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/transactions", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 tanpa cookie vendor_session", rec.Code)
	}
}

func TestVendorTransactionsLintasAccountMenyertakanNamaBisnis(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	// Dua account lewat endpoint vendor sungguhan (bukan seed langsung),
	// supaya business_name-nya benar-benar berasal dari alur create.
	for _, biz := range []struct{ name, email, username string }{
		{"Toko Satu", "satu@t.test", "toko_satu"},
		{"Toko Dua", "dua@t.test", "toko_dua"},
	} {
		body := `{"business_name":"` + biz.name + `","email":"` + biz.email +
			`","username":"` + biz.username + `","plan":"Starter","expires_at":"2027-01-01"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	accounts, err := s.ListAccounts(ctx)
	if err != nil || len(accounts) != 2 {
		t.Fatalf("ListAccounts: %v, len=%d", err, len(accounts))
	}
	for _, acc := range accounts {
		if _, _, err := s.CreateInvoice(ctx, time.Now(), acc.ID, "ORDER-"+acc.Username, 50_000); err != nil {
			t.Fatalf("create invoice %s: %v", acc.ID, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/transactions", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Invoices []struct {
			AccountID    string `json:"account_id"`
			BusinessName string `json:"business_name"`
			ExternalRef  string `json:"external_ref"`
		} `json:"invoices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Invoices) != 2 {
		t.Fatalf("invoices = %+v, mau 2 (lintas account)", body.Invoices)
	}
	names := map[string]bool{}
	for _, inv := range body.Invoices {
		names[inv.BusinessName] = true
	}
	if !names["Toko Satu"] || !names["Toko Dua"] {
		t.Fatalf("business_name yang muncul = %+v, mau memuat Toko Satu dan Toko Dua", names)
	}
}

func TestVendorTransactionsFilterQCariNamaBisnis(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body := `{"business_name":"Toko Unik","email":"unik@t.test","username":"toko_unik","plan":"Starter","expires_at":"2027-01-01"}`
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

	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if _, _, err := s.CreateInvoice(context.Background(), time.Now(), created.Account.ID, "ORDER-X", 10_000); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/transactions?q=Unik", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out struct {
		Invoices []struct {
			BusinessName string `json:"business_name"`
		} `json:"invoices"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Invoices) != 1 || out.Invoices[0].BusinessName != "Toko Unik" {
		t.Fatalf("invoices = %+v, mau 1 baris Toko Unik", out.Invoices)
	}
}
