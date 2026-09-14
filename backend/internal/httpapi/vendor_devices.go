package httpapi

import (
	"log/slog"
	"net/http"
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
