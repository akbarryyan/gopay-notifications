# Onboarding Terpadu whuzpay-pg Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Satu form pendaftaran di whuzpay-pg otomatis membuat akun gopay-notifications, API key, webhook, dan device (QR pairing) di baliknya — tanpa mengubah endpoint publik gopay-notifications dan tanpa pernah menyimpan sesi/device secret ke database.

**Architecture:** Package baru `internal/gopayonboard` di whuzpay-pg/back membungkus panggilan HTTP session-based (cookie jar, dibuat baru per registrasi, tidak pernah dibagi antar request) ke endpoint admin gopay-notifications yang sudah ada. `AuthService.RegisterMerchant` menjalankan cascade best-effort setelah akun whuzpay-pg sendiri berhasil dibuat.

**Tech Stack:** Go (net/http, net/http/cookiejar), PostgreSQL (migrasi tambahan), Next.js/React (form + halaman baru).

## Global Constraints

- Endpoint publik gopay-notifications (`POST /api/v1/signup`, `POST /api/v1/admin/api-keys`, `POST /api/v1/admin/webhooks`, `POST /api/v1/admin/devices`, `PUT /api/v1/admin/account/qris-image`) TIDAK PERNAH diubah — dipanggil apa adanya.
- Sesi gopay-notifications dan device secret TIDAK PERNAH ditulis ke database whuzpay-pg — device secret cuma lewat response registrasi sekali pakai, sesi cuma hidup selama satu `*gopayonboard.Client` per request.
- Registrasi whuzpay-pg WAJIB tetap `201` apa pun yang terjadi di cascade gopay-notifications (network error, email/username taken, gagal sebagian).
- Tidak ada mekanisme klaim akun gopay-notifications lama lewat kecocokan email.

---

### Task 1: Migrasi + repository — kolom `gopay_username` dan `qris_configured_at`

