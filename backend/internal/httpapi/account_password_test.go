package httpapi_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// testClock bisa dimajukan -- pencabutan sesi lama cuma terlihat bila
// sesi dan penggantian password terjadi di detik yang berbeda.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type passwordFixture struct {
	h     http.Handler
	s     *store.Store
	clock *testClock
}

// newPasswordFixture: account acc_1 (username "admin", email acc_1@uji.test).
// withSMTP mengisi SMTP yang menunjuk ke port tertutup -- pengiriman
// sungguhan gagal cepat, tapi fitur dianggap terkonfigurasi.
func newPasswordFixture(t *testing.T, dashboardURL string, withSMTP bool) passwordFixture {
	t.Helper()
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_1", BusinessName: "Toko Uji", Email: "acc_1@uji.test",
		Username: "admin", PlaintextPassword: testAdminPassword,
		Plan: "Business", MaxDevices: 10, ExpiresAt: fixedNow.Add(365 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if withSMTP {
		if err := s.SaveNotificationSettings(ctx, settingsSecretKey(), store.NotificationSettingsUpdate{
			SMTPHost: "127.0.0.1", SMTPPort: 1, SMTPFrom: "no-reply@uji.test", UpdatedBy: "test",
		}); err != nil {
			t.Fatalf("SaveNotificationSettings: %v", err)
		}
	}
	clock := &testClock{now: fixedNow}
	api := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(), clock.Now).
		WithDashboardURL(dashboardURL)
	return passwordFixture{h: api.Handler(), s: s, clock: clock}
}

func jsonRequest(t *testing.T, h http.Handler, cookie *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:5555"
	if cookie != nil {
		req.AddCookie(cookie)
	}
	h.ServeHTTP(rec, req)
	return rec
}

func countResetTokens(t *testing.T, s *store.Store) int {
	t.Helper()
	var n int
	if err := s.Pool().QueryRow(context.Background(), `SELECT count(*) FROM password_reset_tokens`).Scan(&n); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	return n
}

