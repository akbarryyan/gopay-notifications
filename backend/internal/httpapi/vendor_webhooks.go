package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

var validDeliveryStatuses = map[string]bool{
	store.WebhookDeliveryPending:   true,
	store.WebhookDeliveryRetrying:  true,
	store.WebhookDeliveryDelivered: true,
	store.WebhookDeliveryFailed:    true,
}

type vendorDeliveryJSON struct {
	ID            string  `json:"id"`
	AccountID     string  `json:"account_id"`
	BusinessName  string  `json:"business_name"`
	EndpointID    string  `json:"endpoint_id"`
	EndpointName  string  `json:"endpoint_name"`
	EndpointURL   string  `json:"endpoint_url"`
	Event         string  `json:"event"`
	InvoiceID     *string `json:"invoice_id"`
	Status        string  `json:"status"`
	Attempt       int     `json:"attempt"`
	NextAttemptAt *string `json:"next_attempt_at"`
	HTTPStatus    *int    `json:"http_status"`
	DurationMs    *int    `json:"duration_ms"`
	CreatedAt     string  `json:"created_at"`
	DeliveredAt   *string `json:"delivered_at"`
}

func toVendorDeliveryJSON(d store.VendorWebhookDelivery) vendorDeliveryJSON {
	out := vendorDeliveryJSON{
		ID: d.ID, AccountID: d.AccountID, BusinessName: d.BusinessName,
		EndpointID: d.EndpointID, EndpointName: d.EndpointName, EndpointURL: d.EndpointURL,
		Event: d.Event, InvoiceID: d.InvoiceID, Status: d.Status, Attempt: d.Attempt,
		HTTPStatus: d.HTTPStatus, DurationMs: d.DurationMs,
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
	}
	if d.NextAttemptAt != nil {
		s := d.NextAttemptAt.Format(time.RFC3339)
		out.NextAttemptAt = &s
	}
	if d.DeliveredAt != nil {
		s := d.DeliveredAt.Format(time.RFC3339)
		out.DeliveredAt = &s
	}
	return out
}

// handleVendorWebhookDeliveries melayani halaman Webhooks lintas-account
// di Vendor Dashboard -- read-only, pola parsing query param sama persis
// handleVendorTransactions, cuma sumber datanya ListAllWebhookDeliveries.
func (a *API) handleVendorWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
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

	filter := store.WebhookDeliveryFilter{
		Query: r.URL.Query().Get("q"),
	}
	if raw := r.URL.Query().Get("status"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			if !validDeliveryStatuses[s] {
				a.writeError(w, http.StatusBadRequest, "invalid_payload", "status tidak dikenal: "+s)
				return
			}
			filter.Statuses = append(filter.Statuses, s)
		}
	}
	if raw := r.URL.Query().Get("from"); raw != "" {
		from, err := time.Parse("2006-01-02", raw)
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "from harus YYYY-MM-DD")
			return
		}
		filter.From = &from
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		to, err := time.Parse("2006-01-02", raw)
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "to harus YYYY-MM-DD")
			return
		}
		to = to.Add(24*time.Hour - time.Nanosecond)
		filter.To = &to
	}

	deliveries, err := a.store.ListAllWebhookDeliveries(r.Context(), limit, offset, filter)
	if err != nil {
		slog.Error("vendor: ambil webhook deliveries gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]vendorDeliveryJSON, 0, len(deliveries))
	for _, d := range deliveries {
		out = append(out, toVendorDeliveryJSON(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "deliveries": out})
}
