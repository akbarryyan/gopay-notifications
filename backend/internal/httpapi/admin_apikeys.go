package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type apiKeyJSON struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	CreatedAt string  `json:"created_at"`
	RevokedAt *string `json:"revoked_at"`
}

func toAPIKeyJSON(k store.APIKey) apiKeyJSON {
	out := apiKeyJSON{ID: k.ID, Name: k.Name, CreatedAt: k.CreatedAt.Format(time.RFC3339)}
	if k.RevokedAt != nil {
		s := k.RevokedAt.Format(time.RFC3339)
		out.RevokedAt = &s
	}
	return out
}

type createAPIKeyRequest struct {
	Name string `json:"name"`
}

type createAPIKeyResponse struct {
	apiKeyJSON
	// Key mentah, hanya ada di response ini — tidak pernah bisa diambil lagi
	// setelahnya, persis pola token GitHub/Stripe.
	Key string `json:"key"`
}

// handleAdminCreateAPIKey membuat API key baru untuk dipakai website
// merchant memanggil POST/GET /invoices.
func (a *API) handleAdminCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	var req createAPIKeyRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Name == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "name wajib diisi")
		return
	}

	id, err := store.NewAPIKeyID()
	if err != nil {
		slog.Error("generate api key id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		slog.Error("generate api key secret gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.CreateAPIKey(r.Context(), accountID, id, req.Name, hash); err != nil {
		slog.Error("simpan api key gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{
		apiKeyJSON: apiKeyJSON{ID: id, Name: req.Name, CreatedAt: a.now().Format(time.RFC3339)},
		Key:        rawKey,
	})
}

type apiKeysListResponse struct {
	APIKeys []apiKeyJSON `json:"api_keys"`
}

// handleAdminListAPIKeys tidak pernah menyertakan key mentah atau hash-nya —
// hanya metadata yang aman ditampilkan berulang kali.
func (a *API) handleAdminListAPIKeys(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	keys, err := a.store.ListAPIKeys(r.Context(), accountID)
	if err != nil {
		slog.Error("ambil api keys gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]apiKeyJSON, 0, len(keys))
	for _, k := range keys {
		out = append(out, toAPIKeyJSON(k))
	}
	writeJSON(w, http.StatusOK, apiKeysListResponse{APIKeys: out})
}

// handleAdminRevokeAPIKey selalu mencabut — tidak ada jalan mengaktifkan
// kembali key yang sudah dicabut lewat endpoint ini (bikin key baru kalau
// perlu). Idempotent: mencabut yang sudah dicabut tetap 200.
func (a *API) handleAdminRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	id := r.PathValue("keyID")
	err := a.store.RevokeAPIKey(r.Context(), accountID, id)
	if errors.Is(err, store.ErrAPIKeyNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "api key tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("cabut api key gagal", "id", id, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
