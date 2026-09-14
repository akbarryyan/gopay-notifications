package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminSetTelegramChatID(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminPost(t, h, cookie, "/api/v1/admin/account/telegram", `{"telegram_chat_id":"123456789"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	licenseRec := adminGet(t, h, cookie, "/api/v1/admin/license")
	var body struct {
		License struct {
			Email          string  `json:"email"`
			TelegramChatID *string `json:"telegram_chat_id"`
		} `json:"license"`
	}
	json.Unmarshal(licenseRec.Body.Bytes(), &body)
	if body.License.TelegramChatID == nil || *body.License.TelegramChatID != "123456789" {
		t.Fatalf("telegram_chat_id = %v, mau 123456789", body.License.TelegramChatID)
	}
	if body.License.Email == "" {
		t.Fatal("email harus ikut dikirim supaya halaman bisa menyebut ke mana pengingat dikirim")
	}
}

func TestAdminHapusTelegramChatID(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	adminPost(t, h, cookie, "/api/v1/admin/account/telegram", `{"telegram_chat_id":"123456789"}`)
	rec := adminPost(t, h, cookie, "/api/v1/admin/account/telegram", `{"telegram_chat_id":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	licenseRec := adminGet(t, h, cookie, "/api/v1/admin/license")
	var body struct {
		License struct {
			TelegramChatID *string `json:"telegram_chat_id"`
		} `json:"license"`
	}
	json.Unmarshal(licenseRec.Body.Bytes(), &body)
	if body.License.TelegramChatID != nil {
		t.Fatalf("telegram_chat_id = %v, mau null setelah dicabut", body.License.TelegramChatID)
	}
}

func TestAdminSetTelegramUsernameDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	// "@username" tidak bisa dipakai Bot API tanpa chat pernah dimulai --
	// ditolak sekarang, bukan diam-diam bikin pengingat tidak pernah sampai.
	rec := adminPost(t, h, cookie, "/api/v1/admin/account/telegram", `{"telegram_chat_id":"@akbar"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestAdminSetTelegramButuhSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/account/telegram", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 tanpa sesi", rec.Code)
	}
}