**Files:**
- Create: `whuzpay-pg/back/migrations/017_add_onboarding_columns_to_merchant_gopay_credentials.sql`
- Modify: `whuzpay-pg/back/internal/repository/merchant_gopay_credentials_repository.go`
- Test: `whuzpay-pg/back/internal/repository/merchant_gopay_credentials_repository_test.go` (baru — repository ini belum punya test sama sekali; dites lewat `sqlmock` mengikuti pola repository lain di paket ini. Cek `internal/repository/*_test.go` yang sudah ada untuk pola exact sebelum menulis kalau ragu — kalau paket ini ternyata tidak memakai `sqlmock` dan repository lain juga tidak punya test unit, lewati test repository ini dan cukup andalkan Task 5's integration test lewat `PaymentService`/`AuthService` yang sudah pasti punya fake)

**Interfaces:**
- Produces: `GopayCredentials{MerchantID, APIKey, WebhookSecret *string, Username *string, QRISConfiguredAt *time.Time}`, `Upsert(ctx, merchantID uuid.UUID, apiKey, webhookSecret, username *string) error` (SIGNATURE BERUBAH — tambah param `username`), `MarkQRISConfigured(ctx, merchantID uuid.UUID) error` (baru). Dipakai Task 3.

- [ ] **Step 1: Cek dulu apakah repository lain di paket ini punya test unit**

Run: `ls whuzpay-pg/back/internal/repository/*_test.go`

Kalau kosong (tidak ada satu pun), lewati bikin test repository — ikuti pola yang sudah ada (repository di paket ini memang tidak dites langsung, cuma lewat service test dengan fake). Lanjut ke Step 2. Kalau ada test lain yang memakai `sqlmock` atau pola serupa, baca satu contohnya dan tiru pola itu untuk `merchant_gopay_credentials_repository_test.go` sebelum lanjut.

- [ ] **Step 2: Tulis migrasi 017**

```sql
-- Migration: Add onboarding columns to merchant_gopay_credentials
-- Dua kolom baru dari onboarding terpadu (lihat spec
-- 2026-09-17-whuzpay-pg-unified-onboarding-design.md):
-- gopay_username dicatat supaya merchant bisa login langsung ke
-- gopay-notifications kapan saja kalau mau, lepas dari whuzpay-pg.
-- qris_configured_at dicatat HANYA saat upload QRIS lewat wizard
-- onboarding ini berhasil -- kalau merchant upload manual langsung di
-- gopay-notifications belakangan, kolom ini TIDAK ikut ter-update (tidak
-- ada sesi gopay-notifications yang hidup untuk mendeteksinya). Ini
-- keterbatasan yang diterima, bukan bug -- pengecekan qris_not_configured
-- yang sungguhan tetap terjadi live di gopay-notifications saat invoice
-- dibuat, kolom ini cuma untuk tampilan status di Settings whuzpay-pg.

ALTER TABLE merchant_gopay_credentials
    ADD COLUMN gopay_username TEXT,
    ADD COLUMN qris_configured_at TIMESTAMPTZ;

COMMENT ON COLUMN merchant_gopay_credentials.gopay_username IS
    'Username akun gopay-notifications (dibuat manual atau otomatis lewat onboarding terpadu) -- ditampilkan di Settings, akun itu asli dan bisa dipakai login langsung ke gopay-notifications kapan saja.';
COMMENT ON COLUMN merchant_gopay_credentials.qris_configured_at IS
    'Diisi hanya saat QRIS diupload lewat wizard onboarding whuzpay-pg. NULL tidak selalu berarti QRIS belum ada di gopay-notifications -- bisa saja diupload manual langsung di sana, di luar sepengetahuan whuzpay-pg.';
```

- [ ] **Step 3: Modifikasi `GopayCredentials` struct dan `Get`**

Ganti seluruh isi `whuzpay-pg/back/internal/repository/merchant_gopay_credentials_repository.go` menjadi:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrGopayCredentialsNotFound = errors.New("gopay credentials not found")

// GopayCredentials -- semua field independen (bisa salah satu/beberapa
// nil kalau belum diisi/dicapai). Username dan QRISConfiguredAt ditambah
// untuk onboarding terpadu, lihat spec
// 2026-09-17-whuzpay-pg-unified-onboarding-design.md.
type GopayCredentials struct {
	MerchantID       uuid.UUID
	APIKey           *string
	WebhookSecret    *string
	Username         *string
	QRISConfiguredAt *time.Time
}

type MerchantGopayCredentialsRepository struct {
	db *sql.DB
}

func NewMerchantGopayCredentialsRepository(db *sql.DB) *MerchantGopayCredentialsRepository {
	return &MerchantGopayCredentialsRepository{db: db}
}

func (r *MerchantGopayCredentialsRepository) Get(ctx context.Context, merchantID uuid.UUID) (*GopayCredentials, error) {
	c := &GopayCredentials{MerchantID: merchantID}
	query := `SELECT api_key, webhook_secret, gopay_username, qris_configured_at FROM merchant_gopay_credentials WHERE merchant_id = $1`
	err := r.db.QueryRowContext(ctx, query, merchantID).Scan(&c.APIKey, &c.WebhookSecret, &c.Username, &c.QRISConfiguredAt)
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
// (atau NULL kalau baris belum ada sama sekali). username ditambah untuk
// onboarding terpadu -- pemanggil lama (Settings PUT, lihat
// PaymentService.UpdateGopayCredentials) selalu mengirim nil di situ,
// artinya "jangan ubah", persis perilaku sebelum kolom ini ada.
func (r *MerchantGopayCredentialsRepository) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret, username *string) error {
	query := `
		INSERT INTO merchant_gopay_credentials (merchant_id, api_key, webhook_secret, gopay_username, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (merchant_id) DO UPDATE SET
			api_key = CASE WHEN $5 THEN merchant_gopay_credentials.api_key ELSE EXCLUDED.api_key END,
			webhook_secret = CASE WHEN $6 THEN merchant_gopay_credentials.webhook_secret ELSE EXCLUDED.webhook_secret END,
			gopay_username = CASE WHEN $7 THEN merchant_gopay_credentials.gopay_username ELSE EXCLUDED.gopay_username END,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, merchantID, apiKey, webhookSecret, username,
		apiKey == nil, webhookSecret == nil, username == nil)
	if err != nil {
		return fmt.Errorf("failed to upsert gopay credentials: %w", err)
	}
	return nil
}

// MarkQRISConfigured mencatat bahwa QRIS berhasil diupload lewat wizard
// onboarding (lihat internal/gopayonboard.Client.UploadQRISImage) --
// dipanggil TEPAT SEKALI setelah upload sukses, tidak pernah dipanggil
// untuk "unmark".
func (r *MerchantGopayCredentialsRepository) MarkQRISConfigured(ctx context.Context, merchantID uuid.UUID) error {
	query := `
		INSERT INTO merchant_gopay_credentials (merchant_id, qris_configured_at, updated_at)
		VALUES ($1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (merchant_id) DO UPDATE SET
			qris_configured_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, merchantID)
	if err != nil {
		return fmt.Errorf("failed to mark qris configured: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Build**

Run: `cd whuzpay-pg/back && go build ./...`
Expected: gagal di paket lain yang masih memanggil `Upsert` dengan 2 argumen (`internal/service/interfaces.go`, `internal/service/payment_service_test.go`) — itu diperbaiki di Task 3. Untuk sekarang cukup pastikan paket `internal/repository` sendiri compile:

Run: `cd whuzpay-pg/back && go build ./internal/repository/...`
Expected: bersih, tidak ada error.

- [ ] **Step 5: Commit**

```bash
git add whuzpay-pg/back/migrations/017_add_onboarding_columns_to_merchant_gopay_credentials.sql whuzpay-pg/back/internal/repository/merchant_gopay_credentials_repository.go
git commit -m "feat(whuzpay-pg): kolom gopay_username + qris_configured_at di merchant_gopay_credentials"
```

---

### Task 2: Package `internal/gopayonboard`

**Files:**
- Create: `whuzpay-pg/back/internal/gopayonboard/client.go`
- Create: `whuzpay-pg/back/internal/gopayonboard/errors.go`
- Create: `whuzpay-pg/back/internal/gopayonboard/username.go`
- Test: `whuzpay-pg/back/internal/gopayonboard/client_test.go`
- Test: `whuzpay-pg/back/internal/gopayonboard/username_test.go`

**Interfaces:**
- Konsumsi: tidak ada (paket independen, cuma `net/http` standar).
- Produces: `NewClient(baseURL, publicURL string) (*Client, error)`, `(*Client).SignUp(ctx, businessName, email, username, password string) error`, `(*Client).CreateAPIKey(ctx, name string) (key string, err error)`, `(*Client).CreateWebhook(ctx, name, url string, events []string) (secret string, err error)`, `(*Client).CreateDevice(ctx, name string) (deviceID, deviceSecret string, err error)`, `(*Client).UploadQRISImage(ctx, imageBase64 string) error`, `(*Client).PublicBackendURL() string`, `ErrEmailTaken`, `ErrUsernameTaken` (sentinel errors), `SanitizeUsername(email string) string`. Dipakai Task 5.

- [ ] **Step 1: Tulis test username generator dulu (TDD, tidak butuh HTTP)**

```go
package gopayonboard

import "testing"

func TestSanitizeUsername(t *testing.T) {
	cases := map[string]string{
		"budi@toko.com":      "budi",
		"Budi.Santoso@x.co":  "budisantoso",
		"a+b@x.co":           "ab",
		"@x.co":              "merchant",
		"":                   "merchant",
		"12345@x.co":         "12345",
	}
	for in, want := range cases {
		if got := SanitizeUsername(in); got != want {
			t.Errorf("SanitizeUsername(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeUsername_DipotongMaks20Karakter(t *testing.T) {
	got := SanitizeUsername("initigapanjangbangetlebihdari20karakter@x.co")
	if len(got) > 20 {
		t.Errorf("panjang = %d, mau maks 20", len(got))
	}
}
```

- [ ] **Step 2: Jalankan, pastikan gagal (fungsi belum ada)**

Run: `cd whuzpay-pg/back && go test ./internal/gopayonboard/... -run TestSanitizeUsername -v`
Expected: FAIL — `undefined: SanitizeUsername`

- [ ] **Step 3: Implementasi `username.go`**

```go
package gopayonboard

import "strings"

// SanitizeUsername menurunkan username dari bagian sebelum "@" di email --
// huruf kecil, cuma huruf+angka, dipotong maks 20 karakter. Dipakai
// AuthService.RegisterMerchant sebagai basis username akun gopay-notifications
// yang dibuat otomatis (lihat spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.1).
// Kalau bentrok 409 username_taken, pemanggil menambah akhiran angka --
// fungsi ini sendiri tidak tahu apa pun soal collision.
func SanitizeUsername(email string) string {
	local, _, _ := strings.Cut(email, "@")
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	sanitized := b.String()
	if sanitized == "" {
		sanitized = "merchant"
	}
	if len(sanitized) > 20 {
		sanitized = sanitized[:20]
	}
	return sanitized
}
```

- [ ] **Step 4: Jalankan lagi, pastikan lulus**

Run: `cd whuzpay-pg/back && go test ./internal/gopayonboard/... -run TestSanitizeUsername -v`
Expected: PASS, 2 test.

- [ ] **Step 5: Tulis `errors.go`**

```go
package gopayonboard

import "errors"

// ErrEmailTaken/ErrUsernameTaken -- sentinel yang dipetakan dari respons
// 409 gopay-notifications ("error":"email_taken"/"username_taken", lihat
// backend/internal/httpapi/signup.go). AuthService.RegisterMerchant
// memakai ini untuk memutuskan apakah retry username ada gunanya
// (ErrUsernameTaken -- ya) atau tidak (ErrEmailTaken -- tidak, email tidak
// berubah antar percobaan).
var (
	ErrEmailTaken    = errors.New("gopayonboard: email sudah terdaftar di gopay-notifications")
	ErrUsernameTaken = errors.New("gopayonboard: username sudah dipakai di gopay-notifications")
)
```

- [ ] **Step 6: Tulis test client (httptest, meniru pola `internal/provider/gopay/adapter_test.go`)**

```go
package gopayonboard

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignUp_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/signup" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi-palsu"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123"); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
}

func TestSignUp_EmailTaken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"success":false,"error":"email_taken","message":"email sudah dipakai"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestSignUp_UsernameTaken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"success":false,"error":"username_taken","message":"username sudah dipakai"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("err = %v, want ErrUsernameTaken", err)
	}
}

func TestCreateAPIKey_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/api-keys" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"key_1","name":"whuzpay-pg","created_at":"2026-09-17T00:00:00Z","key":"sk_abc123"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	key, err := c.CreateAPIKey(t.Context(), "whuzpay-pg")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if key != "sk_abc123" {
		t.Errorf("key = %q, want sk_abc123", key)
	}
}

func TestCreateWebhook_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/webhooks" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://pg.whuzpay.com/api/v1/provider-webhooks/gopay" {
			t.Errorf("url = %v", body["url"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"wh_1","name":"whuzpay-pg","url":"https://pg.whuzpay.com/api/v1/provider-webhooks/gopay","events":["invoice.paid","invoice.expired"],"enabled":true,"created_at":"2026-09-17T00:00:00Z","secret":"whsec_abc123"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	secret, err := c.CreateWebhook(t.Context(), "whuzpay-pg", "https://pg.whuzpay.com/api/v1/provider-webhooks/gopay", []string{"invoice.paid", "invoice.expired"})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if secret != "whsec_abc123" {
		t.Errorf("secret = %q, want whsec_abc123", secret)
	}
}

func TestCreateDevice_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/devices" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"device_id":"dev_abc123","device_secret":"c2VjcmV0"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, "https://whuzpay.com")
	deviceID, deviceSecret, err := c.CreateDevice(t.Context(), "Toko Budi - Bridge")
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if deviceID != "dev_abc123" || deviceSecret != "c2VjcmV0" {
		t.Errorf("got %q/%q", deviceID, deviceSecret)
	}
	if got := c.PublicBackendURL(); got != "https://whuzpay.com/api/v1" {
		t.Errorf("PublicBackendURL() = %q", got)
	}
}

func TestUploadQRISImage_Success(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a} // PNG magic bytes cukup untuk http.DetectContentType
	b64 := base64.StdEncoding.EncodeToString(png)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/account/qris-image" || r.Method != http.MethodPut {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		if body["content_type"] != "image/png" {
			t.Errorf("content_type = %q, want image/png", body["content_type"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	if err := c.UploadQRISImage(t.Context(), b64); err != nil {
		t.Fatalf("UploadQRISImage: %v", err)
	}
}

func TestSignUp_SesiDipakaiUlangUntukPanggilanBerikutnya(t *testing.T) {
	var sawCookieOnSecondCall bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/signup":
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi-palsu"})
			_, _ = w.Write([]byte(`{"success":true}`))
		case "/api/v1/admin/api-keys":
			if c, err := r.Cookie("admin_session"); err == nil && c.Value == "sesi-palsu" {
				sawCookieOnSecondCall = true
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"key":"sk_abc"}`))
		}
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, srv.URL)
	if err := c.SignUp(t.Context(), "Toko Budi", "budi@toko.com", "budi", "password123"); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
	if _, err := c.CreateAPIKey(t.Context(), "whuzpay-pg"); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if !sawCookieOnSecondCall {
		t.Error("cookie sesi dari SignUp tidak terbawa ke CreateAPIKey -- cookie jar tidak jalan")
	}
}
```

- [ ] **Step 7: Jalankan, pastikan gagal (implementasi belum ada)**

Run: `cd whuzpay-pg/back && go test ./internal/gopayonboard/... -v 2>&1 | head -20`
Expected: FAIL — `undefined: NewClient` dkk.

- [ ] **Step 8: Implementasi `client.go`**

```go
package gopayonboard

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// Client memanggil endpoint session-based gopay-notifications untuk
// menyediakan (provisioning) akun, API key, webhook, device, dan QRIS
// sekali di awal saat merchant baru daftar di whuzpay-pg (lihat spec
// 2026-09-17-whuzpay-pg-unified-onboarding-design.md). SELALU dibuat baru
// per registrasi lewat NewClient -- cookie jar-nya TIDAK PERNAH dibagi
// antar request/merchant, dan tidak pernah ditulis ke database. Kalau
// dibuat sekali lalu dipakai ulang untuk banyak merchant, sesi satu
// merchant bisa bocor ke merchant lain (satu jar = satu domain = satu set
// cookie, dan seluruh panggilan mengarah ke domain gopay-notifications
// yang sama).
//
// Terpisah dari internal/provider/gopay.Adapter: itu memanggil
// POST /invoices per transaksi pakai API key merchant; ini memanggil
// /admin/* pakai sesi, sekali saja, di titik yang berbeda dalam siklus
// hidup akun.
type Client struct {
	baseURL    string // loopback di produksi (GOPAY_BASE_URL)
	publicURL  string // publik (GOPAY_PUBLIC_BASE_URL) -- dipakai QR pairing, HP tidak bisa mengakses loopback
	httpClient *http.Client
}

