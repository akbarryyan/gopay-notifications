# Distribusi APK Android Bridge — Design Spec

## 1. Latar belakang

`whuzpay-pg` (payment gateway aggregator, folder terpisah di repo ini) baru
saja selesai migrasi provider dari Cashi ke gopay-notifications dan
diverifikasi sungguhan di produksi (`pg.whuzpay.com`) — lihat
`docs/superpowers/specs/2026-09-15-whuzpay-pg-gopay-provider-design.md` dan
`docs/qa/qa-report.md` §28. Langkah berikutnya yang disepakati Akbar: bikin
whuzpay-pg benar-benar bisa dipakai merchant lain, bukan cuma Akbar sendiri.

Saat brainstorming ini, ditemukan satu fakta arsitektur yang menentukan
seluruh arah kerja berikutnya: QRIS yang ditampilkan ke pembeli di setiap
invoice adalah **QRIS statis milik akun gopay-notifications merchant itu
sendiri** (`internal/store/qris_image.go` di `backend/`, migrasi 00018).
Dana dari pembayaran QRIS itu **langsung masuk ke rekening/GoPay milik
merchant saat itu juga** — whuzpay-pg maupun gopay-notifications tidak
pernah memegang dana pelanggan. Ini murni lapisan orkestrasi (buat invoice,
deteksi notifikasi lewat HP, cocokkan, update status), bukan payment
gateway custodial seperti Midtrans/Xendit.

Konsekuensinya: **payout/settlement tidak relevan** (tidak ada dana yang
perlu dicairkan), tapi tiap merchant wajib (1) punya akun gopay-notifications
sendiri + upload QRIS mereka sendiri, dan (2) menjalankan HP fisik terpisah
dengan aplikasi Android bridge aktif memantau notifikasi GoPay mereka
sendiri. Model custodial sungguhan (dana ditampung lalu dicairkan berkala)
butuh status PJP/acquirer berlisensi dari Bank Indonesia — di luar scope
development, jadi **sengaja tidak dikejar**. Keputusan yang diambil: terima
model "bring your own device + akun GoPay" sebagai identitas produk, dan
fokus kerja ke membuat proses setup itu semudah mungkin.

Pekerjaan itu dipecah jadi dua sub-project independen (masing-masing bisa
selesai dan diuji sendiri):

1. **Distribusi APK** (spec ini) — prasyarat murni teknis: sampai sekarang
   tidak ada cara bagi siapa pun selain Akbar untuk memasang aplikasi
   Android bridge sama sekali (cuma pernah dibangun via `expo run:android`
   di laptop Akbar dengan toolchain dev lengkap).
2. **Penyederhanaan alur onboarding** (spec terpisah, belum ditulis) —
   menyatukan langkah-langkah yang sekarang tersebar di dua dashboard
   (gopay-notifications + whuzpay-pg) jadi satu panduan yang jelas. Baru
   berguna setelah (1) selesai.

## 2. Tujuan

Merchant mana pun bisa mengunduh dan memasang aplikasi Android bridge
(varian production, `id.akbarryyan.gopaybridge`) tanpa butuh laptop atau
toolchain development apa pun milik Akbar.

## 3. Di luar cakupan

- Play Store — ditunda; app yang minta akses notifikasi + terkait
  pembayaran berisiko tinggi kena penolakan/kebijakan khusus Google, dan
  prosesnya jauh lebih lama dari kebutuhan sekarang.
- Auto-update di dalam aplikasi (cek versi baru, download otomatis) — YAGNI
  untuk rilis pertama; update dilakukan manual dengan menimpa file APK di
  server dan mengumumkan ke merchant.
- Sub-project 2 (penyederhanaan onboarding) — spec terpisah.
- iOS — GoPay Notification Bridge sepenuhnya bergantung pada
  `NotificationListenerService` Android; tidak ada padanannya yang legal di
  iOS (Apple tidak mengizinkan membaca notifikasi app lain).

## 4. Desain

### 4.1. Build profile EAS

File baru `mobile/eas.json`:

```json
{
  "cli": {
    "version": ">= 16.0.0",
    "appVersionSource": "remote"
  },
  "build": {
    "production": {
      "env": {
        "APP_VARIANT": "production"
      },
      "android": {
        "buildType": "apk"
      },
      "distribution": "internal",
      "autoIncrement": true
    }
  }
}
```

- `distribution: "internal"` — hasilnya link unduh langsung dari EAS,
  bukan disubmit ke Play Store.
- `android.buildType: "apk"` — APK biasa yang bisa di-sideload, bukan
  `.aab` yang cuma diterima Play Store.
- `env.APP_VARIANT: "production"` — memastikan hasilnya selalu varian
  production (`id.akbarryyan.gopaybridge`, nama "GoPay Bridge"), tidak
  bergantung pada env shell siapa pun yang menjalankan `eas build`.
- `appVersionSource: "remote"` + `autoIncrement: true` — EAS yang mengurus
  `versionCode` Android naik otomatis tiap build; tidak perlu diingat
  manual.
- Keystore penandatanganan APK: digenerate dan disimpan EAS otomatis di
  akun Expo Akbar saat build pertama. **Wajib tetap sama untuk semua build
  berikutnya** — kalau berganti, HP yang sudah pasang versi lama tidak bisa
  menerima update APK baru tanpa uninstall dulu. Karena dikelola otomatis
  oleh EAS per project, ini aman selama project EAS yang sama terus dipakai
  untuk app ini.

