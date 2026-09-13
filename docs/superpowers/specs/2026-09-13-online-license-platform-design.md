# Platform Lisensi Online — Spec

**Tanggal:** 2026-09-13
**Status:** SUPERSEDED — diganti total oleh
[`2026-09-13-multitenant-accounts-design.md`](2026-09-13-multitenant-accounts-design.md)
(pivot self-hosted → hosted multi-tenant). License Server,
`internal/licenseclient`, `internal/licensecheck` yang dibangun dari spec
ini sudah dibongkar habis. Dibiarkan di sini untuk konteks sejarah.
**Menggantikan:** [`2026-09-13-license-system-design.md`](2026-09-13-license-system-design.md) (offline) — kode
verifikasi signature-nya dipakai ulang, lihat §4.

## 1. Latar belakang

[`docs/license-spec.md`](../../license-spec.md) adalah dokumen bisnis
otoritatif untuk model **Self-Hosted + Annual License** produk ini — 38
bagian, mencakup arsitektur penuh (License Server + Vendor Dashboard +
Installation ID + entitlements + audit log + version compatibility + bot
commands, dst). Spec ini adalah **potongan MVP** dari dokumen itu: setia ke
arsitektur intinya, tapi menunda bagian yang belum ada gunanya karena
fiturnya sendiri belum ada di produk (bot, release/version tracking) atau
belum krusial di skala satu-dua customer pertama (entitlement granular,
rate limiting khusus, replay protection).

Keputusan pokok, dikonfirmasi lewat brainstorming:

| Pertanyaan | Keputusan |
|---|---|
| Model validasi | **Online**: License Server terpisah, bukan file offline lagi |
| Lokasi License Server | VPS yang sama (`whuzpay.com`), subdomain baru `license.whuzpay.com` |
| Saat License Server tidak terjangkau | Toleransi via **grace period 7 hari** (§12 license-spec.md), pakai local state ter-cache |
| Revoke/suspend | Didukung sejak v1 (§18 license-spec.md) |
| Kelola customer/lisensi | **Vendor Dashboard** (Next.js baru, terpisah dari dashboard customer) |
| Entitlement | Cuma `max_devices` ditegakkan; field lain disimpan, belum ditegakkan |
| Installation ID | Dipakai (§6-8 license-spec.md) — license key bukan satu-satunya credential |
| Local state | Ditandatangani Ed25519 oleh License Server (§13) — **pakai ulang** `internal/licensecheck` |

## 2. Komponen

Tiga bagian yang bergerak, masing-masing deployable terpisah:

```text
backend/cmd/licenseserver/        binary baru, database sendiri (gopay_license)
backend/internal/licenseserver/   store + httpapi License Server, paket sendiri
backend/migrations-license/       migrasi goose khusus gopay_license
vendor-dashboard/                 Next.js baru, cuma Akbar yang pakai
backend/internal/licensecheck/    SUDAH ADA, dipakai ULANG oleh licenseserver (sign)
                                   dan backend (verify) -- lihat §4
backend/internal/licenseclient/   BARU -- backend customer memanggil licenseserver
dashboard/                        SUDAH ADA -- halaman /license ditulis ulang
```

**License Server tetap satu Go module dengan `backend/`, bukan module
terpisah** — `internal/licensecheck` (verifikasi Ed25519) dipakai ulang
langsung oleh keduanya (License Server menandatangani, backend customer
memverifikasi) tanpa duplikasi kode kripto, yang mustahil kalau keduanya
module Go berbeda (package `internal/` tidak bisa diimpor lintas module).
Deployment tetap terpisah penuh: binary sendiri (`cmd/licenseserver`),
systemd unit sendiri, database sendiri (`gopay_license`, terpisah total
dari `gopay` milik instalasi Akbar sendiri sebagai customer).

Satu VPS yang sama (`whuzpay.com`) melayani baik instalasi Akbar sendiri
(sebagai customer pertamanya sendiri, `backend/`+`dashboard/`) MAUPUN
License Server + Vendor Dashboard — dua hal yang **konseptual terpisah
total** (database, systemd unit, Caddy site block berbeda) walau kebetulan
satu mesin fisik dan satu Go module.

## 3. License Server — model data

Database Postgres baru `gopay_license`, migrasi lewat `goose` (pola sama
seperti `backend/`).