func NewClient(baseURL, publicURL string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("gopayonboard: buat cookie jar: %w", err)
	}
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		publicURL: strings.TrimRight(publicURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		},
	}, nil
}

// PublicBackendURL adalah nilai backend_url yang WAJIB ditaruh di payload
// QR pairing -- harus URL publik (https://whuzpay.com/api/v1), bukan
// baseURL (bisa loopback http://127.0.0.1:8080 di produksi) karena HP
// tidak bisa mengakses loopback VPS.
func (c *Client) PublicBackendURL() string {
	return c.publicURL + "/api/v1"
}

type apiErrorBody struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (c *Client) SignUp(ctx context.Context, businessName, email, username, password string) error {
	body, err := json.Marshal(map[string]string{
		"business_name": businessName,
		"email":         email,
		"username":      username,
		"password":      password,
	})
	if err != nil {
		return fmt.Errorf("gopayonboard: encode signup request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/signup", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gopayonboard: build signup request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gopayonboard: signup: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var apiErr apiErrorBody
	_ = json.Unmarshal(respBody, &apiErr)
	switch apiErr.Error {
	case "email_taken":
		return ErrEmailTaken
	case "username_taken":
		return ErrUsernameTaken
	}
	return fmt.Errorf("gopayonboard: signup status %d: %s", resp.StatusCode, string(respBody))
}

func (c *Client) CreateAPIKey(ctx context.Context, name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/api-keys", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gopayonboard: build create api key request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gopayonboard: create api key: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("gopayonboard: create api key status %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("gopayonboard: decode create api key response: %w", err)
	}
	return out.Key, nil
}

func (c *Client) CreateWebhook(ctx context.Context, name, url string, events []string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"name":   name,
		"url":    url,
		"events": events,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/webhooks", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gopayonboard: build create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gopayonboard: create webhook: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("gopayonboard: create webhook status %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("gopayonboard: decode create webhook response: %w", err)
	}
	return out.Secret, nil
}

func (c *Client) CreateDevice(ctx context.Context, name string) (deviceID, deviceSecret string, err error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/devices", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("gopayonboard: build create device request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("gopayonboard: create device: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("gopayonboard: create device status %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		DeviceID     string `json:"device_id"`
		DeviceSecret string `json:"device_secret"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", "", fmt.Errorf("gopayonboard: decode create device response: %w", err)
	}
	return out.DeviceID, out.DeviceSecret, nil
}

func (c *Client) UploadQRISImage(ctx context.Context, imageBase64 string) error {
	decoded, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return fmt.Errorf("gopayonboard: decode qris image: %w", err)
	}
	contentType := http.DetectContentType(decoded)

	body, _ := json.Marshal(map[string]string{
		"image_base64": imageBase64,
		"content_type": contentType,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/v1/admin/account/qris-image", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gopayonboard: build upload qris request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gopayonboard: upload qris: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gopayonboard: upload qris status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
```

Tambahkan `"context"` ke import list (dipakai di semua signature method).

- [ ] **Step 9: Jalankan lagi, pastikan semua lulus**

Run: `cd whuzpay-pg/back && go test ./internal/gopayonboard/... -v`
Expected: PASS, seluruh test (username + client).

- [ ] **Step 10: `go vet` bersih**

Run: `cd whuzpay-pg/back && go vet ./internal/gopayonboard/...`
Expected: tidak ada output.

- [ ] **Step 11: Commit**

```bash
git add whuzpay-pg/back/internal/gopayonboard
git commit -m "feat(whuzpay-pg): package gopayonboard -- klien provisioning otomatis ke gopay-notifications"
```

---

### Task 3: Perluas status kredensial (interfaces.go, PaymentService, handler, fake test)

**Files:**
- Modify: `whuzpay-pg/back/internal/service/interfaces.go`
- Modify: `whuzpay-pg/back/internal/service/payment_service.go`
- Modify: `whuzpay-pg/back/internal/service/payment_service_test.go`
- Modify: `whuzpay-pg/back/internal/handler/merchant_handler.go`
- Test: `whuzpay-pg/back/internal/handler/merchant_handler_test.go` (kalau ada test existing untuk `GetGopayCredentials`, cek dulu — sesuaikan; kalau belum ada, lewati, tidak wajib menambah test handler baru untuk perubahan field response murni)

**Interfaces:**
- Konsumsi: `Upsert`/`MarkQRISConfigured` dari Task 1.
- Produces: `PaymentService.GetGopayCredentialsStatus(ctx, merchantID) (apiKeySet, webhookSecretSet, qrisConfigured bool, username string, err error)` (SIGNATURE BERUBAH), `gopayCredentialsStatusResponse{APIKeyConfigured, WebhookSecretConfigured, QRISConfigured bool; GopayUsername string}`. Dipakai Task 7 (frontend `GopayCredentialsStatus` type).

- [ ] **Step 1: Perluas interface di `interfaces.go`**

Ganti:

```go
type gopayCredentialsRepository interface {
	Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error)
	Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error
}
```

Menjadi:

```go
type gopayCredentialsRepository interface {
	Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error)
	Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret, username *string) error
	MarkQRISConfigured(ctx context.Context, merchantID uuid.UUID) error
}
```

- [ ] **Step 2: Perluas `PaymentService.GetGopayCredentialsStatus`/`UpdateGopayCredentials` di `payment_service.go`**

Ganti:

```go
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

Menjadi:

```go
// GetGopayCredentialsStatus TIDAK PERNAH mengembalikan nilai kredensial
// asli -- cuma status terisi/tidak, pola sama smtp_password_set gopay-notifications.
// username dan qrisConfigured ditambah untuk onboarding terpadu (lihat
// spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md).
func (s *PaymentService) GetGopayCredentialsStatus(ctx context.Context, merchantID uuid.UUID) (apiKeySet, webhookSecretSet, qrisConfigured bool, username string, err error) {
	creds, err := s.gopayCredentialsRepo.Get(ctx, merchantID)
	if errors.Is(err, repository.ErrGopayCredentialsNotFound) {
		return false, false, false, "", nil
	}
	if err != nil {
		return false, false, false, "", err
	}
	if creds.Username != nil {
		username = *creds.Username
	}
	return creds.APIKey != nil && *creds.APIKey != "",
		creds.WebhookSecret != nil && *creds.WebhookSecret != "",
		creds.QRISConfiguredAt != nil,
		username, nil
}

// UpdateGopayCredentials -- tri-state per field: nil = biarkan, ""=hapus,
// isi=ganti (pola sama NotificationSettings gopay-notifications). username
// selalu nil di sini -- field itu HANYA diisi oleh cascade onboarding
// otomatis (AuthService.RegisterMerchant), bukan lewat form Settings ini.
func (s *PaymentService) UpdateGopayCredentials(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error {
	return s.gopayCredentialsRepo.Upsert(ctx, merchantID, apiKey, webhookSecret, nil)
}
```

- [ ] **Step 3: Perbaiki `fakeGopayCredsRepo` dan test yang sudah ada di `payment_service_test.go`**

Ganti:

```go
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
```

Menjadi:

```go
func (f *fakeGopayCredsRepo) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret, username *string) error {
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
	if username != nil {
		existing.Username = username
	}
	f.creds[merchantID] = existing
	return nil
}

func (f *fakeGopayCredsRepo) MarkQRISConfigured(ctx context.Context, merchantID uuid.UUID) error {
	existing, ok := f.creds[merchantID]
	if !ok {
		existing = &repository.GopayCredentials{MerchantID: merchantID}
	}
	now := time.Now()
	existing.QRISConfiguredAt = &now
	f.creds[merchantID] = existing
	return nil
}
```

(Tambahkan `"time"` ke import `payment_service_test.go` kalau belum ada.)

Lalu perbaiki pemanggil `GetGopayCredentialsStatus` yang sudah ada (cari `apiKeySet, webhookSet, err :=` di file test ini) supaya menampung 5 nilai balik, bukan 3 — pola: `apiKeySet, webhookSet, _, _, err := svc.GetGopayCredentialsStatus(...)`.

- [ ] **Step 4: Perluas response handler di `merchant_handler.go`**

Ganti:

```go
type gopayCredentialsStatusResponse struct {
	APIKeyConfigured        bool `json:"api_key_configured"`
	WebhookSecretConfigured bool `json:"webhook_secret_configured"`
}
```

Menjadi:

```go
type gopayCredentialsStatusResponse struct {
	APIKeyConfigured        bool   `json:"api_key_configured"`
	WebhookSecretConfigured bool   `json:"webhook_secret_configured"`
	QRISConfigured          bool   `json:"qris_configured"`
	GopayUsername           string `json:"gopay_username,omitempty"`
}
```

Lalu di `GetGopayCredentials`, ganti:

```go
	apiKeySet, webhookSet, err := h.paymentService.GetGopayCredentialsStatus(r.Context(), merchantID)
	if err != nil {
		logger.ErrorfCtx(r.Context(), "Failed to get gopay credentials status for merchant %s: %v", merchantID, err)
		respondError(w, http.StatusInternalServerError, "Failed to load gopay credentials")
		return
	}
	respondJSON(w, http.StatusOK, gopayCredentialsStatusResponse{
		APIKeyConfigured: apiKeySet, WebhookSecretConfigured: webhookSet,
	})
```

Menjadi:

```go
	apiKeySet, webhookSet, qrisConfigured, username, err := h.paymentService.GetGopayCredentialsStatus(r.Context(), merchantID)
	if err != nil {
		logger.ErrorfCtx(r.Context(), "Failed to get gopay credentials status for merchant %s: %v", merchantID, err)
		respondError(w, http.StatusInternalServerError, "Failed to load gopay credentials")
		return
	}
	respondJSON(w, http.StatusOK, gopayCredentialsStatusResponse{
		APIKeyConfigured: apiKeySet, WebhookSecretConfigured: webhookSet,
		QRISConfigured: qrisConfigured, GopayUsername: username,
	})
```

Lalu di `UpdateGopayCredentials`, ganti:

```go
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

Menjadi:

```go
	apiKeySet, webhookSet, qrisConfigured, username, err := h.paymentService.GetGopayCredentialsStatus(r.Context(), merchantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load gopay credentials")
		return
	}
	respondJSON(w, http.StatusOK, gopayCredentialsStatusResponse{
		APIKeyConfigured: apiKeySet, WebhookSecretConfigured: webhookSet,
		QRISConfigured: qrisConfigured, GopayUsername: username,
	})
}
```

- [ ] **Step 5: Build + test seluruh backend**

Run: `cd whuzpay-pg/back && go build ./... && go vet ./...`
Expected: bersih.

Run: `cd whuzpay-pg/back && go test ./internal/service/... ./internal/handler/... -v 2>&1 | tail -40`
Expected: semua PASS.

- [ ] **Step 6: Commit**

```bash
git add whuzpay-pg/back/internal/service/interfaces.go whuzpay-pg/back/internal/service/payment_service.go whuzpay-pg/back/internal/service/payment_service_test.go whuzpay-pg/back/internal/handler/merchant_handler.go
git commit -m "feat(whuzpay-pg): status QRIS + username gopay-notifications di endpoint kredensial merchant"
```

---

### Task 4: Tipe domain — `RegisterRequest` + `RegisterMerchantResponse`

**Files:**
- Modify: `whuzpay-pg/back/internal/domain/merchant/user.go`

**Interfaces:**
- Produces: `RegisterRequest.QRISImageBase64 string` (field baru, opsional), `RegisterMerchantResponse{Token, TokenType string; ExpiresIn int64; User *UserResponse; GopayConnected bool; GopayUsername, GopayMessage string; GopayDevice *GopayDeviceInfo}`, `GopayDeviceInfo{DeviceID, DeviceSecret, BackendURL string}`. Dipakai Task 5 (AuthService), Task 6 (handler), Task 7 (frontend, field-field ini nama JSON-nya jadi acuan `merchant-auth.ts`).

- [ ] **Step 1: Tambah field `QRISImageBase64` ke `RegisterRequest`**

Ganti:

```go
type RegisterRequest struct {
	Name         string `json:"name"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone,omitempty"`
	Password     string `json:"password"`
}
```

Menjadi:

```go
type RegisterRequest struct {
	Name         string `json:"name"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone,omitempty"`
	Password     string `json:"password"`
	// QRISImageBase64 opsional -- lihat spec
	// 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.2. Kalau diisi,
	// diteruskan ke gopay-notifications selagi sesi dari langkah signup
	// (internal/gopayonboard) masih hidup di request yang sama. Validasi
	// ukuran/tipe gambar terjadi di gopay-notifications sendiri saat
	// diteruskan -- tidak divalidasi lagi di sini.
	QRISImageBase64 string `json:"qris_image_base64,omitempty"`
}
```

`Validate()` TIDAK berubah — field ini opsional, tidak ada aturan wajib.

- [ ] **Step 2: Tambah `RegisterMerchantResponse` dan `GopayDeviceInfo`**

Tambahkan di akhir file (setelah `ToUserResponse`):

```go
// GopayDeviceInfo -- device_secret di sini SENGAJA TIDAK PERNAH disimpan
// ke database whuzpay-pg. Field ini cuma ada di response registrasi
// SEKALI, dipegang state React di frontend untuk dirender jadi QR di
// layar "Pasangkan HP" (lihat spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.1/§6).
type GopayDeviceInfo struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	BackendURL   string `json:"backend_url"`
}

