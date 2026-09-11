# GoPay Notification Bridge

Bridge notifikasi GoPay dari HP Android ke backend, sebagai fondasi self-hosted payment gateway.

## Dokumen sumber

Urutan kewenangan bila terjadi perbedaan:

1. [`docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md`](docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md) — spec yang disetujui, paling berwenang
2. [`docs/api-contract.md`](docs/api-contract.md) — kontrak antara backend dan Android
3. [`docs/prd.md`](docs/prd.md), [`docs/detail-project.md`](docs/detail-project.md) — dokumen awal

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

Jangan membangun apa pun dari sub-project 3 kecuali diminta. Invoice, matching, dan webhook berada di luar cakupan saat ini.

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

Untuk laporan QA, butir yang menuntut build atau server ditandai `NEEDS-DEVICE`
sampai Akbar menempelkan buktinya — tidak pernah `PASS` berdasarkan penalaran.

## Perintah

### Backend

```bash
cd backend
make db-up            # Postgres di :5433 lewat docker compose
make migrate          # goose up
make test             # db-up + migrate + go test ./... -p 1
```

`-p 1` wajib. Paket `store` dan `httpapi` sama-sama `TRUNCATE` database test yang
sama, dan Go menjalankan paket secara paralel — tanpa `-p 1` keduanya saling
menghapus data dan gagal secara acak, padahal sendiri-sendiri lulus.

Menjalankan server secara lokal:

```bash
cd backend
export DATABASE_URL='postgres://gopay:gopay@localhost:5433/gopay_test?sslmode=disable'
export DEVICE_SECRET_KEY=$(go run ./cmd/devicetool -genkey)
export LISTEN_ADDR=127.0.0.1:8098
go run ./cmd/server
```

Jebakan yang sudah pernah kena: bila server versi lama masih memegang port,
`go build -o` ke path binary yang sedang berjalan gagal dengan *text file busy*,
dan endpoint baru membalas 404 karena yang melayani adalah binary lama. Matikan
berdasarkan port, bukan pola nama:

```bash
pid=$(ss -ltnpH 'sport = :8098' | grep -oP 'pid=\K[0-9]+' | head -1) && kill "$pid"
```

Membuat device:

```bash
go run ./cmd/devicetool -genkey                 # cetak DEVICE_SECRET_KEY
go run ./cmd/devicetool -name "HP GoPay Utama"  # cetak Device ID + Secret
```

### Mobile

Selalu dengan `APP_VARIANT=development` dan `ANDROID_SERIAL`.

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

npx expo prebuild --platform android --clean   # regenerate android/ dari config plugin
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
