# Sub-project baru — Platform hosted multi-tenant (fase 1: akun & data model)

## 0. Latar belakang dan mengapa ini sub-project baru

Produk ini awalnya dibangun dengan model **self-hosted + lisensi tahunan**:
tiap customer men-deploy backend dan dashboard-nya sendiri di server mereka
sendiri, vendor tidak pernah menyentuh data/infrastruktur mereka. Berdasarkan
model itu, sub-project 5 (`docs/superpowers/specs/2026-09-13-online-license-platform-design.md`)
membangun License Server + `internal/licenseclient` sebagai otoritas pusat
yang divalidasi tiap instalasi customer secara berkala lewat REST API.

Keputusan bisnis berubah: produk ini sekarang jadi **hosted SaaS multi-tenant**,
seperti Midtrans. Customer tidak lagi men-deploy apa pun — mereka daftar,
dapat akun + API key, integrasi lewat REST API ke satu backend yang dihost
vendor, dan login ke satu Customer Dashboard terpusat. Ini mengubah fondasi
arsitektur, bukan cuma menambah fitur, sehingga dipecah jadi beberapa
sub-project sendiri:

1. **Data model multi-tenant & akun (dokumen ini)** — fondasi, semua yang
   lain menunggu ini.
2. Akun customer: signup/login (alur publik, di luar cakupan dokumen ini —
   dokumen ini hanya menyediakan mekanisme pembuatan akun lewat Vendor
   Dashboard, cukup untuk diuji).
3. Customer Dashboard disesuaikan ke model hosted (sebagian besar sudah
   cocok karena sudah berbasis sesi + REST, tinggal menyesuaikan makna
   `account_id` di baliknya).
4. Penyesuaian mobile/Android bridge (Backend URL tetap, alur dapat
   Device ID+Secret berubah).
5. Landing page marketing whuzpay.com (tombol Login/Daftar) — sengaja
   paling akhir karena baru jelas setelah #2 kelar.

Dokumen ini **hanya** mencakup #1. Sisanya adalah sub-project terpisah
dengan spec sendiri nanti.

## 1. Keputusan yang sudah dikonfirmasi

- Model self-hosted **dipensiunkan total**, diganti hosted multi-tenant.
- Satu akun = satu login (username/password) untuk MVP — tidak ada
  multi-user per akun dulu.
- Device dan API key dibuat **swalayan dari Customer Dashboard** setelah
  login — tidak ada lagi CLI (`devicetool`) di sisi customer.
- License Server (`backend/cmd/licenseserver`, `backend/internal/licenseserver`,
  `backend/internal/licenseclient`, `backend/migrations-license`) **dibongkar
  total** dan diserap ke backend utama — alasan pemisahannya (banyak
  instalasi tersebar memvalidasi ke satu otoritas) sudah tidak berlaku
  begitu backend jadi satu.
- Isolasi data: **shared tables + kolom `account_id`** (bukan schema-per-tenant)
  — pola SaaS standar, sesuai skala produk ini sekarang.
- Instalasi `whuzpay.com` yang sekarang **menjadi backend hosted itu sendiri**
  — data toko Akbar dimigrasi jadi akun pertama, bukan instalasi terpisah.