// RegisterMerchantResponse -- gabungan hasil login (Token dkk, SELALU ada
// karena akun whuzpay-pg selalu berhasil dibuat) dan hasil cascade
// onboarding gopay-notifications (GopayConnected dkk, best-effort --
// lihat AuthService.RegisterMerchant). GopayMessage diisi HANYA kalau ada
// bagian cascade yang gagal/dilewati, untuk ditampilkan ke merchant
// sebagai penjelasan kenapa sebagian belum otomatis tersambung.
type RegisterMerchantResponse struct {
	Token          string           `json:"token"`
	TokenType      string           `json:"token_type"`
	ExpiresIn      int64            `json:"expires_in"`
	User           *UserResponse    `json:"user"`
	GopayConnected bool             `json:"gopay_connected"`
	GopayUsername  string           `json:"gopay_username,omitempty"`
	GopayMessage   string           `json:"gopay_message,omitempty"`
	GopayDevice    *GopayDeviceInfo `json:"gopay_device,omitempty"`
}
```

- [ ] **Step 3: Build**

Run: `cd whuzpay-pg/back && go build ./internal/domain/merchant/...`
Expected: bersih (tidak ada pemanggil yang perlu diperbarui di task ini — itu Task 5/6).

- [ ] **Step 4: Commit**

```bash
git add whuzpay-pg/back/internal/domain/merchant/user.go
git commit -m "feat(whuzpay-pg): tipe RegisterMerchantResponse + QRISImageBase64 opsional"
```

---

### Task 5: `AuthService.RegisterMerchant` — auto-login + cascade onboarding

**Files:**
- Modify: `whuzpay-pg/back/internal/service/auth_service.go`
- Test: `whuzpay-pg/back/internal/service/auth_service_test.go`

**Interfaces:**
- Konsumsi: `gopayonboard.NewClient`, `gopayonboard.SanitizeUsername`, `gopayonboard.ErrEmailTaken`/`ErrUsernameTaken` (Task 2); `gopayCredentialsRepository.Upsert`/`MarkQRISConfigured` (Task 1, 3); `merchant.RegisterMerchantResponse`/`GopayDeviceInfo` (Task 4).
- Produces: `AuthService.RegisterMerchant(ctx, req) (*merchant.RegisterMerchantResponse, error)` (SIGNATURE BERUBAH dari `*merchant.MerchantResponse`), `AuthService.WithGopayOnboarding(gopayBaseURL, gopayPublicURL string, repo gopayCredentialsRepository) *AuthService`. Dipakai Task 6 (handler + main.go).

- [ ] **Step 1: Tulis test cascade dulu (TDD) — tambahkan di `auth_service_test.go`**

Cek dulu pola `newTestAuthService`/fake repo yang sudah ada di file ini (`authMerchantRepository`/`authMerchantUserRepository` fake) sebelum menulis test ini, supaya konsisten. Tambahkan:

```go
func TestRegisterMerchant_CascadeSuksesPenuh(t *testing.T) {
	var uploadedQRIS bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/signup":
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi"})
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.URL.Path == "/api/v1/admin/api-keys":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"key":"sk_abc"}`))
		case r.URL.Path == "/api/v1/admin/webhooks":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"secret":"whsec_abc"}`))
		case r.URL.Path == "/api/v1/admin/devices":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"device_id":"dev_abc","device_secret":"c2VjcmV0"}`))
		case r.URL.Path == "/api/v1/admin/account/qris-image":
			uploadedQRIS = true
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	svc := newTestAuthServiceForRegister(t)
	svc.WithGopayOnboarding(srv.URL, srv.URL, newFakeGopayCredsRepo())

	pngB64 := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	resp, err := svc.RegisterMerchant(context.Background(), &merchant.RegisterRequest{
		Name: "Budi", BusinessName: "Toko Budi", Email: "budi@toko.com",
		Password: "password123", QRISImageBase64: pngB64,
	})
	if err != nil {
		t.Fatalf("RegisterMerchant: %v", err)
	}
	if resp.Token == "" {
		t.Error("Token kosong -- auto-login seharusnya tetap terbit walau cascade gopay dijalankan")
	}
	if !resp.GopayConnected {
		t.Error("GopayConnected = false, mau true")
	}
	if resp.GopayUsername != "budi" {
		t.Errorf("GopayUsername = %q, mau budi", resp.GopayUsername)
	}
	if resp.GopayDevice == nil || resp.GopayDevice.DeviceID != "dev_abc" {
		t.Errorf("GopayDevice = %+v", resp.GopayDevice)
	}
	if resp.GopayDevice.BackendURL != srv.URL+"/api/v1" {
		t.Errorf("BackendURL = %q", resp.GopayDevice.BackendURL)
	}
	if !uploadedQRIS {
		t.Error("QRIS tidak pernah diupload")
	}
}

func TestRegisterMerchant_EmailTakenDiGopay_TetapBerhasilDenganFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"success":false,"error":"email_taken","message":"sudah dipakai"}`))
	}))
	defer srv.Close()

	svc := newTestAuthServiceForRegister(t)
	svc.WithGopayOnboarding(srv.URL, srv.URL, newFakeGopayCredsRepo())

	resp, err := svc.RegisterMerchant(context.Background(), &merchant.RegisterRequest{
		Name: "Budi", BusinessName: "Toko Budi", Email: "budi@toko.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("RegisterMerchant harus tetap sukses, dapat: %v", err)
	}
	if resp.Token == "" {
		t.Error("akun whuzpay-pg wajib tetap dibuat + auto-login walau gopay gagal")
	}
	if resp.GopayConnected {
		t.Error("GopayConnected = true, mau false")
	}
	if resp.GopayMessage == "" {
		t.Error("GopayMessage kosong, mau ada penjelasan fallback manual")
	}
}

