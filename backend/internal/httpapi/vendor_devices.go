package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// handleVendorListDevices mengembalikan seluruh device milik SATU account
// customer, untuk halaman detail account di Vendor Dashboard -- sebelumnya
// vendor cuma bisa melihat max_devices (kuota), tidak pernah device
// sungguhannya, jadi tidak bisa bantu troubleshoot ("device saya offline")
// tanpa buka database langsung.
//
// Reuse toAdminDeviceJSON/adminDeviceJSON dari admin_devices.go -- bentuk
// datanya sama persis dengan yang dilihat customer sendiri di Customer
// Dashboard, cuma diakses lewat sesi vendor + accountID dari path, bukan
// dari sesi customer yang login.
func (a *API) handleVendorListDevices(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountID")
	devices, err := a.store.ListDevices(r.Context(), accountID)
	if err != nil {
		slog.Error("vendor: ambil devices gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]adminDeviceJSON, 0, len(devices))
	for _, d := range devices {
		out = append(out, toAdminDeviceJSON(d, a.now()))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "devices": out})
}

type vendorCreateDeviceRequest struct {
	Name string `json:"name"`
}

// handleVendorCreateDevice adalah versi vendor dari handleAdminCreateDevice
// (admin_devices.go) -- sebelumnya SATU-SATUNYA cara vendor menambah device
// untuk customer adalah `cmd/devicetool` lewat SSH ke server, dipakai kalau
// customer minta tolong pasang HP baru tapi belum sempat/tidak bisa lewat
// Customer Dashboard sendiri. Kuota max_devices plan (-1 = unlimited) tetap
// ditegakkan sama persis, cuma accountID dari path (vendor pilih account-nya
// sendiri), bukan dari sesi customer yang login.
func (a *API) handleVendorCreateDevice(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountID")

	var req vendorCreateDeviceRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Name == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "name wajib diisi")
		return
	}

	account, err := a.store.GetAccountByID(r.Context(), accountID)
	if errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	}
	if err != nil {
		slog.Error("vendor: ambil account untuk cek kuota device gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	existing, err := a.store.ListDevices(r.Context(), accountID)
	if err != nil {
		slog.Error("vendor: hitung device untuk cek kuota gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if account.MaxDevices >= 0 && len(existing) >= account.MaxDevices {
		a.writeError(w, http.StatusConflict, "device_limit_reached",
			fmt.Sprintf("plan %s dibatasi %d device -- ganti plan dulu di halaman Accounts", account.Plan, account.MaxDevices))
		return
	}

	deviceID, err := randomDeviceID()
	if err != nil {
		slog.Error("vendor: generate device id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		slog.Error("vendor: acak device secret gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	secretB64 := base64.StdEncoding.EncodeToString(secret)

	if err := a.store.CreateDevice(r.Context(), a.encKey, accountID, deviceID, req.Name, []byte(secretB64)); err != nil {
		slog.Error("vendor: create device gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "DEVICE_CREATED", accountID,
		map[string]string{"device_id": deviceID, "name": req.Name}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	// Device secret ditampilkan sekali di sini, sama seperti alur swalayan
	// customer -- vendor yang membuatkannya wajib meneruskan device_id +
	// device_secret ini ke customer lewat kanal sendiri (WA/telepon), tidak
	// pernah bisa diambil ulang dari sini.
	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true, "device_id": deviceID, "device_secret": secretB64,
	})
}

// handleVendorDeleteDevice adalah versi vendor dari handleAdminDeleteDevice
// -- ditolak 409 dengan pesan yang sama kalau device sudah punya riwayat
// event (store.ErrDeviceHasEvents), vendor harus menonaktifkan saja lewat
// handleVendorSetDeviceEnabled untuk device yang sudah pernah dipakai.
func (a *API) handleVendorDeleteDevice(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountID")
	deviceID := r.PathValue("deviceID")

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
		slog.Error("vendor: delete device gagal", "device_id", deviceID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "DEVICE_DELETED", accountID,
		map[string]string{"device_id": deviceID}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// handleVendorSetDeviceEnabled adalah versi vendor dari
// handleAdminSetDeviceEnabled -- diperlukan karena device yang sudah punya
// riwayat event tidak bisa dihapus sama sekali (lihat komentar
// store.DeleteDevice), jadi nonaktifkan adalah satu-satunya cara vendor
// "mencabut" device semacam itu (HP hilang/rusak/diganti) tanpa SSH.
func (a *API) handleVendorSetDeviceEnabled(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountID")
	deviceID := r.PathValue("deviceID")

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
		slog.Error("vendor: set device enabled gagal", "device_id", deviceID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	action := "DEVICE_DISABLED"
	if req.Enabled {
		action = "DEVICE_ENABLED"
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, action, accountID,
		map[string]string{"device_id": deviceID}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
