package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// maxQRISImageBodyBytes -- BUKAN maxBodyBytes (64KiB) yang dipakai hampir
// semua endpoint lain di auth_middleware.go. Base64 dari gambar 300KB
// (maxQRISImageDecodedBytes) jadi ~400KB, sudah melebihi limit global itu.
const maxQRISImageBodyBytes = 512 << 10 // 512 KiB

// maxQRISImageDecodedBytes membatasi ukuran HASIL decode base64, bukan
// ukuran body request.
const maxQRISImageDecodedBytes = 300 * 1024 // 300 KB

type uploadQRISImageRequest struct {
	ImageBase64 string `json:"image_base64"`
	// ContentType dari klien HANYA dipakai sebagai Content-Type default saat
	// GET -- validasi tipe sesungguhnya selalu dari http.DetectContentType
	// atas isi byte, tidak pernah mempercayai field ini.
	ContentType string `json:"content_type"`
}

// handleAdminUploadQRISImage adalah swalayan upload QRIS statis dari
// Customer Dashboard (Settings) -- lihat spec
// docs/superpowers/specs/2026-09-15-account-qris-image-design.md. Gambar
// ini yang nanti disertakan ke integrator lewat response POST /invoices.
func (a *API) handleAdminUploadQRISImage(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	var req uploadQRISImageRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxQRISImageBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.ImageBase64 == "" || req.ContentType == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "image_base64 dan content_type wajib diisi")
		return
	}

	data, err := base64.StdEncoding.DecodeString(req.ImageBase64)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "image_base64 tidak valid")
		return
	}
	if len(data) > maxQRISImageDecodedBytes {
		a.writeError(w, http.StatusBadRequest, "image_too_large", "gambar maksimal 300KB")
		return
	}

	detected := http.DetectContentType(data)
	if detected != "image/png" && detected != "image/jpeg" {
		a.writeError(w, http.StatusBadRequest, "unsupported_image_type", "gambar harus PNG atau JPEG")
		return
	}

	if err := a.store.UpsertQRISImage(r.Context(), accountID, data, detected); err != nil {
		slog.Error("upload qris image gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	a.logActivity(r, accountID, store.ActivityQRISImageUpdated, nil)

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// handleAdminGetQRISImage mengembalikan gambar mentah (bukan JSON) --
// dipakai langsung sebagai <img src="..."> di Settings, tanpa perlu
// decode base64 di frontend.
func (a *API) handleAdminGetQRISImage(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	img, err := a.store.GetQRISImage(r.Context(), accountID)
	if errors.Is(err, store.ErrQRISImageNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "belum ada gambar QRIS")
		return
	}
	if err != nil {
		slog.Error("ambil qris image gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	w.Header().Set("Content-Type", img.ContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(img.ImageData)
}

// handleAdminDeleteQRISImage idempotent -- menghapus yang sudah tidak ada
// tetap 200, sama pola dengan RevokeAPIKey.
func (a *API) handleAdminDeleteQRISImage(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

	if err := a.store.DeleteQRISImage(r.Context(), accountID); err != nil {
		slog.Error("hapus qris image gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	a.logActivity(r, accountID, store.ActivityQRISImageRemoved, nil)

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
