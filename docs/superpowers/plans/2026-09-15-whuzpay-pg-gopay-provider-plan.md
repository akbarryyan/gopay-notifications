# Provider "gopay" di whuzpay-pg + Pencabutan Cashi Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ganti provider pembayaran whuzpay-pg dari Cashi (satu kredensial global) ke "gopay" (gopay-notifications, kredensial per merchant), hapus Cashi total.

**Architecture:** Perluas interface `PaymentProvider` yang sudah ada dengan parameter `merchantID` di 3 method (`CreatePayment` lewat field baru di request, `GetPaymentStatus`, `ValidateWebhook`), tambah tabel+repository kredensial per merchant (plaintext), adapter baru `internal/provider/gopay/` yang mengimplementasikan interface itu memanggil API gopay-notifications, balik urutan proses webhook (parse dulu baru validasi, supaya secret per-merchant bisa di-resolve), UI + endpoint self-service merchant untuk isi kredensial, cabut Cashi total.

**Tech Stack:** Go 1.25, `database/sql` + `lib/pq` (tanpa ORM), `gorilla/mux`, `google/uuid` (backend `whuzpay-pg/back`); Next.js + React (frontend `whuzpay-pg/front`).

## Global Constraints

- Kredensial (API key + webhook secret gopay-notifications milik tiap merchant) disimpan **polos** (plaintext) — pola sama `merchants.webhook_secret` (migrasi 013). Jangan menambah enkripsi.
- Cashi dihapus **total** — kode adapter, config, env var, referensi dokumentasi, bruno request — bukan sekadar berhenti didaftarkan.
- `merchantID` selalu tersedia di titik pemanggilan (baris `payment` sudah di-load) — tidak perlu query tambahan untuk mendapatkannya.
- Webhook endpoint gopay-notifications yang harus didaftarkan merchant: `{APP_BASE_URL}/api/v1/provider-webhooks/gopay` (path yang sudah ada di whuzpay-pg, generik per nama provider — BUKAN `/webhooks/gopay`).
- Endpoint kredensial merchant: `GET`/`PUT /api/v1/merchant/gopay-credentials`, di bawah `merchantDash` subrouter yang sudah ada (`authMiddleware.RequireMerchant`, sesi JWT merchant — BUKAN API key aggregator).
- `whuzpay-pg/back` tidak punya database test terpisah — seluruh test service-layer pakai fake in-memory (pola `internal/service/payment_fakes_test.go`), bukan Postgres sungguhan. Repository package tidak punya test langsung (konvensi yang sudah ada, jangan diubah).
- Field `UseCustomMerchantName` (fitur "QRIS Custom" Cashi) dihapus total dari `ProviderPaymentRequest` DAN `payment.CreatePaymentRequest` (API publik) — gopay-notifications tidak punya konsep ini.
- Spec: [`docs/superpowers/specs/2026-09-15-whuzpay-pg-gopay-provider-design.md`](../specs/2026-09-15-whuzpay-pg-gopay-provider-design.md).

---

### Task 1: Perluas interface `PaymentProvider`, hapus Cashi total, ijo-kan build

**Files:**
- Modify: `whuzpay-pg/back/internal/provider/adapter.go`
- Modify: `whuzpay-pg/back/internal/domain/provider/provider.go`
- Modify: `whuzpay-pg/back/internal/domain/payment/payment.go`
- Modify: `whuzpay-pg/back/internal/service/payment_service.go`
- Modify: `whuzpay-pg/back/internal/provider/sandbox/adapter.go`
- Modify: `whuzpay-pg/back/internal/config/config.go`
- Modify: `whuzpay-pg/back/.env.example`
- Modify: `whuzpay-pg/back/cmd/api/main.go`
- Modify: `whuzpay-pg/back/cmd/api/router_test.go`
- Modify: `whuzpay-pg/back/internal/handler/fakes_test.go`
- Modify: `whuzpay-pg/back/internal/provider/adapter_test.go`
- Modify: `whuzpay-pg/back/internal/service/payment_fakes_test.go`
- Modify: `whuzpay-pg/back/docs/openapi.yaml`
- Modify: `whuzpay-pg/back/docs/payment-state-machine.md`
- Modify: `whuzpay-pg/back/README.md`
- Delete: `whuzpay-pg/back/internal/provider/cashi/` (seluruh isi: `client.go`, `types.go`, `client_test.go`)
- Delete: `whuzpay-pg/back/bruno/02 - Create Payment (Production, QRIS Custom).bru`

**Interfaces:**
- Produces: `PaymentProvider` interface baru (dipakai Task 2's adapter gopay dan Task 1's sandbox adapter):
  ```go
  type PaymentProvider interface {
      GetName() string
      CreatePayment(ctx context.Context, req *domainProvider.ProviderPaymentRequest) (*domainProvider.ProviderPaymentResponse, error)
      GetPaymentStatus(ctx context.Context, providerReference string, merchantID uuid.UUID) (*domainProvider.NormalizedPaymentStatus, error)
      ValidateWebhook(rawPayload []byte, signature string, merchantID uuid.UUID) error
      ParseWebhook(rawPayload []byte) (*domainProvider.ProviderWebhookPayload, error)
      NormalizeStatus(providerStatus string) string
  }
  ```
  `domainProvider.ProviderPaymentRequest.MerchantID uuid.UUID` (field baru).
  `domainProvider.ProviderGopay = "gopay"` (konstanta baru, `ProviderCashi` dihapus).

- [ ] **Step 1: Ubah interface di `internal/provider/adapter.go`**

Ganti:
```go
type PaymentProvider interface {
	GetName() string

	CreatePayment(ctx context.Context, req *domainProvider.ProviderPaymentRequest) (*domainProvider.ProviderPaymentResponse, error)

	GetPaymentStatus(ctx context.Context, providerReference string) (*domainProvider.NormalizedPaymentStatus, error)

	ValidateWebhook(rawPayload []byte, signature string) error

	ParseWebhook(rawPayload []byte) (*domainProvider.ProviderWebhookPayload, error)

	NormalizeStatus(providerStatus string) string
}
```
jadi:
```go
type PaymentProvider interface {
	GetName() string

	CreatePayment(ctx context.Context, req *domainProvider.ProviderPaymentRequest) (*domainProvider.ProviderPaymentResponse, error)

	// GetPaymentStatus butuh merchantID karena tiap merchant punya
	// kredensial gopay-notifications sendiri (beda dari Cashi yang satu
	// kredensial global) -- adapter yang mengimplementasikan ini mencari
	// kredensial merchant tersebut secara internal.
	GetPaymentStatus(ctx context.Context, providerReference string, merchantID uuid.UUID) (*domainProvider.NormalizedPaymentStatus, error)

	// ValidateWebhook butuh merchantID untuk alasan yang sama --
	// pemanggil (PaymentService.ProcessWebhook) WAJIB memanggil ini
	// SETELAH ParseWebhook + pencarian payment (bukan sebelumnya seperti
	// versi lama), supaya merchantID pemilik payment sudah diketahui.
	ValidateWebhook(rawPayload []byte, signature string, merchantID uuid.UUID) error

	ParseWebhook(rawPayload []byte) (*domainProvider.ProviderWebhookPayload, error)

	NormalizeStatus(providerStatus string) string
}
```

Tambah import `"github.com/google/uuid"` di header file.

- [ ] **Step 2: Ubah `internal/domain/provider/provider.go`**

Ganti blok konstanta:
```go
const (
	ProviderCashi    = "cashi"
	ProviderMidtrans = "midtrans"
	ProviderXendit   = "xendit"
	ProviderDuitku   = "duitku"
)
```
jadi:
```go
const (
	ProviderGopay    = "gopay"
	ProviderMidtrans = "midtrans"
	ProviderXendit   = "xendit"
	ProviderDuitku   = "duitku"
)
```

Ganti `ProviderPaymentRequest`:
```go
type ProviderPaymentRequest struct {
	InternalReference string
	Amount            int64
	Currency          string
	Description       string
	CustomerName      *string
	CustomerEmail     *string
	ExpiresAt         time.Time
	CallbackURL       string
	// UseCustomMerchantName requests that the provider display the
	// merchant's custom name on the payment QR/page instead of its
	// default account name, where supported (e.g. Cashi's QRIS Custom —
	// see docs/cashi-qris-custom.md). Providers that don't support this
	// silently ignore it.
	UseCustomMerchantName bool
}
```
jadi:
```go
type ProviderPaymentRequest struct {
	InternalReference string
	Amount            int64
	Currency          string
	Description       string
	CustomerName      *string
	CustomerEmail     *string
	ExpiresAt         time.Time
	CallbackURL       string
	// MerchantID dipakai adapter yang butuh kredensial per merchant
	// (mis. gopay) untuk mencari API key merchant ini. Adapter dengan
	// kredensial global (tidak ada lagi sejak Cashi dihapus, tapi pola
	// ini dipertahankan untuk provider masa depan) boleh mengabaikannya.
	MerchantID uuid.UUID
}
```
Tambah import `"github.com/google/uuid"`.

