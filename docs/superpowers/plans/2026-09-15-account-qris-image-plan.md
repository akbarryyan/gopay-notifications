# QRIS Statis per Account — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Setiap account gopay-notifications bisa upload gambar QRIS statis miliknya sendiri, dan gambar itu otomatis disertakan tiap kali integrator (mis. whuzpay-pg) membuat invoice lewat API — tanpa itu, `POST /invoices` ditolak jelas, bukan diam-diam sukses tanpa cara bayar.

**Architecture:** Tabel baru `account_qris_images` (1:1 dengan `accounts`, bytea di Postgres — bukan filesystem/S3) diakses lewat layer store baru, diekspos lewat 3 endpoint self-service (`PUT`/`GET`/`DELETE /api/v1/admin/account/qris-image`) untuk Customer Dashboard, dan sebagai data URI di response `POST /api/v1/invoices` (API key) untuk integrator eksternal.

**Tech Stack:** Go 1.x + pgx + goose (backend), Next.js 16 + Tailwind v4 + shadcn/ui (dashboard, pola `useApiData`/`apiFetch`/`SettingsCard` yang sudah ada).

## Global Constraints

- Dua database (gopay-notifications dan whuzpay-pg) tetap terpisah total — tidak ada FK/query lintas database. Spec ini sepenuhnya di dalam database `gopay` gopay-notifications sendiri.
- Gambar disimpan sebagai `BYTEA` di Postgres, bukan filesystem/S3 — konsisten dengan backup `pg_dump`-only produksi yang sudah ada.
- Ukuran gambar dibatasi ≤ 300KB (setelah decode base64); tipe dibatasi PNG/JPEG, divalidasi dari ISI BYTE (`http.DetectContentType`), bukan dari field yang diklaim klien.
- `POST /api/v1/invoices` **menolak** (`409 qris_not_configured`) kalau account belum upload QRIS — invoice tidak dibuat sama sekali, bukan dibuat dengan `qris_image: null`.
- `GET /api/v1/invoices/{id}` (polling) TIDAK ikut mengembalikan `qris_image` — sengaja, sudah didapat sekali dari response create.
- Endpoint upload pakai body limit baru `maxQRISImageBodyBytes` (512 KiB) — BUKAN `maxBodyBytes` (64 KiB) yang dipakai hampir semua endpoint lain, karena base64 dari gambar 300KB sudah ~400KB.
- `make test` (backend) WAJIB `-p 1` — paket `store` dan `httpapi` sama-sama TRUNCATE database test yang sama.
- Tabel baru WAJIB ditambahkan ke DUA daftar TRUNCATE test yang sudah ada: `internal/store/store_test.go` dan `internal/httpapi/auth_middleware_test.go`.
- Setiap perubahan perilaku `POST`/`GET /invoices` WAJIB ikut memperbarui `dashboard/src/app/(dashboard)/api-docs/page.tsx` (aturan proyek, CLAUDE.md).
- Setiap milestone WAJIB memperbarui `docs/qa/qa-report.md` (PASS butuh output command yang ditempel, bukan pernyataan) sebelum dianggap selesai.
- Tidak menyentuh `whuzpay-pg/` sama sekali di plan ini.

---

### Task 1: Migrasi + layer store

**Files:**
- Create: `backend/migrations/00018_account_qris_images.sql`
- Modify: `backend/internal/store/activity.go` (tambah 2 konstanta aksi)
- Create: `backend/internal/store/qris_image.go`
- Test: `backend/internal/store/qris_image_test.go`
- Modify: `backend/internal/store/store_test.go` (tambah `account_qris_images` ke TRUNCATE)

**Interfaces:**
- Produces:
  ```go
  var ErrQRISImageNotFound = errors.New("store: qris image tidak ditemukan")

  type QRISImage struct {
      AccountID   string
      ImageData   []byte
      ContentType string
      UpdatedAt   time.Time
  }

  func (s *Store) UpsertQRISImage(ctx context.Context, accountID string, data []byte, contentType string) error
  func (s *Store) GetQRISImage(ctx context.Context, accountID string) (QRISImage, error) // ErrQRISImageNotFound kalau tidak ada
  func (s *Store) DeleteQRISImage(ctx context.Context, accountID string) error            // idempotent
  func (s *Store) HasQRISImage(ctx context.Context, accountID string) (bool, error)

  const ActivityQRISImageUpdated = "qris_image_updated"
  const ActivityQRISImageRemoved = "qris_image_removed"
  ```
  Task 2 dan Task 3 memanggil fungsi-fungsi ini langsung.

- [ ] **Step 1: Tulis migrasi**

`backend/migrations/00018_account_qris_images.sql`:

```sql
-- +goose Up
-- QRIS statis per account -- lihat spec
-- docs/superpowers/specs/2026-09-15-account-qris-image-design.md. Relasi
-- 1:1 murni dengan accounts: account_id sebagai primary key (bukan id
-- sendiri) karena tidak perlu riwayat/versi -- upload baru menimpa yang
-- lama lewat UPSERT.
CREATE TABLE account_qris_images (
    account_id   TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    image_data   BYTEA NOT NULL,
    content_type TEXT NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- account_activity_log (migrasi 00016) perlu dua aksi baru untuk upload/
-- hapus QRIS dari halaman Settings.
ALTER TABLE account_activity_log DROP CONSTRAINT account_activity_log_action_check;
ALTER TABLE account_activity_log ADD CONSTRAINT account_activity_log_action_check
    CHECK (action IN (
        'login_success', 'login_failed', 'password_changed', 'password_reset',
        'api_key_created', 'api_key_revoked', 'device_added', 'device_deleted',
        'qris_image_updated', 'qris_image_removed'
    ));

-- +goose Down
DELETE FROM account_activity_log WHERE action IN ('qris_image_updated', 'qris_image_removed');
ALTER TABLE account_activity_log DROP CONSTRAINT account_activity_log_action_check;
ALTER TABLE account_activity_log ADD CONSTRAINT account_activity_log_action_check
    CHECK (action IN (
        'login_success', 'login_failed', 'password_changed', 'password_reset',
        'api_key_created', 'api_key_revoked', 'device_added', 'device_deleted'
    ));
DROP TABLE account_qris_images;
```

- [ ] **Step 2: Jalankan migrasi ke database test**

```bash
cd backend && make db-up && make migrate
```
Expected: `OK 00018_account_qris_images.sql`, `successfully migrated database to version: 18`.

- [ ] **Step 3: Tambah konstanta aksi**

Di `backend/internal/store/activity.go`, ubah blok const:

