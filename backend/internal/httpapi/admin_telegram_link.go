package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/telegram"
)

// botUsernameCache menyimpan username bot per token -- tidak berubah selama
// tokennya sama, jadi getMe cukup sekali, bukan tiap customer klik tombol.
type botUsernameCache struct {
	mu      sync.Mutex
	byToken map[string]string
}

func (c *botUsernameCache) get(ctx context.Context, baseURL, token string) (string, error) {
	c.mu.Lock()
	username, ok := c.byToken[token]
	c.mu.Unlock()
	if ok {
		return username, nil
	}

	me, err := telegram.NewWithBaseURL(baseURL, token).GetMe(ctx)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.byToken[token] = me.Username
	c.mu.Unlock()
	return me.Username, nil
}

// handleAdminTelegramLink membuat deep link t.me/<bot>?start=<kode> untuk
// menghubungkan Telegram customer. Chat id-nya disimpan oleh
// internal/telegrambot begitu customer menekan Start di Telegram --
// dashboard cukup menunggu telegram_chat_id di GET /admin/account terisi.
func (a *API) handleAdminTelegramLink(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	settings, err := a.store.GetNotificationSettings(r.Context(), a.settingsSecretKey)
	if err != nil {
		slog.Error("baca notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if settings.TelegramBotToken == "" {
		a.writeError(w, http.StatusServiceUnavailable, "not_available", "notifikasi Telegram belum tersedia")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	botUsername, err := a.botUsernames.get(ctx, a.telegramBaseURL, settings.TelegramBotToken)
	if err != nil || botUsername == "" {
		slog.Error("ambil username bot telegram gagal", "err", err)
		a.writeError(w, http.StatusBadGateway, "telegram_unreachable",
			"tidak dapat menghubungi Telegram, coba lagi beberapa saat lagi")
		return
	}

	// 16 byte acak -> 22 karakter base64url, masuk batas parameter start
	// Telegram (maksimal 64 karakter [A-Za-z0-9_-]).
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		slog.Error("acak kode tautan gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	code := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(code))
	expires, err := a.store.CreateTelegramLinkCode(r.Context(), accountID, hash[:], a.now())
	if err != nil {
		slog.Error("simpan kode tautan telegram gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"url":          "https://t.me/" + botUsername + "?start=" + code,
		"bot_username": botUsername,
		"expires_at":   expires.Format(time.RFC3339),
	})
}