- [ ] **Step 3: Hapus `UseCustomMerchantName` dari `internal/domain/payment/payment.go`**

Hapus field ini beserta komentarnya dari `CreatePaymentRequest`:
```go
	// UseCustomMerchantName requests a custom merchant display name on the
	// ...
	UseCustomMerchantName bool `json:"use_custom_merchant_name,omitempty"`
```

- [ ] **Step 4: Perbarui `internal/service/payment_service.go`**

Di percabangan sandbox (sekitar baris 105-115), hapus `UseCustomMerchantName: req.UseCustomMerchantName,` dari `providerReq`, tambah `MerchantID: req.MerchantID,`:
```go
		providerReq := &provider.ProviderPaymentRequest{
			InternalReference: reference,
			Amount:            req.Amount,
			Currency:          req.Currency,
			Description:       req.Description,
			CustomerName:      req.CustomerName,
			CustomerEmail:     req.CustomerEmail,
			ExpiresAt:         expiresAt,
			CallbackURL:       s.buildCallbackURL(s.sandboxProvider.GetName()),
			MerchantID:        req.MerchantID,
		}
```

Di percabangan produksi (loop `candidateProviders`), sama: hapus `UseCustomMerchantName: req.UseCustomMerchantName,`, tambah `MerchantID: req.MerchantID,`. Hapus SELURUH blok `if req.UseCustomMerchantName { ... logger.InfofCtx(...) }` setelah `providerResp, err = selectedProvider.CreatePayment(ctx, providerReq)` — tidak ada lagi provider yang punya perilaku "custom merchant name" untuk dilog.

Di `reconcilePaymentData` (sekitar baris 305), ganti:
```go
	providerStatus, err := selectedProvider.GetPaymentStatus(ctx, *p.ProviderReference)
```
jadi:
```go
	providerStatus, err := selectedProvider.GetPaymentStatus(ctx, *p.ProviderReference, p.MerchantID)
```

- [ ] **Step 5: Perbarui `internal/provider/sandbox/adapter.go`**

Ubah tanda tangan (isi fungsi tidak berubah, cuma parameter baru diabaikan dengan `_`):
```go
func (a *Adapter) GetPaymentStatus(ctx context.Context, providerReference string, _ uuid.UUID) (*provider.NormalizedPaymentStatus, error) {
```
```go
func (a *Adapter) ValidateWebhook(rawPayload []byte, signature string, _ uuid.UUID) error {
```
Tambah import `"github.com/google/uuid"`.

- [ ] **Step 6: Hapus `internal/provider/cashi/` dan `CashiConfig`**

```bash
rm -rf internal/provider/cashi
```

Di `internal/config/config.go`, ganti field `Cashi CashiConfig` di `Config` struct dan tipe `CashiConfig` dengan:
```go
type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	Gopay    GopayConfig
	Security SecurityConfig
}
```
```go
// GopayConfig -- BaseURL saja, tidak ada API key/secret global karena
// kredensial gopay-notifications disimpan per merchant (lihat
// internal/repository/merchant_gopay_credentials_repository.go, Task 2).
type GopayConfig struct {
	BaseURL string
}
```
Di `Load()`, ganti blok `Cashi: CashiConfig{...}` dengan:
```go
		Gopay: GopayConfig{
			BaseURL: getEnv("GOPAY_BASE_URL", "https://whuzpay.com"),
		},
```
Di `Validate()`, hapus seluruh blok:
```go
		if c.Cashi.APIKey == "" || c.Cashi.SecretKey == "" {
			return fmt.Errorf("Cashi credentials are required in production")
		}
```
(baris `if c.Security.JWTSecret == "change-this-secret" { ... }` di dalam `if c.App.Environment == "production"` yang sama TETAP ada, cuma blok Cashi-nya yang hilang).

- [ ] **Step 7: Perbarui `.env.example`**

Ganti blok:
```
# Cashi Provider (real QRIS — used only for environment=production payments)
# Docs: docs/cashi-api.md
# - CASHI_API_KEY   → header x-api-key on create-order / check-status
# - CASHI_SECRET_KEY → webhook HMAC validation only (not sent as merchant id)
# Sandbox merchant dashboard payments use an in-process mock (no Cashi HTTP).
CASHI_BASE_URL=https://cashi.id
CASHI_API_KEY=your_cashi_api_key_here
CASHI_SECRET_KEY=your_cashi_webhook_secret_here
```
jadi:
```
# gopay-notifications provider (real QRIS — used only for environment=production
# payments). Tidak ada API key/secret global di sini -- tiap merchant
# menyimpan kredensialnya sendiri lewat Settings > Provider Pembayaran GoPay
# (lihat merchant_gopay_credentials di database).
# Sandbox merchant dashboard payments use an in-process mock (no gopay HTTP).
GOPAY_BASE_URL=https://whuzpay.com
```

- [ ] **Step 8: Wiring sementara di `cmd/api/main.go`**

Hapus import `"github.com/akbarryyan/pg-aggregator-back/internal/provider/cashi"`. Hapus blok:
```go
	cashiAdapter := cashi.NewCashiAdapter(
		cfg.Cashi.BaseURL,
		cfg.Cashi.APIKey,
		cfg.Cashi.SecretKey,
	)
```
Ganti:
```go
	providerRouter := provider.NewProviderRouter()
	providerRouter.RegisterProvider(cashiAdapter)
	providerRouter.RegisterProvider(sandboxAdapter)
	// Production QRIS routing uses real Cashi only
	providerRouter.RegisterPaymentMethodProvider("qris", cashiAdapter.GetName())
```
jadi (sementara, adapter gopay didaftarkan Task 3):
```go
	providerRouter := provider.NewProviderRouter()
	providerRouter.RegisterProvider(sandboxAdapter)
	// TODO(Task 3): daftarkan gopayAdapter + RegisterPaymentMethodProvider("qris", ...)
```

- [ ] **Step 9: Perbaiki test fakes -- tanda tangan method baru**

Di keempat file ini, ubah tanda tangan `GetPaymentStatus`/`ValidateWebhook` pada tipe fake (`stubProvider` di `router_test.go`, `fakeProvider` di 3 file lainnya) supaya menerima `merchantID uuid.UUID` sebagai parameter tambahan (isi fungsi tidak berubah, parameter baru diabaikan):

`cmd/api/router_test.go`:
```go
func (p *stubProvider) GetPaymentStatus(ctx context.Context, providerReference string, _ uuid.UUID) (*domainProvider.NormalizedPaymentStatus, error) {
```
```go
func (p *stubProvider) ValidateWebhook(rawPayload []byte, signature string, _ uuid.UUID) error { return nil }
```

`internal/handler/fakes_test.go`, `internal/provider/adapter_test.go`, `internal/service/payment_fakes_test.go` — sama pola, ubah `fakeProvider.GetPaymentStatus`/`fakeProvider.ValidateWebhook` masing-masing. Tambah import `"github.com/google/uuid"` di file yang belum mengimpornya.

- [ ] **Step 10: `go build`, `go vet`, jalankan seluruh test**

```bash
cd whuzpay-pg/back
go build ./...
go vet ./...
go test ./...
```
Expected: build bersih, seluruh test lulus (tidak ada test yang menguji perilaku Cashi-spesifik selain yang baru dihapus di Step 6).

- [ ] **Step 11: Bersihkan dokumentasi dan bruno**

```bash
rm "bruno/02 - Create Payment (Production, QRIS Custom).bru"
```
Di `bruno/03 - Create Payment (Sandbox, comparison).bru`, hapus baris `"use_custom_merchant_name": true` dari body request dan kalimat yang menjelaskan perilaku itu di komentar/docs field-nya (field ini sudah tidak ada di API).

Di `docs/openapi.yaml`, hapus field `use_custom_merchant_name` dari schema `CreatePaymentRequest`, dan ganti kalimat pembuka (`payment providers (Cashi in production, an in-process mock for ...`) serta ringkasan endpoint webhook (`Inbound webhook from a payment provider (e.g. Cashi)`) supaya menyebut "gopay", bukan Cashi.

Di `docs/payment-state-machine.md`, ganti `Provider (mis. Cashi) POST ke ...` jadi `Provider (mis. gopay) POST ke ...`.

Di `README.md`: baris `**Provider**: Cashi (real, production QRIS) + sandbox mock` → `**Provider**: gopay-notifications (real, production QRIS) + sandbox mock`; struktur folder `cashi/ # Adapter Cashi (HTTP real)` → `gopay/ # Adapter gopay-notifications (HTTP real)`; tabel env var `CASHI_API_KEY / CASHI_SECRET_KEY` → baris `GOPAY_BASE_URL` dengan keterangan kredensial per-merchant; diagram alur `(cashi | sandbox)` → `(gopay | sandbox)`; kalimat "Cashi hanyalah satu provider adapter" → "gopay hanyalah satu provider adapter".

- [ ] **Step 12: Commit**

```bash
cd whuzpay-pg/back
git add -A
git commit -m "refactor: perluas PaymentProvider dengan merchantID, hapus Cashi total"
```

---