```go
const (
	ActivityLoginSuccess    = "login_success"
	ActivityLoginFailed     = "login_failed"
	ActivityPasswordChanged = "password_changed"
	ActivityPasswordReset   = "password_reset"
	ActivityAPIKeyCreated   = "api_key_created"
	ActivityAPIKeyRevoked   = "api_key_revoked"
	ActivityDeviceAdded     = "device_added"
	ActivityDeviceDeleted   = "device_deleted"
	ActivityQRISImageUpdated = "qris_image_updated"
	ActivityQRISImageRemoved = "qris_image_removed"
)
```

- [ ] **Step 4: Tulis test store yang gagal dulu**

`backend/internal/store/qris_image_test.go`:

```go
package store_test

import (
	"context"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestUpsertDanGetQRISImage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := s.UpsertQRISImage(ctx, "acc_1", png, "image/png"); err != nil {
		t.Fatalf("UpsertQRISImage: %v", err)
	}

	img, err := s.GetQRISImage(ctx, "acc_1")
	if err != nil {
		t.Fatalf("GetQRISImage: %v", err)
	}
	if string(img.ImageData) != string(png) || img.ContentType != "image/png" || img.AccountID != "acc_1" {
		t.Fatalf("img = %+v, mau data/tipe/account cocok dengan yang di-upload", img)
	}
}

func TestUpsertQRISImageMenimpaBukanMenambah(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("gambar-lama"), "image/png"); err != nil {
		t.Fatalf("upsert pertama: %v", err)
	}
	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("gambar-baru"), "image/jpeg"); err != nil {
		t.Fatalf("upsert kedua: %v", err)
	}

	img, err := s.GetQRISImage(ctx, "acc_1")
	if err != nil {
		t.Fatalf("GetQRISImage: %v", err)
	}
	if string(img.ImageData) != "gambar-baru" || img.ContentType != "image/jpeg" {
		t.Fatalf("img = %+v, mau menimpa jadi gambar-baru/image/jpeg", img)
	}

	var count int
	if err := s.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM account_qris_images WHERE account_id = $1", "acc_1").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, mau 1 baris (upsert, bukan insert baru)", count)
	}
}

func TestGetQRISImageTidakAdaMengembalikanErrNotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	_, err := s.GetQRISImage(ctx, "acc_1")
	if err != store.ErrQRISImageNotFound {
		t.Fatalf("err = %v, mau ErrQRISImageNotFound", err)
	}
}

func TestHasQRISImage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	has, err := s.HasQRISImage(ctx, "acc_1")
	if err != nil || has {
		t.Fatalf("has = %v, err = %v, mau false sebelum upload", has, err)
	}

	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("x"), "image/png"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	has, err = s.HasQRISImage(ctx, "acc_1")
	if err != nil || !has {
		t.Fatalf("has = %v, err = %v, mau true setelah upload", has, err)
	}
}

func TestDeleteQRISImageIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	if err := s.DeleteQRISImage(ctx, "acc_1"); err != nil {
		t.Fatalf("delete pertama (belum ada): %v", err)
	}

	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("x"), "image/png"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := s.DeleteQRISImage(ctx, "acc_1"); err != nil {
		t.Fatalf("delete kedua: %v", err)
	}
	if err := s.DeleteQRISImage(ctx, "acc_1"); err != nil {
		t.Fatalf("delete ketiga (sudah tidak ada): %v", err)
	}

	if _, err := s.GetQRISImage(ctx, "acc_1"); err != store.ErrQRISImageNotFound {
		t.Fatalf("err = %v, mau ErrQRISImageNotFound setelah delete", err)
	}
}

func TestDeleteAccountIkutMenghapusQRISImage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("x"), "image/png"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if _, err := s.Pool().Exec(ctx, "DELETE FROM accounts WHERE id = $1", "acc_1"); err != nil {
		t.Fatalf("delete account: %v", err)
	}

	var count int
	if err := s.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM account_qris_images WHERE account_id = $1", "acc_1").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, mau 0 (ON DELETE CASCADE)", count)
	}
}
```

- [ ] **Step 5: Jalankan test, pastikan gagal (fungsi belum ada)**

```bash
cd backend && TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" go test ./internal/store/... -run QRISImage -v
```
Expected: FAIL, `undefined: store.UpsertQRISImage` dkk.

- [ ] **Step 6: Implementasi store**

`backend/internal/store/qris_image.go`:

```go
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrQRISImageNotFound dikembalikan bila account belum pernah upload QRIS.
var ErrQRISImageNotFound = errors.New("store: qris image tidak ditemukan")

// QRISImage adalah gambar QRIS statis milik satu account -- relasi 1:1,
// upload baru menimpa yang lama (lihat komentar migrasi 00018).
type QRISImage struct {
	AccountID   string
	ImageData   []byte
	ContentType string
	UpdatedAt   time.Time
}

// UpsertQRISImage menyimpan/mengganti gambar QRIS account ini.
func (s *Store) UpsertQRISImage(ctx context.Context, accountID string, data []byte, contentType string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO account_qris_images (account_id, image_data, content_type, updated_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (account_id) DO UPDATE
		 SET image_data = EXCLUDED.image_data, content_type = EXCLUDED.content_type, updated_at = now()`,
		accountID, data, contentType)
	if err != nil {
		return fmt.Errorf("store: upsert qris image: %w", err)
	}
	return nil
}

// GetQRISImage mengambil gambar QRIS account ini.
func (s *Store) GetQRISImage(ctx context.Context, accountID string) (QRISImage, error) {
	var img QRISImage
	img.AccountID = accountID
	err := s.pool.QueryRow(ctx,
		`SELECT image_data, content_type, updated_at FROM account_qris_images WHERE account_id = $1`,
		accountID).Scan(&img.ImageData, &img.ContentType, &img.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return QRISImage{}, ErrQRISImageNotFound
	}
	if err != nil {
		return QRISImage{}, fmt.Errorf("store: get qris image: %w", err)
	}
	return img, nil
}

// DeleteQRISImage menghapus gambar QRIS account ini. Idempotent -- tidak
// error kalau memang belum ada.
func (s *Store) DeleteQRISImage(ctx context.Context, accountID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM account_qris_images WHERE account_id = $1`, accountID)
	if err != nil {
		return fmt.Errorf("store: delete qris image: %w", err)
	}
	return nil
}

// HasQRISImage dipakai handleAdminGetAccount (status ringan, tanpa ambil
// gambar penuh) dan handleCreateInvoice (gerbang sebelum invoice dibuat).
func (s *Store) HasQRISImage(ctx context.Context, accountID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM account_qris_images WHERE account_id = $1)`, accountID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: cek qris image: %w", err)
	}
	return exists, nil
}
```

- [ ] **Step 7: Jalankan test lagi, pastikan lulus**

```bash
cd backend && TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" go test ./internal/store/... -run QRISImage -v
```
Expected: semua `PASS`.

- [ ] **Step 8: Tambah `account_qris_images` ke TRUNCATE `store_test.go`**

Di `backend/internal/store/store_test.go`, ubah baris TRUNCATE jadi:

```go
_, err = s.Pool().Exec(ctx,
    "TRUNCATE notification_events, event_reviews, invoices, api_keys, webhook_deliveries, webhook_endpoints, devices, accounts, vendor_admins, audit_log, notification_settings, notification_log, password_reset_tokens, telegram_link_codes, email_verification_tokens, account_activity_log, plans, account_qris_images RESTART IDENTITY CASCADE")
