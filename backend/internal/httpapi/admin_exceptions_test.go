package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// newAPIWithAdminAndAPIKey menyiapkan API dengan device + API key merchant
// + akun admin sekaligus — dipakai test yang perlu keduanya (membuat
// invoice lewat API key, lalu memeriksa/bertindak lewat sesi admin).
func newAPIWithAdminAndAPIKey(t *testing.T) (http.Handler, string) {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL belum diset. Jalankan: make db-up migrate")
	}

	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)

	if _, err := s.Pool().Exec(ctx,
		"TRUNCATE notification_events, event_reviews, invoices, api_keys, webhook_deliveries, webhook_endpoints, devices, admin_users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.UpsertAdmin(ctx, "admin", testAdminPassword); err != nil {
		t.Fatalf("UpsertAdmin: %v", err)
	}

	keyID, err := store.NewAPIKeyID()
	if err != nil {
		t.Fatalf("NewAPIKeyID: %v", err)
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret: %v", err)
	}
	if err := s.CreateAPIKey(ctx, keyID, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), activeLicense(), func() time.Time { return fixedNow }).Handler()
	return h, rawKey
}

type exceptionEventBody struct {
	EventID    string `json:"event_id"`
	AmountHint *int64 `json:"amount_hint"`
}

func listExceptions(t *testing.T, h http.Handler, cookie *http.Cookie) []exceptionEventBody {
	t.Helper()
	rec := adminGet(t, h, cookie, "/api/v1/admin/exceptions")
	if rec.Code != http.StatusOK {
		t.Fatalf("listExceptions: status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Exceptions []exceptionEventBody `json:"exceptions"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Exceptions
}

func containsEventID(list []exceptionEventBody, eventID string) bool {
	for _, e := range list {
		if e.EventID == eventID {
			return true
		}
	}
	return false
}

func TestAdminExceptionsMenampilkanEventTakCocok(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.CreateDevice(context.Background(), encKey(), "dev_seed", "HP Seed", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	amount := int64(41414)
	title := "Pembayaran QRIS statis diterima"
	if _, err := s.InsertEvent(context.Background(), store.Event{
		EventID:     "evt_00000000000000000000000000000099",
		DeviceID:    "dev_seed",
		Source:      "gopay",
		PackageName: "com.gojek.gopaymerchant",
		Title:       &title,
		AmountHint:  &amount,
		PostedAt:    time.Now(),
		ReceivedAt:  time.Now(),
		RawPayload:  []byte(`{}`),
	}); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	list := listExceptions(t, h, cookie)
	if !containsEventID(list, "evt_00000000000000000000000000000099") {
		t.Fatalf("exceptions = %+v, mau memuat event yang baru disisipkan", list)
	}
}

func TestAdminExceptionsMemerlukanSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/exceptions", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

// TestAdminMatchExceptionUjungKeUjung menguji integrasi penuh: event masuk
// tak cocok -> muncul di exceptions -> dicocokkan manual -> invoice PAID +
// webhook invoice.paid terkirim -> event hilang dari daftar exceptions.
func TestAdminMatchExceptionUjungKeUjung(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fake.Close()

	h, apiKey := newAPIWithAdminAndAPIKey(t)
	cookie := loginAsAdmin(t, h)
	registerWebhook(t, h, cookie, fake.URL, []string{"invoice.paid"})

	inv := decodeInvoice(t, createInvoiceReq(t, h, apiKey, "ORDER-exc-1", 30000))

	// Event masuk dengan nominal BEDA dari unique_amount invoice (customer
	// salah ketik) — matching otomatis tidak akan pernah cocok, jadi
	// mengendap sebagai exception.
	body := `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"salah ketik","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":999999,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`
	if rec := postCallback(t, h, body); rec.Code != http.StatusOK {
		t.Fatalf("status callback = %d (body=%s)", rec.Code, rec.Body.String())
	}

	list := listExceptions(t, h, cookie)
	if !containsEventID(list, "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c") {
		t.Fatalf("exceptions = %+v, mau memuat evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c", list)
	}

	matchBody := `{"invoice_id":"` + inv.ID + `"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/exceptions/evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c/match", strings.NewReader(matchBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status match = %d (body=%s)", rec.Code, rec.Body.String())
	}

	// Invoice harus PAID sekarang.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+inv.ID, nil)
	getReq.Header.Set("Authorization", "Bearer "+apiKey)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	got := decodeInvoice(t, getRec)
	if got.Status != "PAID" {
		t.Fatalf("Status invoice = %s, mau PAID setelah dicocokkan manual", got.Status)
	}

	// Event tidak boleh lagi muncul sebagai exception.
	list = listExceptions(t, h, cookie)
	if containsEventID(list, "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c") {
		t.Fatalf("exceptions = %+v, event yang sudah dicocokkan seharusnya sudah hilang", list)
	}

	// Webhook invoice.paid harus terkirim — tunggu sebentar (async).
	webhooks := adminGet(t, h, cookie, "/api/v1/admin/webhooks")
	var wbody struct {
		Webhooks []webhookBody `json:"webhooks"`
	}
	json.Unmarshal(webhooks.Body.Bytes(), &wbody)
	if len(wbody.Webhooks) != 1 {
		t.Fatalf("webhooks = %+v, mau 1", wbody.Webhooks)
	}
	_ = waitForNonPendingDelivery(t, h, cookie, wbody.Webhooks[0].ID, 5*time.Second)
}

func TestAdminMatchExceptionInvoiceSudahPaid(t *testing.T) {
	h, apiKey := newAPIWithAdminAndAPIKey(t)
	cookie := loginAsAdmin(t, h)

	inv := decodeInvoice(t, createInvoiceReq(t, h, apiKey, "ORDER-exc-2", 30000))

	body := `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"cocok","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":` + strconv.FormatInt(inv.UniqueAmount, 10) + `,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`
	if rec := postCallback(t, h, body); rec.Code != http.StatusOK {
		t.Fatalf("status callback = %d (body=%s)", rec.Code, rec.Body.String())
	}

	// Invoice sudah PAID lewat matching otomatis. Event lain mencoba
	// mencocokkan manual ke invoice yang sama — harus ditolak.
	secondBody := `{"event_id":"evt_00000000000000000000000000000088","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"lain","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":123456,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`
	if rec := postCallback(t, h, secondBody); rec.Code != http.StatusOK {
		t.Fatalf("status callback kedua = %d (body=%s)", rec.Code, rec.Body.String())
	}

	matchBody := `{"invoice_id":"` + inv.ID + `"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/exceptions/evt_00000000000000000000000000000088/match", strings.NewReader(matchBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (invoice sudah PAID)", rec.Code)
	}
}

func TestAdminDismissExceptionBerhasilDanTidakBisaDuaKali(t *testing.T) {
	h, _ := newAPIWithAdminAndAPIKey(t)
	cookie := loginAsAdmin(t, h)

	body := `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"tak cocok","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":77777,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`
	if rec := postCallback(t, h, body); rec.Code != http.StatusOK {
		t.Fatalf("status callback = %d (body=%s)", rec.Code, rec.Body.String())
	}

	dismissBody := `{"note":"transfer pribadi, bukan order"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/exceptions/evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c/dismiss", strings.NewReader(dismissBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status dismiss pertama = %d (body=%s)", rec.Code, rec.Body.String())
	}

	list := listExceptions(t, h, cookie)
	if containsEventID(list, "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c") {
		t.Fatalf("exceptions = %+v, event yang sudah dismiss seharusnya sudah hilang", list)
	}

	req2 := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/exceptions/evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c/dismiss", strings.NewReader(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("status dismiss kedua = %d, mau 409 (sudah pernah di-dismiss)", rec2.Code)
	}
}

func TestAdminDismissExceptionTidakDitemukan(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/exceptions/evt_tidak_ada/dismiss", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
}
