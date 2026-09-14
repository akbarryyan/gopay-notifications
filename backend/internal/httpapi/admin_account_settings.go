package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type setTelegramRequest struct {
	// TelegramChatID kosong berarti mencabut -- customer berhenti menerima
	// pengingat lewat Telegram, emailnya tetap jalan.
	TelegramChatID string `json:"telegram_chat_id"`
}

// handleAdminSetTelegram menyimpan chat id Telegram milik customer sendiri
// untuk pengingat kedaluwarsa. Email selalu dipakai (kolomnya wajib sejak
// akun dibuat); Telegram murni tambahan, jadi endpoint ini boleh diakses
// walau akun sedang tidak aktif (SENGAJA tanpa requireActiveAccount, sama
// seperti handleAdminLicense) -- justru saat akun mau/sudah berakhir
// pengingatnya paling dibutuhkan.
func (a *API) handleAdminSetTelegram(w http.ResponseWriter, r *http.Request) {
	accountID, ok := AccountFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak ditemukan")
		return
	}

	var req setTelegramRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}

	chatID := strings.TrimSpace(req.TelegramChatID)
	var value *string
	if chatID != "" {
		// Chat id Telegram selalu angka (boleh negatif untuk grup). Divalidasi
		// di sini supaya salah tempel -- mis. "@username", yang TIDAK bisa
		// dipakai Bot API tanpa chat pernah dimulai -- ketahuan sekarang,
		// bukan diam-diam bikin pengingat tidak pernah sampai.
		if !isTelegramChatID(chatID) {
			a.writeError(w, http.StatusBadRequest, "invalid_payload",
				"chat id Telegram harus berupa angka (dapat dari bot @userinfobot), bukan username")
			return
		}
		value = &chatID
	}

	if err := a.store.SetAccountTelegramChatID(r.Context(), accountID, value); errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "akun tidak ditemukan")
		return
	} else if err != nil {
		slog.Error("simpan telegram chat id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func isTelegramChatID(s string) bool {
	digits := strings.TrimPrefix(s, "-")
	if digits == "" {
		return false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