func TestRegisterMerchant_GopayTidakBisaDihubungi_TetapBerhasil(t *testing.T) {
	svc := newTestAuthServiceForRegister(t)
	// Sengaja arahkan ke port yang tidak ada listener-nya sama sekali.
	svc.WithGopayOnboarding("http://127.0.0.1:1", "http://127.0.0.1:1", newFakeGopayCredsRepo())

	resp, err := svc.RegisterMerchant(context.Background(), &merchant.RegisterRequest{
		Name: "Budi", BusinessName: "Toko Budi", Email: "budi2@toko.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("RegisterMerchant harus tetap sukses, dapat: %v", err)
	}
	if resp.Token == "" || resp.GopayConnected {
		t.Errorf("resp = %+v", resp)
	}
}

func TestRegisterMerchant_GagalSebagian_ApiKeyTersimpanWebhookKosong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/signup":
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "sesi"})
			_, _ = w.Write([]byte(`{"success":true}`))
		case "/api/v1/admin/api-keys":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"key":"sk_abc"}`))
		case "/api/v1/admin/webhooks":
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/v1/admin/devices":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	svc := newTestAuthServiceForRegister(t)
	credsRepo := newFakeGopayCredsRepo()
	svc.WithGopayOnboarding(srv.URL, srv.URL, credsRepo)

	resp, err := svc.RegisterMerchant(context.Background(), &merchant.RegisterRequest{
		Name: "Budi", BusinessName: "Toko Budi", Email: "budi3@toko.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("RegisterMerchant: %v", err)
	}
	if !resp.GopayConnected {
		t.Error("signup sukses, GopayConnected harus true walau langkah sesudahnya gagal sebagian")
	}
	if resp.GopayDevice != nil {
		t.Error("device gagal dibuat, GopayDevice harus nil")
	}

	// Kredensial API key yang berhasil harus tetap tersimpan meski webhook gagal.
	found := false
	for _, c := range credsRepo.creds {
		if c.APIKey != nil && *c.APIKey == "sk_abc" {
			found = true
			if c.WebhookSecret != nil {
				t.Error("webhook gagal dibuat, WebhookSecret seharusnya tetap nil")
			}
		}
	}
	if !found {
		t.Error("API key yang berhasil didapat tidak tersimpan")
	}
}
```

Tambahkan helper `newTestAuthServiceForRegister(t *testing.T) *AuthService` di file yang sama kalau belum ada helper serupa persis — bangun `AuthService` dengan fake `merchantRepo`/`merchantUserRepo` in-memory yang cukup untuk `RegisterMerchant` (create+get by email keduanya mengembalikan not-found sampai dibuat). Kalau file ini SUDAH punya helper pembangun `AuthService` untuk test registrasi yang ada sekarang (`TestRegisterMerchant_...` versi lama pasti sudah ada), REUSE helper itu, jangan bikin baru — cek isi file ini dulu sebelum menulis pastinya.

Tambahkan import `"encoding/base64"`, `"net/http"`, `"net/http/httptest"` ke `auth_service_test.go`.

- [ ] **Step 2: Jalankan, pastikan gagal (implementasi belum ada)**

Run: `cd whuzpay-pg/back && go test ./internal/service/... -run TestRegisterMerchant -v 2>&1 | head -30`
Expected: gagal kompilasi — `WithGopayOnboarding` belum ada, `RegisterMerchant` masih mengembalikan tipe lama.

- [ ] **Step 3: Refactor `signMerchantToken` + implementasi cascade di `auth_service.go`**

Tambahkan field baru ke struct `AuthService`:

```go
type AuthService struct {
	adminRepo            authAdminRepository
	merchantUserRepo     authMerchantUserRepository
	merchantRepo         authMerchantRepository
	gopayCredentialsRepo gopayCredentialsRepository
	gopayBaseURL         string
	gopayPublicURL       string
	jwtSecret            []byte
}
```

Tambahkan chain method baru (setelah `WithMerchantAuth`):

```go
// WithGopayOnboarding mengaktifkan cascade onboarding otomatis (lihat
// tryConnectGopay) -- gopayBaseURL dan gopayPublicURL BEDA secara
// sengaja: gopayBaseURL boleh loopback (mis. http://127.0.0.1:8080 di
// produksi, lebih cepat, satu mesin dengan gopay-notifications),
// gopayPublicURL WAJIB selalu URL publik (https://whuzpay.com) karena
// dipakai membangun backend_url di payload QR pairing yang harus bisa
// diakses HP lewat internet, bukan loopback VPS.
func (s *AuthService) WithGopayOnboarding(gopayBaseURL, gopayPublicURL string, repo gopayCredentialsRepository) *AuthService {
	s.gopayBaseURL = gopayBaseURL
	s.gopayPublicURL = gopayPublicURL
	s.gopayCredentialsRepo = repo
	return s
}
```

Ganti pembuatan token inline di `LoginMerchant` (blok `claims := MerchantClaims{...}` sampai `signed, err := token.SignedString(s.jwtSecret)`) dengan pemanggilan helper baru, lalu tambahkan helper itu dan cascade-nya. Cari blok ini di `LoginMerchant`:

```go
	now := time.Now().UTC()
	expiresAt := now.Add(merchantTokenTTL)
	claims := MerchantClaims{
		UserID:     u.ID,
		MerchantID: u.MerchantID,
		Email:      u.Email,
		Role:       u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    adminTokenIssuer,
			Audience:  []string{merchantTokenAudience},
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign merchant token: %w", err)
	}
```

Ganti jadi:

```go
	now := time.Now().UTC()
	signed, err := s.signMerchantToken(u, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign merchant token: %w", err)
	}
```

Lalu ganti seluruh fungsi `RegisterMerchant` yang sekarang:

```go
func (s *AuthService) RegisterMerchant(ctx context.Context, req *merchant.RegisterRequest) (*merchant.MerchantResponse, error) {
	if s.merchantRepo == nil || s.merchantUserRepo == nil {
		return nil, fmt.Errorf("merchant auth not configured")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	businessName := strings.TrimSpace(req.BusinessName)
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	if _, err := s.merchantRepo.GetByEmail(ctx, email); err == nil {
		return nil, merchant.ErrMerchantAlreadyExists
	} else if err != merchant.ErrMerchantNotFound {
		return nil, err
	}
	if _, err := s.merchantUserRepo.GetByEmail(ctx, email); err == nil {
		return nil, merchant.ErrMerchantAlreadyExists
	} else if err != merchant.ErrMerchantUserNotFound {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	createdMerchant, err := s.merchantRepo.Create(ctx, &merchant.CreateMerchantRequest{
		Name:         name,
		Email:        email,
		Phone:        phone,
		BusinessName: businessName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create merchant: %w", err)
	}

	_, err = s.merchantUserRepo.Create(ctx, &merchant.User{
		MerchantID:   createdMerchant.ID,
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         "owner",
		IsActive:     true,
	})
	if err != nil {
		if delErr := s.merchantRepo.Delete(ctx, createdMerchant.ID); delErr != nil {
			logger.ErrorfCtx(ctx, "failed to roll back merchant %s after registration failure: %v", createdMerchant.ID, delErr)
		}
		return nil, fmt.Errorf("failed to create merchant owner account: %w", err)
	}

	return merchant.ToMerchantResponse(createdMerchant), nil
}
```

Menjadi:

