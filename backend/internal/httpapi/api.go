// Package httpapi berisi seluruh handler HTTP layanan ingestion.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
	"github.com/akbarryyan/gopay-notifications/backend/internal/telegram"
)

type API struct {
	store               *store.Store
	encKey              []byte
	adminSessionKey     []byte
	webhookSecretKey    []byte
	vendorSessionKey    []byte
	settingsSecretKey   []byte
	now                 func() time.Time
	loginThrottle       *loginThrottle
	vendorLoginThrottle *loginThrottle
	signupThrottle      *loginThrottle
	// passwordResetThrottle membatasi permintaan email reset per IP;
	// passwordChangeThrottle membatasi tebakan password saat ini per account
	// (sesi yang dicuri tidak boleh bisa menebak password tanpa batas).
	passwordResetThrottle  *loginThrottle
	passwordChangeThrottle *loginThrottle
	webhookHTTPClient      *http.Client
	dashboardURL           string
	// telegramBaseURL bisa diganti di test (httptest) supaya tidak pernah
	// menyentuh api.telegram.org sungguhan.
	telegramBaseURL string
	botUsernames    *botUsernameCache
	// background menjalankan pengiriman email di luar request. Lihat
	// handleForgotPassword untuk alasannya.
	background func(func())
}

// New membuat API. Parameter now disuntikkan agar test dapat memalsukan jam.
func New(s *store.Store, encKey []byte, adminSessionKey []byte, webhookSecretKey []byte,
	vendorSessionKey []byte, settingsSecretKey []byte, now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return &API{
		store:                  s,
		encKey:                 encKey,
		adminSessionKey:        adminSessionKey,
		webhookSecretKey:       webhookSecretKey,
		vendorSessionKey:       vendorSessionKey,
		settingsSecretKey:      settingsSecretKey,
		now:                    now,
		loginThrottle:          newLoginThrottle(),
		vendorLoginThrottle:    newLoginThrottle(),
		signupThrottle:         newLoginThrottle(),
		passwordResetThrottle:  newLoginThrottle(),
		passwordChangeThrottle: newLoginThrottle(),
		background:             func(f func()) { go f() },
		telegramBaseURL:        telegram.DefaultBaseURL,
		botUsernames:           &botUsernameCache{byToken: map[string]string{}},
		// Timeout 10 detik sesuai spec §3.1 — server merchant yang lambat
		// tidak boleh menahan worker webhook lebih lama dari itu.
		webhookHTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// WithDashboardURL mengisi alamat publik Customer Dashboard untuk link di
// email reset password. Tanpa ini, lupa password menjawab "belum tersedia".
func (a *API) WithDashboardURL(u string) *API {
	a.dashboardURL = u
	return a
}

// WithTelegramBaseURL dipakai test untuk mengarahkan panggilan Bot API ke
// server palsu.
func (a *API) WithTelegramBaseURL(u string) *API {
	a.telegramBaseURL = u
	return a
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
	mux.Handle("GET /api/v1/device/me", a.requireDevice(a.requireActiveAccount(http.HandlerFunc(a.handleDeviceMe))))
	mux.Handle("POST /api/v1/devices/heartbeat",
		a.requireDevice(a.requireActiveAccount(http.HandlerFunc(a.handleHeartbeat))))
	// Nama connector TIDAK ada di URL. Payload-nya identik untuk setiap
	// sumber dan pembedanya hanya field "source", jadi menambah DANA atau
	// OVO tidak boleh berarti menambah rute.
	mux.Handle("POST /api/v1/events", a.requireDevice(a.requireActiveAccount(http.HandlerFunc(a.handleCallback))))
	mux.HandleFunc("GET /api/v1/sources", a.handleSources)
	// Publik juga, sama alasannya dengan /sources -- section Harga landing
	// page perlu ini sebelum orang login/daftar apa pun.
	mux.HandleFunc("GET /api/v1/pricing-plans", a.handlePublicPricingPlans)

	// Invoice: dipanggil server website merchant, bukan browser. Auth API
	// key (Authorization: Bearer), terpisah dari HMAC device dan cookie
	// admin — lihat docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md.
	mux.Handle("POST /api/v1/invoices", a.requireAPIKey(a.requireActiveAccount(http.HandlerFunc(a.handleCreateInvoice))))
	mux.Handle("GET /api/v1/invoices/{invoiceID}",
		a.requireAPIKey(a.requireActiveAccount(http.HandlerFunc(a.handleGetInvoice))))

	// Dashboard admin. Autentikasi lewat cookie sesi.
	//
	// login, logout, dan license SENGAJA tidak dibungkus requireActiveAccount
	// — admin harus selalu bisa login dan melihat status akunnya sendiri
	// walau akun sedang tidak aktif.
	mux.HandleFunc("POST /api/v1/admin/login", a.handleAdminLogin)
	mux.HandleFunc("POST /api/v1/admin/logout", a.handleAdminLogout)
	// Signup: SATU-SATUNYA endpoint di seluruh backend tanpa auth apa pun
	// (bukan sesi, API key, atau HMAC) -- calon customer belum punya
	// kredensial sampai titik ini. Lihat signup.go.
	mux.HandleFunc("POST /api/v1/signup", a.handleSignup)
	// Lupa password: juga tanpa auth, dengan alasan yang sama. Lihat
	// account_password.go.
	mux.HandleFunc("POST /api/v1/password/forgot", a.handleForgotPassword)
	mux.HandleFunc("POST /api/v1/password/reset", a.handleResetPassword)
	// Verifikasi email: publik juga, dengan alasan yang sama -- link dibuka
	// dari email, bukan dari sesi dashboard yang sedang aktif.
	mux.HandleFunc("POST /api/v1/email/verify", a.handleVerifyEmail)
	mux.Handle("GET /api/v1/admin/license", a.requireAdmin(http.HandlerFunc(a.handleAdminLicense)))
	// Settings akun: SENGAJA tanpa requireActiveAccount, sama seperti
	// /license -- account yang kedaluwarsa tetap harus bisa mengganti
	// password dan memperbarui kontaknya.
	mux.Handle("GET /api/v1/admin/account", a.requireAdmin(http.HandlerFunc(a.handleAdminGetAccount)))
	mux.Handle("PATCH /api/v1/admin/account", a.requireAdmin(http.HandlerFunc(a.handleAdminUpdateAccount)))
	mux.Handle("POST /api/v1/admin/account/password", a.requireAdmin(http.HandlerFunc(a.handleAdminChangePassword)))
	mux.Handle("POST /api/v1/admin/account/email/resend", a.requireAdmin(http.HandlerFunc(a.handleAdminResendVerificationEmail)))
	mux.Handle("GET /api/v1/admin/activity", a.requireAdmin(http.HandlerFunc(a.handleAdminActivityLog)))
	mux.Handle("POST /api/v1/admin/account/telegram", a.requireAdmin(http.HandlerFunc(a.handleAdminSetTelegram)))
	mux.Handle("POST /api/v1/admin/account/telegram/link", a.requireAdmin(http.HandlerFunc(a.handleAdminTelegramLink)))
	mux.Handle("GET /api/v1/admin/overview",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminOverview))))
	mux.Handle("GET /api/v1/admin/devices",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminDevices))))
	mux.Handle("POST /api/v1/admin/devices",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminCreateDevice))))
	mux.Handle("PATCH /api/v1/admin/devices/{deviceID}",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminSetDeviceEnabled))))
	mux.Handle("DELETE /api/v1/admin/devices/{deviceID}",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminDeleteDevice))))
	mux.Handle("GET /api/v1/admin/events", a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleEvents))))
	mux.Handle("GET /api/v1/admin/invoices",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminInvoices))))
	mux.Handle("POST /api/v1/admin/api-keys",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminCreateAPIKey))))
	mux.Handle("GET /api/v1/admin/api-keys",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminListAPIKeys))))
	mux.Handle("PATCH /api/v1/admin/api-keys/{keyID}",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminRevokeAPIKey))))

	// Webhook: dashboard-only (requireAdmin) — merchant tidak pernah
	// memanggil rute ini sendiri, beda dari /invoices. Lihat
	// docs/superpowers/specs/2026-09-13-webhook-delivery-design.md.
	mux.Handle("POST /api/v1/admin/webhooks",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminCreateWebhook))))
	mux.Handle("GET /api/v1/admin/webhooks",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminListWebhooks))))
	mux.Handle("PATCH /api/v1/admin/webhooks/{webhookID}",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminSetWebhookEnabled))))
	mux.Handle("DELETE /api/v1/admin/webhooks/{webhookID}",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminDeleteWebhook))))
	mux.Handle("POST /api/v1/admin/webhooks/{webhookID}/test",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminTestWebhook))))
	mux.Handle("GET /api/v1/admin/webhooks/{webhookID}/deliveries",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminWebhookDeliveries))))

	// Konsol pengecualian: sub-project 3 fase 4. Lihat
	// docs/superpowers/specs/2026-09-13-exception-console-design.md.
	mux.Handle("GET /api/v1/admin/exceptions",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminExceptions))))
	mux.Handle("POST /api/v1/admin/exceptions/{eventID}/match",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminMatchException))))
	mux.Handle("POST /api/v1/admin/exceptions/{eventID}/dismiss",
		a.requireAdmin(a.requireActiveAccount(http.HandlerFunc(a.handleAdminDismissException))))

	// Vendor Dashboard: superadmin (Akbar), sesi terpisah total dari
	// admin_session (customer) -- vendor_session, tabel vendor_admins
	// sendiri. Menggantikan License Server yang dulu terpisah service.
	mux.HandleFunc("POST /api/v1/vendor/login", a.handleVendorLogin)
	mux.HandleFunc("POST /api/v1/vendor/logout", a.handleVendorLogout)
	mux.Handle("GET /api/v1/vendor/overview", a.requireVendor(http.HandlerFunc(a.handleVendorOverview)))
	mux.Handle("POST /api/v1/vendor/accounts", a.requireVendor(http.HandlerFunc(a.handleVendorCreateAccount)))
	mux.Handle("GET /api/v1/vendor/accounts", a.requireVendor(http.HandlerFunc(a.handleVendorListAccounts)))
	mux.Handle("GET /api/v1/vendor/accounts/{accountID}", a.requireVendor(http.HandlerFunc(a.handleVendorGetAccount)))
	mux.Handle("GET /api/v1/vendor/accounts/{accountID}/devices", a.requireVendor(http.HandlerFunc(a.handleVendorListDevices)))
	mux.Handle("POST /api/v1/vendor/accounts/{accountID}/devices", a.requireVendor(http.HandlerFunc(a.handleVendorCreateDevice)))
	mux.Handle("PATCH /api/v1/vendor/accounts/{accountID}/devices/{deviceID}", a.requireVendor(http.HandlerFunc(a.handleVendorSetDeviceEnabled)))
	mux.Handle("DELETE /api/v1/vendor/accounts/{accountID}/devices/{deviceID}", a.requireVendor(http.HandlerFunc(a.handleVendorDeleteDevice)))
	mux.Handle("GET /api/v1/vendor/accounts/{accountID}/api-keys", a.requireVendor(http.HandlerFunc(a.handleVendorListAPIKeys)))
	mux.Handle("DELETE /api/v1/vendor/accounts/{accountID}/api-keys/{keyID}", a.requireVendor(http.HandlerFunc(a.handleVendorRevokeAPIKey)))
	mux.Handle("GET /api/v1/vendor/transactions", a.requireVendor(http.HandlerFunc(a.handleVendorTransactions)))
	mux.Handle("GET /api/v1/vendor/webhook-deliveries", a.requireVendor(http.HandlerFunc(a.handleVendorWebhookDeliveries)))
	mux.Handle("GET /api/v1/vendor/notification-log", a.requireVendor(http.HandlerFunc(a.handleVendorNotificationLog)))
	mux.Handle("POST /api/v1/vendor/accounts/{accountID}/renew", a.requireVendor(http.HandlerFunc(a.handleVendorRenewAccount)))
	mux.Handle("POST /api/v1/vendor/accounts/{accountID}/send-password-reset", a.requireVendor(http.HandlerFunc(a.handleVendorSendPasswordReset)))
	mux.Handle("POST /api/v1/vendor/accounts/{accountID}/plan", a.requireVendor(http.HandlerFunc(a.handleVendorChangePlan)))
	mux.Handle("POST /api/v1/vendor/accounts/{accountID}/suspend", a.requireVendor(http.HandlerFunc(a.handleVendorSuspendAccount)))
	mux.Handle("POST /api/v1/vendor/accounts/{accountID}/revoke", a.requireVendor(http.HandlerFunc(a.handleVendorRevokeAccount)))
	mux.Handle("GET /api/v1/vendor/audit-log", a.requireVendor(http.HandlerFunc(a.handleVendorAuditLog)))
	mux.Handle("GET /api/v1/vendor/plans", a.requireVendor(http.HandlerFunc(a.handleVendorListPlans)))
	mux.Handle("POST /api/v1/vendor/plans", a.requireVendor(http.HandlerFunc(a.handleVendorCreatePlan)))
	mux.Handle("PATCH /api/v1/vendor/plans/{planID}", a.requireVendor(http.HandlerFunc(a.handleVendorUpdatePlan)))
	mux.Handle("DELETE /api/v1/vendor/plans/{planID}", a.requireVendor(http.HandlerFunc(a.handleVendorDeletePlan)))
	mux.Handle("POST /api/v1/vendor/plans/{planID}/move", a.requireVendor(http.HandlerFunc(a.handleVendorMovePlan)))
	mux.Handle("GET /api/v1/vendor/me", a.requireVendor(http.HandlerFunc(a.handleVendorMe)))
	mux.Handle("POST /api/v1/vendor/me/password", a.requireVendor(http.HandlerFunc(a.handleVendorChangePassword)))
	mux.Handle("GET /api/v1/vendor/settings/notifications", a.requireVendor(http.HandlerFunc(a.handleVendorGetNotificationSettings)))
	mux.Handle("PUT /api/v1/vendor/settings/notifications", a.requireVendor(http.HandlerFunc(a.handleVendorSaveNotificationSettings)))
	mux.Handle("POST /api/v1/vendor/settings/notifications/test", a.requireVendor(http.HandlerFunc(a.handleVendorTestNotification)))

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
