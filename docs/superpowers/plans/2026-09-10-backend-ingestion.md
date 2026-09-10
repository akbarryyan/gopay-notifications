# Backend Event Ingestion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Membangun layanan Go yang menerima event notifikasi GoPay dari perangkat Android, memvalidasinya dengan HMAC, dan menyimpannya secara idempoten ke PostgreSQL.

**Architecture:** Satu binary HTTP tanpa framework. Autentikasi HMAC dijalankan sebagai middleware yang membaca body mentah dan memverifikasinya **sebelum** decode JSON. Idempotency bersandar pada unique constraint database (`ON CONFLICT DO NOTHING`), bukan pada pengecekan `SELECT` lebih dulu, sehingga tetap benar saat dua request identik tiba bersamaan.

**Tech Stack:** Go 1.23+, `net/http` (ServeMux pola Go 1.22), `pgx/v5`, PostgreSQL 16, `goose` untuk migrasi, Caddy sebagai reverse proxy.

**Milestone:** M1 dari [spec §8](../specs/2026-09-10-ingestion-and-android-bridge-design.md).

## Global Constraints

- Kontrak API yang mengikat: [`docs/api-contract.md`](../../api-contract.md). Bila plan ini dan kontrak berbeda, kontrak yang menang.
- Go minimal **1.23**. Pola `ServeMux` bermetode (`POST /path`) wajib dipakai — jangan menambahkan router pihak ketiga.
- Dependensi non-test yang diizinkan hanya `github.com/jackc/pgx/v5`. Apa pun di luar itu harus dibahas dulu.
- Tanda tangan **wajib** dibandingkan dengan `hmac.Equal`, tidak pernah `==`.
- Body **wajib** diverifikasi sebagai byte mentah sebelum di-decode JSON.
- Toleransi timestamp **±300 detik**.
- Secret device disimpan terenkripsi AES-256-GCM dengan kunci dari env `DEVICE_SECRET_KEY`.
- Dilarang menulis secret, tanda tangan, atau isi header autentikasi ke log.
- Nama kolom untuk isi notifikasi adalah `body_text`, bukan `text` — menghindari kebingungan dengan tipe `text` Postgres.
- Aturan arah transaksi (allowlist judul) **bukan** bagian plan ini. Plan ini hanya menerima dan menyimpan.

---

### Task 1: Scaffold, Postgres, dan migrasi

**Files:**
- Create: `backend/go.mod`
- Create: `backend/docker-compose.yml`
- Create: `backend/Makefile`
- Create: `backend/migrations/00001_init.sql`
- Create: `backend/internal/store/store.go`
- Test: `backend/internal/store/store_test.go`

**Interfaces:**
- Consumes: tidak ada.
- Produces: `store.New(ctx context.Context, databaseURL string) (*store.Store, error)`, `(*store.Store).Pool() *pgxpool.Pool`, `(*store.Store).Close()`.

- [ ] **Step 1: Inisialisasi module**

```bash
mkdir -p backend && cd backend
go mod init github.com/akbarryyan/gopay-notifications/backend
go get github.com/jackc/pgx/v5@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
```

- [ ] **Step 2: Tulis `backend/docker-compose.yml`**

Port 5433 dipakai agar tidak bentrok dengan PostgreSQL lain yang mungkin sudah jalan di laptop.

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: gopay
      POSTGRES_PASSWORD: gopay
      POSTGRES_DB: gopay_test
    ports:
      - "5433:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gopay"]
      interval: 2s
      timeout: 3s
      retries: 15
```

- [ ] **Step 3: Tulis `backend/migrations/00001_init.sql`**

```sql
-- +goose Up
CREATE TABLE devices (
    device_id    TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    secret_enc   BYTEA NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ
);

CREATE TABLE notification_events (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id     TEXT NOT NULL UNIQUE,
    device_id    TEXT NOT NULL REFERENCES devices(device_id),
    source       TEXT NOT NULL,
    package_name TEXT NOT NULL,
    title        TEXT,
    body_text    TEXT,
    big_text     TEXT,
    amount_hint  BIGINT,
    posted_at    TIMESTAMPTZ NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL,
    ingested_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    raw_payload  JSONB NOT NULL
);

CREATE INDEX notification_events_ingested_at_idx
    ON notification_events (ingested_at DESC);

-- +goose Down
DROP TABLE notification_events;
DROP TABLE devices;
```

- [ ] **Step 4: Tulis `backend/Makefile`**

```makefile
TEST_DATABASE_URL ?= postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable

.PHONY: db-up db-down migrate test

db-up:
	docker compose up -d --wait

db-down:
	docker compose down -v

migrate:
	goose -dir migrations postgres "$(TEST_DATABASE_URL)" up

# -p 1 wajib: paket store dan httpapi sama-sama TRUNCATE database test yang
# sama. Tanpa ini Go menjalankan keduanya paralel dan mereka saling menghapus
# data di tengah jalan — gagal secara acak, bukan karena kodenya salah.
test: db-up migrate
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./... -count=1 -p 1
```

- [ ] **Step 5: Tulis test yang gagal — `backend/internal/store/store_test.go`**

```go
package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// TestPool membuka koneksi ke database test dan mengosongkan seluruh tabel.
// Dipakai ulang oleh test lain di paket ini.
func testStore(t *testing.T) *store.Store {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL belum diset. Jalankan: make db-up migrate")
	}

	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)

	_, err = s.Pool().Exec(ctx,
		"TRUNCATE notification_events, devices RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

func TestNewConnectsAndTablesExist(t *testing.T) {
	s := testStore(t)

	var n int
	err := s.Pool().QueryRow(context.Background(),
		`SELECT count(*) FROM information_schema.tables
		 WHERE table_schema = 'public'
		   AND table_name IN ('devices', 'notification_events')`).Scan(&n)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 2 {
		t.Fatalf("jumlah tabel = %d, mau 2 — migrasi belum dijalankan?", n)
	}
}

func TestNewRejectsBadURL(t *testing.T) {
	_, err := store.New(context.Background(), "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	if err == nil {
		t.Fatal("mau error untuk URL yang tidak bisa dihubungi, dapat nil")
	}
}
```

- [ ] **Step 6: Jalankan test, pastikan gagal**

Run: `cd backend && make test`
Expected: FAIL — `undefined: store.New` (paket `store` belum ada).

- [ ] **Step 7: Tulis `backend/internal/store/store.go`**

```go
// Package store membungkus akses PostgreSQL untuk layanan ingestion.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

// New membuka connection pool dan memastikan database benar-benar dapat dihubungi.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: buka pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}

	return &Store{pool: pool}, nil
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func (s *Store) Close() { s.pool.Close() }
```

- [ ] **Step 8: Jalankan test, pastikan lulus**

Run: `cd backend && make test`
Expected: PASS — `ok ... internal/store`

- [ ] **Step 9: Commit**

```bash
git add backend/
git commit -m "feat(backend): scaffold module, Postgres, dan migrasi awal"
```

---

### Task 2: Enkripsi secret device

**Files:**
- Create: `backend/internal/secretbox/secretbox.go`
- Test: `backend/internal/secretbox/secretbox_test.go`

**Interfaces:**
- Consumes: tidak ada.
- Produces: `secretbox.Seal(key, plaintext []byte) ([]byte, error)`, `secretbox.Open(key, sealed []byte) ([]byte, error)`. `key` wajib 32 byte.

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/secretbox/secretbox_test.go`**

```go
package secretbox_test

import (
	"bytes"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
)

func key32() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestSealOpenRoundTrip(t *testing.T) {
	plain := []byte("rahasia-device-01")

	sealed, err := secretbox.Seal(key32(), plain)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(sealed, plain) {
		t.Fatal("ciphertext masih memuat plaintext")
	}

	got, err := secretbox.Open(key32(), sealed)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("Open = %q, mau %q", got, plain)
	}
}

func TestSealProducesDifferentCiphertextEachTime(t *testing.T) {
	a, err := secretbox.Seal(key32(), []byte("sama"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	b, err := secretbox.Seal(key32(), []byte("sama"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("dua Seal atas plaintext sama menghasilkan ciphertext identik — nonce tidak acak")
	}
}

func TestOpenRejectsWrongKey(t *testing.T) {
	sealed, err := secretbox.Seal(key32(), []byte("rahasia"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	wrong := key32()
	wrong[0] ^= 0xFF

	if _, err := secretbox.Open(wrong, sealed); err == nil {
		t.Fatal("mau error untuk kunci salah, dapat nil")
	}
}

func TestOpenRejectsTamperedCiphertext(t *testing.T) {
	sealed, err := secretbox.Seal(key32(), []byte("rahasia"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	sealed[len(sealed)-1] ^= 0x01

	if _, err := secretbox.Open(key32(), sealed); err == nil {
		t.Fatal("mau error untuk ciphertext yang diubah, dapat nil")
	}
}

func TestOpenRejectsShortInput(t *testing.T) {
	if _, err := secretbox.Open(key32(), []byte{1, 2, 3}); err == nil {
		t.Fatal("mau error untuk input lebih pendek dari nonce, dapat nil")
	}
}

func TestSealRejectsWrongKeySize(t *testing.T) {
	if _, err := secretbox.Seal([]byte("pendek"), []byte("x")); err == nil {
		t.Fatal("mau error untuk kunci bukan 32 byte, dapat nil")
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `cd backend && go test ./internal/secretbox/ -v`
Expected: FAIL — paket `secretbox` belum ada.

- [ ] **Step 3: Tulis `backend/internal/secretbox/secretbox.go`**

```go
// Package secretbox mengenkripsi secret device saat disimpan di database,
// sehingga dump database yang bocor tidak langsung membocorkan secret.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// KeySize adalah panjang kunci yang diwajibkan, dalam byte.
const KeySize = 32

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("secretbox: kunci harus %d byte, dapat %d", KeySize, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secretbox: aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: gcm: %w", err)
	}
	return gcm, nil
}

