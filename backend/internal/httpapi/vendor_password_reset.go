package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// handleVendorSendPasswordReset mengirim link reset password ke email
// account -- dipakai vendor untuk membantu customer yang terkunci
// (lupa password sekaligus lupa akses ke email lamanya, atau minta bantuan
// langsung ke Akbar) tanpa vendor perlu tahu atau mengganti passwordnya
// sendiri.
//
// Beda dari handleForgotPassword (endpoint publik untuk customer sendiri):
// tidak ada risiko enumerasi di sini -- vendor sudah login dan memilih
// account ini sendiri dari daftar -- jadi errornya boleh spesifik
// (not_available, account_revoked), bukan disamarkan jadi "sukses" semua.
func (a *API) handleVendorSendPasswordReset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("accountID")

	acc, err := a.store.GetAccountByID(r.Context(), id)
	if errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("ambil account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	// Account yang dicabut tidak akan pernah dipakai lagi -- tidak ada
	// gunanya mengirim link reset.
	if acc.AdminStatus == "revoked" {
		a.writeError(w, http.StatusConflict, "account_revoked", "account sudah dicabut, tidak bisa dikirimi link reset")
		return
	}

	settings, err := a.store.GetNotificationSettings(r.Context(), a.settingsSecretKey)
	if err != nil {
		slog.Error("baca notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if a.dashboardURL == "" || !settings.EmailConfigured() {
		a.writeError(w, http.StatusServiceUnavailable, "not_available",
			"reset password lewat email belum tersedia -- isi SMTP di Settings dulu")
		return
	}

	// Cooldown yang sama dengan handleForgotPassword: mencegah tombol ini
	// dipencet berkali-kali membanjiri email customer.
	recent, err := a.store.PasswordResetRequestedSince(r.Context(), acc.ID, a.now().Add(-resetRequestCooldown))
	if err != nil {
		slog.Error("cek permintaan reset gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if recent {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts",
			"link reset baru saja dikirim, tunggu beberapa menit sebelum mengirim lagi")
		return
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		slog.Error("acak token reset gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	if err := a.store.CreatePasswordResetToken(r.Context(), acc.ID, hash[:], a.now()); err != nil {
		slog.Error("simpan token reset gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	mail := notify.PasswordResetEmail{
		BusinessName: acc.BusinessName, Username: acc.Username,
		Link:     a.dashboardURL + "/reset-password#token=" + token,
		ValidFor: store.PasswordResetTTL,
	}
	notifier := notifierFromSettings(settings)
	accountID, to := acc.ID, acc.Email
	a.background(func() {
		ctx, cancel := context.WithTimeout(context.Background(), backgroundSendTimeout)
		defer cancel()
		sendErr := notifier.SendEmail(ctx, to, mail.Subject(), mail.Body())
		notify.Record(ctx, a.store, notify.LogMeta{Kind: store.NotificationKindPasswordReset, AccountID: &accountID},
			"email", to, mail.Subject(), sendErr)
		if sendErr != nil {
			slog.Error("kirim email reset password (vendor) gagal", "account_id", accountID, "err", sendErr)
		}
	})

	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "PASSWORD_RESET_SENT", id, nil); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
