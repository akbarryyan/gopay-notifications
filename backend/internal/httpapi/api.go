// Package httpapi berisi seluruh handler HTTP layanan ingestion.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type API struct {
	store             *store.Store
	encKey            []byte
	adminSessionKey   []byte
	webhookSecretKey  []byte
	license           licensecheck.License
	now               func() time.Time
	loginThrottle     *loginThrottle
	webhookHTTPClient *http.Client
}

// New membuat API. Parameter now disuntikkan agar test dapat memalsukan jam.
// license dimuat sekali saat start (lihat cmd/server/main.go) dan disimpan
// di memori — mengganti file lisensi butuh restart proses, sama seperti
// mengganti kunci-kunci lain di atas. Lihat
// docs/superpowers/specs/2026-09-13-license-system-design.md.
func New(s *store.Store, encKey []byte, adminSessionKey []byte, webhookSecretKey []byte,
	lic licensecheck.License, now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return &API{
		store:            s,
		encKey:           encKey,
		adminSessionKey:  adminSessionKey,
		webhookSecretKey: webhookSecretKey,
		license:          lic,
		now:              now,
		loginThrottle:    newLoginThrottle(),
		// Timeout 10 detik sesuai spec §3.1 — server merchant yang lambat
		// tidak boleh menahan worker webhook lebih lama dari itu.
		webhookHTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
	mux.Handle("GET /api/v1/device/me", a.requireLicense(a.requireDevice(http.HandlerFunc(a.handleDeviceMe))))
	mux.Handle("POST /api/v1/devices/heartbeat",
		a.requireLicense(a.requireDevice(http.HandlerFunc(a.handleHeartbeat))))
	// Nama connector TIDAK ada di URL. Payload-nya identik untuk setiap
	// sumber dan pembedanya hanya field "source", jadi menambah DANA atau
	// OVO tidak boleh berarti menambah rute.
	mux.Handle("POST /api/v1/events", a.requireLicense(a.requireDevice(http.HandlerFunc(a.handleCallback))))
	mux.HandleFunc("GET /api/v1/events", a.handleEvents)
	mux.HandleFunc("GET /api/v1/sources", a.handleSources)

	// Invoice: dipanggil server website merchant, bukan browser. Auth API
	// key (Authorization: Bearer), terpisah dari HMAC device dan cookie
	// admin — lihat docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md.
	mux.Handle("POST /api/v1/invoices", a.requireLicense(a.requireAPIKey(http.HandlerFunc(a.handleCreateInvoice))))
	mux.Handle("GET /api/v1/invoices/{invoiceID}",
		a.requireLicense(a.requireAPIKey(http.HandlerFunc(a.handleGetInvoice))))

	// Dashboard admin. Autentikasi lewat cookie sesi, terpisah dari basic
	// auth Caddy yang melindungi GET /api/v1/events di atas — browser yang
	// memanggil fetch() dengan cookie tidak cocok dengan prompt basic auth.
	//
	// login, logout, dan license SENGAJA tidak dibungkus requireLicense —
	// admin harus selalu bisa login dan melihat status lisensinya sendiri
	// walau lisensi sedang tidak aktif. Lihat
	// docs/superpowers/specs/2026-09-13-license-system-design.md.
	mux.HandleFunc("POST /api/v1/admin/login", a.handleAdminLogin)
	mux.HandleFunc("POST /api/v1/admin/logout", a.handleAdminLogout)
	mux.Handle("GET /api/v1/admin/license", a.requireAdmin(http.HandlerFunc(a.handleAdminLicense)))
	mux.Handle("GET /api/v1/admin/overview",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminOverview))))
	mux.Handle("GET /api/v1/admin/devices",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminDevices))))
	mux.Handle("PATCH /api/v1/admin/devices/{deviceID}",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminSetDeviceEnabled))))
	mux.Handle("GET /api/v1/admin/events", a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleEvents))))
	mux.Handle("GET /api/v1/admin/invoices",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminInvoices))))
	mux.Handle("POST /api/v1/admin/api-keys",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminCreateAPIKey))))
	mux.Handle("GET /api/v1/admin/api-keys",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminListAPIKeys))))
	mux.Handle("PATCH /api/v1/admin/api-keys/{keyID}",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminRevokeAPIKey))))

	// Webhook: dashboard-only (requireAdmin) — merchant tidak pernah
	// memanggil rute ini sendiri, beda dari /invoices. Lihat
	// docs/superpowers/specs/2026-09-13-webhook-delivery-design.md.
	mux.Handle("POST /api/v1/admin/webhooks",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminCreateWebhook))))
	mux.Handle("GET /api/v1/admin/webhooks",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminListWebhooks))))
	mux.Handle("PATCH /api/v1/admin/webhooks/{webhookID}",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminSetWebhookEnabled))))
	mux.Handle("DELETE /api/v1/admin/webhooks/{webhookID}",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminDeleteWebhook))))
	mux.Handle("POST /api/v1/admin/webhooks/{webhookID}/test",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminTestWebhook))))
	mux.Handle("GET /api/v1/admin/webhooks/{webhookID}/deliveries",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminWebhookDeliveries))))

	// Konsol pengecualian: sub-project 3 fase 4. Lihat
	// docs/superpowers/specs/2026-09-13-exception-console-design.md.
	mux.Handle("GET /api/v1/admin/exceptions",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminExceptions))))
	mux.Handle("POST /api/v1/admin/exceptions/{eventID}/match",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminMatchException))))
	mux.Handle("POST /api/v1/admin/exceptions/{eventID}/dismiss",
		a.requireLicense(a.requireAdmin(http.HandlerFunc(a.handleAdminDismissException))))

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
