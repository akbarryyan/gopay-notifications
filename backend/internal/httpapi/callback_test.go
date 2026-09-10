package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

const validBody = `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
	`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
	`"title":"Transfer masuk","text":"Rp1 dari icaangg udah masuk ke GoPay kamu.",` +
	`"big_text":null,"posted_at":1789051832829},"amount_hint":1,` +
	`"received_at":"2026-09-10T19:30:33+07:00"}`

func postCallback(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodPost, "/api/v1/callback/gopay", body, fixedNow.Unix(), testSecret))
	return rec
}

func callbackStatus(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Success bool   `json:"success"`
		EventID string `json:"event_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}
	if !body.Success {
		t.Fatalf("success = false, body=%s", rec.Body.String())
	}
	return body.Status
}

func TestCallbackFirstTimeReturnsAccepted(t *testing.T) {
	h := newAPIWithDevice(t)

	rec := postCallback(t, h, validBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := callbackStatus(t, rec); got != "accepted" {
		t.Fatalf("status = %q, mau accepted", got)
	}
}

func TestCallbackSecondTimeReturnsDuplicate(t *testing.T) {
	h := newAPIWithDevice(t)

	postCallback(t, h, validBody)
	rec := postCallback(t, h, validBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
	if got := callbackStatus(t, rec); got != "duplicate" {
		t.Fatalf("status = %q, mau duplicate", got)
	}
}

func TestCallbackRejectsMalformedJSON(t *testing.T) {
	h := newAPIWithDevice(t)

	rec := postCallback(t, h, `{"event_id":`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_payload" {
		t.Fatalf("error = %q, mau invalid_payload", got)
	}
}

func TestCallbackRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			"event_id kosong",
			`{"event_id":"","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"event_id format salah",
			`{"event_id":"bukan-evt","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"device_id tidak cocok dengan yang terautentikasi",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_LAIN","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"source tidak dikenal",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"dana","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"package_name kosong",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"posted_at nol",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":0},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"received_at bukan RFC3339",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"10 Sep 2026"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newAPIWithDevice(t)
			rec := postCallback(t, h, tc.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
			}
			if got := errorCode(t, rec); got != "invalid_payload" {
				t.Fatalf("error = %q, mau invalid_payload", got)
			}
		})
	}
}

func TestCallbackAcceptsNullOptionalFields(t *testing.T) {
	h := newAPIWithDevice(t)

	body := `{"event_id":"evt_00000000000000000000000000000001","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,` +
		`"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`

	rec := postCallback(t, h, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := callbackStatus(t, rec); got != "accepted" {
		t.Fatalf("status = %q, mau accepted", got)
	}
}

func TestCallbackStoresRawPayloadVerbatim(t *testing.T) {
	h := newAPIWithDevice(t)
	postCallback(t, h, validBody)

	s := openTestStore(t)
	var raw string
	err := s.Pool().QueryRow(context.Background(),
		"SELECT raw_payload::text FROM notification_events WHERE event_id = $1",
		"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c").Scan(&raw)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !json.Valid([]byte(raw)) {
		t.Fatal("raw_payload bukan JSON yang sah")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("decode raw_payload: %v", err)
	}
	if got["event_id"] != "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c" {
		t.Fatalf("raw_payload tidak memuat event_id asli: %v", got)
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}
