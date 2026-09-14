package notify_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type fakeSender struct {
	emailErr, tgErr error
	tgEnabled       bool
	emails, tgs     int
}

func (f *fakeSender) SendEmail(context.Context, string, string, string) error {
	f.emails++
	return f.emailErr
}

func (f *fakeSender) SendTelegram(context.Context, string, string) error {
	f.tgs++
	return f.tgErr
}

func (f *fakeSender) TelegramEnabled() bool { return f.tgEnabled }

type fakeLogger struct{ entries []store.NotificationLogEntry }

func (f *fakeLogger) LogNotification(_ context.Context, e store.NotificationLogEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

var testMsg = notify.Message{Subject: "Subjek", EmailBody: "isi", TelegramText: "ringkas"}

func TestDeliverEmailDanTelegramDicatatTerpisah(t *testing.T) {
	sender := &fakeSender{tgEnabled: true}
	logger := &fakeLogger{}
	acc := "acc_1"

	err := notify.Deliver(context.Background(), sender, logger,
		notify.Recipient{Email: "a@t.test", TelegramChatID: "123"}, testMsg,
		notify.LogMeta{Kind: store.NotificationKindDeviceOffline, AccountID: &acc})
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if len(logger.entries) != 2 {
		t.Fatalf("riwayat = %d baris, mau 2", len(logger.entries))
	}
	if logger.entries[0].Channel != "email" || logger.entries[0].Recipient != "a@t.test" ||
		logger.entries[1].Channel != "telegram" || logger.entries[1].Recipient != "123" {
		t.Fatalf("riwayat = %+v, mau email lalu telegram", logger.entries)
	}
	for _, e := range logger.entries {
		if e.Status != store.NotificationStatusSent || e.Error != nil || *e.AccountID != acc {
			t.Fatalf("baris = %+v, mau terkirim tanpa error", e)
		}
	}
}

func TestDeliverEmailGagalTelegramTidakDicoba(t *testing.T) {
	sender := &fakeSender{tgEnabled: true, emailErr: errors.New("535 authentication failed")}
	logger := &fakeLogger{}

	err := notify.Deliver(context.Background(), sender, logger,
		notify.Recipient{Email: "a@t.test", TelegramChatID: "123"}, testMsg, notify.LogMeta{Kind: "expiry_reminder"})

	var tgErr *notify.TelegramError
	if err == nil || errors.As(err, &tgErr) {
		t.Fatalf("err = %v, mau error email (bukan TelegramError)", err)
	}
	if sender.tgs != 0 {
		t.Fatal("Telegram dicoba padahal email gagal -- akan terkirim berulang tiap putaran")
	}
	if len(logger.entries) != 1 || logger.entries[0].Status != store.NotificationStatusFailed ||
		logger.entries[0].Error == nil || !strings.Contains(*logger.entries[0].Error, "535") {
		t.Fatalf("riwayat = %+v, mau satu baris gagal dengan alasan", logger.entries)
	}
}

func TestDeliverTelegramGagalMengembalikanTelegramError(t *testing.T) {
	sender := &fakeSender{tgEnabled: true, tgErr: errors.New("chat not found")}
	logger := &fakeLogger{}

	err := notify.Deliver(context.Background(), sender, logger,
		notify.Recipient{Email: "a@t.test", TelegramChatID: "123"}, testMsg, notify.LogMeta{Kind: "expiry_reminder"})

	var tgErr *notify.TelegramError
	if !errors.As(err, &tgErr) {
		t.Fatalf("err = %v, mau TelegramError", err)
	}
	if len(logger.entries) != 2 || logger.entries[0].Status != store.NotificationStatusSent ||
		logger.entries[1].Status != store.NotificationStatusFailed {
		t.Fatalf("riwayat = %+v, mau email terkirim + telegram gagal", logger.entries)
	}
}

func TestDeliverTanpaChatIDAtauTokenBotCumaEmail(t *testing.T) {
	for name, tc := range map[string]struct {
		chatID    string
		tgEnabled bool
	}{
		"customer tanpa telegram": {"", true},
		"token bot belum diisi":   {"123", false},
	} {
		t.Run(name, func(t *testing.T) {
			sender := &fakeSender{tgEnabled: tc.tgEnabled}
			logger := &fakeLogger{}
			err := notify.Deliver(context.Background(), sender, logger,
				notify.Recipient{Email: "a@t.test", TelegramChatID: tc.chatID}, testMsg, notify.LogMeta{Kind: "expiry_reminder"})
			if err != nil {
				t.Fatalf("Deliver: %v", err)
			}
			if sender.tgs != 0 || len(logger.entries) != 1 {
				t.Fatalf("telegram=%d, riwayat=%d, mau cuma email tercatat", sender.tgs, len(logger.entries))
			}
		})
	}
}

func TestPesanStatusHPMemakaiJamWIB(t *testing.T) {
	last := time.Date(2026, 9, 14, 3, 30, 0, 0, time.UTC) // 10.30 WIB
	msg := notify.DeviceOffline{BusinessName: "Toko Maju", DeviceName: "HP Kasir", LastHeartbeatAt: last}.Message()

	if !strings.Contains(msg.Subject, "HP Kasir") {
		t.Fatalf("subject = %q, mau menyebut nama HP", msg.Subject)
	}
	for _, part := range []string{msg.EmailBody, msg.TelegramText} {
		if !strings.Contains(part, "14 September 2026 10.30 WIB") {
			t.Fatalf("pesan = %q, mau jam dalam WIB", part)
		}
	}
	if !strings.Contains(msg.EmailBody, "Halo Toko Maju") {
		t.Fatalf("email = %q, mau menyapa nama bisnis", msg.EmailBody)
	}

	back := notify.DeviceOnline{BusinessName: "Toko Maju", DeviceName: "HP Kasir",
		OfflineSince: last, BackAt: last.Add(2 * time.Hour)}.Message()
	if !strings.Contains(back.Subject, "kembali online") || !strings.Contains(back.EmailBody, "12.30 WIB") {
		t.Fatalf("pesan online = %+v", back)
	}
}
