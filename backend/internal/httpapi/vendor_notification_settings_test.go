package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func vendorRequest(t *testing.T, h http.Handler, cookie *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	h.ServeHTTP(rec, req)
	return rec
}

type notificationSettingsResponse struct {
	Settings struct {
		SMTPHost            string  `json:"smtp_host"`
		SMTPPort            int     `json:"smtp_port"`
		SMTPUsername        string  `json:"smtp_username"`
		SMTPFrom            string  `json:"smtp_from"`
		SMTPPasswordSet     bool    `json:"smtp_password_set"`
		TelegramBotTokenSet bool    `json:"telegram_bot_token_set"`
		EmailConfigured     bool    `json:"email_configured"`
		UpdatedBy           *string `json:"updated_by"`
	} `json:"settings"`
}

func TestVendorNotificationSettingsButuhSesiVendor(t *testing.T) {
	h := newAPIWithVendor(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/vendor/settings/notifications"},
		{http.MethodPut, "/api/v1/vendor/settings/notifications"},
		{http.MethodPost, "/api/v1/vendor/settings/notifications/test"},
	} {
		rec := vendorRequest(t, h, nil, tc.method, tc.path, `{}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, mau 401", tc.method, tc.path, rec.Code)
		}
	}
}

func TestVendorNotificationSettingsSimpanTanpaMembocorkanKredensial(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	rec := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/settings/notifications", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET awal status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var awal notificationSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &awal); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if awal.Settings.SMTPPort != 587 || awal.Settings.EmailConfigured {
		t.Fatalf("settings awal = %+v, mau port 587 dan belum dikonfigurasi", awal.Settings)
	}

	rec = vendorRequest(t, h, cookie, http.MethodPut, "/api/v1/vendor/settings/notifications",
		`{"smtp_host":"smtp.uji.test","smtp_port":465,"smtp_username":"user","smtp_from":"no-reply@uji.test",
		  "smtp_password":"password-smtp-rahasia","telegram_bot_token":"123456:token-bot-rahasia"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d (body=%s)", rec.Code, rec.Body.String())
	}

	rec = vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/settings/notifications", "")
	body := rec.Body.String()
	if strings.Contains(body, "password-smtp-rahasia") || strings.Contains(body, "token-bot-rahasia") {
		t.Fatalf("response GET memuat kredensial plaintext: %s", body)
	}
	var got notificationSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	s := got.Settings
	if s.SMTPHost != "smtp.uji.test" || s.SMTPPort != 465 || s.SMTPFrom != "no-reply@uji.test" ||
		!s.SMTPPasswordSet || !s.TelegramBotTokenSet || !s.EmailConfigured {
		t.Fatalf("settings = %+v, tidak sesuai yang disimpan", s)
	}
	if s.UpdatedBy == nil || *s.UpdatedBy != "akbar" {
		t.Fatalf("updated_by = %v, mau akbar", s.UpdatedBy)
	}

	// Simpan ulang tanpa field password/token: keduanya harus tetap tersimpan.
	rec = vendorRequest(t, h, cookie, http.MethodPut, "/api/v1/vendor/settings/notifications",
		`{"smtp_host":"smtp.baru.test","smtp_port":587,"smtp_from":"no-reply@uji.test"}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Settings.SMTPPasswordSet || !got.Settings.TelegramBotTokenSet {
		t.Fatalf("settings = %+v, kredensial hilang padahal field tidak dikirim", got.Settings)
	}
}

func TestVendorNotificationSettingsValidasi(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	for name, body := range map[string]string{
		"port di luar rentang": `{"smtp_host":"smtp.uji.test","smtp_port":70000,"smtp_from":"a@uji.test"}`,
		"host tanpa pengirim":  `{"smtp_host":"smtp.uji.test","smtp_port":587}`,
		"pengirim bukan email": `{"smtp_host":"smtp.uji.test","smtp_port":587,"smtp_from":"bukan-email"}`,
		"token bot salah":      `{"telegram_bot_token":"username_bot"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := vendorRequest(t, h, cookie, http.MethodPut, "/api/v1/vendor/settings/notifications", body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestVendorTestNotificationBelumDikonfigurasi(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	for _, body := range []string{
		`{"channel":"email","to":"saya@uji.test"}`,
		`{"channel":"telegram","to":"12345"}`,
	} {
		rec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/settings/notifications/test", body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
		}
		if got := errorCode(t, rec); got != "not_configured" {
			t.Fatalf("error = %q, mau not_configured", got)
		}
	}
}
