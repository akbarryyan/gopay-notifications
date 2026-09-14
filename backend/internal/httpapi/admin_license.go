package httpapi

import (
	"errors"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

const dateOnlyLayout = "2006-01-02"

// requireActiveAccount menolak request kalau account pemilik sesi/API
// key/device tidak operasional (cuma "active"). Menggantikan
// requireLicense (dulu baca file lokal + grace period -- sekarang query
// langsung ke accounts, tidak ada lagi jaringan antar dua service).
//
// SELALU ditaruh SETELAH requireAdmin/requireAPIKey/requireDevice di
// api.go -- account_id belum ada di context sebelum salah satu dari
// ketiganya lolos.
func (a *API) requireActiveAccount(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID, ok := AccountFromContext(r.Context())
		if !ok {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi/kredensial tidak ditemukan")
			return
		}
		acc, err := a.store.GetAccountByID(r.Context(), accountID)
		if errors.Is(err, store.ErrAccountNotFound) {
			a.writeError(w, http.StatusPaymentRequired, "account_not_found", "akun tidak ditemukan")
			return
		}
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
			return
		}
		if !acc.Operational(a.now()) {
			status := acc.DerivedStatus(a.now())
			a.writeError(w, http.StatusPaymentRequired, "account_"+status, accountErrorMessage(status))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func accountErrorMessage(status string) string {
	switch status {
	case "expired":
		return "akun sudah kedaluwarsa"
	case "suspended":
		return "akun sedang disuspend"
	case "revoked":
		return "akun sudah dicabut"
	default:
		return "akun tidak aktif"
	}
}

type accountJSON struct {
	BusinessName string `json:"business_name"`
	// Email ikut dikirim supaya halaman License bisa menyebut ke mana
	// pengingat kedaluwarsa dikirim -- tanpa itu customer tidak tahu
	// alamat mana yang dipakai.
	Email          string  `json:"email"`
	Plan           string  `json:"plan"`
	MaxDevices     int     `json:"max_devices"`
	ExpiresAt      string  `json:"expires_at"`
	DaysRemaining  int     `json:"days_remaining"`
	Status         string  `json:"status"`
	TelegramChatID *string `json:"telegram_chat_id"`
}

// handleAdminLicense mengembalikan status akun untuk Customer Dashboard.
// Nama fungsi & route (/admin/license) dipertahankan -- dashboard sudah
// punya halaman ini, ganti nama endpoint di tahap ini cuma menambah
// risiko tanpa manfaat. SENGAJA tidak dibungkus requireActiveAccount --
// admin harus selalu bisa melihat kenapa akunnya tidak aktif, bukan cuma
// saat aktif.
func (a *API) handleAdminLicense(w http.ResponseWriter, r *http.Request) {
	accountID, ok := AccountFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak ditemukan")
		return
	}
	acc, err := a.store.GetAccountByID(r.Context(), accountID)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	status := acc.DerivedStatus(a.now())
	daysRemaining := int(acc.ExpiresAt.Sub(a.now()).Hours() / 24)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "license": accountJSON{
		BusinessName: acc.BusinessName, Email: acc.Email, Plan: acc.Plan, MaxDevices: acc.MaxDevices,
		ExpiresAt: acc.ExpiresAt.Format(dateOnlyLayout), DaysRemaining: daysRemaining, Status: status,
		TelegramChatID: acc.TelegramChatID,
	}})
}
