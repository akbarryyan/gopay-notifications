package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type eventsResponse struct {
	Events []struct {
		EventID    string  `json:"event_id"`
		Title      *string `json:"title"`
		Text       *string `json:"text"`
		AmountHint *int64  `json:"amount_hint"`
	} `json:"events"`
}

func getEvents(t *testing.T, h http.Handler, query string) eventsResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events"+query, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var out eventsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestEventsEmptyReturnsEmptyArray(t *testing.T) {
	h := newAPIWithDevice(t)

	got := getEvents(t, h, "")

	if got.Events == nil {
		t.Fatal("events = null, mau array kosong — klien tidak boleh dipaksa menangani null")
	}
	if len(got.Events) != 0 {
		t.Fatalf("len = %d, mau 0", len(got.Events))
	}
}

func TestEventsReturnsStoredEvent(t *testing.T) {
	h := newAPIWithDevice(t)
	postCallback(t, h, validBody)

	got := getEvents(t, h, "")

	if len(got.Events) != 1 {
		t.Fatalf("len = %d, mau 1", len(got.Events))
	}
	e := got.Events[0]
	if e.EventID != "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c" {
		t.Fatalf("event_id = %s", e.EventID)
	}
	if e.Title == nil || *e.Title != "Transfer masuk" {
		t.Fatalf("title = %v, mau \"Transfer masuk\"", e.Title)
	}
	if e.AmountHint == nil || *e.AmountHint != 1 {
		t.Fatalf("amount_hint = %v, mau 1", e.AmountHint)
	}
}

func TestEventsRejectsBadLimit(t *testing.T) {
	h := newAPIWithDevice(t)

	for _, q := range []string{"?limit=0", "?limit=-1", "?limit=abc", "?limit=1001", "?offset=-1"} {
		t.Run(q, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events"+q, nil))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, mau 400", rec.Code)
			}
		})
	}
}

func TestEventsFiltersBySource(t *testing.T) {
	h := newAPIWithDevice(t)
	postCallback(t, h, validBody)

	got := getEvents(t, h, "?source=gopay")
	if len(got.Events) != 1 {
		t.Fatalf("len dengan source=gopay = %d, mau 1", len(got.Events))
	}
}

func TestEventsRejectsUnknownSource(t *testing.T) {
	h := newAPIWithDevice(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events?source=dana", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (source belum ada di connector registry)", rec.Code)
	}
}

func TestEventsFiltersByQuery(t *testing.T) {
	h := newAPIWithDevice(t)
	postCallback(t, h, validBody)

	byTitle := getEvents(t, h, "?q=Transfer")
	if len(byTitle.Events) != 1 {
		t.Fatalf("len dengan q=Transfer = %d, mau 1 (cocok ke title)", len(byTitle.Events))
	}

	byDevice := getEvents(t, h, "?q=dev_01ABC")
	if len(byDevice.Events) != 1 {
		t.Fatalf("len dengan q=dev_01ABC = %d, mau 1 (cocok ke device_id)", len(byDevice.Events))
	}

	noMatch := getEvents(t, h, "?q=tidak-akan-ada-yang-cocok")
	if len(noMatch.Events) != 0 {
		t.Fatalf("len dengan q tidak cocok = %d, mau 0", len(noMatch.Events))
	}
}

func TestEventsFiltersByDateRange(t *testing.T) {
	// validBody.received_at = 2026-09-10T19:30:33+07:00 = 2026-09-10T12:30:33Z.
	h := newAPIWithDevice(t)
	postCallback(t, h, validBody)

	sameDay := getEvents(t, h, "?from=2026-09-10&to=2026-09-10")
	if len(sameDay.Events) != 1 {
		t.Fatalf("len dengan rentang hari yang sama = %d, mau 1", len(sameDay.Events))
	}

	before := getEvents(t, h, "?to=2026-09-09")
	if len(before.Events) != 0 {
		t.Fatalf("len dengan to sebelum event = %d, mau 0", len(before.Events))
	}

	after := getEvents(t, h, "?from=2026-09-11")
	if len(after.Events) != 0 {
		t.Fatalf("len dengan from sesudah event = %d, mau 0", len(after.Events))
	}
}

func TestEventsRejectsBadDateRange(t *testing.T) {
	h := newAPIWithDevice(t)

	for _, q := range []string{"?from=bukan-tanggal", "?to=2026/09/10"} {
		t.Run(q, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events"+q, nil))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, mau 400", rec.Code)
			}
		})
	}
}