```sql
customers (id "cus_xxxx", name, created_at)

licenses (
  id "lic_xxxx", customer_id -> customers,
  key_hash bytea unique,      -- SHA-256 dari key mentah, pola sama api_keys
  plan text,                  -- label + sumber default entitlement saat create
  status text check in ('active','suspended','revoked'),
  max_devices int,
  production_installations int default 1,
  uat_installations int default 1,
  issued_at date, expires_at date,
  created_at, updated_at
)

installations (
  id "inst_xxxx", license_id -> licenses,
  environment text check in ('production','uat'),
  product_version text,
  activated_at timestamptz,
  released_at timestamptz null   -- diisi oleh "Reset Installation"
)

audit_log (
  id bigserial, actor text, action text, resource text,
  metadata jsonb, created_at timestamptz
)

admin_users (id, username, password_hash)   -- akun vendor, pola sama backend/
```

`status` yang **dikirim ke client** (`active`/`expiring`/`expired`/
`suspended`/`revoked`) dihitung saat request, bukan kolom tersendiri:

- `revoked`/`suspended` — langsung dari kolom `status`.
- kalau `status = 'active'` dan `now > expires_at` → `expired`.
- kalau `status = 'active'` dan sisa hari `<= 30` → `expiring`.
- selain itu → `active`.

Kuota installation per environment dicek dari **installation yang belum
`released_at`** — `COUNT(*) WHERE license_id=$1 AND environment=$2 AND
released_at IS NULL`, dibandingkan ke `production_installations`/
`uat_installations`.

## 4. Local License State — format dan penandatanganan

**`internal/licensecheck` (sudah ada) dipakai ulang, diperluas.** Format
file tetap dua baris (base64 payload, base64 signature), verifikasi Ed25519
tetap sama. Yang berubah:

- Payload sekarang berisi field lebih banyak: `license_id`, `installation_id`,
  `customer`, `plan`, `status` (activedst — enam nilai di atas, bukan cuma
  4 seperti sebelumnya), `expires_at`, `max_devices`, `validated_at` (kapan
  License Server menghasilkan state ini — **field baru, krusial untuk grace
  period**).
- Penanda tangan sekarang **License Server**, bukan `licensetool` manual —
  ditandatangani otomatis di setiap response `/activate` dan `/validate`,
  dengan key pair Ed25519 milik License Server (private key di env License
  Server, `LICENSE_SIGNING_PRIVATE_KEY`; public key hardcoded di
  `internal/licensecheck` seperti sebelumnya, cuma isinya kunci baru punya
  License Server, bukan kunci lama punya `licensetool`).
- Nama file tetap dikonfigurasi lewat `LICENSE_FILE_PATH` (default
  `/opt/gopay-ingestion/license-state.lic` — nama baru, `-state`, supaya
  jelas ini file yang ditulis OTOMATIS oleh backend, bukan dikirim manual
  seperti `license.lic` dulu).
- `cmd/licensetool` **dihapus** — tidak ada lagi penandatanganan manual.
  Aktivasi sekarang selalu online lewat `/activate`.

## 5. Alur aktivasi (§7, §10 license-spec.md — disederhanakan ke pola `.env` proyek ini)

`docs/license-spec.md` §7 menggambarkan aktivasi lewat form di Customer
Dashboard. Proyek ini **konsisten memakai `.env` + restart** untuk seluruh
secret (`DEVICE_SECRET_KEY`, `ADMIN_SESSION_KEY`, `WEBHOOK_SECRET_KEY`) dan
dashboard yang murni read-only — menambah SATU jalur berbeda (form web
untuk secret) cuma untuk lisensi memecah pola itu tanpa manfaat sepadan.
Aktivasi karena itu memakai mekanisme yang sama:

```text
Vendor buat license di Vendor Dashboard -> key mentah "PB-BUSINESS-XXXX-XXXX-XXXX"
ditampilkan SEKALI, dikirim ke customer di luar sistem (chat/email)
        |
Customer isi .env: LICENSE_KEY=PB-BUSINESS-... , ENVIRONMENT=production
        |
Restart backend customer (systemctl restart gopay-ingestion)
        |
Startup: local license-state.lic belum ada -> otomatis
POST /api/v1/license/activate {license_key, environment, product_version}
                     ke License Server
        |
License Server: key valid? kuota installation environment ini penuh?
        | tidak penuh
        v
Buat baris installations, balas state signed (§4)
        |
backend customer tulis ke LICENSE_FILE_PATH, load ke memori
        |
Dashboard customer (read-only) menampilkan status aktif
```