```go
func (s *AuthService) RegisterMerchant(ctx context.Context, req *merchant.RegisterRequest) (*merchant.RegisterMerchantResponse, error) {
	if s.merchantRepo == nil || s.merchantUserRepo == nil {
		return nil, fmt.Errorf("merchant auth not configured")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	businessName := strings.TrimSpace(req.BusinessName)
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	if _, err := s.merchantRepo.GetByEmail(ctx, email); err == nil {
		return nil, merchant.ErrMerchantAlreadyExists
	} else if err != merchant.ErrMerchantNotFound {
		return nil, err
	}
	if _, err := s.merchantUserRepo.GetByEmail(ctx, email); err == nil {
		return nil, merchant.ErrMerchantAlreadyExists
	} else if err != merchant.ErrMerchantUserNotFound {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	createdMerchant, err := s.merchantRepo.Create(ctx, &merchant.CreateMerchantRequest{
		Name:         name,
		Email:        email,
		Phone:        phone,
		BusinessName: businessName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create merchant: %w", err)
	}

	createdUser, err := s.merchantUserRepo.Create(ctx, &merchant.User{
		MerchantID:   createdMerchant.ID,
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         "owner",
		IsActive:     true,
	})
	if err != nil {
		if delErr := s.merchantRepo.Delete(ctx, createdMerchant.ID); delErr != nil {
			logger.ErrorfCtx(ctx, "failed to roll back merchant %s after registration failure: %v", createdMerchant.ID, delErr)
		}
		return nil, fmt.Errorf("failed to create merchant owner account: %w", err)
	}

	now := time.Now().UTC()
	token, err := s.signMerchantToken(createdUser, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign merchant token: %w", err)
	}
	_ = s.merchantUserRepo.UpdateLastLoginAt(ctx, createdUser.ID, now)

	userResp := merchant.ToUserResponse(createdUser)
	userResp.BusinessName = businessName
	resp := &merchant.RegisterMerchantResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: int64(merchantTokenTTL.Seconds()),
		User:      userResp,
	}

	// Best-effort dari sini -- akun whuzpay-pg di atas SUDAH final. Tidak
	// ada apa pun di bawah ini yang boleh mengubah resp jadi error. Lihat
	// spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.3.
	s.tryConnectGopay(ctx, createdMerchant.ID, businessName, email, req.Password, req.QRISImageBase64, resp)

	return resp, nil
}

func (s *AuthService) signMerchantToken(u *merchant.User, now time.Time) (string, error) {
	expiresAt := now.Add(merchantTokenTTL)
	claims := MerchantClaims{
		UserID:     u.ID,
		MerchantID: u.MerchantID,
		Email:      u.Email,
		Role:       u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    adminTokenIssuer,
			Audience:  []string{merchantTokenAudience},
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// tryConnectGopay menjalankan cascade onboarding gopay-notifications
// (spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.1-§4.3).
// Best-effort murni -- setiap kegagalan dicatat di resp.GopayMessage,
// tidak pernah dikembalikan sebagai error ke pemanggil.
func (s *AuthService) tryConnectGopay(
	ctx context.Context,
	merchantID uuid.UUID,
	businessName, email, password, qrisImageBase64 string,
	resp *merchant.RegisterMerchantResponse,
) {
	if s.gopayCredentialsRepo == nil || s.gopayBaseURL == "" {
		return
	}

	client, err := gopayonboard.NewClient(s.gopayBaseURL, s.gopayPublicURL)
	if err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: buat client gagal untuk merchant %s: %v", merchantID, err)
		resp.GopayMessage = "Sedang ada gangguan menyambungkan otomatis, hubungkan manual lewat Settings."
		return
	}

	baseUsername := gopayonboard.SanitizeUsername(email)
	username := baseUsername
	var signErr error
	for attempt := 0; attempt < 4; attempt++ {
		candidate := baseUsername
		if attempt > 0 {
			candidate = fmt.Sprintf("%s%d", baseUsername, attempt)
		}
		signErr = client.SignUp(ctx, businessName, email, candidate, password)
		if signErr == nil {
			username = candidate
			break
		}
		if !errors.Is(signErr, gopayonboard.ErrUsernameTaken) {
			break // ErrEmailTaken atau error lain -- retry username tidak akan menolong
		}
	}
	if signErr != nil {
		if errors.Is(signErr, gopayonboard.ErrEmailTaken) {
			resp.GopayMessage = "Akun whuzpay-pg berhasil dibuat. Email ini sudah terdaftar di gopay-notifications -- hubungkan manual lewat Settings."
		} else {
			logger.ErrorfCtx(ctx, "gopay onboarding: signup gagal untuk merchant %s: %v", merchantID, signErr)
			resp.GopayMessage = "Akun whuzpay-pg berhasil dibuat. Sedang ada gangguan menyambungkan otomatis, hubungkan manual lewat Settings."
		}
		return
	}
	resp.GopayUsername = username
	resp.GopayConnected = true

	var apiKey, webhookSecret *string
	if key, err := client.CreateAPIKey(ctx, "whuzpay-pg"); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: create api key gagal untuk merchant %s: %v", merchantID, err)
		resp.GopayMessage = "Akun gopay-notifications tersambung, tapi API key gagal dibuat otomatis -- buat manual lewat Settings."
	} else {
		apiKey = &key
	}

	if secret, err := client.CreateWebhook(ctx, "whuzpay-pg",
		"https://pg.whuzpay.com/api/v1/provider-webhooks/gopay",
		[]string{"invoice.paid", "invoice.expired"}); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: create webhook gagal untuk merchant %s: %v", merchantID, err)
		if resp.GopayMessage == "" {
			resp.GopayMessage = "Akun gopay-notifications tersambung, tapi webhook gagal dibuat otomatis -- buat manual lewat Settings."
		}
	} else {
		webhookSecret = &secret
	}

	if err := s.gopayCredentialsRepo.Upsert(ctx, merchantID, apiKey, webhookSecret, &username); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: simpan kredensial gagal untuk merchant %s: %v", merchantID, err)
	}

	if deviceID, deviceSecret, err := client.CreateDevice(ctx, businessName+" - Bridge"); err != nil {
		logger.ErrorfCtx(ctx, "gopay onboarding: create device gagal untuk merchant %s: %v", merchantID, err)
	} else {
		resp.GopayDevice = &merchant.GopayDeviceInfo{
			DeviceID:     deviceID,
			DeviceSecret: deviceSecret,
			BackendURL:   client.PublicBackendURL(),
		}
	}

	if qrisImageBase64 != "" {
		if err := client.UploadQRISImage(ctx, qrisImageBase64); err != nil {
			logger.ErrorfCtx(ctx, "gopay onboarding: upload qris gagal untuk merchant %s: %v", merchantID, err)
		} else if err := s.gopayCredentialsRepo.MarkQRISConfigured(ctx, merchantID); err != nil {
			logger.ErrorfCtx(ctx, "gopay onboarding: tandai qris gagal untuk merchant %s: %v", merchantID, err)
		}
	}
}
```

Tambahkan import baru ke `auth_service.go`: `"errors"`, `"github.com/akbarryyan/pg-aggregator-back/internal/gopayonboard"`.

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `cd whuzpay-pg/back && go test ./internal/service/... -run TestRegisterMerchant -v`
Expected: PASS, 4 test baru + test `RegisterMerchant` lama (kalau ada) tetap lulus dengan tipe balik yang sudah disesuaikan.

- [ ] **Step 5: Jalankan seluruh suite backend**

Run: `cd whuzpay-pg/back && go build ./... && go vet ./... && go test ./... 2>&1 | tail -30`
Expected: bersih, semua PASS.

- [ ] **Step 6: Commit**

```bash
git add whuzpay-pg/back/internal/service/auth_service.go whuzpay-pg/back/internal/service/auth_service_test.go
git commit -m "feat(whuzpay-pg): RegisterMerchant auto-login + cascade onboarding gopay-notifications"
```

---

### Task 6: Handler, config, dan wiring `main.go`

**Files:**
- Modify: `whuzpay-pg/back/internal/handler/auth_handler.go`
- Modify: `whuzpay-pg/back/internal/config/config.go`
- Modify: `whuzpay-pg/back/cmd/api/main.go`
- Modify: `whuzpay-pg/back/.env.example`

**Interfaces:**
- Konsumsi: `AuthService.WithGopayOnboarding` (Task 5).
- Produces: `GopayConfig.PublicBaseURL string`, env var `GOPAY_PUBLIC_BASE_URL`. Dipakai deploy (Task 8, README).

- [ ] **Step 1: `auth_handler.go` -- tidak perlu logika baru, cuma pastikan `resp` (sekarang `*merchant.RegisterMerchantResponse`) tetap dikembalikan `201` apa adanya**

Cek `RegisterMerchant` handler yang sekarang -- barisnya `respondJSON(w, http.StatusCreated, resp)` sudah generik (tidak menyebut tipe `resp` secara eksplisit), jadi TIDAK ADA PERUBAHAN dibutuhkan di file ini untuk perubahan tipe response. Verifikasi ini dengan membaca `RegisterMerchant` di `auth_handler.go` -- kalau memang sudah begitu, lanjut ke Step 2 tanpa mengedit file ini sama sekali.

- [ ] **Step 2: Tambah `PublicBaseURL` ke `GopayConfig`**

Ganti:

```go
// GopayConfig -- BaseURL saja, tidak ada API key/secret global karena
// kredensial gopay-notifications disimpan per merchant (lihat
// internal/repository/merchant_gopay_credentials_repository.go).
type GopayConfig struct {
	BaseURL string
}
```

Menjadi:

```go
// GopayConfig -- BaseURL boleh loopback di produksi (http://127.0.0.1:8080,
// satu mesin dengan gopay-notifications, tidak perlu round-trip keluar).
// PublicBaseURL WAJIB SELALU url publik (https://whuzpay.com) -- dipakai
// membangun backend_url di payload QR pairing onboarding otomatis
// (internal/gopayonboard), yang harus bisa diakses HP lewat internet,
// bukan loopback VPS. Di dev lokal keduanya sama nilainya. Tidak ada API
// key/secret global karena kredensial gopay-notifications disimpan per
// merchant (lihat internal/repository/merchant_gopay_credentials_repository.go).
type GopayConfig struct {
	BaseURL       string
	PublicBaseURL string
}
```