```

- [ ] **Step 9: `go build`, `go vet`, `gofmt -l .`, lalu commit**

```bash
cd backend
go build ./... && go vet ./... && gofmt -l .
git add migrations/00018_account_qris_images.sql internal/store/activity.go internal/store/qris_image.go internal/store/qris_image_test.go internal/store/store_test.go
git commit -m "feat(store): tabel dan layer store QRIS statis per account"
```

---

### Task 2: Endpoint self-service (Customer Dashboard) + status di profil

**Files:**
- Create: `backend/internal/httpapi/admin_qris_image.go`
- Test: `backend/internal/httpapi/admin_qris_image_test.go`
- Modify: `backend/internal/httpapi/admin_account_settings.go` (`accountProfileJSON` + `handleAdminGetAccount`)
- Modify: `backend/internal/httpapi/api.go` (routing)
- Modify: `backend/internal/httpapi/auth_middleware_test.go` (tambah `account_qris_images` ke TRUNCATE)

**Interfaces:**
- Consumes: `store.UpsertQRISImage`, `store.GetQRISImage`, `store.DeleteQRISImage`, `store.HasQRISImage`, `store.ErrQRISImageNotFound`, `store.ActivityQRISImageUpdated`, `store.ActivityQRISImageRemoved` (Task 1); `AccountFromContext`, `a.logActivity`, `writeJSON`, `a.writeError`, `maxBodyBytes`-style pattern dari `auth_middleware.go`.
- Produces: route `PUT`/`GET`/`DELETE /api/v1/admin/account/qris-image`; field baru `qris_image_configured` di `accountProfileJSON` (dipakai Task 4 di frontend).

- [ ] **Step 1: Tulis test HTTP yang gagal dulu**

`backend/internal/httpapi/admin_qris_image_test.go`:

```go
package httpapi_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// tinyPNG adalah PNG 1x1 valid (byte magic number PNG + IHDR minimal cukup
// untuk http.DetectContentType mengenalinya sebagai image/png -- tidak
// perlu gambar utuh untuk keperluan test).
var tinyPNGBase64 = base64.StdEncoding.EncodeToString([]byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
})

func adminPutBody(t *testing.T, h http.Handler, cookie *http.Cookie, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUploadQRISImageBerhasil(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	body := `{"image_base64":"` + tinyPNGBase64 + `","content_type":"image/png"}`
	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	get := adminGet(t, h, cookie, "/api/v1/admin/account/qris-image")
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d (body=%s)", get.Code, get.Body.String())
	}
	if ct := get.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q, mau image/png", ct)
	}
	wantBytes, _ := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if get.Body.String() != string(wantBytes) {
		t.Fatal("body GET tidak sama dengan yang di-upload")
	}
}

