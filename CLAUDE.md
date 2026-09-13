# GoPay Notification Bridge

Bridge notifikasi GoPay dari HP Android ke backend, sebagai fondasi self-hosted payment gateway.

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
5. [`docs/license-spec.md`](docs/license-spec.md) — spec bisnis lisensi (Self-Hosted + Annual License), paling berwenang untuk sub-project 5
6. [`docs/superpowers/specs/2026-09-13-online-license-platform-design.md`](docs/superpowers/specs/2026-09-13-online-license-platform-design.md) — potongan MVP dari spec #5 di atas, dipilih dan diimplementasikan untuk sub-project 5. [`2026-09-13-license-system-design.md`](docs/superpowers/specs/2026-09-13-license-system-design.md) (versi offline) SUPERSEDED oleh dokumen ini.
7. [`docs/api-contract.md`](docs/api-contract.md) — kontrak antara backend dan Android
8. [`docs/dashboard-spec.md`](docs/dashboard-spec.md) — rancangan dashboard penuh. MVP yang sudah dibangun ([`dashboard/README.md`](dashboard/README.md)) hanya subset-nya; jangan menganggap seluruh isi dokumen ini sudah ada.
9. [`docs/prd.md`](docs/prd.md), [`docs/detail-project.md`](docs/detail-project.md) — dokumen awal

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
| 5 | Platform lisensi online (License Server, Vendor Dashboard, `internal/licenseclient`, `requireLicense`) | implementasi selesai, menunggu deploy + `make test` |

Sub-project 3 dipecah jadi 4 fase, urutan dan rinciannya ada di spec #2–#4
di atas. Seluruhnya sudah selesai — kalau ada permintaan fitur baru untuk
gateway ini, itu perluasan di luar keempat fase itu, bukan bagian dari
salah satunya.

Dashboard mencakup halaman yang datanya sungguhan ada: Overview, Devices,
Events, Transactions, API Keys, Webhooks, Exceptions, License. Settings dan
Logs ditampilkan di sidebar sebagai "Segera" (non-aktif).

**Dashboard ini untuk customer (pemilik instalasi), bukan untuk vendor.**
Model self-hosted + annual license berarti tiap customer men-deploy backend
dan dashboard-nya sendiri di server mereka sendiri — vendor tidak pernah
menyentuh data atau infrastruktur mereka. Satu instalasi = satu backend =
satu dashboard = satu customer, bukan satu dashboard multi-tenant milik
vendor untuk memantau semua customer sekaligus. Akun admin yang dibuat lewat
`make dev-admin` adalah akun milik customer itu sendiri.

Konsekuensinya untuk copywriting dan desain dashboard: jangan menyebut
istilah arsitektur internal ("backend", "database", nama service, dsb) di
teks yang tampil ke pengguna — itu bukan urusan customer, dan membocorkannya
bikin produk terasa seperti tool developer, bukan produk jadi. Field API
boleh tetap bernama `backend`/`database` (kontrak sudah ada, sub-project 3
akan menambah komponen lain ke sana), tapi label dan copy yang ditampilkan
di halaman harus diringkas jadi bahasa yang netral, mis. "Status Layanan" /
"Aktif", bukan "Backend: operational".

