package httpapi_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestAdminTelegramLink(t *testing.T) {
	var getMeCalls atomic.Int32
	tg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bot9:token-uji/getMe" {
			getMeCalls.Add(1)
			w.Write([]byte(`{"ok":true,"result":{"id":9,"username":"whuzpay_bot"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer tg.Close()

	s := newTestStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_1", BusinessName: "Toko Uji", Email: "acc_1@uji.test", Username: "admin",
		PlaintextPassword: testAdminPassword, Plan: "Business", MaxDevices: 10, ExpiresAt: fixedNow.Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(),
		func() time.Time { return fixedNow }).WithTelegramBaseURL(tg.URL).Handler()
	session := sessionCookieFrom(adminLogin(t, h, "admin", testAdminPassword))

	profile := func() (available bool, chatID *string) {
		t.Helper()
		var body struct {
			Account struct {
				TelegramAvailable bool    `json:"telegram_available"`
				TelegramChatID    *string `json:"telegram_chat_id"`
			} `json:"account"`
		}
		rec := adminGet(t, h, session, "/api/v1/admin/account")
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v (%s)", err, rec.Body.String())
		}
		return body.Account.TelegramAvailable, body.Account.TelegramChatID
	}

	// Token bot belum diisi vendor.
	if available, _ := profile(); available {
		t.Fatal("telegram_available = true padahal token bot kosong")
	}
	if rec := jsonRequest(t, h, session, http.MethodPost, "/api/v1/admin/account/telegram/link", ""); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("tanpa token status = %d, mau 503", rec.Code)
	}

	token := "9:token-uji"
	if err := s.SaveNotificationSettings(ctx, settingsSecretKey(), store.NotificationSettingsUpdate{
		SMTPPort: 587, TelegramBotToken: &token, UpdatedBy: "akbar",
	}); err != nil {
		t.Fatalf("SaveNotificationSettings: %v", err)
	}
	if available, _ := profile(); !available {
		t.Fatal("telegram_available = false padahal token bot terisi")
	}

	var link struct {
		URL         string `json:"url"`
		BotUsername string `json:"bot_username"`
	}
	for i := 0; i < 2; i++ {
		rec := jsonRequest(t, h, session, http.MethodPost, "/api/v1/admin/account/telegram/link", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("link status = %d body=%s", rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &link); err != nil {
			t.Fatalf("decode: %v", err)
		}
	}
	if getMeCalls.Load() != 1 {
		t.Fatalf("getMe dipanggil %d kali, mau 1 (di-cache per token)", getMeCalls.Load())
	}
	prefix := "https://t.me/whuzpay_bot?start="
	if link.BotUsername != "whuzpay_bot" || !strings.HasPrefix(link.URL, prefix) {
		t.Fatalf("link = %+v", link)
	}

	// Simulasikan bot menerima /start <kode> dari link terakhir.
	code := strings.TrimPrefix(link.URL, prefix)
	hash := sha256.Sum256([]byte(code))
	if _, err := s.ConsumeTelegramLinkCode(ctx, hash[:], "424242", fixedNow.Add(time.Minute)); err != nil {
		t.Fatalf("ConsumeTelegramLinkCode: %v", err)
	}
	if _, chatID := profile(); chatID == nil || *chatID != "424242" {
		t.Fatalf("telegram_chat_id = %v, mau 424242", chatID)
	}

	if rec := jsonRequest(t, h, nil, http.MethodPost, "/api/v1/admin/account/telegram/link", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa sesi status = %d, mau 401", rec.Code)
	}
}
