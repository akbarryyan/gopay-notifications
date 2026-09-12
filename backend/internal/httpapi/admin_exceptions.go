package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type exceptionsListResponse struct {
	Exceptions []eventJSON `json:"exceptions"`
}

// handleAdminExceptions melayani halaman Exceptions — event yang
// amount_hint-nya terisi tapi tidak cocok invoice manapun dan belum
// di-dismiss. Pola filter (limit/offset/q/from/to) sama dengan handleEvents,
// minus source (exception tidak difilter per sumber di MVP ini).
func (a *API) handleAdminExceptions(w http.ResponseWriter, r *http.Request) {
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

	filter := store.ExceptionFilter{Query: r.URL.Query().Get("q")}
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

	exceptions, err := a.store.ListExceptions(r.Context(), limit, offset, filter)
	if err != nil {
		slog.Error("ambil exceptions gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]eventJSON, 0, len(exceptions))
	for _, e := range exceptions {
		out = append(out, toEventJSON(e))
	}
	writeJSON(w, http.StatusOK, exceptionsListResponse{Exceptions: out})
}

type matchExceptionRequest struct {
	InvoiceID string `json:"invoice_id"`
}

// handleAdminMatchException mencocokkan event ke invoice pilihan admin.
// Berhasil → webhook invoice.paid ikut terpicu, persis seperti matching
// otomatis (helper triggerInvoiceWebhook yang sama, dua pemicu).
func (a *API) handleAdminMatchException(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")

	var req matchExceptionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.InvoiceID == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "invoice_id wajib diisi")
		return
	}

	err := a.store.ManualMatchEvent(r.Context(), a.now(), req.InvoiceID, eventID)
	if errors.Is(err, store.ErrInvoiceNotEligibleForMatch) {
		a.writeError(w, http.StatusConflict, "invoice_not_eligible",
			"invoice tidak ditemukan atau sudah PAID")
		return
	}
	if errors.Is(err, store.ErrEventAlreadyMatched) {
		a.writeError(w, http.StatusConflict, "event_already_matched",
			"event ini sudah dipakai untuk invoice lain")
		return
	}
	if err != nil {
		slog.Error("manual match event gagal", "event_id", eventID, "invoice_id", req.InvoiceID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	go a.triggerInvoiceWebhook(store.WebhookEventInvoicePaid, req.InvoiceID)

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

type dismissExceptionRequest struct {
	Note string `json:"note"`
}

// handleAdminDismissException menandai event sebagai sengaja diabaikan.
// Tidak ada "undo" — konsisten dengan pola tidak-ada-un-revoke di seluruh
// sistem ini (API key, webhook).
func (a *API) handleAdminDismissException(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")

	var req dismissExceptionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}

	var note *string
	if req.Note != "" {
		note = &req.Note
	}

	err := a.store.DismissEvent(r.Context(), eventID, note)
	if errors.Is(err, store.ErrEventNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "event tidak ditemukan")
		return
	}
	if errors.Is(err, store.ErrEventAlreadyDismissed) {
		a.writeError(w, http.StatusConflict, "already_dismissed", "event ini sudah pernah diabaikan")
		return
	}
	if err != nil {
		slog.Error("dismiss event gagal", "event_id", eventID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