Dan ganti:

```go
		Gopay: GopayConfig{
			BaseURL: getEnv("GOPAY_BASE_URL", "https://whuzpay.com"),
		},
```

Menjadi:

```go
		Gopay: GopayConfig{
			BaseURL:       getEnv("GOPAY_BASE_URL", "https://whuzpay.com"),
			PublicBaseURL: getEnv("GOPAY_PUBLIC_BASE_URL", "https://whuzpay.com"),
		},
```

- [ ] **Step 3: Wiring di `main.go`**

Cari baris:

```go
	authService := service.NewAuthService(adminRepo, cfg.Security.JWTSecret).
		WithMerchantAuth(merchantUserRepo, merchantRepo)
```

Ganti jadi:

```go
	authService := service.NewAuthService(adminRepo, cfg.Security.JWTSecret).
		WithMerchantAuth(merchantUserRepo, merchantRepo).
		WithGopayOnboarding(cfg.Gopay.BaseURL, cfg.Gopay.PublicBaseURL, gopayCredsRepo)
```

(`gopayCredsRepo` sudah dideklarasikan lebih awal di file yang sama, baris ~54 -- tidak perlu dibuat ulang.)

- [ ] **Step 4: `.env.example`**

Cari baris `GOPAY_BASE_URL=https://whuzpay.com`, tambahkan tepat di bawahnya:

```
GOPAY_PUBLIC_BASE_URL=https://whuzpay.com
```

- [ ] **Step 5: Build seluruh backend**

Run: `cd whuzpay-pg/back && go build ./... && go vet ./...`
Expected: bersih.

- [ ] **Step 6: Jalankan seluruh test backend**

Run: `cd whuzpay-pg/back && go test ./... 2>&1 | tail -30`
Expected: semua PASS.

- [ ] **Step 7: Commit**

```bash
git add whuzpay-pg/back/internal/config/config.go whuzpay-pg/back/cmd/api/main.go whuzpay-pg/back/.env.example
git commit -m "feat(whuzpay-pg): wiring cascade onboarding + GOPAY_PUBLIC_BASE_URL"
```

---

### Task 7: Frontend — form QRIS opsional, layar Pasangkan HP, status Settings

**Files:**
- Modify: `whuzpay-pg/front/lib/merchant-auth.ts`
- Modify: `whuzpay-pg/front/lib/merchant-api.ts`
- Modify: `whuzpay-pg/front/app/register/page.tsx`
- Create: `whuzpay-pg/front/app/pair-device/page.tsx`
- Modify: `whuzpay-pg/front/app/dashboard/settings/page.tsx`

**Interfaces:**
- Konsumsi: field JSON persis dari `merchant.RegisterMerchantResponse`/`GopayDeviceInfo` (Task 4) dan `gopayCredentialsStatusResponse` (Task 3).
- Produces: tidak ada yang dikonsumsi task lain (task terakhir yang menyentuh kode).

- [ ] **Step 1: `merchant-auth.ts` -- tipe response baru + `registerMerchant` mengembalikannya**

Ganti:

```ts
export type RegisteredMerchant = {
  id: string;
  name: string;
  email: string;
  phone: string;
  business_name: string;
  webhook_url?: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export async function registerMerchant(payload: {
  name: string;
  business_name: string;
  email: string;
  phone?: string;
  password: string;
}): Promise<RegisteredMerchant> {
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = (await res.json().catch(() => null)) as
    | RegisteredMerchant
    | ApiErrorBody
    | null;
  if (res.status === 409) {
    throw new Error("Email ini sudah terdaftar. Silakan masuk atau gunakan email lain.");
  }
  if (res.status === 429) {
    throw new Error("Terlalu banyak percobaan. Coba lagi beberapa saat lagi.");
  }
  if (!res.ok) {
    throw new Error("Gagal mendaftar. Periksa kembali data Anda.");
  }
  return body as RegisteredMerchant;
}
```

Menjadi:

```ts
export type GopayDeviceInfo = {
  device_id: string;
  device_secret: string;
  backend_url: string;
};

export type RegisterMerchantResponse = MerchantLoginResponse & {
  gopay_connected: boolean;
  gopay_username?: string;
  gopay_message?: string;
  gopay_device?: GopayDeviceInfo;
};

export async function registerMerchant(payload: {
  name: string;
  business_name: string;
  email: string;
  phone?: string;
  password: string;
  qris_image_base64?: string;
}): Promise<RegisterMerchantResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = (await res.json().catch(() => null)) as
    | RegisterMerchantResponse
    | ApiErrorBody
    | null;
  if (res.status === 409) {
    throw new Error("Email ini sudah terdaftar. Silakan masuk atau gunakan email lain.");
  }
  if (res.status === 429) {
    throw new Error("Terlalu banyak percobaan. Coba lagi beberapa saat lagi.");
  }
  if (!res.ok) {
    throw new Error("Gagal mendaftar. Periksa kembali data Anda.");
  }
  return body as RegisterMerchantResponse;
}
```

(`MerchantLoginResponse` sudah didefinisikan di atasnya di file yang sama -- `RegisterMerchantResponse` memakainya lewat intersection type persis field yang sama yang dikembalikan Go: `token`/`token_type`/`expires_in`/`user`.)

- [ ] **Step 2: `merchant-api.ts` -- perluas `GopayCredentialsStatus`**

Ganti:

```ts
export type GopayCredentialsStatus = {
  api_key_configured: boolean;
  webhook_secret_configured: boolean;
};
```

Menjadi:

```ts
export type GopayCredentialsStatus = {
  api_key_configured: boolean;
  webhook_secret_configured: boolean;
  qris_configured: boolean;
  gopay_username?: string;
};
```

- [ ] **Step 3: `register/page.tsx` -- field QRIS opsional, auto-login, redirect ke `/pair-device`**

Tambahkan import di atas (setelah `import { registerMerchant } from "@/lib/merchant-auth";`):

```tsx
import { saveMerchantSession } from "@/lib/merchant-auth";
```

Tambahkan state baru (setelah `const [confirmPassword, setConfirmPassword] = useState("");`):

```tsx
  const [qrisFile, setQrisFile] = useState<File | null>(null);
```

Tambahkan helper untuk baca file jadi base64 (sebelum `handleSubmit`):

```tsx
  function fileToBase64(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        // reader.result adalah data URL "data:image/png;base64,XXXX" --
        // backend cuma butuh bagian setelah koma.
        resolve(result.split(",")[1] ?? "");
      };
      reader.onerror = () => reject(new Error("Gagal membaca file gambar."));
      reader.readAsDataURL(file);
    });
  }
```

Ganti isi `handleSubmit` (bagian `setLoading(true); try { ... }`):

```tsx
    setLoading(true);
    try {
      const qrisBase64 = qrisFile ? await fileToBase64(qrisFile) : undefined;
      const result = await registerMerchant({
        name: name.trim(),
        business_name: businessName.trim(),
        email: email.trim(),
        phone: phone.trim() || undefined,
        password,
        qris_image_base64: qrisBase64,
      });
      saveMerchantSession(result);
      if (result.gopay_message) {
        toast(result.gopay_message, { icon: "ℹ️" });
      }
      if (result.gopay_device) {
        sessionStorage.setItem("gopay_pairing_device", JSON.stringify(result.gopay_device));
        router.replace("/pair-device");
      } else {
        toast.success("Pendaftaran berhasil.");
        router.replace("/dashboard");
      }
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Tidak bisa terhubung ke server. Coba lagi beberapa saat lagi.",
      );
      setLoading(false);
    }
```

Tambahkan input file di form, tepat sebelum blok `<div>` yang berisi label "Password" (antara field "No. Telepon" dan "Password"):

```tsx
            <div>
              <label htmlFor="qrisImage" className="block text-sm font-medium text-slate-700">
                Gambar QRIS <span className="text-slate-400">(opsional, bisa diisi belakangan)</span>
              </label>
              <input
                id="qrisImage"
                type="file"
                accept="image/png,image/jpeg"
                onChange={(e) => setQrisFile(e.target.files?.[0] ?? null)}
                className="mt-1.5 block w-full text-sm text-slate-600 file:mr-3 file:rounded-md file:border-0 file:bg-brand-navy file:px-3 file:py-2 file:text-sm file:font-medium file:text-white hover:file:bg-brand-navy-light"
              />
              <p className="mt-1 text-xs text-slate-400">
                Boleh dilewati sekarang -- bisa diisi kapan saja lewat Settings setelah masuk.
              </p>
            </div>
```

- [ ] **Step 4: Halaman baru `pair-device/page.tsx`**

