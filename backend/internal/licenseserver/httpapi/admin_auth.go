package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

// vendor_session, bukan admin_session — cookie ini beda domain (
// license.whuzpay.com) dari cookie admin_session milik dashboard customer
// manapun, tapi nama berbeda menghindari kebingungan kalau suatu saat
// keduanya sempat dibuka di tab browser yang sama.
const vendorSessionCookie = "vendor_session"

type adminCtxKey int

const ctxKeyAdmin adminCtxKey = iota

func AdminFromContext(ctx context.Context) bool {
	ok, _ := ctx.Value(ctxKeyAdmin).(bool)
	return ok
}

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *API) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.loginThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts",
			"terlalu banyak percobaan login, coba lagi nanti")
		return
	}

	var req adminLoginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Username == "" || req.Password == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "username dan password wajib diisi")
		return
	}

	admin, err := a.store.GetAdminByUsername(r.Context(), req.Username)
	if errors.Is(err, store.ErrAdminNotFound) {
		a.loginThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "username atau password salah")
		return
	}
	if err != nil {
		slog.Error("ambil admin gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if !admin.VerifyPassword(req.Password) {
		a.loginThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "username atau password salah")
		return
	}
	a.loginThrottle.RecordSuccess(ip)

	token := auth.NewSessionToken(a.adminSessionKey, a.now())
	http.SetCookie(w, &http.Cookie{
		Name:     vendorSessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.SessionDuration.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     vendorSessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(vendorSessionCookie)
		if err != nil || cookie.Value == "" {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak ditemukan")
			return
		}
		if !auth.VerifySessionToken(a.adminSessionKey, cookie.Value, a.now()) {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak valid atau kedaluwarsa")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyAdmin, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