// Seal mengenkripsi plaintext. Nonce acak diletakkan di depan ciphertext.
func Seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("secretbox: nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Open mendekripsi hasil Seal. Ciphertext yang diubah akan ditolak.
func Open(key, sealed []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, errors.New("secretbox: ciphertext lebih pendek dari nonce")
	}
	nonce, ct := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("secretbox: open: %w", err)
	}
	return plain, nil
}
```

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `cd backend && go test ./internal/secretbox/ -v`
Expected: PASS — enam test lulus.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/secretbox/
git commit -m "feat(backend): enkripsi AES-GCM untuk secret device"
```

---

### Task 3: Tanda tangan HMAC

**Files:**
- Create: `backend/internal/auth/hmac.go`
- Test: `backend/internal/auth/hmac_test.go`

**Interfaces:**
- Consumes: tidak ada.
- Produces: `auth.SigningString(deviceID string, timestamp int64, body []byte) string`, `auth.Sign(secret []byte, signingString string) string`, `auth.Verify(secret []byte, signingString, gotSignature string) bool`, `auth.SkewTolerance` (konstanta `time.Duration`), `auth.CheckSkew(timestamp int64, now time.Time) bool`.

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/auth/hmac_test.go`**

```go
package auth_test

import (
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
)

var secret = []byte("secret-device-01")

func TestSigningStringFormat(t *testing.T) {
	// sha256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	got := auth.SigningString("dev_01ABC", 1789036200, nil)
	want := "dev_01ABC\n1789036200\n" +
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("SigningString =\n%q\nmau\n%q", got, want)
	}
}

func TestVerifyAcceptsCorrectSignature(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, s)

	if !auth.Verify(secret, s, sig) {
		t.Fatal("Verify menolak tanda tangan yang benar")
	}
}

func TestVerifyRejectsModifiedBody(t *testing.T) {
	orig := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, orig)

	// satu byte berubah
	tampered := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":2}`))

	if auth.Verify(secret, tampered, sig) {
		t.Fatal("Verify menerima body yang sudah diubah")
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, s)

	if auth.Verify([]byte("secret-lain"), s, sig) {
		t.Fatal("Verify menerima secret yang salah")
	}
}

func TestVerifyRejectsWrongDeviceID(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, s)

	other := auth.SigningString("dev_LAIN", 1789036200, []byte(`{"a":1}`))
	if auth.Verify(secret, other, sig) {
		t.Fatal("Verify menerima device id yang berbeda")
	}
}

func TestVerifyRejectsGarbageSignature(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))

	for _, bad := range []string{"", "zzzz", "00"} {
		if auth.Verify(secret, s, bad) {
			t.Fatalf("Verify menerima tanda tangan sampah %q", bad)
		}
	}
}