func TestForgotPasswordBelumTersedia(t *testing.T) {
	for name, tc := range map[string]struct {
		url  string
		smtp bool
	}{
		"tanpa DASHBOARD_URL": {"", true},
		"tanpa SMTP":          {"https://dash.uji.test", false},
	} {
		t.Run(name, func(t *testing.T) {
			f := newPasswordFixture(t, tc.url, tc.smtp)
			rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/forgot", `{"email":"acc_1@uji.test"}`)
			if rec.Code != http.StatusServiceUnavailable || errorCode(t, rec) != "not_available" {
				t.Fatalf("status = %d body=%s, mau 503 not_available", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestForgotPasswordJawabanSamaUntukEmailTerdaftarDanTidak(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", true)

	unknown := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/forgot", `{"email":"tidak@ada.test"}`)
	known := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/forgot", `{"email":"ACC_1@uji.test"}`)
	if unknown.Code != http.StatusOK || known.Code != http.StatusOK || unknown.Body.String() != known.Body.String() {
		t.Fatalf("unknown=%d %s, known=%d %s -- mau identik", unknown.Code, unknown.Body.String(), known.Code, known.Body.String())
	}
	if n := countResetTokens(t, f.s); n != 1 {
		t.Fatalf("token = %d, mau 1 (cuma untuk email terdaftar)", n)
	}

	// Email dikirim di background dan dicatat -- SMTP uji menunjuk port
	// tertutup, jadi tercatat gagal. Isi email (berisi token) tidak dicatat.
	deadline := time.Now().Add(5 * time.Second)
	for {
		rows, err := f.s.ListNotificationLog(context.Background(), 10, 0,
			store.NotificationLogFilter{Kinds: []string{store.NotificationKindPasswordReset}})
		if err != nil {
			t.Fatalf("ListNotificationLog: %v", err)
		}
		if len(rows) == 1 {
			if rows[0].Recipient != "acc_1@uji.test" || rows[0].Status != store.NotificationStatusFailed ||
				strings.Contains(rows[0].Subject, "token") {
				t.Fatalf("riwayat = %+v", rows[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("email reset tidak pernah tercatat di riwayat notifikasi")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Permintaan kedua dalam 2 menit tidak membuat token/email baru.
	jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/forgot", `{"email":"acc_1@uji.test"}`)
	var created time.Time
	if err := f.s.Pool().QueryRow(context.Background(), `SELECT created_at FROM password_reset_tokens`).Scan(&created); err != nil {
		t.Fatalf("token: %v", err)
	}
	if n := countResetTokens(t, f.s); n != 1 || !created.Equal(fixedNow) {
		t.Fatalf("token = %d (created %v), mau tetap token pertama", n, created)
	}
}

func TestForgotPasswordDibatasiPerIP(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", true)
	for i := 0; i < 5; i++ {
		jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/forgot", `{"email":"tidak@ada.test"}`)
	}
	rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/forgot", `{"email":"tidak@ada.test"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, mau 429", rec.Code)
	}
}

func TestResetPasswordMenggantiPasswordDanMencabutSesiLama(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", false)
	oldSession := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))
	if oldSession == nil {
		t.Fatal("login gagal")
	}

	hash := sha256.Sum256([]byte("token-uji"))
	if err := f.s.CreatePasswordResetToken(context.Background(), "acc_1", hash[:], fixedNow); err != nil {
		t.Fatalf("CreatePasswordResetToken: %v", err)
	}
	f.clock.Advance(5 * time.Minute)

	if rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/reset",
		`{"token":"token-uji","new_password":"pendek"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("password pendek status = %d, mau 400", rec.Code)
	}
	rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/reset",
		`{"token":"token-uji","new_password":"password-baru-123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset status = %d body=%s", rec.Code, rec.Body.String())
	}

	if rec := adminGet(t, f.h, oldSession, "/api/v1/admin/account"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sesi lama status = %d, mau 401 -- sesi sebelum reset harus dicabut", rec.Code)
	}
	if adminLogin(t, f.h, "admin", testAdminPassword).Code != http.StatusUnauthorized {
		t.Fatal("password lama masih bisa login")
	}
	if adminLogin(t, f.h, "admin", "password-baru-123").Code != http.StatusOK {
		t.Fatal("password baru tidak bisa login")
	}

	again := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/password/reset",
		`{"token":"token-uji","new_password":"password-lain-123"}`)
	if again.Code != http.StatusBadRequest || errorCode(t, again) != "invalid_token" {
		t.Fatalf("pakai ulang token status = %d body=%s, mau 400 invalid_token", again.Code, again.Body.String())
	}
}

func TestAdminChangePassword(t *testing.T) {
	f := newPasswordFixture(t, "", false)
	otherSession := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))
	f.clock.Advance(time.Minute)
	session := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))
	f.clock.Advance(time.Minute)

	wrong := jsonRequest(t, f.h, session, http.MethodPost, "/api/v1/admin/account/password",
		`{"current_password":"salah","new_password":"password-baru-123"}`)
	if wrong.Code != http.StatusUnauthorized || errorCode(t, wrong) != "invalid_credentials" {
		t.Fatalf("password saat ini salah: status = %d, mau 401 invalid_credentials", wrong.Code)
	}

	rec := jsonRequest(t, f.h, session, http.MethodPost, "/api/v1/admin/account/password",
		`{"current_password":"`+testAdminPassword+`","new_password":"password-baru-123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ganti password status = %d body=%s", rec.Code, rec.Body.String())
	}
	renewed := sessionCookieFrom(rec)
	if renewed == nil {
		t.Fatal("sesi yang mengganti password tidak diterbitkan ulang -- dia ikut ter-logout")
	}
	if got := adminGet(t, f.h, renewed, "/api/v1/admin/account"); got.Code != http.StatusOK {
		t.Fatalf("sesi baru status = %d, mau 200", got.Code)
	}
	for name, c := range map[string]*http.Cookie{"sesi lain": otherSession, "sesi lama sendiri": session} {
		if got := adminGet(t, f.h, c, "/api/v1/admin/account"); got.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, mau 401", name, got.Code)
		}
	}
}

func TestAdminChangePasswordDibatasiPerAccount(t *testing.T) {
	f := newPasswordFixture(t, "", false)
	session := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))
	for i := 0; i < 5; i++ {
		jsonRequest(t, f.h, session, http.MethodPost, "/api/v1/admin/account/password",
			`{"current_password":"salah","new_password":"password-baru-123"}`)
	}
	rec := jsonRequest(t, f.h, session, http.MethodPost, "/api/v1/admin/account/password",
		`{"current_password":"`+testAdminPassword+`","new_password":"password-baru-123"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, mau 429 walau password benar", rec.Code)
	}
}

func TestAdminAccountProfile(t *testing.T) {
	f := newPasswordFixture(t, "", false)
	if err := f.s.CreateAccount(context.Background(), store.CreateAccountInput{
		ID: "acc_2", BusinessName: "Lain", Email: "lain@uji.test", Username: "lain",
		PlaintextPassword: "rahasia123", Plan: "Business", MaxDevices: 1, ExpiresAt: fixedNow.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	session := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))

	rec := adminGet(t, f.h, session, "/api/v1/admin/account")
	var got struct {
		Account struct {
			Username     string `json:"username"`
			BusinessName string `json:"business_name"`
			Email        string `json:"email"`
		} `json:"account"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Account.Username != "admin" || got.Account.Email != "acc_1@uji.test" {
		t.Fatalf("GET account = %s", rec.Body.String())
	}

	// Ganti nama bisnis saja tidak butuh password.
	if rec := jsonRequest(t, f.h, session, http.MethodPatch, "/api/v1/admin/account",
		`{"business_name":"Toko Baru","email":"acc_1@uji.test"}`); rec.Code != http.StatusOK {
		t.Fatalf("ganti nama status = %d body=%s", rec.Code, rec.Body.String())
	}
	// Ganti email tanpa password yang benar ditolak.
	if rec := jsonRequest(t, f.h, session, http.MethodPatch, "/api/v1/admin/account",
		`{"business_name":"Toko Baru","email":"penyerang@jahat.test"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("ganti email tanpa password status = %d, mau 401", rec.Code)
	}
	for body, want := range map[string]int{
		`{"business_name":"Toko Baru","email":"bukan-email","current_password":"` + testAdminPassword + `"}`:   http.StatusBadRequest,
		`{"business_name":"","email":"acc_1@uji.test"}`:                                                        http.StatusBadRequest,
		`{"business_name":"Toko Baru","email":"lain@uji.test","current_password":"` + testAdminPassword + `"}`: http.StatusConflict,
		`{"business_name":"Toko Baru","email":"baru@uji.test","current_password":"` + testAdminPassword + `"}`: http.StatusOK,
	} {
		if rec := jsonRequest(t, f.h, session, http.MethodPatch, "/api/v1/admin/account", body); rec.Code != want {
			t.Fatalf("body %s status = %d, mau %d (%s)", body, rec.Code, want, rec.Body.String())
		}
	}
	acc, _ := f.s.GetAccountByID(context.Background(), "acc_1")
	if acc.BusinessName != "Toko Baru" || acc.Email != "baru@uji.test" {
		t.Fatalf("account = %+v", acc)
	}
}
