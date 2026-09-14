package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// newVendorPasswordResetFixture: vendor admin "akbar", account "acc_1"
// (email acc_1@uji.test), dan SMTP terisi (host tidak nyata, cukup untuk
// EmailConfigured() -- pengiriman sungguhan tidak diuji di sini).
func newVendorPasswordResetFixture(t *testing.T, dashboardURL string, withSMTP bool) (http.Handler, *store.Store) {
	t.Helper()
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.UpsertVendorAdmin(ctx, "akbar", testVendorPassword); err != nil {
		t.Fatalf("UpsertVendorAdmin: %v", err)
	}
	seedActiveAccount(t, s, "acc_1")
	if withSMTP {
		if err := s.SaveNotificationSettings(ctx, settingsSecretKey(), store.NotificationSettingsUpdate{
			SMTPHost: "127.0.0.1", SMTPPort: 1, SMTPFrom: "no-reply@uji.test", UpdatedBy: "akbar",
		}); err != nil {
			t.Fatalf("SaveNotificationSettings: %v", err)
		}
	}
	api := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(),
		func() time.Time { return fixedNow }).WithDashboardURL(dashboardURL)
	return api.Handler(), s
}

func TestVendorSendPasswordReset(t *testing.T) {
	h, s := newVendorPasswordResetFixture(t, "https://dash.uji.test", true)
	cookie := loginAsVendor(t, h)

	rec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/accounts/acc_1/send-password-reset", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		rows, err := s.ListNotificationLog(context.Background(), 10, 0,
			store.NotificationLogFilter{Kinds: []string{store.NotificationKindPasswordReset}})
		if err != nil {
			t.Fatalf("ListNotificationLog: %v", err)
		}
		if len(rows) == 1 && rows[0].Recipient == "acc_1@uji.test" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("email reset (dikirim vendor) tidak pernah tercatat di riwayat notifikasi")
		}
		time.Sleep(50 * time.Millisecond)
	}

	audit, err := s.ListAuditLog(context.Background(), 50, 0)
	if err != nil {
		t.Fatalf("ListAuditLog: %v", err)
	}
	found := false
	for _, e := range audit {
		if e.Action == "PASSWORD_RESET_SENT" && e.Resource == "acc_1" && e.Actor == "akbar" {
			found = true
		}
	}
	if !found {
		t.Fatalf("audit log = %+v, mau memuat PASSWORD_RESET_SENT oleh akbar untuk acc_1", audit)
	}

	// Cooldown: permintaan kedua langsung ditolak.
	rec = vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/accounts/acc_1/send-password-reset", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, mau 429 (cooldown)", rec.Code)
	}
}

func TestVendorSendPasswordResetBelumTersedia(t *testing.T) {
	for name, tc := range map[string]struct {
		url  string
		smtp bool
	}{
		"tanpa DASHBOARD_URL": {"", true},
		"tanpa SMTP":          {"https://dash.uji.test", false},
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := newVendorPasswordResetFixture(t, tc.url, tc.smtp)
			cookie := loginAsVendor(t, h)
			rec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/accounts/acc_1/send-password-reset", "")
			if rec.Code != http.StatusServiceUnavailable || errorCode(t, rec) != "not_available" {
				t.Fatalf("status = %d body=%s, mau 503 not_available", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestVendorSendPasswordResetAccountTidakDitemukanAtauDicabut(t *testing.T) {
	h, s := newVendorPasswordResetFixture(t, "https://dash.uji.test", true)
	cookie := loginAsVendor(t, h)

	rec := vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/accounts/acc_tidak_ada/send-password-reset", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("account tidak ada: status = %d, mau 404", rec.Code)
	}

	if err := s.SetAccountAdminStatus(context.Background(), "acc_1", "revoked"); err != nil {
		t.Fatalf("SetAccountAdminStatus: %v", err)
	}
	rec = vendorRequest(t, h, cookie, http.MethodPost, "/api/v1/vendor/accounts/acc_1/send-password-reset", "")
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "account_revoked" {
		t.Fatalf("account revoked: status = %d body=%s, mau 409 account_revoked", rec.Code, rec.Body.String())
	}
}

func TestVendorSendPasswordResetButuhSesiVendor(t *testing.T) {
	h, _ := newVendorPasswordResetFixture(t, "https://dash.uji.test", true)
	rec := vendorRequest(t, h, nil, http.MethodPost, "/api/v1/vendor/accounts/acc_1/send-password-reset", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}
