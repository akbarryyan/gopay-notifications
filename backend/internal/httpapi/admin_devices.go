package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type adminDeviceJSON struct {
	DeviceID          string  `json:"device_id"`
	Name              string  `json:"name"`
	Enabled           bool    `json:"enabled"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	LastSeenAt        *string `json:"last_seen_at"`
	HeartbeatAt       *string `json:"heartbeat_at"`
	AppVersion        *string `json:"app_version"`
	AndroidVersion    *string `json:"android_version"`
	ListenerConnected *bool   `json:"listener_connected"`
	PendingCount      *int    `json:"pending_count"`
	FailedCount       *int    `json:"failed_count"`
}

func toAdminDeviceJSON(d store.Device, now time.Time) adminDeviceJSON {
	out := adminDeviceJSON{
		DeviceID:          d.DeviceID,
		Name:              d.Name,
		Enabled:           d.Enabled,
		Status:            string(store.StatusOf(d, now)),
		CreatedAt:         d.CreatedAt.Format(time.RFC3339),
		AppVersion:        d.AppVersion,
		AndroidVersion:    d.AndroidVersion,
		ListenerConnected: d.ListenerConnected,
		PendingCount:      d.PendingCount,
		FailedCount:       d.FailedCount,
	}
	if d.LastSeenAt != nil {
		s := d.LastSeenAt.Format(time.RFC3339)
		out.LastSeenAt = &s
	}
	if d.HeartbeatAt != nil {
		s := d.HeartbeatAt.Format(time.RFC3339)
		out.HeartbeatAt = &s
	}
	return out
}

// handleAdminDevices mengembalikan seluruh device untuk halaman Devices.
func (a *API) handleAdminDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := a.store.ListDevices(r.Context())
	if err != nil {
		slog.Error("ambil devices gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]adminDeviceJSON, 0, len(devices))
	for _, d := range devices {
		out = append(out, toAdminDeviceJSON(d, a.now()))
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

type setDeviceEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// handleAdminSetDeviceEnabled mengaktifkan/menonaktifkan satu device.
//
// Path: PATCH /api/v1/admin/devices/{deviceID}
func (a *API) handleAdminSetDeviceEnabled(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceID")
	if deviceID == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "device id tidak valid")
		return
	}

	var req setDeviceEnabledRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}

	err := a.store.SetDeviceEnabled(r.Context(), deviceID, req.Enabled)
	if errors.Is(err, store.ErrDeviceNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "device tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("set device enabled gagal", "device_id", deviceID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
