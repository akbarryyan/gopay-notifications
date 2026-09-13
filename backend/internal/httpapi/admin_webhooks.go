package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

var validWebhookEvents = map[string]bool{
	store.WebhookEventInvoicePaid:    true,
	store.WebhookEventInvoiceExpired: true,
}

type webhookEndpointJSON struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	URL                string   `json:"url"`
	Events             []string `json:"events"`
	Enabled            bool     `json:"enabled"`
	CreatedAt          string   `json:"created_at"`
	LastDeliveryAt     *string  `json:"last_delivery_at"`
	LastDeliveryStatus *string  `json:"last_delivery_status"`
}

func toWebhookEndpointJSON(w store.WebhookEndpointSummary) webhookEndpointJSON {
	out := webhookEndpointJSON{
		ID:                 w.ID,
		Name:               w.Name,
		URL:                w.URL,
		Events:             w.Events,
		Enabled:            w.Enabled,
		CreatedAt:          w.CreatedAt.Format(time.RFC3339),
		LastDeliveryStatus: w.LastDeliveryStatus,
	}
	if w.LastDeliveryAt != nil {
		s := w.LastDeliveryAt.Format(time.RFC3339)
		out.LastDeliveryAt = &s
	}
	return out
}

type createWebhookRequest struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type createWebhookResponse struct {
	webhookEndpointJSON
	// Secret mentah, hanya ada di response ini — tidak pernah bisa diambil
	// lagi setelahnya, persis pola API key.
	Secret string `json:"secret"`
}

