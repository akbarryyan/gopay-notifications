package store_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func settingsKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i*13 + 5)
	}
	return k
}

func strPtr(s string) *string { return &s }

func TestGetNotificationSettingsBelumPernahDisimpan(t *testing.T) {
	s := testStore(t)

	n, err := s.GetNotificationSettings(context.Background(), settingsKey())
	if err != nil {
		t.Fatalf("GetNotificationSettings: %v", err)
	}
	if n.SMTPPort != store.DefaultSMTPPort || n.SMTPHost != "" || n.UpdatedAt != nil {
		t.Fatalf("settings = %+v, mau kosong dengan port default", n)
	}
	if n.EmailConfigured() {
		t.Fatal("EmailConfigured = true padahal belum ada apa pun")
	}
}

func TestSaveNotificationSettingsRoundtripDanTerenkripsi(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	err := s.SaveNotificationSettings(ctx, settingsKey(), store.NotificationSettingsUpdate{
		SMTPHost: "smtp.uji.test", SMTPPort: 465, SMTPUsername: "user", SMTPFrom: "no-reply@uji.test",
		SMTPPassword: strPtr("password-smtp-rahasia"), TelegramBotToken: strPtr("123:token-rahasia"),
		UpdatedBy: "akbar",
	})
	if err != nil {
		t.Fatalf("SaveNotificationSettings: %v", err)
	}

	n, err := s.GetNotificationSettings(ctx, settingsKey())
	if err != nil {
		t.Fatalf("GetNotificationSettings: %v", err)
	}
	if n.SMTPHost != "smtp.uji.test" || n.SMTPPort != 465 || n.SMTPUsername != "user" ||
		n.SMTPFrom != "no-reply@uji.test" || n.SMTPPassword != "password-smtp-rahasia" ||
		n.TelegramBotToken != "123:token-rahasia" {
		t.Fatalf("settings = %+v, tidak sesuai yang disimpan", n)
	}
	if n.UpdatedBy == nil || *n.UpdatedBy != "akbar" || n.UpdatedAt == nil {
		t.Fatalf("updated_by/at = %v/%v, mau terisi", n.UpdatedBy, n.UpdatedAt)
	}
	if !n.EmailConfigured() {
		t.Fatal("EmailConfigured = false padahal host/port/from terisi")
	}

	var passwordEnc, botEnc []byte
	if err := s.Pool().QueryRow(ctx,
		`SELECT smtp_password_enc, telegram_bot_token_enc FROM notification_settings`).
		Scan(&passwordEnc, &botEnc); err != nil {
		t.Fatalf("query: %v", err)
	}
	if bytes.Contains(passwordEnc, []byte("password-smtp-rahasia")) || bytes.Contains(botEnc, []byte("token-rahasia")) {
		t.Fatal("kredensial tersimpan sebagai plaintext di database")
	}
}

func TestSaveNotificationSettingsNilMempertahankanKosongMenghapus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.SaveNotificationSettings(ctx, settingsKey(), store.NotificationSettingsUpdate{
		SMTPHost: "smtp.uji.test", SMTPPort: 587, SMTPFrom: "a@uji.test",
		SMTPPassword: strPtr("lama"), TelegramBotToken: strPtr("1:lama"), UpdatedBy: "akbar",
	}); err != nil {
		t.Fatalf("simpan pertama: %v", err)
	}

	// Cuma ubah host: password (nil) harus tetap, token ("") harus terhapus.
	if err := s.SaveNotificationSettings(ctx, settingsKey(), store.NotificationSettingsUpdate{
		SMTPHost: "smtp.baru.test", SMTPPort: 587, SMTPFrom: "a@uji.test",
		SMTPPassword: nil, TelegramBotToken: strPtr(""), UpdatedBy: "akbar",
	}); err != nil {
		t.Fatalf("simpan kedua: %v", err)
	}

	n, err := s.GetNotificationSettings(ctx, settingsKey())
	if err != nil {
		t.Fatalf("GetNotificationSettings: %v", err)
	}
	if n.SMTPHost != "smtp.baru.test" {
		t.Fatalf("host = %q, mau smtp.baru.test", n.SMTPHost)
	}
	if n.SMTPPassword != "lama" {
		t.Fatalf("password = %q, mau tetap \"lama\" -- nil berarti biarkan", n.SMTPPassword)
	}
	if n.TelegramBotToken != "" {
		t.Fatalf("token = %q, mau terhapus -- \"\" berarti hapus", n.TelegramBotToken)
	}
}

func TestGetNotificationSettingsKunciSalahGagal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.SaveNotificationSettings(ctx, settingsKey(), store.NotificationSettingsUpdate{
		SMTPHost: "smtp.uji.test", SMTPPort: 587, SMTPFrom: "a@uji.test",
		SMTPPassword: strPtr("rahasia"), UpdatedBy: "akbar",
	}); err != nil {
		t.Fatalf("SaveNotificationSettings: %v", err)
	}

	wrong := make([]byte, 32)
	if _, err := s.GetNotificationSettings(ctx, wrong); err == nil {
		t.Fatal("GetNotificationSettings dengan kunci salah berhasil, mau error")
	}
}
