package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type invoicesListResponse struct {
	Invoices []invoiceJSON `json:"invoices"`
}

var validInvoiceStatuses = map[string]bool{
	store.InvoiceStatusPending: true,
	store.InvoiceStatusPaid:    true,
	store.InvoiceStatusExpired: true,
}

// handleAdminInvoices melayani halaman Transactions di dashboard — pola
// filter (limit/offset/status/q/from/to) sama persis dengan handleEvents.
func (a *API) handleAdminInvoices(w http.ResponseWriter, r *http.Request) {
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

	filter := store.InvoiceFilter{
		Query: r.URL.Query().Get("q"),
	}
	// Boleh lebih dari satu, dipisah koma (mis. status=PENDING,EXPIRED) —
	// dipakai dialog pencocokan manual di konsol pengecualian supaya tidak
	// perlu dua kali panggilan untuk dua status sekaligus.
	if raw := r.URL.Query().Get("status"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			if !validInvoiceStatuses[s] {
				a.writeError(w, http.StatusBadRequest, "invalid_payload", "status tidak dikenal: "+s)
				return
			}
			filter.Statuses = append(filter.Statuses, s)
		}
	}
	// from/to bertanggal saja (YYYY-MM-DD, UTC), sama persis dengan
	// handleEvents — selaras dengan <input type="date"> di dashboard.
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

	invoices, err := a.store.ListInvoices(r.Context(), limit, offset, filter)
	if err != nil {
		slog.Error("ambil invoices gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]invoiceJSON, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, toInvoiceJSON(inv))
	}
	writeJSON(w, http.StatusOK, invoicesListResponse{Invoices: out})
}
