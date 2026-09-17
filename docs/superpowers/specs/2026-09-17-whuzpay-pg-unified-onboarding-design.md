# Onboarding Terpadu whuzpay-pg — Design Spec

## 1. Latar belakang

Sub-project kedua dari rencana "whuzpay-pg jadi produk yang bisa dipakai
orang lain" (sub-project pertama, distribusi APK, sudah selesai —
`docs/superpowers/specs/2026-09-17-android-bridge-apk-distribution-design.md`).

Alur onboarding merchant baru SEKARANG tersebar di dua sistem akun
terpisah total (whuzpay-pg dan gopay-notifications, dua produk berbeda di
repo yang sama), dijalani manual langkah demi langkah:

1. Daftar akun baru di gopay-notifications (`whuzpay.com/register`).
2. Login, upload QRIS statis di Settings gopay-notifications.
3. Buat API key di halaman API Keys gopay-notifications.
4. Buat webhook endpoint di halaman Webhooks gopay-notifications
   (URL harus persis `https://pg.whuzpay.com/api/v1/provider-webhooks/gopay`,
   event `invoice.paid`).
5. Buat device di halaman Devices gopay-notifications, dapat QR pairing.
6. Scan QR itu di app Android, Test Connection.
7. Daftar akun BARU LAGI di whuzpay-pg (`pg.whuzpay.com/register`) — sistem
   akun sendiri, tidak terhubung ke akun gopay-notifications sama sekali.
8. Login ke whuzpay-pg, copy-paste manual API key (langkah 3) dan Webhook
   Secret (langkah 4) ke Settings > "Provider Pembayaran GoPay".
9. Baru bisa mulai terima pembayaran.

Akbar sendiri (penulis kedua produk ini) butuh bantuan langkah-demi-langkah
untuk menjalani alur ini pertama kali — sinyal kuat bahwa merchant lain
akan kesulitan atau menyerah.

**Constraint yang sudah final dari brainstorming sebelumnya (spec APK
distribution §1 dan §6) dan TIDAK berubah di sini:** tidak ada custody
dana (QRIS milik akun gopay-notifications merchant sendiri, dana langsung
ke merchant), tidak ada gerbang approval merchant baru, model "bring your
own device + akun GoPay" diterima sebagai identitas produk. Spec ini
sama sekali tidak mengusulkan ulang hal-hal itu.

**Constraint tambahan yang dikonfirmasi Akbar saat brainstorming spec
ini:** gopay-notifications WAJIB tetap berfungsi penuh sebagai produk
berdiri sendiri untuk customer yang mendaftar langsung di `whuzpay.com`
tanpa pernah tahu/pakai whuzpay-pg — desain di sini tidak boleh mengubah
perilaku endpoint publik gopay-notifications maupun mensyaratkan
keberadaan whuzpay-pg. Merchant yang akunnya dibuat otomatis lewat
whuzpay-pg juga harus tetap bisa login langsung ke gopay-notifications
kapan saja dan memakainya lepas dari whuzpay-pg — akun yang dibuat itu
akun asli biasa, baris yang sama persis dengan customer yang daftar manual.

## 2. Tujuan

Merchant baru cukup mengisi **satu form pendaftaran** di whuzpay-pg untuk
mendapatkan: akun whuzpay-pg, akun gopay-notifications (dibuat otomatis di
baliknya), API key, webhook endpoint, dan device (QR pairing) — semuanya
tersambung, tanpa perlu sadar ada dua dashboard terpisah. Satu-satunya
langkah manual yang wajib tetap ada: mengunggah gambar QRIS (butuh file
milik merchant sendiri) dan memindai QR pairing dengan HP fisik (butuh
kamera sungguhan).

## 3. Di luar cakupan

- Mengubah endpoint publik `POST /api/v1/signup` gopay-notifications —
  dipanggil apa adanya, tidak dimodifikasi sama sekali (lihat §1).
