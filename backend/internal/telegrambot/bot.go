// Package telegrambot membaca pesan yang masuk ke bot Telegram vendor dan
// menautkan chat customer ke account-nya lewat deep link
// t.me/<bot>?start=<kode>.
//
// Long polling (getUpdates), bukan webhook: tidak butuh URL publik, jadi
// sama persis di VPS maupun di laptop saat development. Konsekuensinya satu
// token bot cuma boleh dibaca SATU backend -- backend dev wajib memakai bot
// terpisah dari produksi (lihat ErrConflict).
package telegrambot

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
	"github.com/akbarryyan/gopay-notifications/backend/internal/telegram"
)

const (
	pollTimeout = 50 * time.Second
	// idleWait: jeda saat token bot belum diisi di Vendor Dashboard.
	idleWait = 30 * time.Second
	// errorWait: jeda setelah gagal, supaya tidak memborbardir Telegram
	// (atau log) saat jaringan/database bermasalah.
	errorWait = 15 * time.Second
)

// Store adalah bagian dari *store.Store yang dipakai bot.
type Store interface {
	GetNotificationSettings(ctx context.Context, key []byte) (store.NotificationSettings, error)
	GetTelegramUpdateOffset(ctx context.Context) (int64, error)
	SetTelegramUpdateOffset(ctx context.Context, offset int64) error
	ConsumeTelegramLinkCode(ctx context.Context, codeHash []byte, chatID string, now time.Time) (store.Account, error)
}

// Client adalah bagian dari *telegram.Client yang dipakai bot.
type Client interface {
	GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]telegram.Update, error)
	SendMessage(ctx context.Context, chatID int64, text string) error
}

type Bot struct {
	store     Store
	key       []byte
	newClient func(token string) Client
	now       func() time.Time

	// lastErr mencatat galat terakhir supaya galat yang sama (mis. konflik
	// dengan backend lain) tidak menulis log tiap 15 detik.
	lastErr string
}

func New(s Store, key []byte) *Bot {
	return &Bot{
		store: s, key: key, now: time.Now,
		newClient: func(token string) Client { return telegram.New(token) },
	}
}

// NewWithClient dipakai test.
func NewWithClient(s Store, key []byte, newClient func(token string) Client, now func() time.Time) *Bot {
	return &Bot{store: s, key: key, newClient: newClient, now: now}
}

// Run berjalan sampai ctx selesai.
func (b *Bot) Run(ctx context.Context) {
	for ctx.Err() == nil {
		polled, err := b.PollOnce(ctx)
		switch {
		case ctx.Err() != nil:
			return
		case err != nil:
			b.logErrorOnce(err)
			sleep(ctx, errorWait)
		case !polled:
			sleep(ctx, idleWait)
		default:
			b.lastErr = ""
		}
	}
}

// PollOnce menjalankan satu putaran getUpdates. polled=false berarti token
// bot belum diisi dan tidak ada yang dibaca.
//
// Pengaturan dibaca ulang setiap putaran supaya token yang baru diisi atau
// diganti di Vendor Dashboard langsung dipakai tanpa restart.
func (b *Bot) PollOnce(ctx context.Context) (polled bool, err error) {
	settings, err := b.store.GetNotificationSettings(ctx, b.key)
	if err != nil {
		return false, err
	}
	if settings.TelegramBotToken == "" {
		return false, nil
	}
	offset, err := b.store.GetTelegramUpdateOffset(ctx)
	if err != nil {
		return false, err
	}

	client := b.newClient(settings.TelegramBotToken)
	updates, err := client.GetUpdates(ctx, offset, pollTimeout)
	if err != nil {
		return true, err
	}

	for _, u := range updates {
		if err := b.handle(ctx, client, u); err != nil {
			// Berhenti TANPA memajukan offset: pesan ini dicoba lagi di
			// putaran berikutnya (mis. database sedang tidak bisa diakses).
			return true, err
		}
		if err := b.store.SetTelegramUpdateOffset(ctx, u.UpdateID+1); err != nil {
			return true, err
		}
	}
	return true, nil
}

func (b *Bot) handle(ctx context.Context, client Client, u telegram.Update) error {
	m := u.Message
	if m == nil {
		return nil
	}
	text := strings.TrimSpace(m.Text)
	if text != "/start" && !strings.HasPrefix(text, "/start ") {
		return nil // pesan lain diabaikan -- bot ini bukan bot percakapan
	}
	if m.Chat.Type != "private" {
		b.reply(ctx, client, m.Chat.ID, "Hubungkan Telegram lewat obrolan pribadi dengan bot ini, bukan dari grup.")
		return nil
	}

	code := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
	if code == "" {
		b.reply(ctx, client, m.Chat.ID,
			"Halo! Untuk menerima notifikasi Payment Bridge di sini, buka Settings di dashboard lalu klik \"Hubungkan Telegram\".")
		return nil
	}

	hash := sha256.Sum256([]byte(code))
	acc, err := b.store.ConsumeTelegramLinkCode(ctx, hash[:], strconv.FormatInt(m.Chat.ID, 10), b.now())
	if errors.Is(err, store.ErrTelegramLinkInvalid) {
		b.reply(ctx, client, m.Chat.ID,
			"Link ini sudah tidak berlaku (kedaluwarsa atau sudah dipakai). Buat link baru dari Settings di dashboard.")
		return nil
	}
	if err != nil {
		return err
	}

	slog.Info("telegram terhubung ke account", "account_id", acc.ID)
	b.reply(ctx, client, m.Chat.ID, fmt.Sprintf(
		"✅ Terhubung ke akun %s. Kamu akan menerima notifikasi Payment Bridge di sini: pengingat masa aktif, HP offline/online, dan pergantian password.",
		acc.BusinessName))
	return nil
}

// reply tidak menggagalkan putaran: chat id sudah tersimpan, balasan cuma
// konfirmasi.
func (b *Bot) reply(ctx context.Context, client Client, chatID int64, text string) {
	if err := client.SendMessage(ctx, chatID, text); err != nil {
		slog.Warn("balas pesan telegram gagal", "err", err)
	}
}

func (b *Bot) logErrorOnce(err error) {
	msg := err.Error()
	if msg == b.lastErr {
		return
	}
	b.lastErr = msg
	if errors.Is(err, telegram.ErrConflict) {
		slog.Error("bot telegram tidak bisa membaca pesan: token yang sama dipakai backend lain " +
			"(mis. backend dev di laptop) atau bot punya webhook aktif -- pakai bot terpisah per lingkungan")
		return
	}
	slog.Error("bot telegram gagal membaca pesan", "err", err)
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