- Pembuatan akun untuk sub-project ini lewat Vendor Dashboard (memperluas
  flow "Buat Customer" yang sudah ada), bukan lewat signup publik (itu #2).

## 2. Skema data

### 2.1 Tabel baru: `accounts`

Menggantikan `admin_users` (login) dan menyerap konsep `customers`+`licenses`
dari License Server (identitas pemilik + plan/kuota). Satu baris = satu
customer = satu login.

```sql
CREATE TABLE accounts (
    id            TEXT PRIMARY KEY,           -- "acc_xxxxxxxx"
    business_name TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    username      TEXT NOT NULL UNIQUE,       -- dipakai login, terpisah dari email
    password_hash TEXT NOT NULL,
    plan          TEXT NOT NULL,              -- "Starter" | "Business" | "Enterprise"
    max_devices   INTEGER NOT NULL,           -- -1 = unlimited (Enterprise)
    admin_status  TEXT NOT NULL DEFAULT 'active', -- "active" | "suspended" | "revoked"
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Status yang dikirim ke Customer Dashboard/API tetap 5 nilai turunan
(`active`/`expiring`/`expired`/`suspended`/`revoked`), dihitung dari
`admin_status` + `expires_at` — persis logika `DerivedStatus` yang sudah ada
di `internal/licenseserver/store/license.go`, dipindah jadi
`(Account) DerivedStatus(now time.Time) string` di `internal/store`.
`WarningThresholdDays = 30` tetap dipakai (sudah ada, tidak berubah).

Karena tidak ada lagi jaringan antar dua service, **tidak ada lagi status
`unreachable` atau grace period** — status dihitung langsung dari baris yang
sama di transaksi yang sama, selalu konsisten.

### 2.2 Kolom `account_id` di tabel yang sudah ada

Ditambahkan ke: `devices`, `notification_events`, `invoices`, `api_keys`,
`webhook_endpoints`, `webhook_deliveries`, `event_reviews`.

```sql
ALTER TABLE devices            ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id);
ALTER TABLE invoices           ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id);
ALTER TABLE api_keys           ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id);
ALTER TABLE webhook_endpoints  ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id);
-- notification_events, webhook_deliveries, event_reviews: account_id diturunkan
-- lewat device_id/endpoint_id/event_id yang sudah ada, TAPI tetap didenormalisasi
-- jadi kolom sendiri supaya query listing tidak perlu JOIN berlapis tiap kali,
-- dan supaya index isolasi (lihat 2.3) bisa dipasang langsung.
ALTER TABLE notification_events ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id);
ALTER TABLE webhook_deliveries  ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id);
ALTER TABLE event_reviews       ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id);
```

Migration menambah kolom sebagai **nullable dulu**, backfill (lihat §5), baru
`SET NOT NULL` di langkah terakhir migration yang sama — supaya migration
tetap satu file yang bisa jalan bersih di database kosong (test) maupun
database lama (produksi).

### 2.3 Constraint unik yang di-scope ulang

| Tabel | Constraint lama | Constraint baru |
|---|---|---|
| `invoices` | `UNIQUE(external_ref)` | `UNIQUE(account_id, external_ref)` |
| `invoices` (partial, nominal PENDING) | `UNIQUE(unique_amount) WHERE status='PENDING'` | `UNIQUE(account_id, unique_amount) WHERE status='PENDING'` |
| `api_keys.key_hash` | unik global | **tetap unik global** — hash 32-byte acak, tidak ada untungnya di-scope, tinggal tambah kolom `account_id` buat tahu pemiliknya |
| `devices.device_id` | PK (unik global) | **tetap unik global** — ID acak, sama alasannya |

Alasan invoice di-scope ulang: dua merchant berbeda memang boleh pakai
`external_ref`/nominal yang sama — mereka toko yang berbeda, notifikasi
GoPay-nya juga dari device yang berbeda, tidak ada risiko tabrakan
sungguhan, dan memaksa keunikan global cuma bikin dua merchant saling
menghalangi tanpa alasan.

## 3. Aturan keamanan inti: derivasi `account_id`

**`account_id` tidak pernah diambil dari input client** (body/query/header) —
selalu diturunkan server-side dari kredensial yang sudah diverifikasi:

| Jalur masuk | Sumber `account_id` |
|---|---|
| Cookie sesi dashboard (`admin_session`) | Akun yang login — payload sesi bawa `account_id`, bukan cuma flag boolean seperti sekarang |
| API key (`Authorization: Bearer`) | `api_keys.account_id` milik key itu, di-join saat `VerifyAPIKey` |
| HMAC device (`X-Device-ID` + tanda tangan) | `devices.account_id` milik device itu, di-join saat verifikasi |

Konsekuensi kode:

- `AdminFromContext(ctx) bool` → jadi `AccountFromContext(ctx) (accountID string, ok bool)`. Setiap pemanggil (`admin_*.go` handlers) yang sekarang cuma cek `ok` harus mulai memakai `accountID`.
- `APIKeyFromContext`, konteks HMAC device: `store.APIKey`/`store.Device` yang dikembalikan sudah punya field `AccountID` — tidak perlu context key baru, tinggal dipakai.
- **Setiap fungsi di `internal/store` yang menyentuh tabel-tabel di §2.2 wajib menerima `accountID string` sebagai parameter dan memfilternya di klausa `WHERE`** — bukan mengambil semua baris lalu memfilter di Go. Ini menyentuh hampir seluruh fungsi di `internal/store/{device,invoice,event,webhook,exception,apikey}.go` (lihat daftar lengkap di rencana implementasi, bukan di sini — dokumen ini fokus desain, bukan diff per fungsi).
- `requireLicense` middleware → jadi `requireActiveAccount`: mengambil `Account` dari `account_id` yang sudah diturunkan di atas, cek `Operational()` langsung dari baris itu (tidak ada lagi file lokal/goroutine 24 jam/grace period — semuanya dihapus bersama `internal/licenseclient`).

## 4. Vendor Dashboard

Tetap app terpisah (`vendor-dashboard/`), tetap cuma dipakai Akbar, tapi:

- Endpoint yang dipanggil pindah dari License Server (dibongkar) ke
  endpoint vendor-only baru **di backend utama**: `/api/v1/vendor/*`.
- Auth vendor tetap terpisah total dari sesi customer: tabel baru
  `vendor_admins` (skema sama seperti `admin_users` lama — username +
  password_hash, dibuat lewat CLI internal seperti `admintool` sekarang),
  cookie `vendor_session` (nama berbeda dari `admin_session` customer,
  supaya dua sesi ini tidak mungkin tertukar walau di browser yang sama).
- "Buat Customer" di Vendor Dashboard sekarang langsung membuat baris
  `accounts` (bukan lagi `customers`+`licenses` terpisah): isi
  `business_name`, `email`, `username`, plan, `expires_at`; password awal
  di-generate dan ditampilkan sekali (pola sama seperti License Key dulu —
  tidak bisa dilihat ulang, cuma hash yang tersimpan).
- Audit log (`audit_log`) pindah jadi tabel di database utama juga, tetap
  mencatat aksi vendor terhadap `accounts` (CREATED, RENEWED, SUSPENDED,
  REVOKED, dst — nama event sama seperti sebelumnya, cuma subjeknya
  `accounts` bukan `customers`+`licenses` terpisah).

## 5. Migrasi data whuzpay.com yang sudah ada

Instalasi `whuzpay.com` yang sekarang jalan (device, invoice, webhook, event
milik toko Akbar sendiri) **menjadi akun pertama**, bukan dihapus atau
dipindah ke instalasi lain:

1. Migration SQL menambah kolom `account_id` (nullable) ke seluruh tabel di §2.2.
2. Insert satu baris `accounts` untuk toko Akbar — kredensial diambil dari
   baris `admin_users` yang sekarang ada (username + password_hash-nya
   dipindah langsung, tidak perlu reset password).
3. `UPDATE` seluruh baris lama di tiap tabel supaya `account_id` menunjuk ke
   akun itu.
4. `ALTER COLUMN account_id SET NOT NULL` di tiap tabel, di migration yang
   sama.
5. `admin_users` **dihapus** (`DROP TABLE`) — sudah diserap penuh ke
   `accounts`.

Migration ini harus tetap bisa jalan bersih di database test yang kosong
(tidak ada baris `admin_users` untuk dipindah) — langkah 2 memakai baris
default kalau `admin_users` kosong, atau migration test dijalankan dengan
seed data admin dulu (detail teknis diputuskan di tahap implementasi, bukan
di sini).

## 6. Testing

Kategori test yang **wajib ada dan baru** (belum pernah ada karena dulu
single-tenant): **isolasi antar akun**, di tiap endpoint yang menyentuh data
tabel §2.2:

- Buat dua akun (A dan B), masing-masing dengan device/invoice/webhook/API
  key sendiri.
- Login sebagai A, coba akses/ubah resource milik B lewat ID-nya langsung
  (path param, bukan tebak-tebak) → harus `404`, bukan `403` (403 membocorkan
  bahwa resource itu ADA tapi bukan milik dia; 404 tidak membocorkan apa-apa).
- Ulangi pola yang sama untuk jalur API key dan device HMAC — pastikan API
  key akun A tidak bisa menyentuh device akun B walau device ID-nya ditebak
  benar.

Ini pengujian yang paling penting di seluruh sub-project ini — kesalahan di
sini berarti kebocoran data lintas customer, bukan sekadar bug fungsional.

## 7. Di luar cakupan dokumen ini (sengaja)

- Alur signup publik (form pendaftaran, verifikasi email, dsb) — sub-project #2.
- Landing page whuzpay.com — sub-project #5 (setelah #2-#4).
- Penyesuaian mobile app (Backend URL, alur dapat Device ID+Secret) — sub-project #4.
- Billing/pembayaran otomatis untuk perpanjangan plan — belum dibahas sama sekali, `expires_at`/plan masih diisi manual lewat Vendor Dashboard seperti sekarang.
- Multi-user per akun (team members) — eksplisit ditunda, dikonfirmasi user.

## 8. Ringkasan dampak ke kode yang sudah ada

| Area | Dampak |
|---|---|
| `backend/cmd/licenseserver`, `internal/licenseserver`, `internal/licenseclient`, `migrations-license` | **Dihapus total** |
| `internal/store/admin.go` | Diganti logika `accounts` (§2.1) |
| `internal/store/{device,invoice,event,webhook,exception,apikey}.go` | Setiap fungsi tambah parameter `accountID`, filter query |
| `internal/httpapi/admin_auth.go`, `apikey_auth.go`, `auth_middleware.go` | `AccountFromContext`, sesi bawa `account_id`, `requireActiveAccount` menggantikan `requireLicense` |
| `internal/httpapi/admin_license.go` | Disederhanakan besar — tidak ada lagi status `unreachable`/grace period, cek langsung ke `accounts` |
| `migrations/0000N_accounts.sql` (baru) | Isi §2.1, §2.2, §2.3, §5 |
| `vendor-dashboard/` | `lib/api.ts` arah panggilan pindah ke backend utama; "Buat Customer" → buat `accounts` |
| `dashboard/` (Customer Dashboard) | Sebagian besar tidak berubah — sudah berbasis sesi+REST, cukup ikut makna `account_id` yang baru di baliknya |

Ini adalah perubahan besar yang menyentuh hampir seluruh `internal/store`
dan `internal/httpapi` — rencana implementasi (lewat `writing-plans`) perlu
memecahnya jadi langkah-langkah kecil yang masing-masing tetap lulus test,
bukan satu perubahan raksasa sekaligus.
