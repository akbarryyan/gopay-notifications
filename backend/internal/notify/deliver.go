package notify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// Message adalah satu notifikasi dalam dua bentuk: email lengkap dan versi
// ringkas untuk Telegram (dibaca sambil lalu, jadi dipendekkan, bukan
// disalin mentah).
type Message struct {
	Subject      string
	EmailBody    string
	TelegramText string
}

// Recipient adalah kontak satu account. TelegramChatID kosong berarti
// customer tidak punya/tidak mengisi Telegram.
type Recipient struct {
	Email          string
	TelegramChatID string
}

// LogMeta adalah konteks yang ikut dicatat di riwayat notifikasi.
type LogMeta struct {
	Kind       string
	AccountID  *string
	DeviceID   *string
	DeviceName *string
}

// Logger mencatat satu percobaan kirim. Dipenuhi *store.Store.
type Logger interface {
	LogNotification(ctx context.Context, e store.NotificationLogEntry) error
}

// Deliver mengirim email lebih dulu, lalu Telegram bila token bot terisi
// DAN customer mengisi chat id-nya. Setiap channel dicatat ke riwayat
// sebagai baris sendiri.
//
// Email gagal = seluruhnya gagal (Telegram tidak dicoba): email adalah
// jalur utama yang pasti dimiliki semua customer, dan pemanggil akan
// mencoba ulang putaran berikutnya -- mengirim Telegram sekarang berarti
// customer menerimanya berulang kali selama email terus gagal.
//
// Telegram gagal setelah email terkirim dikembalikan sebagai
// *TelegramError: pemanggil tetap menganggapnya terkirim, supaya
// notifikasi yang emailnya sudah sampai tidak dikirim ulang terus-menerus.
func Deliver(ctx context.Context, sender Sender, logger Logger, to Recipient, msg Message, meta LogMeta) error {
	emailErr := sender.SendEmail(ctx, to.Email, msg.Subject, msg.EmailBody)
	record(ctx, logger, meta, "email", to.Email, msg.Subject, emailErr)
	if emailErr != nil {
		return fmt.Errorf("notify: kirim email ke %s: %w", to.Email, emailErr)
	}

	if to.TelegramChatID == "" || !sender.TelegramEnabled() {
		return nil
	}
	tgErr := sender.SendTelegram(ctx, to.TelegramChatID, msg.TelegramText)
	record(ctx, logger, meta, "telegram", to.TelegramChatID, msg.Subject, tgErr)
	if tgErr != nil {
		return &TelegramError{Err: tgErr}
	}
	return nil
}

// Record mencatat satu percobaan kirim yang tidak lewat Deliver (mis. pesan
// uji dari Vendor Dashboard). Gagal mencatat TIDAK menggagalkan pengiriman
// -- riwayat adalah alat bantu menelusuri, bukan bagian dari pesannya.
func Record(ctx context.Context, logger Logger, meta LogMeta, channel, recipient, subject string, sendErr error) {
	record(ctx, logger, meta, channel, recipient, subject, sendErr)
}

func record(ctx context.Context, logger Logger, meta LogMeta, channel, recipient, subject string, sendErr error) {
	entry := store.NotificationLogEntry{
		AccountID: meta.AccountID, DeviceID: meta.DeviceID, DeviceName: meta.DeviceName,
		Kind: meta.Kind, Channel: channel, Recipient: recipient, Subject: subject,
		Status: store.NotificationStatusSent,
	}
	if sendErr != nil {
		msg := sendErr.Error()
		entry.Status = store.NotificationStatusFailed
		entry.Error = &msg
	}
	// Konteks terpisah dari ctx pengiriman: kalau pengiriman gagal karena
	// tenggatnya habis, kegagalan itu justru yang paling perlu tercatat.
	if err := logger.LogNotification(context.WithoutCancel(ctx), entry); err != nil {
		slog.Error("catat riwayat notifikasi gagal", "kind", meta.Kind, "channel", channel, "err", err)
	}
}

// SenderFactory menyiapkan pengirim untuk SATU putaran pekerjaan berkala.
// Dipanggil ulang tiap putaran (bukan sekali saat start) supaya perubahan
// pengaturan SMTP/Telegram di Vendor Dashboard langsung berlaku tanpa
// restart backend. enabled=false berarti jalur utama (email) belum
// dikonfigurasi dan putaran dilewati.
type SenderFactory func(ctx context.Context) (sender Sender, enabled bool, err error)

// FromSettings adalah SenderFactory produksi: membaca pengaturan dari
// database (didekripsi dengan key) tiap putaran.
func FromSettings(s interface {
	GetNotificationSettings(ctx context.Context, key []byte) (store.NotificationSettings, error)
}, key []byte) SenderFactory {
	return func(ctx context.Context) (Sender, bool, error) {
		settings, err := s.GetNotificationSettings(ctx, key)
		if err != nil {
			return nil, false, err
		}
		if !settings.EmailConfigured() {
			return nil, false, nil
		}
		return New(SMTPConfig{
			Host:     settings.SMTPHost,
			Port:     settings.SMTPPort,
			Username: settings.SMTPUsername,
			Password: settings.SMTPPassword,
			From:     settings.SMTPFrom,
		}, settings.TelegramBotToken), true, nil
	}
}
