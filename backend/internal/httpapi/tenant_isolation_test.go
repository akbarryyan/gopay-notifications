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

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// newAPITwoAccounts menyiapkan API dengan dua akun terpisah (acc_a/acc_b),
// masing-masing dengan device, api key, dan webhook endpoint sendiri --
// dipakai seluruh test isolasi di file ini.
type twoAccountsFixture struct {
	h                  http.Handler
	cookieA, cookieB   *http.Cookie
	apiKeyA, apiKeyB   string
	deviceA, deviceB   string
	webhookA, webhookB string
}

func newAPITwoAccounts(t *testing.T) twoAccountsFixture {
	t.Helper()
	s := newTestStore(t)
	ctx := context.Background()

	mustAccount := func(id, username string) {
		if err := s.CreateAccount(ctx, store.CreateAccountInput{
			ID: id, BusinessName: id, Email: id + "@uji.test", Username: username,
			PlaintextPassword: "rahasia123", Plan: "Business", MaxDevices: 10,
			ExpiresAt: fixedNow.Add(365 * 24 * time.Hour),
		}); err != nil {
			t.Fatalf("create account %s: %v", id, err)
		}
	}
	mustAccount("acc_a", "user_a")
	mustAccount("acc_b", "user_b")

	if err := s.CreateDevice(ctx, encKey(), "acc_a", "dev_a1", "HP A1", []byte("secret-a")); err != nil {
		t.Fatalf("create device a: %v", err)
	}
	if err := s.CreateDevice(ctx, encKey(), "acc_b", "dev_b1", "HP B1", []byte("secret-b")); err != nil {
		t.Fatalf("create device b: %v", err)
	}

	keyIDA, _ := store.NewAPIKeyID()
	rawKeyA, hashA, _ := store.GenerateAPIKeySecret()
	if err := s.CreateAPIKey(ctx, "acc_a", keyIDA, "Key A", hashA); err != nil {
		t.Fatalf("create api key a: %v", err)
	}
	keyIDB, _ := store.NewAPIKeyID()
	rawKeyB, hashB, _ := store.GenerateAPIKeySecret()
	if err := s.CreateAPIKey(ctx, "acc_b", keyIDB, "Key B", hashB); err != nil {
		t.Fatalf("create api key b: %v", err)
	}

	whIDA, _ := store.NewWebhookID()
	_, secretA, _ := store.GenerateWebhookSecret()
	if err := s.CreateWebhookEndpoint(ctx, webhookSecretKey(), "acc_a", whIDA, "Endpoint A",
		"https://a.test/hook", []string{store.WebhookEventInvoicePaid}, secretA); err != nil {
		t.Fatalf("create webhook a: %v", err)
	}
	whIDB, _ := store.NewWebhookID()
	_, secretB, _ := store.GenerateWebhookSecret()
	if err := s.CreateWebhookEndpoint(ctx, webhookSecretKey(), "acc_b", whIDB, "Endpoint B",
		"https://b.test/hook", []string{store.WebhookEventInvoicePaid}, secretB); err != nil {
		t.Fatalf("create webhook b: %v", err)
	}

	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(),
		func() time.Time { return fixedNow }).Handler()

	loginAs := func(username string) *http.Cookie {
		rec := adminLogin(t, h, username, "rahasia123")
		cookie := sessionCookieFrom(rec)
		if cookie == nil {
			t.Fatalf("login %s gagal", username)
		}
		return cookie
	}

	return twoAccountsFixture{
		h:       h,
		cookieA: loginAs("user_a"), cookieB: loginAs("user_b"),
		apiKeyA: rawKeyA, apiKeyB: rawKeyB,
		deviceA: "dev_a1", deviceB: "dev_b1",
		webhookA: whIDA, webhookB: whIDB,
	}
}

// TestIsolasiDeviceLintasAccount: akun B tidak boleh bisa menonaktifkan
// device milik akun A lewat ID langsung, walau ID-nya diketahui persis.
func TestIsolasiDeviceLintasAccount(t *testing.T) {
	f := newAPITwoAccounts(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/devices/"+f.deviceA,
		strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(f.cookieB)
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 (device milik akun lain tidak boleh bisa diubah)", rec.Code)
	}

	// Daftar device akun B tidak boleh berisi device akun A sama sekali.
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/devices", nil)
	listReq.AddCookie(f.cookieB)
	listRec := httptest.NewRecorder()
	f.h.ServeHTTP(listRec, listReq)
	if strings.Contains(listRec.Body.String(), f.deviceA) {
		t.Fatalf("daftar device akun B memuat device akun A: %s", listRec.Body.String())
	}
}

