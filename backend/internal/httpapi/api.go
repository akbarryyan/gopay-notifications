// Package httpapi berisi seluruh handler HTTP layanan ingestion.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type API struct {
	store           *store.Store
	encKey          []byte
	adminSessionKey []byte
	now             func() time.Time
	loginThrottle   *loginThrottle
}

// New membuat API. Parameter now disuntikkan agar test dapat memalsukan jam.
func New(s *store.Store, encKey []byte, adminSessionKey []byte, now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return &API{
		store:           s,
		encKey:          encKey,
		adminSessionKey: adminSessionKey,
		now:             now,
		loginThrottle:   newLoginThrottle(),
	}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
	mux.Handle("GET /api/v1/device/me", a.requireDevice(http.HandlerFunc(a.handleDeviceMe)))
	mux.Handle("POST /api/v1/devices/heartbeat", a.requireDevice(http.HandlerFunc(a.handleHeartbeat)))
	// Nama connector TIDAK ada di URL. Payload-nya identik untuk setiap
	// sumber dan pembedanya hanya field "source", jadi menambah DANA atau
	// OVO tidak boleh berarti menambah rute.
	mux.Handle("POST /api/v1/events", a.requireDevice(http.HandlerFunc(a.handleCallback)))
	mux.HandleFunc("GET /api/v1/events", a.handleEvents)
	mux.HandleFunc("GET /api/v1/sources", a.handleSources)

	// Dashboard admin. Autentikasi lewat cookie sesi, terpisah dari basic
	// auth Caddy yang melindungi GET /api/v1/events di atas — browser yang
	// memanggil fetch() dengan cookie tidak cocok dengan prompt basic auth.
	mux.HandleFunc("POST /api/v1/admin/login", a.handleAdminLogin)
	mux.HandleFunc("POST /api/v1/admin/logout", a.handleAdminLogout)
	mux.Handle("GET /api/v1/admin/overview", a.requireAdmin(http.HandlerFunc(a.handleAdminOverview)))
	mux.Handle("GET /api/v1/admin/devices", a.requireAdmin(http.HandlerFunc(a.handleAdminDevices)))
	mux.Handle("PATCH /api/v1/admin/devices/{deviceID}",
		a.requireAdmin(http.HandlerFunc(a.handleAdminSetDeviceEnabled)))
	mux.Handle("GET /api/v1/admin/events", a.requireAdmin(http.HandlerFunc(a.handleEvents)))

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("tulis response gagal", "err", err)
	}
}

// errorResponse adalah bentuk baku response gagal, sesuai api-contract.md §4.5.
type errorResponse struct {
	Success    bool   `json:"success"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	ServerTime int64  `json:"server_time,omitempty"`
}

func (a *API) writeError(w http.ResponseWriter, status int, code, msg string) {
	resp := errorResponse{Success: false, Error: code, Message: msg}
	if code == "clock_skew" {
		resp.ServerTime = a.now().Unix()
	}
	writeJSON(w, status, resp)
}
