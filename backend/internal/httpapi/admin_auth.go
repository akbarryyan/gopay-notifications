package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

const adminSessionCookie = "admin_session"

type accountCtxKey int

const ctxKeyAccountID accountCtxKey = iota

// AccountFromContext melaporkan account_id pemilik request ini -- sudah
// lolos requireAdmin (sesi dashboard) ATAU requireAPIKey/requireDevice
// (lewat context yang sama, lihat apikey_auth.go/auth_middleware.go).
// Menggantikan AdminFromContext(ctx) bool yang cuma melaporkan "sudah
// login atau belum" -- di model multi-tenant, sekadar tahu "ada sesi
// valid" tidak cukup, setiap query wajib tahu AKUN MANA.
func AccountFromContext(ctx context.Context) (accountID string, ok bool) {
	v, ok := ctx.Value(ctxKeyAccountID).(string)
	return v, ok
}

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleAdminLogin memverifikasi username/password lalu menerbitkan cookie sesi.
//
// Satu instalasi self-hosted melayani satu merchant, sehingga satu akun admin
// sudah cukup — tidak ada manajemen banyak pengguna di MVP ini.
func (a *API) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.loginThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts",
			"terlalu banyak percobaan login, coba lagi nanti")
		return
	}

	var req adminLoginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Username == "" || req.Password == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "username dan password wajib diisi")
		return
	}

	acc, err := a.store.GetAccountByUsername(r.Context(), req.Username)
	if errors.Is(err, store.ErrAccountNotFound) {
		// Sengaja disamakan dengan password salah: username yang tidak
		// terdaftar tidak boleh dapat dibedakan lewat pesan error.
		a.loginThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "username atau password salah")
		return
	}
	if err != nil {
		slog.Error("ambil account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	if !acc.VerifyPassword(req.Password) {
		a.loginThrottle.RecordFailure(ip, a.now())
		a.logActivity(r, acc.ID, store.ActivityLoginFailed, nil)
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "username atau password salah")
		return
	}
	a.loginThrottle.RecordSuccess(ip)

	a.setAdminSessionCookie(w, r, acc.ID)
	a.logActivity(r, acc.ID, store.ActivityLoginSuccess, nil)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// setAdminSessionCookie menerbitkan sesi baru -- dipakai login, signup,
// dan ganti password (supaya yang mengganti tetap login sementara sesi
// lain dicabut).
func (a *API) setAdminSessionCookie(w http.ResponseWriter, r *http.Request, accountID string) {
	token := auth.NewSessionToken(a.adminSessionKey, a.now(), accountID)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.SessionDuration.Seconds()),
	})
}

// handleAdminLogout menghapus cookie sesi.
func (a *API) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// requireAdmin memverifikasi cookie sesi sebelum handler dijalankan.
func (a *API) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(adminSessionCookie)
		if err != nil || cookie.Value == "" {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak ditemukan")
			return
		}
		accountID, ok := auth.VerifySessionToken(a.adminSessionKey, cookie.Value, a.now())
		if !ok {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak valid atau kedaluwarsa")
			return
		}

		// Sesi stateless tidak bisa dicabut satu per satu, jadi setelah
		// password diganti/di-reset, setiap token yang terbit SEBELUM itu
		// ditolak -- termasuk sesi yang mungkin dicuri, yang justru jadi
		// alasan orang mengganti password. Dibandingkan per detik karena
		// waktu terbit token cuma presisi detik.
		if acc, err := a.store.GetAccountByID(r.Context(), accountID); err == nil && acc.PasswordChangedAt != nil {
			issuedAt, _ := auth.SessionIssuedAt(cookie.Value)
			if issuedAt.Before(acc.PasswordChangedAt.Truncate(time.Second)) {
				a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi berakhir karena password diganti")
				return
			}
		} else if err != nil && !errors.Is(err, store.ErrAccountNotFound) {
			slog.Error("cek sesi account gagal", "err", err)
			a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyAccountID, accountID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// clientIP mengambil alamat IP pemanggil, memperhitungkan reverse proxy Caddy.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isSecureRequest melaporkan apakah koneksi ini (atau koneksi asli di depan
// reverse proxy) memakai HTTPS. Caddy meneruskan TLS lewat X-Forwarded-Proto,
// jadi backend sendiri boleh menerima koneksi HTTP polos dari Caddy di
// jaringan lokal tanpa kehilangan flag Secure pada cookie milik browser.
func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