func TestCheckSkewBoundaries(t *testing.T) {
	now := time.Unix(1789036200, 0)

	cases := []struct {
		name   string
		offset time.Duration
		want   bool
	}{
		{"tepat sekarang", 0, true},
		{"299 detik lampau", -299 * time.Second, true},
		{"300 detik lampau", -300 * time.Second, true},
		{"301 detik lampau", -301 * time.Second, false},
		{"299 detik depan", 299 * time.Second, true},
		{"300 detik depan", 300 * time.Second, true},
		{"301 detik depan", 301 * time.Second, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := now.Add(tc.offset).Unix()
			if got := auth.CheckSkew(ts, now); got != tc.want {
				t.Fatalf("CheckSkew(%s) = %v, mau %v", tc.name, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `cd backend && go test ./internal/auth/ -v`
Expected: FAIL — paket `auth` belum ada.

- [ ] **Step 3: Tulis `backend/internal/auth/hmac.go`**

```go
// Package auth mengurus autentikasi HMAC antara perangkat Android dan backend.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// SkewTolerance adalah selisih waktu maksimum yang masih diterima
// antara jam perangkat dan jam server.
const SkewTolerance = 5 * time.Minute

// SigningString membentuk string yang ditandatangani, sesuai api-contract.md §3.2.
//
// Yang di-hash adalah byte mentah body. Jangan pernah memanggil fungsi ini
// dengan hasil re-encode JSON — urutan field dan spasi akan berbeda dari
// yang dikirim perangkat, dan tanda tangan akan gagal secara acak.
func SigningString(deviceID string, timestamp int64, body []byte) string {
	sum := sha256.Sum256(body)
	return deviceID + "\n" +
		strconv.FormatInt(timestamp, 10) + "\n" +
		hex.EncodeToString(sum[:])
}

// Sign menghasilkan tanda tangan heksadesimal untuk signingString.
func Sign(secret []byte, signingString string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingString))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify membandingkan tanda tangan dengan waktu konstan.
func Verify(secret []byte, signingString, gotSignature string) bool {
	want := Sign(secret, signingString)
	return hmac.Equal([]byte(want), []byte(gotSignature))
}

// CheckSkew melaporkan apakah timestamp (Unix detik) masih dalam toleransi.
func CheckSkew(timestamp int64, now time.Time) bool {
	diff := now.Sub(time.Unix(timestamp, 0))
	if diff < 0 {
		diff = -diff
	}
	return diff <= SkewTolerance
}
```

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `cd backend && go test ./internal/auth/ -v`
Expected: PASS — termasuk seluruh tujuh subtest `TestCheckSkewBoundaries`.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/auth/
git commit -m "feat(backend): tanda tangan HMAC dan toleransi timestamp"
```

---

### Task 4: Penyimpanan device dan CLI pembuat device

**Files:**
- Create: `backend/internal/store/device.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/cmd/devicetool/main.go`
- Test: `backend/internal/store/device_test.go`

**Interfaces:**
- Consumes: `store.New`, `secretbox.Seal`, `secretbox.Open`.
- Produces:
  - `type store.Device struct { DeviceID, Name string; Secret []byte; Enabled bool; LastSeenAt *time.Time }`
  - `(*store.Store).CreateDevice(ctx context.Context, key []byte, deviceID, name string, secret []byte) error`
  - `(*store.Store).GetDevice(ctx context.Context, key []byte, deviceID string) (store.Device, error)`
  - `store.ErrDeviceNotFound` (sentinel error)
  - `(*store.Store).TouchDevice(ctx context.Context, deviceID string) error`
  - `config.Load() (config.Config, error)` dengan field `DatabaseURL`, `ListenAddr`, `DeviceSecretKey []byte`

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/store/device_test.go`**

```go
package store_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func encKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 7)
	}
	return k
}

func TestCreateAndGetDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	secret := []byte("secret-abc")

	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP GoPay", secret); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.DeviceID != "dev_01ABC" || got.Name != "HP GoPay" {
		t.Fatalf("device = %+v", got)
	}
	if !got.Enabled {
		t.Fatal("device baru seharusnya enabled")
	}
	if !bytes.Equal(got.Secret, secret) {
		t.Fatalf("Secret = %q, mau %q", got.Secret, secret)
	}
	if got.LastSeenAt != nil {
		t.Fatal("device baru seharusnya belum punya LastSeenAt")
	}
}

func TestSecretIsNotStoredInPlaintext(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP", []byte("secret-abc")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	var raw []byte
	err := s.Pool().QueryRow(ctx,
		"SELECT secret_enc FROM devices WHERE device_id = $1", "dev_01ABC").Scan(&raw)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if bytes.Contains(raw, []byte("secret-abc")) {
		t.Fatal("secret tersimpan sebagai plaintext di kolom secret_enc")
	}
}

func TestGetDeviceUnknownReturnsSentinel(t *testing.T) {
	s := testStore(t)

	_, err := s.GetDevice(context.Background(), encKey(), "dev_TIDAKADA")
	if !errors.Is(err, store.ErrDeviceNotFound) {
		t.Fatalf("err = %v, mau ErrDeviceNotFound", err)
	}
}

func TestTouchDeviceSetsLastSeenAt(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.TouchDevice(ctx, "dev_01ABC"); err != nil {
		t.Fatalf("TouchDevice: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.LastSeenAt == nil {
		t.Fatal("LastSeenAt masih nil setelah TouchDevice")
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `cd backend && make test`
Expected: FAIL — `s.CreateDevice undefined`.

- [ ] **Step 3: Tulis `backend/internal/store/device.go`**

```go
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
	"github.com/jackc/pgx/v5"
)

// ErrDeviceNotFound dikembalikan bila device_id tidak terdaftar.
var ErrDeviceNotFound = errors.New("store: device tidak ditemukan")

type Device struct {
	DeviceID   string
	Name       string
	Secret     []byte
	Enabled    bool
	LastSeenAt *time.Time
}

// CreateDevice menyimpan device baru dengan secret terenkripsi.
func (s *Store) CreateDevice(ctx context.Context, key []byte, deviceID, name string, secret []byte) error {
	enc, err := secretbox.Seal(key, secret)
	if err != nil {
		return fmt.Errorf("store: enkripsi secret: %w", err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO devices (device_id, name, secret_enc) VALUES ($1, $2, $3)`,
		deviceID, name, enc)
	if err != nil {
		return fmt.Errorf("store: insert device: %w", err)
	}
	return nil
}

// GetDevice mengambil device beserta secret yang sudah didekripsi.
func (s *Store) GetDevice(ctx context.Context, key []byte, deviceID string) (Device, error) {
	var (
		d   Device
		enc []byte
	)
	err := s.pool.QueryRow(ctx,
		`SELECT device_id, name, secret_enc, enabled, last_seen_at
		 FROM devices WHERE device_id = $1`, deviceID).
		Scan(&d.DeviceID, &d.Name, &enc, &d.Enabled, &d.LastSeenAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Device{}, ErrDeviceNotFound
	}
	if err != nil {
		return Device{}, fmt.Errorf("store: select device: %w", err)
	}

	secret, err := secretbox.Open(key, enc)
	if err != nil {
		return Device{}, fmt.Errorf("store: dekripsi secret device %s: %w", deviceID, err)
	}
	d.Secret = secret
	return d, nil
}

// TouchDevice memperbarui last_seen_at. Dipanggil setelah autentikasi berhasil,
// sehingga tidak dibutuhkan heartbeat berkala dari perangkat.
func (s *Store) TouchDevice(ctx context.Context, deviceID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE devices SET last_seen_at = now() WHERE device_id = $1`, deviceID)
	if err != nil {
		return fmt.Errorf("store: touch device: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Tulis `backend/internal/config/config.go`**

```go
// Package config membaca konfigurasi layanan dari environment variable.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
)

type Config struct {
	DatabaseURL     string
	ListenAddr      string
	DeviceSecretKey []byte
}

// Load membaca dan memvalidasi seluruh konfigurasi. Konfigurasi yang salah
// harus menghentikan proses saat start, bukan saat request pertama masuk.
func Load() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ListenAddr:  os.Getenv("LISTEN_ADDR"),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL wajib diisi")
	}
	if c.ListenAddr == "" {
		c.ListenAddr = ":8080"
	}

	raw := os.Getenv("DEVICE_SECRET_KEY")
	if raw == "" {
		return Config{}, errors.New("config: DEVICE_SECRET_KEY wajib diisi")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return Config{}, fmt.Errorf("config: DEVICE_SECRET_KEY bukan base64 yang sah: %w", err)
	}
	if len(key) != secretbox.KeySize {
		return Config{}, fmt.Errorf("config: DEVICE_SECRET_KEY harus %d byte setelah decode, dapat %d",
			secretbox.KeySize, len(key))
	}
	c.DeviceSecretKey = key
	return c, nil
}
```

- [ ] **Step 5: Tulis `backend/cmd/devicetool/main.go`**

```go
// Command devicetool membuat device baru beserta secret-nya.
//
// Secret dicetak satu kali ke stdout dan tidak pernah dapat dibaca kembali
// dari database dalam bentuk yang mudah — salin langsung ke Settings aplikasi.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/akbarryyan/gopay-notifications/backend/internal/config"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func main() {
	name := flag.String("name", "", "nama device, contoh: \"HP GoPay Utama\"")
	genKey := flag.Bool("genkey", false, "cetak DEVICE_SECRET_KEY baru lalu keluar")
	flag.Parse()

	if *genKey {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			fail("acak kunci: %v", err)
		}
		fmt.Println(base64.StdEncoding.EncodeToString(key))
		return
	}

	if *name == "" {
		fail("-name wajib diisi")
	}

	cfg, err := config.Load()
	if err != nil {
		fail("%v", err)
	}

	ctx := context.Background()
	s, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fail("%v", err)
	}
	defer s.Close()

	idSuffix := make([]byte, 8)
	if _, err := rand.Read(idSuffix); err != nil {
		fail("acak device id: %v", err)
	}
	deviceID := "dev_" + hex.EncodeToString(idSuffix)

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		fail("acak secret: %v", err)
	}
	secretB64 := base64.StdEncoding.EncodeToString(secret)

	if err := s.CreateDevice(ctx, cfg.DeviceSecretKey, deviceID, *name, []byte(secretB64)); err != nil {
		fail("%v", err)
	}

	fmt.Println("Device berhasil dibuat. Salin dua nilai ini ke Settings aplikasi Android:")
	fmt.Println()
	fmt.Println("  Device ID     :", deviceID)
	fmt.Println("  Device Secret :", secretB64)
	fmt.Println()
	fmt.Println("Secret tidak akan ditampilkan lagi.")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "devicetool: "+format+"\n", args...)
	os.Exit(1)
}
```

Catatan: secret yang disimpan adalah **string base64 itu sendiri** sebagai byte, bukan hasil decode-nya. Perangkat Android menandatangani memakai byte dari string base64 yang sama persis, sehingga tidak ada ambiguitas encoding antara kedua sisi.

- [ ] **Step 6: Jalankan test, pastikan lulus**

Run: `cd backend && make test`
Expected: PASS — empat test device lulus.

- [ ] **Step 7: Verifikasi devicetool secara manual**

```bash
cd backend
export DEVICE_SECRET_KEY=$(go run ./cmd/devicetool -genkey)
export DATABASE_URL='postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable'
go run ./cmd/devicetool -name "HP GoPay Utama"
```

Expected: mencetak `Device ID` dan `Device Secret`. Simpan keduanya — dipakai di Task 8.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/store/device.go backend/internal/store/device_test.go \
        backend/internal/config/ backend/cmd/devicetool/
git commit -m "feat(backend): penyimpanan device dan CLI pembuat device"
```

---

### Task 5: Penyimpanan event dengan idempotency

**Files:**
- Create: `backend/internal/store/event.go`
- Test: `backend/internal/store/event_test.go`

**Interfaces:**
- Consumes: `store.New`, `(*store.Store).CreateDevice`.
- Produces:
  - `type store.Event struct { EventID, DeviceID, Source, PackageName string; Title, BodyText, BigText *string; AmountHint *int64; PostedAt, ReceivedAt time.Time; RawPayload []byte }`
  - `(*store.Store).InsertEvent(ctx context.Context, e store.Event) (inserted bool, err error)`
  - `(*store.Store).ListEvents(ctx context.Context, limit, offset int) ([]store.Event, error)`

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/store/event_test.go`**

```go
package store_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func seedDevice(t *testing.T, s *store.Store) {
	t.Helper()
	err := s.CreateDevice(context.Background(), encKey(), "dev_01ABC", "HP", []byte("secret"))
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
}

func sampleEvent(eventID string) store.Event {
	title := "Transfer masuk"
	body := "Rp1 dari icaangg udah masuk ke GoPay kamu."
	amount := int64(1)
	return store.Event{
		EventID:     eventID,
		DeviceID:    "dev_01ABC",
		Source:      "gopay",
		PackageName: "com.gojek.gopay",
		Title:       &title,
		BodyText:    &body,
		BigText:     &body,
		AmountHint:  &amount,
		PostedAt:    time.Unix(1789051832, 0),
		ReceivedAt:  time.Unix(1789051833, 0),
		RawPayload:  []byte(`{"event_id":"` + eventID + `"}`),
	}
}

func TestInsertEventFirstTimeIsAccepted(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)

	inserted, err := s.InsertEvent(context.Background(), sampleEvent("evt_aaa"))
	if err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}
	if !inserted {
		t.Fatal("inserted = false pada penyisipan pertama")
	}
}

func TestInsertEventSecondTimeIsDuplicate(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	if _, err := s.InsertEvent(ctx, sampleEvent("evt_aaa")); err != nil {
		t.Fatalf("InsertEvent pertama: %v", err)
	}

	inserted, err := s.InsertEvent(ctx, sampleEvent("evt_aaa"))
	if err != nil {
		t.Fatalf("InsertEvent kedua: %v", err)
	}
	if inserted {
		t.Fatal("inserted = true padahal event_id sudah ada")
	}

	var n int
	if err := s.Pool().QueryRow(ctx,
		"SELECT count(*) FROM notification_events").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("jumlah baris = %d, mau 1", n)
	}
}

// Ini inti dari jaminan idempotency: dua request identik yang tiba
// bersamaan harus menghasilkan tepat satu baris dan tepat satu accepted.
func TestInsertEventConcurrentSameIDInsertsOnce(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	const goroutines = 8

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		accepted int
	)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := s.InsertEvent(ctx, sampleEvent("evt_race"))
			if err != nil {
				t.Errorf("InsertEvent: %v", err)
				return
			}
			if ok {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}()
	}

	close(start)
	wg.Wait()

	if accepted != 1 {
		t.Fatalf("accepted = %d, mau tepat 1", accepted)
	}

	var n int
	if err := s.Pool().QueryRow(ctx,
		"SELECT count(*) FROM notification_events").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("jumlah baris = %d, mau 1", n)
	}
}

func TestListEventsNewestFirst(t *testing.T) {
	s := testStore(t)
	seedDevice(t, s)
	ctx := context.Background()

	for _, id := range []string{"evt_1", "evt_2", "evt_3"} {
		if _, err := s.InsertEvent(ctx, sampleEvent(id)); err != nil {
			t.Fatalf("InsertEvent %s: %v", id, err)
		}
	}

	got, err := s.ListEvents(ctx, 2, 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, mau 2", len(got))
	}
	if got[0].EventID != "evt_3" {
		t.Fatalf("event pertama = %s, mau evt_3 (terbaru dulu)", got[0].EventID)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `cd backend && make test`
Expected: FAIL — `s.InsertEvent undefined`.

- [ ] **Step 3: Tulis `backend/internal/store/event.go`**

```go
package store

import (
	"context"
	"fmt"
	"time"
)

type Event struct {
	EventID     string
	DeviceID    string
	Source      string
	PackageName string
	Title       *string
	BodyText    *string
	BigText     *string
	AmountHint  *int64
	PostedAt    time.Time
	ReceivedAt  time.Time
	RawPayload  []byte
}

// InsertEvent menyimpan event dan melaporkan apakah ia benar-benar baru.
//
// Idempotency bersandar pada unique constraint, bukan pada SELECT lebih dulu.
// Pendekatan SELECT-lalu-INSERT salah bila dua request identik tiba bersamaan:
// keduanya akan melihat baris belum ada, lalu keduanya menyisipkan.
func (s *Store) InsertEvent(ctx context.Context, e Event) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO notification_events
		   (event_id, device_id, source, package_name,
		    title, body_text, big_text, amount_hint,
		    posted_at, received_at, raw_payload)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (event_id) DO NOTHING`,
		e.EventID, e.DeviceID, e.Source, e.PackageName,
		e.Title, e.BodyText, e.BigText, e.AmountHint,
		e.PostedAt, e.ReceivedAt, e.RawPayload)
	if err != nil {
		return false, fmt.Errorf("store: insert event: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ListEvents mengembalikan event terbaru lebih dulu.
func (s *Store) ListEvents(ctx context.Context, limit, offset int) ([]Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT event_id, device_id, source, package_name,
		        title, body_text, big_text, amount_hint,
		        posted_at, received_at, raw_payload
		 FROM notification_events
		 ORDER BY ingested_at DESC, id DESC
		 LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list events: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.EventID, &e.DeviceID, &e.Source, &e.PackageName,
			&e.Title, &e.BodyText, &e.BigText, &e.AmountHint,
			&e.PostedAt, &e.ReceivedAt, &e.RawPayload); err != nil {
			return nil, fmt.Errorf("store: scan event: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi events: %w", err)
	}
	return out, nil
}
```

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `cd backend && make test`
Expected: PASS — termasuk `TestInsertEventConcurrentSameIDInsertsOnce`.

- [ ] **Step 5: Jalankan test balapan berulang untuk memastikan bukan kebetulan**

Run: `cd backend && go test ./internal/store/ -run Concurrent -count=20 -race`
Expected: PASS 20 kali berturut-turut, tanpa laporan race.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/store/event.go backend/internal/store/event_test.go
git commit -m "feat(backend): penyimpanan event idempoten lewat ON CONFLICT"
```

---

### Task 6: Endpoint /health dan kerangka server

**Files:**
- Create: `backend/internal/httpapi/api.go`
- Create: `backend/internal/httpapi/health.go`
- Create: `backend/cmd/server/main.go`
- Test: `backend/internal/httpapi/health_test.go`

**Interfaces:**
- Consumes: `store.Store`, `config.Config`.
- Produces: `httpapi.New(s *store.Store, encKey []byte, now func() time.Time) *httpapi.API`, `(*httpapi.API).Handler() http.Handler`.

Parameter `now` disuntikkan agar test dapat memalsukan jam tanpa menunggu waktu nyata.

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/httpapi/health_test.go`**

```go
package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
)

var fixedNow = time.Unix(1789036200, 0)

func newTestAPI(t *testing.T) http.Handler {
	t.Helper()
	return httpapi.New(nil, nil, func() time.Time { return fixedNow }).Handler()
}

func TestHealthReturnsOKAndServerTime(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	newTestAPI(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}

	var body struct {
		Status     string `json:"status"`
		ServerTime int64  `json:"server_time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, mau \"ok\"", body.Status)
	}
	if body.ServerTime != fixedNow.Unix() {
		t.Fatalf("server_time = %d, mau %d", body.ServerTime, fixedNow.Unix())
	}
}

func TestHealthRejectsPost(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)

	newTestAPI(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, mau 405", rec.Code)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `cd backend && go test ./internal/httpapi/ -v`
Expected: FAIL — paket `httpapi` belum ada.

- [ ] **Step 3: Tulis `backend/internal/httpapi/api.go`**

```go
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
	store  *store.Store
	encKey []byte
	now    func() time.Time
}

// New membuat API. Parameter now disuntikkan agar test dapat memalsukan jam.
func New(s *store.Store, encKey []byte, now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return &API{store: s, encKey: encKey, now: now}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
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
```

Catatan: pola `"GET /api/v1/health"` membuat `ServeMux` Go 1.22+ otomatis menjawab `405` untuk metode lain pada path yang sama. Tidak perlu penanganan manual.

- [ ] **Step 4: Tulis `backend/internal/httpapi/health.go`**

```go
package httpapi

import "net/http"

type healthResponse struct {
	Status     string `json:"status"`
	ServerTime int64  `json:"server_time"`
}

// handleHealth tidak memerlukan autentikasi dan menyertakan jam server,
// sehingga perangkat dapat mendeteksi jamnya sendiri meleset sebelum
// mengirim apa pun.
func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:     "ok",
		ServerTime: a.now().Unix(),
	})
}
```

- [ ] **Step 5: Tulis `backend/cmd/server/main.go`**

```go
// Command server menjalankan layanan ingestion event notifikasi GoPay.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/config"
	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("konfigurasi tidak sah", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("koneksi database gagal", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpapi.New(s, cfg.DeviceSecretKey, time.Now).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		slog.Info("server mulai", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server berhenti", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown dimulai")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown gagal", "err", err)
	}
}
```

- [ ] **Step 6: Jalankan test, pastikan lulus**

Run: `cd backend && go test ./internal/httpapi/ -v`
Expected: PASS — dua test.

- [ ] **Step 7: Verifikasi server benar-benar jalan**

```bash
cd backend
export DATABASE_URL='postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable'
go run ./cmd/server &
sleep 2
curl -s localhost:8080/api/v1/health
kill %1
```

Expected: `{"status":"ok","server_time":<angka>}`

- [ ] **Step 8: Commit**

```bash
git add backend/internal/httpapi/ backend/cmd/server/
git commit -m "feat(backend): endpoint /health dan kerangka server"
```

---

### Task 7: Middleware autentikasi HMAC

**Files:**
- Create: `backend/internal/httpapi/auth_middleware.go`
- Modify: `backend/internal/httpapi/api.go` — tambah rute terautentikasi
- Test: `backend/internal/httpapi/auth_middleware_test.go`

**Interfaces:**
- Consumes: `auth.SigningString`, `auth.Sign`, `auth.Verify`, `auth.CheckSkew`, `(*store.Store).GetDevice`, `store.ErrDeviceNotFound`, `(*store.Store).TouchDevice`.
- Produces: `(*API).requireDevice(next http.Handler) http.Handler`, `httpapi.DeviceFromContext(ctx context.Context) (store.Device, bool)`, `httpapi.RawBodyFromContext(ctx context.Context) ([]byte, bool)`.

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/httpapi/auth_middleware_test.go`**

```go
package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

const testSecret = "secret-untuk-test"

func encKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 3)
	}
	return k
}

// newAPIWithDevice menyiapkan API lengkap dengan satu device terdaftar.
func newAPIWithDevice(t *testing.T) http.Handler {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL belum diset. Jalankan: make db-up migrate")
	}

	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)

	if _, err := s.Pool().Exec(ctx,
		"TRUNCATE notification_events, devices RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP Test", []byte(testSecret)); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	return httpapi.New(s, encKey(), func() time.Time { return fixedNow }).Handler()
}

// signedRequest membuat request yang sudah ditandatangani dengan benar.
func signedRequest(method, path, body string, ts int64, secret string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Id", "dev_01ABC")
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))
	s := auth.SigningString("dev_01ABC", ts, []byte(body))
	req.Header.Set("X-Signature", auth.Sign([]byte(secret), s))
	return req
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v (body=%s)", err, rec.Body.String())
	}
	return body.Error
}

