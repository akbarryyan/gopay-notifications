package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type changeVendorPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// handleVendorChangePassword adalah swalayan ganti password vendor sendiri
// -- sebelumnya SATU-SATUNYA cara mengubah password vendor adalah
// `go run ./cmd/admintool` di server, yang berarti Akbar butuh akses
// terminal cuma untuk ganti password sendiri. Password saat ini wajib
// diverifikasi dulu -- sesi yang dicuri (cookie bocor) tidak boleh cukup
// untuk mengunci pemilik akun sungguhan keluar selamanya cuma dengan
// mengganti password tanpa tahu apa-apa.
func (a *API) handleVendorChangePassword(w http.ResponseWriter, r *http.Request) {
	username, ok := VendorFromContext(r.Context())
	if !ok || username == "" {
		a.writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return
	}

	var req changeVendorPasswordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if len(req.NewPassword) < minPasswordLen {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "password baru minimal 8 karakter")
		return
	}

	admin, err := a.store.GetVendorAdminByUsername(r.Context(), username)
	if err != nil {
		slog.Error("ambil vendor admin gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if !admin.VerifyPassword(req.CurrentPassword) {
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "password saat ini salah")
		return
	}

	if err := a.store.UpsertVendorAdmin(r.Context(), username, req.NewPassword); err != nil {
		slog.Error("ubah password vendor gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), username, "VENDOR_PASSWORD_CHANGED", username, nil); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
