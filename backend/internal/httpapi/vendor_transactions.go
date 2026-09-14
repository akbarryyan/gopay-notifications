package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type vendorInvoiceJSON struct {
	invoiceJSON
	AccountID    string `json:"account_id"`
	BusinessName string `json:"business_name"`
}

func toVendorInvoiceJSON(v store.VendorInvoice) vendorInvoiceJSON {
	return vendorInvoiceJSON{
		invoiceJSON:  toInvoiceJSON(v.Invoice),
		AccountID:    v.AccountID,
		BusinessName: v.BusinessName,
	}
}

// handleVendorTransactions melayani halaman Transactions lintas-account di
// Vendor Dashboard -- parsing query param (limit/offset/status/q/from/to)
// sengaja sama persis handleAdminInvoices, cuma sumber datanya
// ListAllInvoices (tanpa scope satu account) dan setiap baris ikut
// menyertakan business_name pemiliknya.
func (a *API) handleVendorTransactions(w http.ResponseWriter, r *http.Request) {
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
	if raw := r.URL.Query().Get("status"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			if !validInvoiceStatuses[s] {
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

	invoices, err := a.store.ListAllInvoices(r.Context(), limit, offset, filter)
	if err != nil {
		slog.Error("vendor: ambil transactions gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]vendorInvoiceJSON, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, toVendorInvoiceJSON(inv))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "invoices": out})
}