### Task 2: Tabel + repository kredensial, adapter "gopay"

**Files:**
- Create: `whuzpay-pg/back/migrations/016_create_merchant_gopay_credentials.sql`
- Create: `whuzpay-pg/back/internal/repository/merchant_gopay_credentials_repository.go`
- Create: `whuzpay-pg/back/internal/provider/gopay/adapter.go`
- Create: `whuzpay-pg/back/internal/provider/gopay/types.go`
- Create: `whuzpay-pg/back/internal/provider/gopay/errors.go`
- Test: `whuzpay-pg/back/internal/provider/gopay/adapter_test.go`

**Interfaces:**
- Consumes: `PaymentProvider` interface (Task 1), `providerPkg.ErrInvalidWebhookSignature` (`internal/provider/errors.go`, sudah ada).
- Produces:
  ```go
  // internal/repository
  type GopayCredentials struct {
      MerchantID    uuid.UUID
      APIKey        *string
      WebhookSecret *string
  }
  var ErrGopayCredentialsNotFound = errors.New("gopay credentials not found")
  func NewMerchantGopayCredentialsRepository(db *sql.DB) *MerchantGopayCredentialsRepository
  func (r *MerchantGopayCredentialsRepository) Get(ctx context.Context, merchantID uuid.UUID) (*GopayCredentials, error)
  func (r *MerchantGopayCredentialsRepository) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error

  // internal/provider/gopay
  func NewAdapter(baseURL string, credsRepo credentialsRepository) *Adapter
  var ErrCredentialsNotConfigured = errors.New(...)
  var ErrQRISNotConfigured = errors.New(...)
  ```
  Task 3 memakai `NewAdapter`, `ErrCredentialsNotConfigured` (untuk `respondCreatePaymentError`). Task 4 memakai `*repository.MerchantGopayCredentialsRepository` (constructor sama, dipakai `PaymentService` juga).

- [ ] **Step 1: Migrasi**

`migrations/016_create_merchant_gopay_credentials.sql`:
```sql
-- Migration: Create merchant_gopay_credentials table
-- Kredensial gopay-notifications milik tiap merchant -- API key untuk
-- memanggil POST /invoices, webhook secret untuk verifikasi
-- X-Webhook-Signature. Disimpan polos (plaintext), pola sama dengan
-- merchants.webhook_secret (migrasi 013): keamanan mengandalkan akses
-- database, bukan enkripsi aplikasi (whuzpay-pg belum punya sistem kripto).

CREATE TABLE IF NOT EXISTS merchant_gopay_credentials (
    merchant_id UUID PRIMARY KEY REFERENCES merchants(id) ON DELETE CASCADE,
    api_key TEXT,
    webhook_secret TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE merchant_gopay_credentials IS 'Kredensial gopay-notifications per merchant. Kedua kolom independen -- merchant boleh isi api_key dulu sebelum sempat bikin webhook endpoint.';
COMMENT ON COLUMN merchant_gopay_credentials.api_key IS 'sk_... dari halaman API Keys gopay-notifications milik merchant ini.';
COMMENT ON COLUMN merchant_gopay_credentials.webhook_secret IS 'whsec_... dari webhook endpoint yang dibuat merchant di gopay-notifications, mengarah ke {APP_BASE_URL}/api/v1/provider-webhooks/gopay.';
```

Jalankan sesuai `scripts/migrate.sh` yang sudah ada (Akbar yang jalankan, butuh Postgres lokal whuzpay-pg jalan -- di luar cakupan agen menjalankan server sungguhan).

- [ ] **Step 2: Repository**

`internal/repository/merchant_gopay_credentials_repository.go`:
```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrGopayCredentialsNotFound = errors.New("gopay credentials not found")

// GopayCredentials -- APIKey/WebhookSecret independen (bisa salah satu
// nil kalau merchant baru isi satu dari dua).
type GopayCredentials struct {
	MerchantID    uuid.UUID
	APIKey        *string
	WebhookSecret *string
}

type MerchantGopayCredentialsRepository struct {
	db *sql.DB
}

func NewMerchantGopayCredentialsRepository(db *sql.DB) *MerchantGopayCredentialsRepository {
	return &MerchantGopayCredentialsRepository{db: db}
}

func (r *MerchantGopayCredentialsRepository) Get(ctx context.Context, merchantID uuid.UUID) (*GopayCredentials, error) {
	c := &GopayCredentials{MerchantID: merchantID}
	query := `SELECT api_key, webhook_secret FROM merchant_gopay_credentials WHERE merchant_id = $1`
	err := r.db.QueryRowContext(ctx, query, merchantID).Scan(&c.APIKey, &c.WebhookSecret)
	if err == sql.ErrNoRows {
		return nil, ErrGopayCredentialsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get gopay credentials: %w", err)
	}
	return c, nil
}

// Upsert menerima *string per field -- pola tri-state: nil = biarkan
// nilai lama, non-nil (termasuk string kosong) = ganti. Kolom yang tidak
// disentuh dipertahankan lewat COALESCE terhadap baris yang sudah ada
// (atau NULL kalau baris belum ada sama sekali).
func (r *MerchantGopayCredentialsRepository) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error {
	query := `
		INSERT INTO merchant_gopay_credentials (merchant_id, api_key, webhook_secret, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (merchant_id) DO UPDATE SET
			api_key = CASE WHEN $4 THEN merchant_gopay_credentials.api_key ELSE EXCLUDED.api_key END,
			webhook_secret = CASE WHEN $5 THEN merchant_gopay_credentials.webhook_secret ELSE EXCLUDED.webhook_secret END,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, merchantID, apiKey, webhookSecret, apiKey == nil, webhookSecret == nil)
	if err != nil {
		return fmt.Errorf("failed to upsert gopay credentials: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Tulis test adapter yang gagal dulu**

`internal/provider/gopay/adapter_test.go`:
```go
package gopay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainProvider "github.com/akbarryyan/pg-aggregator-back/internal/domain/provider"
	providerPkg "github.com/akbarryyan/pg-aggregator-back/internal/provider"
	"github.com/akbarryyan/pg-aggregator-back/internal/repository"
	"github.com/google/uuid"
)

type fakeCredsRepo struct {
	creds map[uuid.UUID]*repository.GopayCredentials
}

func (f *fakeCredsRepo) Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error) {
	c, ok := f.creds[merchantID]
	if !ok {
		return nil, repository.ErrGopayCredentialsNotFound
	}
	return c, nil
}

func strPtr(s string) *string { return &s }

func TestCreatePayment_Success(t *testing.T) {
	merchantID := uuid.New()
	apiKey := "sk_test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+apiKey {
			t.Errorf("Authorization header = %q, want Bearer %s", r.Header.Get("Authorization"), apiKey)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":              "inv_abc123",
			"external_ref":    "ORDER-1",
			"requested_amount": 50000,
			"unique_amount":   50347,
			"status":          "PENDING",
			"created_at":      "2026-09-15T09:30:00Z",
			"expires_at":      "2026-09-15T09:45:00Z",
			"qris_image":      "data:image/png;base64,iVBORw0KGgo",
		})
	}))
	defer server.Close()

	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, APIKey: &apiKey},
	}}
	adapter := NewAdapter(server.URL, repo)

	resp, err := adapter.CreatePayment(context.Background(), &domainProvider.ProviderPaymentRequest{
		InternalReference: "ORDER-1",
		Amount:            50000,
		MerchantID:        merchantID,
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if resp.ProviderReference != "inv_abc123" {
		t.Errorf("ProviderReference = %q, want inv_abc123", resp.ProviderReference)
	}
	if resp.Amount != 50347 {
		t.Errorf("Amount = %d, want 50347 (unique_amount)", resp.Amount)
	}
	if resp.QRISData == nil || *resp.QRISData != "data:image/png;base64,iVBORw0KGgo" {
		t.Errorf("QRISData = %v, want the qris_image data URI", resp.QRISData)
	}
}

func TestCreatePayment_CredentialsNotConfigured(t *testing.T) {
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{}}
	adapter := NewAdapter("http://should-not-be-called.invalid", repo)

	_, err := adapter.CreatePayment(context.Background(), &domainProvider.ProviderPaymentRequest{
		InternalReference: "ORDER-1",
		Amount:            50000,
		MerchantID:        uuid.New(),
	})
	if err != ErrCredentialsNotConfigured {
		t.Errorf("err = %v, want ErrCredentialsNotConfigured", err)
	}
}

func TestCreatePayment_QRISNotConfigured(t *testing.T) {
	merchantID := uuid.New()
	apiKey := "sk_test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"success": "false", "error": "qris_not_configured", "message": "QRIS belum diatur",
		})
	}))
	defer server.Close()

	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, APIKey: &apiKey},
	}}
	adapter := NewAdapter(server.URL, repo)

	_, err := adapter.CreatePayment(context.Background(), &domainProvider.ProviderPaymentRequest{
		InternalReference: "ORDER-1", Amount: 50000, MerchantID: merchantID,
	})
	if err != ErrQRISNotConfigured {
		t.Errorf("err = %v, want ErrQRISNotConfigured", err)
	}
}

func TestValidateWebhook_CorrectSignature(t *testing.T) {
	merchantID := uuid.New()
	secret := "whsec_abc123"
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, WebhookSecret: &secret},
	}}
	adapter := NewAdapter("http://unused.invalid", repo)

	payload := []byte(`{"event":"invoice.paid","invoice":{"id":"inv_abc123"}}`)
	sig := computeHMAC(payload, secret)

	if err := adapter.ValidateWebhook(payload, sig, merchantID); err != nil {
		t.Errorf("ValidateWebhook: %v, want nil", err)
	}
}

func TestValidateWebhook_WrongSignature(t *testing.T) {
	merchantID := uuid.New()
	secret := "whsec_abc123"
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, WebhookSecret: &secret},
	}}
	adapter := NewAdapter("http://unused.invalid", repo)

	payload := []byte(`{"event":"invoice.paid","invoice":{"id":"inv_abc123"}}`)
	err := adapter.ValidateWebhook(payload, "wrong-signature", merchantID)
	if err != providerPkg.ErrInvalidWebhookSignature {
		t.Errorf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

func TestValidateWebhook_SecretNotConfigured(t *testing.T) {
	merchantID := uuid.New()
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{}}
	adapter := NewAdapter("http://unused.invalid", repo)

	err := adapter.ValidateWebhook([]byte(`{}`), "any-signature", merchantID)
	if err != providerPkg.ErrInvalidWebhookSignature {
		t.Errorf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

func TestParseWebhook(t *testing.T) {
	adapter := NewAdapter("http://unused.invalid", &fakeCredsRepo{})
	payload := []byte(`{"event":"invoice.paid","invoice":{"id":"inv_abc123","external_ref":"ORDER-1","status":"PAID","paid_at":"2026-09-15T09:36:00Z"}}`)

	out, err := adapter.ParseWebhook(payload)
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if out.ProviderReference != "inv_abc123" {
		t.Errorf("ProviderReference = %q, want inv_abc123 (invoice.id, bukan external_ref)", out.ProviderReference)
	}
	if out.Status != "paid" {
		t.Errorf("Status = %q, want paid", out.Status)
	}
}

func TestNormalizeStatus(t *testing.T) {
	adapter := NewAdapter("http://unused.invalid", &fakeCredsRepo{})
	cases := map[string]string{"PAID": "paid", "EXPIRED": "expired", "PENDING": "pending", "unknown": "pending"}
	for in, want := range cases {
		if got := adapter.NormalizeStatus(in); got != want {
			t.Errorf("NormalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 4: Jalankan test, pastikan gagal**

```bash
cd whuzpay-pg/back
go test ./internal/provider/gopay/... -v
```
Expected: FAIL (package `gopay` belum ada implementasinya).

- [ ] **Step 5: Implementasi**

`internal/provider/gopay/types.go`:
```go
package gopay

// invoiceResponse -- bentuk respons POST/GET /api/v1/invoices gopay-notifications.
// Field yang tidak dipakai adapter ini (matched_event_id, paid_at) tetap
// didekode supaya unmarshal tidak gagal, meski tidak dipetakan ke ProviderPaymentResponse.
type invoiceResponse struct {
	ID              string  `json:"id"`
	ExternalRef     string  `json:"external_ref"`
	RequestedAmount int64   `json:"requested_amount"`
	UniqueAmount    int64   `json:"unique_amount"`
	Status          string  `json:"status"`
	MatchedEventID  *string `json:"matched_event_id"`
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       string  `json:"expires_at"`
	PaidAt          *string `json:"paid_at"`
	QRISImage       *string `json:"qris_image"`
}

type errorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// webhookPayload -- bentuk body POST {APP_BASE_URL}/api/v1/provider-webhooks/gopay
// dari gopay-notifications (lihat API Docs gopay-notifications §Webhook).
type webhookPayload struct {
	Event   string `json:"event"`
	Invoice struct {
		ID          string  `json:"id"`
		ExternalRef string  `json:"external_ref"`
		Status      string  `json:"status"`
		PaidAt      *string `json:"paid_at"`
	} `json:"invoice"`
	SentAt string `json:"sent_at"`
}
```

`internal/provider/gopay/errors.go`:
```go
package gopay

import "errors"

var (
	// ErrCredentialsNotConfigured: merchant belum mengisi API key
	// gopay-notifications di Settings -- lihat respondCreatePaymentError
	// di payment_handler.go (Task 3) untuk pemetaannya ke HTTP 400.
	ErrCredentialsNotConfigured = errors.New("gopay: merchant belum mengatur API key gopay-notifications")
	// ErrQRISNotConfigured: kredensial ada, tapi account gopay-notifications
	// merchant ini belum upload QRIS -- lihat sub-project 1.
	ErrQRISNotConfigured = errors.New("gopay: merchant belum mengatur QRIS di gopay-notifications")
)
```

`internal/provider/gopay/adapter.go`:
```go
package gopay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	domainProvider "github.com/akbarryyan/pg-aggregator-back/internal/domain/provider"
	providerPkg "github.com/akbarryyan/pg-aggregator-back/internal/provider"
	"github.com/akbarryyan/pg-aggregator-back/internal/repository"
	"github.com/google/uuid"
)

// credentialsRepository -- interface lokal ke package ini (bukan tipe
// konkret repository), supaya adapter bisa diuji dengan fake tanpa
// database sungguhan. *repository.MerchantGopayCredentialsRepository
// memenuhi ini secara struktural.
type credentialsRepository interface {
	Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error)
}