`environment` dibaca dari `ENVIRONMENT` (env var **baru**, `production`
atau `uat`) di `.env` backend customer — bukan ditebak dari nama domain.
`product_version` dari konstanta `internal/version.Current` (string statis,
di-bump manual tiap rilis — **belum** ada mekanisme rilis terkelola, lihat
§9 di luar lingkup).

Kalau kuota installation penuh, `/activate` membalas `409` dengan pesan
yang menyebutkan batasnya — customer diarahkan menghubungi vendor (sama
seperti alur "Installation limit reached" §23 license-spec.md). Kegagalan
aktivasi saat startup dicatat log dan status jadi `invalid`/tetap `missing`
tergantung sebabnya — server tetap menyala (§7 requireLicense v2), admin
bisa login dan melihat pesannya di `/license`, betulkan `LICENSE_KEY`, lalu
restart lagi untuk mencoba ulang.

## 6. Alur validasi periodik (§11-12 license-spec.md)

Goroutine baru di `cmd/server/main.go` backend customer, **terpisah** dari
ticker webhook yang sudah ada (interval beda jauh — 1 menit vs 24 jam,
menyatukannya cuma bikin bingung):

```go
licenseTicker := time.NewTicker(24 * time.Hour)
// tick pertama tidak menunggu 24 jam -- validasi juga dijalankan sekali
// saat startup, supaya restart proses tidak menunda deteksi status baru
// sampai sehari.
```

Tiap tick, `internal/licenseclient.Validate(ctx)`:

1. Baca `license-state.lic` yang ada (butuh `installation_id` dan
   `license_key`-nya sendiri — **license_key disimpan terpisah**, di `.env`
   sebagai `LICENSE_KEY`, bukan di file state, supaya file state boleh
   dibaca ulang tanpa membocorkan key mentah lewat isi file yang mungkin
   ikut ke-backup/ke-log).
2. `POST /api/v1/license/validate` dengan `Authorization: Bearer
   <LICENSE_KEY>`, body `{installation_id, product_version}`.
3. **Berhasil** → tulis ulang `license-state.lic` dengan state baru
   (termasuk `validated_at` = sekarang).
4. **Gagal terhubung** (timeout, DNS, 5xx) → **jangan** menimpa file state.
   Biarkan `requireLicense` menghitung kesegarannya sendiri saat request
   masuk (§7) — ticker cuma mencoba me-refresh, tidak pernah menghukum
   kegagalan jaringan dengan menghapus state yang masih valid.
5. **Ditolak** (`401`/`404` — key dicabut/installation direset dari sisi
   vendor) → **hapus** `license-state.lic`. Ini beda dari kegagalan
   jaringan: server sudah menjawab dengan tegas "installation ini tidak
   lagi sah", jadi harus langsung berhenti dipercaya, tidak menunggu grace
   period habis.

## 7. `requireLicense` v2

Dipanggil tiap request (baca file + verifikasi, murah — bukan panggil
jaringan tiap request, itu tugas ticker di §6):

```text
1. Load license-state.lic, verifikasi signature (internal/licensecheck.Load)
   -> gagal verifikasi / file tidak ada => "missing" (belum aktivasi)
2. status dari payload:
   - active / expiring          => LOLOS
   - expired / suspended / revoked => 402, kode "license_<status>"
3. Kalau status di file "active"/"expiring" TAPI validated_at sudah
   > 7 hari yang lalu (grace period lewat, ticker gagal berkali-kali)
   => 402 "license_unreachable" -- kemungkinan Licensc Server down lama
      atau instalasi ini kehilangan akses internet berkepanjangan
```

Pesan `license_unreachable` di dashboard menjelaskan ini beda dari
`license_expired` — bukan soal masa berlaku, tapi instalasi ini sudah
lama tidak berhasil menghubungi License Server, dan menyarankan memeriksa
koneksi keluar (outbound HTTPS ke `license.whuzpay.com`) atau menghubungi
Akbar kalau server-nya sendiri yang bermasalah.

`GET /api/v1/admin/license` (sudah ada) diperluas: menyertakan
`installation_id`, `environment`, `validated_at`. **Read-only sepenuhnya**
(§5) — kalau statusnya `missing`, dashboard menampilkan pesan yang
mengarahkan admin mengisi `LICENSE_KEY` di `.env` lalu restart, bukan form
input di halaman itu sendiri.

## 8. Vendor Dashboard — lingkup

Next.js baru, autentikasi cookie sesi (pola sama persis dashboard customer,
lewat `admin_users` License Server). Halaman:

