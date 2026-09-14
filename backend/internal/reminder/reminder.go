// Package reminder menjalankan pengingat "akun mendekati kedaluwarsa" ke
// customer. Dipisah dari internal/httpapi karena tidak ada hubungannya
// dengan HTTP: ini pekerjaan berkala yang membaca database dan mengirim
// pesan keluar.
package reminder

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// DefaultWithinDays: pengingat dikirim saat sisa masa aktif <= 7 hari.
//
// Pengingat ini AKTIF menghampiri orang lewat email/Telegram -- terlalu
// dini (mis. sebulan sebelumnya) gampang dilupakan begitu saja dan sudah
// tidak diingat lagi saat benar-benar mendesak.
const DefaultWithinDays = 7

// Store adalah bagian dari *store.Store yang dipakai pekerjaan ini --
// dipersempit jadi interface supaya bisa diuji tanpa database.
type Store interface {
	AccountsNeedingExpiryReminder(ctx context.Context, now time.Time, withinDays int) ([]store.Account, error)
	MarkExpiryReminderSent(ctx context.Context, id string, expiresAt time.Time) error
	notify.Logger
}

type Job struct {
	store      Store
	newSender  notify.SenderFactory
	withinDays int

	// lastEnabled mencatat status putaran sebelumnya, supaya "pengingat
	// tidak aktif" dicatat saat berubah saja -- bukan tiap jam memenuhi log,
	// tapi juga tidak diam-diam mati tanpa jejak sama sekali.
	lastEnabled *bool
}

func New(s Store, newSender notify.SenderFactory, withinDays int) *Job {
	if withinDays <= 0 {
		withinDays = DefaultWithinDays
	}
	return &Job{store: s, newSender: newSender, withinDays: withinDays}
}

// Run mengirim pengingat untuk seluruh akun yang jatuh tempo dan
// mengembalikan berapa yang terkirim.
//
// Kegagalan satu akun TIDAK menghentikan sisanya: satu alamat email yang
// ditolak server tidak boleh membuat customer lain ikut tidak diingatkan.
// Akun yang gagal juga sengaja TIDAK ditandai terkirim, jadi putaran
// berikutnya mencobanya lagi.
func (j *Job) Run(ctx context.Context, now time.Time) (sent int, err error) {
	sender, enabled, err := j.newSender(ctx)
	if err != nil {
		return 0, err
	}
	j.logEnabledTransition(enabled)
	if !enabled {
		return 0, nil
	}

	accounts, err := j.store.AccountsNeedingExpiryReminder(ctx, now, j.withinDays)
	if err != nil {
		return 0, err
	}

	for _, acc := range accounts {
		r := notify.ExpiryReminder{
			BusinessName: acc.BusinessName,
			Email:        acc.Email,
			ExpiresAt:    acc.ExpiresAt,
		}
		to := notify.Recipient{Email: acc.Email}
		if acc.TelegramChatID != nil {
			to.TelegramChatID = *acc.TelegramChatID
		}
		accountID := acc.ID
		sendErr := notify.Deliver(ctx, sender, j.store, to, r.Message(now),
			notify.LogMeta{Kind: store.NotificationKindExpiryReminder, AccountID: &accountID})

		// Telegram gagal sementara email sudah terkirim tetap dihitung
		// terkirim -- kalau tidak, pengingat yang emailnya sudah sampai
		// akan dikirim ulang terus tiap putaran.
		var telegramErr *notify.TelegramError
		if sendErr != nil && errors.As(sendErr, &telegramErr) {
			slog.Warn("pengingat kedaluwarsa: telegram gagal, email sudah terkirim",
				"account_id", acc.ID, "err", sendErr)
			sendErr = nil
		}
		if sendErr != nil {
			slog.Error("pengingat kedaluwarsa gagal dikirim", "account_id", acc.ID, "err", sendErr)
			continue
		}

		if err := j.store.MarkExpiryReminderSent(ctx, acc.ID, acc.ExpiresAt); err != nil {
			// Sudah terkirim tapi gagal ditandai: putaran berikutnya akan
			// mengirim ulang. Dicatat sebagai error supaya kalau ini sering
			// terjadi ketahuan, bukan diam-diam jadi spam ke customer.
			slog.Error("pengingat terkirim tapi gagal ditandai", "account_id", acc.ID, "err", err)
			continue
		}
		sent++
	}
	return sent, nil
}

func (j *Job) logEnabledTransition(enabled bool) {
	if j.lastEnabled != nil && *j.lastEnabled == enabled {
		return
	}
	j.lastEnabled = &enabled
	if enabled {
		slog.Info("pengingat kedaluwarsa aktif")
	} else {
		slog.Warn("pengingat kedaluwarsa TIDAK aktif -- SMTP belum dikonfigurasi di Vendor Dashboard (Settings)")
	}
}
