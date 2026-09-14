package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestLogActivityDanListActivityLog(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	seedAccount(t, s, "acc_2")

	ip := "203.0.113.7"
	if err := s.LogActivity(ctx, store.ActivityLogEntry{
		AccountID: "acc_1", Action: store.ActivityLoginSuccess, IPAddress: &ip,
	}); err != nil {
		t.Fatalf("LogActivity login: %v", err)
	}
	if err := s.LogActivity(ctx, store.ActivityLogEntry{
		AccountID: "acc_1", Action: store.ActivityDeviceAdded,
		Metadata: map[string]any{"device_id": "dev_01", "name": "HP Kasir"},
	}); err != nil {
		t.Fatalf("LogActivity device: %v", err)
	}
	// Milik account lain -- tidak boleh ikut muncul di daftar acc_1.
	if err := s.LogActivity(ctx, store.ActivityLogEntry{AccountID: "acc_2", Action: store.ActivityLoginSuccess}); err != nil {
		t.Fatalf("LogActivity acc_2: %v", err)
	}

	rows, err := s.ListActivityLog(ctx, "acc_1", 50, 0, store.ActivityLogFilter{})
	if err != nil {
		t.Fatalf("ListActivityLog: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, mau 2 (device_added dan login_success), acc_2 tidak boleh ikut", len(rows))
	}
	if rows[0].Action != store.ActivityDeviceAdded {
		t.Fatalf("rows[0].Action = %q, mau device_added (terbaru dulu)", rows[0].Action)
	}
	if rows[0].Metadata == nil {
		t.Fatal("metadata device_added kosong")
	}
	if rows[1].IPAddress == nil || *rows[1].IPAddress != ip {
		t.Fatalf("IPAddress = %v, mau %q", rows[1].IPAddress, ip)
	}

	filtered, err := s.ListActivityLog(ctx, "acc_1", 50, 0, store.ActivityLogFilter{
		Actions: []string{store.ActivityLoginSuccess},
	})
	if err != nil {
		t.Fatalf("ListActivityLog filter: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Action != store.ActivityLoginSuccess {
		t.Fatalf("filtered = %+v, mau cuma login_success", filtered)
	}
}

func TestListActivityLogFilterTanggal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	if err := s.LogActivity(ctx, store.ActivityLogEntry{AccountID: "acc_1", Action: store.ActivityLoginSuccess}); err != nil {
		t.Fatalf("LogActivity: %v", err)
	}

	future := time.Now().Add(24 * time.Hour)
	rows, err := s.ListActivityLog(ctx, "acc_1", 50, 0, store.ActivityLogFilter{From: &future})
	if err != nil {
		t.Fatalf("ListActivityLog: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %d, mau 0 (from di masa depan)", len(rows))
	}
}
