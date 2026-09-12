package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type ctxKey int

const (
	ctxKeyDevice ctxKey = iota
	ctxKeyRawBody
)

// maxBodyBytes membatasi ukuran body yang dibaca sebelum verifikasi.
const maxBodyBytes = 64 << 10 // 64 KiB

// DeviceFromContext mengambil device yang sudah terautentikasi.
func DeviceFromContext(ctx context.Context) (store.Device, bool) {
	d, ok := ctx.Value(ctxKeyDevice).(store.Device)
	return d, ok
}

// RawBodyFromContext mengambil byte body yang sudah diverifikasi.
func RawBodyFromContext(ctx context.Context) ([]byte, bool) {
	b, ok := ctx.Value(ctxKeyRawBody).([]byte)
	return b, ok
}

// requireDevice memverifikasi tanda tangan HMAC sebelum handler dijalankan.
//
// Body dibaca sebagai byte mentah dan diverifikasi lebih dulu, baru
// dikembalikan ke r.Body agar handler dapat men-decode-nya. Membalik urutan
// ini — decode lalu re-encode untuk verifikasi — membuat tanda tangan gagal
// secara acak karena urutan field dan spasi berubah.
func (a *API) requireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.Header.Get("X-Device-Id")
		tsRaw := r.Header.Get("X-Timestamp")
		sig := r.Header.Get("X-Signature")

		if deviceID == "" || tsRaw == "" || sig == "" {
			a.writeError(w, http.StatusUnauthorized, "invalid_signature",
				"header X-Device-Id, X-Timestamp, dan X-Signature wajib ada")
			return
		}

		ts, err := strconv.ParseInt(tsRaw, 10, 64)
		if err != nil {
			a.writeError(w, http.StatusUnauthorized, "invalid_signature",
				"X-Timestamp harus Unix epoch dalam detik")
			return
		}

		// Skew diperiksa sebelum tanda tangan, agar jam yang meleset
		// dilaporkan apa adanya alih-alih tersamar jadi invalid_signature.
		if !auth.CheckSkew(ts, a.now()) {
			a.writeError(w, http.StatusUnauthorized, "clock_skew",
				"selisih jam perangkat dan server melebihi 5 menit")
			return
		}

		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "body tidak dapat dibaca")
			return
		}

		device, err := a.store.GetDevice(r.Context(), a.encKey, deviceID)
		if errors.Is(err, store.ErrDeviceNotFound) {
			// Sengaja dilaporkan sebagai invalid_signature: device yang tidak
			// terdaftar tidak boleh dapat dibedakan dari secret yang salah.
			a.writeError(w, http.StatusUnauthorized, "invalid_signature", "autentikasi gagal")
			return
		}
		if err != nil {
			slog.Error("ambil device gagal", "device_id", deviceID, "err", err)
			a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
			return
		}

		if !auth.Verify(device.Secret, auth.SigningString(deviceID, ts, raw), sig) {
			a.writeError(w, http.StatusUnauthorized, "invalid_signature", "autentikasi gagal")
			return
		}

		// Diperiksa setelah verifikasi tanda tangan, agar keberadaan sebuah
		// device tidak dapat diendus tanpa memegang secret-nya.
		if !device.Enabled {
			a.writeError(w, http.StatusForbidden, "device_disabled", "device dinonaktifkan")
			return
		}

		// Versi aplikasi dicatat dari header. Aplikasi lama tidak
		// mengirimnya, dan itu bukan alasan menolak request.
		appVersion := r.Header.Get("X-App-Version")
		if err := a.store.TouchDevice(r.Context(), deviceID, appVersion); err != nil {
			// Bukan alasan menolak request — cukup dicatat.
			slog.Warn("perbarui last_seen_at gagal", "device_id", deviceID, "err", err)
		}

		// device dimuat SEBELUM TouchDevice, jadi salinan di memori masih
		// memuat versi lama. Tanpa penyelarasan ini, handler melaporkan
		// keadaan yang sudah tidak benar pada request yang sama.
		if appVersion != "" {
			appVersion := appVersion
			device.AppVersion = &appVersion
		}

		ctx := context.WithValue(r.Context(), ctxKeyDevice, device)
		ctx = context.WithValue(ctx, ctxKeyRawBody, raw)
		r = r.WithContext(ctx)
		r.Body = io.NopCloser(bytes.NewReader(raw))

		next.ServeHTTP(w, r)
	})
}
