package store_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func eventWithAmount(t *testing.T, s *store.Store, eventID string, amount int64) {
	t.Helper()
	e := sampleEvent(eventID)
	e.AmountHint = &amount
	if _, err := s.InsertEvent(context.Background(), e); err != nil {
		t.Fatalf("InsertEvent %s: %v", eventID, err)
	}
}

func TestCreateInvoiceMengalokasikanNominalUnik(t *testing.T) {
	s := testStore(t)
	now := time.Now()

	inv, created, err := s.CreateInvoice(context.Background(), now, "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	if !created {
		t.Fatal("created = false, mau true untuk invoice baru")
	}
	if inv.UniqueAmount <= 50000 || inv.UniqueAmount > 50999 {
		t.Fatalf("UniqueAmount = %d, mau di rentang 50001..50999", inv.UniqueAmount)
	}
	if inv.Status != store.InvoiceStatusPending {
		t.Fatalf("Status = %s, mau PENDING", inv.Status)
	}
	if inv.ExpiresAt.Sub(inv.CreatedAt) != 15*time.Minute {
		t.Fatalf("masa berlaku = %s, mau 15 menit", inv.ExpiresAt.Sub(inv.CreatedAt))
	}
}

func TestCreateInvoiceExternalRefSamaAmountSamaIdempotent(t *testing.T) {
	s := testStore(t)
	now := time.Now()
	ctx := context.Background()

	first, created1, err := s.CreateInvoice(ctx, now, "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice pertama: %v", err)
	}
	if !created1 {
		t.Fatal("created pertama = false, mau true")
	}

	second, created2, err := s.CreateInvoice(ctx, now, "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice kedua: %v", err)
	}
	if created2 {
		t.Fatal("created kedua = true, mau false (invoice lama dikembalikan, bukan invoice baru)")
	}
	if second.ID != first.ID || second.UniqueAmount != first.UniqueAmount {
		t.Fatalf("invoice kedua = %+v, mau identik dengan yang pertama %+v", second, first)
	}
}

func TestCreateInvoiceExternalRefSamaAmountBedaDitolak(t *testing.T) {
	s := testStore(t)
	now := time.Now()
	ctx := context.Background()

	if _, _, err := s.CreateInvoice(ctx, now, "ORDER-1", 50000); err != nil {
		t.Fatalf("CreateInvoice pertama: %v", err)
	}

	_, _, err := s.CreateInvoice(ctx, now, "ORDER-1", 75000)
	if err != store.ErrInvoiceRefConflict {
		t.Fatalf("err = %v, mau ErrInvoiceRefConflict", err)
	}
}

func TestCreateInvoiceMenghindariTabrakanNominal(t *testing.T) {
	s := testStore(t)
	now := time.Now()
	ctx := context.Background()

	seen := map[int64]bool{}
	for i := range 5 {
		inv, _, err := s.CreateInvoice(ctx, now, "ORDER-"+string(rune('A'+i)), 50000)
		if err != nil {
			t.Fatalf("CreateInvoice #%d: %v", i, err)
		}
		if seen[inv.UniqueAmount] {
			t.Fatalf("UniqueAmount %d dipakai dua invoice PENDING sekaligus", inv.UniqueAmount)
		}
		seen[inv.UniqueAmount] = true
	}
}

