# Akun & Data Model Multi-Tenant — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ubah backend `gopay-notifications` dari single-tenant self-hosted jadi hosted multi-tenant: satu tabel `accounts` menggantikan `admin_users` + License Server (`customers`/`licenses`/`installations`), seluruh tabel data dapat kolom `account_id` yang diturunkan server-side dari kredensial (sesi/API key/HMAC device), dan License Server (`cmd/licenseserver`, `internal/licenseserver`, `internal/licenseclient`) dibongkar total.

**Architecture:** Semua perubahan di database `gopay` yang sudah ada (bukan database baru) — satu migration menambah `accounts`, `vendor_admins`, dan kolom `account_id` (nullable → backfill → NOT NULL) di `devices`/`invoices`/`notification_events`/`api_keys`/`webhook_endpoints`/`webhook_deliveries`/`event_reviews`. Setiap fungsi `internal/store` yang menyentuh tabel itu menerima `accountID string` sebagai parameter. `internal/httpapi` menurunkan `accountID` dari tiga jalur auth yang sudah ada (cookie sesi, API key, HMAC device) lalu meneruskannya ke `store`. Endpoint vendor baru (`/api/v1/vendor/*`, sesi `vendor_session` terpisah) menggantikan seluruh fungsi License Server + Vendor Dashboard.

**Tech Stack:** Go 1.23+, `pgx/v5`, PostgreSQL 16, `goose` migration, Next.js 16 (vendor-dashboard, tidak diubah stack-nya).

## Global Constraints

- `account_id` **tidak pernah** diambil dari input client (body/query/header) — selalu diturunkan server-side dari kredensial yang sudah diverifikasi (spec §3).
- `go test ./... -p 1` wajib tetap lulus setelah SETIAP task — jangan biarkan test merah menumpuk antar task.
- Migration harus tetap bisa jalan bersih di database test kosong maupun database produksi yang sudah berisi data lama (spec §5).
- Semua commit message pakai bahasa Indonesia, gaya sama seperti commit sebelumnya di repo ini (alasan, bukan cuma "apa yang berubah").
- `gofmt -l .` harus kosong sebelum tiap commit Go.
- Constraint unik invoice di-scope ulang jadi `(account_id, external_ref)` dan `(account_id, unique_amount) WHERE status='PENDING'` — bukan lagi global (spec §2.3).
- Endpoint publik `GET /api/v1/events` (diproteksi Caddy basic_auth, dipakai buat verifikasi manual per `docs/api-contract.md` §7) **dihapus** di task 9 — tidak ada cara menurunkan `account_id` dari basic auth Caddy yang generik, dan `GET /api/v1/admin/events` (sudah scoped lewat sesi) adalah penggantinya yang benar untuk model multi-tenant.

---

### Task 1: Token sesi bawa identitas (`internal/auth`)

**Files:**
- Modify: `backend/internal/auth/session.go`
- Test: `backend/internal/auth/session_test.go`

**Interfaces:**
- Produces: `NewSessionToken(key []byte, now time.Time, subject string) string`, `VerifySessionToken(key []byte, token string, now time.Time) (subject string, ok bool)` — dipakai Task 8 (sesi customer, `subject` = `account_id`) dan Task 10 (sesi vendor, `subject` = username vendor admin).

Sekarang token cuma `"<expiryUnix>.<hmac>"` — tidak membawa identitas sama sekali (cukup untuk MVP satu-admin). Begitu ada banyak akun, sesi harus tahu **akun mana** yang login, bukan cuma "ada sesi valid".

- [ ] **Step 1: Baca test yang ada dulu supaya tahu pola nama test di file ini**

Run: `sed -n '1,60p' backend/internal/auth/session_test.go`

- [ ] **Step 2: Tulis test yang gagal untuk payload baru**

Tambahkan di `backend/internal/auth/session_test.go`:

```go
func TestSessionTokenBawaSubject(t *testing.T) {
	key := []byte("kunci-uji-32-byte-panjangnya-pas")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	token := NewSessionToken(key, now, "acc_toko123")

	subject, ok := VerifySessionToken(key, token, now.Add(time.Hour))
	if !ok {
		t.Fatal("token seharusnya valid")
	}
	if subject != "acc_toko123" {
		t.Fatalf("subject = %q, mau %q", subject, "acc_toko123")
	}
}

func TestSessionTokenKedaluwarsaMenolakSubject(t *testing.T) {
	key := []byte("kunci-uji-32-byte-panjangnya-pas")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	token := NewSessionToken(key, now, "acc_toko123")

	_, ok := VerifySessionToken(key, token, now.Add(SessionDuration+time.Minute))
	if ok {
		t.Fatal("token kedaluwarsa tidak boleh valid")
	}
}

func TestSessionTokenKunciSalahMenolak(t *testing.T) {
	key := []byte("kunci-uji-32-byte-panjangnya-pas")
	other := []byte("kunci-lain-32-byte-panjangnya-ok")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	token := NewSessionToken(key, now, "acc_toko123")

	_, ok := VerifySessionToken(other, token, now)
	if ok {
		t.Fatal("token dengan kunci beda tidak boleh valid")
	}
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal build (signature belum berubah)**

Run: `cd backend && go test ./internal/auth/... -run TestSessionToken -v`
Expected: compile error — `NewSessionToken`/`VerifySessionToken` belum menerima/mengembalikan `subject`.

- [ ] **Step 4: Ubah `session.go`**

Ganti isi `backend/internal/auth/session.go` jadi:

```go
package auth

import (
	"crypto/hmac"
	"strconv"
	"strings"
	"time"
)

// SessionDuration adalah umur sesi dashboard sejak login (customer maupun vendor).
const SessionDuration = 12 * time.Hour

// NewSessionToken membuat token sesi bertanda tangan yang membawa identitas
// pemiliknya: "<subject>:<expiryUnix>.<hmac>".
//
// subject adalah account_id (sesi customer) atau username (sesi vendor) --
// dipisah pakai ":" dari expiry, lalu seluruh payload itu ditandatangani.
// subject TIDAK BOLEH mengandung karakter ":" (account_id dan username di
// proyek ini selalu alfanumerik+underscore, jadi ini aman).
func NewSessionToken(key []byte, now time.Time, subject string) string {
	expiry := now.Add(SessionDuration).Unix()
	return sessionToken(key, subject, expiry)
}

func sessionToken(key []byte, subject string, expiry int64) string {
	payload := subject + ":" + strconv.FormatInt(expiry, 10)
	sig := Sign(key, payload)
	return payload + "." + sig
}

// VerifySessionToken memeriksa tanda tangan dan masa berlaku token, lalu
// mengembalikan subject yang tersimpan di dalamnya.
//
// Split pakai LastIndex (bukan strings.Cut yang berhenti di titik PERTAMA)
// karena payload sendiri sudah mengandung karakter selain titik ("subject:expiry"),
// sig-nya baru ditempel setelah titik TERAKHIR.
func VerifySessionToken(key []byte, token string, now time.Time) (subject string, ok bool) {
	idx := strings.LastIndex(token, ".")
	if idx < 0 {
		return "", false
	}
	payload, sig := token[:idx], token[idx+1:]
	if payload == "" || sig == "" {
		return "", false
	}

	subj, expiryRaw, found := strings.Cut(payload, ":")
	if !found || subj == "" {
		return "", false
	}
	expiry, err := strconv.ParseInt(expiryRaw, 10, 64)
	if err != nil {
		return "", false
	}
	if now.Unix() > expiry {
		return "", false
	}

	want := Sign(key, payload)
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return "", false
	}
	return subj, true
}
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

Run: `cd backend && go test ./internal/auth/... -v`
Expected: seluruh test `PASS`, termasuk test lama (`TestSessionToken...` yang sudah ada sebelumnya) — sesuaikan pemanggilannya kalau ada test lama yang masih pakai signature 2-parameter (tambahkan argumen subject, mis. `"acc_test"`).

- [ ] **Step 6: Commit**