- Mekanisme "klaim akun lama" saat email sudah terdaftar di
  gopay-notifications — celah keamanan (siapa pun bisa mengklaim email
  orang lain). Fallback-nya tetap alur manual yang sudah ada (§5).
- Menyimpan sesi gopay-notifications ke database untuk dipakai belakangan
  (mis. menyelesaikan upload QRIS di hari lain) — lihat §5, sengaja
  dibiarkan jadi alur manual kalau di-skip saat registrasi.
- Perubahan model custody/approval — lihat §1.

## 4. Desain

### 4.1. Alur & komponen baru

Saat form `/register` whuzpay-pg disubmit (field TETAP SAMA seperti
sekarang: `name`, `business_name`, `email`, `phone`, `password` — lihat
`RegisterRequest` di `whuzpay-pg/back/internal/domain/merchant/user.go` —
**tidak ada field wajib baru**, hanya satu field opsional di §4.2), backend
menjalankan cascade ini dalam satu request, pakai satu `http.Client` +
cookie jar sekali pakai (in-memory untuk durasi request itu saja, TIDAK
pernah ditulis ke database):

1. Buat akun whuzpay-pg (logika yang sudah ada, tidak berubah).
2. Panggil `POST /api/v1/signup` gopay-notifications lewat `GOPAY_BASE_URL`
   (loopback, env var yang sama sudah dipakai adapter provider
   `internal/provider/gopay`) — `business_name`/`email`/`password` dari
   form apa adanya, `username` digenerate dari bagian sebelum `@` di email
   (disanitasi jadi huruf kecil+angka+strip, dicoba dengan akhiran angka
   1-3x kalau bentrok `409 username_taken`).
3. Dari cookie sesi (`admin_session`) di response signup, panggil
   berurutan: `POST /api/v1/admin/api-keys` (`{"name": "whuzpay-pg"}`),
   `POST /api/v1/admin/webhooks` (`{"name": "whuzpay-pg", "url":
   "https://pg.whuzpay.com/api/v1/provider-webhooks/gopay", "events":
   ["invoice.paid", "invoice.expired"]}`), `POST /api/v1/admin/devices`
   (`{"name": "<business_name> - Bridge"}`). Ketiganya mengembalikan nilai
   rahasianya langsung di response (`key`/`secret`/`device_secret` —
   sudah demikian sejak awal, tidak perlu endpoint baru di gopay-notifications).
4. Simpan `api_key`, `webhook_secret`, dan `gopay_username` (hasil langkah
   2) ke `merchant_gopay_credentials` (lihat §4.4).
5. Device ID/Secret/`backend_url` dari langkah 3 **TIDAK disimpan ke
   database whuzpay-pg sama sekali** — dikembalikan SEKALI di response
   registrasi, dipegang di state React browser untuk dirender jadi QR di
   layar "Pasangkan HP" (§4.2). Pola sama persis "tampil sekali" yang
   sudah dipakai dialog device gopay-notifications sendiri.
6. (Opsional, lihat §4.2) Kalau merchant mengisi gambar QRIS di form yang
   sama, langkah terakhir cascade ini meneruskannya ke gopay-notifications
   selagi cookie sesi dari langkah 2 masih hidup — bukan request terpisah.

Komponen baru: package `whuzpay-pg/back/internal/gopayonboard` — klien
HTTP khusus keempat panggilan session-based di atas. Terpisah dari
`internal/provider/gopay` yang sudah ada (itu untuk `POST /invoices` per
transaksi pakai API key; ini untuk provisioning sekali di awal pakai sesi
admin — tanggung jawab dan siklus hidup yang berbeda, tidak digabung).

### 4.2. Form QRIS opsional + layar "Pasangkan HP"

- Form `/register` dapat **satu field baru, opsional**: upload gambar QRIS
  (PNG/JPEG, maks 300KB — batas yang sama dengan
  `maxQRISImageDecodedBytes` di gopay-notifications), dengan keterangan
  "boleh dilewati, bisa diisi belakangan lewat Settings". Kalau diisi,
  gambar dikirim base64 dalam request registrasi yang SAMA, diteruskan ke
  `PUT /api/v1/admin/account/qris-image` gopay-notifications selagi
  cookie sesi dari langkah 2 (§4.1) masih hidup di memori request itu —
  konsisten dengan keputusan "tidak pernah simpan sesi ke database".