func TestUploadQRISImageBase64TidakValid(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image",
		`{"image_base64":"bukan-base64-!!!","content_type":"image/png"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct{ Error string `json:"error"` }
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "invalid_payload" {
		t.Fatalf("error = %q, mau invalid_payload", errBody.Error)
	}
}

func TestUploadQRISImageTerlaluBesar(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	big := make([]byte, 300*1024+1)
	bigB64 := base64.StdEncoding.EncodeToString(big)
	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image",
		`{"image_base64":"`+bigB64+`","content_type":"image/png"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct{ Error string `json:"error"` }
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "image_too_large" {
		t.Fatalf("error = %q, mau image_too_large", errBody.Error)
	}
}

func TestUploadQRISImageTipeTidakDidukung(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	// Klaim image/png tapi isinya teks biasa -- membuktikan validasi pakai
	// http.DetectContentType atas ISI byte, bukan field content_type yang
	// diklaim klien.
	textB64 := base64.StdEncoding.EncodeToString([]byte("ini bukan gambar sama sekali, cuma teks biasa"))
	rec := adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image",
		`{"image_base64":"`+textB64+`","content_type":"image/png"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct{ Error string `json:"error"` }
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "unsupported_image_type" {
		t.Fatalf("error = %q, mau unsupported_image_type", errBody.Error)
	}
}

func TestGetQRISImageBelumAdaMengembalikan404(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	rec := adminGet(t, h, cookie, "/api/v1/admin/account/qris-image")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestDeleteQRISImageIdempotenLewatHTTP(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	first := adminDelete(t, h, cookie, "/api/v1/admin/account/qris-image")
	if first.Code != http.StatusOK {
		t.Fatalf("delete pertama status = %d", first.Code)
	}

	body := `{"image_base64":"` + tinyPNGBase64 + `","content_type":"image/png"}`
	adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image", body)

	second := adminDelete(t, h, cookie, "/api/v1/admin/account/qris-image")
	if second.Code != http.StatusOK {
		t.Fatalf("delete kedua status = %d", second.Code)
	}
	get := adminGet(t, h, cookie, "/api/v1/admin/account/qris-image")
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET setelah delete status = %d, mau 404", get.Code)
	}
}

func TestQRISImageEndpointButuhSesi(t *testing.T) {
	h := newAPIWithAdmin(t)
	for _, tc := range []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPut, `{"image_base64":"x","content_type":"image/png"}`},
		{http.MethodDelete, ""},
	} {
		req := httptest.NewRequest(tc.method, "/api/v1/admin/account/qris-image", strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, mau 401 tanpa cookie", tc.method, rec.Code)
		}
	}
}

func TestGetAccountProfileMemuatQRISImageConfigured(t *testing.T) {
	h := newAPIWithAdmin(t)
	cookie := loginAsAdmin(t, h)

	before := adminGet(t, h, cookie, "/api/v1/admin/account")
	var beforeBody struct {
		Account struct {
			QRISImageConfigured bool `json:"qris_image_configured"`
		} `json:"account"`
	}
	json.Unmarshal(before.Body.Bytes(), &beforeBody)
	if beforeBody.Account.QRISImageConfigured {
		t.Fatal("qris_image_configured mau false sebelum upload")
	}

	body := `{"image_base64":"` + tinyPNGBase64 + `","content_type":"image/png"}`
	adminPutBody(t, h, cookie, "/api/v1/admin/account/qris-image", body)

	after := adminGet(t, h, cookie, "/api/v1/admin/account")
	var afterBody struct {
		Account struct {
			QRISImageConfigured bool `json:"qris_image_configured"`
		} `json:"account"`
	}
	json.Unmarshal(after.Body.Bytes(), &afterBody)
	if !afterBody.Account.QRISImageConfigured {
		t.Fatal("qris_image_configured mau true setelah upload")
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

```bash
cd backend && TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" go test ./internal/httpapi/... -run QRISImage -v
```
Expected: FAIL — route belum terdaftar (404) / field belum ada.

- [ ] **Step 3: Implementasi handler**

`backend/internal/httpapi/admin_qris_image.go`:

```go
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
```

- [ ] **Step 4: Daftarkan route**

Di `backend/internal/httpapi/api.go`, tambahkan setelah baris route `admin/account/telegram/link`:

```go
mux.Handle("PUT /api/v1/admin/account/qris-image", a.requireAdmin(http.HandlerFunc(a.handleAdminUploadQRISImage)))
mux.Handle("GET /api/v1/admin/account/qris-image", a.requireAdmin(http.HandlerFunc(a.handleAdminGetQRISImage)))
mux.Handle("DELETE /api/v1/admin/account/qris-image", a.requireAdmin(http.HandlerFunc(a.handleAdminDeleteQRISImage)))
```

- [ ] **Step 5: Tambah `qris_image_configured` ke profil**

Di `backend/internal/httpapi/admin_account_settings.go`, ubah `accountProfileJSON`:

```go
type accountProfileJSON struct {
	Username       string  `json:"username"`
	BusinessName   string  `json:"business_name"`
	Email          string  `json:"email"`
	TelegramChatID *string `json:"telegram_chat_id"`
	TelegramAvailable bool `json:"telegram_available"`
	EmailVerified bool `json:"email_verified"`
	// QRISImageConfigured: false berarti POST /invoices akan ditolak
	// 409 qris_not_configured sampai account ini upload QRIS di Settings.
	QRISImageConfigured bool `json:"qris_image_configured"`
}
```

Lalu di `handleAdminGetAccount`, tambahkan pemanggilan `HasQRISImage` sebelum `writeJSON`:

```go
	hasQRIS, err := a.store.HasQRISImage(r.Context(), accountID)
	if err != nil {
		slog.Error("cek qris image gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "account": accountProfileJSON{
		Username: acc.Username, BusinessName: acc.BusinessName, Email: acc.Email, TelegramChatID: acc.TelegramChatID,
		TelegramAvailable: settings.TelegramBotToken != "", EmailVerified: acc.EmailVerifiedAt != nil,
		QRISImageConfigured: hasQRIS,
	}})
```

(Field ini SENGAJA tidak ditambahkan ke `handleAdminUpdateAccount` — endpoint itu tidak menyentuh QRIS, dan halaman Settings sudah reload profil penuh lewat `getAccountProfile` setelah upload/hapus, bukan dari response PATCH profil.)

- [ ] **Step 6: Tambah `account_qris_images` ke TRUNCATE `auth_middleware_test.go`**

```go
"TRUNCATE notification_events, event_reviews, invoices, api_keys, webhook_deliveries, webhook_endpoints, devices, accounts, vendor_admins, audit_log, notification_settings, notification_log, password_reset_tokens, telegram_link_codes, email_verification_tokens, account_activity_log, plans, account_qris_images RESTART IDENTITY CASCADE"
```

- [ ] **Step 7: Jalankan test lagi, pastikan lulus**

```bash
cd backend && TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" go test ./internal/httpapi/... -run "QRISImage|GetAccountProfile" -v
```
Expected: semua `PASS`.

- [ ] **Step 8: `go build`, `go vet`, `gofmt -l .`, commit**

```bash
cd backend
go build ./... && go vet ./... && gofmt -l .
git add internal/httpapi/admin_qris_image.go internal/httpapi/admin_qris_image_test.go internal/httpapi/admin_account_settings.go internal/httpapi/api.go internal/httpapi/auth_middleware_test.go
git commit -m "feat(httpapi): endpoint self-service upload/lihat/hapus QRIS statis"
```

---

### Task 3: `POST /invoices` menyertakan QRIS + gagal cepat kalau belum diatur

**Files:**
- Modify: `backend/internal/httpapi/invoices.go`
- Test: `backend/internal/httpapi/invoices_test.go`

**Interfaces:**
- Consumes: `store.HasQRISImage`, `store.GetQRISImage`, `store.ErrQRISImageNotFound` (Task 1).
- Produces: field baru `qris_image *string` di `invoiceJSON`, dipakai integrator manapun (termasuk whuzpay-pg di sub-project 2 nanti).

- [ ] **Step 1: Tulis test yang gagal dulu**

Tambahkan ke `backend/internal/httpapi/invoices_test.go`:

```go
func TestCreateInvoiceGagalTanpaQRISImage(t *testing.T) {
	h, apiKey := newAPIWithAPIKey(t)

	rec := createInvoiceReq(t, h, apiKey, "ORDER-NOQRIS", 50000)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct{ Error string `json:"error"` }
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "qris_not_configured" {
		t.Fatalf("error = %q, mau qris_not_configured", errBody.Error)
	}
}

func TestCreateInvoiceGagalTanpaQRISImageTidakMenyimpanApaPun(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedActiveAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	id, _ := store.NewAPIKeyID()
	rawKey, hash, _ := store.GenerateAPIKeySecret()
	if err := s.CreateAPIKey(ctx, "acc_1", id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(), func() time.Time { return fixedNow }).Handler()

	createInvoiceReq(t, h, rawKey, "ORDER-NOQRIS-2", 50000)

	invoices, err := s.ListInvoices(ctx, "acc_1", 10, 0, store.InvoiceFilter{})
	if err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}
	if len(invoices) != 0 {
		t.Fatalf("invoices = %+v, mau kosong -- gagal 409 tidak boleh menyimpan invoice", invoices)
	}
}

