package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
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
	signupTrialDays = 3
	// signupPlan HARUS tetap cocok nama salah satu baris di tabel plans
	// (dikelola vendor lewat halaman Plans) -- kuota device trial diambil
	// dari sana saat signup, bukan angka tetap lagi. Jangan mengganti nama
	// plan "Starter" atau menghapusnya tanpa mengganti nilai ini juga,
	// kalau tidak signup swalayan berhenti bekerja (menjawab 503).
	signupPlan     = "Starter"
	minPasswordLen = 8
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
	req.Email = strings.TrimSpace(req.Email)
	if req.BusinessName == "" || req.Email == "" || req.Username == "" {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "business_name, email, dan username wajib diisi")
		return
	}
	if !isEmailAddress(req.Email) {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "format email tidak valid")
		return
	}
	if len(req.Password) < minPasswordLen {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "password minimal 8 karakter")
		return
	}

	// Kuota device trial diambil dari plan "Starter" yang dikelola vendor
	// (halaman Plans) -- kalau vendor menghapus/mengganti nama plan itu
	// tanpa memperbarui konstanta signupPlan, signup swalayan sengaja
	// berhenti (503) alih-alih diam-diam memakai kuota yang salah.
	plan, err := a.store.GetPlanByName(r.Context(), signupPlan)
	if errors.Is(err, store.ErrPlanNotFound) {
		slog.Error("plan trial signup tidak ditemukan", "plan", signupPlan)
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusServiceUnavailable, "not_available",
			"pendaftaran sedang tidak tersedia, coba lagi nanti")
		return
	}
	if err != nil {
		slog.Error("ambil plan trial signup gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
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
		PlaintextPassword: req.Password, Plan: signupPlan, MaxDevices: plan.MaxDevices,
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

	// Best-effort, di background -- account customer sudah jadi dan harus
	// tetap bisa dipakai walau email verifikasi gagal terkirim (mis. SMTP
	// belum diisi vendor). Customer bisa minta kirim ulang dari Settings.
	a.sendVerificationEmail(r.Context(), store.Account{ID: id, BusinessName: req.BusinessName, Email: req.Email})

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