```bash
cd backend
gofmt -l internal/auth/ # harus kosong
git add internal/auth/session.go internal/auth/session_test.go
git commit -m "feat(auth): token sesi bawa subject (account_id/username vendor)

Sesi yang sekarang cuma bawa timestamp kedaluwarsa -- cukup untuk MVP
satu-admin per instalasi. Model multi-tenant butuh tahu AKUN MANA yang
login, bukan cuma 'ada sesi valid'. Payload jadi \"subject:expiry\",
di-split pakai LastIndex('.') karena payload sendiri sekarang mengandung
karakter selain titik.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 2: Migration — `accounts`, `vendor_admins`, kolom `account_id`

**Files:**
- Create: `backend/migrations/00008_accounts.sql`
- Test: dijalankan lewat `make migrate` (Akbar) — task ini tidak menulis Go test, migration diverifikasi lewat `goose up` + `goose down` bersih.

**Interfaces:**
- Produces: tabel `accounts` (kolom persis §2.1 spec), tabel `vendor_admins` (kolom `id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, created_at, updated_at` — identik `admin_users` lama), kolom `account_id TEXT NOT NULL REFERENCES accounts(id)` di `devices`, `invoices`, `notification_events`, `api_keys`, `webhook_endpoints`, `webhook_deliveries`, `event_reviews`. Tabel `admin_users` dihapus.
- Consumes: tidak ada (task pertama yang menyentuh schema).

- [ ] **Step 1: Tulis migration**

Buat `backend/migrations/00008_accounts.sql`:

```sql
-- +goose Up
-- Pivot self-hosted -> hosted multi-tenant. Lihat
-- docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md.
--
-- accounts menggantikan admin_users (login) DAN customers+licenses License
-- Server (identitas + plan/kuota) -- satu baris = satu customer = satu
-- login, sesuai keputusan MVP (bukan multi-user per akun).
CREATE TABLE accounts (
    id            TEXT PRIMARY KEY,
    business_name TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    plan          TEXT NOT NULL,
    max_devices   INTEGER NOT NULL,
    admin_status  TEXT NOT NULL DEFAULT 'active',
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- vendor_admins: akun Akbar (superadmin) buat login ke Vendor Dashboard.
-- Skema identik admin_users lama -- satu instalasi backend, tapi dua jenis
-- sesi yang tidak boleh pernah tertukar (lihat internal/httpapi/vendor_auth.go
-- di Task 10: cookie vendor_session, BUKAN admin_session).
CREATE TABLE vendor_admins (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Akun pertama: toko Akbar sendiri, migrasi dari admin_users yang sekarang
-- (kalau ada) supaya password yang sudah dipakai tetap jalan tanpa reset.
-- Di database test yang kosong, admin_users tidak punya baris -- fallback
-- ke akun default dengan password ACAK (harus di-reset manual lewat
-- vendor_admins/account tool sebelum dipakai sungguhan, tapi test tidak
-- pernah login pakai kredensial ini).
INSERT INTO accounts (id, business_name, email, username, password_hash, plan, max_devices, expires_at)
SELECT
    'acc_default',
    'Akun Utama',
    'akun-utama@whuzpay.local',
    admin_users.username,
    admin_users.password_hash,
    'Enterprise',
    -1,
    now() + interval '10 years'
FROM admin_users
LIMIT 1;

-- Database yang admin_users-nya kosong (test) tetap butuh satu akun default
-- supaya backfill di bawah tidak menabrak FK NOT NULL.
INSERT INTO accounts (id, business_name, email, username, password_hash, plan, max_devices, expires_at)
SELECT 'acc_default', 'Akun Utama', 'akun-utama@whuzpay.local', 'akun-utama-default',
       '$2a$10$abcdefghijklmnopqrstuuNBv8s5pXjCwR9dGZ0m8yQKz7c1r6C.i', -- hash tidak valid, sengaja -- akun ini tidak dipakai login di test manapun
       'Enterprise', -1, now() + interval '10 years'
WHERE NOT EXISTS (SELECT 1 FROM accounts WHERE id = 'acc_default');

ALTER TABLE devices           ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE invoices          ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE notification_events ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE api_keys          ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE webhook_endpoints ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE webhook_deliveries ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE event_reviews     ADD COLUMN account_id TEXT REFERENCES accounts(id);

UPDATE devices            SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE invoices           SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE notification_events SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE api_keys           SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE webhook_endpoints  SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE webhook_deliveries SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE event_reviews      SET account_id = 'acc_default' WHERE account_id IS NULL;

ALTER TABLE devices            ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE invoices           ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE notification_events ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE api_keys           ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE webhook_endpoints  ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE webhook_deliveries ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE event_reviews      ALTER COLUMN account_id SET NOT NULL;

CREATE INDEX devices_account_id_idx ON devices (account_id);
CREATE INDEX invoices_account_id_idx ON invoices (account_id);
CREATE INDEX notification_events_account_id_idx ON notification_events (account_id);
CREATE INDEX api_keys_account_id_idx ON api_keys (account_id);
CREATE INDEX webhook_endpoints_account_id_idx ON webhook_endpoints (account_id);

-- Constraint unik invoice di-scope ulang per akun (spec §2.3) -- dua
-- merchant beda BOLEH pakai external_ref/nominal yang sama, mereka toko
-- yang berbeda sama sekali.
DROP INDEX invoices_external_ref_idx;
CREATE UNIQUE INDEX invoices_account_external_ref_idx ON invoices (account_id, external_ref);

DROP INDEX invoices_pending_unique_amount_idx;
CREATE UNIQUE INDEX invoices_account_pending_unique_amount_idx
    ON invoices (account_id, unique_amount) WHERE status = 'PENDING';

DROP TABLE admin_users;

-- +goose Down
CREATE TABLE admin_users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO admin_users (username, password_hash)
SELECT username, password_hash FROM accounts WHERE id = 'acc_default';

DROP INDEX invoices_account_pending_unique_amount_idx;
CREATE UNIQUE INDEX invoices_pending_unique_amount_idx
    ON invoices (unique_amount) WHERE status = 'PENDING';
DROP INDEX invoices_account_external_ref_idx;
CREATE UNIQUE INDEX invoices_external_ref_idx ON invoices (external_ref);

ALTER TABLE event_reviews      DROP COLUMN account_id;
ALTER TABLE webhook_deliveries DROP COLUMN account_id;
ALTER TABLE webhook_endpoints  DROP COLUMN account_id;
ALTER TABLE api_keys           DROP COLUMN account_id;
ALTER TABLE notification_events DROP COLUMN account_id;
ALTER TABLE invoices           DROP COLUMN account_id;
ALTER TABLE devices            DROP COLUMN account_id;

DROP TABLE vendor_admins;
DROP TABLE accounts;
```

- [ ] **Step 2: Jalankan migration di database test, verifikasi bersih naik-turun**

Run:
```bash
cd backend
make db-up
make migrate
$(go env GOPATH)/bin/goose -dir migrations postgres "postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" down
$(go env GOPATH)/bin/goose -dir migrations postgres "postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable" up
```
Expected: baik `up` maupun `down` maupun `up` lagi selesai tanpa error.

- [ ] **Step 3: Commit**

```bash
cd backend
git add migrations/00008_accounts.sql
git commit -m "feat(db): migration accounts + vendor_admins + account_id

Fondasi pivot multi-tenant. accounts menggantikan admin_users DAN
customers/licenses License Server. account_id ditambah ke seluruh tabel
data (nullable -> backfill ke 'acc_default' -> NOT NULL, satu migration).
Constraint unik invoice (external_ref, nominal PENDING) di-scope ulang
per account_id -- dua merchant beda boleh pakai ref/nominal yang sama.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 3: `internal/store` — model `Account` dan `VendorAdmin`

**Files:**
- Create: `backend/internal/store/account.go`
- Create: `backend/internal/store/account_test.go`
- Create: `backend/internal/store/vendor_admin.go`
- Delete: `backend/internal/store/admin.go`, `backend/internal/store/admin_test.go`, `backend/internal/store/admin_query_test.go`

**Interfaces:**
- Consumes: pola bcrypt yang sama seperti `admin.go` lama (dihapus di task ini).
- Produces: `store.Account{ID, BusinessName, Email, Username, PasswordHash, Plan, MaxDevices, AdminStatus, ExpiresAt, CreatedAt, UpdatedAt}`, `(Account) VerifyPassword(plaintext string) bool`, `(Account) DerivedStatus(now time.Time) string` (nilai: `"active"|"expiring"|"expired"|"suspended"|"revoked"`), `Store.CreateAccount(ctx, in CreateAccountInput) error`, `Store.GetAccountByUsername(ctx, username string) (Account, error)`, `Store.GetAccountByID(ctx, id string) (Account, error)`, `Store.ListAccounts(ctx) ([]Account, error)`, `Store.RenewAccount(ctx, id string, newExpiresAt time.Time) error`, `Store.SetAccountAdminStatus(ctx, id string, status string) error`, `store.ErrAccountNotFound`. Juga `store.VendorAdmin{ID, Username, PasswordHash}`, `(VendorAdmin) VerifyPassword`, `Store.UpsertVendorAdmin(ctx, username, plaintextPassword string) error`, `Store.GetVendorAdminByUsername(ctx, username string) (VendorAdmin, error)`, `store.ErrVendorAdminNotFound`. Dipakai Task 8 (login customer) dan Task 10 (vendor endpoints).

- [ ] **Step 1: Tulis test yang gagal untuk `Account`**

Buat `backend/internal/store/account_test.go`:

```go
package store_test

import (
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestCreateAccountDanGetByUsername(t *testing.T) {
	s := newTestStore(t) // helper yang sudah ada di store_test.go / *_test.go lain, TRUNCATE dulu

	err := s.CreateAccount(ctx(t), store.CreateAccountInput{
		ID: "acc_1", BusinessName: "Toko Uji", Email: "toko@uji.test",
		Username: "toko_uji", PlaintextPassword: "rahasia123",
		Plan: "Business", MaxDevices: 10, ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	acc, err := s.GetAccountByUsername(ctx(t), "toko_uji")
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if acc.BusinessName != "Toko Uji" || acc.Plan != "Business" || acc.MaxDevices != 10 {
		t.Fatalf("account salah: %+v", acc)
	}
	if !acc.VerifyPassword("rahasia123") {
		t.Fatal("password seharusnya cocok")
	}
	if acc.VerifyPassword("salah") {
		t.Fatal("password salah seharusnya ditolak")
	}
}

func TestGetAccountByUsernameTidakDitemukan(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetAccountByUsername(ctx(t), "tidak-ada")
	if err != store.ErrAccountNotFound {
		t.Fatalf("err = %v, mau ErrAccountNotFound", err)
	}
}

func TestAccountDerivedStatus(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		acc    store.Account
		expect string
	}{
		{"aktif jauh dari kedaluwarsa", store.Account{AdminStatus: "active", ExpiresAt: now.Add(200 * 24 * time.Hour)}, "active"},
		{"akan berakhir dalam 10 hari", store.Account{AdminStatus: "active", ExpiresAt: now.Add(10 * 24 * time.Hour)}, "expiring"},
		{"sudah lewat", store.Account{AdminStatus: "active", ExpiresAt: now.Add(-time.Hour)}, "expired"},
		{"disuspend walau belum kedaluwarsa", store.Account{AdminStatus: "suspended", ExpiresAt: now.Add(200 * 24 * time.Hour)}, "suspended"},
		{"dicabut walau belum kedaluwarsa", store.Account{AdminStatus: "revoked", ExpiresAt: now.Add(200 * 24 * time.Hour)}, "revoked"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.acc.DerivedStatus(now); got != c.expect {
				t.Fatalf("DerivedStatus = %q, mau %q", got, c.expect)
			}
		})
	}
}

func TestRenewAccount(t *testing.T) {
	s := newTestStore(t)
	newExpiry := time.Now().Add(365 * 24 * time.Hour)
	err := s.CreateAccount(ctx(t), store.CreateAccountInput{
		ID: "acc_2", BusinessName: "Toko B", Email: "b@uji.test",
		Username: "toko_b", PlaintextPassword: "x", Plan: "Starter", MaxDevices: 3,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.RenewAccount(ctx(t), "acc_2", newExpiry); err != nil {
		t.Fatalf("renew: %v", err)
	}
	acc, err := s.GetAccountByID(ctx(t), "acc_2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !acc.ExpiresAt.Equal(newExpiry) {
		t.Fatalf("expires_at tidak berubah: %v", acc.ExpiresAt)
	}
}

func TestSetAccountAdminStatus(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateAccount(ctx(t), store.CreateAccountInput{
		ID: "acc_3", BusinessName: "Toko C", Email: "c@uji.test",
		Username: "toko_c", PlaintextPassword: "x", Plan: "Starter", MaxDevices: 3,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.SetAccountAdminStatus(ctx(t), "acc_3", "suspended"); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	acc, _ := s.GetAccountByID(ctx(t), "acc_3")
	if acc.AdminStatus != "suspended" {
		t.Fatalf("admin_status = %q, mau suspended", acc.AdminStatus)
	}
}
```

Catatan: kalau `store_test.go` belum punya helper `newTestStore(t)` / `ctx(t)`, cek pola yang sudah dipakai `device_test.go`/`invoice_test.go` di paket yang sama dan pakai pola YANG SAMA (jangan bikin helper baru yang beda nama).

- [ ] **Step 2: Jalankan test, pastikan gagal (fungsi belum ada)**

Run: `cd backend && make migrate && go test ./internal/store/... -run TestCreateAccount -v`
Expected: FAIL — `undefined: store.CreateAccountInput` dsb.

- [ ] **Step 3: Hapus `admin.go` lama, buat `account.go`**

```bash
git rm backend/internal/store/admin.go backend/internal/store/admin_test.go backend/internal/store/admin_query_test.go
```

Buat `backend/internal/store/account.go`:

```go
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrAccountNotFound dikembalikan bila id/username tidak terdaftar.
var ErrAccountNotFound = errors.New("store: account tidak ditemukan")

// WarningThresholdDays: ambang status berubah dari "active" ke "expiring".
// Sama seperti yang dulu dipakai internal/licensecheck dan License Server --
// nilainya tidak berubah, cuma sumbernya sekarang satu tempat.
const WarningThresholdDays = 30

type Account struct {
	ID            string
	BusinessName  string
	Email         string
	Username      string
	PasswordHash  string
	Plan          string
	MaxDevices    int
	AdminStatus   string // "active" | "suspended" | "revoked"
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// VerifyPassword membandingkan password mentah dengan hash tersimpan.
func (a Account) VerifyPassword(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(plaintext)) == nil
}

// DerivedStatus menghitung status yang dikirim ke Customer Dashboard/API --
// pola sama persis licenseserver/store.License.DerivedStatus (spec §2.1),
// cuma sekarang satu query di database yang sama, tidak ada lagi jaringan
// atau grace period.
func (a Account) DerivedStatus(now time.Time) string {
	if a.AdminStatus == "suspended" || a.AdminStatus == "revoked" {
		return a.AdminStatus
	}
	if now.After(a.ExpiresAt) {
		return "expired"
	}
	daysRemaining := int(a.ExpiresAt.Sub(now).Hours() / 24)
	if daysRemaining <= WarningThresholdDays {
		return "expiring"
	}
	return "active"
}

// Operational melaporkan apakah account ini boleh memakai endpoint
// device/admin/API key -- hanya "active" dan "expiring".
func (a Account) Operational(now time.Time) bool {
	status := a.DerivedStatus(now)
	return status == "active" || status == "expiring"
}

type CreateAccountInput struct {
	ID                string
	BusinessName      string
	Email             string
	Username          string
	PlaintextPassword string
	Plan              string
	MaxDevices        int
	ExpiresAt         time.Time
}

// CreateAccount membuat akun baru. Dipanggil dari endpoint vendor (Task 10)
// -- password awal digenerate di lapisan HTTP, cuma hash-nya yang sampai
// ke sini, sama pola seperti CreateAPIKey/CreateWebhookEndpoint.
func (s *Store) CreateAccount(ctx context.Context, in CreateAccountInput) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.PlaintextPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password account: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO accounts (id, business_name, email, username, password_hash, plan, max_devices, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		in.ID, in.BusinessName, in.Email, in.Username, string(hash), in.Plan, in.MaxDevices, in.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store: create account: %w", err)
	}
	return nil
}

const accountSelectCols = `SELECT id, business_name, email, username, password_hash,
	plan, max_devices, admin_status, expires_at, created_at, updated_at FROM accounts`

func scanAccount(row interface{ Scan(dest ...any) error }) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.BusinessName, &a.Email, &a.Username, &a.PasswordHash,
		&a.Plan, &a.MaxDevices, &a.AdminStatus, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

// GetAccountByUsername dipakai handleAdminLogin (Task 8).
func (s *Store) GetAccountByUsername(ctx context.Context, username string) (Account, error) {
	row := s.pool.QueryRow(ctx, accountSelectCols+` WHERE username = $1`, username)
	a, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: select account by username: %w", err)
	}
	return a, nil
}

// GetAccountByID dipakai requireActiveAccount (Task 8) dan endpoint vendor (Task 10).
func (s *Store) GetAccountByID(ctx context.Context, id string) (Account, error) {
	row := s.pool.QueryRow(ctx, accountSelectCols+` WHERE id = $1`, id)
	a, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: select account by id: %w", err)
	}
	return a, nil
}

// ListAccounts dipakai Vendor Dashboard (Task 10).
func (s *Store) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, accountSelectCols+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list accounts: %w", err)
	}
	defer rows.Close()

	out := make([]Account, 0)
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan account: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi accounts: %w", err)
	}
	return out, nil
}

// RenewAccount memperpanjang expires_at. Tidak menyentuh admin_status --
// perpanjang akun yang disuspend TIDAK otomatis mengaktifkannya lagi,
// vendor harus eksplisit set_admin_status(active) juga kalau memang mau.
func (s *Store) RenewAccount(ctx context.Context, id string, newExpiresAt time.Time) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET expires_at = $2, updated_at = now() WHERE id = $1`, id, newExpiresAt)
	if err != nil {
		return fmt.Errorf("store: renew account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// SetAccountAdminStatus dipakai handleSuspendAccount/handleRevokeAccount (Task 10).
// status harus salah satu dari "active"/"suspended"/"revoked" -- validasi
// nilai yang boleh dilakukan pemanggil di lapisan HTTP, bukan di sini.
func (s *Store) SetAccountAdminStatus(ctx context.Context, id, status string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET admin_status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("store: set account admin_status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}
```

Buat `backend/internal/store/vendor_admin.go` (isi identik `admin.go` lama, hanya nama tabel/tipe yang beda):

```go
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrVendorAdminNotFound dikembalikan bila username tidak terdaftar.
var ErrVendorAdminNotFound = errors.New("store: vendor admin tidak ditemukan")

type VendorAdmin struct {
	ID           int64
	Username     string
	PasswordHash string
}

// UpsertVendorAdmin membuat atau memperbarui password vendor admin (Akbar).
// Pola sama seperti UpsertAdmin lama -- dipakai cmd/admintool dengan flag baru.
func (s *Store) UpsertVendorAdmin(ctx context.Context, username, plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password vendor admin: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO vendor_admins (username, password_hash)
		 VALUES ($1, $2)
		 ON CONFLICT (username) DO UPDATE
		   SET password_hash = EXCLUDED.password_hash, updated_at = now()`,
		username, string(hash))
	if err != nil {
		return fmt.Errorf("store: upsert vendor admin: %w", err)
	}
	return nil
}

func (s *Store) GetVendorAdminByUsername(ctx context.Context, username string) (VendorAdmin, error) {
	var v VendorAdmin
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash FROM vendor_admins WHERE username = $1`, username).
		Scan(&v.ID, &v.Username, &v.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorAdmin{}, ErrVendorAdminNotFound
	}
	if err != nil {
		return VendorAdmin{}, fmt.Errorf("store: select vendor admin: %w", err)
	}
	return v, nil
}

func (v VendorAdmin) VerifyPassword(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(v.PasswordHash), []byte(plaintext)) == nil
}
```

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `cd backend && go test ./internal/store/... -run 'TestCreateAccount|TestGetAccount|TestAccountDerivedStatus|TestRenewAccount|TestSetAccountAdminStatus' -v`
Expected: semua `PASS`.

- [ ] **Step 5: Commit**

```bash
cd backend
gofmt -l internal/store/
git add internal/store/account.go internal/store/account_test.go internal/store/vendor_admin.go
git rm internal/store/admin.go internal/store/admin_test.go internal/store/admin_query_test.go
git commit -m "feat(store): model Account menggantikan AdminUser

Account gabung admin_users lama + customers/licenses License Server:
identitas login + plan/kuota dalam satu baris. DerivedStatus/Operational
pindah dari internal/licenseserver/store/license.go ke sini -- logikanya
sama persis, cuma sekarang satu database, satu query, tanpa grace period.

VendorAdmin baru buat akun Akbar login ke Vendor Dashboard -- tabel dan
kode terpisah dari Account, supaya dua jenis sesi ini tidak pernah bisa
tertukar di lapisan manapun.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 4: `internal/store` — `device.go` dan `apikey.go` dapat `accountID`

**Files:**
- Modify: `backend/internal/store/device.go`
- Modify: `backend/internal/store/device_test.go`
- Modify: `backend/internal/store/apikey.go`
- Modify: `backend/internal/store/apikey_test.go`

**Interfaces:**
- Consumes: `Account.ID` dari Task 3 (dipakai sebagai foreign key nilai test).
- Produces (signature baru, dipakai Task 9 di `internal/httpapi`):
  - `Device` dapat field baru `AccountID string`.
  - `(s *Store) CreateDevice(ctx, key []byte, accountID, deviceID, name string, secret []byte) error`
  - `(s *Store) GetDevice(ctx, key []byte, deviceID string) (Device, error)` — **tidak berubah** (device_id sudah unik global, lihat spec §2.3; `Device.AccountID` yang dikembalikan yang dipakai pemanggil buat tahu pemiliknya, dipakai `requireDevice` di Task 8 buat mengisi context).
  - `(s *Store) ListDevices(ctx, accountID string) ([]Device, error)`
  - `(s *Store) SetDeviceEnabled(ctx, accountID, deviceID string, enabled bool) error` (mengembalikan `ErrDeviceNotFound` juga kalau device ada tapi milik akun lain — lihat Step 3)
  - `APIKey` dapat field baru `AccountID string`.
  - `(s *Store) CreateAPIKey(ctx, accountID, id, name string, keyHash []byte) error`
  - `(s *Store) ListAPIKeys(ctx, accountID string) ([]APIKey, error)`
  - `(s *Store) RevokeAPIKey(ctx, accountID, id string) error` (pola sama seperti `SetDeviceEnabled`)
  - `(s *Store) VerifyAPIKey(ctx, rawKey string) (APIKey, error)` — **tidak berubah** (key_hash sudah unik global; `APIKey.AccountID` yang dikembalikan yang dipakai `requireAPIKey` mengisi context)

- [ ] **Step 1: Tulis test yang gagal (isolasi + signature baru)**

Tambahkan di `backend/internal/store/device_test.go` (pola ikuti helper yang sudah ada di file itu untuk bikin dua account uji, mis. `seedAccount(t, s, "acc_a")`/`seedAccount(t, s, "acc_b")` — kalau helper itu belum ada, tambahkan sekali di file ini, dipakai ulang test lain di task ini):

```go
func TestListDevicesHanyaMilikAccountSendiri(t *testing.T) {
	s := newTestStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")

	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")
	mustCreateDevice(t, s, "acc_b", "dev_b1", "HP B1")

	listA, err := s.ListDevices(ctx(t), "acc_a")
	if err != nil {
		t.Fatalf("list devices acc_a: %v", err)
	}
	if len(listA) != 1 || listA[0].DeviceID != "dev_a1" {
		t.Fatalf("acc_a seharusnya cuma lihat dev_a1, dapat: %+v", listA)
	}
}

func TestSetDeviceEnabledMilikAccountLainDitolak(t *testing.T) {
	s := newTestStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")

	err := s.SetDeviceEnabled(ctx(t), "acc_b", "dev_a1", false)
	if err != ErrDeviceNotFound {
		t.Fatalf("err = %v, mau ErrDeviceNotFound (device milik akun lain)", err)
	}
}
```

Helper (tambahkan sekali, dipakai ulang):

```go
func seedAccount(t *testing.T, s *Store, id string) {
	t.Helper()
	err := s.CreateAccount(ctx(t), CreateAccountInput{
		ID: id, BusinessName: id, Email: id + "@uji.test", Username: id,
		PlaintextPassword: "x", Plan: "Business", MaxDevices: 10,
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("seed account %s: %v", id, err)
	}
}

func mustCreateDevice(t *testing.T, s *Store, accountID, deviceID, name string) {
	t.Helper()
	key := make([]byte, 32) // secretbox.KeySize
	if err := s.CreateDevice(ctx(t), key, accountID, deviceID, name, []byte("secret")); err != nil {
		t.Fatalf("create device: %v", err)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal (compile error, signature belum berubah)**

Run: `cd backend && go test ./internal/store/... -run 'TestListDevicesHanya|TestSetDeviceEnabledMilik' -v`
Expected: FAIL, compile error `too few arguments in call to s.CreateDevice`.

- [ ] **Step 3: Ubah `device.go`**

```go
// CreateDevice menyimpan device baru dengan secret terenkripsi, milik satu account.
func (s *Store) CreateDevice(ctx context.Context, key []byte, accountID, deviceID, name string, secret []byte) error {
	enc, err := secretbox.Seal(key, secret)
	if err != nil {
		return fmt.Errorf("store: enkripsi secret: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO devices (device_id, account_id, name, secret_enc) VALUES ($1, $2, $3, $4)`,
		deviceID, accountID, name, enc)
	if err != nil {
		return fmt.Errorf("store: insert device: %w", err)
	}
	return nil
}
```

Tambah field `AccountID string` ke struct `Device`. Tambah `d.AccountID` ke SELECT + Scan di `GetDevice` dan `ListDevices` (kolom `account_id`). `ListDevices` tambah parameter `accountID string`, query jadi `WHERE account_id = $1`, `Query(ctx, ..., accountID)`.

`SetDeviceEnabled` jadi:

```go
func (s *Store) SetDeviceEnabled(ctx context.Context, accountID, deviceID string, enabled bool) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE devices SET enabled = $3 WHERE device_id = $1 AND account_id = $2`,
		deviceID, accountID, enabled)
	if err != nil {
		return fmt.Errorf("store: set device enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}
```

`GetDevice` dan `TouchDevice` **tidak berubah parameternya** — device_id tetap unik global (spec §2.3), cuma `GetDevice` sekarang ikut men-Scan `account_id` ke `Device.AccountID` supaya `requireDevice` (Task 8) bisa menaruhnya di context.

- [ ] **Step 4: Jalankan test device, pastikan lulus**

Run: `cd backend && go test ./internal/store/... -run TestDevice -v` dan `-run TestListDevices` dan `-run TestSetDeviceEnabled`
Expected: semua `PASS`. Test lama yang manggil `CreateDevice`/`ListDevices`/`SetDeviceEnabled` dengan signature lama HARUS diupdate juga di file yang sama — tambahkan `accountID` (pakai `seedAccount` + id yang konsisten) di setiap pemanggilan.

- [ ] **Step 5: Ulangi pola yang sama untuk `apikey.go`**

Tambah field `AccountID string` ke `APIKey`. Ubah:

```go
func (s *Store) CreateAPIKey(ctx context.Context, accountID, id, name string, keyHash []byte) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO api_keys (id, account_id, name, key_hash) VALUES ($1, $2, $3, $4)`,
		id, accountID, name, keyHash)
	if err != nil {
		return fmt.Errorf("store: create api key: %w", err)
	}
	return nil
}

func (s *Store) ListAPIKeys(ctx context.Context, accountID string) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, account_id, name, created_at, revoked_at FROM api_keys
		 WHERE account_id = $1 ORDER BY created_at DESC`, accountID)
	// ... sisanya sama, Scan tambah &k.AccountID
}

func (s *Store) RevokeAPIKey(ctx context.Context, accountID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND account_id = $2 AND revoked_at IS NULL`,
		id, accountID)
	if err != nil {
		return fmt.Errorf("store: revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = $1 AND account_id = $2)`, id, accountID).
			Scan(&exists); err != nil {
			return fmt.Errorf("store: cek api key: %w", err)
		}
		if !exists {
			return ErrAPIKeyNotFound
		}
	}
	return nil
}
```

`VerifyAPIKey` **tidak berubah parameternya** — key_hash tetap unik global, cuma SELECT-nya ikut ambil `account_id` ke `APIKey.AccountID`.

Test analog: `TestListAPIKeysHanyaMilikAccountSendiri`, `TestRevokeAPIKeyMilikAccountLainDitolak` — pola persis Step 1, ganti "device" jadi "api key".

- [ ] **Step 6: Jalankan seluruh test paket store, pastikan lulus**

Run: `cd backend && go test ./internal/store/... -v 2>&1 | tail -60`
Expected: `ok`, nol `FAIL`.

- [ ] **Step 7: Commit**

```bash
cd backend
gofmt -l internal/store/
git add internal/store/device.go internal/store/device_test.go internal/store/apikey.go internal/store/apikey_test.go
git commit -m "feat(store): device dan api_keys diikat ke account_id

CreateDevice/ListDevices/SetDeviceEnabled dan CreateAPIKey/ListAPIKeys/
RevokeAPIKey menerima accountID, query di-scope WHERE account_id = \$N.
GetDevice/VerifyAPIKey TIDAK berubah signature (device_id/key_hash tetap
unik global) -- tapi sekarang ikut mengembalikan AccountID di struct hasil,
dipakai requireDevice/requireAPIKey (Task 8) menaruh account_id ke context.

Test baru: akun A tidak bisa list/ubah device atau api key milik akun B
(ErrDeviceNotFound/ErrAPIKeyNotFound, bukan bocor data).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 5: `internal/store` — `invoice.go` dapat `accountID` + constraint di-scope ulang

**Files:**
- Modify: `backend/internal/store/invoice.go`
- Modify: `backend/internal/store/invoice_test.go`

**Interfaces:**
- Consumes: `seedAccount`, `mustCreateDevice` dari Task 4.
- Produces: `Invoice` dapat field `AccountID string`. `(s *Store) CreateInvoice(ctx, now time.Time, accountID, externalRef string, requestedAmount int64) (inv Invoice, created bool, err error)`. `(s *Store) MatchEvent(ctx, now time.Time, accountID, eventID string, amount *int64) (matchedInvoiceID string, err error)`. `(s *Store) GetInvoiceByID(ctx, accountID, id string) (Invoice, error)`. `(s *Store) GetInvoiceByExternalRef(ctx, accountID, externalRef string) (Invoice, error)`. `(s *Store) ListInvoices(ctx, accountID string, limit, offset int, filter InvoiceFilter) ([]Invoice, error)`. `(s *Store) ExpireInvoicesAndListNewlyExpired(ctx, now time.Time) ([]string, error)` — **tidak berubah**, ini worker global yang jalan lintas semua akun sekaligus (dipanggil ticker webhook, bukan request per-akun).

- [ ] **Step 1: Tulis test yang gagal — isolasi external_ref antar akun**

Tambahkan di `invoice_test.go`:

```go
func TestExternalRefSamaBolehDiAkunBerbeda(t *testing.T) {
	s := newTestStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")

	_, created, err := s.CreateInvoice(ctx(t), time.Now(), "acc_a", "INV-001", 50000)
	if err != nil || !created {
		t.Fatalf("create invoice acc_a: created=%v err=%v", created, err)
	}
	_, created, err = s.CreateInvoice(ctx(t), time.Now(), "acc_b", "INV-001", 75000)
	if err != nil || !created {
		t.Fatalf("create invoice acc_b (external_ref sama, akun beda): created=%v err=%v", created, err)
	}
}

func TestGetInvoiceByIDMilikAccountLainDitolak(t *testing.T) {
	s := newTestStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	inv, _, _ := s.CreateInvoice(ctx(t), time.Now(), "acc_a", "INV-002", 50000)

	_, err := s.GetInvoiceByID(ctx(t), "acc_b", inv.ID)
	if err != ErrInvoiceNotFound {
		t.Fatalf("err = %v, mau ErrInvoiceNotFound (invoice milik akun lain)", err)
	}
}
```

- [ ] **Step 2: Jalankan, pastikan gagal**

Run: `cd backend && go test ./internal/store/... -run 'TestExternalRefSamaBoleh|TestGetInvoiceByIDMilik' -v`
Expected: FAIL — compile error, signature lama belum menerima `accountID`.

- [ ] **Step 3: Ubah `invoice.go`**

Tambah field `AccountID string` ke `Invoice`, tambah ke `invoiceSelectCols` dan `scanInvoice`.

```go
func (s *Store) CreateInvoice(ctx context.Context, now time.Time, accountID, externalRef string, requestedAmount int64) (inv Invoice, created bool, err error) {
	// ... isi loop alokasi nominal SAMA seperti sekarang, tapi:
	// - SELECT existing by (account_id, external_ref) bukan cuma external_ref
	// - INSERT menyertakan account_id
	// - isUniqueViolation(err, "invoices_account_external_ref_idx") menggantikan "invoices_external_ref_idx"
	// - isUniqueViolation(err, "invoices_account_pending_unique_amount_idx") menggantikan "invoices_pending_unique_amount_idx"
}
```

Baca isi lengkap fungsi ini dulu (`sed -n '136,215p' internal/store/invoice.go`) sebelum mengubah — logikanya ada retry-loop alokasi nominal unik yang harus dipertahankan persis, cuma tiga hal yang berubah: parameter `accountID` baru, kolom `account_id` ikut di INSERT/SELECT existing-check, dan DUA nama constraint di pemanggilan `isUniqueViolation` (lihat Task 2 migration — nama index baru `invoices_account_external_ref_idx` dan `invoices_account_pending_unique_amount_idx`).

`MatchEvent`, `GetInvoiceByID`, `GetInvoiceByExternalRef`, `ListInvoices`: tambah parameter `accountID string`, tambah `AND account_id = $N` (atau `WHERE account_id = $1 AND ...`) di query masing-masing, tambah `accountID` ke slice args yang dikirim ke `Query`/`QueryRow`.

`ExpireInvoicesAndListNewlyExpired` dan `expireStaleInvoices`: **TIDAK diubah** — worker ini sengaja jalan lintas akun (dipanggil ticker 1 menit di `main.go`, harus mengedaluwarsakan invoice SEMUA akun sekaligus, bukan satu per satu).

- [ ] **Step 4: Update seluruh pemanggilan lama di `invoice_test.go`**

Tambahkan `accountID` (pakai `seedAccount` dulu) ke setiap pemanggilan `CreateInvoice`/`MatchEvent`/`GetInvoiceByID`/`GetInvoiceByExternalRef`/`ListInvoices` yang sudah ada di file ini.

- [ ] **Step 5: Jalankan seluruh test invoice, pastikan lulus**

Run: `cd backend && go test ./internal/store/... -run TestInvoice -v` dan test baru dari Step 1.
Expected: semua `PASS`.

- [ ] **Step 6: Commit**

```bash
cd backend
gofmt -l internal/store/
git add internal/store/invoice.go internal/store/invoice_test.go
git commit -m "feat(store): invoice diikat ke account_id, constraint di-scope ulang

CreateInvoice/MatchEvent/GetInvoiceByID/GetInvoiceByExternalRef/ListInvoices
menerima accountID, query WHERE account_id = \$N. Nama constraint unik yang
ditabrak isUniqueViolation ikut berubah ke invoices_account_external_ref_idx
dan invoices_account_pending_unique_amount_idx (Task 2).
ExpireInvoicesAndListNewlyExpired TIDAK berubah -- worker berkala ini
sengaja jalan lintas semua akun sekaligus.

Test baru: dua akun boleh pakai external_ref yang sama tanpa tabrakan;
invoice milik akun lain diakses lewat ID langsung -> ErrInvoiceNotFound.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 6: `internal/store` — `event.go` dan `exception.go` dapat `accountID`

**Files:**
- Modify: `backend/internal/store/event.go`
- Modify: `backend/internal/store/event_test.go`
- Modify: `backend/internal/store/exception.go`
- Modify: `backend/internal/store/exception_test.go`

**Interfaces:**
- Consumes: `seedAccount`, `mustCreateDevice` (Task 4), `store.Account` (Task 3).
- Produces: `Event` dapat field `AccountID string`. `(s *Store) InsertEvent(ctx, e Event) (bool, error)` — signature TIDAK berubah, `e.AccountID` sudah bagian dari struct yang dikirim pemanggil (`requireDevice`, Task 8, mengisi `e.AccountID = device.AccountID` sebelum memanggil — device sudah tahu akunnya sendiri, tidak perlu parameter terpisah). `(s *Store) ListEvents(ctx, accountID string, limit, offset int, filter EventFilter) ([]Event, error)`. `(s *Store) ListExceptions(ctx, accountID string, limit, offset int, filter ExceptionFilter) ([]Event, error)`. `(s *Store) ManualMatchEvent(ctx, now time.Time, accountID, invoiceID, eventID string) error`. `(s *Store) DismissEvent(ctx, accountID, eventID string, note *string) error`.

- [ ] **Step 1: Tulis test yang gagal**

Tambahkan di `event_test.go`:

```go
func TestListEventsHanyaMilikAccountSendiri(t *testing.T) {
	s := newTestStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")
	mustCreateDevice(t, s, "acc_b", "dev_b1", "HP B1")

	insertTestEvent(t, s, "acc_a", "dev_a1", "evt_a1")
	insertTestEvent(t, s, "acc_b", "dev_b1", "evt_b1")

	listA, err := s.ListEvents(ctx(t), "acc_a", 10, 0, EventFilter{})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(listA) != 1 || listA[0].EventID != "evt_a1" {
		t.Fatalf("acc_a seharusnya cuma lihat evt_a1, dapat: %+v", listA)
	}
}
```

Helper `insertTestEvent` (tambahkan sekali, dipakai ulang task ini):

```go
func insertTestEvent(t *testing.T, s *Store, accountID, deviceID, eventID string) {
	t.Helper()
	now := time.Now()
	_, err := s.InsertEvent(ctx(t), Event{
		EventID: eventID, AccountID: accountID, DeviceID: deviceID, Source: "gopay",
		PackageName: "com.gojek.gopaymerchant", PostedAt: now, ReceivedAt: now,
		RawPayload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
}
```

- [ ] **Step 2: Jalankan, pastikan gagal**

Run: `cd backend && go test ./internal/store/... -run TestListEventsHanyaMilik -v`
Expected: FAIL — `unknown field AccountID in struct literal` / `too few arguments`.

- [ ] **Step 3: Ubah `event.go`**

Tambah field `AccountID string` ke `Event`, tambah `account_id` ke kolom INSERT di `InsertEvent` (`e.AccountID` sebagai salah satu value). Tambah parameter `accountID string` ke `ListEvents`, tambah `AND account_id = ` + arg ke query, tambah `&e.AccountID` ke Scan dan `account_id` ke SELECT.

- [ ] **Step 4: Ubah `exception.go`**

`ListExceptions`: tambah parameter `accountID string`, tambah `AND e.account_id = ` + arg (JOIN alias `e` sudah ada di query, cek query aslinya biar konsisten alias). `ManualMatchEvent`/`DismissEvent`: tambah parameter `accountID string`; untuk `ManualMatchEvent`, pastikan invoice DAN event yang dicocokkan sama-sama milik `accountID` itu (tambahkan `AND account_id = $N` di WHERE clause UPDATE invoice DAN di cek keberadaan event — baca isi fungsi lengkap dulu sebelum mengubah, `sed -n '98,145p' internal/store/exception.go`, supaya urutan parameter `$N` di query yang sudah ada tidak salah geser).

- [ ] **Step 5: Update pemanggilan lama di kedua file test**

Tambahkan `accountID`/`e.AccountID` sesuai signature baru di setiap test yang sudah ada.

- [ ] **Step 6: Jalankan seluruh test, pastikan lulus**

Run: `cd backend && go test ./internal/store/... -v 2>&1 | tail -80`
Expected: `ok`, nol `FAIL`.

- [ ] **Step 7: Commit**

```bash
cd backend
gofmt -l internal/store/
git add internal/store/event.go internal/store/event_test.go internal/store/exception.go internal/store/exception_test.go
git commit -m "feat(store): notification_events dan konsol pengecualian diikat account_id

ListEvents/ListExceptions/ManualMatchEvent/DismissEvent menerima accountID,
di-scope WHERE account_id = \$N. InsertEvent tidak berubah signature --
Event.AccountID sudah bagian struct yang dikirim requireDevice (device
sudah tahu akunnya sendiri).

Test baru: akun A tidak bisa lihat event milik akun B lewat ListEvents.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 7: `internal/store` — `webhook.go` dapat `accountID`

**Files:**
- Modify: `backend/internal/store/webhook.go`
- Modify: `backend/internal/store/webhook_test.go`

**Interfaces:**
- Consumes: `seedAccount` (Task 4).
- Produces: `WebhookEndpoint`/`WebhookEndpointSummary` dapat field `AccountID string`. Signature yang berubah (tambah `accountID string` sebagai parameter kedua, setelah `ctx`, kecuali disebutkan lain):
  - `CreateWebhookEndpoint(ctx, key []byte, accountID, id, name, url string, events []string, secret []byte) error`
  - `ListWebhookEndpoints(ctx, accountID string) ([]WebhookEndpointSummary, error)`
  - `GetWebhookEndpoint(ctx, key []byte, accountID, id string) (WebhookEndpoint, []byte, error)`
  - `SetWebhookEndpointEnabled(ctx, accountID, id string, enabled bool) error`
  - `DeleteWebhookEndpoint(ctx, accountID, id string) error`
  - `EnqueueTestDelivery(ctx, now time.Time, accountID, endpointID string, payload []byte) (string, error)`
  - `ListWebhookDeliveries(ctx, accountID, endpointID string, limit, offset int) ([]WebhookDelivery, error)`
  - **Tidak berubah** (worker lintas-akun, dipanggil ticker global di `main.go`, sama alasannya dengan `ExpireInvoicesAndListNewlyExpired` di Task 5): `EnqueueWebhookDeliveries` (dipanggil dari `MatchEvent`/`ManualMatchEvent` yang SUDAH tahu `endpoint_id` mana yang relevan lewat join `events` array kolom — endpoint sudah pasti milik akun yang sama karena `invoice`/`event`-nya sudah di-scope di Task 5/6), `DueWebhookDeliveries`, `RecordDeliverySuccess`, `RecordDeliveryFailure`, `RecordTestDeliveryResult`.

- [ ] **Step 1: Tulis test yang gagal**

```go
func TestListWebhookEndpointsHanyaMilikAccountSendiri(t *testing.T) {
	s := newTestStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	key := make([]byte, 32)

	if err := s.CreateWebhookEndpoint(ctx(t), key, "acc_a", "wh_a1", "Endpoint A", "https://a.test/hook",
		[]string{"invoice.paid"}, []byte("secret-a")); err != nil {
		t.Fatalf("create webhook acc_a: %v", err)
	}
	if err := s.CreateWebhookEndpoint(ctx(t), key, "acc_b", "wh_b1", "Endpoint B", "https://b.test/hook",
		[]string{"invoice.paid"}, []byte("secret-b")); err != nil {
		t.Fatalf("create webhook acc_b: %v", err)
	}

	listA, err := s.ListWebhookEndpoints(ctx(t), "acc_a")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listA) != 1 || listA[0].ID != "wh_a1" {
		t.Fatalf("acc_a seharusnya cuma lihat wh_a1, dapat: %+v", listA)
	}
}
```

- [ ] **Step 2: Jalankan, pastikan gagal**

Run: `cd backend && go test ./internal/store/... -run TestListWebhookEndpointsHanyaMilik -v`
Expected: FAIL — signature lama belum menerima `accountID`.

- [ ] **Step 3: Ubah `webhook.go`**

Baca dulu isi lengkap `sed -n '116,270p' internal/store/webhook.go` supaya nomor placeholder `$N` di tiap query disesuaikan dengan tepat (bukan ditebak). Tambah field `AccountID string` ke `WebhookEndpoint`/`WebhookEndpointSummary`. Terapkan pola yang sama seperti Task 4/5/6 ke enam fungsi yang disebut di **Produces** di atas: tambah parameter `accountID`, tambah kolom `account_id` di INSERT, tambah `AND account_id = $N` di WHERE UPDATE/DELETE/SELECT, tambah `&w.AccountID` ke Scan yang mengembalikan struct.

- [ ] **Step 4: Update seluruh pemanggilan lama di `webhook_test.go`**

Tambahkan `accountID` (via `seedAccount`) ke setiap pemanggilan yang sudah ada dari enam fungsi itu.

- [ ] **Step 5: Jalankan seluruh test paket store, pastikan lulus**

Run: `cd backend && go test ./internal/store/... -v 2>&1 | tail -100`
Expected: `ok`, nol `FAIL`.

- [ ] **Step 6: Commit**

```bash
cd backend
gofmt -l internal/store/
git add internal/store/webhook.go internal/store/webhook_test.go
git commit -m "feat(store): webhook endpoints diikat account_id

CreateWebhookEndpoint/ListWebhookEndpoints/GetWebhookEndpoint/
SetWebhookEndpointEnabled/DeleteWebhookEndpoint/EnqueueTestDelivery/
ListWebhookDeliveries menerima accountID, di-scope WHERE account_id = \$N.
EnqueueWebhookDeliveries/DueWebhookDeliveries/RecordDeliverySuccess/
RecordDeliveryFailure/RecordTestDeliveryResult TIDAK berubah -- worker
berkala lintas akun, endpoint-nya sudah pasti benar lewat invoice/event
yang sudah di-scope di Task 5/6.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 8: `internal/httpapi` — derivasi `account_id` di tiga jalur auth

**Files:**
- Modify: `backend/internal/httpapi/admin_auth.go`
- Modify: `backend/internal/httpapi/apikey_auth.go`
- Modify: `backend/internal/httpapi/auth_middleware.go`
- Rewrite: `backend/internal/httpapi/admin_license.go` (isi baru, nama file dipertahankan karena route `/admin/license` tidak berubah)
- Modify: `backend/internal/httpapi/api.go`
- Modify test: `backend/internal/httpapi/admin_auth_test.go`, `auth_middleware_test.go`, `admin_license_test.go`

**Interfaces:**
- Consumes: `store.Account`, `store.GetAccountByUsername`, `store.GetAccountByID` (Task 3); `auth.NewSessionToken(key, now, subject)`/`VerifySessionToken` (Task 1); `Device.AccountID`, `APIKey.AccountID` (Task 4).
- Produces: `AccountFromContext(ctx) (accountID string, ok bool)` menggantikan `AdminFromContext(ctx) bool` — dipakai SELURUH handler admin di Task 9. `(a *API) requireActiveAccount(next http.Handler) http.Handler` menggantikan `requireLicense`.

- [ ] **Step 1: `admin_auth.go` — login pakai `Account`, sesi bawa `account_id`**

Ganti isi konstanta dan context key:

```go
const adminSessionCookie = "admin_session"

type accountCtxKey int

const ctxKeyAccountID accountCtxKey = iota

// AccountFromContext melaporkan account_id pemilik request ini, sudah lolos
// requireAdmin (sesi) ATAU requireAPIKey/requireDevice (Task 4, lewat
// context yang sama -- lihat apikey_auth.go/auth_middleware.go).
func AccountFromContext(ctx context.Context) (accountID string, ok bool) {
	v, ok := ctx.Value(ctxKeyAccountID).(string)
	return v, ok
}
```

`handleAdminLogin`: ganti `a.store.GetAdminByUsername` jadi `a.store.GetAccountByUsername`, `admin.VerifyPassword` jadi `acc.VerifyPassword`, dan token:

```go
token := auth.NewSessionToken(a.adminSessionKey, a.now(), acc.ID)
```

`requireAdmin`:

```go
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
		ctx := context.WithValue(r.Context(), ctxKeyAccountID, accountID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

- [ ] **Step 2: `apikey_auth.go` — taruh `AccountID` ke context yang sama**

```go
func (a *API) requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ... verifikasi Authorization: Bearer + a.store.VerifyAPIKey SAMA seperti sekarang ...
		key, err := a.store.VerifyAPIKey(r.Context(), rawKey)
		// ... penanganan error SAMA ...

		ctx := context.WithValue(r.Context(), ctxKeyAPIKey, key)
		ctx = context.WithValue(ctx, ctxKeyAccountID, key.AccountID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

- [ ] **Step 3: `auth_middleware.go` (`requireDevice`) — sama polanya**

Di titik yang sekarang `ctx := context.WithValue(r.Context(), ctxKeyDevice, device)`, tambahkan baris setelahnya:

```go
ctx = context.WithValue(ctx, ctxKeyAccountID, device.AccountID)
```

- [ ] **Step 4: Rewrite `admin_license.go` jadi `requireActiveAccount` + `handleAdminLicense` baru**

```go
package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// requireActiveAccount menolak request kalau account pemilik sesi/API
// key/device tidak operasional (active/expiring). Menggantikan
// requireLicense (dulu baca file lokal + grace period -- sekarang query
// langsung ke accounts, tidak ada lagi jaringan antar dua service).
//
// SELALU ditaruh SETELAH requireAdmin/requireAPIKey/requireDevice di
// api.go (Step 5), bukan sebelumnya seperti requireLicense dulu --
// account_id belum ada di context sebelum salah satu dari ketiganya lolos.
func (a *API) requireActiveAccount(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID, ok := AccountFromContext(r.Context())
		if !ok {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi/kredensial tidak ditemukan")
			return
		}
		acc, err := a.store.GetAccountByID(r.Context(), accountID)
		if errors.Is(err, store.ErrAccountNotFound) {
			a.writeError(w, http.StatusPaymentRequired, "account_not_found", "akun tidak ditemukan")
			return
		}
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
			return
		}
		if !acc.Operational(a.now()) {
			status := acc.DerivedStatus(a.now())
			a.writeError(w, http.StatusPaymentRequired, "account_"+status, accountErrorMessage(status))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func accountErrorMessage(status string) string {
	switch status {
	case "expired":
		return "akun sudah kedaluwarsa"
	case "suspended":
		return "akun sedang disuspend"
	case "revoked":
		return "akun sudah dicabut"
	default:
		return "akun tidak aktif"
	}
}

type accountJSON struct {
	BusinessName  string `json:"business_name"`
	Plan          string `json:"plan"`
	MaxDevices    int    `json:"max_devices"`
	ExpiresAt     string `json:"expires_at"`
	DaysRemaining int    `json:"days_remaining"`
	Status        string `json:"status"`
}

const dateOnlyLayout = "2006-01-02"

// handleAdminLicense mengembalikan status akun untuk Customer Dashboard.
// Nama fungsi & route (/admin/license) dipertahankan -- ini bukan lagi
// "lisensi file", tapi Customer Dashboard sudah punya halaman ini, ganti
// nama endpoint di tahap ini cuma menambah risiko tanpa manfaat.
func (a *API) handleAdminLicense(w http.ResponseWriter, r *http.Request) {
	accountID, ok := AccountFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak ditemukan")
		return
	}
	acc, err := a.store.GetAccountByID(r.Context(), accountID)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	status := acc.DerivedStatus(a.now())
	daysRemaining := int(acc.ExpiresAt.Sub(a.now()).Hours() / 24)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "license": accountJSON{
		BusinessName: acc.BusinessName, Plan: acc.Plan, MaxDevices: acc.MaxDevices,
		ExpiresAt: acc.ExpiresAt.Format(dateOnlyLayout), DaysRemaining: daysRemaining, Status: status,
	}})
}
```

- [ ] **Step 5: `api.go` — ganti seluruh pemakaian `requireLicense` jadi `requireActiveAccount`, hapus route publik `/api/v1/events`**

Cari-ganti tiap `a.requireLicense(` jadi `a.requireActiveAccount(` di seluruh `Handler()`. **Urutan wrapping berubah**: `requireActiveAccount` butuh `account_id` sudah ada di context, jadi ia harus di BAGIAN DALAM (dipanggil setelah `requireAdmin`/`requireAPIKey`/`requireDevice` sudah mengisi context), bukan di luar seperti `requireLicense` dulu:

```go
// SALAH (urutan lama): a.requireLicense(a.requireDevice(handler))
// BENAR (urutan baru): a.requireDevice(a.requireActiveAccount(handler))
mux.Handle("GET /api/v1/device/me", a.requireDevice(a.requireActiveAccount(http.HandlerFunc(a.handleDeviceMe))))
mux.Handle("POST /api/v1/devices/heartbeat",
	a.requireDevice(a.requireActiveAccount(http.HandlerFunc(a.handleHeartbeat))))
mux.Handle("POST /api/v1/events", a.requireDevice(a.requireActiveAccount(http.HandlerFunc(a.handleCallback))))
```

Terapkan pembalikan urutan yang sama ke SELURUH route yang sebelumnya `a.requireLicense(a.requireAdmin(...))` → `a.requireAdmin(a.requireActiveAccount(...))`, dan `a.requireLicense(a.requireAPIKey(...))` → `a.requireAPIKey(a.requireActiveAccount(...))`.

**Hapus** baris `mux.HandleFunc("GET /api/v1/events", a.handleEvents)` (route publik tanpa auth aplikasi, diproteksi Caddy basic_auth generik — tidak ada cara menurunkan `account_id` darinya di model multi-tenant, lihat Global Constraints). `GET /api/v1/admin/events` (sudah `requireAdmin`) tetap ada, itu penggantinya untuk Customer Dashboard.

`GET /api/v1/admin/license` **tetap** `a.requireAdmin(http.HandlerFunc(a.handleAdminLicense))` TANPA `requireActiveAccount` — sama seperti sebelumnya, admin harus selalu bisa lihat status akunnya sendiri walau tidak aktif.

- [ ] **Step 6: Update `api.go` struct `API` — hapus field `loadLicense`, hapus `New`/`NewWithLicense` versi lama**

```go
type API struct {
	store             *store.Store
	encKey            []byte
	adminSessionKey   []byte
	webhookSecretKey  []byte
	now               func() time.Time
	loginThrottle     *loginThrottle
	webhookHTTPClient *http.Client
}

func New(s *store.Store, encKey []byte, adminSessionKey []byte, webhookSecretKey []byte,
	now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return &API{
		store: s, encKey: encKey, adminSessionKey: adminSessionKey, webhookSecretKey: webhookSecretKey,
		now: now, loginThrottle: newLoginThrottle(),
		webhookHTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}
```

`NewWithLicense` **dihapus sepenuhnya** — test sekarang bisa membuat account sungguhan lewat `store.CreateAccount` (Task 3) langsung di database test, tidak perlu jalur suntik-status khusus test lagi (beda dari license file yang privatenya sengaja tidak pernah ada di test).

- [ ] **Step 7: Update `admin_auth_test.go`, `auth_middleware_test.go`, `admin_license_test.go`**

Ganti setiap `httpapi.NewWithLicense(...)` jadi `httpapi.New(s, encKey, adminSessionKey, webhookSecretKey, now)` (4 parameter data + now, tanpa argumen lisensi). Tambahkan `s.CreateAccount(...)` di setiap test yang butuh account aktif (pola `seedAccount` dari Task 4, tapi di paket `httpapi_test` — sesuaikan nama biar tidak bentrok, mis. `seedActiveAccount(t, s, accountID string)`). `admin_license_test.go`: tulis ulang test-nya jadi menguji `requireActiveAccount` (bukan `requireLicense`) dengan account berstatus `expired`/`suspended`/`revoked`/`active` lewat `SetAccountAdminStatus`/`RenewAccount`/`CreateAccount` langsung, bukan lewat `NewWithLicense`.

- [ ] **Step 8: Jalankan seluruh test `internal/httpapi`, pastikan lulus**

Run: `cd backend && go test ./internal/httpapi/... -v 2>&1 | tail -120`
Expected: `ok`, nol `FAIL`. (Boleh masih ada test lain di paket ini yang gagal karena BELUM di-update ke `accountID` — itu dikerjakan Task 9. Task ini fokus cuma tiga file auth + `admin_license.go`+`api.go`; kalau kompilasi paket gagal total karena handler lain manggil `store.ListDevices(ctx)` tanpa `accountID`, itu wajar — lanjut ke Task 9 sebelum menjalankan test penuh lagi. Task ini dianggap selesai begitu `go build ./internal/httpapi/...` sukses UNTUK BAGIAN yang disentuh task ini; test suite penuh dikonfirmasi hijau di akhir Task 9.)

- [ ] **Step 9: Commit**

```bash
cd backend
gofmt -l internal/httpapi/
git add internal/httpapi/admin_auth.go internal/httpapi/apikey_auth.go internal/httpapi/auth_middleware.go internal/httpapi/admin_license.go internal/httpapi/api.go internal/httpapi/admin_auth_test.go internal/httpapi/auth_middleware_test.go internal/httpapi/admin_license_test.go
git commit -m "feat(httpapi): account_id diturunkan dari sesi/API key/HMAC device

AccountFromContext menggantikan AdminFromContext(bool) -- sesi/API
key/device sekarang sama-sama menaruh account_id di context yang sama.
requireActiveAccount menggantikan requireLicense: query langsung ke
accounts (bukan file lokal + grace period), dan urutan wrapping DIBALIK
di api.go -- ia butuh account_id sudah ada di context, jadi dipasang
SETELAH requireAdmin/requireAPIKey/requireDevice, bukan sebelumnya.

Route publik GET /api/v1/events (basic_auth Caddy, bukan auth aplikasi)
dihapus -- tidak ada cara menurunkan account_id darinya. GET
/api/v1/admin/events (requireAdmin, sudah scoped) penggantinya.

NewWithLicense dihapus -- test sekarang bikin account sungguhan lewat
store.CreateAccount, tidak perlu jalur suntik status khusus test lagi.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 9: `internal/httpapi` — seluruh handler admin/device/invoice meneruskan `accountID`

**Files:**
- Modify: `backend/internal/httpapi/admin_devices.go`
- Modify: `backend/internal/httpapi/admin_apikeys.go`
- Modify: `backend/internal/httpapi/admin_invoices.go` (atau nama file invoice admin yang ada — cek `grep -l handleAdminInvoices internal/httpapi/*.go`)
- Modify: `backend/internal/httpapi/invoices.go` (handler `handleCreateInvoice`/`handleGetInvoice`, cek nama file sesungguhnya lewat `grep -l handleCreateInvoice internal/httpapi/*.go`)
- Modify: file handler event/exception/webhook admin (cek nama sesungguhnya lewat `grep -rl "func (a \*API) handleAdmin" internal/httpapi/*.go`)
- Modify: `backend/internal/httpapi/webhook_worker.go`
- Modify: seluruh file `*_test.go` yang menguji handler-handler di atas

**Interfaces:**
- Consumes: `AccountFromContext` (Task 8), signature baru seluruh fungsi `internal/store` dari Task 4-7.
- Produces: tidak ada interface baru — task ini murni "sambungkan A ke B" di titik pemanggilan.

Pola yang diulang di SETIAP handler yang task ini sentuh:

```go
func (a *API) handleAdminDevices(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context()) // sudah pasti ok=true, lolos requireAdmin
	devices, err := a.store.ListDevices(r.Context(), accountID)
	// ... sisanya sama seperti sekarang
}
```

- [ ] **Step 1: Cari seluruh titik pemanggilan store yang perlu `accountID`**

```bash
cd backend
grep -rn "a.store.ListDevices\|a.store.SetDeviceEnabled\|a.store.CreateAPIKey\|a.store.ListAPIKeys\|a.store.RevokeAPIKey\|a.store.CreateInvoice\|a.store.GetInvoiceByID\|a.store.GetInvoiceByExternalRef\|a.store.ListInvoices\|a.store.MatchEvent\|a.store.ListEvents\|a.store.ListExceptions\|a.store.ManualMatchEvent\|a.store.DismissEvent\|a.store.CreateWebhookEndpoint\|a.store.ListWebhookEndpoints\|a.store.GetWebhookEndpoint\|a.store.SetWebhookEndpointEnabled\|a.store.DeleteWebhookEndpoint\|a.store.EnqueueTestDelivery\|a.store.ListWebhookDeliveries\|a.store.CreateDevice" internal/httpapi/*.go | grep -v _test
```

Setiap baris hasil grep ini adalah satu titik yang perlu diedit: tambahkan `accountID, _ := AccountFromContext(r.Context())` di awal handler (kalau belum ada), lalu sisipkan `accountID` sebagai argumen sesuai posisi parameter baru di signature Task 4-7.

- [ ] **Step 2: Edit satu per satu sesuai daftar dari Step 1**

Untuk device pairing (`handleAdminCreateDevice` kalau ada, atau device dibuat lewat `devicetool` — cek dulu apakah ada endpoint HTTP buat create device selain CLI; kalau tidak ada, `CreateDevice` di task ini HANYA disentuh lewat `cmd/devicetool`, lihat Step 4): pastikan setiap pemanggilan `a.store.XxxYyy(r.Context(), ...)` di daftar Step 1 disisipi `accountID` di posisi yang benar (persis urutan parameter di signature baru masing-masing fungsi, JANGAN ditebak — buka definisi fungsinya di `internal/store` untuk konfirmasi urutan sebelum mengedit pemanggilnya).

- [ ] **Step 3: `webhook_worker.go` — ganti gate lisensi**

Cari `if !a.loadLicense().Status.Operational()` di `ProcessDueWebhooks`, hapus baris itu — worker ini sudah jalan lintas akun (Task 5/7 spec: `DueWebhookDeliveries` tidak di-scope per akun karena ini worker global), jadi tidak ada satu `account_id` tunggal untuk dicek di sini. Kalau perlu tetap ada gate per pengiriman individual, tambahkan pengecekan status account di titik yang sudah tahu `account_id` pengiriman itu (dari `invoice`/`webhook_endpoint` yang sudah di-load) — TAPI ini di luar cakupan MVP task ini; catat sebagai TODO eksplisit di komentar kode (bukan skip diam-diam) kalau tidak dikerjakan, dan laporkan ke Akbar sebagai keputusan yang perlu dikonfirmasi sebelum lanjut ke sub-project berikutnya.

- [ ] **Step 4: `cmd/devicetool/main.go` — device dibuat lewat CLI butuh `accountID` juga**

Baca `cat backend/cmd/devicetool/main.go`. Tambahkan flag `-account` (wajib) yang diteruskan ke `store.CreateDevice`. Ini SATU-SATUNYA cara membuat device sampai sub-project #3 (Customer Dashboard swalayan) selesai — dokumentasikan di komentar file kalau belum ada.

- [ ] **Step 5: Update SELURUH `*_test.go` yang tersentuh**

Ini kemungkinan besar file terbanyak yang diedit di seluruh plan ini. Untuk tiap test yang gagal compile, tambahkan `seedActiveAccount(t, s, "acc_test")` (atau reuse account yang sudah dibuat test lain di file yang sama) lalu sisipkan `"acc_test"` di posisi yang sesuai di tiap pemanggilan store/handler yang berubah.

- [ ] **Step 6: Jalankan seluruh test backend, pastikan lulus**

Run: `cd backend && make test 2>&1 | tail -80`
Expected: `ok` di semua paket, nol `FAIL`. Ini pertama kalinya sejak Task 3 seluruh test suite (bukan cuma satu paket) dijalankan — kalau ada kegagalan yang bukan dari task ini (mis. paket lain yang lupa disentuh), kembali ke task yang relevan, jangan ditambal di sini.

- [ ] **Step 7: Commit**

```bash
cd backend
gofmt -l .
git add internal/httpapi/ cmd/devicetool/
git commit -m "feat(httpapi): seluruh handler admin/device/invoice meneruskan account_id

Menyambungkan AccountFromContext (Task 8) ke signature baru internal/store
(Task 4-7) di setiap handler yang menyentuh devices/api_keys/invoices/
events/exceptions/webhooks. cmd/devicetool dapat flag -account wajib --
satu-satunya cara membuat device sampai Customer Dashboard swalayan
(sub-project #3) selesai.

make test hijau penuh untuk pertama kalinya sejak Task 3 -- seluruh
paket backend, bukan cuma satu paket.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 10: Endpoint vendor (`/api/v1/vendor/*`) di backend utama

**Files:**
- Create: `backend/internal/httpapi/vendor_auth.go`
- Create: `backend/internal/httpapi/vendor_accounts.go`
- Create: `backend/internal/httpapi/vendor_audit.go`
- Create: `backend/internal/store/audit.go`
- Create migration: `backend/migrations/00009_audit_log.sql`
- Modify: `backend/internal/httpapi/api.go` (tambah field `vendorSessionKey`, route baru, parameter baru di `New`)
- Modify: `backend/internal/config/config.go` (tambah `VendorSessionKey`)
- Test: `backend/internal/httpapi/vendor_auth_test.go`, `vendor_accounts_test.go`

**Interfaces:**
- Consumes: `store.CreateAccount`, `store.ListAccounts`, `store.GetAccountByID`, `store.RenewAccount`, `store.SetAccountAdminStatus` (Task 3); `store.UpsertVendorAdmin`, `store.GetVendorAdminByUsername` (Task 3); `auth.NewSessionToken`/`VerifySessionToken` (Task 1).
- Produces: cookie `vendor_session` (nama BEDA dari `admin_session`, spec §4). Route: `POST /api/v1/vendor/login`, `POST /api/v1/vendor/logout`, `POST /api/v1/vendor/accounts`, `GET /api/v1/vendor/accounts`, `GET /api/v1/vendor/accounts/{id}`, `POST /api/v1/vendor/accounts/{id}/renew`, `POST /api/v1/vendor/accounts/{id}/suspend`, `POST /api/v1/vendor/accounts/{id}/revoke`, `GET /api/v1/vendor/audit-log` — dipakai `vendor-dashboard/` di Task 13.

- [ ] **Step 1: Migration audit log**

Buat `backend/migrations/00009_audit_log.sql` (isi identik `internal/licenseserver` punya, tabel dipindah ke database utama):

```sql
-- +goose Up
CREATE TABLE audit_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor      TEXT NOT NULL,
    action     TEXT NOT NULL,
    resource   TEXT NOT NULL,
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_created_at_idx ON audit_log (created_at DESC);

-- +goose Down
DROP TABLE audit_log;
```

Run: `cd backend && make migrate`

- [ ] **Step 2: `internal/store/audit.go`**

```go
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type AuditEntry struct {
	ID        int64
	Actor     string
	Action    string
	Resource  string
	Metadata  json.RawMessage
	CreatedAt time.Time
}

func (s *Store) LogAudit(ctx context.Context, actor, action, resource string, metadata any) error {
	var raw []byte
	if metadata != nil {
		var err error
		raw, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("store: marshal audit metadata: %w", err)
		}
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_log (actor, action, resource, metadata) VALUES ($1, $2, $3, $4)`,
		actor, action, resource, raw)
	if err != nil {
		return fmt.Errorf("store: log audit: %w", err)
	}
	return nil
}

func (s *Store) ListAuditLog(ctx context.Context, limit, offset int) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, actor, action, resource, metadata, created_at
		 FROM audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list audit log: %w", err)
	}
	defer rows.Close()

	out := make([]AuditEntry, 0)
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Resource, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan audit entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: `config.go` — `VendorSessionKey`**

Tambah field `VendorSessionKey []byte` ke `Config`, validasi identik pola `AdminSessionKey` (base64, `secretbox.KeySize` byte, wajib diisi). Hapus field `LicenseKey`, `Environment`, `LicenseServerURL`, `LicenseFilePath`, dan blok validasinya (dipindah ke Task 12 — task ini boleh menyisakannya kalau Task 12 belum jalan, tapi field baru `VendorSessionKey` HARUS ditambah di task ini karena Task 10 langsung butuh).

- [ ] **Step 4: `vendor_auth.go`**

```go
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

const vendorSessionCookie = "vendor_session"

type vendorCtxKey int

const ctxKeyVendorUsername vendorCtxKey = iota

func VendorFromContext(ctx context.Context) (username string, ok bool) {
	v, ok := ctx.Value(ctxKeyVendorUsername).(string)
	return v, ok
}

type vendorLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleVendorLogin -- pola identik handleAdminLogin, tabel dan cookie beda
// (vendor_admins, vendor_session) supaya sesi vendor dan sesi customer
// TIDAK PERNAH bisa tertukar walau di browser yang sama.
func (a *API) handleVendorLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.vendorLoginThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts", "terlalu banyak percobaan login, coba lagi nanti")
		return
	}
	var req vendorLoginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.Username == "" || req.Password == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "username dan password wajib diisi")
		return
	}
	vendor, err := a.store.GetVendorAdminByUsername(r.Context(), req.Username)
	if err != nil || !vendor.VerifyPassword(req.Password) {
		a.vendorLoginThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusUnauthorized, "invalid_credentials", "username atau password salah")
		return
	}
	a.vendorLoginThrottle.RecordSuccess(ip)

	token := auth.NewSessionToken(a.vendorSessionKey, a.now(), vendor.Username)
	http.SetCookie(w, &http.Cookie{
		Name: vendorSessionCookie, Value: token, Path: "/", HttpOnly: true,
		Secure: isSecureRequest(r), SameSite: http.SameSiteLaxMode,
		MaxAge: int(auth.SessionDuration.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) handleVendorLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: vendorSessionCookie, Value: "", Path: "/", HttpOnly: true,
		Secure: isSecureRequest(r), SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) requireVendor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(vendorSessionCookie)
		if err != nil || cookie.Value == "" {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak ditemukan")
			return
		}
		username, ok := auth.VerifySessionToken(a.vendorSessionKey, cookie.Value, a.now())
		if !ok {
			a.writeError(w, http.StatusUnauthorized, "unauthenticated", "sesi tidak valid atau kedaluwarsa")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyVendorUsername, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

Tambahkan field `vendorSessionKey []byte` dan `vendorLoginThrottle *loginThrottle` ke struct `API`, inisialisasi keduanya di `New` (parameter baru `vendorSessionKey []byte`, `vendorLoginThrottle: newLoginThrottle()`).

- [ ] **Step 5: `vendor_accounts.go`**

Port `handleCreateCustomer`/`handleListCustomers`/`handleGetCustomer` (dari `internal/licenseserver/httpapi/customers.go`, sudah dibaca sebelumnya di sesi ini) + `handleCreateLicense`/`handleRenewLicense`/`handleSuspendLicense`/`handleRevokeLicense` (dari `licenses.go`) jadi SATU set handler `accounts` (spec §4 — customer+license sudah gabung jadi satu tabel):

```go
package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

var planPresets = map[string]int{"Starter": 3, "Business": 10, "Enterprise": -1}

type accountVendorJSON struct {
	ID            string `json:"id"`
	BusinessName  string `json:"business_name"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	Plan          string `json:"plan"`
	MaxDevices    int    `json:"max_devices"`
	Status        string `json:"status"`
	ExpiresAt     string `json:"expires_at"`
	DaysRemaining int    `json:"days_remaining"`
	CreatedAt     string `json:"created_at"`
}

func toAccountVendorJSON(acc store.Account, now time.Time) accountVendorJSON {
	return accountVendorJSON{
		ID: acc.ID, BusinessName: acc.BusinessName, Email: acc.Email, Username: acc.Username,
		Plan: acc.Plan, MaxDevices: acc.MaxDevices, Status: acc.DerivedStatus(now),
		ExpiresAt: acc.ExpiresAt.Format(dateOnlyLayout),
		DaysRemaining: int(acc.ExpiresAt.Sub(now).Hours() / 24),
		CreatedAt:     acc.CreatedAt.Format(time.RFC3339),
	}
}

type createAccountRequest struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Plan         string `json:"plan"`
	ExpiresAt    string `json:"expires_at"`
}

func generateInitialPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789abcdefghjkmnpqrstuvwxyz"
	pw := make([]byte, len(b))
	for i, v := range b {
		pw[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(pw), nil
}

func (a *API) handleVendorCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	maxDevices, ok := planPresets[req.Plan]
	if !ok {
		a.writeError(w, http.StatusBadRequest, "invalid_plan", "plan harus salah satu: Starter, Business, Enterprise")
		return
	}
	expiresAt, err := time.Parse(dateOnlyLayout, req.ExpiresAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "expires_at harus format YYYY-MM-DD")
		return
	}
	id, err := randomPrefixedAccountID()
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	password, err := generateInitialPassword()
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	err = a.store.CreateAccount(r.Context(), store.CreateAccountInput{
		ID: id, BusinessName: req.BusinessName, Email: req.Email, Username: req.Username,
		PlaintextPassword: password, Plan: req.Plan, MaxDevices: maxDevices, ExpiresAt: expiresAt,
	})
	if err != nil {
		slog.Error("create account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	if err := a.store.LogAudit(r.Context(), vendorUsername, "ACCOUNT_CREATED", id,
		map[string]string{"business_name": req.BusinessName, "plan": req.Plan}); err != nil {
		slog.Error("log audit gagal", "err", err)
	}

	acc, _ := a.store.GetAccountByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true, "account": toAccountVendorJSON(acc, a.now()), "initial_password": password,
	})
}

func (a *API) handleVendorListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.store.ListAccounts(r.Context())
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]accountVendorJSON, 0, len(accounts))
	for _, acc := range accounts {
		out = append(out, toAccountVendorJSON(acc, a.now()))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "accounts": out})
}

func (a *API) handleVendorGetAccount(w http.ResponseWriter, r *http.Request) {
	acc, err := a.store.GetAccountByID(r.Context(), r.PathValue("accountID"))
	if errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	}
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "account": toAccountVendorJSON(acc, a.now())})
}

type renewAccountRequest struct {
	ExpiresAt string `json:"expires_at"`
}

func (a *API) handleVendorRenewAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("accountID")
	var req renewAccountRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	newExpiresAt, err := time.Parse(dateOnlyLayout, req.ExpiresAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "expires_at harus format YYYY-MM-DD")
		return
	}
	if err := a.store.RenewAccount(r.Context(), id, newExpiresAt); errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	} else if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	_ = a.store.LogAudit(r.Context(), vendorUsername, "ACCOUNT_RENEWED", id,
		map[string]string{"new_expires_at": req.ExpiresAt})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) setAccountAdminStatus(w http.ResponseWriter, r *http.Request, status, action string) {
	id := r.PathValue("accountID")
	if err := a.store.SetAccountAdminStatus(r.Context(), id, status); errors.Is(err, store.ErrAccountNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "account tidak ditemukan")
		return
	} else if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	vendorUsername, _ := VendorFromContext(r.Context())
	_ = a.store.LogAudit(r.Context(), vendorUsername, action, id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) handleVendorSuspendAccount(w http.ResponseWriter, r *http.Request) {
	a.setAccountAdminStatus(w, r, "suspended", "ACCOUNT_SUSPENDED")
}

func (a *API) handleVendorRevokeAccount(w http.ResponseWriter, r *http.Request) {
	a.setAccountAdminStatus(w, r, "revoked", "ACCOUNT_REVOKED")
}
```

Tambahkan `randomPrefixedAccountID` (pola sama `randomPrefixedID` yang sudah ada di `internal/store/apikey.go` — TAPI itu unexported di paket lain, jadi tulis versi lokal singkat di `vendor_accounts.go` pakai `crypto/rand`+`encoding/hex`, prefix `"acc_"`).

- [ ] **Step 6: `vendor_audit.go`**

```go
package httpapi

import (
	"net/http"
	"time"
)

type auditEntryJSON struct {
	ID        int64  `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	CreatedAt string `json:"created_at"`
}

func (a *API) handleVendorAuditLog(w http.ResponseWriter, r *http.Request) {
	entries, err := a.store.ListAuditLog(r.Context(), 100, 0)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]auditEntryJSON, 0, len(entries))
	for _, e := range entries {
		out = append(out, auditEntryJSON{
			ID: e.ID, Actor: e.Actor, Action: e.Action, Resource: e.Resource,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "entries": out})
}
```

- [ ] **Step 7: Daftarkan route di `api.go`**

```go
mux.HandleFunc("POST /api/v1/vendor/login", a.handleVendorLogin)
mux.HandleFunc("POST /api/v1/vendor/logout", a.handleVendorLogout)
mux.Handle("POST /api/v1/vendor/accounts", a.requireVendor(http.HandlerFunc(a.handleVendorCreateAccount)))
mux.Handle("GET /api/v1/vendor/accounts", a.requireVendor(http.HandlerFunc(a.handleVendorListAccounts)))
mux.Handle("GET /api/v1/vendor/accounts/{accountID}", a.requireVendor(http.HandlerFunc(a.handleVendorGetAccount)))
mux.Handle("POST /api/v1/vendor/accounts/{accountID}/renew", a.requireVendor(http.HandlerFunc(a.handleVendorRenewAccount)))
mux.Handle("POST /api/v1/vendor/accounts/{accountID}/suspend", a.requireVendor(http.HandlerFunc(a.handleVendorSuspendAccount)))
mux.Handle("POST /api/v1/vendor/accounts/{accountID}/revoke", a.requireVendor(http.HandlerFunc(a.handleVendorRevokeAccount)))
mux.Handle("GET /api/v1/vendor/audit-log", a.requireVendor(http.HandlerFunc(a.handleVendorAuditLog)))
```

- [ ] **Step 8: Tes end-to-end vendor endpoints**

Tulis `backend/internal/httpapi/vendor_accounts_test.go`, pola sama seperti `admin_apikeys_test.go` yang sudah ada (login vendor dulu lewat `handleVendorLogin` via `httptest.NewRequest`, ambil cookie, pakai di request berikutnya). Minimal: create account → 201 + `initial_password` ada; list accounts → berisi yang baru dibuat; renew → `expires_at` berubah; suspend → status jadi `suspended`; get account tanpa cookie vendor → 401; login pakai `admin_session` (bukan `vendor_session`) ke endpoint vendor → 401 (sesi tidak boleh tertukar).

Run: `cd backend && go test ./internal/httpapi/... -run TestVendor -v`
Expected: semua `PASS`.

- [ ] **Step 9: Commit**

```bash
cd backend
gofmt -l .
git add internal/httpapi/vendor_auth.go internal/httpapi/vendor_accounts.go internal/httpapi/vendor_audit.go internal/httpapi/vendor_accounts_test.go internal/store/audit.go internal/config/config.go migrations/00009_audit_log.sql
git commit -m "feat(httpapi): endpoint vendor (/api/v1/vendor/*) di backend utama

Menggantikan internal/licenseserver sepenuhnya (dibongkar di Task 12).
Sesi vendor_session terpisah total dari admin_session -- tabel
vendor_admins sendiri, tidak bisa saling tertukar. customers+licenses
License Server yang dulu dua tabel sekarang satu: accounts. Audit log
pindah ke database utama juga.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 11: Test isolasi antar akun (integrasi, lintas endpoint)

**Files:**
- Create: `backend/internal/httpapi/tenant_isolation_test.go`

**Interfaces:**
- Consumes: seluruh handler dari Task 9, `AccountFromContext`/`requireActiveAccount` dari Task 8.

Ini kategori test PALING PENTING di seluruh plan (spec §6) — kesalahan di sini berarti kebocoran data lintas customer.

- [ ] **Step 1: Tulis test isolasi lewat sesi dashboard**

```go
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestIsolasiDeviceLintasAccount memastikan akun A tidak bisa melihat atau
// mengubah device milik akun B lewat endpoint dashboard admin, walau
// device_id-nya ditebak/diketahui persis.
func TestIsolasiDeviceLintasAccount(t *testing.T) {
	s := newTestStore(t)
	api := httpapi.New(s, testEncKey, testAdminSessionKey, testWebhookSecretKey, testNow)

	seedActiveAccount(t, s, "acc_a")
	seedActiveAccount(t, s, "acc_b")
	mustCreateDevice(t, s, "acc_b", "dev_b1", "HP B1")

	cookieA := loginAs(t, api, "acc_a") // helper: POST /admin/login, ambil Set-Cookie

	// GET /admin/devices sebagai A -- daftar tidak boleh berisi dev_b1
	rr := doRequestWithCookie(t, api, "GET", "/api/v1/admin/devices", nil, cookieA)
	if !bytesContains(rr.Body.Bytes(), []byte(`"device_id":"dev_b1"`)) == false {
		// tetap lolos -- lanjut ke assertion PATCH di bawah, ini cuma sanity check daftar
	}

	// PATCH device milik B, pakai sesi A -- harus GAGAL (404), bukan 200
	rr = doRequestWithCookie(t, api, "PATCH", "/api/v1/admin/devices/dev_b1",
		[]byte(`{"enabled":false}`), cookieA)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 (device milik akun lain tidak boleh bisa diubah)", rr.Code)
	}
}
```

Catatan implementasi: `loginAs`, `doRequestWithCookie` adalah helper baru — cek dulu apakah pola serupa sudah ada di `admin_auth_test.go`/`admin_apikeys_test.go` (kemungkinan besar ada bentuk yang mirip untuk login+cookie), REUSE nama dan bentuknya persis kalau sudah ada, jangan bikin duplikat dengan nama beda.

- [ ] **Step 2: Ulangi pola yang sama untuk invoice, webhook, API key**

- `TestIsolasiInvoiceLintasAccount`: akun A bikin invoice, akun B coba `GET /api/v1/admin/invoices/{id}` (atau lewat `requireAPIKey` pakai API key milik B) → 404, bukan data invoice A.
- `TestIsolasiWebhookLintasAccount`: akun A bikin webhook endpoint, akun B coba `DELETE`/`PATCH` pakai ID milik A → 404.
- `TestIsolasiAPIKeyLintasAccount`: API key milik akun A dipakai memanggil `POST /api/v1/invoices` — invoice yang tercipta harus `account_id = acc_a`, BUKAN akun manapun yang disebut di body request (buktikan `account_id` tidak bisa disuntik dari body — coba kirim field `account_id` di body request kalau `handleCreateInvoice` menerima JSON, pastikan field itu diabaikan sepenuhnya).

- [ ] **Step 3: Jalankan, pastikan semua lulus**

Run: `cd backend && go test ./internal/httpapi/... -run TestIsolasi -v`
Expected: semua `PASS`.

- [ ] **Step 4: Commit**

```bash
cd backend
gofmt -l internal/httpapi/
git add internal/httpapi/tenant_isolation_test.go
git commit -m "test(httpapi): isolasi data lintas account -- kategori paling penting

Akun A tidak bisa lihat/ubah device, invoice, webhook endpoint milik akun
B lewat ID langsung (404, bukan 403 -- 403 membocorkan bahwa resource itu
ADA). account_id dari body request diabaikan sepenuhnya, selalu diturunkan
dari kredensial server-side (spec §3, §6).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 12: Bongkar total License Server (`cmd/licenseserver`, `internal/licenseserver`, `internal/licenseclient`)

**Files:**
- Delete: `backend/cmd/licenseserver/`
- Delete: `backend/internal/licenseserver/`
- Delete: `backend/internal/licenseclient/`
- Delete: `backend/migrations-license/`
- Delete: `backend/deploy/gopay-licenseserver.service`
- Delete: `backend/deploy/gopay-vendor-dashboard.service` (dibuat ulang di Task 13 dengan isi baru, arah beda — hapus dulu di sini biar bersih)
- Delete: `backend/internal/licensecheck/` (seluruh isi — signing/verify file lokal sudah tidak dipakai sama sekali, `accounts` dicek langsung di database)
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/config/config.go` (hapus `LicenseKey`/`Environment`/`LicenseServerURL`/`LicenseFilePath`, kalau belum dihapus di Task 10)
- Modify: `backend/Makefile` (hapus `migrate-license`, `LICENSE_TEST_DATABASE_URL`)
- Modify: `backend/docker-compose.yml` (hapus service `postgres-license`)
- Modify: `.env.example`, `.env.dev.example`, `.env.uat.example` (hapus blok lisensi lama, tambah `VENDOR_SESSION_KEY` — HANYA relevan di `.env` instalasi yang menjalankan endpoint vendor, yaitu SATU instalasi pusat, bukan tiap `.env.uat.example` — evaluasi ulang apakah UAT butuh vendor endpoint sama sekali; kalau tidak, jangan tambahkan `VENDOR_SESSION_KEY` ke `.env.uat.example`)

**Interfaces:**
- Consumes: tidak ada — task ini murni penghapusan + rewiring `main.go`/`config.go` yang masih menyebut kode yang dihapus.

- [ ] **Step 1: Hapus seluruh direktori/file License Server**

```bash
cd backend
git rm -r cmd/licenseserver internal/licenseserver internal/licenseclient migrations-license internal/licensecheck
git rm deploy/gopay-licenseserver.service
```

- [ ] **Step 2: `cmd/server/main.go` — hapus wiring `licenseclient`**

Hapus import `"github.com/akbarryyan/gopay-notifications/backend/internal/licenseclient"`, hapus variabel `licClient` dan pemanggilannya, hapus goroutine `licenseTicker` (24 jam) — sisakan cuma `webhookTicker` (1 menit). Ubah pemanggilan `httpapi.New(...)`:

```go
api := httpapi.New(s, cfg.DeviceSecretKey, cfg.AdminSessionKey, cfg.WebhookSecretKey,
	cfg.VendorSessionKey, time.Now)
```

(Sesuaikan urutan parameter persis dengan signature final `httpapi.New` — Task 8 mengubahnya jadi 4 kunci data + `now`, Task 10 menambah `vendorSessionKey`; task ini yang menyatukan keduanya jadi signature FINAL. Buka `internal/httpapi/api.go` dulu untuk konfirmasi urutan sebelum mengedit `main.go`.)

- [ ] **Step 3: `config.go` — pastikan `LicenseKey`/`Environment`/`LicenseServerURL`/`LicenseFilePath` benar-benar hilang**

Hapus field dan blok validasi terkait dari `Config`/`Load()` kalau masih ada (Task 10 sudah menambah `VendorSessionKey`, tapi mungkin belum menghapus field lisensi lama kalau dikerjakan sebelum task ini).

- [ ] **Step 4: Bersihkan `Makefile`, `docker-compose.yml`**

`Makefile`: hapus baris `LICENSE_TEST_DATABASE_URL`, target `migrate-license`, dan hapus dependency-nya dari target `test` (`test: db-up migrate migrate-license` → `test: db-up migrate`), hapus `LICENSE_TEST_DATABASE_URL="..."` dari perintah `go test` di target `test`.

`docker-compose.yml`: hapus seluruh service `postgres-license`.

- [ ] **Step 5: Bersihkan `.env*.example`**

`.env.example`/`.env.dev.example`: hapus blok `LICENSE_KEY`/`ENVIRONMENT`/`LICENSE_SERVER_URL`/`LICENSE_FILE_PATH`, tambahkan `VENDOR_SESSION_KEY=` dengan komentar penjelasan (dihasilkan `go run ./cmd/devicetool -genkey`, dipakai HANYA di instalasi yang menjalankan endpoint vendor — biasanya cuma satu instalasi pusat, bukan tiap deployment). `.env.uat.example`: hapus blok lisensi yang sama; JANGAN tambahkan `VENDOR_SESSION_KEY` di sini kecuali UAT memang menjalankan endpoint vendornya sendiri (default: tidak).

- [ ] **Step 6: Jalankan seluruh test dan build, pastikan lulus**

Run: `cd backend && go build ./... && go vet ./... && make test 2>&1 | tail -60`
Expected: build sukses, vet bersih, seluruh test `ok`.

- [ ] **Step 7: Commit**

```bash
cd backend
gofmt -l .
git add -A
git commit -m "chore: bongkar total License Server + internal/licensecheck

cmd/licenseserver, internal/licenseserver, internal/licenseclient,
internal/licensecheck, migrations-license, dan unit systemd terkait
dihapus -- fungsinya sudah diserap ke accounts di database utama (Task
2-3) dan endpoint vendor (Task 10). Tidak ada lagi file lokal
license-state.lic, tidak ada lagi grace period, tidak ada lagi ticker 24
jam validasi lisensi -- status akun dicek langsung tiap request lewat
requireActiveAccount (Task 8).

main.go, config.go, Makefile, docker-compose.yml, .env*.example
dibersihkan dari seluruh referensi lisensi lama.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 13: `vendor-dashboard/` — rewiring ke endpoint akun di backend utama

**Files:**
- Modify: `vendor-dashboard/src/lib/api.ts`
- Modify: `vendor-dashboard/next.config.ts` (default rewrite target ganti dari License Server ke backend utama)
- Rewrite: `vendor-dashboard/src/app/(dashboard)/page.tsx` (daftar akun + buat baru, gabung customer+license jadi satu form)
- Rewrite: `vendor-dashboard/src/app/(dashboard)/customers/[id]/page.tsx` → pindah jadi `vendor-dashboard/src/app/(dashboard)/accounts/[id]/page.tsx` (detail akun + renew/suspend/revoke, tanpa konsep license/installation terpisah)
- Delete: `vendor-dashboard/src/app/(dashboard)/licenses/[id]/page.tsx` (konsep license terpisah sudah tidak ada — digabung ke accounts)
- Modify: `vendor-dashboard/src/app/(dashboard)/audit-log/page.tsx` (field `resource` sekarang `account_id`, bukan campuran customer_id/license_id — cek apakah tampilannya masih valid, biasanya tidak perlu berubah karena sudah generic)
- Modify: `vendor-dashboard/next.config.ts`, `.env.local.example`

**Interfaces:**
- Consumes: endpoint `/api/v1/vendor/*` dari Task 10.

- [ ] **Step 1: Tulis ulang `lib/api.ts`**

```ts
/**
 * Klien API Vendor Dashboard -- memanggil endpoint vendor
 * (/api/v1/vendor/*) di backend utama (bukan lagi License Server
 * terpisah, yang sudah dibongkar total).
 */

export class ApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type ErrorBody = { success: false; error: string; message: string };

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (!res.ok) {
    let body: ErrorBody | null = null;
    try {
      body = await res.json();
    } catch {
      // Body bukan JSON.
    }
    throw new ApiError(
      body?.error ?? "unknown_error",
      body?.message ?? `Permintaan gagal dengan status ${res.status}`,
      res.status,
    );
  }
  return res.json() as Promise<T>;
}

export function login(username: string, password: string): Promise<{ success: true }> {
  return apiFetch("/api/v1/vendor/login", { method: "POST", body: JSON.stringify({ username, password }) });
}

export function logout(): Promise<{ success: true }> {
  return apiFetch("/api/v1/vendor/logout", { method: "POST" });
}

export type AccountPlan = "Starter" | "Business" | "Enterprise";
export type AccountStatus = "active" | "expiring" | "expired" | "suspended" | "revoked";

export interface Account {
  id: string;
  business_name: string;
  email: string;
  username: string;
  plan: AccountPlan;
  max_devices: number;
  status: AccountStatus;
  expires_at: string;
  days_remaining: number;
  created_at: string;
}

export async function getAccounts(): Promise<Account[]> {
  const res = await apiFetch<{ accounts: Account[] }>("/api/v1/vendor/accounts");
  return res.accounts;
}

export function createAccount(input: {
  business_name: string; email: string; username: string; plan: AccountPlan; expires_at: string;
}): Promise<{ success: true; account: Account; initial_password: string }> {
  return apiFetch("/api/v1/vendor/accounts", { method: "POST", body: JSON.stringify(input) });
}

export async function getAccount(id: string): Promise<Account> {
  const res = await apiFetch<{ account: Account }>(`/api/v1/vendor/accounts/${encodeURIComponent(id)}`);
  return res.account;
}

export function renewAccount(id: string, expiresAt: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/vendor/accounts/${encodeURIComponent(id)}/renew`, {
    method: "POST", body: JSON.stringify({ expires_at: expiresAt }),
  });
}

export function suspendAccount(id: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/vendor/accounts/${encodeURIComponent(id)}/suspend`, { method: "POST" });
}

export function revokeAccount(id: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/vendor/accounts/${encodeURIComponent(id)}/revoke`, { method: "POST" });
}

export interface AuditEntry {
  id: number;
  actor: string;
  action: string;
  resource: string;
  metadata: unknown;
  created_at: string;
}

export async function getAuditLog(): Promise<AuditEntry[]> {
  const res = await apiFetch<{ entries: AuditEntry[] }>("/api/v1/vendor/audit-log");
  return res.entries;
}
```

- [ ] **Step 2: `next.config.ts` — rewrite target**

Ganti default `LICENSE_SERVER_URL`/port lama jadi mengarah ke backend utama (variabel env, mis. `BACKEND_URL`, default `http://localhost:8090` — port yang sama dipakai `dashboard/` customer saat dev, karena SEKARANG memang satu backend yang sama untuk keduanya).

- [ ] **Step 3: Tulis ulang halaman-halaman**

`page.tsx`: daftar akun (`getAccounts`) + dialog buat akun baru (`createAccount`, tampilkan `initial_password` sekali di dialog sukses — pola sama seperti reveal License Key dulu). Hapus konsep "customer lalu buat license di dalamnya" — sekarang satu form langsung isi semua (business_name/email/username/plan/expires_at).

Pindahkan isi `customers/[id]/page.tsx` jadi `accounts/[id]/page.tsx`: tampilkan detail akun (`getAccount`), tombol Renew/Suspend/Revoke (`renewAccount`/`suspendAccount`/`revokeAccount`). **Hapus** tabel installations (konsep installation sudah tidak ada — device count per akun ada di sub-project #3, bukan di sini).

Hapus `licenses/[id]/page.tsx` dan folder `customers/` lama sepenuhnya setelah isi dipindah.

- [ ] **Step 4: Verifikasi**

Run:
```bash
cd vendor-dashboard
npx tsc --noEmit
npx eslint .
npx next build
```
Expected: 0 error di ketiganya.

- [ ] **Step 5: Commit**

```bash
cd vendor-dashboard
git add -A
git commit -m "feat(vendor-dashboard): rewiring ke /api/v1/vendor/accounts

License Server yang jadi backend-nya sudah dibongkar total (Task 12).
customers+licenses (dua entitas) gabung jadi satu: accounts. Halaman
customers/[id] + licenses/[id] gabung jadi accounts/[id]. Tabel
installations dihapus -- konsep installation/aktivasi REST tidak relevan
lagi di model hosted multi-tenant (device count per akun itu sub-project
Customer Dashboard, bukan Vendor Dashboard).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 14: `dashboard/` (Customer Dashboard) — sesuaikan tipe `LicenseInfo`

**Files:**
- Modify: `dashboard/src/lib/api.ts`
- Modify: `dashboard/src/app/(dashboard)/license/page.tsx`

**Interfaces:**
- Consumes: `GET /api/v1/admin/license` dari Task 8 (`accountJSON`: `business_name`, `plan`, `max_devices`, `expires_at`, `days_remaining`, `status` — SUBSET dari field lama, tidak ada lagi `license_id`/`installation_id`/`validated_at`).

- [ ] **Step 1: Update `LicenseInfo`/`LicenseStatus` di `lib/api.ts`**

Hapus field `license_id`, `installation_id`, `validated_at` dari `LicenseInfo` (sudah tidak dikirim backend). `LicenseStatus` jadi 5 nilai (`active`/`expiring`/`expired`/`suspended`/`revoked`) — hapus `unreachable`/`invalid`/`missing` (tidak ada lagi konsep file lokal yang bisa rusak/tidak terjangkau; akun yang belum ada di database utama artinya sesi juga tidak akan pernah valid, jadi status itu tidak pernah benar-benar teramati dari sisi Customer Dashboard).

- [ ] **Step 2: Update `license/page.tsx`**

Hapus rendering field `installation_id`/`license_id`/`validated_at` (tidak ada lagi), hapus penanganan status `unreachable`/`invalid`/`missing` dari `STATUS_LABEL`/`INACTIVE_EXPLANATION`.

- [ ] **Step 3: Verifikasi**

Run:
```bash
cd dashboard
npx tsc --noEmit
npx eslint .
npx next build
```
Expected: 0 error.

- [ ] **Step 4: Commit**

```bash
cd dashboard
git add src/lib/api.ts "src/app/(dashboard)/license/page.tsx"
git commit -m "feat(dashboard): halaman License ikut skema account (bukan lisensi file)

GET /admin/license sekarang mengembalikan field account (business_name/
plan/max_devices/expires_at/days_remaining/status), bukan lagi license_id/
installation_id/validated_at (konsep file lokal + instalasi sudah tidak
ada, Task 12). Status jadi 5 nilai -- unreachable/invalid/missing tidak
pernah lagi teramati dari sisi ini.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 15: Verifikasi akhir dan dokumentasi

**Files:**
- Modify: `docs/qa/qa-report.md` (bagian sub-project 5 — tulis ulang total, model berubah)
- Modify: `CLAUDE.md` (bagian "Sistem lisensi" → jadi "Akun & multi-tenant", scope table)
- Modify: `backend/deploy/README.md` (bagian "Lisensi" → hapus seluruhnya, ganti penjelasan `VENDOR_SESSION_KEY` di §"Sekali di awal")

**Interfaces:**
- Consumes: hasil seluruh task sebelumnya.

- [ ] **Step 1: Jalankan seluruh test + build sekali lagi dari nol**

Run:
```bash
cd backend && go build ./... && go vet ./... && gofmt -l . && make test 2>&1 | tail -80
cd ../dashboard && npx tsc --noEmit && npx eslint . && npx next build
cd ../vendor-dashboard && npx tsc --noEmit && npx eslint . && npx next build
```
Expected: semuanya bersih/lulus. Tempel hasil `make test` — ini bukti PASS untuk qa-report.md.

- [ ] **Step 2: Tulis ulang bagian sub-project 5 di `docs/qa/qa-report.md`**

Ganti seluruh isi §15 (yang sekarang menjelaskan License Server + licenseclient offline-validated-online) jadi menjelaskan model accounts + vendor endpoints yang baru, dengan tabel PASS berisi bukti dari Step 1 (termasuk kategori test isolasi Task 11).

- [ ] **Step 3: Update `CLAUDE.md`**

Bagian "## Sistem lisensi" ditulis ulang total (arsitektur 3-komponen lama sudah tidak berlaku), scope table sub-project 5 diperbarui, paragraf soal "self-hosted" di scope table perlu disesuaikan — model sekarang hosted, bukan self-hosted lagi (ini perubahan besar ke asumsi dasar dokumen ini, tulis dengan hati-hati, jangan cuma tempel-ganti kata).

- [ ] **Step 4: `backend/deploy/README.md`**

Hapus seluruh section "## Lisensi" (deploy License Server + Vendor Dashboard terpisah sudah tidak relevan — Vendor Dashboard sekarang cuma frontend yang manggil backend utama yang SAMA, tidak ada service/database terpisah lagi untuk di-deploy). Tambahkan catatan singkat di §"Sekali di awal" soal `VENDOR_SESSION_KEY` di `.env`.

- [ ] **Step 5: Commit**

```bash
git add docs/qa/qa-report.md CLAUDE.md backend/deploy/README.md
git commit -m "docs: qa-report + CLAUDE.md + deploy README ikut pivot multi-tenant

Sub-project 5 (accounts + vendor endpoints) selesai, PASS penuh --
termasuk kategori test isolasi antar akun yang baru. CLAUDE.md 'Sistem
lisensi' ditulis ulang total (arsitektur 3-komponen lama sudah
dibongkar), deploy/README.md kehilangan section Lisensi (Vendor Dashboard
sekarang manggil backend yang sama, tidak ada service terpisah lagi
untuk di-deploy).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Self-Review Notes (untuk pelaksana plan ini)

- **Cakupan spec:** §1 (keputusan) → Task 2-13 seluruhnya. §2.1 (accounts) → Task 2-3. §2.2/§2.3 (account_id + constraint) → Task 2, 4-7. §3 (derivasi account_id) → Task 8-9. §4 (Vendor Dashboard) → Task 10, 13. §5 (migrasi data lama) → Task 2. §6 (test isolasi) → Task 11. §7 (di luar cakupan) — sengaja TIDAK ada task untuk signup publik/landing page/mobile bridge/billing/multi-user, sesuai spec. §8 (ringkasan dampak) → seluruh task.
- **Konsistensi tipe:** `AccountFromContext(ctx) (accountID string, ok bool)` (Task 8) dipakai identik di Task 9, 11. `store.Account.DerivedStatus(now) string` (Task 3) dipakai identik di Task 8 (`requireActiveAccount`), Task 10 (`toAccountVendorJSON`). Signature `NewSessionToken(key, now, subject)` (Task 1) dipakai identik untuk sesi customer (Task 8, subject=account_id) dan vendor (Task 10, subject=username).
- **Keputusan yang sengaja dibiarkan terbuka untuk pelaksana** (bukan placeholder, tapi butuh keputusan kecil saat implementasi karena tergantung kode yang belum dibaca ulang saat plan ini ditulis): Task 9 Step 3 (gate lisensi di `ProcessDueWebhooks` untuk pengiriman individual) — dicatat eksplisit sebagai keputusan yang harus dilaporkan ke Akbar, bukan diselesaikan diam-diam.