const ProviderName = "gopay"

type Adapter struct {
	baseURL    string
	credsRepo  credentialsRepository
	httpClient *http.Client
}

func NewAdapter(baseURL string, credsRepo credentialsRepository) *Adapter {
	return &Adapter{
		baseURL:    strings.TrimRight(baseURL, "/"),
		credsRepo:  credsRepo,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *Adapter) GetName() string { return ProviderName }

func (a *Adapter) CreatePayment(ctx context.Context, req *domainProvider.ProviderPaymentRequest) (*domainProvider.ProviderPaymentResponse, error) {
	creds, err := a.credsRepo.Get(ctx, req.MerchantID)
	if err != nil || creds.APIKey == nil || *creds.APIKey == "" {
		return nil, ErrCredentialsNotConfigured
	}

	body, _ := json.Marshal(map[string]interface{}{
		"external_ref": req.InternalReference,
		"amount":       req.Amount,
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/v1/invoices", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gopay: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+*creds.APIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gopay: create invoice: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusConflict {
		var errResp errorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error == "qris_not_configured" {
			return nil, ErrQRISNotConfigured
		}
		return nil, fmt.Errorf("gopay: create invoice ditolak: %s", string(respBody))
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gopay: create invoice status %d: %s", resp.StatusCode, string(respBody))
	}

	var inv invoiceResponse
	if err := json.Unmarshal(respBody, &inv); err != nil {
		return nil, fmt.Errorf("gopay: decode invoice response: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, inv.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("gopay: parse expires_at: %w", err)
	}

	return &domainProvider.ProviderPaymentResponse{
		ProviderReference: inv.ID,
		ProviderName:      ProviderName,
		Status:            "pending",
		Amount:            inv.UniqueAmount,
		QRISData:          inv.QRISImage,
		ExpiresAt:         expiresAt,
		RawResponse:       map[string]interface{}{"external_ref": inv.ExternalRef},
	}, nil
}

func (a *Adapter) GetPaymentStatus(ctx context.Context, providerReference string, merchantID uuid.UUID) (*domainProvider.NormalizedPaymentStatus, error) {
	creds, err := a.credsRepo.Get(ctx, merchantID)
	if err != nil || creds.APIKey == nil || *creds.APIKey == "" {
		return nil, ErrCredentialsNotConfigured
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/v1/invoices/"+providerReference, nil)
	if err != nil {
		return nil, fmt.Errorf("gopay: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+*creds.APIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gopay: get invoice: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gopay: get invoice status %d: %s", resp.StatusCode, string(respBody))
	}

	var inv invoiceResponse
	if err := json.Unmarshal(respBody, &inv); err != nil {
		return nil, fmt.Errorf("gopay: decode invoice response: %w", err)
	}

	return &domainProvider.NormalizedPaymentStatus{
		Status:            a.NormalizeStatus(inv.Status),
		ProviderReference: inv.ID,
	}, nil
}

func (a *Adapter) ValidateWebhook(rawPayload []byte, signature string, merchantID uuid.UUID) error {
	if signature == "" {
		return providerPkg.ErrInvalidWebhookSignature
	}
	creds, err := a.credsRepo.Get(context.Background(), merchantID)
	if err != nil || creds.WebhookSecret == nil || *creds.WebhookSecret == "" {
		return providerPkg.ErrInvalidWebhookSignature
	}

	mac := hmac.New(sha256.New, []byte(*creds.WebhookSecret))
	mac.Write(rawPayload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return providerPkg.ErrInvalidWebhookSignature
	}
	return nil
}

func (a *Adapter) ParseWebhook(rawPayload []byte) (*domainProvider.ProviderWebhookPayload, error) {
	var wh webhookPayload
	if err := json.Unmarshal(rawPayload, &wh); err != nil {
		return nil, fmt.Errorf("gopay: decode webhook payload: %w", err)
	}

	var paidAt *time.Time
	if wh.Invoice.PaidAt != nil {
		t, err := time.Parse(time.RFC3339, *wh.Invoice.PaidAt)
		if err == nil {
			paidAt = &t
		}
	}

	return &domainProvider.ProviderWebhookPayload{
		ProviderName:      ProviderName,
		ProviderReference: wh.Invoice.ID,
		Status:            a.NormalizeStatus(wh.Invoice.Status),
		PaidAt:            paidAt,
		RawPayload:        map[string]interface{}{"event": wh.Event, "external_ref": wh.Invoice.ExternalRef},
	}, nil
}

func (a *Adapter) NormalizeStatus(providerStatus string) string {
	switch strings.ToUpper(providerStatus) {
	case "PAID":
		return "paid"
	case "EXPIRED":
		return "expired"
	default:
		return "pending"
	}
}

// computeHMAC dipakai test (dan boleh dipakai ulang siapa pun yang perlu
// menghasilkan tanda tangan uji secara manual).
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
```

- [ ] **Step 6: Jalankan test lagi, pastikan lulus**

```bash
cd whuzpay-pg/back
go test ./internal/provider/gopay/... -v
```
Expected: semua `PASS`.

- [ ] **Step 7: `go build`, `go vet`, seluruh test, commit**

```bash
cd whuzpay-pg/back
go build ./... && go vet ./... && go test ./...
git add migrations/016_create_merchant_gopay_credentials.sql internal/repository/merchant_gopay_credentials_repository.go internal/provider/gopay/
git commit -m "feat: tabel+repository kredensial dan adapter provider gopay"
```

---

### Task 3: Wiring adapter + perbaikan alur webhook

**Files:**
- Modify: `whuzpay-pg/back/cmd/api/main.go`
- Modify: `whuzpay-pg/back/internal/handler/webhook_handler.go`
- Modify: `whuzpay-pg/back/internal/service/payment_webhook_service.go`
- Modify: `whuzpay-pg/back/internal/handler/payment_handler.go`
- Test: `whuzpay-pg/back/internal/service/payment_webhook_service_test.go`
- Test: `whuzpay-pg/back/internal/handler/payment_handler_test.go`

**Interfaces:**
- Consumes: `gopay.NewAdapter`, `gopay.ErrCredentialsNotConfigured` (Task 2); `repository.NewMerchantGopayCredentialsRepository` (Task 2).

- [ ] **Step 1: Tulis test yang gagal dulu -- urutan webhook per-merchant**

Tambahkan ke `internal/service/payment_webhook_service_test.go` (lihat isi file itu dulu untuk pola fake yang dipakai -- `fakeProvider`, `fakePaymentRepo`, dari `payment_fakes_test.go`, sama package `service`):
```go
func TestProcessWebhook_PerMerchantSecret(t *testing.T) {
	// Dua merchant, dua secret webhook BEDA terdaftar di provider yang sama --
	// membuktikan urutan baru (parse->cari payment->validate) benar-benar
	// mengambil secret milik MERCHANT YANG TEPAT, bukan yang pertama
	// ditemukan atau tetap (perilaku lama, satu secret global, akan salah
	// di sini kalau urutannya tidak diperbaiki).
	paymentRepo := newFakePaymentRepo()
	merchantAID, merchantBID := uuid.New(), uuid.New()

	paymentA := &payment.Payment{
		ID: uuid.New(), Reference: "REF-A", MerchantID: merchantAID,
		Status: payment.StatusPending, ProviderName: "fake",
	}
	refA := "prov-ref-a"
	paymentA.ProviderReference = &refA
	paymentRepo.Create(context.Background(), paymentA)

	fp := &fakeProvider{
		name: "fake",
		validateFn: func(payload []byte, sig string, merchantID uuid.UUID) error {
			if merchantID != merchantAID {
				t.Errorf("ValidateWebhook dipanggil dengan merchantID = %v, want %v (merchant A)", merchantID, merchantAID)
			}
			return nil
		},
		parseFn: func(payload []byte) (*domainProvider.ProviderWebhookPayload, error) {
			return &domainProvider.ProviderWebhookPayload{ProviderReference: refA, Status: "paid"}, nil
		},
	}
	router := providerPkg.NewProviderRouter()
	router.RegisterProvider(fp)

	webhookEventRepo := newFakeWebhookEventRepo()
	svc := NewPaymentService(paymentRepo, newFakeMerchantProviderConfigRepo(), webhookEventRepo, router, "http://localhost:8080")

	if err := svc.ProcessWebhook(context.Background(), "fake", []byte(`{}`), "any-sig"); err != nil {
		t.Fatalf("ProcessWebhook: %v", err)
	}
	_ = merchantBID // dipakai skenario lanjutan kalau perlu menambah payment merchant B
}
```

(Sesuaikan nama helper `newFakeWebhookEventRepo`/`newFakeMerchantProviderConfigRepo`/tipe `fakeProvider` PERSIS dengan yang sudah ada di `payment_fakes_test.go` -- lihat file itu dulu sebelum menulis test ini; kalau `fakeProvider` di sana belum punya field `validateFn`/`parseFn` yang bisa di-override per test, tambahkan sebagai field opsional pada struct `fakeProvider` yang sudah ada, dipanggil dari method `ValidateWebhook`/`ParseWebhook` kalau non-nil, fallback ke perilaku default kalau nil -- supaya test lain yang sudah memakai `fakeProvider` tanpa override tidak ikut berubah perilakunya.)

- [ ] **Step 2: Jalankan, pastikan gagal**

```bash
cd whuzpay-pg/back
go test ./internal/service/... -run ProcessWebhook_PerMerchantSecret -v
```
Expected: FAIL -- `ValidateWebhook` masih dipanggil sebelum payment ditemukan, jadi `merchantID` yang diterima adapter salah (nol/`uuid.Nil`) atau urutan pemanggilan salah.

- [ ] **Step 3: Balik urutan di `internal/service/payment_webhook_service.go`**

Ganti:
```go
	selectedProvider, err := s.getProviderByName(providerName)
	if err != nil {
		s.failWebhookEvent(ctx, event.ID, nil, "", "rejected", "rejected", err)
		return err
	}

	if err := selectedProvider.ValidateWebhook(rawPayload, signature); err != nil {
		logger.ErrorfCtx(ctx, "Webhook validation failed: %v", err)
		s.failWebhookEvent(ctx, event.ID, nil, "", "rejected", "rejected", payment.ErrWebhookValidationFailed)
		return payment.ErrWebhookValidationFailed
	}

	webhookPayload, err := selectedProvider.ParseWebhook(rawPayload)
	if err != nil {
		if errors.Is(err, providerPkg.ErrTestWebhookEvent) {
			logger.InfofCtx(ctx, "Ignoring test webhook event from provider: %s", providerName)
			s.failWebhookEvent(ctx, event.ID, nil, "", "ignored", "ignored", providerPkg.ErrTestWebhookEvent)
			return nil
		}
		logger.ErrorfCtx(ctx, "Failed to parse webhook: %v", err)
		s.failWebhookEvent(ctx, event.ID, nil, "", "rejected", "rejected", payment.ErrInvalidProviderReference)
		return payment.ErrInvalidProviderReference
	}

	logger.InfofCtx(ctx, "Webhook parsed: provider_reference=%s, status=%s", webhookPayload.ProviderReference, webhookPayload.Status)

	p, err := s.paymentRepo.GetByProviderReference(ctx, webhookPayload.ProviderReference)
	if err != nil {
		logger.ErrorfCtx(ctx, "Payment not found for provider reference: %s", webhookPayload.ProviderReference)
		s.failWebhookEvent(ctx, event.ID, nil, webhookPayload.ProviderReference, webhookEventType(webhookPayload.Status), webhookPayload.Status, err)
		return err
	}
```
jadi (parse dan cari payment DULUAN, validasi belakangan dengan `p.MerchantID`):
```go
	selectedProvider, err := s.getProviderByName(providerName)
	if err != nil {
		s.failWebhookEvent(ctx, event.ID, nil, "", "rejected", "rejected", err)
		return err
	}

	// Parse dan cari payment DULU, validasi tanda tangan BELAKANGAN --
	// beda dari urutan lama (validate lalu parse). Alasan: tiap merchant
	// (gopay) punya webhook secret sendiri, cuma diketahui SETELAH payment
	// (dan MerchantID-nya) ditemukan. Ini aman -- provider_reference yang
	// dibaca di sini adalah referensi publik, bukan rahasia, dan tidak ada
	// state yang berubah sebelum ValidateWebhook di bawah lulus.
	webhookPayload, err := selectedProvider.ParseWebhook(rawPayload)
	if err != nil {
		if errors.Is(err, providerPkg.ErrTestWebhookEvent) {
			logger.InfofCtx(ctx, "Ignoring test webhook event from provider: %s", providerName)
			s.failWebhookEvent(ctx, event.ID, nil, "", "ignored", "ignored", providerPkg.ErrTestWebhookEvent)
			return nil
		}
		logger.ErrorfCtx(ctx, "Failed to parse webhook: %v", err)
		s.failWebhookEvent(ctx, event.ID, nil, "", "rejected", "rejected", payment.ErrInvalidProviderReference)
		return payment.ErrInvalidProviderReference
	}

	logger.InfofCtx(ctx, "Webhook parsed: provider_reference=%s, status=%s", webhookPayload.ProviderReference, webhookPayload.Status)

	p, err := s.paymentRepo.GetByProviderReference(ctx, webhookPayload.ProviderReference)
	if err != nil {
		logger.ErrorfCtx(ctx, "Payment not found for provider reference: %s", webhookPayload.ProviderReference)
		s.failWebhookEvent(ctx, event.ID, nil, webhookPayload.ProviderReference, webhookEventType(webhookPayload.Status), webhookPayload.Status, err)
		return err
	}

	if err := selectedProvider.ValidateWebhook(rawPayload, signature, p.MerchantID); err != nil {
		logger.ErrorfCtx(ctx, "Webhook validation failed: %v", err)
		s.failWebhookEvent(ctx, event.ID, &p.ID, webhookPayload.ProviderReference, "rejected", "rejected", payment.ErrWebhookValidationFailed)
		return payment.ErrWebhookValidationFailed
	}
```
(Baris-baris SETELAH ini -- cek status terminal/duplikat, terapkan update -- TIDAK berubah, cuma dipindah ke bawah blok Validate yang baru.)

- [ ] **Step 4: Jalankan test, pastikan lulus**

```bash
cd whuzpay-pg/back
go test ./internal/service/... -run ProcessWebhook -v
```
Expected: `PASS`, termasuk test lama yang sudah ada di file itu (urutan baru tidak mengubah hasil akhir untuk kasus yang sudah dites -- cuma urutan pemanggilan internalnya).

- [ ] **Step 5: Ganti header signature di `internal/handler/webhook_handler.go`**

Ganti:
```go
	signature := r.Header.Get("x-gateway-signature")
```
jadi:
```go
	// Nama header gopay-notifications, bukan konvensi Cashi lama
	// ("x-gateway-signature") yang sudah dihapus bersama Cashi.
	signature := r.Header.Get("X-Webhook-Signature")
```

- [ ] **Step 6: Wiring di `cmd/api/main.go`**

Tambah import:
```go
	"github.com/akbarryyan/pg-aggregator-back/internal/provider/gopay"
```
Tambah setelah baris `merchantAPIKeyRepo := repository.NewMerchantAPIKeyRepository(db)` (atau baris sejenis di dekat repo lain):
```go
	gopayCredsRepo := repository.NewMerchantGopayCredentialsRepository(db)
```
Ganti blok sementara dari Task 1 Step 8:
```go
	providerRouter := provider.NewProviderRouter()
	providerRouter.RegisterProvider(sandboxAdapter)
	// TODO(Task 3): daftarkan gopayAdapter + RegisterPaymentMethodProvider("qris", ...)
```
jadi:
```go
	gopayAdapter := gopay.NewAdapter(cfg.Gopay.BaseURL, gopayCredsRepo)

	providerRouter := provider.NewProviderRouter()
	providerRouter.RegisterProvider(gopayAdapter)
	providerRouter.RegisterProvider(sandboxAdapter)
	// Production QRIS routing uses real gopay-notifications only
	providerRouter.RegisterPaymentMethodProvider("qris", gopayAdapter.GetName())
```

- [ ] **Step 7: Tambah kasus error di `respondCreatePaymentError` (`internal/handler/payment_handler.go`)**

Tambah import `"github.com/akbarryyan/pg-aggregator-back/internal/provider/gopay"`. Ubah:
```go
func respondCreatePaymentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, providerPkg.ErrProviderNotAvailable):
		respondError(w, http.StatusServiceUnavailable, "Payment provider is temporarily unavailable")
	case errors.Is(err, providerPkg.ErrUnsupportedPaymentMethod):
		respondError(w, http.StatusBadRequest, "Unsupported payment method")
	case errors.Is(err, payment.ErrProviderError):
		respondError(w, http.StatusBadGateway, "Payment provider error")
	default:
		respondError(w, http.StatusInternalServerError, "Failed to create payment")
	}
}
```
jadi:
```go
func respondCreatePaymentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, providerPkg.ErrProviderNotAvailable):
		respondError(w, http.StatusServiceUnavailable, "Payment provider is temporarily unavailable")
	case errors.Is(err, providerPkg.ErrUnsupportedPaymentMethod):
		respondError(w, http.StatusBadRequest, "Unsupported payment method")
	case errors.Is(err, gopay.ErrCredentialsNotConfigured):
		respondError(w, http.StatusBadRequest, "Merchant belum mengatur kredensial gopay-notifications (lihat Settings)")
	case errors.Is(err, gopay.ErrQRISNotConfigured):
		respondError(w, http.StatusBadRequest, "Merchant belum mengatur QRIS di akun gopay-notifications-nya")
	case errors.Is(err, payment.ErrProviderError):
		respondError(w, http.StatusBadGateway, "Payment provider error")
	default:
		respondError(w, http.StatusInternalServerError, "Failed to create payment")
	}
}
```

- [ ] **Step 8: Tes handler untuk kasus kredensial belum diatur**

Tambahkan ke `internal/handler/payment_handler_test.go` (lihat pola `fakeProvider` di `internal/handler/fakes_test.go` untuk cara membuatnya mengembalikan `gopay.ErrCredentialsNotConfigured` dari `CreatePayment` -- tambahkan field function-override opsional pada `fakeProvider` seperti Step 1, kalau belum ada):
```go
func TestCreatePayment_CredentialsNotConfigured(t *testing.T) {
	// ... setup handler dengan fakeProvider yang CreatePayment-nya
	// mengembalikan gopay.ErrCredentialsNotConfigured ...
	rec := httptest.NewRecorder()
	// ... kirim request POST /payments ...
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
```

- [ ] **Step 9: `go build`, `go vet`, seluruh test, commit**

```bash
cd whuzpay-pg/back
go build ./... && go vet ./... && go test ./...
git add cmd/api/main.go internal/handler/webhook_handler.go internal/service/payment_webhook_service.go internal/service/payment_webhook_service_test.go internal/handler/payment_handler.go internal/handler/payment_handler_test.go internal/handler/fakes_test.go internal/service/payment_fakes_test.go
git commit -m "feat: daftarkan provider gopay, urutan webhook per-merchant, error kredensial belum diatur"
```

---

### Task 4: Endpoint self-service kredensial merchant

**Files:**
- Modify: `whuzpay-pg/back/internal/service/interfaces.go`
- Modify: `whuzpay-pg/back/internal/service/payment_service.go`
- Modify: `whuzpay-pg/back/internal/handler/merchant_handler.go`
- Modify: `whuzpay-pg/back/cmd/api/main.go`
- Test: `whuzpay-pg/back/internal/service/payment_service_test.go`

**Interfaces:**
- Consumes: `repository.GopayCredentials`, `repository.ErrGopayCredentialsNotFound`, `*repository.MerchantGopayCredentialsRepository` (Task 2).
- Produces:
  ```go
  func (s *PaymentService) WithGopayCredentialsRepo(repo gopayCredentialsRepository) *PaymentService
  func (s *PaymentService) GetGopayCredentialsStatus(ctx context.Context, merchantID uuid.UUID) (apiKeySet, webhookSecretSet bool, err error)
  func (s *PaymentService) UpdateGopayCredentials(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error
  ```
  Route baru dipakai frontend Task 5: `GET`/`PUT /api/v1/merchant/gopay-credentials`.

- [ ] **Step 1: Tulis test yang gagal dulu**

Tambahkan ke `internal/service/payment_service_test.go`:
```go
type fakeGopayCredsRepo struct {
	creds map[uuid.UUID]*repository.GopayCredentials
}

func newFakeGopayCredsRepo() *fakeGopayCredsRepo {
	return &fakeGopayCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{}}
}

func (f *fakeGopayCredsRepo) Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error) {
	c, ok := f.creds[merchantID]
	if !ok {
		return nil, repository.ErrGopayCredentialsNotFound
	}
	return c, nil
}

