package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type createInvoiceRequest struct {
	ExternalRef string `json:"external_ref"`
	Amount      int64  `json:"amount"`
}

type invoiceJSON struct {
	ID              string  `json:"id"`
	ExternalRef     string  `json:"external_ref"`
	RequestedAmount int64   `json:"requested_amount"`
	UniqueAmount    int64   `json:"unique_amount"`
	Status          string  `json:"status"`
	MatchedEventID  *string `json:"matched_event_id"`
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       string  `json:"expires_at"`
	PaidAt          *string `json:"paid_at"`
}

func toInvoiceJSON(inv store.Invoice) invoiceJSON {
	out := invoiceJSON{
		ID:              inv.ID,
		ExternalRef:     inv.ExternalRef,
		RequestedAmount: inv.RequestedAmount,
		UniqueAmount:    inv.UniqueAmount,
		Status:          inv.Status,
		MatchedEventID:  inv.MatchedEventID,
		CreatedAt:       inv.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       inv.ExpiresAt.Format(time.RFC3339),
	}
	if inv.PaidAt != nil {
		s := inv.PaidAt.Format(time.RFC3339)
		out.PaidAt = &s
	}
	return out
}

// handleCreateInvoice dipanggil dari server website merchant (bukan
// browser), diautentikasi lewat requireAPIKey.
func (a *API) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req createInvoiceRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.ExternalRef == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "external_ref wajib diisi")
		return
	}
	if req.Amount <= 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "amount harus lebih besar dari 0")
		return
	}

	inv, created, err := a.store.CreateInvoice(r.Context(), a.now(), req.ExternalRef, req.Amount)
	switch {
	case errors.Is(err, store.ErrInvoiceRefConflict):
		a.writeError(w, http.StatusConflict, "external_ref_conflict",
			"external_ref sudah dipakai invoice lain dengan amount berbeda")
		return
	case errors.Is(err, store.ErrInvoiceAllocationFull):
		a.writeError(w, http.StatusServiceUnavailable, "allocation_full",
			"tidak ada nominal unik yang tersedia untuk amount ini, coba lagi sebentar lagi")
		return
	case err != nil:
		slog.Error("create invoice gagal", "external_ref", req.ExternalRef, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, toInvoiceJSON(inv))
}

// handleGetInvoice dipakai merchant untuk polling status sebelum webhook
// (fase 2) ada.
func (a *API) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("invoiceID")
	inv, err := a.store.GetInvoiceByID(r.Context(), id)
	if errors.Is(err, store.ErrInvoiceNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "invoice tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("ambil invoice gagal", "id", id, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, toInvoiceJSON(inv))
}
