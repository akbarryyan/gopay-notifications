# Landing Page & Self-Service Signup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tambahkan halaman landing publik (`/`) dan form registrasi swalayan (`/register`) ke `dashboard/`, didukung endpoint backend baru `POST /api/v1/signup` yang membuat account Starter trial 3 hari langsung aktif tanpa campur tangan vendor.

**Architecture:** Backend dapat satu endpoint publik baru (tanpa middleware auth apa pun selain rate limit per IP) yang memakai ulang `store.CreateAccount` (diperluas dengan sentinel error email/username bentrok) dan pola set-cookie yang sama dengan login biasa (auto-login). Dashboard customer (`dashboard/`) dapat dua route publik baru di luar route group `(dashboard)`, dan `Overview` (yang sekarang di `/`) pindah ke `/overview` supaya `/` bisa jadi landing page.

**Tech Stack:** Go (backend, pola sama seluruh proyek ini), Next.js 16 App Router + Tailwind v4 + shadcn/ui (dashboard/).

## Global Constraints

- Spec: `docs/superpowers/specs/2026-09-13-landing-signup-design.md`.
- `POST /api/v1/signup` adalah SATU-SATUNYA endpoint di seluruh backend yang tidak butuh sesi/API key/HMAC — hanya rate limit per IP.
- Plan trial hasil signup: `Plan="Starter"`, `MaxDevices=3`, `ExpiresAt = now + 3*24*time.Hour`.
- `gofmt -l .` kosong dan `go vet ./...` bersih sebelum tiap commit Go. `npx tsc --noEmit`/`npx eslint .`/`npx next build` bersih sebelum tiap commit di `dashboard/`.
- Commit message bahasa Indonesia, gaya sama seperti commit lain di repo ini (alasan, bukan cuma "apa yang berubah").

---

### Task 1: `store.CreateAccount` mendeteksi email/username bentrok

**Files:**
- Modify: `backend/internal/store/account.go`
- Modify: `backend/internal/store/account_test.go`

**Interfaces:**
- Produces: `store.ErrAccountEmailTaken`, `store.ErrAccountUsernameTaken` (dua `error` sentinel baru) — dipakai Task 2 (`handleSignup`) dan Task 3 (`handleVendorCreateAccount`).

- [ ] **Step 1: Tulis test yang gagal**

Tambahkan di `backend/internal/store/account_test.go`:

```go
func TestCreateAccountEmailBentrokDitolak(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_a", BusinessName: "Toko A", Email: "sama@uji.test",
		Username: "toko_a", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create pertama: %v", err)
	}

	err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_b", BusinessName: "Toko B", Email: "sama@uji.test",
		Username: "toko_b", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != store.ErrAccountEmailTaken {
		t.Fatalf("err = %v, mau ErrAccountEmailTaken", err)
	}
}

func TestCreateAccountUsernameBentrokDitolak(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_a", BusinessName: "Toko A", Email: "a@uji.test",
		Username: "sama_username", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create pertama: %v", err)
	}

	err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_b", BusinessName: "Toko B", Email: "b@uji.test",
		Username: "sama_username", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != store.ErrAccountUsernameTaken {
		t.Fatalf("err = %v, mau ErrAccountUsernameTaken", err)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run:
```bash
cd backend
export TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable"
go test ./internal/store/... -run 'TestCreateAccountEmailBentrok|TestCreateAccountUsernameBentrok' -v
```
Expected: FAIL — `undefined: store.ErrAccountEmailTaken`.

- [ ] **Step 3: Tambahkan sentinel error dan deteksi constraint**

Di `backend/internal/store/account.go`, tambah import `"github.com/jackc/pgx/v5/pgconn"`, tambah dua var dekat `ErrAccountNotFound`:

```go
// ErrAccountEmailTaken/ErrAccountUsernameTaken dikembalikan CreateAccount
// kalau email/username sudah dipakai account lain (constraint UNIQUE di
// migration 00008_accounts.sql, nama constraint otomatis Postgres untuk
// kolom polos: "accounts_email_key"/"accounts_username_key").
var (
	ErrAccountEmailTaken    = errors.New("store: email sudah dipakai")
	ErrAccountUsernameTaken = errors.New("store: username sudah dipakai")
)

func isAccountUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
	}
	return false
}
```

Ubah `CreateAccount` — ganti blok pengecekan error setelah `s.pool.Exec`:

```go
	_, err = s.pool.Exec(ctx,
		`INSERT INTO accounts (id, business_name, email, username, password_hash, plan, max_devices, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		in.ID, in.BusinessName, in.Email, in.Username, string(hash), in.Plan, in.MaxDevices, in.ExpiresAt)
	if isAccountUniqueViolation(err, "accounts_email_key") {
		return ErrAccountEmailTaken
	}
	if isAccountUniqueViolation(err, "accounts_username_key") {
		return ErrAccountUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("store: create account: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `go test ./internal/store/... -run TestCreateAccount -v`
Expected: semua `PASS`, termasuk dua test baru.

- [ ] **Step 5: Jalankan seluruh test paket store, pastikan tidak ada regresi**

Run: `go test ./internal/store/... -v 2>&1 | tail -20`
Expected: `ok`, nol `FAIL`.

- [ ] **Step 6: Commit**

```bash
cd backend
gofmt -l internal/store/
git add internal/store/account.go internal/store/account_test.go
git commit -m "feat(store): CreateAccount mendeteksi email/username bentrok

ErrAccountEmailTaken/ErrAccountUsernameTaken menggantikan error generik
saat constraint UNIQUE accounts_email_key/accounts_username_key
terlanggar -- dipakai endpoint signup publik (Task 2) dan
handleVendorCreateAccount (Task 3) untuk membalas 409 yang benar,
bukan 500.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 2: `POST /api/v1/signup` — endpoint publik baru

**Files:**
- Create: `backend/internal/httpapi/signup.go`
- Create: `backend/internal/httpapi/signup_test.go`
- Modify: `backend/internal/httpapi/api.go`

**Interfaces:**
- Consumes: `store.ErrAccountEmailTaken`/`ErrAccountUsernameTaken` (Task 1), `randomPrefixedAccountID()` (sudah ada di `vendor_accounts.go`), `auth.NewSessionToken` (`internal/auth`), `a.store.LogAudit`.
- Produces: `a.handleSignup(w http.ResponseWriter, r *http.Request)`, field baru `signupThrottle *loginThrottle` di struct `API`.

- [ ] **Step 1: Tulis test yang gagal**

Buat `backend/internal/httpapi/signup_test.go`:

```go
package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func signupReq(t *testing.T, h http.Handler, businessName, email, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"business_name":"` + businessName + `","email":"` + email + `","username":"` + username + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSignupBerhasilMenerbitkanCookieDanAccount(t *testing.T) {
	h := newAPIWithAdmin(t) // pola sudah ada: store kosong, tanpa account manapun dulu

	rec := signupReq(t, h, "Toko Baru", "baru@uji.test", "toko_baru", "password123")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if sessionCookieFrom(rec) == nil {
		t.Fatal("signup berhasil seharusnya langsung menerbitkan cookie admin_session (auto-login)")
	}

	// Login pakai kredensial yang baru saja didaftarkan, buktikan account beneran tersimpan.
	loginRec := adminLogin(t, h, "toko_baru", "password123")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login setelah signup gagal: status = %d", loginRec.Code)
	}
}

func TestSignupEmailBentrokDitolak(t *testing.T) {
	h := newAPIWithAdmin(t) // account "admin" dengan email acc_1@uji.test sudah ada (lihat newAPIWithAdmin)

	rec := signupReq(t, h, "Toko Lain", "acc_1@uji.test", "toko_lain", "password123")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "email_taken" {
		t.Fatalf("error = %q, mau email_taken", body.Error)
	}
}

func TestSignupUsernameBentrokDitolak(t *testing.T) {
	h := newAPIWithAdmin(t) // account "admin" sudah ada (lihat newAPIWithAdmin)

	rec := signupReq(t, h, "Toko Lain", "lain@uji.test", "admin", "password123")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "username_taken" {
		t.Fatalf("error = %q, mau username_taken", body.Error)
	}
}

func TestSignupPasswordPendekDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := signupReq(t, h, "Toko Baru", "baru2@uji.test", "toko_baru2", "pendek")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestSignupFieldKosongDitolak(t *testing.T) {
	h := newAPIWithAdmin(t)

	rec := signupReq(t, h, "", "baru3@uji.test", "toko_baru3", "password123")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestSignupDibatasiSetelahBanyakPercobaan(t *testing.T) {
	h := newAPIWithAdmin(t)

	var rec *httptest.ResponseRecorder
	for i := 0; i < 5; i++ {
		// Password pendek supaya tiap percobaan gagal validasi (400), bukan
		// berhasil bikin account -- yang penting throttle menghitung PER
		// PERCOBAAN, bukan per kegagalan tertentu.
		rec = signupReq(t, h, "Toko X", "x@uji.test", "toko_x", "pendek")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan ke-%d: status = %d, mau 400", i+1, rec.Code)
		}
	}

	rec = signupReq(t, h, "Toko X", "x2@uji.test", "toko_x2", "password123")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, mau 429", rec.Code)
	}
}

func TestSignupTidakButuhCredentialApaPun(t *testing.T) {
	// Membuktikan endpoint ini SENGAJA publik -- request tanpa cookie, tanpa
	// Authorization header, tanpa header device HMAC apa pun, dan tetap
	// berhasil (selama field valid).
	h := newAPIWithAdmin(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signup",
		strings.NewReader(`{"business_name":"Toko Tanpa Auth","email":"tanpaauth@uji.test","username":"tanpa_auth","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 tanpa credential apa pun (body=%s)", rec.Code, rec.Body.String())
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/httpapi/... -run TestSignup -v`
Expected: FAIL — `404 page not found` (route belum terdaftar).

- [ ] **Step 3: Buat `signup.go`**

```go
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type signupRequest struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

const (
	signupTrialDays  = 3
	signupPlan       = "Starter"
	signupMaxDevices = 3
	minPasswordLen   = 8
)

// handleSignup adalah SATU-SATUNYA endpoint di seluruh backend yang tidak
// dibungkus requireAdmin/requireAPIKey/requireDevice/requireVendor apa pun
// -- sengaja publik, calon customer belum punya kredensial apa pun sampai
// titik ini. Cuma dilindungi rate limit per IP (signupThrottle), pola sama
// persis loginThrottle yang sudah dipakai handleAdminLogin/handleVendorLogin.
func (a *API) handleSignup(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.signupThrottle.Allowed(ip, a.now()) {
		a.writeError(w, http.StatusTooManyRequests, "too_many_attempts", "terlalu banyak percobaan, coba lagi nanti")
		return
	}

	var req signupRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}
	if req.BusinessName == "" || req.Email == "" || req.Username == "" {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "business_name, email, dan username wajib diisi")
		return
	}
	if len(req.Password) < minPasswordLen {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "password minimal 8 karakter")
		return
	}

	id, err := randomPrefixedAccountID()
	if err != nil {
		slog.Error("generate account id gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	now := a.now()
	err = a.store.CreateAccount(r.Context(), store.CreateAccountInput{
		ID: id, BusinessName: req.BusinessName, Email: req.Email, Username: req.Username,
		PlaintextPassword: req.Password, Plan: signupPlan, MaxDevices: signupMaxDevices,
		ExpiresAt: now.Add(signupTrialDays * 24 * time.Hour),
	})
	if errors.Is(err, store.ErrAccountEmailTaken) {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusConflict, "email_taken", "email sudah dipakai akun lain")
		return
	}
	if errors.Is(err, store.ErrAccountUsernameTaken) {
		a.signupThrottle.RecordFailure(ip, a.now())
		a.writeError(w, http.StatusConflict, "username_taken", "username sudah dipakai akun lain")
		return
	}
	if err != nil {
		slog.Error("create account (signup) gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	a.signupThrottle.RecordSuccess(ip)

	if err := a.store.LogAudit(r.Context(), "signup", "ACCOUNT_CREATED", id,
		map[string]string{"business_name": req.BusinessName, "plan": signupPlan}); err != nil {
		slog.Error("log audit signup gagal", "err", err)
	}

	// Auto-login: langsung terbitkan cookie sesi, sama persis
	// handleAdminLogin -- customer tidak perlu login manual setelah daftar.
	token := auth.NewSessionToken(a.adminSessionKey, now, id)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.SessionDuration.Seconds()),
	})

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
```

- [ ] **Step 4: Daftarkan route di `api.go`**

Tambah field baru ke struct `API` (dekat `loginThrottle`/`vendorLoginThrottle`):

```go
signupThrottle *loginThrottle
```

Inisialisasi di `New(...)`:

```go
signupThrottle: newLoginThrottle(),
```

Daftarkan route — taruh dekat `POST /api/v1/admin/login` (sama-sama publik):

```go
mux.HandleFunc("POST /api/v1/signup", a.handleSignup)
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

Run: `go test ./internal/httpapi/... -run TestSignup -v`
Expected: semua `PASS`.

- [ ] **Step 6: Jalankan seluruh test backend, pastikan tidak ada regresi**

Run:
```bash
cd backend
go build ./... && go vet ./...
export TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable"
make test 2>&1 | tail -20
```
Expected: build+vet bersih, seluruh test `ok`.

- [ ] **Step 7: Commit**

```bash
cd backend
gofmt -l .
git add internal/httpapi/signup.go internal/httpapi/signup_test.go internal/httpapi/api.go
git commit -m "feat(httpapi): POST /api/v1/signup -- registrasi swalayan publik

Satu-satunya endpoint di seluruh backend tanpa requireAdmin/requireAPIKey/
requireDevice/requireVendor -- calon customer belum punya kredensial apa
pun sampai titik ini. Dilindungi signupThrottle (rate limit per IP, tipe
sama persis loginThrottle). Account yang lahir: plan Starter, max_devices
3, trial 3 hari, langsung auto-login (cookie admin_session diterbitkan
di response yang sama, tidak perlu login manual terpisah).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 3: `handleVendorCreateAccount` pakai sentinel error yang sama

**Files:**
- Modify: `backend/internal/httpapi/vendor_accounts.go`
- Modify: `backend/internal/httpapi/vendor_accounts_test.go`

**Interfaces:**
- Consumes: `store.ErrAccountEmailTaken`/`ErrAccountUsernameTaken` (Task 1).

- [ ] **Step 1: Tulis test yang gagal**

Tambahkan di `backend/internal/httpapi/vendor_accounts_test.go`:

```go
func TestVendorCreateAccountEmailBentrokMengembalikan409(t *testing.T) {
	h := newAPIWithVendor(t)
	cookie := loginAsVendor(t, h)

	body1 := `{"business_name":"Toko A","email":"bentrok@uji.test","username":"toko_a","plan":"Starter","expires_at":"2027-01-01"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.AddCookie(cookie)
	h.ServeHTTP(httptest.NewRecorder(), req1)

	body2 := `{"business_name":"Toko B","email":"bentrok@uji.test","username":"toko_b","plan":"Starter","expires_at":"2027-01-01"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/vendor/accounts", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("status = %d, mau 409 (body=%s)", rec2.Code, rec2.Body.String())
	}
	var out struct {
		Error string `json:"error"`
	}
	json.Unmarshal(rec2.Body.Bytes(), &out)
	if out.Error != "email_taken" {
		t.Fatalf("error = %q, mau email_taken", out.Error)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/httpapi/... -run TestVendorCreateAccountEmailBentrok -v`
Expected: FAIL — status sekarang `500`, bukan `409`.

- [ ] **Step 3: Ubah `handleVendorCreateAccount` di `vendor_accounts.go`**

Cari blok ini:

```go
	err = a.store.CreateAccount(r.Context(), store.CreateAccountInput{
		ID: id, BusinessName: req.BusinessName, Email: req.Email, Username: req.Username,
		PlaintextPassword: password, Plan: req.Plan, MaxDevices: maxDevices, ExpiresAt: expiresAt,
	})
	if err != nil {
		slog.Error("create account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
```

Ganti jadi:

```go
	err = a.store.CreateAccount(r.Context(), store.CreateAccountInput{
		ID: id, BusinessName: req.BusinessName, Email: req.Email, Username: req.Username,
		PlaintextPassword: password, Plan: req.Plan, MaxDevices: maxDevices, ExpiresAt: expiresAt,
	})
	if errors.Is(err, store.ErrAccountEmailTaken) {
		a.writeError(w, http.StatusConflict, "email_taken", "email sudah dipakai akun lain")
		return
	}
	if errors.Is(err, store.ErrAccountUsernameTaken) {
		a.writeError(w, http.StatusConflict, "username_taken", "username sudah dipakai akun lain")
		return
	}
	if err != nil {
		slog.Error("create account gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
```

Tambahkan `"errors"` ke import `vendor_accounts.go` kalau belum ada (cek dulu — `errors.Is` sudah dipakai di fungsi lain di file yang sama, kemungkinan besar sudah ter-import).

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `go test ./internal/httpapi/... -run TestVendor -v`
Expected: semua `PASS`.

- [ ] **Step 5: Commit**

```bash
cd backend
gofmt -l internal/httpapi/
git add internal/httpapi/vendor_accounts.go internal/httpapi/vendor_accounts_test.go
git commit -m "fix(httpapi): handleVendorCreateAccount balas 409 saat email/username bentrok

Sebelumnya cuma menghasilkan 500 generik -- sekarang pakai sentinel error
yang sama dengan endpoint signup publik (Task 1-2).

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 4: `dashboard/` — restrukturisasi routing (Overview → `/overview`, `/` & `/register` publik)

**Files:**
- Create: `dashboard/src/app/(dashboard)/overview/page.tsx` (isi dipindah dari file lama)
- Delete: `dashboard/src/app/(dashboard)/page.tsx`
- Modify: `dashboard/src/components/dashboard/nav-items.ts`
- Modify: `dashboard/src/app/login/page.tsx`
- Modify: `dashboard/src/proxy.ts`

**Interfaces:**
- Tidak ada interface baru — task ini murni pemindahan route dan penyesuaian redirect, tidak mengubah signature fungsi apa pun.

- [ ] **Step 1: Pindahkan halaman Overview**

```bash
cd dashboard
mkdir -p "src/app/(dashboard)/overview"
git mv "src/app/(dashboard)/page.tsx" "src/app/(dashboard)/overview/page.tsx"
```

- [ ] **Step 2: Ubah `nav-items.ts`**

Ganti baris:

```ts
{ label: "Overview", href: "/", icon: LayoutDashboard },
```

jadi:

```ts
{ label: "Overview", href: "/overview", icon: LayoutDashboard },
```

- [ ] **Step 3: Ubah redirect setelah login di `login/page.tsx`**

Cari `router.push("/");` (setelah `await login(username, password);`), ganti jadi `router.push("/overview");`.

- [ ] **Step 4: Ubah `proxy.ts` — `/` dan `/register` jadi publik**

Ganti isi `src/proxy.ts`:

```ts
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * Gerbang navigasi, bukan gerbang keamanan.
 *
 * Ini hanya memeriksa APAKAH cookie sesi ada, bukan apakah ia masih valid —
 * itu tidak bisa diverifikasi di sini tanpa memanggil backend pada setiap
 * navigasi. Backend tetap satu-satunya pihak yang memutuskan sah tidaknya
 * sesi lewat requireAdmin pada setiap panggilan API; ini murni mencegah
 * kedipan halaman kosong sebelum redirect ke /login.
 *
 * "/" dan "/register" SENGAJA publik (landing page + form signup) --
 * berbeda dari seluruh path lain di sini yang wajib sesi.
 */
const PUBLIC_PATHS = new Set(["/", "/register", "/login"]);

export function proxy(request: NextRequest) {
  const hasSession = request.cookies.has("admin_session");
  const { pathname } = request.nextUrl;

  if (pathname === "/login") {
    if (hasSession) {
      return NextResponse.redirect(new URL("/overview", request.url));
    }
    return NextResponse.next();
  }

  if (PUBLIC_PATHS.has(pathname)) {
    return NextResponse.next();
  }

  if (!hasSession) {
    const loginUrl = new URL("/login", request.url);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Semua path kecuali aset statis Next.js, favicon, dan API (backend
    // sendiri yang menegakkan auth untuk /api/*).
    "/((?!api|_next/static|_next/image|favicon.ico).*)",
  ],
};
```

- [ ] **Step 5: Verifikasi build (belum ada halaman `/` dan `/register` baru — itu Task 6-7, tapi restrukturisasi ini sendiri harus tetap valid)**

Run:
```bash
cd dashboard
rm -rf .next && npx next typegen
npx tsc --noEmit
npx eslint .
```
Expected: 0 error. (`npx next build` BELUM dijalankan di task ini — tanpa halaman `/` dan `/register`, root route `/` akan 404 dulu sampai Task 7-8 selesai; itu wajar, bukan regresi.)

- [ ] **Step 6: Commit**

```bash
cd dashboard
git add "src/app/(dashboard)/overview/page.tsx" src/components/dashboard/nav-items.ts src/app/login/page.tsx src/proxy.ts
git rm "src/app/(dashboard)/page.tsx" 2>/dev/null || true
git commit -m "feat(dashboard): Overview pindah ke /overview, / dan /register jadi publik

Menyiapkan / untuk jadi landing page (Task 8) dan /register untuk form
signup (Task 7) -- keduanya dikecualikan dari gerbang sesi di proxy.ts.
nav-items.ts dan redirect setelah login ikut diperbarui ke /overview.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 5: `dashboard/src/lib/api.ts` — fungsi `signup()`

**Files:**
- Modify: `dashboard/src/lib/api.ts`

**Interfaces:**
- Produces: `signup(input: {business_name, email, username, password}): Promise<{success: true}>` — dipakai Task 7 (`/register`).

- [ ] **Step 1: Tambahkan fungsi**

Di bagian `// --- Auth ---` (dekat `login`/`logout`):

```ts
export function signup(input: {
  business_name: string;
  email: string;
  username: string;
  password: string;
}): Promise<{ success: true }> {
  return apiFetch("/api/v1/signup", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
```

- [ ] **Step 2: Verifikasi**

Run: `cd dashboard && npx tsc --noEmit`
Expected: 0 error.

- [ ] **Step 3: Commit**

```bash
cd dashboard
git add src/lib/api.ts
git commit -m "feat(dashboard): fungsi signup() di lib/api.ts

Dipakai halaman /register (Task 7) -- POST /api/v1/signup.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 6: Halaman `/register`

**Files:**
- Create: `dashboard/src/app/register/page.tsx`

**Interfaces:**
- Consumes: `signup()` (Task 5), pola form dari `dashboard/src/app/login/page.tsx` (Card shadcn, `ApiError` handling).

- [ ] **Step 1: Buat halaman**

```tsx
"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ApiError, signup } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const [businessName, setBusinessName] = useState("");
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await signup({
        business_name: businessName,
        email,
        username,
        password,
      });
      // Cookie sesi sudah diterapkan browser sebelum baris ini jalan
      // (signup() melakukan auto-login di respons yang sama).
      router.push("/overview");
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "email_taken") {
          setError("Email ini sudah dipakai akun lain.");
        } else if (err.code === "username_taken") {
          setError("Username ini sudah dipakai akun lain.");
        } else if (err.code === "too_many_attempts") {
          setError("Terlalu banyak percobaan. Coba lagi dalam beberapa menit.");
        } else {
          setError(err.message || "Pendaftaran gagal, periksa kembali isian kamu.");
        }
      } else {
        setError("Tidak dapat menghubungi server. Periksa koneksi Anda.");
      }
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-secondary/40 px-4 py-10">
      <Card className="w-full max-w-sm rounded-2xl border-none py-8 shadow-sm ring-1 ring-border/60">
        <CardHeader className="items-center text-center">
          <span className="mb-2 flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <Zap className="size-5" fill="currentColor" strokeWidth={0} />
          </span>
          <CardTitle className="text-xl">Daftar Gratis 3 Hari</CardTitle>
          <CardDescription>Tanpa kartu kredit, langsung aktif.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="business-name">Nama bisnis</Label>
              <Input
                id="business-name"
                value={businessName}
                onChange={(e) => setBusinessName(e.target.value)}
                placeholder="Toko Contoh"
                autoFocus
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="kamu@contoh.test"
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="toko_contoh"
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                minLength={8}
                required
              />
            </div>
            <Button type="submit" disabled={loading} className="mt-2">
              {loading ? "Memproses..." : "Daftar"}
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              Sudah punya akun?{" "}
              <Link href="/login" className="font-medium text-foreground hover:underline">
                Masuk
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
```

- [ ] **Step 2: Verifikasi**

Run:
```bash
cd dashboard
npx next typegen
npx tsc --noEmit
npx eslint .
```
Expected: 0 error.

- [ ] **Step 3: Commit**

```bash
cd dashboard
git add src/app/register/page.tsx
git commit -m "feat(dashboard): halaman /register -- form signup swalayan

Pola sama persis /login (Card shadcn, ApiError handling). Sukses ->
langsung /overview (cookie sudah ter-set oleh signup(), auto-login).
email_taken/username_taken ditampilkan sebagai pesan spesifik -- beda
dari login, di sini tidak ada alasan keamanan menyamakan pesan error.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 7: Halaman `/` (landing page)

**Files:**
- Create: `dashboard/src/app/page.tsx`

**Interfaces:**
- Tidak konsumsi API apa pun — halaman statis murni, form/CTA-nya cuma `<Link>` ke `/register`/`/login`.

- [ ] **Step 1: Buat halaman**

```tsx
import Link from "next/link";
import { Zap, Smartphone, Repeat, Webhook } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const FEATURES = [
  {
    icon: Repeat,
    title: "Nominal unik",
    desc: "Tiap invoice dapat nominal yang sedikit berbeda -- tidak ada lagi tebak-tebak siapa yang bayar berapa.",
  },
  {
    icon: Webhook,
    title: "Webhook otomatis",
    desc: "invoice.paid dan invoice.expired terkirim otomatis ke website kamu, lengkap dengan retry kalau gagal.",
  },
  {
    icon: Smartphone,
    title: "Dashboard realtime",
    desc: "Overview, Devices, Events, Transactions -- semua ter-update otomatis begitu notifikasi GoPay masuk.",
  },
  {
    icon: Zap,
    title: "Konsol pengecualian",
    desc: "Transaksi yang nominalnya tidak cocok otomatis, tidak hilang begitu saja -- tetap kelihatan dan bisa dicocokkan manual.",
  },
];

const STEPS = [
  "Pasang aplikasi Android di HP yang menerima notifikasi GoPay Merchant.",
  "Notifikasi pembayaran otomatis tercatat dan dikirim ke sistem kami.",
  "Invoice otomatis lunas begitu nominal cocok, webhook langsung terkirim ke website kamu.",
];

export default function LandingPage() {
  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex items-center justify-between px-6 py-4">
        <span className="flex items-center gap-2 font-semibold">
          <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Zap className="size-4" fill="currentColor" strokeWidth={0} />
          </span>
          Payment Bridge
        </span>
        <nav className="flex items-center gap-2">
          <Link href="/login" className={buttonVariants({ variant: "ghost", size: "sm" })}>
            Masuk
          </Link>
          <Link href="/register" className={buttonVariants({ size: "sm" })}>
            Daftar Gratis 3 Hari
          </Link>
        </nav>
      </header>

      <main className="flex-1">
        {/* Hero */}
        <section className="mx-auto flex max-w-2xl flex-col items-center gap-6 px-6 py-20 text-center">
          <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
            Pantau pembayaran GoPay tanpa daftar ke Midtrans
          </h1>
          <p className="text-lg text-muted-foreground">
            Notifikasi GoPay Merchant di HP kamu jadi invoice otomatis lunas
            dan webhook ke website kamu — tanpa integrasi payment gateway
            yang ribet.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Link href="/register" className={buttonVariants({ size: "lg" })}>
              Daftar Gratis 3 Hari
            </Link>
            <Link href="/login" className={buttonVariants({ variant: "outline", size: "lg" })}>
              Masuk
            </Link>
          </div>
        </section>

        {/* Cara kerja */}
        <section className="mx-auto max-w-3xl px-6 py-12">
          <h2 className="mb-8 text-center text-2xl font-semibold">Cara kerja</h2>
          <ol className="flex flex-col gap-6 sm:flex-row">
            {STEPS.map((step, i) => (
              <li key={i} className="flex flex-1 flex-col items-center gap-3 text-center">
                <span className="flex size-9 items-center justify-center rounded-full bg-primary text-sm font-semibold text-primary-foreground">
                  {i + 1}
                </span>
                <p className="text-sm text-muted-foreground">{step}</p>
              </li>
            ))}
          </ol>
        </section>

        {/* Fitur */}
        <section className="mx-auto max-w-4xl px-6 py-12">
          <h2 className="mb-8 text-center text-2xl font-semibold">Fitur</h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {FEATURES.map((f) => (
              <Card key={f.title} className="border-none shadow-sm ring-1 ring-border/60">
                <CardHeader className="flex-row items-center gap-3 space-y-0">
                  <span className="flex size-9 items-center justify-center rounded-lg bg-secondary">
                    <f.icon className="size-4.5" />
                  </span>
                  <CardTitle className="text-base">{f.title}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm text-muted-foreground">{f.desc}</CardContent>
              </Card>
            ))}
          </div>
        </section>

        {/* CTA penutup */}
        <section className="mx-auto flex max-w-2xl flex-col items-center gap-4 px-6 py-20 text-center">
          <h2 className="text-2xl font-semibold">Coba sekarang, gratis 3 hari</h2>
          <Link href="/register" className={buttonVariants({ size: "lg" })}>
            Daftar Gratis 3 Hari
          </Link>
        </section>
      </main>

      <footer className="border-t border-border/60 px-6 py-6 text-center text-sm text-muted-foreground">
        Payment Bridge © {new Date().getFullYear()}
      </footer>
    </div>
  );
}
```

- [ ] **Step 2: Verifikasi lengkap (build penuh sekarang sudah bisa jalan — `/` dan `/register` sudah ada)**

Run:
```bash
cd dashboard
rm -rf .next && npx next typegen
npx tsc --noEmit
npx eslint .
npx next build
```
Expected: 0 error di ketiganya, `next build` menampilkan route `/`, `/register`, `/login`, `/overview`, dan seluruh route dashboard lama.

- [ ] **Step 3: Commit**

```bash
cd dashboard
git add src/app/page.tsx
git commit -m "feat(dashboard): halaman / -- landing page publik

5 section: hero, cara kerja (3 langkah), fitur (cuma yang sungguhan ada
di produk ini -- nominal unik, webhook, dashboard realtime, konsol
pengecualian), CTA penutup, footer. Tidak ada section harga/testimoni --
belum ada model harga pasca-trial atau customer nyata untuk dikutip.

npx tsc --noEmit, npx eslint ., npx next build: 0 error, seluruh route
(termasuk / dan /register yang baru) build bersih.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 8: Verifikasi akhir + `CLAUDE.md`

**Files:**
- Modify: `CLAUDE.md`

**Interfaces:**
- Tidak ada — task dokumentasi penutup.

- [ ] **Step 1: Jalankan seluruh verifikasi dari nol**

```bash
cd backend
go build ./... && go vet ./... && gofmt -l .
export TEST_DATABASE_URL="postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable"
make test 2>&1 | tail -20

cd ../dashboard
rm -rf .next && npx next typegen
npx tsc --noEmit && npx eslint . && npx next build
```
Expected: semuanya bersih. Tempel jumlah test yang lulus.

- [ ] **Step 2: Tambahkan catatan singkat di `CLAUDE.md`**

Di bagian "### Dashboard" (dekat baris yang menyebut halaman-halaman yang datanya sungguhan ada), tambahkan satu paragraf:

```markdown
Sejak sub-project #2+#6 (spec
docs/superpowers/specs/2026-09-13-landing-signup-design.md): `/` adalah
landing page publik dan `/register` form signup swalayan (plan Starter,
trial 3 hari, langsung aktif tanpa campur tangan vendor) -- keduanya di
luar route group `(dashboard)`, dikecualikan dari gerbang sesi di
`proxy.ts`. Overview yang dulu di `/` sekarang di `/overview`.
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: catat landing page + signup swalayan di CLAUDE.md

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Self-Review Notes

- **Cakupan spec:** §1 (ruang lingkup) → Task 4, 6, 7. §2 (routing) → Task 4. §3 (endpoint signup) → Task 1-2. §4 (lib/api.ts) → Task 5. §5 (halaman /register) → Task 6. §6 (landing page) → Task 7. §7 (testing) → Task 1-2 Step test, Task 7 Step 2 (frontend, tanpa unit test terpisah sesuai spec). §8 (risiko) — tidak butuh task, sudah jadi keputusan desain yang tercermin di implementasi (trial 3 hari, rate limit).
- **Konsistensi tipe:** `store.ErrAccountEmailTaken`/`ErrAccountUsernameTaken` (Task 1) dipakai identik di Task 2 (`handleSignup`) dan Task 3 (`handleVendorCreateAccount`). `signup()` (Task 5) dipakai identik di Task 6 (`/register`).
