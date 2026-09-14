package httpapi_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func accountProfile(t *testing.T, h http.Handler, session *http.Cookie) struct {
	EmailVerified bool `json:"email_verified"`
} {
	t.Helper()
	var body struct {
		Account struct {
			EmailVerified bool `json:"email_verified"`
		} `json:"account"`
	}
	rec := adminGet(t, h, session, "/api/v1/admin/account")
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	return body.Account
}

func TestSignupMengirimEmailVerifikasi(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", true)

	rec := signupReq(t, f.h, "Toko Baru", "baru@uji.test", "toko_baru", "password123")
	if rec.Code != http.StatusOK {
		t.Fatalf("signup: status = %d body=%s", rec.Code, rec.Body.String())
	}
	session := sessionCookieFrom(rec)
	if p := accountProfile(t, f.h, session); p.EmailVerified {
		t.Fatal("email_verified = true sebelum pernah diverifikasi")
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		rows, err := f.s.ListNotificationLog(context.Background(), 10, 0,
			store.NotificationLogFilter{Kinds: []string{store.NotificationKindEmailVerification}})
		if err != nil {
			t.Fatalf("ListNotificationLog: %v", err)
		}
		if len(rows) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("email verifikasi tidak pernah tercatat di riwayat notifikasi setelah signup/update profil")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestVerifyEmailSekaliPakai(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", false)
	session := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))

	hash := sha256.Sum256([]byte("kode-uji"))
	if _, err := f.s.CreateEmailVerificationToken(context.Background(), "acc_1", hash[:], fixedNow); err != nil {
		t.Fatalf("CreateEmailVerificationToken: %v", err)
	}

	rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/email/verify", `{"token":"kode-uji"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if p := accountProfile(t, f.h, session); !p.EmailVerified {
		t.Fatal("email_verified masih false setelah verifikasi")
	}

	again := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/email/verify", `{"token":"kode-uji"}`)
	if again.Code != http.StatusBadRequest || errorCode(t, again) != "invalid_token" {
		t.Fatalf("pakai ulang token: status = %d body=%s, mau 400 invalid_token", again.Code, again.Body.String())
	}

	invalid := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/email/verify", `{"token":"tidak-ada"}`)
	if invalid.Code != http.StatusBadRequest || errorCode(t, invalid) != "invalid_token" {
		t.Fatalf("token tidak dikenal: status = %d, mau 400 invalid_token", invalid.Code)
	}
}

func TestResendVerificationEmail(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", true)
	session := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))

	rec := jsonRequest(t, f.h, session, http.MethodPost, "/api/v1/admin/account/email/resend", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	// Cooldown: permintaan kedua langsung setelahnya ditolak.
	rec = jsonRequest(t, f.h, session, http.MethodPost, "/api/v1/admin/account/email/resend", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, mau 429 (cooldown)", rec.Code)
	}

	if rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/admin/account/email/resend", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa sesi status = %d, mau 401", rec.Code)
	}
}

func TestResendVerificationEmailSudahTerverifikasi(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", true)
	session := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))

	hash := sha256.Sum256([]byte("kode-uji"))
	if _, err := f.s.CreateEmailVerificationToken(context.Background(), "acc_1", hash[:], fixedNow); err != nil {
		t.Fatalf("CreateEmailVerificationToken: %v", err)
	}
	if rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/email/verify", `{"token":"kode-uji"}`); rec.Code != http.StatusOK {
		t.Fatalf("verify: status = %d", rec.Code)
	}

	rec := jsonRequest(t, f.h, session, http.MethodPost, "/api/v1/admin/account/email/resend", "")
	var body struct {
		AlreadyVerified bool `json:"already_verified"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.AlreadyVerified {
		t.Fatalf("resend setelah terverifikasi = %s, mau already_verified true", rec.Body.String())
	}
}

func TestGantiEmailMengirimVerifikasiBaru(t *testing.T) {
	f := newPasswordFixture(t, "https://dash.uji.test", true)
	session := sessionCookieFrom(adminLogin(t, f.h, "admin", testAdminPassword))

	hash := sha256.Sum256([]byte("kode-lama"))
	if _, err := f.s.CreateEmailVerificationToken(context.Background(), "acc_1", hash[:], fixedNow); err != nil {
		t.Fatalf("CreateEmailVerificationToken: %v", err)
	}
	if rec := jsonRequest(t, f.h, nil, http.MethodPost, "/api/v1/email/verify", `{"token":"kode-lama"}`); rec.Code != http.StatusOK {
		t.Fatalf("verify: status = %d", rec.Code)
	}
	if p := accountProfile(t, f.h, session); !p.EmailVerified {
		t.Fatal("belum terverifikasi sebelum ganti email")
	}

	rec := jsonRequest(t, f.h, session, http.MethodPatch, "/api/v1/admin/account",
		`{"business_name":"Toko Uji","email":"baru@uji.test","current_password":"`+testAdminPassword+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ganti email: status = %d body=%s", rec.Code, rec.Body.String())
	}
	if p := accountProfile(t, f.h, session); p.EmailVerified {
		t.Fatal("email_verified masih true setelah ganti ke alamat baru yang belum diverifikasi")
	}
}
