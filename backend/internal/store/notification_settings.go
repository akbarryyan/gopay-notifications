package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
	"github.com/jackc/pgx/v5"
)

// DefaultSMTPPort dipakai selama vendor belum pernah menyimpan pengaturan
// apa pun -- 587 (submission + STARTTLS), port paling umum di provider
// email manapun.
const DefaultSMTPPort = 587

// NotificationSettings adalah konfigurasi pengiriman pengingat kedaluwarsa,
// diatur vendor lewat Vendor Dashboard. SMTPPassword dan TelegramBotToken
// di sini SUDAH didekripsi -- struct ini tidak boleh pernah dikirim apa
// adanya lewat HTTP; lapisan HTTP cuma melaporkan apakah keduanya terisi.
type NotificationSettings struct {
	SMTPHost         string
	SMTPPort         int
	SMTPUsername     string
	SMTPPassword     string
	SMTPFrom         string
	TelegramBotToken string
	// UpdatedAt/UpdatedBy nil berarti belum pernah disimpan sama sekali.
	UpdatedAt *time.Time
	UpdatedBy *string
}

// EmailConfigured melaporkan apakah pengingat punya jalur kirim utama.
// Telegram sendiri tidak cukup -- tidak semua customer punya/mengisi
// Telegram, jadi tanpa SMTP sebagian customer tidak akan pernah diingatkan,
// dan pengingat yang cuma sampai ke sebagian orang lebih berbahaya daripada
// tidak ada sama sekali (bikin merasa sudah aman).
func (n NotificationSettings) EmailConfigured() bool {
	return n.SMTPHost != "" && n.SMTPPort != 0 && n.SMTPFrom != ""
}

// GetNotificationSettings membaca pengaturan dan mendekripsi kredensialnya.
// Belum pernah disimpan bukan error: dikembalikan nilai kosong dengan port
// default, supaya pemanggil (Vendor Dashboard, pekerjaan pengingat) tidak
// perlu membedakan "belum ada baris" dari "semua kolom kosong".
func (s *Store) GetNotificationSettings(ctx context.Context, key []byte) (NotificationSettings, error) {
	var (
		n                   NotificationSettings
		passwordEnc, botEnc []byte
		updatedAt           time.Time
		updatedBy           *string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT smtp_host, smtp_port, smtp_username, smtp_password_enc, smtp_from,
		        telegram_bot_token_enc, updated_at, updated_by
		 FROM notification_settings WHERE id = TRUE`).
		Scan(&n.SMTPHost, &n.SMTPPort, &n.SMTPUsername, &passwordEnc, &n.SMTPFrom,
			&botEnc, &updatedAt, &updatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return NotificationSettings{SMTPPort: DefaultSMTPPort}, nil
	}
	if err != nil {
		return NotificationSettings{}, fmt.Errorf("store: baca notification settings: %w", err)
	}

	if passwordEnc != nil {
		plain, err := secretbox.Open(key, passwordEnc)
		if err != nil {
			return NotificationSettings{}, fmt.Errorf("store: dekripsi password smtp: %w", err)
		}
		n.SMTPPassword = string(plain)
	}
	if botEnc != nil {
		plain, err := secretbox.Open(key, botEnc)
		if err != nil {
			return NotificationSettings{}, fmt.Errorf("store: dekripsi token bot telegram: %w", err)
		}
		n.TelegramBotToken = string(plain)
	}
	n.UpdatedAt = &updatedAt
	n.UpdatedBy = updatedBy
	return n, nil
}

// NotificationSettingsUpdate adalah perubahan yang dikirim vendor.
//
// SMTPPassword dan TelegramBotToken bertiga-nilai, sengaja: nil = biarkan
// yang tersimpan, "" = hapus, selain itu = ganti. Tanpa pembeda "biarkan",
// form di dashboard (yang tidak pernah menerima kredensial lama untuk
// ditampilkan ulang) akan menghapus password tiap kali vendor cuma
// mengubah host atau port.
type NotificationSettingsUpdate struct {
	SMTPHost         string
	SMTPPort         int
	SMTPUsername     string
	SMTPFrom         string
	SMTPPassword     *string
	TelegramBotToken *string
	UpdatedBy        string
}

// sealOptional mengenkripsi kredensial yang akan disimpan. Mengembalikan
// (nil, false) untuk "biarkan", (nil, true) untuk "hapus" (ditulis NULL).
func sealOptional(key []byte, v *string) (enc []byte, changed bool, err error) {
	if v == nil {
		return nil, false, nil
	}
	if *v == "" {
		return nil, true, nil
	}
	enc, err = secretbox.Seal(key, []byte(*v))
	if err != nil {
		return nil, false, err
	}
	return enc, true, nil
}

// SaveNotificationSettings menyimpan pengaturan dalam satu upsert.
// Kolom kredensial cuma ditimpa bila flag "changed"-nya true -- CASE di
// bawah yang membuat nil ("biarkan") benar-benar mempertahankan nilai lama.
func (s *Store) SaveNotificationSettings(ctx context.Context, key []byte, u NotificationSettingsUpdate) error {
	passwordEnc, passwordChanged, err := sealOptional(key, u.SMTPPassword)
	if err != nil {
		return fmt.Errorf("store: enkripsi password smtp: %w", err)
	}
	botEnc, botChanged, err := sealOptional(key, u.TelegramBotToken)
	if err != nil {
		return fmt.Errorf("store: enkripsi token bot telegram: %w", err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO notification_settings
		   (id, smtp_host, smtp_port, smtp_username, smtp_from,
		    smtp_password_enc, telegram_bot_token_enc, updated_at, updated_by)
		 VALUES (TRUE, $1, $2, $3, $4, $5, $6, now(), $9)
		 ON CONFLICT (id) DO UPDATE SET
		   smtp_host     = EXCLUDED.smtp_host,
		   smtp_port     = EXCLUDED.smtp_port,
		   smtp_username = EXCLUDED.smtp_username,
		   smtp_from     = EXCLUDED.smtp_from,
		   smtp_password_enc = CASE WHEN $7::boolean
		     THEN EXCLUDED.smtp_password_enc ELSE notification_settings.smtp_password_enc END,
		   telegram_bot_token_enc = CASE WHEN $8::boolean
		     THEN EXCLUDED.telegram_bot_token_enc ELSE notification_settings.telegram_bot_token_enc END,
		   telegram_update_offset = CASE WHEN $8::boolean
		     THEN 0 ELSE notification_settings.telegram_update_offset END,
		   updated_at = now(),
		   updated_by = EXCLUDED.updated_by`,
		u.SMTPHost, u.SMTPPort, u.SMTPUsername, u.SMTPFrom,
		passwordEnc, botEnc, passwordChanged, botChanged, u.UpdatedBy)
	if err != nil {
		return fmt.Errorf("store: simpan notification settings: %w", err)
	}
	return nil
}
