package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestStatusOfMurniTanpaDatabase(t *testing.T) {
	now := time.Unix(1789200000, 0)
	baru := now.Add(-1 * time.Minute)
	lama := now.Add(-store.HeartbeatTolerance - time.Minute)
	batas := now.Add(-store.HeartbeatTolerance)

	cases := []struct {
		nama string
		dev  store.Device
		mau  store.DeviceStatus
	}{
		{"dinonaktifkan menang atas segalanya",
			store.Device{Enabled: false, HeartbeatAt: &baru}, store.DeviceDisabled},
		{"belum pernah heartbeat",
			store.Device{Enabled: true}, store.DevicePending},
		{"heartbeat baru",
			store.Device{Enabled: true, HeartbeatAt: &baru}, store.DeviceOnline},
		{"tepat di batas toleransi masih online",
			store.Device{Enabled: true, HeartbeatAt: &batas}, store.DeviceOnline},
		{"lewat batas toleransi",
			store.Device{Enabled: true, HeartbeatAt: &lama}, store.DeviceOffline},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			if got := store.StatusOf(tc.dev, now); got != tc.mau {
				t.Fatalf("StatusOf = %s, mau %s", got, tc.mau)
			}
		})
	}
}

func TestToleransiLebihLuasDariInterval(t *testing.T) {
	// Android menunda periodic work saat Doze. Toleransi sebesar satu
	// interval akan menghasilkan alarm palsu terus-menerus, dan status yang
	// sering salah adalah status yang diabaikan orang.
	if store.HeartbeatTolerance <= store.HeartbeatInterval {
		t.Fatalf("toleransi %v harus lebih besar dari interval %v",
			store.HeartbeatTolerance, store.HeartbeatInterval)
	}
}

func TestRecordHeartbeatMenyimpanKondisi(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	seedAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	sebelum, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if store.StatusOf(sebelum, time.Now()) != store.DevicePending {
		t.Fatal("device baru seharusnya PENDING sebelum heartbeat pertama")
	}

	err = s.RecordHeartbeat(ctx, "dev_01ABC", store.Heartbeat{
		AndroidVersion:    "13",
		ListenerConnected: true,
		PendingCount:      2,
		FailedCount:       1,
	})
	if err != nil {
		t.Fatalf("RecordHeartbeat: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.HeartbeatAt == nil {
		t.Fatal("HeartbeatAt masih nil")
	}
	if got.AndroidVersion == nil || *got.AndroidVersion != "13" {
		t.Fatalf("AndroidVersion = %v", got.AndroidVersion)
	}
	if got.ListenerConnected == nil || !*got.ListenerConnected {
		t.Fatalf("ListenerConnected = %v", got.ListenerConnected)
	}
	if got.PendingCount == nil || *got.PendingCount != 2 {
		t.Fatalf("PendingCount = %v", got.PendingCount)
	}
	if got.FailedCount == nil || *got.FailedCount != 1 {
		t.Fatalf("FailedCount = %v", got.FailedCount)
	}
	if store.StatusOf(got, time.Now()) != store.DeviceOnline {
		t.Fatal("setelah heartbeat seharusnya ONLINE")
	}
}

func TestRecordHeartbeatJugaMemperbaruiLastSeen(t *testing.T) {
	// Heartbeat adalah komunikasi. Kalau last_seen_at tidak ikut bergerak,
	// dashboard akan melaporkan dua kebenaran yang saling bertentangan.
	s := testStore(t)
	ctx := context.Background()

	seedAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.RecordHeartbeat(ctx, "dev_01ABC", store.Heartbeat{ListenerConnected: true}); err != nil {
		t.Fatalf("RecordHeartbeat: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.LastSeenAt == nil {
		t.Fatal("LastSeenAt masih nil setelah heartbeat")
	}
}

func TestHeartbeatVersiAndroidKosongTidakMenimpa(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	seedAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.RecordHeartbeat(ctx, "dev_01ABC", store.Heartbeat{AndroidVersion: "13"}); err != nil {
		t.Fatalf("heartbeat pertama: %v", err)
	}
	if err := s.RecordHeartbeat(ctx, "dev_01ABC", store.Heartbeat{AndroidVersion: ""}); err != nil {
		t.Fatalf("heartbeat kedua: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.AndroidVersion == nil || *got.AndroidVersion != "13" {
		t.Fatalf("AndroidVersion = %v, mau tetap 13", got.AndroidVersion)
	}
}