func TestAuthAcceptsValidSignature(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestAuthRejectsWrongSecret(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), "secret-salah"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_signature" {
		t.Fatalf("error = %q, mau invalid_signature", got)
	}
}

func TestAuthRejectsUnknownDevice(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	req := signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret)
	req.Header.Set("X-Device-Id", "dev_TIDAKADA")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_signature" {
		t.Fatalf("error = %q, mau invalid_signature — device tidak dikenal tidak boleh dibedakan dari secret salah", got)
	}
}

func TestAuthRejectsClockSkewWithServerTime(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	skewed := fixedNow.Add(-301 * time.Second).Unix()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", skewed, testSecret))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if got := errorCode(t, rec); got != "clock_skew" {
		t.Fatalf("error = %q, mau clock_skew", got)
	}

	var body struct {
		ServerTime int64 `json:"server_time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ServerTime != fixedNow.Unix() {
		t.Fatalf("server_time = %d, mau %d", body.ServerTime, fixedNow.Unix())
	}
}

func TestAuthRejectsMissingHeaders(t *testing.T) {
	h := newAPIWithDevice(t)

	for _, drop := range []string{"X-Device-Id", "X-Timestamp", "X-Signature"} {
		t.Run("tanpa "+drop, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret)
			req.Header.Del(drop)

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, mau 401", rec.Code)
			}
		})
	}
}

func TestAuthRejectsNonNumericTimestamp(t *testing.T) {
	h := newAPIWithDevice(t)
	rec := httptest.NewRecorder()

	req := signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret)
	req.Header.Set("X-Timestamp", "kemarin")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

func TestAuthRejectsDisabledDevice(t *testing.T) {
	h := newAPIWithDevice(t)

	// nonaktifkan device lewat koneksi terpisah
	url := os.Getenv("TEST_DATABASE_URL")
	s, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if _, err := s.Pool().Exec(context.Background(),
		"UPDATE devices SET enabled = false WHERE device_id = $1", "dev_01ABC"); err != nil {
		t.Fatalf("disable: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodGet, "/api/v1/device/me", "", fixedNow.Unix(), testSecret))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, mau 403", rec.Code)
	}
	if got := errorCode(t, rec); got != "device_disabled" {
		t.Fatalf("error = %q, mau device_disabled", got)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `cd backend && make test`
Expected: FAIL — rute `/api/v1/device/me` belum terdaftar, jadi statusnya 404 bukan 200.

- [ ] **Step 3: Tulis `backend/internal/httpapi/auth_middleware.go`**

```go
package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type ctxKey int

const (
	ctxKeyDevice ctxKey = iota
	ctxKeyRawBody
)

// maxBodyBytes membatasi ukuran body yang dibaca sebelum verifikasi.
const maxBodyBytes = 64 << 10 // 64 KiB

// DeviceFromContext mengambil device yang sudah terautentikasi.
func DeviceFromContext(ctx context.Context) (store.Device, bool) {
	d, ok := ctx.Value(ctxKeyDevice).(store.Device)
	return d, ok
}

// RawBodyFromContext mengambil byte body yang sudah diverifikasi.
func RawBodyFromContext(ctx context.Context) ([]byte, bool) {
	b, ok := ctx.Value(ctxKeyRawBody).([]byte)
	return b, ok
}

// requireDevice memverifikasi tanda tangan HMAC sebelum handler dijalankan.
//
// Body dibaca sebagai byte mentah dan diverifikasi lebih dulu, baru
// dikembalikan ke r.Body agar handler dapat men-decode-nya. Membalik urutan
// ini — decode lalu re-encode untuk verifikasi — membuat tanda tangan gagal
// secara acak karena urutan field dan spasi berubah.
func (a *API) requireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.Header.Get("X-Device-Id")
		tsRaw := r.Header.Get("X-Timestamp")
		sig := r.Header.Get("X-Signature")

		if deviceID == "" || tsRaw == "" || sig == "" {
			a.writeError(w, http.StatusUnauthorized, "invalid_signature",
				"header X-Device-Id, X-Timestamp, dan X-Signature wajib ada")
			return
		}

		ts, err := strconv.ParseInt(tsRaw, 10, 64)
		if err != nil {
			a.writeError(w, http.StatusUnauthorized, "invalid_signature",
				"X-Timestamp harus Unix epoch dalam detik")
			return
		}

		if !auth.CheckSkew(ts, a.now()) {
			a.writeError(w, http.StatusUnauthorized, "clock_skew",
				"selisih jam perangkat dan server melebihi 5 menit")
			return
		}

		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "body tidak dapat dibaca")
			return
		}

		device, err := a.store.GetDevice(r.Context(), a.encKey, deviceID)
		if errors.Is(err, store.ErrDeviceNotFound) {
			// Sengaja dilaporkan sebagai invalid_signature: device yang tidak
			// terdaftar tidak boleh dapat dibedakan dari secret yang salah.
			a.writeError(w, http.StatusUnauthorized, "invalid_signature", "autentikasi gagal")
			return
		}
		if err != nil {
			slog.Error("ambil device gagal", "device_id", deviceID, "err", err)
			a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
			return
		}

		if !auth.Verify(device.Secret, auth.SigningString(deviceID, ts, raw), sig) {
			a.writeError(w, http.StatusUnauthorized, "invalid_signature", "autentikasi gagal")
			return
		}

		if !device.Enabled {
			a.writeError(w, http.StatusForbidden, "device_disabled", "device dinonaktifkan")
			return
		}

		if err := a.store.TouchDevice(r.Context(), deviceID); err != nil {
			// Bukan alasan menolak request — cukup dicatat.
			slog.Warn("perbarui last_seen_at gagal", "device_id", deviceID, "err", err)
		}

		ctx := context.WithValue(r.Context(), ctxKeyDevice, device)
		ctx = context.WithValue(ctx, ctxKeyRawBody, raw)
		r = r.WithContext(ctx)
		r.Body = io.NopCloser(bytes.NewReader(raw))

		next.ServeHTTP(w, r)
	})
}
```

Urutan pemeriksaan disengaja: skew diperiksa **sebelum** tanda tangan, agar jam yang meleset dilaporkan sebagai `clock_skew` alih-alih tersamar menjadi `invalid_signature`. Pemeriksaan `enabled` diletakkan **setelah** verifikasi tanda tangan, agar keberadaan sebuah device tidak dapat diendus tanpa memegang secret-nya.

- [ ] **Step 4: Daftarkan rute terautentikasi — `backend/internal/httpapi/api.go`**

Ganti isi fungsi `Handler`:

```go
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
	mux.Handle("GET /api/v1/device/me", a.requireDevice(http.HandlerFunc(a.handleDeviceMe)))
	return mux
}
```

- [ ] **Step 5: Tulis `backend/internal/httpapi/device.go`**

```go
package httpapi

