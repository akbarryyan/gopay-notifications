package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

type customerJSON struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func toCustomerJSON(c store.Customer) customerJSON {
	return customerJSON{ID: c.ID, Name: c.Name, CreatedAt: c.CreatedAt.Format(time.RFC3339)}
}

type createCustomerRequest struct {
	Name string `json:"name"`
}

func (a *API) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req createCustomerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Name == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "name wajib diisi")
		return
	}

	id, err := store.NewCustomerID()
	if err != nil {
		slog.Error("generate customer id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.CreateCustomer(r.Context(), id, req.Name); err != nil {
		slog.Error("create customer gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), "vendor", "CUSTOMER_CREATED", id,
		map[string]string{"name": req.Name}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true, "customer": toCustomerJSON(store.Customer{ID: id, Name: req.Name, CreatedAt: a.now()}),
	})
}

func (a *API) handleListCustomers(w http.ResponseWriter, r *http.Request) {
	customers, err := a.store.ListCustomers(r.Context())
	if err != nil {
		slog.Error("list customers gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]customerJSON, 0, len(customers))
	for _, c := range customers {
		out = append(out, toCustomerJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "customers": out})
}

func (a *API) handleGetCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("customerID")
	customer, err := a.store.GetCustomer(r.Context(), id)
	if errors.Is(err, store.ErrCustomerNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "customer tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("get customer gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	licenses, err := a.store.ListLicensesByCustomer(r.Context(), id)
	if err != nil {
		slog.Error("list licenses gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]licenseJSON, 0, len(licenses))
	for _, l := range licenses {
		out = append(out, toLicenseJSON(l, a.now()))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true, "customer": toCustomerJSON(customer), "licenses": out,
	})
}
