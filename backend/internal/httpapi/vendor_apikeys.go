package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// handleVendorListAPIKeys mengembalikan seluruh API key milik SATU account
// customer, untuk halaman detail account di Vendor Dashboard -- reuse
// apiKeyJSON/toAPIKeyJSON dari admin_apikeys.go (bentuk data sama persis
// yang dilihat customer sendiri). Key mentah/hash tidak pernah disimpan
// sejak dibuat (lihat store.CreateAPIKey), jadi tidak ada jalan bagi
// endpoint ini membocorkannya -- vendor cuma bisa lihat metadata dan
// mencabut, tidak pernah melihat/memulihkan key.
func (a *API) handleVendorListAPIKeys(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountID")
	keys, err := a.store.ListAPIKeys(r.Context(), accountID)
	if err != nil {
		slog.Error("vendor: ambil api keys gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]apiKeyJSON, 0, len(keys))
	for _, k := range keys {
		out = append(out, toAPIKeyJSON(k))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "api_keys": out})
}

// handleVendorRevokeAPIKey mencabut satu API key milik account tertentu --
// dipakai saat customer lapor key-nya bocor/kepakai website lama dan tidak
// sempat/tidak bisa cabut sendiri lewat Customer Dashboard. Idempotent
// sama seperti versi customer (store.RevokeAPIKey): mencabut yang sudah
// dicabut tetap sukses.
func (a *API) handleVendorRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountID")
	keyID := r.PathValue("keyID")

	err := a.store.RevokeAPIKey(r.Context(), accountID, keyID)
	if errors.Is(err, store.ErrAPIKeyNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "api key tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("vendor: cabut api key gagal", "id", keyID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "API_KEY_REVOKED", accountID,
		map[string]string{"key_id": keyID}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
