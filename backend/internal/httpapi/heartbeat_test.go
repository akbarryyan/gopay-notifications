package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const heartbeatBody = `{"android_version":"13","listener_connected":true,` +
	`"pending_count":2,"failed_count":1}`

func postHeartbeat(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodPost, "/api/v1/devices/heartbeat",
		body, fixedNow.Unix(), testSecret))
	return rec
}

func TestHeartbeatDiterimaDanMengembalikanJamServer(t *testing.T) {
	h := newAPIWithDevice(t)

	rec := postHeartbeat(t, h, heartbeatBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Success    bool   `json:"success"`
		Status     string `json:"status"`
		ServerTime int64  `json:"server_time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Success || body.Status != "ok" {
		t.Fatalf("body = %+v", body)
	}
	if body.ServerTime != fixedNow.Unix() {
		t.Fatalf("server_time = %d, mau %d", body.ServerTime, fixedNow.Unix())
	}
}

func TestHeartbeatMengubahStatusDeviceJadiOnline(t *testing.T) {
	h := newAPIWithDevice(t)

	// Sebelum heartbeat pertama, device baru berstatus PENDING.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))
	var sebelum struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sebelum); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sebelum.Status != "PENDING" {
		t.Fatalf("status awal = %q, mau PENDING", sebelum.Status)
	}

	postHeartbeat(t, h, heartbeatBody)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))
	var sesudah struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sesudah); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sesudah.Status != "ONLINE" {
		t.Fatalf("status sesudah = %q, mau ONLINE", sesudah.Status)
	}
}

func TestHeartbeatMenolakPayloadTakBerbentuk(t *testing.T) {
	h := newAPIWithDevice(t)

	rec := postHeartbeat(t, h, `{"pending_count":`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_payload" {
		t.Fatalf("error = %q", got)
	}
}

func TestHeartbeatMenolakHitunganNegatif(t *testing.T) {
	h := newAPIWithDevice(t)

	for _, body := range []string{
		`{"listener_connected":true,"pending_count":-1,"failed_count":0}`,
		`{"listener_connected":true,"pending_count":0,"failed_count":-5}`,
	} {
		rec := postHeartbeat(t, h, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d untuk %s, mau 400", rec.Code, body)
		}
	}
}

func TestHeartbeatMemerlukanAutentikasi(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	// Tanpa header HMAC sama sekali.
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/devices/heartbeat", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestHeartbeatListenerTidakTerikatTetapDiterima(t *testing.T) {
	// Laporan "listener tidak terikat" adalah informasi paling berharga dari
	// heartbeat. Menolaknya berarti membuang justru sinyal yang dicari.
	h := newAPIWithDevice(t)

	rec := postHeartbeat(t, h,
		`{"android_version":"13","listener_connected":false,"pending_count":0,"failed_count":0}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
}