- Alasan opsional (bukan wajib di form): sebagian orang daftar dulu, baru
  mencari file QRIS-nya belakangan. Mewajibkannya di form yang sama
  berisiko menaikkan angka batal daftar.
- Kalau di-skip: merchant tetap lanjut (API key/webhook/device sudah
  otomatis tersambung), tapi payment production akan ditolak
  `qris_not_configured` (perilaku yang sudah ada) sampai QRIS diisi. Card
  "Provider Pembayaran GoPay" di Settings whuzpay-pg dapat status
  "QRIS: belum diisi" + link ke halaman Settings gopay-notifications
  (`whuzpay.com`) untuk menyelesaikannya di sana — TIDAK dicoba
  diotomatisasi lagi belakangan (butuh sesi baru yang sudah tidak ada;
  lihat §3).
- Setelah submit (dengan atau tanpa QRIS), whuzpay-pg **auto-login**
  (JWT langsung terbit — perubahan dari sekarang yang redirect ke
  `/login` manual, lihat `whuzpay-pg/front/app/register/page.tsx`) lalu
  masuk ke layar baru **"Pasangkan HP"**: QR code dari device yang sudah
  dibuat di §4.1 langkah 3, plus link ke `/download-app` (sub-project 1)
  untuk unduh aplikasinya. Layar ini murni menampilkan data dari response
  registrasi — tidak ada panggilan API baru ke gopay-notifications di
  titik ini.

### 4.3. Penanganan gagal

Prinsip: **akun whuzpay-pg wajib tetap berhasil dibuat** apa pun yang
terjadi di sisi gopay-notifications — produk sendiri tidak boleh
terhambat gangguan di produk sister.

- **Email/username sudah dipakai** di gopay-notifications (merchant
  ternyata sudah pernah daftar di sana sebelumnya) — setelah retry
  username 2-3x tetap gagal (kalau bentrok di email, retry username tidak
  menolong), cascade berhenti di situ. Akun whuzpay-pg tetap jadi, pesan
  ke merchant: "Akun whuzpay-pg berhasil dibuat. Email ini sudah
  terdaftar di gopay-notifications — hubungkan manual lewat Settings."
  Diarahkan ke card "Provider Pembayaran GoPay" yang SUDAH ADA sekarang
  (isi API key + webhook secret manual). Tidak ada mekanisme klaim akun
  lama (lihat §3).
- **gopay-notifications tidak bisa dihubungi / error 500** — sama: akun
  whuzpay-pg tetap jadi, cascade berhenti di titik gagalnya, pesan
  "Sedang ada gangguan menyambungkan otomatis, hubungkan manual lewat
  Settings."
- **Gagal di tengah cascade** (mis. signup+API key sukses, webhook
  gagal) — apa pun yang sudah didapat tetap disimpan (API key tetap
  kepakai), yang gagal dibiarkan kosong. Otomatis didukung tanpa kerja
  tambahan karena `merchant_gopay_credentials` sudah didesain tri-state
  sejak migrasi 016 (boleh salah satu kolom kosong).
- **Device gagal dibuat** — layar "Pasangkan HP" menampilkan pesan "Buat
  device manual lewat gopay-notifications" (alur lama tetap ada sebagai
  fallback) alih-alih QR kosong/error.

Tidak ada rollback/transaksi lintas dua sistem — sengaja, karena tiap
keping yang berhasil didapat tetap berguna sendiri-sendiri, dan fallback
manual yang sudah ada menutupi apa pun yang gagal.

### 4.4. Perubahan data

Migrasi baru `017_add_gopay_username_to_merchant_gopay_credentials.sql`:

