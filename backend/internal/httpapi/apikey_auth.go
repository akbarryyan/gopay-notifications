package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type apiKeyCtxKey int

const ctxKeyAPIKey apiKeyCtxKey = iota

// APIKeyFromContext mengembalikan API key yang lolos requireAPIKey.
func APIKeyFromContext(ctx context.Context) (store.APIKey, bool) {
	k, ok := ctx.Value(ctxKeyAPIKey).(store.APIKey)
	return k, ok
}

// requireAPIKey melindungi endpoint invoice yang dipanggil server website
// merchant — terpisah dari HMAC device dan cookie sesi admin.
func (a *API) requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "API key tidak ditemukan")
			return
		}
		rawKey := strings.TrimPrefix(header, prefix)

		key, err := a.store.VerifyAPIKey(r.Context(), rawKey)
		if errors.Is(err, store.ErrAPIKeyNotFound) {
			// Key salah dan key benar-tapi-dicabut sengaja disamakan —
			// pola yang sama dengan login admin, supaya status "dicabut"
			// tidak bocor lewat perbedaan respons.
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "API key tidak valid")
			return
		}
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyAPIKey, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
