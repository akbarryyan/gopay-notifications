package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// sendVerificationEmail membuat token baru dan mengirim link verifikasi di
// background. Dipanggil dari signup dan dari ganti email di Settings.
// Gagal diam-diam (dicatat sebagai log error) -- alur yang memanggilnya
// (daftar, ganti profil) tidak boleh gagal cuma karena email verifikasi
// tidak terkirim; customer masih bisa minta kirim ulang dari Settings.
func (a *API) sendVerificationEmail(ctx context.Context, acc store.Account) {
	if a.dashboardURL == "" {
		return
	}
	settings, err := a.store.GetNotificationSettings(ctx, a.settingsSecretKey)
	if err != nil {
		slog.Error("baca notification settings gagal", "err", err)
		return
	}
	if !settings.EmailConfigured() {
		return
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		slog.Error("acak token verifikasi email gagal", "err", err)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	if _, err := a.store.CreateEmailVerificationToken(ctx, acc.ID, hash[:], a.now()); err != nil {
		slog.Error("simpan token verifikasi email gagal", "err", err)
		return
	}

	mail := notify.EmailVerification{
		BusinessName: acc.BusinessName,
		Link:         a.dashboardURL + "/verify-email?token=" + token,
	}
	notifier := notifierFromSettings(settings)
	accountID, to := acc.ID, acc.Email
	a.background(func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), backgroundSendTimeout)
		defer cancel()
		sendErr := notifier.SendEmail(bgCtx, to, mail.Subject(), mail.Body())
		notify.Record(bgCtx, a.store, notify.LogMeta{Kind: store.NotificationKindEmailVerification, AccountID: &accountID},
			"email", to, mail.Subject(), sendErr)
		if sendErr != nil {
			slog.Error("kirim email verifikasi gagal", "account_id", accountID, "err", sendErr)
		}
	})
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

// handleVerifyEmail memakai token dari link verifikasi. Publik (tidak
// dibungkus requireAdmin): link dibuka dari email, dan sesi di browser
// yang membukanya belum tentu sesi account yang sama (mis. dibuka di HP).
func (a *API) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_token", "link verifikasi tidak valid atau sudah kedaluwarsa")
		return
	}

	hash := sha256.Sum256([]byte(token))
	acc, err := a.store.ConsumeEmailVerificationToken(r.Context(), hash[:], a.now())
	if errors.Is(err, store.ErrEmailVerificationInvalid) {
		a.writeError(w, http.StatusBadRequest, "invalid_token", "link verifikasi tidak valid atau sudah kedaluwarsa")
		return
	}
	if err != nil {
		slog.Error("verifikasi email gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "business_name": acc.BusinessName})
}

// handleAdminResendVerificationEmail dipakai tombol "Kirim ulang" di
// Settings. Session sudah tahu account-nya, jadi tidak ada risiko
// enumerasi email seperti di handleForgotPassword -- cukup cooldown per
// account supaya tombol tidak bisa dipencet berkali-kali.
func (a *API) handleAdminResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	acc, err := a.store.GetAccountByID(r.Context(), accountID)
	if err != nil {
		slog.Error("ambil account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if acc.EmailVerifiedAt != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "already_verified": true})
		return
	}
	if a.dashboardURL == "" {
		a.writeError(w, http.StatusServiceUnavailable, "not_available", "verifikasi email belum tersedia")
		return
	}
	settings, err := a.store.GetNotificationSettings(r.Context(), a.settingsSecretKey)
	if err != nil {
		slog.Error("baca notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if !settings.EmailConfigured() {
		a.writeError(w, http.StatusServiceUnavailable, "not_available", "verifikasi email belum tersedia")
		return
	}

	recent, err := a.store.EmailVerificationRequestedSince(r.Context(), accountID, a.now().Add(-resetRequestCooldown))
	if err != nil {
		slog.Error("cek permintaan verifikasi gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if recent {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts",
			"tunggu sebentar sebelum meminta email verifikasi lagi")
		return
	}

	a.sendVerificationEmail(r.Context(), acc)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
