package reminder_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/reminder"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type fakeStore struct {
	accounts []store.Account
	marked   []string
	markErr  error
	logged   []store.NotificationLogEntry
}

func (f *fakeStore) AccountsNeedingExpiryReminder(_ context.Context, _ time.Time, _ int) ([]store.Account, error) {
	return f.accounts, nil
}

func (f *fakeStore) MarkExpiryReminderSent(_ context.Context, id string, _ time.Time) error {
	if f.markErr != nil {
		return f.markErr
	}
	f.marked = append(f.marked, id)
	return nil
}

func (f *fakeStore) LogNotification(_ context.Context, e store.NotificationLogEntry) error {
	f.logged = append(f.logged, e)
	return nil
}

type fakeSender struct {
	sent        []string         // email tujuan yang berhasil
	telegramFor []string         // chat id yang berhasil
	errFor      map[string]error // per email
	tgErr       error
}

func (f *fakeSender) SendEmail(_ context.Context, to, _, _ string) error {
	if err, ok := f.errFor[to]; ok {
		return err
	}
	f.sent = append(f.sent, to)
	return nil
}

func (f *fakeSender) SendTelegram(_ context.Context, chatID, _ string) error {
	if f.tgErr != nil {
		return f.tgErr
	}
	f.telegramFor = append(f.telegramFor, chatID)
	return nil
}

func (f *fakeSender) TelegramEnabled() bool { return true }

func staticSender(s notify.Sender) notify.SenderFactory {
	return func(context.Context) (notify.Sender, bool, error) { return s, true, nil }
}

func akun(id, email string, telegram *string) store.Account {
	return store.Account{
		ID: id, BusinessName: "Toko " + id, Email: email,
		ExpiresAt: time.Now().Add(3 * 24 * time.Hour), TelegramChatID: telegram,
	}
}

func TestRunMengirimDanMenandaiSemuaAkun(t *testing.T) {
	chatID := "12345"
	st := &fakeStore{accounts: []store.Account{
		akun("acc_1", "a@t.test", nil),
		akun("acc_2", "b@t.test", &chatID),
	}}
	sender := &fakeSender{}

	sent, err := reminder.New(st, staticSender(sender), 7).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sent != 2 {
		t.Fatalf("sent = %d, mau 2", sent)
	}
	if len(st.marked) != 2 {
		t.Fatalf("marked = %v, mau dua akun ditandai", st.marked)
	}
	if len(sender.telegramFor) != 1 || sender.telegramFor[0] != chatID {
		t.Fatalf("telegram = %v, mau cuma ke %s", sender.telegramFor, chatID)
	}
	// 2 email + 1 telegram, semuanya tercatat sebagai expiry_reminder.
	if len(st.logged) != 3 {
		t.Fatalf("riwayat = %d baris, mau 3", len(st.logged))
	}
	for _, e := range st.logged {
		if e.Kind != store.NotificationKindExpiryReminder || e.AccountID == nil || e.Status != store.NotificationStatusSent {
			t.Fatalf("baris riwayat = %+v, mau expiry_reminder terkirim dengan account_id", e)
		}
	}
}

func TestRunSatuAkunGagalTidakMenghentikanSisanya(t *testing.T) {
	st := &fakeStore{accounts: []store.Account{
		akun("acc_gagal", "gagal@t.test", nil),
		akun("acc_sukses", "sukses@t.test", nil),
	}}
	sender := &fakeSender{errFor: map[string]error{"gagal@t.test": errors.New("smtp ditolak")}}

	sent, err := reminder.New(st, staticSender(sender), 7).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sent != 1 {
		t.Fatalf("sent = %d, mau 1 (yang gagal tidak dihitung)", sent)
	}
	// Yang gagal TIDAK ditandai, supaya putaran berikutnya mencoba lagi.
	if len(st.marked) != 1 || st.marked[0] != "acc_sukses" {
		t.Fatalf("marked = %v, mau cuma acc_sukses", st.marked)
	}
}

func TestRunTelegramGagalTetapDianggapTerkirim(t *testing.T) {
	chatID := "999"
	st := &fakeStore{accounts: []store.Account{akun("acc_1", "a@t.test", &chatID)}}
	// Email sudah terkirim, cuma Telegram yang gagal -- kalau ini dianggap
	// gagal total, pengingat yang emailnya sudah sampai akan dikirim ulang
	// terus tiap putaran.
	sender := &fakeSender{tgErr: errors.New("bot api 400")}

	sent, err := reminder.New(st, staticSender(sender), 7).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sent != 1 {
		t.Fatalf("sent = %d, mau 1", sent)
	}
	if len(st.marked) != 1 {
		t.Fatalf("marked = %v, mau ditandai walau Telegram gagal", st.marked)
	}
}

func TestRunGagalMenandaiTidakDihitungTerkirim(t *testing.T) {
	st := &fakeStore{
		accounts: []store.Account{akun("acc_1", "a@t.test", nil)},
		markErr:  errors.New("database sibuk"),
	}
	sender := &fakeSender{}

	sent, err := reminder.New(st, staticSender(sender), 7).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sent != 0 {
		t.Fatalf("sent = %d, mau 0 -- gagal ditandai berarti akan dikirim ulang", sent)
	}
}

func TestRunDilewatiBilaSMTPBelumDikonfigurasi(t *testing.T) {
	st := &fakeStore{accounts: []store.Account{akun("acc_1", "a@t.test", nil)}}
	sender := &fakeSender{}
	disabled := func(context.Context) (notify.Sender, bool, error) { return sender, false, nil }

	sent, err := reminder.New(st, disabled, 7).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sent != 0 || len(sender.sent) != 0 || len(st.marked) != 0 {
		t.Fatalf("sent=%d, dikirim=%d, ditandai=%d -- mau tidak ada apa pun saat SMTP belum dikonfigurasi",
			sent, len(sender.sent), len(st.marked))
	}
}

func TestRunMembacaPengaturanTiapPutaran(t *testing.T) {
	st := &fakeStore{accounts: []store.Account{akun("acc_1", "a@t.test", nil)}}
	sender := &fakeSender{}
	calls := 0
	enabled := false
	factory := func(context.Context) (notify.Sender, bool, error) {
		calls++
		return sender, enabled, nil
	}
	job := reminder.New(st, factory, 7)

	job.Run(context.Background(), time.Now()) // SMTP belum diisi
	enabled = true                            // vendor mengisi SMTP di dashboard
	sent, _ := job.Run(context.Background(), time.Now())

	if calls != 2 {
		t.Fatalf("factory dipanggil %d kali, mau 2 -- pengaturan harus dibaca ulang tiap putaran", calls)
	}
	if sent != 1 {
		t.Fatalf("sent = %d, mau 1 -- perubahan pengaturan harus berlaku tanpa restart", sent)
	}
}
