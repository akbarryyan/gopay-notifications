// Package devicealert mengabari customer saat HP bridge mereka berhenti
// mengirim kabar (OFFLINE), dan sekali lagi saat HP itu kembali online.
//
// Tanpa ini, HP yang dibunuh OEM atau kehilangan internet baru ketahuan saat
// customer sadar ada pembayaran yang tidak tercatat -- padahal backend sudah
// tahu sejak heartbeat melewati toleransi (store.HeartbeatTolerance).
package devicealert

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// Interval jarak antar putaran. Deteksi paling lambat HeartbeatTolerance +
// Interval setelah heartbeat terakhir -- 5 menit cukup rapat dibanding
// toleransi 45 menit, tanpa query tiap menit.
const Interval = 5 * time.Minute

// Store adalah bagian dari *store.Store yang dipakai pekerjaan ini.
type Store interface {
	DevicesNeedingOfflineAlert(ctx context.Context, now time.Time) ([]store.DeviceAlert, error)
	DevicesRecoveredFromOffline(ctx context.Context, now time.Time) ([]store.DeviceAlert, error)
	MarkDeviceOfflineAlerted(ctx context.Context, deviceID string, heartbeatAt time.Time) error
	ClearDeviceOfflineAlert(ctx context.Context, deviceID string) error
	notify.Logger
}

type Job struct {
	store     Store
	newSender notify.SenderFactory
}

func New(s Store, newSender notify.SenderFactory) *Job {
	return &Job{store: s, newSender: newSender}
}

// Result menghitung pemberitahuan yang terkirim dalam satu putaran.
type Result struct {
	Offline int
	Online  int
}

// Run memproses HP yang baru offline lalu HP yang sudah kembali online.
//
// Selama SMTP belum dikonfigurasi, putaran dilewati tanpa log -- pekerjaan
// pengingat kedaluwarsa sudah mencatat status aktif/tidak aktif yang sama.
//
// Kegagalan satu HP tidak menghentikan sisanya, dan HP yang gagal
// dikabarkan TIDAK ditandai, jadi putaran berikutnya mencobanya lagi.
func (j *Job) Run(ctx context.Context, now time.Time) (Result, error) {
	var res Result

	sender, enabled, err := j.newSender(ctx)
	if err != nil {
		return res, err
	}
	if !enabled {
		return res, nil
	}

	offline, err := j.store.DevicesNeedingOfflineAlert(ctx, now)
	if err != nil {
		return res, err
	}
	for _, d := range offline {
		msg := notify.DeviceOffline{
			BusinessName: d.BusinessName, DeviceName: d.DeviceName, LastHeartbeatAt: d.HeartbeatAt,
		}.Message()
		if !j.deliver(ctx, sender, d, msg, store.NotificationKindDeviceOffline) {
			continue
		}
		if err := j.store.MarkDeviceOfflineAlerted(ctx, d.DeviceID, d.HeartbeatAt); err != nil {
			slog.Error("peringatan offline terkirim tapi gagal ditandai", "device_id", d.DeviceID, "err", err)
			continue
		}
		res.Offline++
	}

	recovered, err := j.store.DevicesRecoveredFromOffline(ctx, now)
	if err != nil {
		return res, err
	}
	for _, d := range recovered {
		msg := notify.DeviceOnline{
			BusinessName: d.BusinessName, DeviceName: d.DeviceName,
			OfflineSince: d.OfflineSince, BackAt: d.HeartbeatAt,
		}.Message()
		if !j.deliver(ctx, sender, d, msg, store.NotificationKindDeviceOnline) {
			continue
		}
		if err := j.store.ClearDeviceOfflineAlert(ctx, d.DeviceID); err != nil {
			slog.Error("pemberitahuan online terkirim tapi gagal ditandai", "device_id", d.DeviceID, "err", err)
			continue
		}
		res.Online++
	}
	return res, nil
}

// deliver mengembalikan true bila pemberitahuan dianggap terkirim (email
// sampai; Telegram boleh gagal, alasannya sama dengan pengingat kedaluwarsa).
func (j *Job) deliver(ctx context.Context, sender notify.Sender, d store.DeviceAlert, msg notify.Message, kind string) bool {
	to := notify.Recipient{Email: d.Email}
	if d.TelegramChatID != nil {
		to.TelegramChatID = *d.TelegramChatID
	}
	accountID, deviceID, deviceName := d.AccountID, d.DeviceID, d.DeviceName
	err := notify.Deliver(ctx, sender, j.store, to, msg, notify.LogMeta{
		Kind: kind, AccountID: &accountID, DeviceID: &deviceID, DeviceName: &deviceName,
	})

	var telegramErr *notify.TelegramError
	if err != nil && errors.As(err, &telegramErr) {
		slog.Warn("status HP: telegram gagal, email sudah terkirim", "kind", kind, "device_id", d.DeviceID, "err", err)
		return true
	}
	if err != nil {
		slog.Error("status HP gagal dikabarkan", "kind", kind, "device_id", d.DeviceID, "err", err)
		return false
	}
	return true
}
