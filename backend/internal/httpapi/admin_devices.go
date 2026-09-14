package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	accountID, _ := AccountFromContext(r.Context())
	devices, err := a.store.ListDevices(r.Context(), accountID)
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

type createDeviceRequest struct {
	Name string `json:"name"`
}

// randomDeviceID mengikuti pola persis cmd/devicetool: "dev_" + hex 8 byte
// acak.
func randomDeviceID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("acak device id: %w", err)
	}
	return "dev_" + hex.EncodeToString(b), nil
}

// handleAdminCreateDevice adalah swalayan tambah device dari Customer
// Dashboard sendiri -- sebelumnya SATU-SATUNYA cara membuat device adalah
// cmd/devicetool di server, yang berarti tiap customer baru mau pasang HP
// kedua/ketiga harus minta tolong vendor generate ID+secret manual. Dengan
// endpoint ini, customer bisa lakukan sendiri, persis alur API key/webhook
// secret yang sudah ada: ditampilkan sekali, tidak bisa dilihat lagi.
//
// Kuota max_devices (dari plan akun, -1 berarti unlimited) ditegakkan di
// sini, bukan cuma di UI -- kalau tidak, customer Starter (3 device) bisa
// menambah device tanpa batas lewat panggilan API langsung, melewati
// batasan plan yang jadi dasar harga.
//
// Format secret SENGAJA sama persis dengan cmd/devicetool: 32 byte acak
// di-base64-encode dulu, string base64 itu (bukan byte mentahnya) yang
// dienkripsi & disimpan -- device Android yang sudah dikonfigurasi manual
// lewat devicetool dan yang dibuat lewat endpoint ini harus punya bentuk
// secret yang sama, supaya tidak ada dua cara device menghitung HMAC-nya.
func (a *API) handleAdminCreateDevice(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	var req createDeviceRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Name == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "name wajib diisi")
		return
	}

	account, err := a.store.GetAccountByID(r.Context(), accountID)
	if err != nil {
		slog.Error("ambil account untuk cek kuota device gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	existing, err := a.store.ListDevices(r.Context(), accountID)
	if err != nil {
		slog.Error("hitung device untuk cek kuota gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if account.MaxDevices >= 0 && len(existing) >= account.MaxDevices {
		a.writeError(w, http.StatusConflict, "device_limit_reached",
			fmt.Sprintf("plan %s dibatasi %d device -- hubungi kami untuk upgrade plan", account.Plan, account.MaxDevices))
		return
	}

	deviceID, err := randomDeviceID()
	if err != nil {
		slog.Error("generate device id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		slog.Error("acak device secret gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	secretB64 := base64.StdEncoding.EncodeToString(secret)

	if err := a.store.CreateDevice(r.Context(), a.encKey, accountID, deviceID, req.Name, []byte(secretB64)); err != nil {
		slog.Error("create device gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	a.logActivity(r, accountID, store.ActivityDeviceAdded, map[string]any{"device_id": deviceID, "name": req.Name})

	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true, "device_id": deviceID, "device_secret": secretB64,
	})
}

// handleAdminDeleteDevice menghapus device permanen -- membebaskan slot
// kuota, beda dari PATCH .../enabled yang cuma menonaktifkan (device tetap
// ada, tetap terhitung ke max_devices). Ditolak 409 kalau device ini sudah
// punya riwayat event (lihat store.ErrDeviceHasEvents) -- nonaktifkan saja
// untuk device yang sudah pernah dipakai.
func (a *API) handleAdminDeleteDevice(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	deviceID := r.PathValue("deviceID")
	if deviceID == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "device id tidak valid")
		return
	}

	err := a.store.DeleteDevice(r.Context(), accountID, deviceID)
	if errors.Is(err, store.ErrDeviceNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "device tidak ditemukan")
		return
	}
	if errors.Is(err, store.ErrDeviceHasEvents) {
		a.writeError(w, http.StatusConflict, "device_has_events",
			"device ini sudah punya riwayat event -- nonaktifkan saja, tidak bisa dihapus")
		return
	}
	if err != nil {
		slog.Error("delete device gagal", "device_id", deviceID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	a.logActivity(r, accountID, store.ActivityDeviceDeleted, map[string]any{"device_id": deviceID})

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

type setDeviceEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// handleAdminSetDeviceEnabled mengaktifkan/menonaktifkan satu device.
//
// Path: PATCH /api/v1/admin/devices/{deviceID}
func (a *API) handleAdminSetDeviceEnabled(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
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

	err := a.store.SetDeviceEnabled(r.Context(), accountID, deviceID, req.Enabled)
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