// handleAdminCreateWebhook membuat endpoint webhook baru.
func (a *API) handleAdminCreateWebhook(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	var req createWebhookRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Name == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "name wajib diisi")
		return
	}
	if req.URL == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "url wajib diisi")
		return
	}
	if len(req.Events) == 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "events wajib berisi minimal satu event")
		return
	}
	for _, e := range req.Events {
		if !validWebhookEvents[e] {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "event tidak dikenal: "+e)
			return
		}
	}

	id, err := store.NewWebhookID()
	if err != nil {
		slog.Error("generate webhook id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	rawSecret, secretBytes, err := store.GenerateWebhookSecret()
	if err != nil {
		slog.Error("generate webhook secret gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	if err := a.store.CreateWebhookEndpoint(r.Context(), a.webhookSecretKey, accountID, id, req.Name, req.URL, req.Events, secretBytes); err != nil {
		slog.Error("simpan webhook endpoint gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	writeJSON(w, http.StatusCreated, createWebhookResponse{
		webhookEndpointJSON: webhookEndpointJSON{
			ID:        id,
			Name:      req.Name,
			URL:       req.URL,
			Events:    req.Events,
			Enabled:   true,
			CreatedAt: a.now().Format(time.RFC3339),
		},
		Secret: rawSecret,
	})
}

type webhooksListResponse struct {
	Webhooks []webhookEndpointJSON `json:"webhooks"`
}

// handleAdminListWebhooks tidak pernah menyertakan secret — hanya metadata
// yang aman ditampilkan berulang kali.
func (a *API) handleAdminListWebhooks(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	endpoints, err := a.store.ListWebhookEndpoints(r.Context(), accountID)
	if err != nil {
		slog.Error("ambil webhook endpoints gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]webhookEndpointJSON, 0, len(endpoints))
	for _, e := range endpoints {
		out = append(out, toWebhookEndpointJSON(e))
	}
	writeJSON(w, http.StatusOK, webhooksListResponse{Webhooks: out})
}

type setWebhookEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// handleAdminSetWebhookEnabled mengaktifkan/menonaktifkan endpoint. Tidak
// ada endpoint "edit" URL/events/nama — hapus lalu buat baru, konsisten
// dengan API key (tidak ada un-revoke).
func (a *API) handleAdminSetWebhookEnabled(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	id := r.PathValue("webhookID")
	var req setWebhookEnabledRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}

	err := a.store.SetWebhookEndpointEnabled(r.Context(), accountID, id, req.Enabled)
	if errors.Is(err, store.ErrWebhookNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "webhook tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("set webhook enabled gagal", "id", id, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleAdminDeleteWebhook menghapus endpoint. ON DELETE CASCADE di migrasi
// ikut menghapus riwayat deliveries-nya.
func (a *API) handleAdminDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	id := r.PathValue("webhookID")
	err := a.store.DeleteWebhookEndpoint(r.Context(), accountID, id)
	if errors.Is(err, store.ErrWebhookNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "webhook tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("hapus webhook gagal", "id", id, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

type testWebhookResponse struct {
	Delivered  bool `json:"delivered"`
	HTTPStatus int  `json:"http_status"`
	DurationMs int  `json:"duration_ms"`
}

// handleAdminTestWebhook mengirim payload event "test" seketika (sinkron,
// bukan enqueue-lalu-lapor-nanti) — pengguna sedang menonton dashboard
// menunggu jawaban endpoint-nya benar atau tidak, beda dari invoice.paid
// yang sengaja async supaya tidak menahan respons ke device.
//
// Payload test tidak pernah menyentuh tabel invoices (invoice_id NULL).
func (a *API) handleAdminTestWebhook(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	id := r.PathValue("webhookID")
	endpoint, secret, err := a.store.GetWebhookEndpoint(r.Context(), a.webhookSecretKey, accountID, id)
	if errors.Is(err, store.ErrWebhookNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "webhook tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("ambil webhook endpoint gagal", "id", id, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	now := a.now()
	payload, err := buildTestPayload(now)
	if err != nil {
		slog.Error("bangun payload test gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	outcome := a.sendWebhook(r.Context(), endpoint.URL, secret, store.WebhookEventTest, payload)

	deliveryID, derr := a.store.EnqueueTestDelivery(r.Context(), now, accountID, id, payload)
	if derr != nil {
		slog.Error("enqueue test delivery gagal", "err", derr)
	} else if rerr := a.store.RecordTestDeliveryResult(r.Context(), deliveryID, now, outcome.Success, outcome.HTTPStatus, outcome.DurationMs); rerr != nil {
		slog.Error("catat hasil test delivery gagal", "err", rerr)
	}

	writeJSON(w, http.StatusOK, testWebhookResponse{
		Delivered:  outcome.Success,
		HTTPStatus: outcome.HTTPStatus,
		DurationMs: outcome.DurationMs,
	})
}

type webhookDeliveryJSON struct {
	ID          string  `json:"id"`
	Event       string  `json:"event"`
	InvoiceID   *string `json:"invoice_id"`
	Status      string  `json:"status"`
	Attempt     int     `json:"attempt"`
	HTTPStatus  *int    `json:"http_status"`
	DurationMs  *int    `json:"duration_ms"`
	CreatedAt   string  `json:"created_at"`
	DeliveredAt *string `json:"delivered_at"`
}

func toWebhookDeliveryJSON(d store.WebhookDelivery) webhookDeliveryJSON {
	out := webhookDeliveryJSON{
		ID:         d.ID,
		Event:      d.Event,
		InvoiceID:  d.InvoiceID,
		Status:     d.Status,
		Attempt:    d.Attempt,
		HTTPStatus: d.HTTPStatus,
		DurationMs: d.DurationMs,
		CreatedAt:  d.CreatedAt.Format(time.RFC3339),
	}
	if d.DeliveredAt != nil {
		s := d.DeliveredAt.Format(time.RFC3339)
		out.DeliveredAt = &s
	}
	return out
}

type webhookDeliveriesResponse struct {
	Deliveries []webhookDeliveryJSON `json:"deliveries"`
}

// handleAdminWebhookDeliveries mengembalikan riwayat pengiriman satu
// endpoint untuk baris yang diperluas di dashboard.
func (a *API) handleAdminWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	id := r.PathValue("webhookID")
	limit, err := intParam(r, "limit", 50)
	if err != nil || limit < 1 || limit > 1000 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "limit harus bilangan bulat 1..1000")
		return
	}
	offset, err := intParam(r, "offset", 0)
	if err != nil || offset < 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "offset harus bilangan bulat >= 0")
		return
	}

	deliveries, err := a.store.ListWebhookDeliveries(r.Context(), accountID, id, limit, offset)
	if err != nil {
		slog.Error("ambil webhook deliveries gagal", "id", id, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]webhookDeliveryJSON, 0, len(deliveries))
	for _, d := range deliveries {
		out = append(out, toWebhookDeliveryJSON(d))
	}
	writeJSON(w, http.StatusOK, webhookDeliveriesResponse{Deliveries: out})
}
