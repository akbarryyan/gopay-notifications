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

// newAPIForWebhookWorker menyiapkan API dengan device + API key merchant,
// dan mengembalikan *httpapi.API secara langsung (bukan cuma http.Handler)
// supaya test bisa memanggil ProcessDueWebhooks manual dengan now palsu.
func newAPIForWebhookWorker(t *testing.T) (*httpapi.API, string) {
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
		"TRUNCATE notification_events, event_reviews, invoices, api_keys, webhook_deliveries, webhook_endpoints, devices RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
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

	api := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), func() time.Time { return fixedNow })
	return api, rawKey
}

func registerWebhook(t *testing.T, h http.Handler, cookie *http.Cookie, url string, events []string) string {
	t.Helper()
	eventsJSON, _ := json.Marshal(events)
	body := `{"name":"test","url":"` + url + `","events":` + string(eventsJSON) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("registerWebhook: status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body2 webhookBody
	json.Unmarshal(rec.Body.Bytes(), &body2)
	return body2.ID
}

type webhookDeliveryRow struct {
	Event  string `json:"event"`
	Status string `json:"status"`
}

func listDeliveries(t *testing.T, h http.Handler, cookie *http.Cookie, webhookID string) []webhookDeliveryRow {
	t.Helper()
	rec := adminGet(t, h, cookie, "/api/v1/admin/webhooks/"+webhookID+"/deliveries")
	if rec.Code != http.StatusOK {
		t.Fatalf("listDeliveries: status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Deliveries []webhookDeliveryRow `json:"deliveries"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Deliveries
}

// waitForNonPendingDelivery menunggu sampai delivery pertama endpoint itu
// benar-benar selesai ditulis ke database (status bukan lagi PENDING) —
// bukan cuma menunggu server palsu menerima request-nya. Percobaan
// pertama dikirim dari goroutine terpisah (triggerInvoiceWebhook); server
// palsu menerima request SEBELUM goroutine itu sempat menuliskan hasilnya
// ke webhook_deliveries, jadi menyinkronkan lewat sinyal di sisi server
// palsu saja tidak cukup — races dengan pemanggilan ProcessDueWebhooks
// manual berikutnya di test yang sama.
func waitForNonPendingDelivery(t *testing.T, h http.Handler, cookie *http.Cookie, webhookID string, timeout time.Duration) webhookDeliveryRow {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		rows := listDeliveries(t, h, cookie, webhookID)
		if len(rows) > 0 && rows[0].Status != "PENDING" {
			return rows[0]
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("delivery endpoint %s masih PENDING setelah %s", webhookID, timeout)
	return webhookDeliveryRow{}
}

// TestWebhookInvoicePaidTerpicuOtomatisLewatCallback menguji integrasi
// ujung-ke-ujung: webhook didaftarkan, invoice dibuat, event yang cocok
// masuk lewat POST /events — webhook invoice.paid harus terkirim tanpa
// langkah tambahan apa pun (percobaan pertama sinkron di goroutine
// terpisah, jadi test menunggu sebentar sampai request ke server palsu
// selesai).
func TestWebhookInvoicePaidTerpicuOtomatisLewatCallback(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fake.Close()

	api, apiKey := newAPIForWebhookWorker(t)
	h := api.Handler()
	cookie := loginAsAdmin(t, h)
	webhookID := registerWebhook(t, h, cookie, fake.URL, []string{"invoice.paid"})

	inv := decodeInvoice(t, createInvoiceReq(t, h, apiKey, "ORDER-webhook-1", 40000))

	body := `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"cocok","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":` + strconv.FormatInt(inv.UniqueAmount, 10) + `,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`
	rec := postCallback(t, h, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status callback = %d (body=%s)", rec.Code, rec.Body.String())
	}

	delivery := waitForNonPendingDelivery(t, h, cookie, webhookID, 5*time.Second)
	if delivery.Event != "invoice.paid" {
		t.Fatalf("event yang tercatat = %q, mau invoice.paid", delivery.Event)
	}
	if delivery.Status != "DELIVERED" {
		t.Fatalf("status delivery = %q, mau DELIVERED (server palsu membalas 200)", delivery.Status)
	}
}

// TestProcessDueWebhooksMengirimInvoiceExpiredTepatSekali menguji bahwa
// invoice yang kedaluwarsa memicu invoice.expired sekali saja walau
// ProcessDueWebhooks dipanggil berkali-kali.
func TestProcessDueWebhooksMengirimInvoiceExpiredTepatSekali(t *testing.T) {
	var deliveryCount int
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deliveryCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer fake.Close()

	api, apiKey := newAPIForWebhookWorker(t)
	h := api.Handler()
	cookie := loginAsAdmin(t, h)
	registerWebhook(t, h, cookie, fake.URL, []string{"invoice.expired"})

	createInvoiceReq(t, h, apiKey, "ORDER-expired-1", 40000)

	// Invoice di atas baru dibuat dengan waktu "sekarang" (fixedNow) — supaya
	// benar-benar kedaluwarsa, panggil ProcessDueWebhooks dengan now jauh di
	// masa depan (melewati masa berlaku 15 menit).
	ctx := context.Background()
	future := fixedNow.Add(20 * time.Minute)
	if err := api.ProcessDueWebhooks(ctx, future); err != nil {
		t.Fatalf("ProcessDueWebhooks pertama: %v", err)
	}
	if deliveryCount != 1 {
		t.Fatalf("deliveryCount setelah putaran pertama = %d, mau 1", deliveryCount)
	}

	// Putaran kedua tidak boleh mengirim ulang — invoice itu sudah EXPIRED,
	// bukan "baru saja" EXPIRED lagi.
	if err := api.ProcessDueWebhooks(ctx, future.Add(1*time.Minute)); err != nil {
		t.Fatalf("ProcessDueWebhooks kedua: %v", err)
	}
	if deliveryCount != 1 {
		t.Fatalf("deliveryCount setelah putaran kedua = %d, mau tetap 1 (tidak dobel)", deliveryCount)
	}
}

// TestProcessDueWebhooksRetrySampaiBerhasil menguji retry sungguhan: server
// palsu membalas 500 di percobaan pertama (dipicu otomatis lewat callback,
// pola yang sama dengan TestWebhookInvoicePaidTerpicuOtomatisLewatCallback),
// lalu 200 di percobaan kedua yang dijalankan manual lewat ProcessDueWebhooks
// setelah backoff-nya lewat.
func TestProcessDueWebhooksRetrySampaiBerhasil(t *testing.T) {
	var attempt int
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer fake.Close()

	api, apiKey := newAPIForWebhookWorker(t)
	h := api.Handler()
	cookie := loginAsAdmin(t, h)
	webhookID := registerWebhook(t, h, cookie, fake.URL, []string{"invoice.paid"})

	inv := decodeInvoice(t, createInvoiceReq(t, h, apiKey, "ORDER-retry-1", 40000))

	body := `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
		`"title":"Pembayaran QRIS statis diterima","text":"cocok","big_text":null,"posted_at":1789051832829},` +
		`"amount_hint":` + strconv.FormatInt(inv.UniqueAmount, 10) + `,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`
	if rec := postCallback(t, h, body); rec.Code != http.StatusOK {
		t.Fatalf("status callback = %d (body=%s)", rec.Code, rec.Body.String())
	}

	// Menunggu sampai delivery BENAR-BENAR selesai ditulis RETRYING di
	// database (bukan cuma menunggu server palsu menerima request) — baru
	// setelah itu aman memanggil ProcessDueWebhooks manual tanpa balapan
	// dengan tulisan goroutine percobaan pertama yang masih berjalan.
	delivery := waitForNonPendingDelivery(t, h, cookie, webhookID, 5*time.Second)
	if delivery.Status != "RETRYING" {
		t.Fatalf("status setelah percobaan pertama = %q, mau RETRYING (gagal 500)", delivery.Status)
	}
	if attempt != 1 {
		t.Fatalf("attempt setelah percobaan pertama = %d, mau 1 (gagal 500)", attempt)
	}

	ctx := context.Background()
	now := fixedNow // a.now() di dalam server memakai jam palsu yang sama

	// Belum jatuh tempo — backoff 1 menit belum lewat.
	if err := api.ProcessDueWebhooks(ctx, now.Add(30*time.Second)); err != nil {
		t.Fatalf("ProcessDueWebhooks sebelum backoff: %v", err)
	}
	if attempt != 1 {
		t.Fatalf("attempt sebelum backoff lewat = %d, mau tetap 1", attempt)
	}

	// Backoff 1 menit sudah lewat — retry kedua harus berhasil (server
	// palsu membalas 200 pada percobaan ke-2).
	if err := api.ProcessDueWebhooks(ctx, now.Add(90*time.Second)); err != nil {
		t.Fatalf("ProcessDueWebhooks setelah backoff: %v", err)
	}
	if attempt != 2 {
		t.Fatalf("attempt setelah backoff lewat = %d, mau 2", attempt)
	}

	// Tidak ada lagi yang due — sudah DELIVERED.
	if err := api.ProcessDueWebhooks(ctx, now.Add(1*time.Hour)); err != nil {
		t.Fatalf("ProcessDueWebhooks final: %v", err)
	}
	if attempt != 2 {
		t.Fatalf("attempt setelah DELIVERED = %d, mau tetap 2 (tidak dikirim lagi)", attempt)
	}
}
