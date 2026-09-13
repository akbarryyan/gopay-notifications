package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// newAPIWithAPIKey menyiapkan API lengkap dengan satu API key merchant aktif.
// Mengembalikan handler dan key mentahnya (dipakai di header Authorization).
func newAPIWithAPIKey(t *testing.T) (http.Handler, string) {
	t.Helper()

	s := newTestStore(t)
	ctx := context.Background()
	seedActiveAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	id, err := store.NewAPIKeyID()
	if err != nil {
		t.Fatalf("NewAPIKeyID: %v", err)
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret: %v", err)
	}
	if err := s.CreateAPIKey(ctx, "acc_1", id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), func() time.Time { return fixedNow }).Handler()
	return h, rawKey
}

func createInvoiceReq(t *testing.T, h http.Handler, apiKey, externalRef string, amount int64) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"external_ref":"` + externalRef + `","amount":` + strconv.FormatInt(amount, 10) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type invoiceBody struct {
	ID              string  `json:"id"`
	ExternalRef     string  `json:"external_ref"`
	RequestedAmount int64   `json:"requested_amount"`
	UniqueAmount    int64   `json:"unique_amount"`
	Status          string  `json:"status"`
	MatchedEventID  *string `json:"matched_event_id"`
}

func decodeInvoice(t *testing.T, rec *httptest.ResponseRecorder) invoiceBody {
	t.Helper()
	var body invoiceBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}
	return body
}

func TestCreateInvoiceBerhasil(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	rec := createInvoiceReq(t, h, apiKey, "ORDER-1", 50000)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201 (body=%s)", rec.Code, rec.Body.String())
	}

	inv := decodeInvoice(t, rec)
	if inv.Status != "PENDING" {
		t.Fatalf("Status = %s, mau PENDING", inv.Status)
	}
	if inv.UniqueAmount <= 50000 || inv.UniqueAmount > 50999 {
		t.Fatalf("UniqueAmount = %d, mau di rentang 50001..50999", inv.UniqueAmount)
	}
}

func TestCreateInvoiceExternalRefSamaAmountSamaMengembalikan200(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	first := createInvoiceReq(t, h, apiKey, "ORDER-1", 50000)
	if first.Code != http.StatusCreated {
		t.Fatalf("status pertama = %d, mau 201", first.Code)
	}
	firstInv := decodeInvoice(t, first)

	second := createInvoiceReq(t, h, apiKey, "ORDER-1", 50000)
	if second.Code != http.StatusOK {
		t.Fatalf("status kedua = %d, mau 200 (retry aman)", second.Code)
	}
	secondInv := decodeInvoice(t, second)
	if secondInv.ID != firstInv.ID {
		t.Fatalf("ID kedua = %s, mau sama dengan pertama %s", secondInv.ID, firstInv.ID)
	}
}

func TestCreateInvoiceExternalRefSamaAmountBeda409(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	createInvoiceReq(t, h, apiKey, "ORDER-1", 50000)
	rec := createInvoiceReq(t, h, apiKey, "ORDER-1", 75000)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409", rec.Code)
	}
}

func TestCreateInvoiceTanpaAPIKeyDitolak(t *testing.T) {
	h, _ := newAPIWithAPIKey(t)

	rec := createInvoiceReq(t, h, "", "ORDER-1", 50000)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestCreateInvoiceAPIKeySalahDitolak(t *testing.T) {
	h, _ := newAPIWithAPIKey(t)

	rec := createInvoiceReq(t, h, "sk_bukan-key-yang-benar", "ORDER-1", 50000)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestCreateInvoiceAmountNolAtauNegatifDitolak(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	for _, amount := range []int64{0, -1} {
		rec := createInvoiceReq(t, h, apiKey, "ORDER-1", amount)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("amount=%d: status = %d, mau 400", amount, rec.Code)
		}
	}
}

func TestGetInvoiceBerhasil(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	created := decodeInvoice(t, createInvoiceReq(t, h, apiKey, "ORDER-1", 50000))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	got := decodeInvoice(t, rec)
	if got.ID != created.ID {
		t.Fatalf("ID = %s, mau %s", got.ID, created.ID)
	}
}

func TestGetInvoiceTidakDitemukan(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/inv_tidak_ada", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}

// TestInvoiceCocokLewatCallback menguji integrasi ujung-ke-ujung: invoice
// dibuat lewat API merchant, lalu event dengan amount_hint yang sama masuk
// lewat POST /api/v1/events (jalur device, HMAC) — matching harus otomatis
// terjadi tanpa langkah tambahan apa pun.
func TestInvoiceCocokLewatCallback(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	inv := decodeInvoice(t, createInvoiceReq(t, h, apiKey, "ORDER-1", 50000))

	body := `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"cocok","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":` + strconv.FormatInt(inv.UniqueAmount, 10) + `,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`

	rec := postCallback(t, h, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status callback = %d (body=%s)", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+inv.ID, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, req)
	got := decodeInvoice(t, getRec)

	if got.Status != "PAID" {
		t.Fatalf("Status = %s, mau PAID setelah event cocok masuk", got.Status)
	}
	if got.MatchedEventID == nil || *got.MatchedEventID != "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c" {
		t.Fatalf("MatchedEventID = %v, mau evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c", got.MatchedEventID)
	}
}
