package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type activityRow struct {
	Action    string          `json:"action"`
	IPAddress *string         `json:"ip_address"`
	Metadata  json.RawMessage `json:"metadata"`
}

func fetchActivity(t *testing.T, h http.Handler, cookie *http.Cookie, query string) []activityRow {
	t.Helper()
	path := "/api/v1/admin/activity"
	if query != "" {
		path += "?" + query
	}
	rec := adminGet(t, h, cookie, path)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Activity []activityRow `json:"activity"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	return body.Activity
}

func TestActivityLogButuhSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/activity", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestActivityLogMencatatLoginBerhasilDanGagal(t *testing.T) {
	h := newAPIWithAdmin(t)

	wrong := adminLogin(t, h, "admin", "password-salah")
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("login salah status = %d, mau 401", wrong.Code)
	}
	cookie := loginAsAdmin(t, h)

	rows := fetchActivity(t, h, cookie, "")
	if len(rows) != 2 {
		t.Fatalf("activity = %+v, mau 2 baris (login gagal lalu login berhasil)", rows)
	}
	if rows[0].Action != "login_success" || rows[1].Action != "login_failed" {
		t.Fatalf("activity = %+v, mau login_success lalu login_failed (terbaru dulu)", rows)
	}
	if rows[0].IPAddress == nil {
		t.Fatal("IPAddress kosong untuk login_success")
	}
}

func TestActivityLogMencatatGantiPasswordApiKeyDanDevice(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	if rec := jsonRequest(t, h, cookie, http.MethodPost, "/api/v1/admin/account/password",
		`{"current_password":"`+testAdminPassword+`","new_password":"password-baru-123"}`); rec.Code != http.StatusOK {
		t.Fatalf("ganti password: status = %d body=%s", rec.Code, rec.Body.String())
	}

	created := createAPIKeyReq(t, h, cookie, "Website Toko")
	var key struct {
		ID string `json:"id"`
	}
	json.Unmarshal(created.Body.Bytes(), &key)
	revokeReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/"+key.ID, nil)
	revokeReq.AddCookie(cookie)
	revokeRec := httptest.NewRecorder()
	h.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke api key: status = %d body=%s", revokeRec.Code, revokeRec.Body.String())
	}

	deviceRec := adminPost(t, h, cookie, "/api/v1/admin/devices", `{"name":"HP Kasir"}`)
	var device struct {
		DeviceID string `json:"device_id"`
	}
	json.Unmarshal(deviceRec.Body.Bytes(), &device)
	if rec := adminDelete(t, h, cookie, "/api/v1/admin/devices/"+device.DeviceID); rec.Code != http.StatusOK {
		t.Fatalf("delete device: status = %d body=%s", rec.Code, rec.Body.String())
	}

	rows := fetchActivity(t, h, cookie, "")
	actions := make([]string, len(rows))
	for i, r := range rows {
		actions[i] = r.Action
	}
	want := []string{"device_deleted", "device_added", "api_key_revoked", "api_key_created", "password_changed", "login_success"}
	if len(actions) != len(want) {
		t.Fatalf("actions = %v, mau %v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("actions = %v, mau %v", actions, want)
		}
	}

	filtered := fetchActivity(t, h, cookie, "action=device_added,device_deleted")
	if len(filtered) != 2 {
		t.Fatalf("filtered = %+v, mau 2 (device_added, device_deleted)", filtered)
	}

	if rec := adminGet(t, h, cookie, "/api/v1/admin/activity?action=bukan_aksi"); rec.Code != http.StatusBadRequest {
		t.Fatalf("action tidak dikenal: status = %d, mau 400", rec.Code)
	}
}

func TestActivityLogTidakBocorLintasAccount(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie1 := loginAsAdmin(t, h)

	// acc_1 (admin) login sekali, lalu account lain dibuat lewat signup --
	// riwayat masing-masing harus terpisah, tidak tercampur.
	signupRec := signupReq(t, h, "Toko Lain", "lain@uji.test", "toko_lain", "password123")
	cookie2 := sessionCookieFrom(signupRec)

	rows1 := fetchActivity(t, h, cookie1, "")
	rows2 := fetchActivity(t, h, cookie2, "")
	if len(rows1) != 1 || rows1[0].Action != "login_success" {
		t.Fatalf("rows1 = %+v, mau cuma 1 login_success milik acc_1 sendiri", rows1)
	}
	if len(rows2) != 0 {
		t.Fatalf("rows2 = %+v, mau kosong -- signup tidak mencatat login_success", rows2)
	}
}
