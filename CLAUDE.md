# GoPay Notification Bridge

Bridge notifikasi GoPay dari HP Android ke backend, sebagai fondasi self-hosted payment gateway.

## Pindah AI

[`docs/ai-handoff-prompt.md`](docs/ai-handoff-prompt.md) memuat prompt siap
tempel untuk AI lain, berikut daftar jebakan yang sudah pernah memakan waktu di
repo ini. Perbarui bagian "Keadaan saat ini" di sana setiap milestone selesai.

## Dokumen sumber

Urutan kewenangan bila terjadi perbedaan:

1. [`docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md`](docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md) — spec yang disetujui, paling berwenang
2. [`docs/api-contract.md`](docs/api-contract.md) — kontrak antara backend dan Android
3. [`docs/dashboard-spec.md`](docs/dashboard-spec.md) — rancangan dashboard penuh. MVP yang sudah dibangun ([`dashboard/README.md`](dashboard/README.md)) hanya subset kecilnya; jangan menganggap seluruh isi dokumen ini sudah ada.
4. [`docs/prd.md`](docs/prd.md), [`docs/detail-project.md`](docs/detail-project.md) — dokumen awal

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
| 3 | Gateway: invoice, nominal unik, matching, webhook | ditunda |
| 4 | Dashboard admin (Next.js) — Overview, Devices, Events | sedang dikerjakan |

Jangan membangun apa pun dari sub-project 3 kecuali diminta. Invoice, matching, dan webhook berada di luar cakupan saat ini.

Dashboard sengaja hanya mencakup tiga halaman yang datanya sungguhan ada.
Transactions, Webhooks, License, Settings ditampilkan di sidebar sebagai
"Segera" (non-aktif) — bukan dibangun sebagai halaman kosong yang menebak
bentuk data sub-project 3 sebelum sub-project itu sendiri ada.

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

Hanya tiga halaman yang datanya sungguhan ada: Overview, Devices, Events.
Sisanya (Transactions, Webhooks, License, Settings) ditampilkan di sidebar
sebagai "Segera", non-aktif — jangan membangun halaman untuk data sub-project
3 yang bentuknya belum pasti.

Perlu akun admin dulu sebelum bisa login: `cd backend && make dev-admin`.

`ADMIN_SESSION_KEY` di backend wajib beda dari `DEVICE_SECRET_KEY` — keduanya
sama-sama dihasilkan lewat `go run ./cmd/devicetool -genkey`, jangan memakai
nilai yang sama untuk keduanya.
