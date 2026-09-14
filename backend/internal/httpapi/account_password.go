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
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// resetRequestCooldown: satu account paling banyak menerima satu email
// reset per 2 menit, walau permintaannya datang dari banyak IP -- throttle
// per IP saja tidak mencegah alamat korban dibanjiri email.
const resetRequestCooldown = 2 * time.Minute

// backgroundSendTimeout membatasi pengiriman email di luar request.
const backgroundSendTimeout = 60 * time.Second

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// handleForgotPassword mengirim link reset ke email account.
//
// Jawabannya SELALU sama ("kalau terdaftar, link dikirim") untuk email
// terdaftar maupun tidak, dan emailnya dikirim di background -- kalau
// dikirim di dalam request, email terdaftar menjawab beberapa detik lebih
// lambat (percakapan SMTP) dan perbedaan waktu itu cukup untuk mengetahui
// email mana yang punya akun.
//
// Yang boleh dibedakan cuma "fitur belum tersedia" (DASHBOARD_URL atau SMTP
// belum diisi) -- itu keadaan server, bukan informasi tentang account.
func (a *API) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.passwordResetThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts",
			"terlalu banyak permintaan reset password, coba lagi nanti")
		return
	}
	// Setiap permintaan dihitung, bukan cuma yang gagal: yang dibatasi di
	// sini jumlah email yang bisa dipicu, dan permintaan "berhasil" justru
	// yang mengirim email.
	a.passwordResetThrottle.RecordFailure(ip, a.now())

	var req forgotPasswordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "email wajib diisi")
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
			"reset password lewat email belum tersedia, hubungi admin")
		return
	}

	ok := func() {
		writeJSON(w, http.StatusOK, map[string]any{"success": true})
	}

	acc, err := a.store.GetAccountByEmail(r.Context(), email)
	if errors.Is(err, store.ErrAccountNotFound) {
		ok()
		return
	}
	if err != nil {
		slog.Error("cari account untuk reset gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	// Account yang dicabut tidak akan pernah dipakai lagi -- tidak ada
	// gunanya mengirim link reset, dan jawabannya tetap disamakan.
	if acc.AdminStatus == "revoked" {
		ok()
		return
	}
	recent, err := a.store.PasswordResetRequestedSince(r.Context(), acc.ID, a.now().Add(-resetRequestCooldown))
	if err != nil {
		slog.Error("cek permintaan reset gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if recent {
		ok()
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

	// Token di FRAGMENT (#), bukan query string: fragment tidak pernah
	// dikirim browser ke server, jadi tidak tercatat di log akses Caddy/Next
	// maupun header Referer.
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
			slog.Error("kirim email reset password gagal", "account_id", accountID, "err", sendErr)
		}
	})
	ok()
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (a *API) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if len(req.NewPassword) < minPasswordLen {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "password baru minimal 8 karakter")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_token", "link reset tidak valid atau sudah kedaluwarsa")
		return
	}

	hash := sha256.Sum256([]byte(token))
	acc, err := a.store.ResetPasswordWithToken(r.Context(), hash[:], req.NewPassword, a.now())
	if errors.Is(err, store.ErrResetTokenInvalid) {
		a.writeError(w, http.StatusBadRequest, "invalid_token", "link reset tidak valid atau sudah kedaluwarsa")
		return
	}
	if err != nil {
		slog.Error("reset password gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	a.logActivity(r, acc.ID, store.ActivityPasswordReset, nil)
	a.notifyPasswordChanged(acc, true)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// notifyPasswordChanged mengabari pemilik akun di background. Dilewati
// diam-diam bila SMTP belum diisi -- ganti password tetap harus berhasil.
func (a *API) notifyPasswordChanged(acc store.Account, viaReset bool) {
	msg := notify.PasswordChanged{
		BusinessName: acc.BusinessName, Username: acc.Username, At: a.now(), ViaReset: viaReset,
	}.Message()
	to := notify.Recipient{Email: acc.Email}
	if acc.TelegramChatID != nil {
		to.TelegramChatID = *acc.TelegramChatID
	}
	accountID := acc.ID

	a.background(func() {
		ctx, cancel := context.WithTimeout(context.Background(), backgroundSendTimeout)
		defer cancel()
		settings, err := a.store.GetNotificationSettings(ctx, a.settingsSecretKey)
		if err != nil {
			slog.Error("baca notification settings gagal", "err", err)
			return
		}
		if !settings.EmailConfigured() {
			return
		}
		err = notify.Deliver(ctx, notifierFromSettings(settings), a.store, to, msg,
			notify.LogMeta{Kind: store.NotificationKindPasswordChanged, AccountID: &accountID})
		if err != nil {
			slog.Warn("kabar password diganti gagal dikirim", "account_id", accountID, "err", err)
		}
	})
}

func notifierFromSettings(s store.NotificationSettings) *notify.Notifier {
	return notify.New(notify.SMTPConfig{
		Host: s.SMTPHost, Port: s.SMTPPort, Username: s.SMTPUsername,
		Password: s.SMTPPassword, From: s.SMTPFrom,
	}, s.TelegramBotToken)
}