// TestIsolasiInvoiceLintasAccount: invoice milik akun A tidak bisa diakses
// lewat API key akun B, walau ID invoice-nya diketahui persis.
func TestIsolasiInvoiceLintasAccount(t *testing.T) {
	f := newAPITwoAccounts(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/invoices",
		strings.NewReader(`{"external_ref":"ORDER-A","amount":50000}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+f.apiKeyA)
	createRec := httptest.NewRecorder()
	f.h.ServeHTTP(createRec, createReq)
	var inv struct {
		ID string `json:"id"`
	}
	json.Unmarshal(createRec.Body.Bytes(), &inv)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+inv.ID, nil)
	getReq.Header.Set("Authorization", "Bearer "+f.apiKeyB)
	getRec := httptest.NewRecorder()
	f.h.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 (invoice milik akun lain)", getRec.Code)
	}
}

// TestIsolasiAPIKeyTidakBisaDisuntikDariBody: account_id yang dipakai
// menyimpan invoice SELALU diturunkan dari API key yang terautentikasi,
// bukan dari field apa pun di body request -- membuktikan handler tidak
// pernah membaca account_id dari input client.
func TestIsolasiAPIKeyTidakBisaDisuntikDariBody(t *testing.T) {
	f := newAPITwoAccounts(t)

	// Body ini bahkan tidak punya field account_id (createInvoiceRequest
	// tidak mendefinisikannya sama sekali) -- kalau pun disisipkan di JSON,
	// json.Decoder mengabaikan field tak dikenal begitu saja.
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/invoices",
		strings.NewReader(`{"external_ref":"ORDER-X","amount":30000,"account_id":"acc_b"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+f.apiKeyA)
	createRec := httptest.NewRecorder()
	f.h.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("status = %d (body=%s)", createRec.Code, createRec.Body.String())
	}
	var inv struct {
		ID string `json:"id"`
	}
	json.Unmarshal(createRec.Body.Bytes(), &inv)

	// Invoice itu harus kelihatan lewat sesi akun A (bukan akun B walau
	// "account_id":"acc_b" ada di body request tadi).
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices", nil)
	listReq.AddCookie(f.cookieA)
	listRec := httptest.NewRecorder()
	f.h.ServeHTTP(listRec, listReq)
	if !strings.Contains(listRec.Body.String(), "ORDER-X") {
		t.Fatalf("invoice ORDER-X seharusnya muncul di akun A: %s", listRec.Body.String())
	}

	listReqB := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices", nil)
	listReqB.AddCookie(f.cookieB)
	listRecB := httptest.NewRecorder()
	f.h.ServeHTTP(listRecB, listReqB)
	if strings.Contains(listRecB.Body.String(), "ORDER-X") {
		t.Fatalf("invoice ORDER-X TIDAK boleh muncul di akun B: %s", listRecB.Body.String())
	}
}

// TestIsolasiWebhookLintasAccount: akun B tidak bisa menghapus atau
// men-test-kirim webhook endpoint milik akun A.
func TestIsolasiWebhookLintasAccount(t *testing.T) {
	f := newAPITwoAccounts(t)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/webhooks/"+f.webhookA, nil)
	delReq.AddCookie(f.cookieB)
	delRec := httptest.NewRecorder()
	f.h.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNotFound {
		t.Fatalf("delete status = %d, mau 404 (webhook milik akun lain)", delRec.Code)
	}

	testReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/webhooks/"+f.webhookA+"/test", nil)
	testReq.AddCookie(f.cookieB)
	testRec := httptest.NewRecorder()
	f.h.ServeHTTP(testRec, testReq)
	if testRec.Code != http.StatusNotFound {
		t.Fatalf("test-delivery status = %d, mau 404 (webhook milik akun lain)", testRec.Code)
	}
}

// TestIsolasiEventLintasAccountLewatDeviceHMAC: event yang masuk lewat
// device HMAC akun A tidak boleh terlihat di daftar event akun B.
func TestIsolasiEventLintasAccountLewatDeviceHMAC(t *testing.T) {
	f := newAPITwoAccounts(t)

	body := `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"` + f.deviceA + `",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"cocok","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":12345,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`

	ts := fixedNow.Unix()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Id", f.deviceA)
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))
	sig := auth.Sign([]byte("secret-a"), auth.SigningString(f.deviceA, ts, []byte(body)))
	req.Header.Set("X-Signature", sig)
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("callback status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	listB := httptest.NewRequest(http.MethodGet, "/api/v1/admin/events", nil)
	listB.AddCookie(f.cookieB)
	recB := httptest.NewRecorder()
	f.h.ServeHTTP(recB, listB)
	if strings.Contains(recB.Body.String(), "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c") {
		t.Fatalf("event milik akun A muncul di daftar akun B: %s", recB.Body.String())
	}

	listA := httptest.NewRequest(http.MethodGet, "/api/v1/admin/events", nil)
	listA.AddCookie(f.cookieA)
	recA := httptest.NewRecorder()
	f.h.ServeHTTP(recA, listA)
	if !strings.Contains(recA.Body.String(), "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c") {
		t.Fatalf("event seharusnya muncul di daftar akun A sendiri: %s", recA.Body.String())
	}
}