func (f *fakeGopayCredsRepo) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error {
	existing, ok := f.creds[merchantID]
	if !ok {
		existing = &repository.GopayCredentials{MerchantID: merchantID}
	}
	if apiKey != nil {
		existing.APIKey = apiKey
	}
	if webhookSecret != nil {
		existing.WebhookSecret = webhookSecret
	}
	f.creds[merchantID] = existing
	return nil
}

func TestGetGopayCredentialsStatus_BelumDiatur(t *testing.T) {
	svc := NewPaymentService(newFakePaymentRepo(), newFakeMerchantProviderConfigRepo(), newFakeWebhookEventRepo(), providerPkg.NewProviderRouter(), "http://localhost:8080").
		WithGopayCredentialsRepo(newFakeGopayCredsRepo())

	apiKeySet, webhookSet, err := svc.GetGopayCredentialsStatus(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetGopayCredentialsStatus: %v", err)
	}
	if apiKeySet || webhookSet {
		t.Errorf("apiKeySet=%v webhookSet=%v, want false false", apiKeySet, webhookSet)
	}
}

func TestUpdateGopayCredentials_LaluGetStatus(t *testing.T) {
	repo := newFakeGopayCredsRepo()
	svc := NewPaymentService(newFakePaymentRepo(), newFakeMerchantProviderConfigRepo(), newFakeWebhookEventRepo(), providerPkg.NewProviderRouter(), "http://localhost:8080").
		WithGopayCredentialsRepo(repo)
	merchantID := uuid.New()

	apiKey := "sk_test"
	if err := svc.UpdateGopayCredentials(context.Background(), merchantID, &apiKey, nil); err != nil {
		t.Fatalf("UpdateGopayCredentials (api key saja): %v", err)
	}
	apiKeySet, webhookSet, _ := svc.GetGopayCredentialsStatus(context.Background(), merchantID)
	if !apiKeySet || webhookSet {
		t.Errorf("apiKeySet=%v webhookSet=%v, want true false (baru isi api key)", apiKeySet, webhookSet)
	}

	secret := "whsec_test"
	if err := svc.UpdateGopayCredentials(context.Background(), merchantID, nil, &secret); err != nil {
		t.Fatalf("UpdateGopayCredentials (webhook secret): %v", err)
	}
	apiKeySet, webhookSet, _ = svc.GetGopayCredentialsStatus(context.Background(), merchantID)
	if !apiKeySet || !webhookSet {
		t.Errorf("apiKeySet=%v webhookSet=%v, want true true (dua-duanya sudah diisi)", apiKeySet, webhookSet)
	}
}
```
(Sesuaikan nama helper fake yang sudah ada persis seperti di `payment_fakes_test.go`; tambahkan import `"github.com/akbarryyan/pg-aggregator-back/internal/repository"` kalau belum ada di file test ini.)

- [ ] **Step 2: Jalankan, pastikan gagal**

```bash
cd whuzpay-pg/back
go test ./internal/service/... -run GopayCredentials -v
```
Expected: FAIL -- `WithGopayCredentialsRepo`/`GetGopayCredentialsStatus`/`UpdateGopayCredentials` belum ada.

- [ ] **Step 3: Tambah interface di `internal/service/interfaces.go`**

```go
type gopayCredentialsRepository interface {
	Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error)
	Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error
}
```

- [ ] **Step 4: Implementasi di `internal/service/payment_service.go`**

Tambah field ke struct `PaymentService`:
```go
	gopayCredentialsRepo gopayCredentialsRepository
