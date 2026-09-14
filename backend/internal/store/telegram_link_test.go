package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestTelegramLinkCodeSekaliPakai(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC()

	expires, err := s.CreateTelegramLinkCode(ctx, "acc_1", tokenHash("kode-a"), now)
	if err != nil || !expires.Equal(now.Add(store.TelegramLinkTTL)) {
		t.Fatalf("CreateTelegramLinkCode = %v, %v", expires, err)
	}
	acc, err := s.ConsumeTelegramLinkCode(ctx, tokenHash("kode-a"), "987654321", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ConsumeTelegramLinkCode: %v", err)
	}
	if acc.ID != "acc_1" || acc.TelegramChatID == nil || *acc.TelegramChatID != "987654321" {
		t.Fatalf("account = %+v, mau chat id tersimpan", acc)
	}
	if _, err := s.ConsumeTelegramLinkCode(ctx, tokenHash("kode-a"), "111", now.Add(2*time.Minute)); !errors.Is(err, store.ErrTelegramLinkInvalid) {
		t.Fatalf("pakai ulang: err = %v, mau ErrTelegramLinkInvalid", err)
	}
	stored, _ := s.GetAccountByID(ctx, "acc_1")
	if *stored.TelegramChatID != "987654321" {
		t.Fatalf("chat id tertimpa pemakaian ulang kode: %v", *stored.TelegramChatID)
	}
}

func TestTelegramLinkCodeKedaluwarsaDanDigantikan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC()

	if _, err := s.CreateTelegramLinkCode(ctx, "acc_1", tokenHash("basi"), now); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.ConsumeTelegramLinkCode(ctx, tokenHash("basi"), "1", now.Add(store.TelegramLinkTTL+time.Second)); !errors.Is(err, store.ErrTelegramLinkInvalid) {
		t.Fatalf("kedaluwarsa: err = %v", err)
	}

	if _, err := s.CreateTelegramLinkCode(ctx, "acc_1", tokenHash("lama"), now); err != nil {
		t.Fatalf("create lama: %v", err)
	}
	if _, err := s.CreateTelegramLinkCode(ctx, "acc_1", tokenHash("baru"), now); err != nil {
		t.Fatalf("create baru: %v", err)
	}
	if _, err := s.ConsumeTelegramLinkCode(ctx, tokenHash("lama"), "1", now); !errors.Is(err, store.ErrTelegramLinkInvalid) {
		t.Fatalf("kode lama masih berlaku: err = %v", err)
	}
	if _, err := s.ConsumeTelegramLinkCode(ctx, tokenHash("baru"), "1", now); err != nil {
		t.Fatalf("kode terbaru: %v", err)
	}
}

func TestTelegramUpdateOffsetDiresetSaatTokenBerganti(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if off, err := s.GetTelegramUpdateOffset(ctx); err != nil || off != 0 {
		t.Fatalf("offset awal = %d, %v", off, err)
	}
	save := func(token *string) {
		t.Helper()
		if err := s.SaveNotificationSettings(ctx, settingsKey(), store.NotificationSettingsUpdate{
			SMTPPort: 587, TelegramBotToken: token, UpdatedBy: "akbar",
		}); err != nil {
			t.Fatalf("SaveNotificationSettings: %v", err)
		}
	}
	save(strPtr("1:bot-lama"))
	if err := s.SetTelegramUpdateOffset(ctx, 42); err != nil {
		t.Fatalf("SetTelegramUpdateOffset: %v", err)
	}

	save(nil) // token tidak diubah -> offset tetap
	if off, _ := s.GetTelegramUpdateOffset(ctx); off != 42 {
		t.Fatalf("offset = %d, mau tetap 42", off)
	}
	save(strPtr("2:bot-baru")) // token diganti -> offset kembali 0
	if off, _ := s.GetTelegramUpdateOffset(ctx); off != 0 {
		t.Fatalf("offset = %d, mau 0 setelah token diganti", off)
	}
}