import (
	"net/http"
	"time"
)

type deviceMeResponse struct {
	DeviceID   string  `json:"device_id"`
	Name       string  `json:"name"`
	Enabled    bool    `json:"enabled"`
	LastSeenAt *string `json:"last_seen_at"`
}

// handleDeviceMe melayani tombol Test Connection di aplikasi Android.
func (a *API) handleDeviceMe(w http.ResponseWriter, r *http.Request) {
	device, ok := DeviceFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "device tidak ada di context")
		return
	}

	resp := deviceMeResponse{
		DeviceID: device.DeviceID,
		Name:     device.Name,
		Enabled:  device.Enabled,
	}
	if device.LastSeenAt != nil {
		s := device.LastSeenAt.Format(time.RFC3339)
		resp.LastSeenAt = &s
	}
	writeJSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 6: Jalankan test, pastikan lulus**

Run: `cd backend && make test`
Expected: PASS — seluruh test auth, termasuk tiga subtest header hilang.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/httpapi/
git commit -m "feat(backend): middleware HMAC dan endpoint /device/me"
```

---

### Task 8: Endpoint POST /callback/gopay

**Files:**
- Create: `backend/internal/httpapi/callback.go`
- Modify: `backend/internal/httpapi/api.go` — daftarkan rute
- Test: `backend/internal/httpapi/callback_test.go`

**Interfaces:**
- Consumes: `(*API).requireDevice`, `RawBodyFromContext`, `DeviceFromContext`, `(*store.Store).InsertEvent`, `store.Event`.
- Produces: rute `POST /api/v1/callback/gopay`.

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/httpapi/callback_test.go`**