```tsx
"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { GopayDeviceInfo } from "@/lib/merchant-auth";

export default function PairDevicePage() {
  const router = useRouter();
  const [device, setDevice] = useState<GopayDeviceInfo | null>(null);

  useEffect(() => {
    const raw = sessionStorage.getItem("gopay_pairing_device");
    if (!raw) {
      router.replace("/dashboard");
      return;
    }
    try {
      setDevice(JSON.parse(raw) as GopayDeviceInfo);
    } catch {
      router.replace("/dashboard");
    }
    // Sekali pakai -- kalau halaman ini di-refresh, QR tidak bisa
    // ditampilkan ulang (device secret memang sengaja tidak pernah
    // disimpan, lihat spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §6).
    sessionStorage.removeItem("gopay_pairing_device");
  }, [router]);

  if (!device) return null;

  const qrPayload = JSON.stringify({
    v: 1,
    backend_url: device.backend_url,
    device_id: device.device_id,
    device_secret: device.device_secret,
  });

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center gap-6 px-6 text-center">
      <h1 className="text-2xl font-extrabold text-brand-navy">Pasangkan HP kamu</h1>
      <p className="text-sm text-slate-500">
        Unduh aplikasi Android bridge, lalu pindai QR ini dari menu Pengaturan
        di aplikasi untuk menghubungkan HP dengan akun kamu.
      </p>

      <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={`https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${encodeURIComponent(qrPayload)}`}
          alt="QR pairing device"
          width={220}
          height={220}
        />
      </div>

      <p className="text-xs text-rose-500">
        QR ini memuat kunci perangkat -- jangan screenshot atau bagikan ke orang lain.
      </p>

      <a
        href="https://whuzpay.com/download-app"
        target="_blank"
        rel="noreferrer"
        className="rounded-md bg-brand-yellow px-6 py-3 text-sm font-bold uppercase tracking-wide text-brand-navy-dark hover:bg-brand-yellow-dark"
      >
        Unduh Aplikasi Android
      </a>

      <Link href="/dashboard" className="text-sm font-medium text-brand-navy hover:text-brand-navy-light">
        Lewati, lanjut ke dashboard →
      </Link>
    </main>
  );
}
```

Catatan: dipakai layanan publik `api.qrserver.com` untuk render QR (bukan library `react-qr-code` seperti `dashboard/` gopay-notifications) supaya tidak menambah dependency baru di `whuzpay-pg/front` untuk satu halaman ini. Kalau Akbar lebih suka konsisten dengan `dashboard/` (library lokal, tidak memanggil layanan pihak ketiga untuk merender data sensitif ini), ganti jadi `npm install react-qr-code` di `whuzpay-pg/front` dan pakai komponen `<QRCode value={qrPayload} size={220} />` sama seperti pola di `dashboard/src/app/(dashboard)/devices/page.tsx` -- **secara keamanan library lokal lebih baik** (data QR yang memuat device secret tidak perlu dikirim ke server pihak ketiga sama sekali untuk dirender), jadi task ini akan pakai opsi itu:

Ganti pendekatan di atas -- jalankan dulu:

```bash
cd whuzpay-pg/front && npm install react-qr-code
```

Lalu di `pair-device/page.tsx`, ganti import dan elemen `<img>`:

```tsx
import QRCode from "react-qr-code";
```

Dan ganti blok `<div className="rounded-xl ...">...</div>` menjadi:

```tsx
      <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        <QRCode value={qrPayload} size={220} />
      </div>
```

Hapus import `useState`... (tetap dipakai, jangan dihapus) -- cukup hapus baris `img` dan komentar eslint-disable yang menyertainya.

- [ ] **Step 5: Settings -- tampilkan username + status QRIS di `GopayCredentialsCard`**

Di `whuzpay-pg/front/app/dashboard/settings/page.tsx`, tambahkan tepat setelah blok `<ol>...</ol>` instruksi 4 langkah di dalam `GopayCredentialsCard` (sebelum `{loading ? (...) : (`):

```tsx
      {status?.gopay_username && (
        <p className="mt-3 text-[12.5px] text-[#6b7c93]">
          Akun gopay-notifications kamu: <span className="font-mono font-semibold">{status.gopay_username}</span>{" "}
          -- bisa dipakai login langsung ke{" "}
          <a href="https://whuzpay.com/login" target="_blank" rel="noreferrer" className="text-[#3b9eff] hover:underline">
            whuzpay.com
          </a>{" "}
          kapan saja.
        </p>
      )}
      {status && !status.qris_configured && (
        <p className="mt-2 rounded-lg bg-[#fff7e6] px-3 py-2 text-[12.5px] text-[#c27a00]">
          QRIS belum diisi -- pembayaran production akan ditolak sampai kamu{" "}
          <a href="https://whuzpay.com/settings" target="_blank" rel="noreferrer" className="font-semibold underline">
            upload QRIS langsung di gopay-notifications
          </a>
          .
        </p>
      )}
```

- [ ] **Step 6: Type-check, lint, build**

Run: `cd whuzpay-pg/front && npx tsc --noEmit && npx eslint . && npx next build`
Expected: tidak ada error, `/pair-device` muncul di daftar route hasil build.

- [ ] **Step 7: Commit**

```bash
git add whuzpay-pg/front/lib/merchant-auth.ts whuzpay-pg/front/lib/merchant-api.ts whuzpay-pg/front/app/register/page.tsx whuzpay-pg/front/app/pair-device whuzpay-pg/front/app/dashboard/settings/page.tsx whuzpay-pg/front/package.json whuzpay-pg/front/package-lock.json
git commit -m "feat(whuzpay-pg): form QRIS opsional, layar Pasangkan HP, status onboarding di Settings"
```

---

### Task 8: Verifikasi manual (`NEEDS-DEVICE`, Akbar menjalankan)

Tidak ada file kode diubah di task ini.

- [ ] **Step 1: Registrasi sungguhan tanpa QRIS**

Daftar lewat `/register` produksi (tanpa isi field QRIS). Login ke `whuzpay.com` (gopay-notifications) sebagai vendor atau cek langsung di database -- pastikan akun baru benar-benar dibuat di sana dengan `business_name`/`email` yang sama, dan API key + webhook endpoint (mengarah ke `pg.whuzpay.com/api/v1/provider-webhooks/gopay`) benar-benar ada.

- [ ] **Step 2: Registrasi sungguhan DENGAN QRIS**

Ulangi dengan email berbeda, isi field QRIS. Cek di gopay-notifications bahwa gambar QRIS akun itu benar-benar tersimpan.

- [ ] **Step 3: Layar Pasangkan HP**

Setelah registrasi (test manapun), pastikan diarahkan ke `/pair-device`, QR-nya valid -- scan pakai aplikasi Android (unduh dari `/download-app`), Test Connection sukses.

- [ ] **Step 4: Email sudah terdaftar di gopay-notifications**

Daftar ketiga kalinya pakai email PERSIS yang sama dengan test #1 (yang sudah punya akun gopay-notifications). Pastikan akun whuzpay-pg tetap berhasil dibuat, pesan fallback muncul, dan mengisi kredensial manual lewat Settings masih berfungsi seperti sebelumnya.

- [ ] **Step 5: Pembayaran sungguhan pakai kredensial otomatis**

Pakai akun dari test #2, buat payment (tombol "+"), bayar via GoPay sungguhan, pastikan alur end-to-end (yang sudah `PASS` di qa-report.md §28) tetap berfungsi lewat kredensial yang dibuat OTOMATIS ini.

- [ ] **Step 6: Update `docs/qa/qa-report.md`**

Tambahkan section baru untuk sub-project ini (pola sama section-section sebelumnya), tempel bukti dari Step 1-5.

---

## Self-review (sudah dilakukan penulis plan)

**Cakupan spec:** §4.1 (cascade signup→API key→webhook→device) → Task 5. §4.2 (form QRIS opsional, auto-login, layar Pasangkan HP) → Task 5 (backend) + Task 7 (frontend). §4.3 (penanganan gagal: email/username taken, unreachable, gagal sebagian, device gagal) → Task 5 (4 test skenario persis mengikuti §4.3). §4.4 (migrasi `gopay_username`) → Task 1 (plus `qris_configured_at` yang diperlukan Task 3/7 untuk menampilkan status QRIS, detail implementasi yang tidak eksplisit di spec tapi konsisten dengan "status QRIS (isi/belum)" di §4.2 spec). §5 pengujian → tiap task punya step test sendiri, Task 8 untuk NEEDS-DEVICE. §6 keputusan sengaja → tercermin di komentar kode Task 1, 2, 5 (sesi/device secret tidak disimpan, tidak ada rollback lintas sistem, endpoint publik tidak diubah).

**Placeholder scan:** tidak ada "TBD"/"tulis lengkap nanti" -- setiap step kode berisi isi lengkap siap tempel.

**Konsistensi tipe/nama:** `RegisterMerchantResponse`/`GopayDeviceInfo` (Task 4) dipakai identik di Task 5 (Go) dan Task 7 (field JSON yang sama persis di TypeScript: `gopay_connected`/`gopay_username`/`gopay_message`/`gopay_device`/`device_id`/`device_secret`/`backend_url`). `gopayCredentialsRepository.Upsert(ctx, merchantID, apiKey, webhookSecret, username *string)` konsisten dipakai di Task 1 (repository), Task 3 (interface + PaymentService, dengan `nil` untuk `username`), dan Task 5 (AuthService, dengan `&username` terisi). `MarkQRISConfigured` konsisten muncul di Task 1 (repository), Task 3 (interface + fake test), dan Task 5 (dipanggil di `tryConnectGopay`). Nama field response Settings (`qris_configured`/`gopay_username`) konsisten Task 3 (Go JSON tag) dan Task 7 (`GopayCredentialsStatus` TypeScript type).
