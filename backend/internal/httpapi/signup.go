package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type signupRequest struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

const (
	signupTrialDays  = 3
	signupPlan       = "Starter"
	signupMaxDevices = 3
	minPasswordLen   = 8
)

// handleSignup adalah SATU-SATUNYA endpoint di seluruh backend yang tidak
// dibungkus requireAdmin/requireAPIKey/requireDevice/requireVendor apa pun
// -- sengaja publik, calon customer belum punya kredensial apa pun sampai
// titik ini. Cuma dilindungi rate limit per IP (signupThrottle), pola sama
// persis loginThrottle yang sudah dipakai handleAdminLogin/handleVendorLogin.
func (a *API) handleSignup(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.signupThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts", "terlalu banyak percobaan, coba lagi nanti")
		return
	}

	var req signupRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.BusinessName == "" || req.Email == "" || req.Username == "" {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "business_name, email, dan username wajib diisi")
		return
	}
	if len(req.Password) < minPasswordLen {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "password minimal 8 karakter")
		return
	}

	id, err := randomPrefixedAccountID()
	if err != nil {
		slog.Error("generate account id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	now := a.now()
	err = a.store.CreateAccount(r.Context(), store.CreateAccountInput{
		ID: id, BusinessName: req.BusinessName, Email: req.Email, Username: req.Username,
		PlaintextPassword: req.Password, Plan: signupPlan, MaxDevices: signupMaxDevices,
		ExpiresAt: now.Add(signupTrialDays * 24 * time.Hour),
	})
	if errors.Is(err, store.ErrAccountEmailTaken) {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusConflict, "email_taken", "email sudah dipakai akun lain")
		return
	}
	if errors.Is(err, store.ErrAccountUsernameTaken) {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusConflict, "username_taken", "username sudah dipakai akun lain")
		return
	}
	if err != nil {
		slog.Error("create account (signup) gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	a.signupThrottle.RecordSuccess(ip)

	if err := a.store.LogAudit(r.Context(), "signup", "ACCOUNT_CREATED", id,
		map[string]string{"business_name": req.BusinessName, "plan": signupPlan}); err != nil {
		slog.Error("log audit signup gagal", "err", err)
	}

	// Auto-login: langsung terbitkan cookie sesi, sama persis
	// handleAdminLogin -- customer tidak perlu login manual setelah daftar.
	token := auth.NewSessionToken(a.adminSessionKey, now, id)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.SessionDuration.Seconds()),
	})

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
