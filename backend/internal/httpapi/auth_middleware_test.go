package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

const testSecret = "secret-untuk-test"

func encKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 3)
	}
	return k
}

func adminSessionKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i*5 + 1)
	}
	return k
}

func webhookSecretKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i*7 + 2)
	}
	return k
}

// newAPIWithDevice menyiapkan API lengkap dengan satu device terdaftar.
func newAPIWithDevice(t *testing.T) http.Handler {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL belum diset. Jalankan: make db-up migrate")
	}

	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)

	if _, err := s.Pool().Exec(ctx,
		"TRUNCATE notification_events, event_reviews, invoices, api_keys, webhook_deliveries, webhook_endpoints, devices RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	return httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), func() time.Time { return fixedNow }).Handler()
}

// signedRequest membuat request yang sudah ditandatangani dengan benar.
func signedRequest(method, path, body string, ts int64, secret string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Id", "dev_01ABC")
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))
	s := auth.SigningString("dev_01ABC", ts, []byte(body))
	req.Header.Set("X-Signature", auth.Sign([]byte(secret), s))
	return req
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v (body=%s)", err, rec.Body.String())
	}
	return body.Error
}

func TestAuthAcceptsValidSignature(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestAuthRejectsWrongSecret(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), "secret-salah"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_signature" {
		t.Fatalf("error = %q, mau invalid_signature", got)
	}
}

func TestAuthRejectsUnknownDevice(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	req := signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret)
	req.Header.Set("X-Device-Id", "dev_TIDAKADA")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_signature" {
		t.Fatalf("error = %q, mau invalid_signature — device tidak dikenal tidak boleh dibedakan dari secret salah", got)
	}
}

func TestAuthRejectsClockSkewWithServerTime(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	skewed := fixedNow.Add(-301 * time.Second).Unix()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", skewed, testSecret))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if got := errorCode(t, rec); got != "clock_skew" {
		t.Fatalf("error = %q, mau clock_skew", got)
	}

	var body struct {
		ServerTime int64 `json:"server_time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ServerTime != fixedNow.Unix() {
		t.Fatalf("server_time = %d, mau %d", body.ServerTime, fixedNow.Unix())
	}
}

func TestAuthRejectsMissingHeaders(t *testing.T) {
	h := newAPIWithDevice(t)

	for _, drop := range []string{"X-Device-Id", "X-Timestamp", "X-Signature"} {
		t.Run("tanpa "+drop, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret)
			req.Header.Del(drop)

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, mau 401", rec.Code)
			}
		})
	}
}

func TestAuthRejectsNonNumericTimestamp(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	req := signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret)
	req.Header.Set("X-Timestamp", "kemarin")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestAuthMencatatVersiAplikasiDariHeader(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	req := signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret)
	req.Header.Set("X-App-Version", "1.4.2")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}

	var body struct {
		AppVersion *string `json:"app_version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.AppVersion == nil || *body.AppVersion != "1.4.2" {
		t.Fatalf("app_version = %v, mau 1.4.2", body.AppVersion)
	}
}

func TestAuthTanpaHeaderVersiTetapDiterima(t *testing.T) {
	// Aplikasi versi lama tidak mengirim X-App-Version. Itu bukan alasan
	// menolak request dan memutus pembayaran.
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
}

func TestAuthRejectsDisabledDevice(t *testing.T) {
	h := newAPIWithDevice(t)

	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if _, err := s.Pool().Exec(context.Background(),
		"UPDATE devices SET enabled = false WHERE device_id = $1", "dev_01ABC"); err != nil {
		t.Fatalf("disable: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, mau 403", rec.Code)
	}
	if got := errorCode(t, rec); got != "device_disabled" {
		t.Fatalf("error = %q, mau device_disabled", got)
	}
}
