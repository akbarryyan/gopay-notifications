# Prompt Handoff untuk AI Lain

Tempel seluruh isi blok di bawah ke AI baru sebagai pesan pertama. Ia ditulis
agar berlaku di alat apa pun, bukan hanya Claude Code.

Perbarui bagian **Keadaan saat ini** setiap kali ada milestone selesai —
sisanya jarang berubah.

---

```text
Kamu akan melanjutkan pekerjaan pada repo ini. Sebelum menulis kode apa pun,
baca dan pahami dulu. Jangan menebak; repo ini punya banyak keputusan yang
sengaja menyimpang dari bawaan, dan semuanya beralasan.

## Apa yang dibangun

Payment Notification Bridge — "payment gateway tanpa perlu mendaftar ke
Midtrans". Aplikasi Android membaca notifikasi pembayaran GoPay Merchant lewat
NotificationListenerService, mengubahnya jadi event terstruktur, dan
mengirimkannya ber-HMAC ke backend Go milik merchant sendiri. Backend
mencocokkannya ke invoice lalu mengirim webhook ke website merchant.

Model bisnis: self-hosted + lisensi tahunan. Merchant menjalankan backend dan
HP-nya sendiri; uang tidak pernah lewat rekening vendor.

## Urutan membaca, dan kewenangannya

Bila dua dokumen berbeda, yang lebih atas menang:

1. CLAUDE.md — aturan kerja, perintah, dan keputusan yang tidak boleh
   dilanggar diam-diam. Baca ini lebih dulu, selalu.
2. docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md —
   spec yang disetujui. §9 memuat daftar penyimpangan dari dokumen awal.
   JANGAN "memperbaiki" implementasi agar kembali sesuai PRD tanpa memeriksa
   daftar itu; setiap penyimpangan punya alasan tertulis.
3. docs/api-contract.md — kontrak antara backend dan Android. Kalau kode dan
   dokumen ini berbeda, dokumen ini yang benar.
4. docs/qa/qa-report.md — satu-satunya sumber kebenaran tentang APA YANG SUDAH
   TERBUKTI. Jangan percaya klaim di dokumen lain tanpa melihat baris PASS-nya
   di sini beserta buktinya.
5. docs/qa/qa-rules.md — aturan QA. Mengikat.
6. docs/self-hosted-annual-license.md, docs/dashboard-spec.md — arah produk.
   Keduanya draft dan memuat beberapa hal yang bertabrakan dengan arsitektur;
   catatan tabrakannya ada di riwayat commit.
7. docs/prd.md, docs/detail-project.md — dokumen awal. ARSIP. Jangan disunting;
   ia jejak asal keputusan.

## Aturan kerja yang wajib dipatuhi

Pemilik repo (Akbar) menjalankan sendiri semua build dan server. Kamu MENULIS
kode, lalu MEMBERIKAN perintahnya, lalu MENUNGGU keluarannya ditempelkan.
Jangan menjalankan: expo run:android, gradlew, npm run, go run ./cmd/server,
expo start, docker compose up.

Perintah baca-saja yang cepat boleh kamu jalankan sendiri: git status, grep,
adb devices, adb shell pm list packages, adb logcat, go build, go vet, curl ke
server yang sudah berjalan, dan query SELECT ke database.

Untuk laporan QA: PASS menuntut keluaran perintah yang DITEMPEL, bukan
pernyataan bahwa sesuatu berfungsi. Hal yang hanya dapat diuji di perangkat
atau VPS ditandai NEEDS-DEVICE, tidak pernah PASS berdasarkan penalaran. Butir
yang belum tersentuh ditandai PENDING dan tidak dihapus dari tabel.

Setiap milestone selesai, perbarui docs/qa/qa-report.md sebelum menyatakannya
selesai.

## Keputusan arsitektur yang tidak boleh dilanggar diam-diam

Kotlin memiliki seluruh pipeline data; TypeScript hanya UI. Runtime JavaScript
mati saat aplikasi tertutup, jadi pengiriman HTTP tidak boleh ditulis di
TypeScript. Native adalah satu-satunya pemilik database.

Parsing nominal di HP bersifat display-only. Backend melakukan ekstraksi
otoritatifnya sendiri dari teks mentah.

Arah transaksi ditentukan backend lewat allowlist judul, bukan blocklist. Yang
tidak dikenali ditolak, bukan ditebak. Salah di sini berarti order ditandai
lunas padahal tidak ada uang masuk.

event_id = sha256(packageName | title | text | when)[:32]. Bukan
notificationKey (konstan di GoPay) dan bukan postTime (berubah tiap repost).

Tanda tangan HMAC dihitung dari byte mentah body dan diverifikasi SEBELUM
decode JSON. Bandingkan dengan hmac.Equal, jangan ==.

Nama connector tidak boleh masuk URL. Rutenya POST /api/v1/events; pembeda
sumber hanya field source, divalidasi terhadap internal/connector.

## Fakta lapangan yang mahal ditemukan

Sumber pembayaran adalah com.gojek.gopaymerchant. JANGAN memantau
com.gojek.gopay bersamaan — kedua aplikasi melaporkan pembayaran yang SAMA
dengan teks identik, dan karena packageName ikut jadi bahan event_id, satu
pembayaran akan menghasilkan dua event dan terhitung dua kali. Idempotency
tidak menolong.

Judul notifikasi pembayaran: "Pembayaran QRIS statis diterima". Teksnya
"Rp 1 di <nama merchant>." — perhatikan spasi setelah Rp, dan tidak ada nomor
referensi transaksi sama sekali. Itu yang mengunci pendekatan nominal unik.

Aplikasi merchant juga memasang notifikasi dengan title, text, dan bigText
ketiganya null — ringkasan grup. Dilewati tanpa disimpan.

Notifikasi transaksi datang lewat channel "Promotions and Marketing". Jangan
pernah menyarankan mematikan channel notifikasi apa pun milik aplikasi sumber;
mematikannya mematikan seluruh sistem tanpa gejala.

Perangkat target OPPO CPH2365, Android 13, ColorOS — pembunuh background
process paling agresif. Setiap klaim soal ketahanan service harus diuji di
sana, tidak boleh diasumsikan dari perilaku Android standar.

## Jebakan yang sudah pernah memakan waktu

Jangan ulangi ini:

go test ./... menjalankan paket secara PARALEL, dan paket store serta httpapi
sama-sama TRUNCATE database test yang sama. Wajib -p 1. Tanpa itu gagal secara
acak padahal sendiri-sendiri lulus.

BUILD SUCCESSFUL dari gradle TIDAK berarti test jalan. Baca
mobile/modules/gopay-listener/android/build/test-results/testDebugUnitTest/*.xml
dan periksa atribut tests, failures, errors, beserta waktu berkasnya.

Room 2.6.x tidak dapat membaca metadata Kotlin 2.1. Wajib 2.8.x. Jangan
diturunkan.

Plugin KSP tidak ada di classpath buildscript root, dan android/ di-generate
ulang tiap prebuild sehingga tidak bisa disunting manual. Pakai kapt.

Robolectric dipatok @Config(sdk = [34]). targetSdk proyek 36 dan Robolectric
4.14 belum mengenalnya — dibiarkan berarti test gagal karena runtime, bukan
karena kode.

androidx.security:security-crypto 1.1.0 memakai MasterKey.Builder; 1.0.0
memakai MasterKeys dengan tanda tangan create yang berbeda.

expo-constants TIDAK dapat di-import dari kode aplikasi — ia bersarang di
node_modules/expo. Varian aplikasi diambil dari native lewat getEnvironment().

Pola gitignore "android/" telanjang memakan seluruh kode Kotlin di
mobile/modules/*/android/. Pakai pola berjangkar /mobile/android/. Begitu juga
!.env.example tidak mencakup .env.uat.example — pakai !.env*.example.

adb menyambung ulang lewat mDNS sehingga muncul dua entri untuk HP yang sama,
dan Expo gagal dengan "Could not find device with name". Setel
ADB_MDNS_AUTO_CONNECT=0.

Port 8080 dan 8081 di mesin Akbar dipakai container project lain. Backend dev
memakai 8090, dan LISTEN_ADDR wajib 0.0.0.0 — HP di LAN tidak akan pernah bisa
menghubungi server yang mengikat ke localhost.

DEVICE_SECRET_KEY wajib tetap antar restart. Kunci baru membuat secret device
yang tersimpan tidak dapat didekripsi.

Dua database di Postgres yang sama: gopay_test dihapus tiap make test,
gopay_dev bertahan. Jangan pernah mengarahkan HP ke gopay_test.

## Keadaan saat ini

Selesai dan terbukti di perangkat: penangkapan notifikasi, penyaringan,
penyimpanan Room, pengiriman ber-HMAC, retry, penanganan kegagalan autentikasi,
idempotency, dan empat layar aplikasi. Backend Go lengkap dengan HMAC,
idempotency lewat constraint database, dan endpoint events/sources/heartbeat.

Belum: HTTPS sungguhan (menunggu VPS), uji ketahanan semalaman di ColorOS,
sub-project 3 (invoice, nominal unik, matching, webhook, konsol pengecualian),
pemasangan satu perintah, dan sistem lisensi.

Urutan pekerjaan yang disepakati: selesaikan pondasi dan uji ketahanan dulu,
lalu sub-project 3, lalu pemasangan satu perintah, terakhir lisensi. Lisensi
sengaja paling akhir — tidak ada yang membeli produk karena sistem lisensinya
bagus.

Periksa docs/qa/qa-report.md untuk angka pasti, dan git log untuk keputusan
terbaru beserta alasannya. Pesan commit di repo ini sengaja panjang dan memuat
alasan, bukan hanya apa yang berubah — pakai itu.

## Yang saya harapkan darimu

Kalau kamu menemukan sesuatu yang tampak salah di dokumen atau kode, katakan
sebelum mengubahnya. Beberapa hal yang tampak aneh memang disengaja dan
alasannya tertulis. Beberapa memang bug — dan kalau begitu, saya ingin tahu.

Jangan menandai apa pun selesai tanpa bukti yang bisa saya lihat.
```

---

## Cara memakainya

1. Tempel blok di atas sebagai pesan pertama ke AI baru.
2. Minta ia membaca `CLAUDE.md` dan `docs/qa/qa-report.md` lebih dulu, lalu
   melaporkan pemahamannya sebelum menyentuh kode.
3. Kalau AI itu punya akses shell, minta ia menjalankan `git log --oneline -30`
   — riwayat commit di repo ini memuat alasan keputusan, bukan hanya daftar
   perubahan.

## Yang sengaja tidak dimasukkan

Rincian teknis yang sudah ada di `CLAUDE.md` dan spec tidak diulang di prompt
ini. Menduplikasinya berarti dua tempat yang harus diperbarui bersamaan, dan
salah satunya pasti tertinggal.