### 4.2. `expo-dev-client` jadi kondisional per varian

`mobile/app.config.ts` saat ini memasukkan plugin `'expo-dev-client'` untuk
SEMUA varian (baris 60-61), termasuk production. Profile EAS `production`
tetap menghasilkan build standalone (tidak butuh Metro/dev server
menyala) terlepas dari ini, tapi kalau paketnya tetap ikut terbundel, ada
dev-menu (bisa terpicu gesture shake) yang menempel di aplikasi yang
dipegang merchant sungguhan — tidak perlu dan berpotensi membingungkan
pengguna non-teknis.

Perubahan: `expo-dev-client` cuma masuk plugin list untuk varian
`development` dan `uat`, mengikuti pola yang sudah ada di file yang sama
untuk `cleartext`/`permissions` per varian (bukan pola baru).

### 4.3. Hosting APK & halaman unduh

APK hasil `eas build` **tidak** dirujuk langsung dari link expo.dev (bisa
berubah/kadaluwarsa di luar kendali kita). Alurnya:

1. Akbar menjalankan `eas build --platform android --profile production`.
2. Setelah selesai, Akbar mengunduh APK-nya sekali dari halaman hasil build.
3. File itu ditaruh di `dashboard/public/downloads/gopay-bridge.apk`
   (Next.js Customer Dashboard gopay-notifications, sudah live di
   `whuzpay.com`) — URL-nya jadi permanen:
   `https://whuzpay.com/downloads/gopay-bridge.apk`.
4. Deploy ulang `dashboard/` seperti proses update rutin biasa
   (`backend/deploy/README.md`).

Halaman baru **"Unduh Aplikasi Android"** di `dashboard/src/app/download-app/page.tsx`
— route publik di luar route group `(dashboard)`, ditambahkan ke daftar
pengecualian gerbang sesi di `proxy.ts`, mengikuti pola `/register` yang
sudah dikecualikan di sana. Isinya:

- Penjelasan singkat: aplikasi ini yang memantau notifikasi GoPay di HP dan
  melaporkannya ke backend.
- Syarat minimum: Android 8.0 (API 26) ke atas.
- Tombol unduh besar → `https://whuzpay.com/downloads/gopay-bridge.apk`.
- Instruksi singkat pasang manual: aktifkan "Izinkan sumber tidak dikenal"
  saat memasang APK yang diunduh dari browser, buka aplikasinya, izinkan
  Notification Access saat diminta.

**Halaman ini publik, tanpa perlu login** — merchant baru bisa mengunduh
aplikasinya duluan sebelum atau sambil membuat akun gopay-notifications,
tidak perlu login dulu baru bisa mengunduh.

Update ke versi baru nanti: ulangi langkah 1-4 di atas (build → unduh →
timpa file → deploy ulang). Tidak ada mekanisme notifikasi
"ada versi baru" ke pengguna existing di rilis pertama ini (lihat §3).

## 5. Pengujian

**Otomatis (bisa Claude jalankan):**
- `npx tsc --noEmit` di `mobile/` setelah `app.config.ts` diubah.
- Review `eas.json` terhadap skema resmi (`cli`/`build` valid, tidak ada
  typo field).

**Manual, `NEEDS-DEVICE` (Akbar yang jalankan dan tempel buktinya):**
1. `eas build --platform android --profile production` — build selesai
   tanpa error, hasilnya APK (bukan AAB).
2. Pasang APK itu ke HP yang **belum pernah** menjalankan
   `expo run:android`/Metro untuk aplikasi ini (membuktikan benar-benar
   standalone, bukan kebetulan masih terhubung ke laptop Akbar).
3. Buka aplikasinya — harus langsung masuk ke layar Settings/aplikasi itu
   sendiri, BUKAN layar "development client" yang meminta scan
   QR/connect ke dev server.
4. Alur penuh: pasang → izinkan Notification Access → scan QR pairing dari
   Customer Dashboard → Test Connection sukses.
5. Halaman `https://whuzpay.com/download-app` bisa dibuka tanpa login,
   tombol unduh benar-benar mengunduh file `.apk` yang valid dari
   `https://whuzpay.com/downloads/gopay-bridge.apk` (bisa dipasang, bukan
   file korup/HTML error page).

## 6. Keputusan yang sengaja diambil (jangan diubah diam-diam)

- **Tidak ada payout/settlement** — dana tidak pernah ditampung whuzpay-pg
  atau gopay-notifications; ini properti arsitektur, bukan fitur yang
  belum dibangun. Lihat §1.
- **Tidak ada gerbang approval merchant baru** — merchant tetap langsung
  aktif (`IsActive: true`) begitu daftar, sama seperti sekarang. Keputusan
  eksplisit Akbar saat brainstorming: karena tidak ada custody dana, risiko
  fraud ke whuzpay-pg sendiri rendah; approval gate bisa ditambahkan nanti
  kalau memang jadi masalah nyata, bukan dikerjakan preemptif sekarang.
- **Distribusi APK langsung, bukan Play Store** — keputusan sadar demi
  kecepatan, bukan kelalaian. Play Store bisa dipertimbangkan lagi nanti
  kalau volume merchant sudah cukup besar untuk membenarkan proses
  reviewnya.
