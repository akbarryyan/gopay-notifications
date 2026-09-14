package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// notificationSettingsJSON SENGAJA tidak punya field password atau token
// sama sekali -- cuma apakah keduanya terisi. Kredensial yang sudah
// tersimpan tidak pernah dikirim balik lewat HTTP, bahkan ke vendor
// sendiri: layar yang bisa menampilkan password adalah layar yang bisa
// dipotret, di-screenshot, atau tertinggal terbuka.
type notificationSettingsJSON struct {
	SMTPHost            string  `json:"smtp_host"`
	SMTPPort            int     `json:"smtp_port"`
	SMTPUsername        string  `json:"smtp_username"`
	SMTPFrom            string  `json:"smtp_from"`
	SMTPPasswordSet     bool    `json:"smtp_password_set"`
	TelegramBotTokenSet bool    `json:"telegram_bot_token_set"`
	EmailConfigured     bool    `json:"email_configured"`
	UpdatedAt           *string `json:"updated_at"`
	UpdatedBy           *string `json:"updated_by"`
}

func toNotificationSettingsJSON(n store.NotificationSettings) notificationSettingsJSON {
	out := notificationSettingsJSON{
		SMTPHost: n.SMTPHost, SMTPPort: n.SMTPPort, SMTPUsername: n.SMTPUsername, SMTPFrom: n.SMTPFrom,
		SMTPPasswordSet:     n.SMTPPassword != "",
		TelegramBotTokenSet: n.TelegramBotToken != "",
		EmailConfigured:     n.EmailConfigured(),
		UpdatedBy:           n.UpdatedBy,
	}
	if n.UpdatedAt != nil {
		s := n.UpdatedAt.Format(time.RFC3339)
		out.UpdatedAt = &s
	}
	return out
}

func (a *API) handleVendorGetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := a.store.GetNotificationSettings(r.Context(), a.settingsSecretKey)
	if err != nil {
		slog.Error("baca notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "settings": toNotificationSettingsJSON(settings)})
}

type saveNotificationSettingsRequest struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPFrom     string `json:"smtp_from"`
	// Pointer supaya tiga keadaan terbedakan: field tidak dikirim/null =
	// biarkan yang tersimpan, "" = hapus, isi = ganti. Lihat
	// store.NotificationSettingsUpdate.
	SMTPPassword     *string `json:"smtp_password"`
	TelegramBotToken *string `json:"telegram_bot_token"`
}

