package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// webhookPayload adalah bentuk JSON yang dikirim ke merchant. Sama untuk
// event invoice.paid/invoice.expired/test — invoiceJSON sengaja identik
// dengan bentuk GET /invoices/{id}, supaya merchant yang sudah polling
// tidak perlu belajar bentuk data baru begitu pindah ke webhook.
//
// Dibangun SEKALI saat delivery di-enqueue (buildInvoicePayload/
// buildTestPayload) dan disimpan apa adanya di kolom payload — percobaan
// ulang mengirim ulang byte yang persis sama, bukan membangun ulang dari
// state invoice yang mungkin sudah berubah.
type webhookPayload struct {
	Event   string       `json:"event"`
	Invoice *invoiceJSON `json:"invoice"`
	SentAt  string       `json:"sent_at"`
}

func buildInvoicePayload(event string, inv store.Invoice, sentAt time.Time) ([]byte, error) {
	j := toInvoiceJSON(inv)
	return json.Marshal(webhookPayload{Event: event, Invoice: &j, SentAt: sentAt.Format(time.RFC3339)})
}

func buildTestPayload(sentAt time.Time) ([]byte, error) {
	return json.Marshal(webhookPayload{Event: store.WebhookEventTest, Invoice: nil, SentAt: sentAt.Format(time.RFC3339)})
}

func signWebhookBody(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// deliveryOutcome adalah hasil satu percobaan pengiriman — tidak menyentuh
// database sama sekali, supaya bisa diuji terpisah dari logic status/retry.
type deliveryOutcome struct {
	Success    bool
	HTTPStatus int // 0 bila gagal terhubung sama sekali (timeout, DNS, dst)
	DurationMs int
}

// sendWebhook mengirim payload (byte yang sama persis tiap percobaan) ke
// satu endpoint. Status 2xx dianggap sukses; selain itu (termasuk gagal
// terhubung) dianggap gagal.
func (a *API) sendWebhook(ctx context.Context, url string, secret []byte, event string, payload []byte) deliveryOutcome {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		// URL tidak sah — bukan sesuatu yang bisa "coba lagi nanti" dan
		// membaik sendiri, tapi tetap diperlakukan sebagai kegagalan biasa
		// (retry akan gagal dengan cara yang sama, lalu menyerah wajar).
		return deliveryOutcome{Success: false}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("X-Webhook-Signature", signWebhookBody(secret, payload))

	start := time.Now()
	resp, err := a.webhookHTTPClient.Do(req)
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		// Timeout, DNS gagal, koneksi ditolak, dst — bentuk "gagal" yang
		// normal dari sisi delivery, harus dicatat sebagai gagal lalu
		// di-retry, bukan error yang menghentikan proses.
		return deliveryOutcome{Success: false, DurationMs: duration}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) // pastikan koneksi bisa dipakai ulang (keep-alive)

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	return deliveryOutcome{Success: success, HTTPStatus: resp.StatusCode, DurationMs: duration}
}
