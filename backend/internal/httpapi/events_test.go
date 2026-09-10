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
