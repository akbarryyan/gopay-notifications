package store_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func seedDevice(t *testing.T, s *store.Store) {
	t.Helper()
	err := s.CreateDevice(context.Background(), encKey(), "dev_01ABC", "HP", []byte("secret"))
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
}

func sampleEvent(eventID string) store.Event {
	title := "Transfer masuk"
	body := "Rp1 dari icaangg udah masuk ke GoPay kamu."
	amount := int64(1)
	return store.Event{
		EventID:     eventID,
		DeviceID:    "dev_01ABC",
		Source:      "gopay",
		PackageName: "com.gojek.gopay",
		Title:       &title,
		BodyText:    &body,
		BigText:     &body,
		AmountHint:  &amount,
		PostedAt:    time.Unix(1789051832, 0),
		ReceivedAt:  time.Unix(1789051833, 0),
		RawPayload:  []byte(`{"event_id":"` + eventID + `"}`),
	}
}

func TestInsertEventFirstTimeIsAccepted(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)

	inserted, err := s.InsertEvent(context.Background(), sampleEvent("evt_aaa"))
	if err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}
	if !inserted {
		t.Fatal("inserted = false pada penyisipan pertama")
	}
}

func TestInsertEventSecondTimeIsDuplicate(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	if _, err := s.InsertEvent(ctx, sampleEvent("evt_aaa")); err != nil {
		t.Fatalf("InsertEvent pertama: %v", err)
	}

	inserted, err := s.InsertEvent(ctx, sampleEvent("evt_aaa"))
	if err != nil {
		t.Fatalf("InsertEvent kedua: %v", err)
	}
	if inserted {
		t.Fatal("inserted = true padahal event_id sudah ada")
	}

	var n int
	if err := s.Pool().QueryRow(ctx,
		"SELECT count(*) FROM notification_events").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("jumlah baris = %d, mau 1", n)
	}
}

// Ini inti dari jaminan idempotency: dua request identik yang tiba
// bersamaan harus menghasilkan tepat satu baris dan tepat satu accepted.
func TestInsertEventConcurrentSameIDInsertsOnce(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	const goroutines = 8

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		accepted int
	)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := s.InsertEvent(ctx, sampleEvent("evt_race"))
			if err != nil {
				t.Errorf("InsertEvent: %v", err)
				return
			}
			if ok {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}()
	}

	close(start)
	wg.Wait()

	if accepted != 1 {
		t.Fatalf("accepted = %d, mau tepat 1", accepted)
	}

	var n int
	if err := s.Pool().QueryRow(ctx,
		"SELECT count(*) FROM notification_events").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("jumlah baris = %d, mau 1", n)
	}
}

func TestListEventsNewestFirst(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	for _, id := range []string{"evt_1", "evt_2", "evt_3"} {
		if _, err := s.InsertEvent(ctx, sampleEvent(id)); err != nil {
			t.Fatalf("InsertEvent %s: %v", id, err)
		}
	}

	got, err := s.ListEvents(ctx, 2, 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, mau 2", len(got))
	}
	if got[0].EventID != "evt_3" {
		t.Fatalf("event pertama = %s, mau evt_3 (terbaru dulu)", got[0].EventID)
	}
}
