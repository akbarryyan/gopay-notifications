package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

// planPresets memetakan nama plan ke max_devices default (§17
// license-spec.md, dipangkas ke satu entitlement yang ditegakkan — lihat
// spec §9 untuk daftar yang ditunda). -1 berarti unlimited.
var planPresets = map[string]int{
	"Starter":    3,
	"Business":   10,
	"Enterprise": -1,
}

const dateOnlyLayout = "2006-01-02"

type licenseJSON struct {
	ID                      string `json:"id"`
	CustomerID              string `json:"customer_id"`
	Plan                    string `json:"plan"`
	Status                  string `json:"status"`
	MaxDevices              int    `json:"max_devices"`
	ProductionInstallations int    `json:"production_installations"`
	UATInstallations        int    `json:"uat_installations"`
	IssuedAt                string `json:"issued_at"`
	ExpiresAt               string `json:"expires_at"`
	DaysRemaining           int    `json:"days_remaining"`
}

func toLicenseJSON(l store.License, now time.Time) licenseJSON {
	expiresEndOfDay := l.ExpiresAt.Add(24*time.Hour - time.Second)
	return licenseJSON{
		ID:                      l.ID,
		CustomerID:              l.CustomerID,
		Plan:                    l.Plan,
		Status:                  l.DerivedStatus(now, licensecheck.WarningThresholdDays),
		MaxDevices:              l.MaxDevices,
		ProductionInstallations: l.ProductionInstallations,
		UATInstallations:        l.UATInstallations,
		IssuedAt:                l.IssuedAt.Format(dateOnlyLayout),
		ExpiresAt:               l.ExpiresAt.Format(dateOnlyLayout),
		DaysRemaining:           int(expiresEndOfDay.Sub(now).Hours() / 24),
	}
}

type createLicenseRequest struct {
	Plan      string `json:"plan"`
	ExpiresAt string `json:"expires_at"`
}

type createLicenseResponse struct {
	licenseJSON
	// Key mentah — hanya ada di response ini, tidak pernah bisa diambil
	// lagi setelahnya. Pola sama persis API key/webhook secret di dashboard
	// customer.
	Key string `json:"key"`
}

func (a *API) handleCreateLicense(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customerID")
	if _, err := a.store.GetCustomer(r.Context(), customerID); errors.Is(err, store.ErrCustomerNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "customer tidak ditemukan")
		return
	} else if err != nil {
		slog.Error("get customer gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	var req createLicenseRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	maxDevices, ok := planPresets[req.Plan]
	if !ok {
		a.writeError(w, http.StatusBadRequest, "invalid_plan", "plan harus salah satu: Starter, Business, Enterprise")
		return
	}
	expiresAt, err := time.Parse(dateOnlyLayout, req.ExpiresAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "expires_at harus format YYYY-MM-DD")
		return
	}

	id, err := store.NewLicenseID()
	if err != nil {
		slog.Error("generate license id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	rawKey, keyHash, err := store.GenerateLicenseKey(req.Plan)
	if err != nil {
		slog.Error("generate license key gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	issuedAt := a.now()
	in := store.CreateLicenseInput{
		ID: id, CustomerID: customerID, KeyHash: keyHash, Plan: req.Plan, MaxDevices: maxDevices,
		ProductionInstallations: 1, UATInstallations: 1,
		IssuedAt: issuedAt, ExpiresAt: expiresAt,
	}
	if err := a.store.CreateLicense(r.Context(), in); err != nil {
		slog.Error("create license gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), "vendor", "LICENSE_CREATED", id,
		map[string]any{"customer_id": customerID, "plan": req.Plan, "expires_at": req.ExpiresAt}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	l := store.License{
		ID: id, CustomerID: customerID, Plan: req.Plan, Status: store.AdminStatusActive,
		MaxDevices: maxDevices, ProductionInstallations: 1, UATInstallations: 1,
		IssuedAt: issuedAt, ExpiresAt: expiresAt,
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"license": createLicenseResponse{licenseJSON: toLicenseJSON(l, a.now()), Key: rawKey},
	})
}

func (a *API) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("licenseID")
	l, err := a.store.GetLicenseByID(r.Context(), id)
	if errors.Is(err, store.ErrLicenseNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "license tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("get license gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	installations, err := a.store.ListInstallationsByLicense(r.Context(), id)
	if err != nil {
		slog.Error("list installations gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]installationJSON, 0, len(installations))
	for _, inst := range installations {
		out = append(out, toInstallationJSON(inst))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true, "license": toLicenseJSON(l, a.now()), "installations": out,
	})
}

type renewLicenseRequest struct {
	ExpiresAt string `json:"expires_at"`
}

func (a *API) handleRenewLicense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("licenseID")
	var req renewLicenseRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	newExpiresAt, err := time.Parse(dateOnlyLayout, req.ExpiresAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "expires_at harus format YYYY-MM-DD")
		return
	}

	if err := a.store.RenewLicense(r.Context(), id, newExpiresAt); errors.Is(err, store.ErrLicenseNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "license tidak ditemukan")
		return
	} else if err != nil {
		slog.Error("renew license gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), "vendor", "LICENSE_RENEWED", id,
		map[string]string{"new_expires_at": req.ExpiresAt}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) setLicenseStatus(w http.ResponseWriter, r *http.Request, status store.AdminStatus, action string) {
	id := r.PathValue("licenseID")
	if err := a.store.SetLicenseStatus(r.Context(), id, status); errors.Is(err, store.ErrLicenseNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "license tidak ditemukan")
		return
	} else if err != nil {
		slog.Error(action+" gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), "vendor", action, id, nil); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) handleSuspendLicense(w http.ResponseWriter, r *http.Request) {
	a.setLicenseStatus(w, r, store.AdminStatusSuspended, "LICENSE_SUSPENDED")
}

func (a *API) handleRevokeLicense(w http.ResponseWriter, r *http.Request) {
	a.setLicenseStatus(w, r, store.AdminStatusRevoked, "LICENSE_REVOKED")
}