func (a *API) handleVendorSaveNotificationSettings(w http.ResponseWriter, r *http.Request) {
	var req saveNotificationSettingsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}

	req.SMTPHost = strings.TrimSpace(req.SMTPHost)
	req.SMTPUsername = strings.TrimSpace(req.SMTPUsername)
	req.SMTPFrom = strings.TrimSpace(req.SMTPFrom)
	if req.SMTPPort == 0 {
		req.SMTPPort = store.DefaultSMTPPort
	}
	if req.SMTPPort < 1 || req.SMTPPort > 65535 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "port SMTP harus 1..65535")
		return
	}
	// Host terisi tapi alamat pengirim kosong adalah konfigurasi setengah
	// jadi yang diam-diam membuat pengingat tidak pernah aktif -- ditolak
	// sekarang, bukan ketahuan seminggu kemudian. Mengosongkan host (untuk
	// mematikan pengingat) tetap boleh.
	if req.SMTPHost != "" && req.SMTPFrom == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "alamat pengirim wajib diisi bila host SMTP diisi")
		return
	}
	if req.SMTPFrom != "" && !strings.Contains(req.SMTPFrom, "@") {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			`alamat pengirim harus berupa email, mis. "no-reply@domain.com" atau "Nama <no-reply@domain.com>"`)
		return
	}
	if req.TelegramBotToken != nil {
		trimmed := strings.TrimSpace(*req.TelegramBotToken)
		// Token bot selalu berbentuk "<angka>:<rahasia>" -- salah tempel
		// (mis. username bot) ketahuan sekarang.
		if trimmed != "" && !strings.Contains(trimmed, ":") {
			a.writeError(w, http.StatusBadRequest, "invalid_payload",
				"token bot Telegram tidak valid -- bentuknya 123456:ABC-DEF..., didapat dari @BotFather")
			return
		}
		req.TelegramBotToken = &trimmed
	}

	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.SaveNotificationSettings(r.Context(), a.settingsSecretKey, store.NotificationSettingsUpdate{
		SMTPHost: req.SMTPHost, SMTPPort: req.SMTPPort, SMTPUsername: req.SMTPUsername, SMTPFrom: req.SMTPFrom,
		SMTPPassword: req.SMTPPassword, TelegramBotToken: req.TelegramBotToken,
		UpdatedBy: vendorUsername,
	}); err != nil {
		slog.Error("simpan notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	// Audit tanpa nilai kredensial -- cuma apakah berubah.
	if err := a.store.LogAudit(r.Context(), vendorUsername, "NOTIFICATION_SETTINGS_UPDATED", "notification_settings",
		map[string]any{
			"smtp_host": req.SMTPHost, "smtp_port": req.SMTPPort, "smtp_from": req.SMTPFrom,
			"smtp_password_changed":      req.SMTPPassword != nil,
			"telegram_bot_token_changed": req.TelegramBotToken != nil,
		}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	settings, err := a.store.GetNotificationSettings(r.Context(), a.settingsSecretKey)
	if err != nil {
		slog.Error("baca ulang notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "settings": toNotificationSettingsJSON(settings)})
}

type testNotificationRequest struct {
	Channel string `json:"channel"` // "email" | "telegram"
	To      string `json:"to"`
}

// handleVendorTestNotification mengirim pesan uji memakai pengaturan yang
// SUDAH TERSIMPAN (bukan isi form yang belum disimpan) -- yang diuji harus
// persis konfigurasi yang nanti dipakai pekerjaan pengingat.
//
// Pesan galat SMTP/Telegram diteruskan apa adanya ke vendor: tanpa alasan
// sebenarnya ("535 authentication failed", "chat not found") salah
// konfigurasi mustahil ditelusuri. Aman karena endpoint ini khusus vendor,
// dan galat Telegram sudah dibersihkan dari token di internal/notify.
func (a *API) handleVendorTestNotification(w http.ResponseWriter, r *http.Request) {
	var req testNotificationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	req.To = strings.TrimSpace(req.To)
	if req.To == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "tujuan pesan uji wajib diisi")
		return
	}

	settings, err := a.store.GetNotificationSettings(r.Context(), a.settingsSecretKey)
	if err != nil {
		slog.Error("baca notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	notifier := notify.New(notify.SMTPConfig{
		Host: settings.SMTPHost, Port: settings.SMTPPort, Username: settings.SMTPUsername,
		Password: settings.SMTPPassword, From: settings.SMTPFrom,
	}, settings.TelegramBotToken)

	// Tenggat tersendiri: request HTTP ini dipegang vendor yang sedang
	// menunggu di layar, jangan sampai menggantung lebih lama dari ini.
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	switch req.Channel {
	case "email":
		if !settings.EmailConfigured() {
			a.writeError(w, http.StatusBadRequest, "not_configured", "SMTP belum dikonfigurasi -- simpan pengaturan dulu")
			return
		}
		err = notifier.SendTestEmail(ctx, req.To)
		notify.Record(ctx, a.store, notify.LogMeta{Kind: store.NotificationKindTest}, "email", req.To, notify.TestEmailSubject, err)
	case "telegram":
		if settings.TelegramBotToken == "" {
			a.writeError(w, http.StatusBadRequest, "not_configured", "token bot Telegram belum disimpan")
			return
		}
		if !isTelegramChatID(req.To) {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "chat id Telegram harus berupa angka")
			return
		}
		err = notifier.SendTestTelegram(ctx, req.To)
		notify.Record(ctx, a.store, notify.LogMeta{Kind: store.NotificationKindTest}, "telegram", req.To, "Pesan uji Telegram", err)
	default:
		a.writeError(w, http.StatusBadRequest, "invalid_payload", `channel harus "email" atau "telegram"`)
		return
	}

	if err != nil {
		a.writeError(w, http.StatusBadGateway, "send_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
