package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

var (
	validNotificationKinds = map[string]bool{
		store.NotificationKindExpiryReminder: true,
		store.NotificationKindDeviceOffline:  true,
		store.NotificationKindDeviceOnline:   true,
		store.NotificationKindTest:           true,
	}
	validNotificationStatuses = map[string]bool{
		store.NotificationStatusSent:   true,
		store.NotificationStatusFailed: true,
	}
	validNotificationChannels = map[string]bool{"email": true, "telegram": true}
)

type vendorNotificationLogJSON struct {
	ID           int64   `json:"id"`
	AccountID    *string `json:"account_id"`
	BusinessName *string `json:"business_name"`
	DeviceID     *string `json:"device_id"`
	DeviceName   *string `json:"device_name"`
	Kind         string  `json:"kind"`
	Channel      string  `json:"channel"`
	Recipient    string  `json:"recipient"`
	Subject      string  `json:"subject"`
	Status       string  `json:"status"`
	Error        *string `json:"error"`
	CreatedAt    string  `json:"created_at"`
}

// csvListParam memecah query param "a,b" dan memvalidasi tiap nilainya.
func csvListParam(r *http.Request, name string, valid map[string]bool) ([]string, string) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, ""
	}
	var out []string
	for _, v := range strings.Split(raw, ",") {
		if !valid[v] {
			return nil, name + " tidak dikenal: " + v
		}
		out = append(out, v)
	}
	return out, ""
}

// handleVendorNotificationLog melayani halaman Riwayat Notifikasi di
// Vendor Dashboard -- read-only, lintas semua account, pola parsing query
// param sama dengan handleVendorWebhookDeliveries.
func (a *API) handleVendorNotificationLog(w http.ResponseWriter, r *http.Request) {
	limit, err := intParam(r, "limit", 50)
	if err != nil || limit < 1 || limit > 1000 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "limit harus bilangan bulat 1..1000")
		return
	}
	offset, err := intParam(r, "offset", 0)
	if err != nil || offset < 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "offset harus bilangan bulat >= 0")
		return
	}

	filter := store.NotificationLogFilter{Query: r.URL.Query().Get("q")}
	var msg string
	if filter.Kinds, msg = csvListParam(r, "kind", validNotificationKinds); msg != "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", msg)
		return
	}
	if filter.Statuses, msg = csvListParam(r, "status", validNotificationStatuses); msg != "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", msg)
		return
	}
	if filter.Channels, msg = csvListParam(r, "channel", validNotificationChannels); msg != "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", msg)
		return
	}
	if raw := r.URL.Query().Get("from"); raw != "" {
		from, err := time.Parse("2006-01-02", raw)
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "from harus YYYY-MM-DD")
			return
		}
		filter.From = &from
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		to, err := time.Parse("2006-01-02", raw)
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "to harus YYYY-MM-DD")
			return
		}
		to = to.Add(24*time.Hour - time.Nanosecond)
		filter.To = &to
	}

	rows, err := a.store.ListNotificationLog(r.Context(), limit, offset, filter)
	if err != nil {
		slog.Error("vendor: ambil riwayat notifikasi gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]vendorNotificationLogJSON, 0, len(rows))
	for _, n := range rows {
		out = append(out, vendorNotificationLogJSON{
			ID: n.ID, AccountID: n.AccountID, BusinessName: n.BusinessName,
			DeviceID: n.DeviceID, DeviceName: n.DeviceName, Kind: n.Kind, Channel: n.Channel,
			Recipient: n.Recipient, Subject: n.Subject, Status: n.Status, Error: n.Error,
			CreatedAt: n.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "notifications": out})
}