- `/login`
- `/` — daftar customer, tombol "Buat customer"
- `/customers/[id]` — daftar license customer itu, tombol "Buat license"
  (plan, `max_devices`, `expires_at` — key mentah ditampilkan **sekali**
  dalam dialog, pola sama persis API key/webhook secret di dashboard
  customer)
- `/licenses/[id]` — detail satu license: status, daftar installations
  (environment, `activated_at`, tombol **Reset Installation** kalau belum
  di-release), tombol Renew (ubah `expires_at`), Suspend, Revoke
- `/audit-log` — tabel `audit_log`, terbaru dulu, filter by resource

Tidak ada halaman Releases/Versions/Entitlements-editor kompleks di v1 —
`plan` yang dipilih saat create license cuma preset `max_devices` (Starter
3, Business 10, Enterprise -1/unlimited), tidak ada UI entitlement custom
per-license.

## 9. Eksplisit di luar lingkup

- **Version compatibility check** (§31 license-spec.md) — `product_version`
  dikirim dan disimpan di `installations`, tapi License Server tidak
  menolak versi lama. Perlu mekanisme rilis/changelog dulu supaya
  "minimum_supported_version" berarti sesuatu.
- **Entitlement selain `max_devices`** (`max_webhooks`, `telegram_bot`,
  `whatsapp_bot`, `advanced_logs`, dst., §17) — field-field itu belum
  ditegakkan karena fiturnya sendiri (bot, dst.) belum ada di produk.
- **Bot license commands** (§28) — tidak ada bot di produk ini sama sekali.
- **Rate limiting khusus validasi/aktivasi, replay protection/nonce**
  (§30) — dicatat sebagai hardening lanjutan. Endpoint `/activate` dan
  `/validate` untuk v1 dilindungi throttle login yang sama polanya dengan
  `loginThrottle` admin yang sudah ada (per IP), belum ada mekanisme
  nonce/timestamp khusus.
- **Renewal warning multi-tahap** (30/14/7/3/1 hari, §21) — dashboard
  customer cuma menampilkan satu ambang (`expiring`, 30 hari), bukan
  rangkaian notifikasi bertingkat.
- **Notifikasi keluar** (email/Telegram/WhatsApp saat mendekati expired)
  — sama seperti versi offline sebelumnya, hanya banner di dashboard.
- **RBAC granular di Vendor Dashboard** (§30 "Strict RBAC") — satu akun
  admin vendor untuk MVP, sama seperti pola satu-admin di dashboard
  customer.
- **Audit log untuk aksi customer** (aktivasi dari sisi mereka) — audit
  log v1 cuma mencatat aksi vendor di Vendor Dashboard, bukan aktivitas
  instalasi customer.

## 10. Testing

- `license-server/internal/store`: create customer/license, hitung status
  (active/expiring/expired/suspended/revoked) dari kombinasi `status` DB +
  `expires_at`, kuota installation (penuh vs belum, per environment
  terpisah), reset installation membuka kuota lagi.
- `license-server/internal/httpapi`: `/activate` sukses, ditolak (key
  salah, kuota penuh), `/validate` sukses, ditolak (installation
  di-release/tidak cocok license), signature di response valid (verifikasi
  pakai public key test terpisah dari yang hardcoded produksi — pola sama
  seperti test `internal/licensecheck` yang sudah ada).
- `backend/internal/licenseclient`: pakai `httptest.Server` memalsukan
  License Server — validate sukses menulis ulang file, validate gagal
  jaringan TIDAK menimpa file, validate ditolak (401/404) menghapus file.
- `backend/internal/httpapi`: `requireLicense` v2 — active/expiring lolos,
  expired/suspended/revoked ditolak 402, active-tapi-`validated_at` basi
  (>7 hari) ditolak `license_unreachable`, endpoint activate memanggil
  License Server dan menyimpan hasilnya (pakai `httptest.Server` yang sama).
- `vendor-dashboard` dan perubahan `dashboard`: `npx tsc --noEmit`, `npx
  eslint .`, `npx next build` — sama seperti pola dashboard customer,
  tidak ada e2e.
- `NEEDS-DEVICE`: deploy `license-server`+`vendor-dashboard` sungguhan ke
  `license.whuzpay.com`, aktivasi instalasi `whuzpay.com` yang sudah ada
  lewat key sungguhan, verifikasi `/license` di dashboard customer, lalu
  uji suspend/revoke dari Vendor Dashboard dan pastikan instalasi customer
  benar-benar terblokir setelah validasi berikutnya.
