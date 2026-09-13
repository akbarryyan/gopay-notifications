package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

var validEnvironments = map[string]bool{"production": true, "uat": true}

// buildAndSignState membangun local license state dan menandatanganinya —
// dipakai baik oleh /activate maupun /validate, satu-satunya tempat yang
// tahu bentuk IssueInput. license.Status di sini SELALU AdminStatusActive
// (satu-satunya status yang lolos sampai titik ini di kedua handler),
// tapi field admin_status tetap dikirim apa adanya supaya
// internal/licensecheck.Load di sisi backend customer yang memutuskan
// status turunannya (active/expiring/expired), bukan diasumsikan di sini.
func (a *API) buildAndSignState(ctx context.Context, l store.License, inst store.Installation) (string, error) {
	customer, err := a.store.GetCustomer(ctx, l.CustomerID)
	if err != nil {
		return "", err
	}
	in := licensecheck.IssueInput{
		LicenseID:      l.ID,
		InstallationID: inst.ID,
		Customer:       customer.Name,
		Plan:           l.Plan,
		AdminStatus:    string(l.Status),
		MaxDevices:     l.MaxDevices,
		IssuedAt:       l.IssuedAt,
		ExpiresAt:      l.ExpiresAt,
		ValidatedAt:    a.now(),
	}
	return licensecheck.Issue(a.signingPrivateKey, in)
}

type activateRequest struct {
	LicenseKey     string `json:"license_key"`
	Environment    string `json:"environment"`
	ProductVersion string `json:"product_version"`
}

// handleActivate (§7, §10 license-spec.md) — dipanggil backend customer
// SEKALI saat instalasi pertama kali diaktivasi, dipicu dari
// POST /api/v1/admin/license/activate di backend customer.
func (a *API) handleActivate(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.activateThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts", "terlalu banyak percobaan, coba lagi nanti")
		return
	}

	var req activateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.LicenseKey == "" || !validEnvironments[req.Environment] || req.ProductVersion == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"license_key, environment (production/uat), dan product_version wajib diisi")
		return
	}

	keyHash := hashLicenseKey(req.LicenseKey)
	l, err := a.store.GetLicenseByKeyHash(r.Context(), keyHash)
	if errors.Is(err, store.ErrLicenseNotFound) {
		a.activateThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusUnauthorized, "invalid_license_key", "license key tidak valid")
		return
	}
	if err != nil {
		slog.Error("get license by key gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	a.activateThrottle.RecordSuccess(ip)

	inst, err := a.store.Activate(r.Context(), l, req.Environment, req.ProductVersion)
	if errors.Is(err, store.ErrLicenseNotActive) {
		a.writeError(w, http.StatusForbidden, "license_not_active", "license ini sedang suspended atau revoked")
		return
	}
	if errors.Is(err, store.ErrInstallationLimitReached) {
		a.writeError(w, http.StatusConflict, "installation_limit_reached",
			"kuota installation "+req.Environment+" untuk license ini sudah penuh — hubungi vendor")
		return
	}
	if err != nil {
		slog.Error("activate gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	state, err := a.buildAndSignState(r.Context(), l, inst)
	if err != nil {
		slog.Error("sign local license state gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), "installation:"+inst.ID, "INSTALLATION_CREATED", l.ID,
		map[string]string{"environment": req.Environment, "product_version": req.ProductVersion}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":         true,
		"installation_id": inst.ID,
		"license_state":   state,
	})
}