// TestCreateInvoiceMenulisTransisiExpiredSebelumAlokasi menguji mekanisme
// lazy-expire secara langsung: sebuah baris PENDING yang expires_at-nya
// sudah lewat disisipkan langsung lewat SQL mentah (supaya unique_amount-nya
// diketahui pasti, tidak bergantung pada offset acak CreateInvoice), lalu
// CreateInvoice apa pun dipanggil sekali. Baris lama itu harus sudah
// berstatus EXPIRED sesudahnya — bukan status yang dihitung ulang saat
// dibaca, tapi transisi yang benar-benar dituliskan (lihat store.Invoice
// di invoice.go untuk alasan predicate index-nya).
func TestCreateInvoiceMenulisTransisiExpiredSebelumAlokasi(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	_, err := s.Pool().Exec(ctx,
		`INSERT INTO invoices (id, external_ref, requested_amount, unique_amount, status, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		"inv_stale", "ORDER-lama", int64(50000), int64(50123), store.InvoiceStatusPending,
		past, past.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("seed invoice stale: %v", err)
	}

	if _, _, err := s.CreateInvoice(ctx, now, "ORDER-baru", 20000); err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	got, err := s.GetInvoiceByID(ctx, "inv_stale")
	if err != nil {
		t.Fatalf("GetInvoiceByID: %v", err)
	}
	if got.Status != store.InvoiceStatusExpired {
		t.Fatalf("Status invoice lama = %s, mau EXPIRED setelah CreateInvoice lain dipanggil", got.Status)
	}
}

func TestMatchEventMenandaiInvoicePaid(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	inv, _, err := s.CreateInvoice(ctx, now, "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	eventWithAmount(t, s, "evt_match_1", inv.UniqueAmount)

	matched, err := s.MatchEvent(ctx, now, "evt_match_1", &inv.UniqueAmount)
	if err != nil {
		t.Fatalf("MatchEvent: %v", err)
	}
	if !matched {
		t.Fatal("matched = false, mau true")
	}

	got, err := s.GetInvoiceByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInvoiceByID: %v", err)
	}
	if got.Status != store.InvoiceStatusPaid {
		t.Fatalf("Status = %s, mau PAID", got.Status)
	}
	if got.MatchedEventID == nil || *got.MatchedEventID != "evt_match_1" {
		t.Fatalf("MatchedEventID = %v, mau evt_match_1", got.MatchedEventID)
	}
	if got.PaidAt == nil {
		t.Fatal("PaidAt masih nil setelah matched")
	}
}

func TestMatchEventTidakCocokTidakMengubahApaPun(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()

	inv, _, err := s.CreateInvoice(ctx, now, "ORDER-1", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	wrongAmount := inv.UniqueAmount + 1
	matched, err := s.MatchEvent(ctx, now, "evt_tidak_cocok", &wrongAmount)
	if err != nil {
		t.Fatalf("MatchEvent: %v", err)
	}
	if matched {
		t.Fatal("matched = true, mau false — nominal tidak cocok invoice manapun")
	}

	got, err := s.GetInvoiceByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInvoiceByID: %v", err)
	}
	if got.Status != store.InvoiceStatusPending {
		t.Fatalf("Status = %s, mau tetap PENDING", got.Status)
	}
}

func TestMatchEventAmountNilDilewati(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	matched, err := s.MatchEvent(ctx, time.Now(), "evt_tanpa_amount", nil)
	if err != nil {
		t.Fatalf("MatchEvent: %v", err)
	}
	if matched {
		t.Fatal("matched = true, mau false untuk amount nil")
	}
}

// TestMatchEventRaceHanyaSatuYangMenang: dua event dengan amount yang sama
// datang nyaris bersamaan, hanya satu yang boleh mencocokkan invoice —
// pola yang sama dengan TestInsertEventConcurrentSameIDInsertsOnce, memakai
// UPDATE...RETURNING atomik, bukan SELECT lalu UPDATE terpisah.
func TestMatchEventRaceHanyaSatuYangMenang(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	inv, _, err := s.CreateInvoice(ctx, now, "ORDER-race", 50000)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	const goroutines = 8
	// matched_event_id punya foreign key ke notification_events — setiap
	// event kandidat harus benar-benar ada dulu sebelum dicocokkan,
	// persis seperti alur sungguhan (matching selalu berjalan SETELAH
	// InsertEvent sukses, tidak pernah terhadap event_id yang belum ada).
	eventIDs := make([]string, goroutines)
	for i := range goroutines {
		eventIDs[i] = "evt_race_" + string(rune('a'+i))
		eventWithAmount(t, s, eventIDs[i], inv.UniqueAmount)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		matched int
	)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			ok, err := s.MatchEvent(ctx, now, eventIDs[i], &inv.UniqueAmount)
			if err != nil {
				t.Errorf("MatchEvent: %v", err)
				return
			}
			if ok {
				mu.Lock()
				matched++
				mu.Unlock()
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if matched != 1 {
		t.Fatalf("matched = %d, mau tepat 1", matched)
	}
}

func TestListInvoicesFilterStatusDanQuery(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()
	now := time.Now()

	inv1, _, err := s.CreateInvoice(ctx, now, "ORDER-cari-1", 10000)
	if err != nil {
		t.Fatalf("CreateInvoice 1: %v", err)
	}
	if _, _, err := s.CreateInvoice(ctx, now, "ORDER-lain-2", 20000); err != nil {
		t.Fatalf("CreateInvoice 2: %v", err)
	}
	eventWithAmount(t, s, "evt_list_1", inv1.UniqueAmount)
	if _, err := s.MatchEvent(ctx, now, "evt_list_1", &inv1.UniqueAmount); err != nil {
		t.Fatalf("MatchEvent: %v", err)
	}

	paid, err := s.ListInvoices(ctx, 50, 0, store.InvoiceFilter{Status: store.InvoiceStatusPaid})
	if err != nil {
		t.Fatalf("ListInvoices status=PAID: %v", err)
	}
	if len(paid) != 1 || paid[0].ExternalRef != "ORDER-cari-1" {
		t.Fatalf("ListInvoices status=PAID = %+v, mau 1 baris ORDER-cari-1", paid)
	}

	byQuery, err := s.ListInvoices(ctx, 50, 0, store.InvoiceFilter{Query: "cari"})
	if err != nil {
		t.Fatalf("ListInvoices q=cari: %v", err)
	}
	if len(byQuery) != 1 || byQuery[0].ExternalRef != "ORDER-cari-1" {
		t.Fatalf("ListInvoices q=cari = %+v, mau 1 baris ORDER-cari-1", byQuery)
	}
}