```
Tambah method (dekat `WithSandboxProvider`):
```go
// WithGopayCredentialsRepo wires per-merchant gopay-notifications
// credential storage -- dipakai halaman Settings merchant (lihat
// MerchantHandler.GetGopayCredentials/UpdateGopayCredentials).
func (s *PaymentService) WithGopayCredentialsRepo(repo gopayCredentialsRepository) *PaymentService {
	s.gopayCredentialsRepo = repo
	return s
}

// GetGopayCredentialsStatus TIDAK PERNAH mengembalikan nilai kredensial
// asli -- cuma status terisi/tidak, pola sama smtp_password_set gopay-notifications.
func (s *PaymentService) GetGopayCredentialsStatus(ctx context.Context, merchantID uuid.UUID) (apiKeySet, webhookSecretSet bool, err error) {
	creds, err := s.gopayCredentialsRepo.Get(ctx, merchantID)
	if errors.Is(err, repository.ErrGopayCredentialsNotFound) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return creds.APIKey != nil && *creds.APIKey != "", creds.WebhookSecret != nil && *creds.WebhookSecret != "", nil
}

// UpdateGopayCredentials -- tri-state per field: nil = biarkan, ""=hapus,
// isi=ganti (pola sama NotificationSettings gopay-notifications).
func (s *PaymentService) UpdateGopayCredentials(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error {
	return s.gopayCredentialsRepo.Upsert(ctx, merchantID, apiKey, webhookSecret)
}
```
Tambah import `"github.com/akbarryyan/pg-aggregator-back/internal/repository"` kalau belum ada (dipakai `repository.ErrGopayCredentialsNotFound`).

- [ ] **Step 5: Jalankan test lagi, pastikan lulus**

```bash
cd whuzpay-pg/back
go test ./internal/service/... -run GopayCredentials -v
```
Expected: `PASS`.

- [ ] **Step 6: Handler + route**

Di `internal/handler/merchant_handler.go`, tambah (dekat `GetWebhookSecret`/`RegenerateWebhookSecret`):
```go
type gopayCredentialsStatusResponse struct {
	APIKeyConfigured        bool `json:"api_key_configured"`
	WebhookSecretConfigured bool `json:"webhook_secret_configured"`
}

// GetGopayCredentials mengembalikan status saja -- tidak pernah nilai
// aslinya (pola sama GetWebhookSecret sebenarnya BEDA -- webhook secret
// aggregator ini boleh ditampilkan balik karena generated-by-us; ini
// kredensial gopay-notifications MILIK PIHAK LAIN yang di-paste merchant,
// jadi jangan pernah dikembalikan).
func (h *MerchantHandler) GetGopayCredentials(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := middleware.MerchantIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	apiKeySet, webhookSet, err := h.paymentService.GetGopayCredentialsStatus(r.Context(), merchantID)
	if err != nil {
		logger.ErrorfCtx(r.Context(), "Failed to get gopay credentials status for merchant %s: %v", merchantID, err)
		respondError(w, http.StatusInternalServerError, "Failed to load gopay credentials")
		return
	}
	respondJSON(w, http.StatusOK, gopayCredentialsStatusResponse{
		APIKeyConfigured: apiKeySet, WebhookSecretConfigured: webhookSet,
	})
}

type updateGopayCredentialsRequest struct {
	APIKey        *string `json:"api_key"`
	WebhookSecret *string `json:"webhook_secret"`
}

// UpdateGopayCredentials -- tri-state: field absen di JSON (nil setelah
// decode) berarti biarkan, "" berarti hapus, isi berarti ganti.
func (h *MerchantHandler) UpdateGopayCredentials(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := middleware.MerchantIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req updateGopayCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.paymentService.UpdateGopayCredentials(r.Context(), merchantID, req.APIKey, req.WebhookSecret); err != nil {
		logger.ErrorfCtx(r.Context(), "Failed to update gopay credentials for merchant %s: %v", merchantID, err)
		respondError(w, http.StatusInternalServerError, "Failed to update gopay credentials")
		return
	}
	apiKeySet, webhookSet, err := h.paymentService.GetGopayCredentialsStatus(r.Context(), merchantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load gopay credentials")
		return
	}
	respondJSON(w, http.StatusOK, gopayCredentialsStatusResponse{
		APIKeyConfigured: apiKeySet, WebhookSecretConfigured: webhookSet,
	})
}
```

Di `cmd/api/main.go`, wiring `paymentService`:
```go
	paymentService := service.NewPaymentService(paymentRepo, merchantProviderConfigRepo, webhookEventRepo, providerRouter, cfg.App.URL).
		WithMerchantCallbackDeps(merchantRepo, callbackRepo).
		WithSandboxProvider(sandboxAdapter).
		WithGopayCredentialsRepo(gopayCredsRepo)
```
Route (dekat `merchantDash.HandleFunc("/webhook-secret", ...)`), pola `sensitiveRateLimiter` sama seperti `webhook-secret/regenerate` karena ini juga aksi mutasi kredensial:
```go
	merchantDash.HandleFunc("/gopay-credentials", merchantHandler.GetGopayCredentials).Methods("GET")
	merchantDash.Handle("/gopay-credentials", sensitiveRateLimiter.Limit(http.HandlerFunc(merchantHandler.UpdateGopayCredentials))).Methods("PUT")
```

- [ ] **Step 7: `go build`, `go vet`, seluruh test, commit**

```bash
cd whuzpay-pg/back
go build ./... && go vet ./... && go test ./...
git add internal/service/interfaces.go internal/service/payment_service.go internal/service/payment_service_test.go internal/handler/merchant_handler.go cmd/api/main.go
git commit -m "feat: endpoint self-service kredensial gopay-notifications merchant"
```

---

### Task 5: UI merchant + pembersihan admin frontend

**Files:**
- Modify: `whuzpay-pg/front/lib/merchant-api.ts`
- Modify: `whuzpay-pg/front/app/dashboard/settings/page.tsx`
- Modify: `whuzpay-pg/front/app/components/admin/MerchantProviderConfigSection.tsx`

**Interfaces:**
- Consumes: `GET`/`PUT /api/v1/merchant/gopay-credentials` (Task 4).

- [ ] **Step 1: Tambah fungsi API client di `lib/merchant-api.ts`**

Lihat pola `fetchMerchantWebhookSecret`/`regenerateMerchantWebhookSecret` yang sudah ada di file itu, ikuti gaya yang sama (fetch wrapper, base URL, error handling). Tambahkan:
```typescript
export interface GopayCredentialsStatus {
  api_key_configured: boolean;
  webhook_secret_configured: boolean;
}

export async function fetchGopayCredentialsStatus(): Promise<GopayCredentialsStatus> {
  return apiFetch<GopayCredentialsStatus>("/api/v1/merchant/gopay-credentials", { method: "GET" });
}

export async function updateGopayCredentials(input: {
  api_key?: string;
  webhook_secret?: string;
}): Promise<GopayCredentialsStatus> {
  return apiFetch<GopayCredentialsStatus>("/api/v1/merchant/gopay-credentials", {
    method: "PUT",
    body: JSON.stringify(input),
  });
}
```
(Ganti `apiFetch` dengan nama helper fetch yang sesungguhnya dipakai file ini -- cek definisi/impor di bagian atas `merchant-api.ts` sebelum menulis, supaya konsisten persis dengan `fetchMerchantWebhookSecret` dkk.)

- [ ] **Step 2: Card baru di `app/dashboard/settings/page.tsx`**

Tambahkan komponen baru (pola sama `WebhookSecretCard` yang sudah ada di file ini -- `Card` dari `../../components/admin/ui`, `useState`+`useEffect` load status, `toast` hasil aksi):
```tsx
function GopayCredentialsCard() {
  const [status, setStatus] = useState<GopayCredentialsStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [apiKeyInput, setApiKeyInput] = useState("");
  const [webhookSecretInput, setWebhookSecretInput] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const s = await fetchGopayCredentialsStatus();
        if (!cancelled) setStatus(s);
      } catch {
        if (!cancelled) toast.error("Gagal memuat status kredensial gopay");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleSaveApiKey(e: FormEvent) {
    e.preventDefault();
    if (!apiKeyInput) return;
    setSaving(true);
    try {
      const s = await updateGopayCredentials({ api_key: apiKeyInput });
      setStatus(s);
      setApiKeyInput("");
      toast.success("API key gopay disimpan");
    } catch {
      toast.error("Gagal menyimpan API key");
    } finally {
      setSaving(false);
    }
  }

  async function handleSaveWebhookSecret(e: FormEvent) {
    e.preventDefault();
    if (!webhookSecretInput) return;
    setSaving(true);
    try {
      const s = await updateGopayCredentials({ webhook_secret: webhookSecretInput });
      setStatus(s);
      setWebhookSecretInput("");
      toast.success("Webhook secret gopay disimpan");
    } catch {
      toast.error("Gagal menyimpan webhook secret");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card title="Provider Pembayaran GoPay">
      <ol className="mb-4 list-decimal space-y-1 pl-5 text-sm text-muted-foreground">
        <li>Daftar/login ke gopay-notifications, upload QRIS di halaman Settings-nya.</li>
        <li>Buat API key di halaman API Keys gopay-notifications.</li>
        <li>
          Buat webhook endpoint mengarah ke{" "}
          <code>{"{APP_BASE_URL}"}/api/v1/provider-webhooks/gopay</code> dengan event{" "}
          <code>invoice.paid</code> di halaman Webhooks gopay-notifications.
        </li>
        <li>Tempel API key dan webhook secret yang didapat ke form di bawah ini.</li>
      </ol>

      {loading ? (
        <LoadingBlock />
      ) : (
        <div className="space-y-4">
          <form onSubmit={handleSaveApiKey} className="flex items-end gap-2">
            <div className="flex-1">
              <label className="mb-1 block text-sm font-medium">
                API Key {status?.api_key_configured ? "(sudah diatur)" : "(belum diatur)"}
              </label>
              <input
                type="password"
                className="w-full rounded border px-3 py-2 text-sm"
                placeholder="sk_..."
                value={apiKeyInput}
                onChange={(e) => setApiKeyInput(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={saving || !apiKeyInput}>
              Simpan
            </Button>
          </form>

          <form onSubmit={handleSaveWebhookSecret} className="flex items-end gap-2">
            <div className="flex-1">
              <label className="mb-1 block text-sm font-medium">
                Webhook Secret {status?.webhook_secret_configured ? "(sudah diatur)" : "(belum diatur)"}
              </label>
              <input
                type="password"
                className="w-full rounded border px-3 py-2 text-sm"
                placeholder="whsec_..."
                value={webhookSecretInput}
                onChange={(e) => setWebhookSecretInput(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={saving || !webhookSecretInput}>
              Simpan
            </Button>
          </form>
        </div>
      )}
    </Card>
  );
}
```
Tambah import (`fetchGopayCredentialsStatus`, `updateGopayCredentials`, `type GopayCredentialsStatus` dari `@/lib/merchant-api`), render `<GopayCredentialsCard />` di komponen halaman utama (dekat `<WebhookSecretCard />`).

- [ ] **Step 3: Ganti default provider di `MerchantProviderConfigSection.tsx`**

Ganti:
```typescript
const emptyForm = {
  provider_name: "cashi",
  ...
};
```
jadi:
```typescript
const emptyForm = {
  provider_name: "gopay",
  ...
};
```
Ganti fallback `useState<string[]>(["cashi"])` jadi `useState<string[]>(["gopay"])`.

- [ ] **Step 4: Build**

```bash
cd whuzpay-pg/front
npm run build
```
Expected: build bersih, tidak ada error TypeScript.

- [ ] **Step 5: Commit**

```bash
cd whuzpay-pg/front
git add lib/merchant-api.ts app/dashboard/settings/page.tsx app/components/admin/MerchantProviderConfigSection.tsx
git commit -m "feat: UI kredensial provider gopay di Settings merchant"
```

---

### Task 6: Verifikasi akhir + QA report

**Files:**
- Modify: `docs/qa/qa-report.md` (root repo, BUKAN di dalam `whuzpay-pg/`)

- [ ] **Step 1: Jalankan seluruh suite backend whuzpay-pg**

```bash
cd whuzpay-pg/back
go build ./... && go vet ./... && go test ./... -v 2>&1 | tail -100
```
Tempel ringkasan hasilnya (jumlah PASS, nama paket) ke laporan QA.

- [ ] **Step 2: Jalankan build frontend whuzpay-pg**

```bash
cd whuzpay-pg/front
npm run build
```

- [ ] **Step 3: Tulis laporan QA**

Tambahkan section baru "## Sub-project 2: Provider gopay di whuzpay-pg + pencabutan Cashi" di `docs/qa/qa-report.md` (root repo), tabel PASS/FAIL/NEEDS-DEVICE:
- Seluruh test Go di atas (Task 1-4) -- PASS dengan output ditempel.
- Build frontend -- PASS dengan output ditempel.
- Grep memastikan tidak ada sisa referensi Cashi fungsional (`grep -rn -i cashi whuzpay-pg/ --include=*.go --include=*.ts --include=*.tsx`) -- PASS, nihil di luar CHANGELOG/riwayat git.
- Uji end-to-end sungguhan (merchant isi kredensial asli, buat payment, scan QR, bayar, webhook gopay-notifications sungguhan sampai ke whuzpay-pg) -- `NEEDS-DEVICE`, butuh whuzpay-pg bisa diakses publik (tunnel atau deploy) supaya gopay-notifications bisa mengirim webhook ke situ.

- [ ] **Step 4: Commit**

```bash
git add docs/qa/qa-report.md
git commit -m "docs(qa): laporan QA sub-project 2 -- provider gopay di whuzpay-pg"
```

---

## Self-Review (dilakukan penulis plan, bukan langkah eksekusi)

- **Cakupan spec:** §2 (interface) → Task 1. §3 (kredensial) → Task 2. §4 (adapter) → Task 2. §5 (urutan webhook) → Task 3. §6 (cabut Cashi) → Task 1 (kode) + Task 1 Step 11 (dokumentasi/bruno). §7 (UI merchant) → Task 4 (backend) + Task 5 (frontend). §8 (testing) → tercakup di tiap task. §9 (di luar cakupan) → sengaja tidak ada task.
- **Placeholder scan:** tidak ada TBD/TODO literal kecuali satu komentar kode `// TODO(Task 3): ...` yang SENGAJA ada di Task 1 Step 8 sebagai penanda status antar-task (dihapus lagi di Task 3 Step 6) -- bukan placeholder yang dibiarkan.
- **Konsistensi tipe:** `PaymentProvider` interface (Task 1) ↔ implementasi `gopay.Adapter` (Task 2) ↔ implementasi `sandbox.Adapter` (Task 1) -- ketiganya method set sama persis. `repository.GopayCredentials{MerchantID, APIKey, WebhookSecret}` dipakai identik di Task 2 (definisi+adapter) dan Task 4 (`gopayCredentialsRepository` interface, `PaymentService`). Path webhook (`/api/v1/provider-webhooks/gopay`) dan path kredensial (`/api/v1/merchant/gopay-credentials`) konsisten di seluruh task (spec, Task 3, Task 4, Task 5) setelah koreksi.
