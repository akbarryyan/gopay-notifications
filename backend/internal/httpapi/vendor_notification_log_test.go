package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestVendorNotificationLog(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.UpsertVendorAdmin(ctx, "akbar", testVendorPassword); err != nil {
		t.Fatalf("UpsertVendorAdmin: %v", err)
	}
	seedActiveAccount(t, s, "acc_1")
	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(),
		func() time.Time { return fixedNow }).Handler()

	acc, device := "acc_1", "HP Kasir"
	errMsg := "chat not found"
	for _, e := range []store.NotificationLogEntry{
		{AccountID: &acc, DeviceName: &device, Kind: store.NotificationKindDeviceOffline, Channel: "email",
			Recipient: "acc_1@uji.test", Subject: "HP offline", Status: store.NotificationStatusSent},
		{AccountID: &acc, DeviceName: &device, Kind: store.NotificationKindDeviceOffline, Channel: "telegram",
			Recipient: "123", Subject: "HP offline", Status: store.NotificationStatusFailed, Error: &errMsg},
	} {
		if err := s.LogNotification(ctx, e); err != nil {
			t.Fatalf("LogNotification: %v", err)
		}
	}

	if rec := vendorRequest(t, h, nil, http.MethodGet, "/api/v1/vendor/notification-log", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa sesi status = %d, mau 401", rec.Code)
	}

	cookie := loginAsVendor(t, h)
	rec := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/notification-log?status=failed", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Notifications []struct {
			BusinessName *string `json:"business_name"`
			DeviceName   *string `json:"device_name"`
			Kind         string  `json:"kind"`
			Channel      string  `json:"channel"`
			Status       string  `json:"status"`
			Error        *string `json:"error"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Notifications) != 1 {
		t.Fatalf("notifications = %d, mau 1 (filter status=failed)", len(body.Notifications))
	}
	n := body.Notifications[0]
	if n.Channel != "telegram" || n.Error == nil || *n.Error != errMsg || n.BusinessName == nil ||
		*n.BusinessName != "acc_1" || n.DeviceName == nil || *n.DeviceName != device {
		t.Fatalf("baris = %+v", n)
	}

	for _, q := range []string{"kind=bukan", "status=bukan", "channel=sms", "from=kemarin"} {
		rec := vendorRequest(t, h, cookie, http.MethodGet, "/api/v1/vendor/notification-log?"+q, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, mau 400", q, rec.Code)
		}
	}
}
