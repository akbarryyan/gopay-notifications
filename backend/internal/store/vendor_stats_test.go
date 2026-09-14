package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// seedAccountWithExpiry sama seperti seedAccount, tapi expires_at bisa
// diatur -- dipakai test yang butuh account dengan status tertentu
// (active/expiring/expired) lewat DerivedStatus.
func seedAccountWithExpiry(t *testing.T, s *store.Store, id string, expiresAt time.Time) {
	t.Helper()
	err := s.CreateAccount(context.Background(), store.CreateAccountInput{
		ID: id, BusinessName: id, Email: id + "@uji.test", Username: id,
		PlaintextPassword: "rahasia123", Plan: "Business", MaxDevices: 10,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("seed account %s: %v", id, err)
	}
}

func TestVendorOverviewStatsMenghitungRingkasanLintasAccount(t *testing.T) {
	s := testStore(t)
	now := time.Now()

	seedAccountWithExpiry(t, s, "acc_active", now.Add(200*24*time.Hour))  // > 30 hari -> active
	seedAccountWithExpiry(t, s, "acc_expiring", now.Add(10*24*time.Hour)) // <= 30 hari -> expiring
	seedAccountWithExpiry(t, s, "acc_expired", now.Add(-24*time.Hour))    // sudah lewat -> expired
	if err := s.SetAccountAdminStatus(context.Background(), "acc_expired", "suspended"); err != nil {
		t.Fatalf("SetAccountAdminStatus: %v", err)
	}
	// acc_expired sekarang admin_status=suspended -- DerivedStatus harus
	// melaporkan "suspended" (admin_status menang atas kedaluwarsa tanggal),
	// jadi dipakai juga sebagai kasus uji "suspended", bukan "expired".

	if err := s.CreateDevice(context.Background(), encKey(), "acc_active", "dev_x", "HP", []byte("secret")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	stats, err := s.VendorOverviewStats(context.Background(), now)
	if err != nil {
		t.Fatalf("VendorOverviewStats: %v", err)
	}

	if stats.TotalAccounts != 3 {
		t.Errorf("TotalAccounts = %d, mau 3", stats.TotalAccounts)
	}
	if stats.Active != 1 {
		t.Errorf("Active = %d, mau 1", stats.Active)
	}
	if stats.Expiring != 1 {
		t.Errorf("Expiring = %d, mau 1", stats.Expiring)
	}
	if stats.Suspended != 1 {
		t.Errorf("Suspended = %d, mau 1", stats.Suspended)
	}
	if stats.NewThisWeek != 3 {
		t.Errorf("NewThisWeek = %d, mau 3 (semua baru dibuat)", stats.NewThisWeek)
	}
	if stats.TotalDevices != 1 {
		t.Errorf("TotalDevices = %d, mau 1", stats.TotalDevices)
	}
}

func TestVendorOverviewStatsMenjumlahkanNominalLunasLintasAccount(t *testing.T) {
	s := testStore(t)
	now := time.Now()
	seedAccountWithExpiry(t, s, "acc_1", now.Add(365*24*time.Hour))
	seedAccountWithExpiry(t, s, "acc_2", now.Add(365*24*time.Hour))
	if err := s.CreateDevice(context.Background(), encKey(), "acc_1", "dev_1", "HP 1", []byte("secret")); err != nil {
		t.Fatalf("CreateDevice acc_1: %v", err)
	}
	if err := s.CreateDevice(context.Background(), encKey(), "acc_2", "dev_2", "HP 2", []byte("secret")); err != nil {
		t.Fatalf("CreateDevice acc_2: %v", err)
	}

	payInvoice(t, s, "acc_1", "dev_1", "ORDER-1", 50_000, now)
	payInvoice(t, s, "acc_2", "dev_2", "ORDER-2", 70_000, now)

	stats, err := s.VendorOverviewStats(context.Background(), now)
	if err != nil {
		t.Fatalf("VendorOverviewStats: %v", err)
	}
	if stats.TotalPaidAmountRp != 120_000 {
		t.Errorf("TotalPaidAmountRp = %d, mau 120000 (jumlah acc_1 + acc_2)", stats.TotalPaidAmountRp)
	}
}

func TestVendorDailyStatsLintasAccountTanpaFilter(t *testing.T) {
	s := testStore(t)
	now := time.Now()
	seedAccountWithExpiry(t, s, "acc_1", now.Add(365*24*time.Hour))
	seedAccountWithExpiry(t, s, "acc_2", now.Add(365*24*time.Hour))
	if err := s.CreateDevice(context.Background(), encKey(), "acc_1", "dev_1", "HP 1", []byte("secret")); err != nil {
		t.Fatalf("CreateDevice acc_1: %v", err)
	}
	if err := s.CreateDevice(context.Background(), encKey(), "acc_2", "dev_2", "HP 2", []byte("secret")); err != nil {
		t.Fatalf("CreateDevice acc_2: %v", err)
	}

	payInvoice(t, s, "acc_1", "dev_1", "ORDER-1", 40_000, now)
	payInvoice(t, s, "acc_2", "dev_2", "ORDER-2", 60_000, now)

	daily, err := s.VendorDailyStats(context.Background(), now, 3)
	if err != nil {
		t.Fatalf("VendorDailyStats: %v", err)
	}
	if len(daily) != 3 {
		t.Fatalf("len(daily) = %d, mau 3", len(daily))
	}

	today := daily[len(daily)-1]
	if today.NewAccounts != 2 {
		t.Errorf("NewAccounts hari ini = %d, mau 2 (acc_1 + acc_2)", today.NewAccounts)
	}
	if today.PaidAmountRp != 100_000 {
		t.Errorf("PaidAmountRp hari ini = %d, mau 100000 (40000 + 60000)", today.PaidAmountRp)
	}
}