```sql
ALTER TABLE merchant_gopay_credentials
    ADD COLUMN gopay_username TEXT;

COMMENT ON COLUMN merchant_gopay_credentials.gopay_username IS
    'Username akun gopay-notifications yang dibuat otomatis saat onboarding
     terpadu (lihat spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md).
     Ditampilkan di Settings supaya merchant bisa login langsung ke
     gopay-notifications kapan saja kalau mau -- akun itu asli, bukan
     ''milik'' whuzpay-pg.';
```

Tidak ada tabel/kolom baru untuk sesi atau device secret — keduanya
sengaja tidak pernah disimpan (§4.1, §4.2).

## 5. Pengujian

**Otomatis (bisa Claude jalankan):**
- Unit test Go untuk tiap fungsi `internal/gopayonboard` (sukses,
  `email_taken`/`username_taken`, network error, response tidak valid)
  pakai `httptest.Server` yang meniru gopay-notifications — tidak perlu
  gopay-notifications sungguhan jalan.
- Unit test `RegisterMerchant`/handler untuk skenario: cascade sukses
  penuh, gagal sebagian (API key dapat, webhook gagal), gagal total sejak
  signup — pastikan response akhir tetap `201` di semua kasus dan data
  yang sudah didapat tetap tersimpan.
- Frontend: `npx tsc --noEmit`, `npx eslint .`, `npx next build` di
  `whuzpay-pg/front`.

**Manual, `NEEDS-DEVICE` (Akbar yang jalankan dan tempel buktinya):**
1. Registrasi sungguhan lewat form `/register` produksi (tanpa QRIS
   dulu) — cek langsung di gopay-notifications (`whuzpay.com`) bahwa
   akun, API key, dan webhook endpoint benar-benar dibuat sesuai yang
   diisi di form whuzpay-pg.
2. Registrasi kedua DENGAN QRIS diisi di form — cek gambar QRIS
   benar-benar tersimpan di akun gopay-notifications yang baru itu.
3. Layar "Pasangkan HP" menampilkan QR yang valid — scan dari app Android
   (diunduh dari `/download-app`), Test Connection sukses.
4. Registrasi ketiga pakai email yang SUDAH terdaftar di
   gopay-notifications (dari test #1) — pastikan akun whuzpay-pg tetap
   jadi, pesan fallback manual muncul, dan alur manual di Settings masih
   berfungsi seperti sebelumnya.
5. Buat payment sungguhan pakai akun dari test #2 (yang sudah ada QRIS),
   bayar via GoPay, pastikan alur end-to-end (yang sudah pernah dibuktikan
   `PASS` di qa-report.md §28) tetap jalan lewat kredensial yang
   dibuat OTOMATIS ini, bukan cuma yang diisi manual.

## 6. Keputusan yang sengaja diambil (jangan diubah diam-diam)

- **Endpoint publik gopay-notifications tidak diubah sama sekali.**
  whuzpay-pg cuma jadi pemanggil baru dari endpoint yang sudah ada;
  customer yang daftar langsung di `whuzpay.com` tidak terpengaruh apa
  pun oleh spec ini (lihat §1).
- **Tidak ada mekanisme klaim akun gopay-notifications lama** lewat
  kecocokan email — itu lubang keamanan. Fallback-nya selalu alur manual
  yang sudah ada.
- **Sesi gopay-notifications dan device secret tidak pernah disimpan ke
  database whuzpay-pg** — device secret cuma lewat response sekali pakai
  (state browser), sesi cuma hidup selama satu request HTTP. Konsekuensi
  yang diterima: kalau merchant skip QRIS saat registrasi, menyelesaikannya
  belakangan HARUS lewat gopay-notifications langsung (bukan wizard
  whuzpay-pg lagi) — bukan regresi dari sekarang, cuma tidak
  seseamless jalur "sekali jalan".
- **Tidak ada rollback lintas sistem saat cascade gagal sebagian** —
  setiap keping yang berhasil didapat tetap disimpan dan berguna sendiri,
  memanfaatkan desain tri-state `merchant_gopay_credentials` yang sudah
  ada sejak migrasi 016.
