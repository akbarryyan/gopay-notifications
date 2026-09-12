package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type heartbeatRequest struct {
	AndroidVersion    string `json:"android_version"`
	ListenerConnected bool   `json:"listener_connected"`
	PendingCount      int    `json:"pending_count"`
	FailedCount       int    `json:"failed_count"`
}

type heartbeatResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	// Dikembalikan supaya perangkat dapat mendeteksi jamnya meleset tanpa
	// memanggil /health terpisah.
	ServerTime int64 `json:"server_time"`
}

// handleHeartbeat menerima laporan kondisi berkala dari perangkat.
//
// Tanpa heartbeat, last_seen_at hanya bergerak saat ada pembayaran — dan
// perangkat yang dibunuh OEM baru diketahui ketika sebuah pembayaran terlewat.
func (a *API) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	device, ok := DeviceFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "device tidak ada di context")
		return
	}
	raw, ok := RawBodyFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "body tidak ada di context")
		return
	}

	var req heartbeatRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.PendingCount < 0 || req.FailedCount < 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"pending_count dan failed_count tidak boleh negatif")
		return
	}

	err := a.store.RecordHeartbeat(r.Context(), device.DeviceID, store.Heartbeat{
		AndroidVersion:    req.AndroidVersion,
		ListenerConnected: req.ListenerConnected,
		PendingCount:      req.PendingCount,
		FailedCount:       req.FailedCount,
	})
	if err != nil {
		slog.Error("rekam heartbeat gagal", "device_id", device.DeviceID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	// Izin aktif tetapi listener tidak terikat adalah gejala service dibunuh
	// diam-diam. Dicatat sebagai peringatan supaya terlihat di log server.
	if !req.ListenerConnected {
		slog.Warn("listener tidak terikat di perangkat", "device_id", device.DeviceID)
	}

	writeJSON(w, http.StatusOK, heartbeatResponse{
		Success:    true,
		Status:     "ok",
		ServerTime: a.now().Unix(),
	})
}