```go
package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const validBody = `{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC",` +
	`"source":"gopay","notification":{"package_name":"com.gojek.gopay",` +
	`"title":"Transfer masuk","text":"Rp1 dari icaangg udah masuk ke GoPay kamu.",` +
	`"big_text":null,"posted_at":1789051832829},"amount_hint":1,` +
	`"received_at":"2026-09-10T19:30:33+07:00"}`

func postCallback(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(http.MethodPost, "/api/v1/callback/gopay", body, fixedNow.Unix(), testSecret))
	return rec
}

func callbackStatus(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Success bool   `json:"success"`
		EventID string `json:"event_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}
	if !body.Success {
		t.Fatalf("success = false, body=%s", rec.Body.String())
	}
	return body.Status
}

func TestCallbackFirstTimeReturnsAccepted(t *testing.T) {
	h := newAPIWithDevice(t)

	rec := postCallback(t, h, validBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := callbackStatus(t, rec); got != "accepted" {
		t.Fatalf("status = %q, mau accepted", got)
	}
}

func TestCallbackSecondTimeReturnsDuplicate(t *testing.T) {
	h := newAPIWithDevice(t)

	postCallback(t, h, validBody)
	rec := postCallback(t, h, validBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
	if got := callbackStatus(t, rec); got != "duplicate" {
		t.Fatalf("status = %q, mau duplicate", got)
	}
}

func TestCallbackRejectsMalformedJSON(t *testing.T) {
	h := newAPIWithDevice(t)

	rec := postCallback(t, h, `{"event_id":`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_payload" {
		t.Fatalf("error = %q, mau invalid_payload", got)
	}
}

func TestCallbackRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			"event_id kosong",
			`{"event_id":"","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"event_id format salah",
			`{"event_id":"bukan-evt","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"device_id tidak cocok dengan yang terautentikasi",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_LAIN","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"source tidak dikenal",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"dana","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"package_name kosong",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"posted_at nol",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":0},"amount_hint":null,"received_at":"2026-09-10T19:30:33+07:00"}`,
		},
		{
			"received_at bukan RFC3339",
			`{"event_id":"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c","device_id":"dev_01ABC","source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,"received_at":"10 Sep 2026"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newAPIWithDevice(t)
			rec := postCallback(t, h, tc.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
			}
			if got := errorCode(t, rec); got != "invalid_payload" {
				t.Fatalf("error = %q, mau invalid_payload", got)
			}
		})
	}
}

func TestCallbackAcceptsNullOptionalFields(t *testing.T) {
	h := newAPIWithDevice(t)

	body := `{"event_id":"evt_00000000000000000000000000000001","device_id":"dev_01ABC",` +
		`"source":"gopay","notification":{"package_name":"com.gojek.gopay","title":null,` +
		`"text":null,"big_text":null,"posted_at":1789051832829},"amount_hint":null,` +
		`"received_at":"2026-09-10T19:30:33+07:00"}`

	rec := postCallback(t, h, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := callbackStatus(t, rec); got != "accepted" {
		t.Fatalf("status = %q, mau accepted", got)
	}
}

func TestCallbackStoresRawPayloadVerbatim(t *testing.T) {
	h := newAPIWithDevice(t)
	postCallback(t, h, validBody)

	s := openTestStore(t)
	var raw string
	err := s.Pool().QueryRow(contextTODO(),
		"SELECT raw_payload::text FROM notification_events WHERE event_id = $1",
		"evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c").Scan(&raw)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !json.Valid([]byte(raw)) {
		t.Fatal("raw_payload bukan JSON yang sah")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("decode raw_payload: %v", err)
	}
	if got["event_id"] != "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c" {
		t.Fatalf("raw_payload tidak memuat event_id asli: %v", got)
	}
}
```

- [ ] **Step 2: Tambahkan dua helper yang dipakai test di atas — `backend/internal/httpapi/callback_test.go`**

Sisipkan di akhir berkas test:

```go
func contextTODO() context.Context { return context.Background() }

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}
```

Tambahkan `"context"`, `"os"`, dan `"github.com/akbarryyan/gopay-notifications/backend/internal/store"` ke blok import berkas test ini.

- [ ] **Step 3: Jalankan test, pastikan gagal**

Run: `cd backend && make test`
Expected: FAIL — rute `POST /api/v1/callback/gopay` belum ada, status 404.

- [ ] **Step 4: Tulis `backend/internal/httpapi/callback.go`**

```go
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// eventIDPattern mengikat bentuk event_id ke formula di api-contract.md §4.2.
var eventIDPattern = regexp.MustCompile(`^evt_[0-9a-f]{32}$`)

type callbackRequest struct {
	EventID      string `json:"event_id"`
	DeviceID     string `json:"device_id"`
	Source       string `json:"source"`
	Notification struct {
		PackageName string  `json:"package_name"`
		Title       *string `json:"title"`
		Text        *string `json:"text"`
		BigText     *string `json:"big_text"`
		PostedAt    int64   `json:"posted_at"`
	} `json:"notification"`
	AmountHint *int64 `json:"amount_hint"`
	ReceivedAt string `json:"received_at"`
}

type callbackResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

func (a *API) handleCallback(w http.ResponseWriter, r *http.Request) {
	device, ok := DeviceFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "device tidak ada di context")
		return
	}
	raw, ok := RawBodyFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "body tidak ada di context")
		return
	}

	var req callbackRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}

	if !eventIDPattern.MatchString(req.EventID) {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"event_id harus berbentuk evt_ diikuti 32 karakter heksadesimal")
		return
	}
	if req.DeviceID != device.DeviceID {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"device_id di body tidak sama dengan device yang terautentikasi")
		return
	}
	if req.Source != "gopay" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "source tidak dikenal")
		return
	}
	if req.Notification.PackageName == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "package_name wajib diisi")
		return
	}
	if req.Notification.PostedAt <= 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"posted_at wajib Unix epoch milidetik yang positif")
		return
	}

	receivedAt, err := time.Parse(time.RFC3339, req.ReceivedAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"received_at wajib RFC3339 dengan offset zona waktu")
		return
	}

	inserted, err := a.store.InsertEvent(r.Context(), store.Event{
		EventID:     req.EventID,
		DeviceID:    device.DeviceID,
		Source:      req.Source,
		PackageName: req.Notification.PackageName,
		Title:       req.Notification.Title,
		BodyText:    req.Notification.Text,
		BigText:     req.Notification.BigText,
		AmountHint:  req.AmountHint,
		PostedAt:    time.UnixMilli(req.Notification.PostedAt),
		ReceivedAt:  receivedAt,
		RawPayload:  raw,
	})
	if err != nil {
		slog.Error("simpan event gagal", "event_id", req.EventID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	status := "duplicate"
	if inserted {
		status = "accepted"
	}
	slog.Info("event diterima", "event_id", req.EventID, "device_id", device.DeviceID, "status", status)

	writeJSON(w, http.StatusOK, callbackResponse{
		Success: true,
		EventID: req.EventID,
		Status:  status,
	})
}
```

`raw` disimpan langsung sebagai `raw_payload`, bukan hasil re-encode — sesuai [spec §4.4](../specs/2026-09-10-ingestion-and-android-bridge-design.md), bentuk asli notifikasi akan dibutuhkan saat sub-project 3 menyusun aturan matching.

- [ ] **Step 5: Daftarkan rute — `backend/internal/httpapi/api.go`**

Tambahkan satu baris di `Handler`, setelah rute `device/me`:

```go
	mux.Handle("POST /api/v1/callback/gopay", a.requireDevice(http.HandlerFunc(a.handleCallback)))
```

- [ ] **Step 6: Jalankan test, pastikan lulus**

Run: `cd backend && make test`
Expected: PASS — termasuk tujuh subtest `TestCallbackRejectsInvalidFields`.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/httpapi/
git commit -m "feat(backend): endpoint callback dengan validasi dan idempotency"
```

---

### Task 9: Endpoint GET /events

**Files:**
- Create: `backend/internal/httpapi/events.go`
- Modify: `backend/internal/httpapi/api.go` — daftarkan rute
- Test: `backend/internal/httpapi/events_test.go`

**Interfaces:**
- Consumes: `(*store.Store).ListEvents`.
- Produces: rute `GET /api/v1/events`, dilindungi basic auth di Caddy (bukan HMAC).

- [ ] **Step 1: Tulis test yang gagal — `backend/internal/httpapi/events_test.go`**

```go
package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type eventsResponse struct {
	Events []struct {
		EventID    string  `json:"event_id"`
		Title      *string `json:"title"`
		Text       *string `json:"text"`
		AmountHint *int64  `json:"amount_hint"`
	} `json:"events"`
}

func getEvents(t *testing.T, h http.Handler, query string) eventsResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events"+query, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var out eventsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestEventsEmptyReturnsEmptyArray(t *testing.T) {
	h := newAPIWithDevice(t)

	got := getEvents(t, h, "")

	if got.Events == nil {
		t.Fatal("events = null, mau array kosong — klien tidak boleh dipaksa menangani null")
	}
	if len(got.Events) != 0 {
		t.Fatalf("len = %d, mau 0", len(got.Events))
	}
}

func TestEventsReturnsStoredEvent(t *testing.T) {
	h := newAPIWithDevice(t)
	postCallback(t, h, validBody)

	got := getEvents(t, h, "")

	if len(got.Events) != 1 {
		t.Fatalf("len = %d, mau 1", len(got.Events))
	}
	e := got.Events[0]
	if e.EventID != "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c" {
		t.Fatalf("event_id = %s", e.EventID)
	}
	if e.Title == nil || *e.Title != "Transfer masuk" {
		t.Fatalf("title = %v, mau \"Transfer masuk\"", e.Title)
	}
	if e.AmountHint == nil || *e.AmountHint != 1 {
		t.Fatalf("amount_hint = %v, mau 1", e.AmountHint)
	}
}

func TestEventsRejectsBadLimit(t *testing.T) {
	h := newAPIWithDevice(t)

	for _, q := range []string{"?limit=0", "?limit=-1", "?limit=abc", "?limit=1001", "?offset=-1"} {
		t.Run(q, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events"+q, nil))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, mau 400", rec.Code)
			}
		})
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `cd backend && make test`
Expected: FAIL — status 404 karena rute belum ada.

- [ ] **Step 3: Tulis `backend/internal/httpapi/events.go`**

```go
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type eventJSON struct {
	EventID     string          `json:"event_id"`
	DeviceID    string          `json:"device_id"`
	Source      string          `json:"source"`
	PackageName string          `json:"package_name"`
	Title       *string         `json:"title"`
	Text        *string         `json:"text"`
	BigText     *string         `json:"big_text"`
	AmountHint  *int64          `json:"amount_hint"`
	PostedAt    string          `json:"posted_at"`
	ReceivedAt  string          `json:"received_at"`
	RawPayload  json.RawMessage `json:"raw_payload"`
}

type eventsListResponse struct {
	Events []eventJSON `json:"events"`
}

// handleEvents dipakai untuk verifikasi manual: dibuka di browser lewat Caddy
// yang melindunginya dengan basic auth. Bukan halaman admin.
func (a *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit, err := intParam(r, "limit", 50)
	if err != nil || limit < 1 || limit > 1000 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "limit harus bilangan bulat 1..1000")
		return
	}
	offset, err := intParam(r, "offset", 0)
	if err != nil || offset < 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "offset harus bilangan bulat >= 0")
		return
	}

	events, err := a.store.ListEvents(r.Context(), limit, offset)
	if err != nil {
		slog.Error("ambil events gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	// Selalu array, tidak pernah null.
	out := make([]eventJSON, 0, len(events))
	for _, e := range events {
		out = append(out, eventJSON{
			EventID:     e.EventID,
			DeviceID:    e.DeviceID,
			Source:      e.Source,
			PackageName: e.PackageName,
			Title:       e.Title,
			Text:        e.BodyText,
			BigText:     e.BigText,
			AmountHint:  e.AmountHint,
			PostedAt:    e.PostedAt.Format(time.RFC3339),
			ReceivedAt:  e.ReceivedAt.Format(time.RFC3339),
			RawPayload:  json.RawMessage(e.RawPayload),
		})
	}

	writeJSON(w, http.StatusOK, eventsListResponse{Events: out})
}

func intParam(r *http.Request, name string, def int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def, nil
	}
	return strconv.Atoi(raw)
}
```

- [ ] **Step 4: Daftarkan rute — `backend/internal/httpapi/api.go`**

```go
	mux.HandleFunc("GET /api/v1/events", a.handleEvents)
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

Run: `cd backend && make test`
Expected: PASS — termasuk lima subtest parameter tidak sah.

- [ ] **Step 6: Jalankan seluruh test dengan race detector**

Run: `cd backend && TEST_DATABASE_URL='postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable' go test ./... -race -count=1 -p 1`
Expected: PASS, tanpa laporan race.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/httpapi/
git commit -m "feat(backend): endpoint /events untuk verifikasi manual"
```

---

### Task 10: Deploy ke VPS

**Files:**
- Create: `backend/Caddyfile`
- Create: `backend/deploy/gopay-ingestion.service`
- Create: `backend/deploy/README.md`
- Create: `backend/.env.example`

**Interfaces:**
- Consumes: binary `cmd/server`, `config.Load`.
- Produces: layanan berjalan di VPS dengan HTTPS.

- [ ] **Step 1: Tulis `backend/.env.example`**

```bash
# Salin ke .env di VPS lalu isi. Jangan pernah commit .env yang sudah terisi.
DATABASE_URL=postgres://gopay:GANTI_PASSWORD@localhost:5432/gopay?sslmode=disable
LISTEN_ADDR=127.0.0.1:8080

# Hasilkan dengan: go run ./cmd/devicetool -genkey
# Mengganti nilai ini membuat seluruh secret device yang tersimpan tidak dapat didekripsi.
DEVICE_SECRET_KEY=
```

- [ ] **Step 2: Tulis `backend/Caddyfile`**

```caddyfile
gopay.example.com {
	encode zstd gzip

	# /events dilindungi basic auth — ini yang dibuka di browser.
	# Hash dibuat dengan: caddy hash-password
	@events path /api/v1/events*
	basic_auth @events {
		admin GANTI_DENGAN_HASH_BCRYPT
	}

	reverse_proxy 127.0.0.1:8080

	log {
		output file /var/log/caddy/gopay.log
		# Header autentikasi tidak boleh masuk log.
		format json
	}
}
```

- [ ] **Step 3: Tulis `backend/deploy/gopay-ingestion.service`**

```ini
[Unit]
Description=GoPay notification ingestion
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=gopay
Group=gopay
WorkingDirectory=/opt/gopay-ingestion
EnvironmentFile=/opt/gopay-ingestion/.env
ExecStart=/opt/gopay-ingestion/server
Restart=always
RestartSec=5

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/gopay-ingestion

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 4: Tulis `backend/deploy/README.md`**

```markdown
# Deploy ke VPS

## Sekali di awal

1. Pasang PostgreSQL 16 dan Caddy.
2. Buat user sistem dan direktori:

   ```bash
   sudo useradd --system --home /opt/gopay-ingestion --shell /usr/sbin/nologin gopay
   sudo mkdir -p /opt/gopay-ingestion
   sudo chown gopay:gopay /opt/gopay-ingestion
   ```

3. Buat database dan user Postgres:

   ```bash
   sudo -u postgres createuser gopay --pwprompt
   sudo -u postgres createdb gopay --owner gopay
   ```

4. Salin `.env.example` ke `/opt/gopay-ingestion/.env`, isi seluruh nilainya.
   `DEVICE_SECRET_KEY` dihasilkan dengan `go run ./cmd/devicetool -genkey`.

   ```bash
   sudo chmod 600 /opt/gopay-ingestion/.env
   sudo chown gopay:gopay /opt/gopay-ingestion/.env
   ```

5. Pasang unit systemd:

   ```bash
   sudo cp deploy/gopay-ingestion.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable gopay-ingestion
   ```

6. Isi `Caddyfile` dengan domain sungguhan dan hash basic auth
   (`caddy hash-password`), lalu salin ke `/etc/caddy/Caddyfile` dan
   `sudo systemctl reload caddy`.

## Tiap rilis

```bash
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
scp server VPS:/tmp/server
ssh VPS 'sudo systemctl stop gopay-ingestion \
  && sudo mv /tmp/server /opt/gopay-ingestion/server \
  && sudo chown gopay:gopay /opt/gopay-ingestion/server \
  && sudo chmod 755 /opt/gopay-ingestion/server \
  && sudo systemctl start gopay-ingestion'
```

Migrasi dijalankan terpisah:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

## Membuat device untuk HP

Di VPS:

```bash
cd /opt/gopay-ingestion
sudo -u gopay env $(cat .env | xargs) ./devicetool -name "HP GoPay Utama"
```

Salin `Device ID` dan `Device Secret` ke Settings aplikasi Android.
Secret tidak akan ditampilkan lagi.
```

- [ ] **Step 5: Deploy dan verifikasi HTTPS**

```bash
curl -s https://<domain>/api/v1/health
```

Expected: `{"status":"ok","server_time":<angka>}` lewat HTTPS dengan sertifikat sah.

- [ ] **Step 6: Verifikasi basic auth pada /events**

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://<domain>/api/v1/events
curl -s -o /dev/null -w '%{http_code}\n' -u admin:<password> https://<domain>/api/v1/events
```

Expected: `401` lalu `200`.

- [ ] **Step 7: Verifikasi HTTP dialihkan ke HTTPS**

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://<domain>/api/v1/health
```

Expected: `308` — Caddy mengalihkan otomatis, tidak ada layanan yang melayani HTTP polos.

- [ ] **Step 8: Buat device sungguhan untuk HP**

Jalankan `devicetool` di VPS sesuai `deploy/README.md`, simpan `Device ID` dan `Device Secret` untuk dipakai plan Android.

- [ ] **Step 9: Commit**

```bash
git add backend/Caddyfile backend/deploy/ backend/.env.example
git commit -m "feat(backend): berkas deploy VPS dengan Caddy dan systemd"
```

- [ ] **Step 10: Jalankan QA milestone M1**

Ikuti [`docs/qa/qa-rules.md`](../../qa/qa-rules.md) dan perbarui [`docs/qa/qa-report.md`](../../qa/qa-report.md).

Butir yang harus berubah status pada siklus ini:

| Butir | Bukti yang diharapkan |
|---|---|
| FR-06 Authentication | Output `go test ./internal/auth/ -v` dan test middleware |
| FR-08 Duplicate prevention | Output `TestInsertEventConcurrentSameIDInsertsOnce -count=20` |
| Backend mengidentifikasi perangkat pengirim | Output test `/device/me` |
| HTTPS untuk production | Output `curl` Step 5 dan Step 7 |
| Device ID tersedia | Output `devicetool` |
| Credential tidak hardcoded | `grep -rn 'DEVICE_SECRET_KEY\|secret' backend --include='*.go'` menunjukkan hanya pembacaan env |
| `hmac.Equal` dipakai | `grep -n 'hmac.Equal' backend/internal/auth/hmac.go` |
| Body diverifikasi mentah sebelum decode | Kutipan `auth_middleware.go` + test body dimodifikasi |
| Idempotency memakai constraint database | `grep -n 'ON CONFLICT' backend/internal/store/event.go` |
| Build production menolak HTTP polos | Output `curl` Step 7 |
| Tabel kode → tindakan, baris `200 accepted`, `200 duplicate`, `400`, `401 invalid_signature`, `401 clock_skew`, `403` | Output test callback dan middleware |

Baris `429`, `5xx`, dan `timeout` tetap `PENDING` — perilakunya ada di sisi Android, bukan backend.

- [ ] **Step 11: Commit laporan QA**

```bash
git add docs/qa/qa-report.md
git commit -m "docs(qa): laporan QA milestone M1"
```

---

## Self-Review

**Spec coverage.** Seluruh bagian spec yang menyangkut backend tercakup: §2.5 auth HMAC (Task 3, 7), §3.2 tumpukan Go (Task 1), §4.4 tabel backend dan idempotency (Task 4, 5), §5 kontrak API (Task 6, 7, 8, 9), §7 strategi pengujian Go (Task 3, 5, 7, 8). Aturan arah transaksi (§2.3) sengaja di luar cakupan — ia berada di sub-project 3, dan plan ini hanya menerima serta menyimpan.

**Placeholder scan.** Tidak ada TBD, TODO, atau "tambahkan error handling yang sesuai". Setiap langkah kode memuat kode yang benar-benar dapat dijalankan. Nilai yang memang harus diganti manusia — domain, password, hash bcrypt — ditulis eksplisit sebagai `GANTI_...` di berkas deploy, bukan sebagai instruksi kabur.

**Type consistency.** `store.Event.BodyText` konsisten dipakai di Task 5, 8, dan 9; kolom SQL-nya `body_text`; field JSON-nya `text` sesuai kontrak. `auth.SigningString` dipanggil dengan urutan argumen sama di Task 3, 7, dan test Task 8. `httpapi.New(store, encKey, now)` konsisten di Task 6, 7, dan `cmd/server`. `encKey()` didefinisikan dua kali di paket berbeda (`store_test` dan `httpapi_test`) — sengaja, karena keduanya paket test terpisah.
