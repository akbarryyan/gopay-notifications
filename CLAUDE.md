# GoPay Notification Bridge

Bridge notifikasi GoPay dari HP Android ke backend, sebagai fondasi payment
gateway hosted multi-tenant (dulu self-hosted — lihat catatan pivot di
"Pemecahan scope").

## Pindah AI

[`docs/ai-handoff-prompt.md`](docs/ai-handoff-prompt.md) memuat prompt siap
tempel untuk AI lain, berikut daftar jebakan yang sudah pernah memakan waktu di
repo ini. Perbarui bagian "Keadaan saat ini" di sana setiap milestone selesai.

## Dokumen sumber

Urutan kewenangan bila terjadi perbedaan:

1. [`docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md`](docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md) — spec ingestion + Android bridge, paling berwenang untuk sub-project 1+2
2. [`docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md`](docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md) — spec invoice, nominal unik, matching, API key untuk sub-project 3 fase 1
3. [`docs/superpowers/specs/2026-09-13-webhook-delivery-design.md`](docs/superpowers/specs/2026-09-13-webhook-delivery-design.md) — spec webhook delivery untuk sub-project 3 fase 2
4. [`docs/superpowers/specs/2026-09-13-exception-console-design.md`](docs/superpowers/specs/2026-09-13-exception-console-design.md) — spec konsol pengecualian untuk sub-project 3 fase 4
5. [`docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md`](docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md) — spec pivot arsitektur self-hosted → hosted multi-tenant, paling berwenang untuk sub-project 5 (fase 1: akun & data model). Rencana implementasi: [`docs/superpowers/plans/2026-09-13-multitenant-accounts-plan.md`](docs/superpowers/plans/2026-09-13-multitenant-accounts-plan.md).
6. [`docs/license-spec.md`](docs/license-spec.md) — spec bisnis lisensi (Self-Hosted + Annual License) yang melatarbelakangi keputusan awal. SUPERSEDED sebagai model bisnis sejak pivot ke hosted (lihat #5) — dibaca untuk konteks sejarah, bukan lagi rujukan aktif.
7. [`docs/superpowers/specs/2026-09-13-online-license-platform-design.md`](docs/superpowers/specs/2026-09-13-online-license-platform-design.md), [`2026-09-13-license-system-design.md`](docs/superpowers/specs/2026-09-13-license-system-design.md) — dua spec lisensi sebelumnya (online lalu offline). Keduanya SUPERSEDED oleh #5 — License Server, `internal/licenseclient`, `internal/licensecheck` yang dibangun dari spec-spec ini sudah dibongkar total.
8. [`docs/api-contract.md`](docs/api-contract.md) — kontrak antara backend dan Android
9. [`docs/dashboard-spec.md`](docs/dashboard-spec.md) — rancangan dashboard penuh. MVP yang sudah dibangun ([`dashboard/README.md`](dashboard/README.md)) hanya subset-nya; jangan menganggap seluruh isi dokumen ini sudah ada.
10. [`docs/prd.md`](docs/prd.md), [`docs/detail-project.md`](docs/detail-project.md) — dokumen awal, mengasumsikan model self-hosted lama

Spec lebih berwenang daripada PRD karena memuat keputusan yang sengaja menyimpang dari PRD dan sudah disetujui. Penyimpangan itu terdaftar di §9 spec — jangan "memperbaiki" implementasi agar kembali sesuai PRD tanpa memeriksa daftar itu lebih dulu.

## QA wajib

**Setiap kali sebuah milestone selesai diimplementasikan, jalankan QA sesuai [`docs/qa/qa-rules.md`](docs/qa/qa-rules.md) dan perbarui [`docs/qa/qa-report.md`](docs/qa/qa-report.md) sebelum menyatakan milestone itu selesai.**

Milestone tanpa laporan QA yang diperbarui belum selesai, sekalipun seluruh kodenya sudah ditulis dan seluruh testnya lulus.

Aturan yang paling mudah dilanggar dan paling penting ditegakkan:

- `PASS` menuntut keluaran perintah yang ditempel, bukan pernyataan bahwa sesuatu berfungsi.
- Hal yang hanya dapat diuji di HP sungguhan ditandai `NEEDS-DEVICE`, tidak pernah `PASS` berdasarkan penalaran.
- Butir requirement yang belum tersentuh ditandai `PENDING`, tidak dihapus dari tabel.

## Pemecahan scope

| # | Sub-project | Status |
|---|---|---|
| 1 | Event ingestion (Go) | sedang dikerjakan |
| 2 | Android bridge (Expo + Kotlin) | sedang dikerjakan |
| 3 | Gateway: invoice, nominal unik, matching, webhook, API key, konsol pengecualian | seluruh 4 fase selesai |
| 4 | Dashboard admin (Next.js) — Overview, Devices, Events, Transactions, API Keys, Webhooks, Exceptions, License | sedang dikerjakan |
| 5 | Platform akun multi-tenant — fase 1 (tabel `accounts`, `account_id` di seluruh tabel data, endpoint vendor, Vendor Dashboard) | fase 1 selesai (`make test` PASS), deploy VPS `NEEDS-DEVICE`. Fase 2 (signup publik) dan fase 6 (landing page) sudah selesai — lihat di bawah. Fase 3-5 (penyesuaian lanjutan Customer Dashboard swalayan tambah device, mobile bridge) belum dimulai |

Sub-project 3 dipecah jadi 4 fase, urutan dan rinciannya ada di spec #2–#4
di atas. Seluruhnya sudah selesai — kalau ada permintaan fitur baru untuk
gateway ini, itu perluasan di luar keempat fase itu, bukan bagian dari
salah satunya.

Dashboard mencakup halaman yang datanya sungguhan ada: Overview, Devices,
Events, Transactions, API Keys, Webhooks, Exceptions, API Docs, License,
Settings.
Logs ditampilkan di sidebar sebagai "Segera" (non-aktif).

**PIVOT ARSITEKTUR (2026-09-13):** produk ini berubah dari self-hosted
(tiap customer deploy backend+dashboard sendiri) jadi **hosted SaaS
multi-tenant seperti Midtrans** — customer daftar, dapat akun, integrasi
lewat REST API ke SATU backend yang dihost vendor (Akbar), login ke
Customer Dashboard yang sama untuk semua customer (data dipisah per akun,
bukan per instalasi). Keputusan ini dipecah jadi 6 sub-project di
[`docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md`](docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md)
§0; **baru #1 (fondasi: akun & data model) yang selesai**. Paragraf
"Dashboard ini untuk customer, bukan vendor" tetap berlaku secara login
(satu akun = satu login), TAPI kalimat "satu instalasi = satu backend =
satu dashboard" di bawah ini **sudah tidak akurat** — backend sekarang
satu untuk semua customer, dipisahkan lewat `account_id` per baris, bukan
lewat instalasi terpisah. `make dev-admin`/`admintool` yang dulu membuat
akun customer sekarang cuma membuat **akun vendor**; akun customer dibuat
vendor lewat Vendor Dashboard (`POST /api/v1/vendor/accounts`).

Konsekuensinya untuk copywriting dan desain dashboard: jangan menyebut
istilah arsitektur internal ("backend", "database", nama service, dsb) di
teks yang tampil ke pengguna — itu bukan urusan customer, dan membocorkannya
bikin produk terasa seperti tool developer, bukan produk jadi. Field API
boleh tetap bernama `backend`/`database` (kontrak sudah ada, sub-project 3
akan menambah komponen lain ke sana), tapi label dan copy yang ditampilkan
di halaman harus diringkas jadi bahasa yang netral, mis. "Status Layanan" /
"Aktif", bukan "Backend: operational".

Sistem akun (sub-project 5) menggantikan sistem lisensi lama secara total.
Detail lengkap di bagian "Sistem akun multi-tenant" di bawah.

## Sistem akun multi-tenant

Spec: [`docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md`](docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md).
Menggantikan **total** dua platform lisensi sebelumnya (online lalu
offline, keduanya SUPERSEDED) — License Server, `internal/licenseclient`,
`internal/licensecheck` yang dibangun dari spec-spec itu sudah dibongkar
habis, bukan sekadar diperluas.

**Model sekarang:** satu tabel `accounts` di database utama (`gopay`) —
gabungan akun login + plan/kuota, satu baris = satu customer = satu login.
Seluruh tabel data (`devices`, `invoices`, `notification_events`,
`api_keys`, `webhook_endpoints`, `webhook_deliveries`, `event_reviews`)
punya kolom `account_id`, **selalu diturunkan server-side** dari sesi
dashboard, API key, atau HMAC device yang sudah diautentikasi — tidak
pernah dipercaya dari body/query/header request manapun.

`requireActiveAccount` menggantikan `requireLicense` — mengecek status
akun (`active`/`expired`/`suspended`/`revoked`, lihat catatan "expiring"
yang dicabut di §"Pengingat kedaluwarsa ke customer") langsung ke
`accounts` tiap request, tanpa file lokal atau grace period (tidak ada
lagi jaringan antar dua service untuk dicek).

**Vendor Dashboard** (`vendor-dashboard/`, Next.js terpisah, cuma Akbar
yang pakai) sekarang manggil endpoint vendor **di backend utama**
(`/api/v1/vendor/*`, sesi `vendor_session` + tabel `vendor_admins`
terpisah total dari sesi customer `admin_session`) — bukan lagi service
License Server yang berdiri sendiri. Vendor bikin/kelola account langsung
(business_name/email/username/plan/expires_at), password awal digenerate
dan ditampilkan sekali.

Device dibuat lewat `cmd/devicetool -account <id> -name "..."` (flag
`-account` sekarang wajib) sampai Customer Dashboard punya fitur swalayan
tambah device (sub-project #3 di spec §0, belum dikerjakan).

## Keputusan arsitektur yang tidak boleh dilanggar diam-diam

- **Kotlin memiliki pipeline data; TypeScript hanya UI.** JS runtime mati saat aplikasi tertutup, jadi pengiriman HTTP tidak boleh ditulis di TypeScript.
- **Native adalah satu-satunya pemilik database.** TypeScript membaca lewat native module, tidak pernah membuka file DB sendiri.
- **Parsing nominal di HP bersifat display-only.** Backend melakukan ekstraksi otoritatifnya sendiri dari teks mentah.
- **Arah transaksi (uang masuk vs keluar) ditentukan backend, bukan HP.** Salah dalam hal ini berarti order ditandai lunas padahal tidak ada uang masuk.
- **Sumber pembayaran adalah GoPay Merchant**, package **`com.gojek.gopaymerchant`** (berubah 2026-09-11, menggantikan akun pribadi sepenuhnya). Tetap disimpan sebagai daftar yang dapat diedit, bukan konstanta.
- **Jangan pernah memantau `com.gojek.gopay` bersamaan dengan `com.gojek.gopaymerchant`.** Keduanya melaporkan pembayaran yang sama dengan teks identik; karena `packageName` ikut jadi bahan `event_id`, satu pembayaran akan menghasilkan dua event dan terhitung dua kali. Idempotency tidak menolong.
- **Arah transaksi ditentukan allowlist judul, bukan blocklist.** Yang tidak dikenali ditolak, bukan ditebak. Entri awal: `Pembayaran QRIS statis diterima`. Entri `Transfer masuk` berasal dari akun pribadi dan tidak boleh dipakai.
- **`event_id` dihitung dari `packageName | title | text | when`** — bukan dari `notificationKey` (konstan di GoPay) atau `postTime` (berubah tiap repost).
- **Tanda tangan HMAC dihitung dari byte mentah body**, diverifikasi sebelum decode JSON.
- **Nama connector tidak boleh masuk URL.** Rutenya `POST /api/v1/events`; pembeda sumber hanya field `source`, divalidasi terhadap `internal/connector`. Menambah DANA atau OVO berarti menambah satu entri di registry, bukan menambah rute.
- **Heartbeat tiap 15 menit, toleransi 45 menit.** Ini membalik Open Question #14 di spec secara sadar: untuk produk berbayar, menunggu pembayaran untuk tahu HP mati tidak dapat diterima. Toleransi harus selalu beberapa kali interval — Android menunda periodic work saat Doze, dan status yang sering salah adalah status yang diabaikan.
- **`listener_connected: false` di heartbeat tidak boleh ditolak.** Izin aktif tetapi listener tidak terikat adalah gejala service dibunuh OEM — itu justru sinyal yang dicari, bukan payload yang cacat.
- **`X-App-Version` tidak ikut ditandatangani.** Backend harus tahu versinya untuk memverifikasi, jadi ia mustahil menjadi bagian tanda tangan. Aplikasi lama tidak mengirimnya dan itu bukan alasan menolak request.

## Perangkat target

OPPO CPH2365, Android 13, ColorOS — pembunuh background process paling agresif. Tersambung lewat adb wifi di `192.168.1.66:41721`; pakai `ANDROID_SERIAL` agar tooling tidak bingung bila muncul dua entri adb untuk HP yang sama. Setiap keputusan soal ketahanan service harus diuji di sana, tidak boleh diasumsikan dari perilaku Android standar.

Notifikasi transaksi GoPay dapat datang lewat channel **"Promotions and Marketing"** — begitu yang teramati di aplikasi akun pribadi. Jangan pernah menyarankan mematikan channel notifikasi apa pun milik aplikasi sumber; mematikannya mematikan seluruh sistem tanpa gejala.

## Pembagian kerja

**Claude menulis kode; Akbar menjalankan build dan server.** Jangan menjalankan
`expo run:android`, `gradlew`, `npm run`, `go run ./cmd/server`, `expo start`,
atau `docker compose up`. Tulis kodenya, lalu berikan perintahnya untuk
dijalankan Akbar, dan tunggu keluarannya ditempelkan.

Perintah baca-saja yang cepat masih boleh dijalankan sendiri: `git status`,
`adb devices`, `adb shell pm list packages`, `go vet`, `go build`, `grep`.

Untuk dashboard: `npm install`, `npx tsc --noEmit`, `npx eslint .`, dan
`npx next build` boleh dijalankan sendiri — bentuknya verifikasi satu kali
jalan, bukan proses yang tetap hidup. `npm run dev` tetap milik Akbar.

Untuk laporan QA, butir yang menuntut build atau server ditandai `NEEDS-DEVICE`
sampai Akbar menempelkan buktinya — tidak pernah `PASS` berdasarkan penalaran.

## Perintah

### Update server produksi

Produksi: `whuzpay.com` + `vendor.whuzpay.com` di AWS EC2
(`ssh -i ~/vps-aws-trial.pem ubuntu@13.60.252.148`), systemd + Caddy, tanpa
Docker. Langkah update setelah `git push` (build di laptop, upload `tar |
ssh`, migrasi lewat terowongan SSH, cadangan `.prev`, rollback) ada di
[`backend/deploy/README.md`](backend/deploy/README.md) §"Update rutin
setelah ada perubahan kode". Kalau Akbar bertanya cara update VPS, arahkan
ke sana dan sebutkan bagian mana yang perlu dijalankan (U1 backend, U2
dashboard, U3 vendor-dashboard, U5 bila ada env var baru) sesuai perubahan
yang di-push.

### Backend

```bash
cd backend
make db-up            # Postgres di :5433 lewat docker compose
make migrate          # goose up, database test
make test             # db-up + migrate + go test ./... -p 1
```

`-p 1` wajib. Paket `store` dan `httpapi` sama-sama `TRUNCATE` database test yang
sama, dan Go menjalankan paket secara paralel — tanpa `-p 1` keduanya saling
menghapus data dan gagal secara acak, padahal sendiri-sendiri lulus.

**Dua database di instance Postgres yang sama**, dan bedanya penting:

| | Dipakai untuk | Umur data |
|---|---|---|
| `gopay_test` | `make test` | **dihapus tiap kali test jalan** (`TRUNCATE`) |
| `gopay_dev` | backend lokal untuk aplikasi varian `development` | bertahan |

Jangan pernah mengarahkan HP ke `gopay_test`: `make test` berikutnya akan
menghapus event sungguhan tanpa peringatan.

Menjalankan backend lokal supaya HP dapat menghubunginya:

```bash
cd backend
cp .env.dev.example .env.dev
echo "DEVICE_SECRET_KEY=$(make -s dev-key)" >> .env.dev

make run-dev        # buat gopay_dev bila belum ada, migrasi, lalu jalankan server
make dev-device     # cetak Device ID + Secret untuk diisi di HP
make dev-url        # cetak Backend URL sesuai alamat LAN laptop saat ini
```

Dua hal yang mudah salah dan sudah dijaga di `.env.dev.example`:

- `LISTEN_ADDR` wajib `0.0.0.0`, bukan `127.0.0.1` — HP di LAN tidak akan
  pernah bisa menghubungi server yang hanya mengikat ke localhost.
- Port default `8090`, bukan 8080. Port 8080 dan 8081 di mesin ini dipakai
  container project lain (`soundbox-mqtt-*`). Bila bentrok, server Go gagal
  bind dan mati, sementara HP diam-diam bicara ke aplikasi lain dan Test
  Connection menjawab `http_404`. `make run-dev` memeriksanya lebih dulu dan
  berhenti dengan menyebut proses pemiliknya.
- `DEVICE_SECRET_KEY` wajib tetap antar restart. Kunci baru membuat secret
  device yang sudah tersimpan tidak dapat didekripsi, dan device harus dibuat
  ulang tiap kali server dinyalakan.

Jebakan yang sudah pernah kena: bila server versi lama masih memegang port,
`go build -o` ke path binary yang sedang berjalan gagal dengan *text file busy*,
dan endpoint baru membalas 404 karena yang melayani adalah binary lama. Matikan
berdasarkan port, bukan pola nama:

```bash
pid=$(ss -ltnpH 'sport = :8098' | grep -oP 'pid=\K[0-9]+' | head -1) && kill "$pid"
```

Membuat device di lingkungan lain (UAT/produksi), dengan env yang sesuai:

```bash
go run ./cmd/devicetool -genkey                 # cetak DEVICE_SECRET_KEY
go run ./cmd/devicetool -name "HP GoPay Utama"  # cetak Device ID + Secret
```

Mengisi `gopay_dev` dengan banyak data sekaligus (dev lokal saja —
**jangan pernah** ke `gopay_test`/UAT/produksi): `cmd/seedtool` membuat
beberapa account customer sekaligus beserta device/invoice/event/API
key/webhook-nya, lewat `Store` yang sama seperti server sungguhan
(password/secret ikut ter-enkripsi/ter-hash persis alur asli, jadi
account hasil seed bisa langsung dipakai login sungguhan).

```bash
go run ./cmd/seedtool -accounts 8   # default password123 untuk semua account
```

`make db-down` menghapus volume Docker — **`gopay_dev` ikut hilang**, bukan
hanya data test.

Produksi di-backup setiap hari oleh `gopay-backup.timer`
(`backend/deploy/gopay-backup.sh`: `pg_dump` terverifikasi, retensi lokal
14 hari, opsional unggah ke S3). Cara pasang, S3, dan restore di
`backend/deploy/README.md` §"Backup database otomatis". `.env` sengaja
tidak ikut dibackup — kunci enkripsi disimpan terpisah dari dump.

Kesehatan server produksi dipantau dua lapis: `gopay-monitor.timer`
(`backend/deploy/gopay-monitor.sh`, tiap 5 menit, cek service, URL publik,
sertifikat, disk, memori, umur backup, lalu Telegram ke vendor **hanya saat
status berubah**) dan UptimeRobot dari luar untuk kasus VPS mati total,
yang tidak bisa dideteksi skrip di server itu sendiri. Cara pasang di
`backend/deploy/README.md` §"Monitoring dan peringatan".

### Lingkungan

Tiga varian, package berbeda sehingga dapat terpasang bersamaan dan datanya
disandera Android di sandbox masing-masing:

| Varian | Package | Backend |
|---|---|---|
| `development` | `id.akbarryyan.gopaybridge.dev` | laptop, HTTP polos diizinkan |
| `uat` | `id.akbarryyan.gopaybridge.uat` | `uat.<domain>`, port 8081 |
| `production` | `id.akbarryyan.gopaybridge` | `<domain>`, port 8080 |

`DEVICE_SECRET_KEY` UAT dan produksi **wajib berbeda**. Device terdaftar per
database, jadi HP UAT yang salah diarahkan ke produksi ditolak
`401 invalid_signature` — dan jaminan itu hilang bila kuncinya dipakai ulang.

Domain diedit sekali di konstanta `DOMAIN` pada `mobile/src/lib/env.ts`.
Backend URL ditanam sebagai nilai awal per varian dan hanya diisi bila Settings
masih kosong, sehingga perubahan manual tidak pernah ditimpa.

Varian di UI diambil dari package name aplikasi yang benar-benar terpasang
(`getEnvironment()` di native), bukan dari konfigurasi build — supaya tidak bisa
menyimpang dari kenyataan.

### Mobile

Selalu dengan `APP_VARIANT` dan `ANDROID_SERIAL`. Tanpa `APP_VARIANT`, build
menghasilkan **production**.

adb menyambung ulang sendiri lewat mDNS sehingga muncul **dua entri untuk HP yang
sama** — satu lewat IP, satu bernama `adb-c62dac4f-4meRBj (2)._adb-tls-connect._tcp`.
Expo memotong nama mDNS itu lalu gagal dengan `Could not find device with name`.
Matikan penemuan otomatisnya sekali di awal sesi:

```bash
export ADB_MDNS_AUTO_CONNECT=0
adb kill-server && adb start-server
adb connect 192.168.1.66:41721
adb devices          # harus tinggal satu baris
```

Bila APK sudah terbangun tetapi pemasangannya yang gagal, pasang langsung tanpa
build ulang:

```bash
adb -s 192.168.1.66:41721 install -r android/app/build/outputs/apk/debug/app-debug.apk
```

```bash
cd mobile
export ANDROID_SERIAL=192.168.1.66:41721
export APP_VARIANT=development

npx expo prebuild --platform android --clean   # wajib saat berpindah varian: package name berubah
npx expo run:android                           # build + pasang ke HP (lama saat pertama)
npx expo start --dev-client                    # dev server, setelah aplikasi terpasang
npx tsc --noEmit                               # periksa tipe
```

Unit test Kotlin:

```bash
cd mobile/android
./gradlew :gopay-listener:testDebugUnitTest
```

`android/` tidak di-commit dan tidak boleh diedit manual — seluruh perubahan
manifest ditulis sebagai config plugin di `mobile/plugins/`.

HTTP polos hanya aktif di build development lewat `usesCleartextTraffic` di
`expo-build-properties`. Jangan menggantinya dengan daftar host yang disebut
satu per satu: alamat LAN laptop berubah tiap ganti jaringan, dan
`base-config cleartextTrafficPermitted="false"` ikut memblokir Metro sehingga
aplikasi gagal start.

#### Pairing lewat QR (swalayan, tanpa ketik manual)

Di halaman Devices (`dashboard/`), dialog "Device dibuat" sekarang juga
menampilkan QR code (`react-qr-code`) berisi JSON
`{v, backend_url, device_id, device_secret}` — nilai yang SAMA dengan yang
sudah ditampilkan untuk disalin manual, cuma dikodekan berbeda. Di
aplikasi Android, tombol "Scan QR dari Dashboard" di `SettingsScreen`
(`expo-camera`, `CameraView` + `useCameraPermissions`, plugin config di
`app.config.ts`) memindai lalu langsung memanggil
`GopayListener.saveSettings(...)` dan `testConnection()` yang sudah ada —
**tidak ada perubahan Kotlin sama sekali**, karena jalur penyimpanan
settings sudah sepenuhnya bisa dikendalikan dari JS sejak awal.

**`backend_url` di QR WAJIB menyertakan `/api/v1`**, persis format nilai
bawaan per varian di `mobile/src/lib/env.ts` — kode native menempelkan
`/events`/`/health`/dst APA ADANYA di belakang `backendUrl`
(`Uploader.kt`), tidak pernah menyisipkan `/api/v1` sendiri. Dashboard
membangunnya dari `window.location.origin + "/api/v1"`. Salah taruh
akhiran ini membuat QR **tampak berhasil dipindai** tapi Test Connection
gagal — jebakan yang sudah pernah kena sekali di sesi yang menulis fitur
ini, sebelum sempat diuji ke HP.

QR memuat Device Secret mentah — sama sensitifnya dengan nilai yang sudah
ditampilkan untuk disalin manual, bukan celah baru; UI menandainya jangan
di-screenshot/dibagikan.

### Dashboard

Next.js 16 (App Router) + Tailwind v4 + shadcn/ui (base-ui, bukan Radix —
komposisi trigger memakai prop `render`, bukan `asChild`). Folder `dashboard/`.

```bash
cd dashboard
npm install
cp .env.local.example .env.local   # isi BACKEND_URL bila backend bukan di :8090
npm run dev                        # Akbar yang menjalankan
```

Satu origin dengan backend lewat routing path, bukan CORS: di produksi Caddy
merutekan `/api/*` ke backend Go dan sisanya ke Next.js; saat dev,
`next.config.ts` me-rewrite `/api/*` ke `BACKEND_URL` lokal.

Autentikasi lewat cookie sesi HttpOnly dari `POST /api/v1/admin/login`.
`src/proxy.ts` (Next 16 mengganti nama `middleware.ts` menjadi `proxy.ts`)
hanya memeriksa keberadaan cookie untuk mencegah kedipan halaman kosong —
validitas sesi sesungguhnya selalu diputuskan backend lewat `requireAdmin`.

Halaman yang datanya sungguhan ada: Overview, Devices, Events, Transactions,
API Keys, Webhooks, Exceptions, License, Logs (riwayat aktivitas akun),
Settings (profil dengan verifikasi email, ganti password, hubungkan
Telegram), dan API Docs.

**API Docs (`/api-docs`)** adalah dokumentasi integrasi untuk customer:
autentikasi API key, `POST`/`GET /api/v1/invoices`, payload dan tanda
tangan webhook (`X-Webhook-Signature` = hex HMAC-SHA256 body mentah dengan
secret `whsec_…` apa adanya), percobaan ulang (5 kali, jeda 1/2/4/8 menit),
dan kode error, dengan contoh curl/Node/PHP. Isinya ditulis tangan dari
kode, bukan dibangkitkan otomatis, jadi **setiap perubahan perilaku API
invoice atau webhook wajib ikut memperbarui halaman ini**. Daftar file
sumbernya ada di komentar atas `api-docs/page.tsx`.

**Section Harga di landing page (`/`, `PricingSection` di
`components/landing/pricing-faq-footer.tsx`) diambil dari
`GET /api/v1/pricing-plans`** (publik, tanpa sesi) — bukan array tetap di
kode lagi, datanya diatur vendor lewat halaman Plans (lihat §"Vendor
Dashboard" di bawah). Komponennya jadi client component (`fetch` di
`useEffect`) khusus untuk ini — satu-satunya cara Next.js di sini memanggil
backend tanpa `BACKEND_URL` di produksi (lihat catatan "Dashboard produksi
tidak butuh BACKEND_URL" di bawah). Layout/styling kartu SENGAJA
dipertahankan sama; yang berubah cuma sumber datanya. Baris harga
(`price_label`/`price_period`) tidak ditampilkan sama sekali untuk plan
yang belum diisi harganya vendor — jangan mengubahnya jadi menampilkan
"Rp 0" atau angka bawaan apa pun.

### Verifikasi email saat signup

Migrasi 00015 (`accounts.email_verified_at`, tabel
`email_verification_tokens`, pola sama persis `password_reset_tokens`:
hash SHA-256, sekali pakai). Dikirim otomatis setelah `POST /api/v1/signup`
dan setiap kali email diganti dari Settings (`UpdateAccountProfile`
mengosongkan `email_verified_at` bila email BEDA dari yang tersimpan).
Link berlaku 24 jam (`store.EmailVerificationTTL`), dibuka di
`/verify-email?token=...` — **query string, bukan fragment** seperti reset
password, karena verifikasi email bukan kunci akun (tidak bisa dipakai
mengambil alih apa pun), jadi aman tercatat di log akses.

**Pengingat, bukan gerbang.** Account tetap berfungsi penuh selama belum
diverifikasi — `accountProfileJSON.email_verified` cuma dipakai
menampilkan banner + tombol "Kirim email verifikasi"
(`POST /api/v1/admin/account/email/resend`, dibatasi cooldown yang sama
dengan reset password) di halaman Settings. Validasi FORMAT email (bukan
verifikasi) sudah wajib sejak signup — `isEmailAddress()` dipakai ulang di
`handleSignup` dan `handleAdminUpdateAccount`.

### Password customer: reset lewat email dan ganti dari Settings

`/forgot-password` dan `/reset-password` publik (dikecualikan di
`proxy.ts`), backend `POST /api/v1/password/forgot` dan `/reset`. Butuh
`DASHBOARD_URL` di env backend (alamat publik dashboard untuk link di
email) **dan** SMTP terisi di Vendor Dashboard; kalau salah satu kosong,
lupa password menjawab `503 not_available`.

Keputusan yang tidak boleh dibalik diam-diam:

- **Jawaban "lupa password" identik untuk email terdaftar maupun tidak,
  dan email dikirim di background.** Mengirim di dalam request membuat
  email terdaftar menjawab beberapa detik lebih lambat (percakapan SMTP)
  — cukup untuk mengetahui email mana yang punya akun.
- **Link reset memakai `DASHBOARD_URL` dari config, tidak pernah dari
  header Host/Origin**, dan token di fragment (`#token=`), bukan query
  string — fragment tidak pernah sampai ke server, jadi tidak tercatat di
  log akses Caddy/Next. Halaman reset menghapusnya dari address bar
  setelah dibaca.
- **Token disimpan sebagai SHA-256** (`password_reset_tokens`), berlaku
  30 menit, sekali pakai; permintaan baru membatalkan link sebelumnya;
  maksimal satu email per account per 2 menit (selain batas per IP).
- **Ganti/reset password mencabut semua sesi lama.** Sesi customer
  stateless, jadi `requireAdmin` menolak token yang terbit sebelum
  `accounts.password_changed_at` (waktu terbit diturunkan dari expiry
  token, `auth.SessionIssuedAt`). Ganti password dari Settings
  menerbitkan ulang cookie sesi yang sedang dipakai supaya yang mengganti
  tidak ikut ter-logout.
- **Ganti email wajib password saat ini** — email adalah tujuan link
  reset, sesi yang dicuri tidak boleh cukup untuk memindahkannya.
- Setiap reset/ganti password mengirim pemberitahuan ke pemilik akun
  (`password_changed`, email + Telegram); email reset sendiri
  (`password_reset`) **cuma email**, tidak pernah ke Telegram karena
  berisi link yang setara kunci akun. Keduanya tercatat di
  `notification_log` tanpa isi pesannya.

Grafik tren di Overview (`EventsTrendChart`) pakai Recharts, biaxial —
sumbu kiri jumlah event, sumbu kanan total nominal invoice yang lunas
per hari (`GET /overview` mengembalikan `events.daily[].paid_amount_rp`,
selain `count` yang sudah ada). Bukan SVG tangan sendiri seperti
sebelumnya.

Untuk mengisi dashboard dev lokal dengan banyak data sekaligus (banyak
account, device, invoice/event campuran status, API key, webhook +
riwayat delivery) tanpa klik manual satu-satu, pakai `cmd/seedtool`
(lihat bagian Backend di atas) — HANYA untuk `gopay_dev`, jangan pernah
ke `gopay_test`/UAT/produksi.

Sejak sub-project #2+#6 (spec
docs/superpowers/specs/2026-09-13-landing-signup-design.md): `/` adalah
landing page publik dan `/register` form signup swalayan (plan Starter,
trial 3 hari, langsung aktif tanpa campur tangan vendor) — keduanya di
luar route group `(dashboard)`, dikecualikan dari gerbang sesi di
`proxy.ts`. Overview yang dulu di `/` sekarang di `/overview`.

Perlu account dulu sebelum bisa login — **bukan lagi** `make dev-admin`
(itu sekarang membuat **akun vendor**, dipakai Vendor Dashboard, lihat di
bawah). Untuk dev lokal: jalankan backend (`make run-dev`), buka Vendor
Dashboard, login pakai akun vendor, buat account customer baru dari sana —
username+password awal yang muncul itu yang dipakai login ke
`dashboard/`.

Lima kunci di backend, lima tujuan berbeda, semuanya dihasilkan lewat
`go run ./cmd/devicetool -genkey` tapi **wajib bernilai beda satu sama
lain**: `DEVICE_SECRET_KEY` (enkripsi secret device), `ADMIN_SESSION_KEY`
(tanda tangan cookie sesi customer, `admin_session`), `WEBHOOK_SECRET_KEY`
(enkripsi secret webhook — dipakai ulang tiap kirim payload, beda dari
kunci lain yang cuma menandatangani/memverifikasi), `VENDOR_SESSION_KEY`
(tanda tangan cookie sesi vendor, `vendor_session` — lihat "Sistem akun
multi-tenant" di atas, sesi vendor dan customer tidak boleh pernah
tertukar), `SETTINGS_SECRET_KEY` (enkripsi password SMTP + token bot
Telegram yang diatur vendor dan disimpan di tabel `notification_settings`
— sengaja beda dari `WEBHOOK_SECRET_KEY` supaya rotasi salah satunya tidak
merusak yang lain).

Backend punya empat goroutine latar di `cmd/server/main.go`, sengaja
terpisah karena kadensinya berbeda jauh:

1. **Worker webhook (1 menit)** — mendeteksi invoice yang baru kedaluwarsa
   dan mengeksekusi retry pengiriman yang jatuh tempo.
2. **Pengingat kedaluwarsa (1 jam)** — mengirim email (dan Telegram bila
   customer mengisinya) ke customer yang masa aktifnya tinggal ≤ 7 hari.
   Pengaturan SMTP/Telegram dibaca ulang dari database tiap putaran; selama
   host/port/alamat pengirim SMTP belum diisi di Vendor Dashboard, putaran
   dilewati (dicatat saat status aktif/nonaktif berubah).
3. **Peringatan HP offline (5 menit)** — `internal/devicealert`: mengabari
   customer saat HP bridge tidak mengirim heartbeat > 45 menit, lalu sekali
   lagi saat kembali online. Memakai pengaturan SMTP/Telegram yang sama.
4. **Bot Telegram (long polling)** — `internal/telegrambot`: membaca
   `/start <kode>` dari customer yang menekan "Hubungkan Telegram" dan
   menyimpan chat id-nya. Lihat bagian Telegram di bawah.

Goroutine validasi lisensi 24-jam yang dulu ada (License Server) sudah
dihapus bersama seluruh platform lisensi lama — status akun sekarang dicek
langsung ke database tiap request (`requireActiveAccount`), bukan
diperiksa berkala.

### Pengingat kedaluwarsa ke customer

`internal/notify` (penyusunan pesan + pengiriman SMTP/Telegram) dan
`internal/reminder` (pekerjaan berkalanya). Konfigurasinya (host, port,
username, password, alamat pengirim SMTP, token bot Telegram) **disimpan di
database**, tabel singleton `notification_settings`, diatur di Vendor
Dashboard > Settings — bukan env var. Password dan token dienkripsi
`SETTINGS_SECRET_KEY` dan **tidak pernah dikirim balik lewat API**
(response cuma `smtp_password_set`/`telegram_bot_token_set`); `PUT`
membedakan field tidak dikirim (biarkan), `""` (hapus), dan isi (ganti).
`POST /api/v1/vendor/settings/notifications/test` mengirim pesan uji
memakai pengaturan yang sudah tersimpan.

Pengiriman selalu lewat `notify.Deliver`, yang mencatat **setiap
percobaan per channel** ke tabel `notification_log` (email dan Telegram
baris terpisah, beserta alasan gagal) — dilihat vendor di halaman
Notifications (`GET /api/v1/vendor/notification-log`). Email gagal berarti
Telegram tidak dicoba sama sekali, supaya customer tidak menerima Telegram
berulang tiap putaran selama email terus gagal.

Peringatan HP offline (`internal/devicealert`) memakai pola dedupe yang
sama dengan pengingat: `devices.offline_alert_for` menyimpan NILAI
`heartbeat_at` saat peringatan dikirim. Heartbeat baru → `heartbeat_at`
melewati nilai itu → pemberitahuan "kembali online" lalu kolom
dikosongkan. HP yang heartbeat terakhirnya lebih tua dari 24 jam
(`store.OfflineAlertWindow`) sengaja dilewati, supaya mengaktifkan SMTP
tidak langsung memborbardir customer dengan peringatan HP lama yang sudah
berminggu-minggu tidak dipakai.

Dua keputusan yang tidak boleh dibalik diam-diam:

- **Email jalur utama, Telegram cuma tambahan.** Alamat email pasti
  dimiliki tiap account (kolom wajib); Telegram dihubungkan customer
  sendiri dari halaman `/settings` dashboard mereka dan belum tentu ada. Karena itu
  `NotificationSettings.EmailConfigured()` menuntut SMTP terisi — pengingat yang cuma
  sampai ke sebagian customer lebih berbahaya daripada tidak ada sama
  sekali, karena bikin merasa sudah aman.
- **Dedupe lewat `accounts.expiry_reminder_sent_for`**, yang menyimpan
  NILAI `expires_at` yang pengingatnya sudah dikirim — bukan "kapan
  terakhir kirim". Begitu akun diperpanjang, `expires_at` berubah dan
  pengingat periode berikutnya otomatis terbuka lagi. Kalau diganti jadi
  timestamp biasa, perpanjangan tidak akan pernah mereset apa pun dan
  customer tidak pernah diingatkan lagi setelah pengingat pertama.

**Telegram dihubungkan lewat deep link, bukan chat id yang diketik.** Bot
Telegram dilarang memulai obrolan dengan orang yang belum pernah menekan
Start di bot itu, jadi chat id yang diketik tangan (dulu lewat
`@userinfobot`) selalu gagal `chat not found`. Alurnya sekarang: Settings →
"Hubungkan Telegram" → `POST /api/v1/admin/account/telegram/link`
(kode sekali pakai 15 menit, disimpan SHA-256 di `telegram_link_codes`) →
`t.me/<bot>?start=<kode>` → customer tekan Start → `internal/telegrambot`
menyimpan chat id dari pesan itu. Dashboard menunggu lewat polling
`GET /admin/account`.

- **Long polling (`getUpdates`), bukan webhook** — tidak butuh URL publik,
  jadi sama di VPS dan laptop. Konsekuensinya **satu token bot hanya boleh
  dibaca satu backend**: backend dev di laptop wajib memakai bot terpisah
  dari produksi, kalau tidak keduanya saling memutus (`409 Conflict`,
  dicatat sekali di log sebagai `telegram.ErrConflict`).
- Posisi `getUpdates` disimpan di
  `notification_settings.telegram_update_offset`, direset ke 0 saat token
  bot diganti. Tanpa itu, setiap restart memproses ulang `/start` 24 jam
  terakhir.

Ambangnya 7 hari (`reminder.DefaultWithinDays`) — angka ini tetap ada dan
tidak berubah. Yang berubah (2026-09-14, permintaan eksplisit Akbar):
`store.WarningThresholdDays` (30 hari) beserta status pasif "expiring"
yang dulu dipakai dashboard **dicabut total**, bukan sekadar
disembunyikan. `Account.DerivedStatus` sekarang cuma 4 nilai
(`active`/`expired`/`suspended`/`revoked`) — account tetap "active" sampai
PERSIS melewati `expires_at`, tidak lagi ditandai "akan berakhir"
berminggu-minggu sebelumnya. Alasan pencabutannya: sisa waktu yang masih
lama terasa membingungkan ditandai status peringatan, dan itu bukan
keadaan yang butuh tindakan. Pengingat aktif ke customer (email/Telegram,
7 hari sebelum benar-benar habis) tetap satu-satunya jalur "hampir habis"
yang dipertahankan — jangan menambahkan kembali status peringatan pasif
di dashboard tanpa diminta ulang.

### Logs (riwayat aktivitas akun)

`account_activity_log` (migrasi 00016) mencatat 8 jenis aktivitas:
`login_success`, `login_failed`, `password_changed`, `password_reset`,
`api_key_created`, `api_key_revoked`, `device_added`, `device_deleted` —
lewat `a.logActivity(r, accountID, action, metadata)`
(`internal/httpapi/activity_log.go`), dipanggil dari handler yang
bersangkutan setelah aksinya berhasil. Gagal mencatat TIDAK menggagalkan
aksinya (pola sama dengan `LogAudit`/`notify.Deliver`).

`GET /api/v1/admin/activity` SELALU di-scope satu account dari sesi —
**beda** dari `GET /vendor/notification-log` (Vendor Dashboard, sengaja
lintas account). Halaman `/logs` di Customer Dashboard, dulu "Segera" di
sidebar, sekarang aktif.

Lingkupnya sengaja BEDA dari "Logs" di `docs/dashboard-spec.md` (draf lama
membayangkan log teknis kategori SYSTEM/DEVICE/EVENT/WEBHOOK/LICENSE/AUTH
untuk troubleshooting) — yang dibangun ini log KEAMANAN akun untuk
customer sendiri ("curiga ada akses yang bukan dari saya"), permintaan
langsung Akbar yang lebih berguna daripada log teknis mentah.

### Vendor: kirim link reset password untuk customer yang terkunci

`POST /api/v1/vendor/accounts/{id}/send-password-reset` (tombol "Kirim
link reset password" di halaman detail account) memakai alur reset yang
sama (`store.CreatePasswordResetToken` + `notify.PasswordResetEmail`),
tapi endpoint TERPISAH dari `handleForgotPassword` — di sini vendor sudah
login dan memilih account-nya sendiri dari daftar, jadi tidak ada risiko
enumerasi email dan errornya boleh spesifik (`not_available`,
`account_revoked`), bukan disamarkan jadi "sukses" untuk semua kasus
seperti endpoint publik.

### Vendor Dashboard

Next.js terpisah total dari `dashboard/` (folder `vendor-dashboard/`) —
cuma Akbar yang pakai, sesi `vendor_session`, manggil `/api/v1/vendor/*`
di backend utama. Sama scaffold shadcn/ui + token warna dengan
`dashboard/` (termasuk token `--sidebar-*` yang ternyata sudah ada di
`globals.css` sejak awal tapi baru dipakai belakangan), jadi banyak
komponen (`Sheet`, `StatCard`, pola `AppShell` sidebar collapsible)
sengaja disalin dari `dashboard/` alih-alih dibangun ulang dari nol.

```bash
cd vendor-dashboard
npm install
cp .env.local.example .env.local
npm run dev                        # Akbar yang menjalankan
```

Halaman:

- **Dashboard** (`/`, tujuan redirect setelah login) — ringkasan lintas
  SEMUA account: total/status/account baru minggu ini/total device/total
  pendapatan seumur hidup, grafik biaxial 14 hari (account baru vs
  nominal lunas, Recharts, pola sama dengan Overview `dashboard/`), dan
  5 account terbaru. Sumber datanya `GET /api/v1/vendor/overview` —
  **beda** dari `GET /overview` customer yang selalu di-scope satu
  account dari sesi; endpoint vendor ini sengaja lintas account karena
  itu memang tugas vendor.
- **Accounts** (`/accounts`, dulu di `/`) — daftar semua account + form
  buat account baru + link ke halaman detail (`/accounts/{id}`: renew,
  suspend, revoke). Dropdown Plan diisi dinamis dari `GET /vendor/plans`
  (lihat halaman Plans di bawah), bukan lagi tiga nilai tetap di kode.
- **Plans** (`/plans`) — kelola paket: nama, kuota device (`max_devices`,
  -1 = unlimited), harga (`price_label`/`price_period`, teks bebas —
  kosong berarti section Harga TIDAK menampilkan baris harga sama sekali,
  bukan "Rp 0"), deskripsi, daftar fitur, `highlighted`
  ("Direkomendasikan"), `visible`, dan urutan tampil (panah naik/turun,
  `POST .../move`). Tabel `plans` (migrasi 00017) — **satu sumber
  kebenaran ganda**: kuota device saat account dibuat/diganti plan
  (`GetPlanByName`, menggantikan `planPresets` yang dulu hardcode di
  `vendor_accounts.go`) DAN teks yang tampil di section Harga landing page
  publik (`GET /api/v1/pricing-plans`, tanpa auth, cuma plan
  `visible=true`). `accounts.plan` TETAP teks bebas (bukan foreign key) —
  menghapus/mengganti sebuah plan TIDAK mengubah account yang sudah
  memakai namanya, karena `max_devices` sudah tersalin ke kolom account
  itu sendiri sejak dibuat/terakhir diganti (lihat komentar
  `store.DeletePlan`). Signup swalayan (`signupPlan = "Starter"` di
  `signup.go`) mengambil kuota trial dari baris bernama persis "Starter"
  di tabel ini — **jangan mengganti nama atau menghapus plan itu tanpa
  memperbarui konstanta itu juga**, kalau tidak signup publik berhenti
  (menjawab `503 not_available`, bukan diam-diam salah kuota).
- **Audit Log** (`/audit-log`) — riwayat aksi vendor (`LogAudit`).
- **Transactions** (`/transactions`) dan **Webhooks** (`/webhooks`) —
  invoice dan riwayat webhook delivery lintas SEMUA account, bisa
  diekspor CSV.
- **Notifications** (`/notifications`) — riwayat email/Telegram yang
  dikirim ke customer (pengingat kedaluwarsa, HP offline/online, pesan
  uji), per channel, dengan alasan gagal.
- **Settings** (`/settings`) — pengaturan SMTP + bot Telegram untuk
  notifikasi ke customer (lihat "Pengingat kedaluwarsa ke customer" di
  atas).
- **Profile** (`/profile`, tidak di sidebar) — ganti password vendor.
  Dibuka dari menu avatar di header (`user-menu.tsx`), yang juga memuat
  Keluar dengan modal konfirmasi. Nama di avatar dari
  `GET /api/v1/vendor/me`.

Warna badge status account (`active`/`expired`/`suspended`/`revoked` —
tidak ada lagi "expiring", lihat §"Pengingat kedaluwarsa ke customer")
disatukan di `src/lib/account-status.ts`, dipakai bersama oleh halaman
Accounts dan Dashboard supaya tidak diam-diam berbeda. Kartu ringkasan di
Dashboard Vendor yang dulu "Akan Berakhir" sekarang "Kedaluwarsa"
(`accounts.expired`, link `?status=expired`).