func TestCreateInvoiceMenyertakanQRISImage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedActiveAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	id, _ := store.NewAPIKeyID()
	rawKey, hash, _ := store.GenerateAPIKeySecret()
	if err := s.CreateAPIKey(ctx, "acc_1", id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := s.UpsertQRISImage(ctx, "acc_1", png, "image/png"); err != nil {
		t.Fatalf("UpsertQRISImage: %v", err)
	}
	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(), func() time.Time { return fixedNow }).Handler()

	rec := createInvoiceReq(t, h, rawKey, "ORDER-QRIS", 50000)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201 (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		QRISImage *string `json:"qris_image"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.QRISImage == nil || !strings.HasPrefix(*body.QRISImage, "data:image/png;base64,") {
		t.Fatalf("qris_image = %v, mau data URI image/png", body.QRISImage)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(*body.QRISImage, "data:image/png;base64,"))
	if err != nil || string(decoded) != string(png) {
		t.Fatalf("qris_image tidak decode balik ke byte yang sama dengan yang di-upload")
	}
}

func TestGetInvoiceTidakMenyertakanQRISImage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedActiveAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	id, _ := store.NewAPIKeyID()
	rawKey, hash, _ := store.GenerateAPIKeySecret()
	if err := s.CreateAPIKey(ctx, "acc_1", id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("x"), "image/png"); err != nil {
		t.Fatalf("UpsertQRISImage: %v", err)
	}
	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(), func() time.Time { return fixedNow }).Handler()

	created := createInvoiceReq(t, h, rawKey, "ORDER-POLL", 50000)
	inv := decodeInvoice(t, created)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+inv.ID, nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var body struct {
		QRISImage *string `json:"qris_image"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.QRISImage != nil {
		t.Fatalf("qris_image = %v, mau tidak ada di response GET (polling)", *body.QRISImage)
	}
}
```

Tambahkan import `encoding/base64` di header file test ini kalau belum ada.

- [ ] **Step 2: Jalankan test, pastikan gagal**

```bash
cd backend && TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" go test ./internal/httpapi/... -run "CreateInvoice|GetInvoiceTidakMenyertakan" -v
```
Expected: `TestCreateInvoiceBerhasil` (test lama) mulai FAIL karena sekarang butuh QRIS -- ini pertanda perlu diperbaiki di Step 3 (lihat catatan test lama).

**Catatan penting:** `TestCreateInvoiceBerhasil` dan test lain yang sudah ada di `invoices_test.go` yang memakai `newAPIWithAPIKey` akan mulai gagal karena `newAPIWithAPIKey` belum meng-upload QRIS. Perbaiki helper itu, BUKAN menulis ulang tiap test:

- [ ] **Step 2b: Tambahkan upload QRIS ke `newAPIWithAPIKey`**

Di `backend/internal/httpapi/invoices_test.go`, ubah `newAPIWithAPIKey`:

```go
func newAPIWithAPIKey(t *testing.T) (http.Handler, string) {
	t.Helper()

	s := newTestStore(t)
	ctx := context.Background()
	seedActiveAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	// QRIS wajib ada sebelum POST /invoices bisa berhasil -- lihat
	// TestCreateInvoiceGagalTanpaQRISImage untuk test kasus sebaliknya.
	if err := s.UpsertQRISImage(ctx, "acc_1", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png"); err != nil {
		t.Fatalf("UpsertQRISImage: %v", err)
	}

	id, err := store.NewAPIKeyID()
	if err != nil {
		t.Fatalf("NewAPIKeyID: %v", err)
	}
	rawKey, hash, err := store.GenerateAPIKeySecret()
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret: %v", err)
	}
	if err := s.CreateAPIKey(ctx, "acc_1", id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(), func() time.Time { return fixedNow }).Handler()
	return h, rawKey
}
```

`TestCreateInvoiceGagalTanpaQRISImage` di Step 1 sengaja TIDAK memakai `newAPIWithAPIKey` (yang sekarang selalu ada QRIS) -- itu build API key manual tanpa upload QRIS, persis pola `TestCreateInvoiceGagalTanpaQRISImageTidakMenyimpanApaPun`. Perbaiki jadi:

```go
func TestCreateInvoiceGagalTanpaQRISImage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedActiveAccount(t, s, "acc_1")
	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	id, _ := store.NewAPIKeyID()
	rawKey, hash, _ := store.GenerateAPIKeySecret()
	if err := s.CreateAPIKey(ctx, "acc_1", id, "Website utama", hash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	h := httpapi.New(s, encKey(), adminSessionKey(), webhookSecretKey(), vendorSessionKey(), settingsSecretKey(), func() time.Time { return fixedNow }).Handler()

	rec := createInvoiceReq(t, h, rawKey, "ORDER-NOQRIS", 50000)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var errBody struct{ Error string `json:"error"` }
	json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Error != "qris_not_configured" {
		t.Fatalf("error = %q, mau qris_not_configured", errBody.Error)
	}
}
```

- [ ] **Step 3: Implementasi**

Di `backend/internal/httpapi/invoices.go`, ubah `invoiceJSON` dan `toInvoiceJSON`:

```go
type invoiceJSON struct {
	ID              string  `json:"id"`
	ExternalRef     string  `json:"external_ref"`
	RequestedAmount int64   `json:"requested_amount"`
	UniqueAmount    int64   `json:"unique_amount"`
	Status          string  `json:"status"`
	MatchedEventID  *string `json:"matched_event_id"`
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       string  `json:"expires_at"`
	PaidAt          *string `json:"paid_at"`
	// QrisImage HANYA terisi di response POST /invoices (create) -- lihat
	// handleCreateInvoice. GET /invoices/{id} (polling) sengaja tidak
	// mengisi ini, sudah didapat sekali dari response create.
	QrisImage *string `json:"qris_image,omitempty"`
}
```

(`ExternalRef` dst tidak berubah -- cuma menambah field baru di akhir struct.)

Ubah `handleCreateInvoice`:

```go
func (a *API) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	var req createInvoiceRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.ExternalRef == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "external_ref wajib diisi")
		return
	}
	if req.Amount <= 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "amount harus lebih besar dari 0")
		return
	}

	// Gerbang sebelum invoice dibuat sama sekali -- invoice tanpa cara bayar
	// tidak berguna buat integrator (lihat spec
	// docs/superpowers/specs/2026-09-15-account-qris-image-design.md §3.3).
	img, err := a.store.GetQRISImage(r.Context(), accountID)
	if errors.Is(err, store.ErrQRISImageNotFound) {
		a.writeError(w, http.StatusConflict, "qris_not_configured",
			"QRIS belum diatur -- upload di halaman Settings dulu")
		return
	}
	if err != nil {
		slog.Error("ambil qris image gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	inv, created, err := a.store.CreateInvoice(r.Context(), a.now(), accountID, req.ExternalRef, req.Amount)
	switch {
	case errors.Is(err, store.ErrInvoiceRefConflict):
		a.writeError(w, http.StatusConflict, "external_ref_conflict",
			"external_ref sudah dipakai invoice lain dengan amount berbeda")
		return
	case errors.Is(err, store.ErrInvoiceAllocationFull):
		a.writeError(w, http.StatusServiceUnavailable, "allocation_full",
			"tidak ada nominal unik yang tersedia untuk amount ini, coba lagi sebentar lagi")
		return
	case err != nil:
		slog.Error("create invoice gagal", "external_ref", req.ExternalRef, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	out := toInvoiceJSON(inv)
	dataURI := "data:" + img.ContentType + ";base64," + base64.StdEncoding.EncodeToString(img.ImageData)
	out.QrisImage = &dataURI
	writeJSON(w, status, out)
}
```

Tambahkan import `"encoding/base64"` di header `invoices.go`. `handleGetInvoice` TIDAK diubah sama sekali (tetap memanggil `toInvoiceJSON(inv)` tanpa mengisi `QrisImage`, jadi field itu `omitempty` dan hilang dari JSON polling).

- [ ] **Step 4: Jalankan seluruh test invoices, pastikan lulus**

```bash
cd backend && TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" go test ./internal/httpapi/... -run Invoice -v
```
Expected: semua `PASS`, termasuk test lama yang sekarang lulus lagi berkat Step 2b.

- [ ] **Step 5: `make test` penuh (backend), `go vet`, `gofmt -l .`, commit**

```bash
cd backend
make test
go vet ./... && gofmt -l .
git add internal/httpapi/invoices.go internal/httpapi/invoices_test.go
git commit -m "feat(httpapi): POST /invoices menyertakan qris_image, tolak 409 kalau belum diatur"
```

---

### Task 4: Frontend — API client, halaman Settings, halaman Logs

**Files:**
- Modify: `dashboard/src/lib/api.ts`
- Modify: `dashboard/src/app/(dashboard)/settings/page.tsx`
- Modify: `dashboard/src/app/(dashboard)/logs/page.tsx`

**Interfaces:**
- Consumes: `PUT`/`GET`/`DELETE /api/v1/admin/account/qris-image`, `qris_image_configured` di `AccountProfile` (Task 2).
- Produces: `uploadQRISImage(base64, contentType)`, `deleteQRISImage()`, konstanta URL `qrisImageURL = "/api/v1/admin/account/qris-image"` dipakai langsung sebagai `<img src>`.

- [ ] **Step 1: Tambah tipe dan fungsi di `lib/api.ts`**

Ubah interface `AccountProfile` (tambah field di akhir):

```typescript
export interface AccountProfile {
  username: string;
  business_name: string;
  email: string;
  telegram_chat_id: string | null;
  telegram_available: boolean;
  email_verified: boolean;
  /** false berarti POST /invoices ditolak 409 qris_not_configured. */
  qris_image_configured: boolean;
}
```

Tambahkan setelah `resendVerificationEmail`:

```typescript
// --- QRIS statis (dipakai integrator lewat POST /invoices) -------------

/** URL gambar langsung -- dipakai sebagai <img src>, browser kirim cookie sesi otomatis. */
export const qrisImageURL = "/api/v1/admin/account/qris-image";

export function uploadQRISImage(
  imageBase64: string,
  contentType: string,
): Promise<{ success: true }> {
  return apiFetch("/api/v1/admin/account/qris-image", {
    method: "PUT",
    body: JSON.stringify({ image_base64: imageBase64, content_type: contentType }),
  });
}

export function deleteQRISImage(): Promise<{ success: true }> {
  return apiFetch("/api/v1/admin/account/qris-image", { method: "DELETE" });
}
```

Ubah union type `ActivityAction` (tambah dua nilai baru):

```typescript
export type ActivityAction =
  | "login_success"
  | "login_failed"
  | "password_changed"
  | "password_reset"
  | "api_key_created"
  | "api_key_revoked"
  | "device_added"
  | "device_deleted"
  | "qris_image_updated"
  | "qris_image_removed";
```

- [ ] **Step 2: `npx tsc --noEmit`, pastikan gagal**

```bash
cd dashboard && npx tsc --noEmit
```
Expected: FAIL di `logs/page.tsx` -- `Record<ActivityAction, string>` (`ACTION_LABEL`, `ACTION_BADGE`) sekarang kurang dua key wajib.

- [ ] **Step 3: Lengkapi `logs/page.tsx`**

Tambah import `QrCode` dari `lucide-react` (baris import lucide yang sudah ada):

```typescript
import { Globe, KeyRound, LogIn, QrCode, RotateCw, Search, Smartphone } from "lucide-react";
```

Ubah `ACTION_LABEL`:

```typescript
const ACTION_LABEL: Record<ActivityAction, string> = {
  login_success: "Login berhasil",
  login_failed: "Login gagal",
  password_changed: "Password diganti",
  password_reset: "Password direset lewat email",
  api_key_created: "API key dibuat",
  api_key_revoked: "API key dicabut",
  device_added: "Device ditambahkan",
  device_deleted: "Device dihapus",
  qris_image_updated: "QRIS diperbarui",
  qris_image_removed: "QRIS dihapus",
};
```

Ubah `ACTION_BADGE`:

```typescript
const ACTION_BADGE: Record<ActivityAction, string> = {
  login_success: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  login_failed: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  password_changed: "border-transparent bg-sky-500/15 text-sky-700 dark:text-sky-400",
  password_reset: "border-transparent bg-sky-500/15 text-sky-700 dark:text-sky-400",
  api_key_created: "border-transparent bg-slate-500/15 text-slate-700 dark:text-slate-300",
  api_key_revoked: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  device_added: "border-transparent bg-slate-500/15 text-slate-700 dark:text-slate-300",
  device_deleted: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  qris_image_updated: "border-transparent bg-slate-500/15 text-slate-700 dark:text-slate-300",
  qris_image_removed: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
};
```

Ubah `ActionIcon`:

```typescript
function ActionIcon({ action }: { action: ActivityAction }) {
  switch (action) {
    case "login_success":
    case "login_failed":
      return <LogIn className="size-3.5" />;
    case "api_key_created":
    case "api_key_revoked":
      return <KeyRound className="size-3.5" />;
    case "device_added":
    case "device_deleted":
      return <Smartphone className="size-3.5" />;
    case "qris_image_updated":
    case "qris_image_removed":
      return <QrCode className="size-3.5" />;
    default:
      return null;
  }
}
```

- [ ] **Step 4: `npx tsc --noEmit` lagi, pastikan lulus**

```bash
cd dashboard && npx tsc --noEmit
```
Expected: bersih, 0 error.

- [ ] **Step 5: Tambah card "QRIS Pembayaran" di Settings**

Di `dashboard/src/app/(dashboard)/settings/page.tsx`, tambah import (gabung ke import lucide-react dan ke import dari `@/lib/api` yang sudah ada):

```typescript
import { Bell, Eye, EyeOff, KeyRound, Loader2, MailWarning, QrCode, RotateCw, Send, Trash2, Unlink, UserRound } from "lucide-react";
```

```typescript
import {
  ApiError,
  changePassword,
  createTelegramLink,
  deleteQRISImage,
  getAccountProfile,
  qrisImageURL,
  resendVerificationEmail,
  setTelegramChatID,
  updateAccountProfile,
  uploadQRISImage,
  type AccountProfile,
  type TelegramLink,
} from "@/lib/api";
```

Ubah grid di `SettingsPage` supaya card baru muncul (`lg:col-span-2`, sejajar dengan `NotificationCard`):

```tsx
          <div className="grid items-start gap-6 lg:grid-cols-2">
            <ProfileCard
              key={`${data.business_name}|${data.email}`}
              profile={data}
              onSaved={reload}
            />
            <PasswordCard />
            <div className="lg:col-span-2">
              <QRISImageCard key={String(data.qris_image_configured)} profile={data} onSaved={reload} />
            </div>
            <div className="lg:col-span-2">
              <NotificationCard key={data.telegram_chat_id ?? ""} profile={data} onSaved={reload} />
            </div>
          </div>
```

Tambahkan komponen baru (taruh dekat `PasswordCard`, memakai wrapper `SettingsCard` yang sudah ada di file ini):

```tsx
const MAX_QRIS_IMAGE_BYTES = 300 * 1024;

function QRISImageCard({
  profile,
  onSaved,
}: {
  profile: AccountProfile;
  onSaved: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  // Dipakai memaksa <img> reload setelah upload -- src yang sama persis
  // tidak akan di-refetch browser tanpa ini.
  const [cacheBust, setCacheBust] = useState(0);

  async function onFileSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = ""; // supaya memilih file yang sama lagi tetap trigger onChange
    if (!file) return;

    if (!["image/png", "image/jpeg"].includes(file.type)) {
      toast.error("Hanya file PNG atau JPEG yang didukung.");
      return;
    }
    if (file.size > MAX_QRIS_IMAGE_BYTES) {
      toast.error("Ukuran gambar maksimal 300KB.");
      return;
    }

    setBusy(true);
    try {
      const dataUrl = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result as string);
        reader.onerror = () => reject(reader.error);
        reader.readAsDataURL(file);
      });
      // readAsDataURL menghasilkan "data:image/png;base64,XXXX" -- backend
      // cuma butuh bagian base64-nya, content_type dikirim terpisah.
      const base64 = dataUrl.split(",")[1] ?? "";
      await uploadQRISImage(base64, file.type);
      toast.success("QRIS berhasil diperbarui.");
      setCacheBust((n) => n + 1);
      onSaved();
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(err.message);
      } else {
        toast.error("Gagal upload QRIS.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function onDelete() {
    setBusy(true);
    try {
      await deleteQRISImage();
      toast.success("QRIS dihapus.");
      onSaved();
    } catch {
      toast.error("Gagal menghapus QRIS.");
    } finally {
      setBusy(false);
      setConfirmingDelete(false);
    }
  }

  return (
    <SettingsCard
      icon={<QrCode className="size-4 text-muted-foreground" />}
      title="QRIS Pembayaran"
      description="Gambar QRIS statis milikmu sendiri -- disertakan otomatis tiap kali sistem integrasi kamu membuat invoice lewat API. Tanpa ini, pembuatan invoice akan ditolak."
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
        {profile.qris_image_configured ? (
          // eslint-disable-next-line @next/next/no-img-element -- gambar dari
          // backend sendiri (bukan aset build), bukan kandidat next/image.
          <img
            src={`${qrisImageURL}?v=${cacheBust}`}
            alt="QRIS"
            className="size-32 rounded-xl border border-border/60 object-contain p-2"
          />
        ) : (
          <div className="flex size-32 items-center justify-center rounded-xl border border-dashed text-xs text-muted-foreground">
            Belum diatur
          </div>
        )}
        <div className="flex flex-1 flex-col gap-2">
          <Label htmlFor="qris-image-input">{profile.qris_image_configured ? "Ganti gambar" : "Upload gambar"}</Label>
          <Input
            id="qris-image-input"
            type="file"
            accept="image/png,image/jpeg"
            disabled={busy}
            onChange={onFileSelected}
          />
          <p className="text-xs text-muted-foreground">PNG atau JPEG, maksimal 300KB.</p>
          {profile.qris_image_configured && (
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="w-fit text-destructive hover:text-destructive"
              disabled={busy}
              onClick={() => setConfirmingDelete(true)}
            >
              <Trash2 className="mr-1.5 size-3.5" />
              Hapus
            </Button>
          )}
        </div>
      </div>

      <AlertDialog open={confirmingDelete} onOpenChange={(open) => !open && setConfirmingDelete(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus gambar QRIS?</AlertDialogTitle>
            <AlertDialogDescription>
              Pembuatan invoice lewat API akan langsung ditolak sampai kamu upload ulang.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onDelete} disabled={busy}>
              Hapus
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsCard>
  );
}
```

Tambahkan `import type { ChangeEvent } from "react";` bila `React.ChangeEvent` tidak resolve tanpa itu (cek gaya import React yang sudah dipakai file ini -- kalau file ini tidak `import React from "react"` sama sekali, ganti tanda tangan jadi `onFileSelected(e: ChangeEvent<HTMLInputElement>)` dan tambahkan `ChangeEvent` ke named import `"use client"` yang sudah ada di baris pertama file, mis. `import { useEffect, useState, type ChangeEvent, type FormEvent, type ReactNode } from "react";`).

- [ ] **Step 6: `npx tsc --noEmit`, `npx eslint .`**

```bash
cd dashboard && npx tsc --noEmit && npx eslint .
```
Expected: bersih, 0 error/warning.

- [ ] **Step 7: `npx next build` (clean)**

```bash
cd dashboard && rm -rf .next && npx next build
```
Expected: build sukses, route `/settings` dan `/logs` ter-generate tanpa error.

- [ ] **Step 8: Commit**

```bash
cd dashboard
git add src/lib/api.ts "src/app/(dashboard)/settings/page.tsx" "src/app/(dashboard)/logs/page.tsx"
git commit -m "feat(dashboard): halaman Settings kelola QRIS statis, Logs kenali aktivitasnya"
```

---

### Task 5: Dokumentasi (API Docs, CLAUDE.md, QA report)

**Files:**
- Modify: `dashboard/src/app/(dashboard)/api-docs/page.tsx`
- Modify: `CLAUDE.md`
- Modify: `docs/qa/qa-report.md`

**Interfaces:**
- Consumes: perilaku final Task 1–4 (field `qris_image`, error `qris_not_configured`, endpoint self-service).

- [ ] **Step 1: Perbarui komentar sumber di atas `api-docs/page.tsx`**

```typescript
/**
 * Dokumentasi integrasi untuk server website customer. Isinya WAJIB sama
 * dengan perilaku backend -- sumbernya:
 *   backend/internal/httpapi/invoices.go, apikey_auth.go (invoice + auth)
 *   backend/internal/httpapi/webhook_send.go (payload + tanda tangan)
 *   backend/internal/store/invoice.go (nominal unik, 15 menit, idempotensi)
 *   backend/internal/store/webhook.go (5 percobaan, jeda 1/2/4/8 menit)
 *   backend/internal/store/qris_image.go (gambar QRIS di response create)
 * Kalau salah satu berubah, halaman ini ikut diubah.
 */
```

- [ ] **Step 2: Update contoh response JSON `POST /invoices`**

Di fungsi yang merender bagian "Membuat invoice" (JSON contoh setelah `Respons 201 Created`), tambahkan field:

```typescript
            code: `{
  "id": "inv_3f9c2a7d1e8b4c6a9d0e5f7a2b1c3d4e",
  "external_ref": "ORDER-1001",
  "requested_amount": 150000,
  "unique_amount": 150347,
  "status": "PENDING",
  "matched_event_id": null,
  "created_at": "2026-09-14T09:30:00Z",
  "expires_at": "2026-09-14T09:45:00Z",
  "paid_at": null,
  "qris_image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
}`,
```

- [ ] **Step 3: Tambah baris ke `InvoiceFields()`**

```typescript
        ["paid_at", "string | null", "Waktu invoice ditandai lunas."],
        [
          "qris_image",
          "string",
          <>
            QRIS statis milikmu, data URI siap pakai (<C>&lt;img src=&#123;qris_image&#125;&gt;</C>).
            Selalu terisi -- kalau belum diatur, permintaan ini sudah ditolak <C>409
            qris_not_configured</C> sebelum sampai sini. Cuma ada di respons <C>POST</C>, tidak
            diulang di <C>GET /invoices/&#123;id&#125;</C>.
          </>,
        ],
```

- [ ] **Step 4: Tambah baris ke tabel error `Errors()`**

```typescript
  const rows: [string, string, string][] = [
    ["400", "invalid_payload", "Body bukan JSON, external_ref kosong, atau amount ≤ 0."],
    ["401", "unauthenticated", "API key tidak ada, salah, atau sudah dicabut."],
    [
      "402",
      "account_expired / account_suspended / account_revoked",
      "Akun tidak aktif. Cek halaman License.",
    ],
    ["404", "not_found", "Invoice tidak ditemukan di akun ini."],
    ["409", "external_ref_conflict", "external_ref sudah dipakai dengan amount berbeda."],
    [
      "409",
      "qris_not_configured",
      "Belum upload gambar QRIS di Settings. Invoice tidak dibuat -- upload dulu, lalu coba lagi.",
    ],
    [
      "503",
      "allocation_full",
      "Semua nominal unik untuk amount ini sedang dipakai invoice lain yang masih PENDING. Coba lagi sebentar lagi.",
    ],
    ["500", "internal", "Kesalahan di sisi kami. Aman dicoba ulang dengan external_ref yang sama."],
  ];
```

- [ ] **Step 5: `npx tsc --noEmit`, `npx eslint .`, `npx next build`**

```bash
cd dashboard && npx tsc --noEmit && npx eslint . && rm -rf .next && npx next build
```
Expected: bersih.

- [ ] **Step 6: Perbarui `CLAUDE.md`**

Tambahkan paragraf baru di bawah bagian **API Docs (`/api-docs`)** yang sudah ada (cari teks `daftar file sumbernya ada di komentar atas`):

```markdown
**QRIS statis per account** (migrasi 00018, `internal/store/qris_image.go`):
tiap account upload gambar QRIS-nya sendiri lewat Settings
(`PUT`/`GET`/`DELETE /api/v1/admin/account/qris-image`, disimpan `BYTEA` di
Postgres -- bukan filesystem/S3, konsisten dengan backup `pg_dump`-only).
`POST /api/v1/invoices` menyertakan gambar itu sebagai data URI
(`qris_image`) di response, dan MENOLAK `409 qris_not_configured` kalau
account belum upload -- invoice tanpa cara bayar tidak dibuat sama sekali.
`GET /invoices/{id}` (polling) sengaja tidak mengulang field ini. Ini
sub-project 1 dari rencana migrasi `whuzpay-pg/` (payment gateway
aggregator, folder terpisah di repo ini) dari provider Cashi ke
gopay-notifications sendiri -- lihat
`docs/superpowers/specs/2026-09-15-account-qris-image-design.md`.
Sub-project 2 (provider adapter di `whuzpay-pg/`) belum dikerjakan.
```

- [ ] **Step 7: Tulis laporan QA**

Tambahkan section baru di akhir `docs/qa/qa-report.md` (nomor lanjut dari section terakhir yang ada), memuat tabel PASS/FAIL/NEEDS-DEVICE untuk:
- Semua test store (`TestUpsertDanGetQRISImage`, `TestUpsertQRISImageMenimpaBukanMenambah`, dst dari Task 1) — PASS dengan output `go test` yang ditempel.
- Semua test HTTP self-service (Task 2) — PASS dengan output ditempel.
- Semua test invoice (Task 3, termasuk test lama yang diperbaiki) — PASS dengan output ditempel.
- `npx tsc --noEmit` / `npx eslint .` / `npx next build` dashboard (Task 4-5) — PASS dengan output ditempel.
- Uji sungguhan upload gambar asli lewat browser dan lihat hasilnya tampil benar di Settings — `NEEDS-DEVICE` (butuh Akbar `npm run dev`).
- `make test` full suite backend — PASS dengan output ditempel.

- [ ] **Step 8: Commit**

```bash
git add "dashboard/src/app/(dashboard)/api-docs/page.tsx" CLAUDE.md docs/qa/qa-report.md
git commit -m "docs: perbarui API Docs, CLAUDE.md, dan QA report untuk QRIS statis per account"
```

---

## Self-Review (dilakukan penulis plan, bukan langkah eksekusi)

- **Cakupan spec:** §2 (skema) → Task 1. §3.1 (endpoint self-service) → Task 2. §3.2 (profil) → Task 2 Step 5. §3.3 (invoice) → Task 3. §3.4 (store) → Task 1. §4 (Settings UI) → Task 4. §5 (dokumentasi) → Task 5. §6 (testing) → tercakup di tiap task. §7 (di luar cakupan) → sengaja tidak ada task.
- **Placeholder scan:** tidak ada TBD/TODO; setiap step kode punya isi lengkap.
- **Konsistensi tipe:** `QRISImage{AccountID, ImageData, ContentType, UpdatedAt}` dipakai identik di Task 1 (definisi) dan Task 3 (`img.ContentType`, `img.ImageData`); `accountProfileJSON.QRISImageConfigured` (Task 2) ↔ `AccountProfile.qris_image_configured` (Task 4) cocok lewat JSON tag `qris_image_configured`; `ActivityAction` union (Task 4 lib/api.ts) mencakup persis 2 nilai baru dari `store.ActivityQRISImageUpdated`/`ActivityQRISImageRemoved` (Task 1).
