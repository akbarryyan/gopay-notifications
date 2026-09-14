package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func alertKey() []byte { return make([]byte, 32) }

// seedDeviceWithHeartbeat membuat device milik accountID dengan heartbeat_at
// diatur langsung -- RecordHeartbeat selalu memakai now() database.
func seedDeviceWithHeartbeat(t *testing.T, s *store.Store, accountID, deviceID string, heartbeatAt *time.Time) {
	t.Helper()
	ctx := context.Background()
	if err := s.CreateDevice(ctx, alertKey(), accountID, deviceID, "HP "+deviceID, []byte("rahasia")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if heartbeatAt != nil {
		setHeartbeat(t, s, deviceID, *heartbeatAt)
	}
}

func setHeartbeat(t *testing.T, s *store.Store, deviceID string, at time.Time) {
	t.Helper()
	if _, err := s.Pool().Exec(context.Background(),
		`UPDATE devices SET heartbeat_at = $2, last_seen_at = $2 WHERE device_id = $1`, deviceID, at); err != nil {
		t.Fatalf("set heartbeat: %v", err)
	}
}

func alertIDs(alerts []store.DeviceAlert) map[string]bool {
	out := map[string]bool{}
	for _, a := range alerts {
		out[a.DeviceID] = true
	}
	return out
}

func TestDevicesNeedingOfflineAlert(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	seedAccount(t, s, "acc_1")
	hourAgo := now.Add(-time.Hour)
	fresh := now.Add(-10 * time.Minute)
	stale := now.Add(-48 * time.Hour)
	seedDeviceWithHeartbeat(t, s, "acc_1", "dev_offline", &hourAgo)
	seedDeviceWithHeartbeat(t, s, "acc_1", "dev_online", &fresh)
	seedDeviceWithHeartbeat(t, s, "acc_1", "dev_basi", &stale)
	seedDeviceWithHeartbeat(t, s, "acc_1", "dev_pending", nil)
	seedDeviceWithHeartbeat(t, s, "acc_1", "dev_disabled", &hourAgo)
	if err := s.SetDeviceEnabled(ctx, "acc_1", "dev_disabled", false); err != nil {
		t.Fatalf("SetDeviceEnabled: %v", err)
	}

	seedAccount(t, s, "acc_suspend")
	seedDeviceWithHeartbeat(t, s, "acc_suspend", "dev_acc_suspend", &hourAgo)
	if err := s.SetAccountAdminStatus(ctx, "acc_suspend", "suspended"); err != nil {
		t.Fatalf("SetAccountAdminStatus: %v", err)
	}

	got, err := s.DevicesNeedingOfflineAlert(ctx, now)
	if err != nil {
		t.Fatalf("DevicesNeedingOfflineAlert: %v", err)
	}
	ids := alertIDs(got)
	if len(ids) != 1 || !ids["dev_offline"] {
		t.Fatalf("device = %v, mau cuma dev_offline (online, basi >24 jam, pending, disabled, account suspend dilewati)", ids)
	}
	if got[0].Email != "acc_1@uji.test" || got[0].BusinessName != "acc_1" || !got[0].HeartbeatAt.Equal(hourAgo) {
		t.Fatalf("alert = %+v, kontak/heartbeat tidak sesuai", got[0])
	}

	// Setelah ditandai tidak muncul lagi.
	if err := s.MarkDeviceOfflineAlerted(ctx, "dev_offline", got[0].HeartbeatAt); err != nil {
		t.Fatalf("MarkDeviceOfflineAlerted: %v", err)
	}
	got, _ = s.DevicesNeedingOfflineAlert(ctx, now)
	if len(got) != 0 {
		t.Fatalf("device = %v, mau kosong setelah ditandai", alertIDs(got))
	}
	// Belum kembali: tidak dianggap pulih.
	rec, _ := s.DevicesRecoveredFromOffline(ctx, now)
	if len(rec) != 0 {
		t.Fatalf("pulih = %v, mau kosong -- heartbeat belum bergerak", alertIDs(rec))
	}
}

func TestDevicesRecoveredFromOfflineLaluBisaOfflineLagi(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	seedAccount(t, s, "acc_1")
	hourAgo := now.Add(-time.Hour)
	seedDeviceWithHeartbeat(t, s, "acc_1", "dev_1", &hourAgo)
	if err := s.MarkDeviceOfflineAlerted(ctx, "dev_1", hourAgo); err != nil {
		t.Fatalf("MarkDeviceOfflineAlerted: %v", err)
	}

	// Heartbeat baru masuk.
	setHeartbeat(t, s, "dev_1", now.Add(-time.Minute))
	rec, err := s.DevicesRecoveredFromOffline(ctx, now)
	if err != nil {
		t.Fatalf("DevicesRecoveredFromOffline: %v", err)
	}
	if len(rec) != 1 || rec[0].DeviceID != "dev_1" || !rec[0].OfflineSince.Equal(hourAgo) {
		t.Fatalf("pulih = %+v, mau dev_1 dengan OfflineSince = heartbeat lama", rec)
	}

	if err := s.ClearDeviceOfflineAlert(ctx, "dev_1"); err != nil {
		t.Fatalf("ClearDeviceOfflineAlert: %v", err)
	}
	rec, _ = s.DevicesRecoveredFromOffline(ctx, now)
	if len(rec) != 0 {
		t.Fatalf("pulih = %v, mau kosong setelah dibersihkan", alertIDs(rec))
	}

	// Offline lagi satu jam kemudian -> peringatan baru terbuka.
	later := now.Add(time.Hour)
	got, _ := s.DevicesNeedingOfflineAlert(ctx, later)
	if len(got) != 1 {
		t.Fatalf("device = %v, mau dev_1 bisa diberi peringatan lagi", alertIDs(got))
	}
}

func TestNotificationLogCatatDanFilter(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	acc := "acc_1"
	errMsg := "535 authentication failed"
	entries := []store.NotificationLogEntry{
		{AccountID: &acc, Kind: store.NotificationKindExpiryReminder, Channel: "email",
			Recipient: "acc_1@uji.test", Subject: "Akun berakhir", Status: store.NotificationStatusSent},
		{AccountID: &acc, Kind: store.NotificationKindDeviceOffline, Channel: "email",
			Recipient: "acc_1@uji.test", Subject: "HP offline", Status: store.NotificationStatusFailed, Error: &errMsg},
		{Kind: store.NotificationKindTest, Channel: "telegram",
			Recipient: "12345", Subject: "Pesan uji", Status: store.NotificationStatusSent},
	}
	for _, e := range entries {
		if err := s.LogNotification(ctx, e); err != nil {
			t.Fatalf("LogNotification: %v", err)
		}
	}

	all, err := s.ListNotificationLog(ctx, 50, 0, store.NotificationLogFilter{})
	if err != nil {
		t.Fatalf("ListNotificationLog: %v", err)
	}
	if len(all) != 3 || all[0].Kind != store.NotificationKindTest || all[0].BusinessName != nil {
		t.Fatalf("semua = %+v, mau 3 baris terbaru dulu, pesan uji tanpa account", all)
	}
	if all[2].BusinessName == nil || *all[2].BusinessName != "acc_1" {
		t.Fatalf("business_name = %v, mau acc_1 dari join", all[2].BusinessName)
	}

	failed, _ := s.ListNotificationLog(ctx, 50, 0, store.NotificationLogFilter{Statuses: []string{"failed"}})
	if len(failed) != 1 || failed[0].Error == nil || *failed[0].Error != errMsg {
		t.Fatalf("gagal = %+v, mau satu baris dengan alasan", failed)
	}
	byKind, _ := s.ListNotificationLog(ctx, 50, 0, store.NotificationLogFilter{
		Kinds: []string{store.NotificationKindExpiryReminder, store.NotificationKindTest}})
	if len(byKind) != 2 {
		t.Fatalf("filter kind = %d baris, mau 2", len(byKind))
	}
	byChannel, _ := s.ListNotificationLog(ctx, 50, 0, store.NotificationLogFilter{Channels: []string{"telegram"}})
	if len(byChannel) != 1 {
		t.Fatalf("filter channel = %d baris, mau 1", len(byChannel))
	}
	byQuery, _ := s.ListNotificationLog(ctx, 50, 0, store.NotificationLogFilter{Query: "acc_1"})
	if len(byQuery) != 2 {
		t.Fatalf("filter q = %d baris, mau 2", len(byQuery))
	}
}
