package store_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestListExceptionsMengecualikanYangSudahCocok(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	inv, _, err := s.CreateInvoice(ctx, now, "acc_1", "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	eventWithAmount(t, s, "evt_cocok", inv.UniqueAmount)
	if _, err := s.MatchEvent(ctx, now, "acc_1", "evt_cocok", &inv.UniqueAmount); err != nil {
		t.Fatalf("MatchEvent: %v", err)
	}

	got, err := s.ListExceptions(ctx, 50, 0, store.ExceptionFilter{})
	if err != nil {
		t.Fatalf("ListExceptions: %v", err)
	}
	for _, e := range got {
		if e.EventID == "evt_cocok" {
			t.Fatal("evt_cocok sudah matched, seharusnya tidak muncul di ListExceptions")
		}
	}
}

func TestListExceptionsMengecualikanYangSudahDismiss(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	eventWithAmount(t, s, "evt_diabaikan", 12345)
	if err := s.DismissEvent(ctx, "evt_diabaikan", nil); err != nil {
		t.Fatalf("DismissEvent: %v", err)
	}

	got, err := s.ListExceptions(ctx, 50, 0, store.ExceptionFilter{})
	if err != nil {
		t.Fatalf("ListExceptions: %v", err)
	}
	for _, e := range got {
		if e.EventID == "evt_diabaikan" {
			t.Fatal("evt_diabaikan sudah dismiss, seharusnya tidak muncul di ListExceptions")
		}
	}
}

func TestListExceptionsMenampilkanYangBelumCocok(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	eventWithAmount(t, s, "evt_belum_cocok", 99999)

	got, err := s.ListExceptions(ctx, 50, 0, store.ExceptionFilter{})
	if err != nil {
		t.Fatalf("ListExceptions: %v", err)
	}
	found := false
	for _, e := range got {
		if e.EventID == "evt_belum_cocok" {
			found = true
		}
	}
	if !found {
		t.Fatal("evt_belum_cocok belum cocok invoice manapun, seharusnya muncul di ListExceptions")
	}
}

func TestListExceptionsMengecualikanAmountHintNil(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	e := sampleEvent("evt_tanpa_amount")
	e.AmountHint = nil
	if _, err := s.InsertEvent(ctx, e); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	got, err := s.ListExceptions(ctx, 50, 0, store.ExceptionFilter{})
	if err != nil {
		t.Fatalf("ListExceptions: %v", err)
	}
	for _, ex := range got {
		if ex.EventID == "evt_tanpa_amount" {
			t.Fatal("event tanpa amount_hint tidak boleh pernah muncul sebagai exception")
		}
	}
}

func TestManualMatchEventBerhasilKeInvoicePending(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	inv, _, err := s.CreateInvoice(ctx, now, "acc_1", "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	// Nominal event SENGAJA beda dari unique_amount invoice — ini justru
	// skenario khas konsol pengecualian: customer salah ketik nominal.
	eventWithAmount(t, s, "evt_manual", inv.UniqueAmount+7)

	if err := s.ManualMatchEvent(ctx, now, inv.ID, "evt_manual"); err != nil {
		t.Fatalf("ManualMatchEvent: %v", err)
	}

	got, err := s.GetInvoiceByID(ctx, "acc_1", inv.ID)
	if err != nil {
		t.Fatalf("GetInvoiceByID: %v", err)
	}
	if got.Status != store.InvoiceStatusPaid {
		t.Fatalf("Status = %s, mau PAID", got.Status)
	}
	if got.MatchedEventID == nil || *got.MatchedEventID != "evt_manual" {
		t.Fatalf("MatchedEventID = %v, mau evt_manual", got.MatchedEventID)
	}
}

func TestManualMatchEventBerhasilKeInvoiceExpired(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour)
	inv, _, err := s.CreateInvoice(ctx, past, "acc_1", "ORDER-telat", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	now := time.Now()
	if _, err := s.ExpireInvoicesAndListNewlyExpired(ctx, now); err != nil {
		t.Fatalf("ExpireInvoicesAndListNewlyExpired: %v", err)
	}
	eventWithAmount(t, s, "evt_telat", inv.UniqueAmount)

	if err := s.ManualMatchEvent(ctx, now, inv.ID, "evt_telat"); err != nil {
		t.Fatalf("ManualMatchEvent ke invoice EXPIRED: %v", err)
	}

	got, err := s.GetInvoiceByID(ctx, "acc_1", inv.ID)
	if err != nil {
		t.Fatalf("GetInvoiceByID: %v", err)
	}
	if got.Status != store.InvoiceStatusPaid {
		t.Fatalf("Status = %s, mau PAID (bayar telat tetap boleh dicocokkan manual)", got.Status)
	}
}

func TestManualMatchEventGagalKeInvoicePaid(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	inv, _, err := s.CreateInvoice(ctx, now, "acc_1", "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	eventWithAmount(t, s, "evt_pertama", inv.UniqueAmount)
	if _, err := s.MatchEvent(ctx, now, "acc_1", "evt_pertama", &inv.UniqueAmount); err != nil {
		t.Fatalf("MatchEvent: %v", err)
	}

	eventWithAmount(t, s, "evt_kedua", 11111)
	err = s.ManualMatchEvent(ctx, now, inv.ID, "evt_kedua")
	if err != store.ErrInvoiceNotEligibleForMatch {
		t.Fatalf("err = %v, mau ErrInvoiceNotEligibleForMatch (invoice sudah PAID)", err)
	}
}

func TestManualMatchEventGagalEventSudahDipakaiInvoiceLain(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	invA, _, err := s.CreateInvoice(ctx, now, "acc_1", "ORDER-A", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice A: %v", err)
	}
	invB, _, err := s.CreateInvoice(ctx, now, "acc_1", "ORDER-B", 70000)
	if err != nil {
		t.Fatalf("CreateInvoice B: %v", err)
	}
	eventWithAmount(t, s, "evt_rebutan", 12345)

	if err := s.ManualMatchEvent(ctx, now, invA.ID, "evt_rebutan"); err != nil {
		t.Fatalf("ManualMatchEvent pertama: %v", err)
	}

	err = s.ManualMatchEvent(ctx, now, invB.ID, "evt_rebutan")
	if err != store.ErrEventAlreadyMatched {
		t.Fatalf("err = %v, mau ErrEventAlreadyMatched", err)
	}
}

// TestManualMatchEventRaceHanyaSatuYangMenang: dua admin mencocokkan event
// yang sama ke dua invoice berbeda nyaris bersamaan — constraint
// invoices_matched_event_id_idx yang menjamin hanya satu bisa menang, bukan
// urutan pemanggilan.
func TestManualMatchEventRaceHanyaSatuYangMenang(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	const invoiceCount = 8
	invoiceIDs := make([]string, invoiceCount)
	for i := 0; i < invoiceCount; i++ {
		inv, _, err := s.CreateInvoice(ctx, now, "acc_1", "ORDER-race-"+string(rune('a'+i)), int64(10000+i*1000))
		if err != nil {
			t.Fatalf("CreateInvoice #%d: %v", i, err)
		}
		invoiceIDs[i] = inv.ID
	}
	eventWithAmount(t, s, "evt_direbut", 55555)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		matched int
	)
	start := make(chan struct{})

	for i := 0; i < invoiceCount; i++ {
		wg.Add(1)
		go func(invoiceID string) {
			defer wg.Done()
			<-start
			if err := s.ManualMatchEvent(ctx, now, invoiceID, "evt_direbut"); err == nil {
				mu.Lock()
				matched++
				mu.Unlock()
			}
		}(invoiceIDs[i])
	}
	close(start)
	wg.Wait()

	if matched != 1 {
		t.Fatalf("matched = %d, mau tepat 1", matched)
	}
}

func TestDismissEventBerhasil(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	eventWithAmount(t, s, "evt_abaikan", 22222)

	note := "transfer pribadi, bukan order"
	if err := s.DismissEvent(ctx, "evt_abaikan", &note); err != nil {
		t.Fatalf("DismissEvent: %v", err)
	}
}

func TestDismissEventDuaKaliDitolak(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	eventWithAmount(t, s, "evt_abaikan_2x", 33333)

	if err := s.DismissEvent(ctx, "evt_abaikan_2x", nil); err != nil {
		t.Fatalf("DismissEvent pertama: %v", err)
	}
	err := s.DismissEvent(ctx, "evt_abaikan_2x", nil)
	if err != store.ErrEventAlreadyDismissed {
		t.Fatalf("err = %v, mau ErrEventAlreadyDismissed", err)
	}
}

func TestDismissEventTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.DismissEvent(context.Background(), "evt_tidak_pernah_ada", nil)
	if err != store.ErrEventNotFound {
		t.Fatalf("err = %v, mau ErrEventNotFound", err)
	}
}
