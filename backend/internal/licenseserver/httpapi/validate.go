package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

func hashLicenseKey(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

type validateRequest struct {
	InstallationID string `json:"installation_id"`
	ProductVersion string `json:"product_version"`
}

// handleValidate (§11 license-spec.md) — dipanggil backend customer tiap
// 24 jam. License key SAJA tidak cukup (§6): installation_id juga harus
// cocok dan masih terikat (belum di-reset vendor).
func (a *API) handleValidate(w http.ResponseWriter, r *http.Request) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		a.writeError(w, http.StatusUnauthorized, "unauthenticated", "license key tidak ditemukan")
		return
	}
	rawKey := strings.TrimPrefix(header, prefix)

	var req validateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.InstallationID == "" || req.ProductVersion == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"installation_id dan product_version wajib diisi")
		return
	}

	l, err := a.store.GetLicenseByKeyHash(r.Context(), hashLicenseKey(rawKey))
	if errors.Is(err, store.ErrLicenseNotFound) {
		a.writeError(w, http.StatusUnauthorized, "invalid_license_key", "license key tidak valid")
		return
	}
	if err != nil {
		slog.Error("get license by key gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	inst, err := a.store.GetActiveInstallation(r.Context(), l.ID, req.InstallationID)
	if errors.Is(err, store.ErrInstallationNotFound) {
		// Installation sudah di-reset vendor (migrasi VPS, atau dicabut) —
		// backend customer (internal/licenseclient) akan menghapus local
		// state-nya begitu menerima 404 ini, bukan cuma membiarkan grace
		// period berjalan seolah ini sekadar gangguan jaringan.
		a.writeError(w, http.StatusNotFound, "installation_not_found",
			"installation ini sudah tidak terikat ke license — aktivasi ulang diperlukan")
		return
	}
	if err != nil {
		slog.Error("get active installation gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	state, err := a.buildAndSignState(r.Context(), l, inst)
	if err != nil {
		slog.Error("sign local license state gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), "installation:"+inst.ID, "LICENSE_VALIDATED", l.ID, nil); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "license_state": state})
}
