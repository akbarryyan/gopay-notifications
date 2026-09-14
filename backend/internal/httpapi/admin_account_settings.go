package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
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

type accountProfileJSON struct {
	Username       string  `json:"username"`
	BusinessName   string  `json:"business_name"`
	Email          string  `json:"email"`
	TelegramChatID *string `json:"telegram_chat_id"`
	// TelegramAvailable false berarti vendor belum mengisi token bot --
	// tombol "Hubungkan Telegram" tidak ditampilkan.
	TelegramAvailable bool `json:"telegram_available"`
}

// handleAdminGetAccount melayani halaman Settings Customer Dashboard.
func (a *API) handleAdminGetAccount(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	acc, err := a.store.GetAccountByID(r.Context(), accountID)
	if errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "akun tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("ambil account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	settings, err := a.store.GetNotificationSettings(r.Context(), a.settingsSecretKey)
	if err != nil {
		slog.Error("baca notification settings gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "account": accountProfileJSON{
		Username: acc.Username, BusinessName: acc.BusinessName, Email: acc.Email, TelegramChatID: acc.TelegramChatID,
		TelegramAvailable: settings.TelegramBotToken != "",
	}})
}

type updateAccountRequest struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	// CurrentPassword wajib HANYA bila email berubah. Email adalah tujuan
	// link reset password -- sesi yang dicuri tidak boleh cukup untuk
	// memindahkannya ke alamat penyerang lalu mengambil alih akun lewat
	// "lupa password".
	CurrentPassword string `json:"current_password"`
}

func (a *API) handleAdminUpdateAccount(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	var req updateAccountRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	businessName := strings.TrimSpace(req.BusinessName)
	email := strings.TrimSpace(req.Email)
	if businessName == "" || email == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "nama bisnis dan email wajib diisi")
		return
	}
	if !isEmailAddress(email) {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "format email tidak valid")
		return
	}

	acc, err := a.store.GetAccountByID(r.Context(), accountID)
	if err != nil {
		slog.Error("ambil account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	if !strings.EqualFold(email, acc.Email) {
		if !a.passwordChangeThrottle.Allowed(accountID, a.now()) {
			a.writeError(w, http.StatusTooManyRequests, "too_many_attempts", "terlalu banyak percobaan, coba lagi nanti")
			return
		}
		if !acc.VerifyPassword(req.CurrentPassword) {
			a.passwordChangeThrottle.RecordFailure(accountID, a.now())
			a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "password saat ini salah")
			return
		}
	}

	err = a.store.UpdateAccountProfile(r.Context(), accountID, businessName, email)
	if errors.Is(err, store.ErrAccountEmailTaken) {
		a.writeError(w, http.StatusConflict, "email_taken", "email sudah dipakai akun lain")
		return
	}
	if err != nil {
		slog.Error("ubah profil account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "account": accountProfileJSON{
		Username: acc.Username, BusinessName: businessName, Email: email, TelegramChatID: acc.TelegramChatID,
	}})
}

type changeAccountPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// handleAdminChangePassword mengganti password dari Settings. Password saat
// ini wajib benar (dengan batas percobaan per account), seluruh sesi lain
// dicabut lewat password_changed_at, dan sesi yang sedang dipakai langsung
// diterbitkan ulang supaya yang mengganti tidak ikut ter-logout.
func (a *API) handleAdminChangePassword(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	var req changeAccountPasswordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if len(req.NewPassword) < minPasswordLen {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "password baru minimal 8 karakter")
		return
	}
	if !a.passwordChangeThrottle.Allowed(accountID, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts", "terlalu banyak percobaan, coba lagi nanti")
		return
	}

	acc, err := a.store.GetAccountByID(r.Context(), accountID)
	if err != nil {
		slog.Error("ambil account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if !acc.VerifyPassword(req.CurrentPassword) {
		a.passwordChangeThrottle.RecordFailure(accountID, a.now())
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "password saat ini salah")
		return
	}
	a.passwordChangeThrottle.RecordSuccess(accountID)

	if err := a.store.ChangeAccountPassword(r.Context(), accountID, req.NewPassword, a.now()); err != nil {
		slog.Error("ganti password account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	a.setAdminSessionCookie(w, r, accountID)
	a.notifyPasswordChanged(acc, false)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// isEmailAddress menerima alamat polos saja ("a@b.c"), bukan bentuk
// "Nama <a@b.c>" yang juga lolos mail.ParseAddress.
func isEmailAddress(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && strings.Contains(s[strings.LastIndex(s, "@"):], ".")
}
