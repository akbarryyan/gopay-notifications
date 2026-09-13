package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
)

const vendorSessionCookie = "vendor_session"

type vendorCtxKey int

const ctxKeyVendorUsername vendorCtxKey = iota

// VendorFromContext melaporkan username vendor admin pemilik sesi ini.
func VendorFromContext(ctx context.Context) (username string, ok bool) {
	v, ok := ctx.Value(ctxKeyVendorUsername).(string)
	return v, ok
}

type vendorLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleVendorLogin -- pola identik handleAdminLogin, tabel dan cookie beda
// (vendor_admins, vendor_session) supaya sesi vendor dan sesi customer
// TIDAK PERNAH bisa tertukar walau di browser yang sama.
func (a *API) handleVendorLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.vendorLoginThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts", "terlalu banyak percobaan login, coba lagi nanti")
		return
	}
	var req vendorLoginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Username == "" || req.Password == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "username dan password wajib diisi")
		return
	}
	vendor, err := a.store.GetVendorAdminByUsername(r.Context(), req.Username)
	if err != nil || !vendor.VerifyPassword(req.Password) {
		a.vendorLoginThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "username atau password salah")
		return
	}
	a.vendorLoginThrottle.RecordSuccess(ip)

	token := auth.NewSessionToken(a.vendorSessionKey, a.now(), vendor.Username)
	http.SetCookie(w, &http.Cookie{
		Name: vendorSessionCookie, Value: token, Path: "/", HttpOnly: true,
		Secure: isSecureRequest(r), SameSite: http.SameSiteLaxMode,
		MaxAge: int(auth.SessionDuration.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) handleVendorLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: vendorSessionCookie, Value: "", Path: "/", HttpOnly: true,
		Secure: isSecureRequest(r), SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) requireVendor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(vendorSessionCookie)
		if err != nil || cookie.Value == "" {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak ditemukan")
			return
		}
		username, ok := auth.VerifySessionToken(a.vendorSessionKey, cookie.Value, a.now())
		if !ok {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak valid atau kedaluwarsa")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyVendorUsername, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
