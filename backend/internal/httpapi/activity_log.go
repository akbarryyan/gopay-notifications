package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// logActivity mencatat satu baris riwayat aktivitas akun (halaman Logs di
// Customer Dashboard), mengambil IP dan User-Agent dari request yang
// sedang diproses. Gagal mencatat cuma dicatat sebagai log server --
// TIDAK PERNAH menggagalkan aksi yang memicunya (login/ganti password/dsb
// tetap harus berhasil walau baris riwayatnya gagal ditulis).
func (a *API) logActivity(r *http.Request, accountID, action string, metadata map[string]any) {
	ip := clientIP(r)
	ua := r.Header.Get("User-Agent")
	entry := store.ActivityLogEntry{AccountID: accountID, Action: action, Metadata: metadata}
	if ip != "" {
		entry.IPAddress = &ip
	}
	if ua != "" {
		entry.UserAgent = &ua
	}
	if err := a.store.LogActivity(r.Context(), entry); err != nil {
		slog.Error("catat activity log gagal", "account_id", accountID, "action", action, "err", err)
	}
}
