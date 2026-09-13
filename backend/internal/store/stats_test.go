package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// payInvoice membuat invoice lalu langsung melunasinya pada waktu `paidAt`
// tertentu -- helper dipakai test yang butuh kontrol presisi atas paid_at,
// beda dari alur normal (MatchEvent dipicu event sungguhan). Event
// pencocokannya tetap disisipkan sungguhan (bukan id karangan) karena
// matched_event_id punya foreign key ke notification_events, yang
// device_id-nya sendiri punya foreign key ke devices -- deviceID yang
// dioper harus device yang benar-benar sudah dibuat lebih dulu.
func payInvoice(t *testing.T, s *store.Store, accountID, deviceID, externalRef string, amount int64, paidAt time.Time) {
	t.Helper()
	ctx := context.Background()
	inv, _, err := s.CreateInvoice(ctx, paidAt, accountID, externalRef, amount)
	if err != nil {
		t.Fatalf("CreateInvoice %s: %v", externalRef, err)
	}

	eventID := "evt_" + externalRef
	e := sampleEvent(eventID)
	e.AccountID = accountID
	e.DeviceID = deviceID
	e.AmountHint = &inv.UniqueAmount
	e.PostedAt, e.ReceivedAt = paidAt, paidAt
	if _, err := s.InsertEvent(ctx, e); err != nil {
		t.Fatalf("InsertEvent %s: %v", eventID, err)
	}

	matched, err := s.MatchEvent(ctx, paidAt, accountID, eventID, &inv.UniqueAmount)
	if err != nil {
		t.Fatalf("MatchEvent %s: %v", externalRef, err)
	}
	if matched != inv.ID {
		t.Fatalf("MatchEvent %s: matched = %q, mau %q", externalRef, matched, inv.ID)
	}
}

func TestDailyEventCountsMenjumlahkanNominalLunasPerHariSesuaiPaidAt(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_1")
	seedAccount(t, s, "acc_2")
	if err := s.CreateDevice(context.Background(), encKey(), "acc_1", "dev_1", "HP 1", []byte("secret")); err != nil {
		t.Fatalf("CreateDevice acc_1: %v", err)
	}
	if err := s.CreateDevice(context.Background(), encKey(), "acc_2", "dev_2", "HP 2", []byte("secret")); err != nil {
		t.Fatalf("CreateDevice acc_2: %v", err)
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	payInvoice(t, s, "acc_1", "dev_1", "ORDER-TODAY", 50_000, today)
	payInvoice(t, s, "acc_1", "dev_1", "ORDER-YDAY", 30_000, yesterday)
	// Akun lain, hari yang sama -- tidak boleh ikut terjumlah ke acc_1.
	payInvoice(t, s, "acc_2", "dev_2", "ORDER-OTHER", 999_000, today)

	daily, err := s.DailyEventCounts(context.Background(), "acc_1", now, 2)
	if err != nil {
		t.Fatalf("DailyEventCounts: %v", err)
	}
	if len(daily) != 2 {
		t.Fatalf("len(daily) = %d, mau 2", len(daily))
	}

	gotYesterday, gotToday := daily[0], daily[1]
	if gotYesterday.PaidAmountRp != 30_000 {
		t.Errorf("PaidAmountRp kemarin = %d, mau 30000", gotYesterday.PaidAmountRp)
	}
	if gotToday.PaidAmountRp != 50_000 {
		t.Errorf("PaidAmountRp hari ini = %d, mau 50000 (tanpa nominal acc_2)", gotToday.PaidAmountRp)
	}
}

func TestDailyEventCountsNolKalauBelumAdaPembayaran(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_1")

	daily, err := s.DailyEventCounts(context.Background(), "acc_1", time.Now(), 3)
	if err != nil {
		t.Fatalf("DailyEventCounts: %v", err)
	}
	for _, d := range daily {
		if d.PaidAmountRp != 0 {
			t.Errorf("PaidAmountRp di %s = %d, mau 0", d.Date, d.PaidAmountRp)
		}
		if d.Count != 0 {
			t.Errorf("Count di %s = %d, mau 0", d.Date, d.Count)
		}
	}
}
