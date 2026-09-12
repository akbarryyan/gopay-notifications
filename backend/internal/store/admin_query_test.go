package store_test

import (
	"context"
	"testing"
	"time"
)

func TestListDevicesTidakMengembalikanSecret(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, encKey(), "dev_a", "HP A", []byte("secret-a")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	got, err := s.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, mau 1", len(got))
	}
	if got[0].Secret != nil {
		t.Fatal("ListDevices tidak boleh membocorkan secret, terenkripsi maupun tidak")
	}
}

func TestListDevicesTerbaruLebihDulu(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, encKey(), "dev_lama", "Lama", []byte("s")); err != nil {
		t.Fatalf("CreateDevice lama: %v", err)
	}
	if err := s.CreateDevice(ctx, encKey(), "dev_baru", "Baru", []byte("s")); err != nil {
		t.Fatalf("CreateDevice baru: %v", err)
	}

	got, err := s.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, mau 2", len(got))
	}
	if got[0].DeviceID != "dev_baru" {
		t.Fatalf("device pertama = %s, mau dev_baru (terbaru dulu)", got[0].DeviceID)
	}
}

func TestSetDeviceEnabled(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, encKey(), "dev_a", "HP A", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	if err := s.SetDeviceEnabled(ctx, "dev_a", false); err != nil {
		t.Fatalf("SetDeviceEnabled: %v", err)
	}
	got, err := s.GetDevice(ctx, encKey(), "dev_a")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.Enabled {
		t.Fatal("device seharusnya dinonaktifkan")
	}
}

func TestSetDeviceEnabledDeviceTakDikenal(t *testing.T) {
	s := testStore(t)

	err := s.SetDeviceEnabled(context.Background(), "dev_tidak_ada", false)
	if err == nil {
		t.Fatal("mau error untuk device yang tidak ada, dapat nil")
	}
}

func TestEventStatsMenghitungHariIniDanTujuhHari(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	now := time.Now()

	// Satu event hari ini.
	if _, err := s.InsertEvent(ctx, sampleEvent("evt_hari_ini")); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	stats, err := s.EventStats(ctx, now)
	if err != nil {
		t.Fatalf("EventStats: %v", err)
	}
	if stats.Today != 1 {
		t.Fatalf("Today = %d, mau 1", stats.Today)
	}
	if stats.Last7Days != 1 {
		t.Fatalf("Last7Days = %d, mau 1", stats.Last7Days)
	}
	if stats.Total != 1 {
		t.Fatalf("Total = %d, mau 1", stats.Total)
	}
	if stats.LatestEventAt == nil {
		t.Fatal("LatestEventAt masih nil")
	}
}

func TestEventStatsKosongTidakError(t *testing.T) {
	s := testStore(t)

	stats, err := s.EventStats(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("EventStats: %v", err)
	}
	if stats.Today != 0 || stats.Last7Days != 0 || stats.Total != 0 {
		t.Fatalf("stats = %+v, mau semua nol", stats)
	}
	if stats.LatestEventAt != nil {
		t.Fatal("LatestEventAt seharusnya nil bila belum ada event")
	}
}

func TestDailyEventCountsMengisiNolUntukHariSepi(t *testing.T) {
	s := testStore(t)

	got, err := s.DailyEventCounts(context.Background(), time.Now(), 14)
	if err != nil {
		t.Fatalf("DailyEventCounts: %v", err)
	}
	if len(got) != 14 {
		t.Fatalf("len = %d, mau 14 (harus tetap terisi walau tidak ada event)", len(got))
	}
	for _, d := range got {
		if d.Count != 0 {
			t.Fatalf("Count = %d di %s, mau 0 tanpa event", d.Count, d.Date)
		}
	}
	if !got[len(got)-1].Date.Equal(got[0].Date.AddDate(0, 0, 13)) {
		t.Fatalf("titik terakhir = %s, mau 13 hari setelah titik pertama (%s)", got[len(got)-1].Date, got[0].Date)
	}
}

func TestDailyEventCountsMenghitungHariIni(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	if _, err := s.InsertEvent(ctx, sampleEvent("evt_a")); err != nil {
		t.Fatalf("InsertEvent a: %v", err)
	}
	if _, err := s.InsertEvent(ctx, sampleEvent("evt_b")); err != nil {
		t.Fatalf("InsertEvent b: %v", err)
	}

	got, err := s.DailyEventCounts(ctx, time.Now(), 14)
	if err != nil {
		t.Fatalf("DailyEventCounts: %v", err)
	}

	today := got[len(got)-1]
	if today.Count != 2 {
		t.Fatalf("Count hari ini = %d, mau 2", today.Count)
	}
	for _, d := range got[:len(got)-1] {
		if d.Count != 0 {
			t.Fatalf("Count di %s = %d, mau 0 (event hanya diinsert hari ini)", d.Date, d.Count)
		}
	}
}