Sistem lisensi (sub-project 5) sekarang online: License Server + Vendor
Dashboard adalah komponen vendor yang sebelumnya sengaja ditunda ("baru
masuk akal begitu customer lebih dari satu") — sudah dibangun lebih awal
dari rencana karena `docs/license-spec.md` mensyaratkan validasi online.
Detail lengkap di bagian "Sistem lisensi" di bawah.

## Sistem lisensi

Spec lengkap: [`docs/superpowers/specs/2026-09-13-online-license-platform-design.md`](docs/superpowers/specs/2026-09-13-online-license-platform-design.md)
(mengikuti [`docs/license-spec.md`](docs/license-spec.md), dokumen bisnis
otoritatif untuk model Self-Hosted + Annual License). Menggantikan versi
offline murni yang sempat dibangun sebelumnya
([`2026-09-13-license-system-design.md`](docs/superpowers/specs/2026-09-13-license-system-design.md),
SUPERSEDED) — `internal/licensecheck` dipakai ulang, cuma yang
menandatangani sekarang License Server, bukan CLI manual.

**Tiga komponen, tiga kepemilikan berbeda:**

- `backend/cmd/licenseserver` + `backend/internal/licenseserver` — License
  Server, **milik vendor (Akbar)**, database sendiri (`gopay_license`),
  di-deploy terpisah di `license.whuzpay.com`. Satu Go module dengan
  `backend/` (supaya bisa memakai ulang `internal/licensecheck` untuk
  menandatangani), tapi database dan proses run-time-nya terpisah total
  dari instalasi customer manapun — termasuk instalasi Akbar sendiri
  sebagai customer pertamanya di `whuzpay.com`.
- `vendor-dashboard/` — Next.js baru, **cuma Akbar yang pakai**, kelola
  customer/license/installation/audit log. Terpisah total dari
  `dashboard/` (dashboard customer) — jangan pernah dicampur.
- `backend/internal/licenseclient` + `backend/internal/licensecheck` (baris
  verifikasi) — **milik tiap instalasi customer**, memvalidasi ke License
  Server tiap 24 jam (goroutine terpisah dari ticker webhook di
  `cmd/server/main.go`), hasil signed di-cache lokal (`license-state.lic`)
  dengan grace period 7 hari kalau License Server tak terjangkau.

Aktivasi lewat `.env`: `LICENSE_KEY` + `ENVIRONMENT` (`production`/`uat`),
lalu restart — **bukan** form di dashboard, konsisten dengan pola seluruh
kunci lain di proyek ini. Tanpa lisensi aktif/akan-berakhir,
`requireLicense` menolak `402` seluruh endpoint device/admin/API key —
hanya login dan halaman dashboard `/license` (read-only) yang tetap bisa
diakses. Cara deploy License Server + Vendor Dashboard dan menerbitkan
lisensi customer ada di `backend/deploy/README.md` §"Lisensi".

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

`make db-down` menghapus volume Docker — **`gopay_dev` ikut hilang**, bukan
hanya data test.

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
API Keys, Webhooks, Exceptions, License. Sisanya (Settings, Logs)
ditampilkan di sidebar sebagai "Segera", non-aktif.

Perlu akun admin dulu sebelum bisa login: `cd backend && make dev-admin`.

Tiga kunci di backend, tiga tujuan berbeda, semuanya dihasilkan lewat
`go run ./cmd/devicetool -genkey` tapi **wajib bernilai beda satu sama
lain**: `DEVICE_SECRET_KEY` (enkripsi secret device), `ADMIN_SESSION_KEY`
(tanda tangan cookie sesi admin), `WEBHOOK_SECRET_KEY` (enkripsi secret
webhook — dipakai ulang tiap kirim payload, beda dari dua kunci lain yang
cuma menandatangani/memverifikasi). Kunci penanda tangan lisensi (Ed25519)
adalah pasangan terpisah lagi milik **License Server**, bukan bagian dari
tiga kunci backend customer ini — lihat "Sistem lisensi" di atas.

Backend punya satu goroutine berkala (`time.Ticker`, 1 menit, di
`cmd/server/main.go`) yang memproses webhook — mendeteksi invoice yang baru
kedaluwarsa dan mengeksekusi retry pengiriman yang jatuh tempo. Ini
satu-satunya proses latar belakang di backend saat ini; kalau menambah yang
serupa nanti, pertimbangkan apakah masih masuk akal digabung ke ticker yang
sama atau butuh ticker terpisah. Lisensi tidak ikut ticker ini — dimuat
sekali saat start (lihat "Sistem lisensi"), bukan diperiksa berkala.
