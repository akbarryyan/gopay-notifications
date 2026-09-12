package httpapi

import (
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type deviceMeResponse struct {
	DeviceID   string  `json:"device_id"`
	Name       string  `json:"name"`
	Enabled    bool    `json:"enabled"`
	Status     string  `json:"status"`
	LastSeenAt *string `json:"last_seen_at"`
	AppVersion *string `json:"app_version"`
}

// handleDeviceMe melayani tombol Test Connection di aplikasi Android.
func (a *API) handleDeviceMe(w http.ResponseWriter, r *http.Request) {
	device, ok := DeviceFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "device tidak ada di context")
		return
	}

	resp := deviceMeResponse{
		DeviceID:   device.DeviceID,
		Name:       device.Name,
		Enabled:    device.Enabled,
		Status:     string(store.StatusOf(device, a.now())),
		AppVersion: device.AppVersion,
	}
	if device.LastSeenAt != nil {
		s := device.LastSeenAt.Format(time.RFC3339)
		resp.LastSeenAt = &s
	}
	writeJSON(w, http.StatusOK, resp)
}
