// Package httpapi berisi seluruh handler HTTP License Server: endpoint
// publik /activate dan /validate (dipanggil backend tiap instalasi
// customer) serta endpoint admin untuk Vendor Dashboard.
//
// Lihat docs/superpowers/specs/2026-09-13-online-license-platform-design.md.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

type API struct {
	store             *store.Store
	adminSessionKey   []byte
	signingPrivateKey string
	now               func() time.Time
	loginThrottle     *loginThrottle
	activateThrottle  *loginThrottle
}

// New membuat API. signingPrivateKeyBase64 menandatangani SETIAP local
// license state yang dikirim lewat /activate dan /validate — lihat
// internal/licensecheck.Issue.
func New(s *store.Store, adminSessionKey []byte, signingPrivateKeyBase64 string, now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return &API{
		store:             s,
		adminSessionKey:   adminSessionKey,
		signingPrivateKey: signingPrivateKeyBase64,
		now:               now,
		loginThrottle:     newLoginThrottle(),
		activateThrottle:  newLoginThrottle(),
	}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", a.handleHealth)

	// Dipanggil backend tiap instalasi customer — bukan browser.
	mux.HandleFunc("POST /api/v1/license/activate", a.handleActivate)
	mux.HandleFunc("POST /api/v1/license/validate", a.handleValidate)

	// Vendor Dashboard.
	mux.HandleFunc("POST /api/v1/admin/login", a.handleAdminLogin)
	mux.HandleFunc("POST /api/v1/admin/logout", a.handleAdminLogout)
	mux.Handle("POST /api/v1/admin/customers", a.requireAdmin(http.HandlerFunc(a.handleCreateCustomer)))
	mux.Handle("GET /api/v1/admin/customers", a.requireAdmin(http.HandlerFunc(a.handleListCustomers)))
	mux.Handle("GET /api/v1/admin/customers/{customerID}", a.requireAdmin(http.HandlerFunc(a.handleGetCustomer)))
	mux.Handle("POST /api/v1/admin/customers/{customerID}/licenses",
		a.requireAdmin(http.HandlerFunc(a.handleCreateLicense)))
	mux.Handle("GET /api/v1/admin/licenses/{licenseID}", a.requireAdmin(http.HandlerFunc(a.handleGetLicense)))
	mux.Handle("POST /api/v1/admin/licenses/{licenseID}/renew",
		a.requireAdmin(http.HandlerFunc(a.handleRenewLicense)))
	mux.Handle("POST /api/v1/admin/licenses/{licenseID}/suspend",
		a.requireAdmin(http.HandlerFunc(a.handleSuspendLicense)))
	mux.Handle("POST /api/v1/admin/licenses/{licenseID}/revoke",
		a.requireAdmin(http.HandlerFunc(a.handleRevokeLicense)))
	mux.Handle("POST /api/v1/admin/installations/{installationID}/reset",
		a.requireAdmin(http.HandlerFunc(a.handleResetInstallation)))
	mux.Handle("GET /api/v1/admin/audit-log", a.requireAdmin(http.HandlerFunc(a.handleListAuditLog)))

	return mux
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "server_time": a.now().Unix()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("tulis response gagal", "err", err)
	}
}

type errorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (a *API) writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorResponse{Success: false, Error: code, Message: msg})
}

const maxBodyBytes = 1 << 20 // 1 MiB, sama seperti backend/internal/httpapi
