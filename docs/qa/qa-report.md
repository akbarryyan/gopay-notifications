# QA Report

**Milestone terakhir diperiksa:** M1, M2, M3 selesai · **M4 selesai lewat backend lokal** — pengiriman, retry, kegagalan autentikasi, duplikat, dan error server semuanya terbukti di perangkat. Sisa M4 hanya HTTPS, yang menunggu VPS
**Tanggal:** 2026-09-10
**Ringkasan:** `PASS` 61 · `FAIL` 0 · `BLOCKED` 0 · `NEEDS-DEVICE` 2 · `PENDING` 5

---

## 1. Ringkasan

Seluruh sisi backend selesai dan terbukti bekerja di mesin development: 4 paket, seluruh test lulus termasuk dengan race detector, ditambah uji end-to-end lewat HTTP sungguhan yang menembus rantai lengkap HMAC → validasi → Postgres.

Yang **belum** terbukti adalah apa pun yang menuntut VPS: sertifikat HTTPS sungguhan, basic auth Caddy, dan systemd. Berkas konfigurasinya sudah ditulis dan build silang `linux/amd64` berhasil, tetapi tidak satu pun dapat diverifikasi dari mesin development. Empat butir itu ditandai `NEEDS-DEVICE`.

**M2 sebagian selesai.** Aplikasi terpasang dan berjalan di OPPO CPH2365, `NotificationListenerService` terdaftar di APK yang benar-benar terpasang, dan Notification Access sudah diberikan sehingga status ikatan menunjukkan `ya`. Yang belum: konfirmasi setelan anti-ColorOS, dan penangkapan notifikasi GoPay sungguhan yang baru ada setelah Task 8.

`AmountParser`, `EventIdBuilder`, dan `Signer` lulus seluruh 20 unit testnya. Empat di antaranya membandingkan keluaran Kotlin dengan vektor yang dihasilkan dari implementasi Go di `backend/internal/auth` — kedua implementasi HMAC karena itu terbukti sepakat, bukan sekadar sama-sama tampak benar.

Setelan anti-ColorOS sudah dikerjakan. Ketahanannya belum terbukti: itu baru diuji di M6, saat HP didiamkan semalaman.

**Bukti pokok — seluruh test:**

```
$ make test
ok  github.com/akbarryyan/gopay-notifications/backend/internal/auth       0.003s
ok  github.com/akbarryyan/gopay-notifications/backend/internal/httpapi    0.610s
ok  github.com/akbarryyan/gopay-notifications/backend/internal/secretbox  0.004s
ok  github.com/akbarryyan/gopay-notifications/backend/internal/store      0.273s

$ go test ./... -race -count=1 -p 1
ok  .../internal/auth 1.019s   ok .../internal/httpapi 1.983s
ok  .../internal/secretbox 1.018s   ok .../internal/store 1.443s
```

**Bukti pokok — end-to-end lewat HTTP sungguhan:**

```
1. kirim pertama   → 200 {"success":true,"event_id":"evt_3f9a...","status":"accepted"}
2. kirim ulang     → 200 {"success":true,"event_id":"evt_3f9a...","status":"duplicate"}
3. tanda tangan salah → 401 {"success":false,"error":"invalid_signature",...}
4. jam meleset 10m → 401 {"success":false,"error":"clock_skew",...,"server_time":1789054130}
5. baris di DB     → 1
6. last_seen_at    → t
7. secret di log   → 0 kemunculan
```

---

## 2. Butir `NEEDS-DEVICE` yang menunggu

Empat butir menunggu akses VPS. Langkahnya ada di [`backend/deploy/README.md`](../../backend/deploy/README.md).

| # | Langkah | Hasil yang diharapkan | Laporkan |
|---|---|---|---|
| 1 | `curl -s https://<domain>/api/v1/health` | `{"status":"ok","server_time":...}` lewat sertifikat sah | Keluaran lengkap |
| 2 | `curl -s -o /dev/null -w '%{http_code}' https://<domain>/api/v1/events` | `401` | Kode HTTP |
| 3 | `curl -s -o /dev/null -w '%{http_code}' -u admin:<pw> https://<domain>/api/v1/events` | `200` | Kode HTTP |
| 4 | `curl -s -o /dev/null -w '%{http_code}' http://<domain>/api/v1/health` | `308` — Caddy mengalihkan, tidak ada layanan di HTTP polos | Kode HTTP |

Selain itu, `devicetool -name "HP GoPay Utama"` perlu dijalankan di VPS untuk menghasilkan `Device ID` dan `Device Secret` sungguhan yang dipakai M4.

Dua butir sisi Android yang sempat menunggu sudah selesai 2026-09-10: setelan anti-ColorOS dan unit test Kotlin. Keduanya kini tercatat sebagai `PASS` di bawah.

**Perubahan scope 2026-09-11** menambah tiga butir baru. Sumber pembayaran berpindah dari akun GoPay pribadi ke **GoPay Merchant**, sepenuhnya, sehingga package dan judul notifikasi yang terkonfirmasi sehari sebelumnya tidak lagi berlaku sebagai nilai produksi:

| # | Langkah | Hasil yang diharapkan | Laporkan |
|---|---|---|---|
| 5 | `adb shell pm list packages \| grep -i -E 'gojek\|gopay\|gobiz\|merchant'` | Nama package aplikasi GoPay Merchant | Keluarannya |
| 6 | Terima satu pembayaran sungguhan, jangan swipe notifikasinya, lalu `adb shell dumpsys notification --noredact` | `android.title` dan `android.text` notifikasi pembayaran masuk | Kedua nilainya |
| 7 | Periksa teks notifikasi itu | Ada tidaknya **nomor referensi transaksi** | Ya atau tidak, beserta bentuknya |

Butir 5–7 **selesai 2026-09-11**; hasilnya tercatat di §2.6.2 spec. Notifikasi merchant tidak memuat nomor referensi, sehingga nominal unik tetap satu-satunya jalur matching di sub-project 3.

Butir 8 (enkripsi konfigurasi) **selesai 2026-09-11** dan kini tercatat `PASS`.

Butir 9 (penyaringan notifikasi non-merchant) **selesai 2026-09-11**. Seluruh sisa butir yang menunggu kini menyangkut **akses VPS**.

---

## 3. Functional Requirements — [prd.md §11](../prd.md)

| # | Requirement | Milestone | Status | Bukti |
|---|---|---|---|---|
| FR-01 | Notification Access terdeteksi | M2 | `PASS` | `adb shell settings get secure enabled_notification_listeners` memuat `id.akbarryyan.gopaybridge.dev/expo.modules.gopaylistener.GoPayListenerService`. Diverifikasi ulang setelah package diganti, 2026-09-11 |
| FR-02 | Notification Listener menerima event | M2 | `PASS` | Mode Discovery mencatat `com.whatsapp` saat pesan masuk — listener menerima event dari sistem. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| FR-03 | Hanya memproses event GoPay Merchant | M3 | `PASS` | WhatsApp dikirim ke HP; tab Riwayat tetap kosong — notifikasi non-merchant tidak tercatat sebagai event. Diperkuat 12 test `CapturePolicyTest`, termasuk `aplikasi GoPay pribadi dilewati selama tidak dipantau`. OPPO CPH2365, 2026-09-11 |
| FR-04 | Parsing nominal dan `event_id` (unit) | M3 | `PASS` | `./gradlew :gopay-listener:testDebugUnitTest` → BUILD SUCCESSFUL; `build/test-results/testDebugUnitTest/*.xml` mencatat `tests=20 failures=0 errors=0 skipped=0` (AmountParserTest 8, EventIdBuilderTest 5, SignerTest 7) |
| FR-04 | Notifikasi jadi event terstruktur (pipeline) | M3 | `PASS` | Pembayaran QRIS sungguhan muncul di Riwayat dengan nominal dan status `PENDING`. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| FR-05 | Kirim event ke backend (jalur HTTP) | M4 | `PASS` | Pembayaran QRIS Rp3 sungguhan: HP menandai `SENT`, dan baris tersimpan di `gopay_dev` dengan `amount_hint=3`, `title=Pembayaran QRIS statis diterima`, `package_name=com.gojek.gopaymerchant`, serta `raw_payload` utuh. Backend lokal, 2026-09-11 |
| FR-05 | Kirim event via **HTTPS** | M4 | `PENDING` | Jalur pengiriman terbukti, tetapi lewat HTTP ke backend lokal. HTTPS menuntut VPS dengan sertifikat |
| FR-06 | Autentikasi request — sisi backend | M1 | `PASS` | `TestVerifyRejectsModifiedBody`, `TestVerifyRejectsWrongSecret`, `TestAuthRejectsWrongSecret`, `TestAuthRejectsUnknownDevice`, `TestAuthRejectsDisabledDevice`, `TestAuthRejectsMissingHeaders` (3 subtest), e2e no. 3 |
| FR-06 | Autentikasi request — sisi Android | M4 | `PASS` | Tombol Test Connection di HP menjawab "Terhubung sebagai HP GoPay Dev", dan `last_seen_at` device di `gopay_dev` terisi — itu hanya dijalankan setelah `auth.Verify` lolos di middleware, jadi tanda tangan Kotlin terbukti cocok dengan verifikasi Go pada request sungguhan. Backend lokal, 2026-09-11 |
| FR-07 | Retry untuk error yang dapat dipulihkan | M4 | `PASS` | Backend lokal dimatikan, pembayaran sungguhan diterima → event bertahan `PENDING` dengan `lastError=network` dan `attemptCount` bertambah. Backend dinyalakan lagi → status berpindah sendiri ke `SENT` tanpa campur tangan. OPPO CPH2365, 2026-09-11 |
| FR-08 | Pencegahan duplikat — sisi backend | M1 | `PASS` | `TestInsertEventConcurrentSameIDInsertsOnce` (8 goroutine, `-count=20 -race`), `TestCallbackSecondTimeReturnsDuplicate`, e2e no. 2 dan 5 |
| FR-08 | Pencegahan duplikat — sisi Android | M3 | `PASS` | `EventDaoTest.menolak event_id yang sama tanpa melempar exception` — primary key menolak penyisipan kedua dan mengembalikan `-1`. `./gradlew :gopay-listener:testDebugUnitTest` → `tests=49 failures=0 errors=0 skipped=0` (AmountParser 10, EventIdBuilder 5, Signer 7, EventDao 12, Settings 10, DiscoveryLog 5) |
| FR-09 | Pencatatan status event | M3 | `PASS` | Riwayat menampilkan status `PENDING` untuk event yang belum terkirim. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| FR-10 | Indikator listener aktif | M5 | `PASS` | Dashboard menampilkan Notification Access `aktif` dan Listener `terikat`. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |

---

## 4. Acceptance Criteria — [prd.md §16](../prd.md)

| Kriteria | Milestone | Status | Bukti |
|---|---|---|---|
| User dapat memberikan Notification Access | M2 | `PASS` | Izin diberikan lewat tombol di aplikasi; status berubah jadi aktif. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Aplikasi mendeteksi notifikasi baru | M2 | `PASS` | Pembayaran QRIS sungguhan tertangkap dan muncul di Riwayat. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Membedakan GoPay Merchant dari aplikasi lain | M3 | `PASS` | WhatsApp dikirim ke HP; tab Riwayat tetap kosong — notifikasi non-merchant tidak tercatat sebagai event. Diperkuat 12 test `CapturePolicyTest`, termasuk `aplikasi GoPay pribadi dilewati selama tidak dipantau`. OPPO CPH2365, 2026-09-11 |
| Notifikasi jadi event terstruktur | M3 | `PASS` | Event memuat nominal hasil parsing dan status; Dashboard menampilkan "Event terakhir". Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Event terkirim via HTTPS | M4 | `PENDING` | Pengiriman terbukti lewat HTTP lokal; HTTPS menunggu VPS |
| Backend mengidentifikasi perangkat pengirim | M1 | `PASS` | `TestAuthAcceptsValidSignature`, `TestTouchDeviceSetsLastSeenAt`, e2e no. 6 (`last_seen_at` = `t`) |
| Event sama tidak diproses dua kali | M1 | `PASS` | `TestInsertEventConcurrentSameIDInsertsOnce`, e2e no. 5 (tepat 1 baris setelah 2 kiriman identik) |
| Event terkirim setelah koneksi normal kembali | M4 | `PASS` | Backend lokal dimatikan, pembayaran sungguhan diterima → event bertahan `PENDING` dengan `lastError=network` dan `attemptCount` bertambah. Backend dinyalakan lagi → status berpindah sendiri ke `SENT` tanpa campur tangan. OPPO CPH2365, 2026-09-11 |
| User melihat status listener | M5 | `PASS` | Baris Listener di Dashboard. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| User melihat status komunikasi backend | M5 | `PASS` | Baris Backend di Dashboard melaporkan `belum dikonfigurasi` dengan benar. Keadaan `terhubung` menyusul setelah VPS. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| User melihat event/history dasar | M5 | `PASS` | Tab Riwayat menampilkan event beserta nominal, waktu, dan status. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Tidak memproses notifikasi selain GoPay Merchant | M3 | `PASS` | WhatsApp dikirim ke HP; tab Riwayat tetap kosong — notifikasi non-merchant tidak tercatat sebagai event. Diperkuat 12 test `CapturePolicyTest`, termasuk `aplikasi GoPay pribadi dilewati selama tidak dipantau`. OPPO CPH2365, 2026-09-11 |

---

## 5. Definition of Done — [detail-project.md §42](../detail-project.md)

| Butir | Milestone | Status | Bukti |
|---|---|---|---|
| Aplikasi berjalan di Android | M2 | `PASS` | Diverifikasi ulang setelah package diganti: `adb shell pm list packages \| grep gopaybridge` → `package:id.akbarryyan.gopaybridge.dev`, dan package lama `id.manjo.*` sudah tidak ada. OPPO CPH2365, 2026-09-11 |
| TypeScript sebagai application language | M2 | `PASS` | `npx tsc --noEmit` bersih; `App.tsx` dan `modules/gopay-listener/index.ts` |
| Kotlin untuk Notification Listener | M2 | `PASS` | `adb shell dumpsys package id.akbarryyan.gopaybridge.dev` → `id.akbarryyan.gopaybridge.dev/expo.modules.gopaylistener.GoPayListenerService` dengan permission `BIND_NOTIFICATION_LISTENER_SERVICE` dan action `android.service.notification.NotificationListenerService`. Diverifikasi ulang 2026-09-11 |
| Notification Access dapat diaktifkan | M2 | `PASS` | Tombol membuka `Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS`; izin diberikan dan status berubah jadi aktif |
| Listener berjalan di background | M6 | `PENDING` | — |
| Notifikasi GoPay Merchant terdeteksi | M2 | `PASS` | Pembayaran QRIS sungguhan tertangkap. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Notifikasi aplikasi lain diabaikan | M3 | `PASS` | WhatsApp dikirim ke HP; tab Riwayat tetap kosong — notifikasi non-merchant tidak tercatat sebagai event. Diperkuat 12 test `CapturePolicyTest`, termasuk `aplikasi GoPay pribadi dilewati selama tidak dipantau`. OPPO CPH2365, 2026-09-11 |
| Data notification dapat diekstrak | M3 | `PASS` | Nominal tampil di Riwayat dan Dashboard, hasil `AmountParser` atas teks sungguhan. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Event ID dibuat | M3 | `PASS` | Event tersimpan; `eventId` adalah primary key sehingga baris tidak akan ada tanpanya. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Lapisan penyimpanan lokal (Room) | M3 | `PASS` | 12 test `EventDaoTest`, termasuk pembacaan kolom mentah yang mengunci enum tersimpan sebagai TEXT `PENDING` |
| Event tersimpan lokal dari notifikasi sungguhan | M3 | `PASS` | Event bertahan di Riwayat setelah pembayaran sungguhan. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Event tersimpan di backend | M1 | `PASS` | `TestCallbackStoresRawPayloadVerbatim`, e2e no. 7 (`GET /events` mengembalikan event) |
| Event dapat dikirim ke backend | M4 | `PASS` | Pembayaran QRIS Rp3 sungguhan: HP menandai `SENT`, dan baris tersimpan di `gopay_dev` dengan `amount_hint=3`, `title=Pembayaran QRIS statis diterima`, `package_name=com.gojek.gopaymerchant`, serta `raw_payload` utuh. Backend lokal, 2026-09-11 |
| HTTPS untuk production | M1 | `NEEDS-DEVICE` | `Caddyfile` ditulis, build silang OK; sertifikat belum diverifikasi — lihat §2 no. 1 |
| Authentication ditegakkan backend | M1 | `PASS` | Sama dengan FR-06 sisi backend |
| Authentication dikirim Android | M4 | `PASS` | Tombol Test Connection di HP menjawab "Terhubung sebagai HP GoPay Dev", dan `last_seen_at` device di `gopay_dev` terisi — itu hanya dijalankan setelah `auth.Verify` lolos di middleware, jadi tanda tangan Kotlin terbukti cocok dengan verifikasi Go pada request sungguhan. Backend lokal, 2026-09-11 |
| Retry mechanism berjalan | M4 | `PASS` | Backend lokal dimatikan, pembayaran sungguhan diterima → event bertahan `PENDING` dengan `lastError=network` dan `attemptCount` bertambah. Backend dinyalakan lagi → status berpindah sendiri ke `SENT` tanpa campur tangan. OPPO CPH2365, 2026-09-11 |
| Duplicate event ditangani backend | M1 | `PASS` | Sama dengan FR-08 sisi backend |
| Duplicate event ditangani Android | M3 | `PASS` | Event yang sudah `SENT` dikirim ulang lewat tombol di layar Debug → backend menjawab `duplicate`, kartu menampilkan `Backend: duplicate`, status tetap `SENT` dan bukan error. Jumlah baris di `gopay_dev` tetap **5**, tanpa satu pun `event_id` ganda. 2026-09-11 |
| Dashboard menampilkan listener status | M5 | `PASS` | Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Dashboard menampilkan backend status | M5 | `PASS` | Melaporkan `belum dikonfigurasi` dengan benar. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| History event tersedia | M5 | `PASS` | Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Device ID tersedia | M1 | `PASS` | `devicetool -name "HP GoPay Utama"` → `Device ID : dev_18576e55cfa02f73` |
| Credential tidak hardcoded | M1 | `PASS` | `grep -rn 'DEVICE_SECRET_KEY' --include='*.go'` hanya menemukan pembacaan `os.Getenv` di `internal/config/config.go:33` dan teks bantuan flag |
| Menangani network failure | M4 | `PASS` | Backend lokal dimatikan, pembayaran sungguhan diterima → event bertahan `PENDING` dengan `lastError=network` dan `attemptCount` bertambah. Backend dinyalakan lagi → status berpindah sendiri ke `SENT` tanpa campur tangan. OPPO CPH2365, 2026-09-11 |
| Dapat diuji saat UI tidak terbuka | M6 | `PENDING` | — |
| Diuji pada physical Android device | M6 | `PASS` | Seluruh pengujian sejak M2 dilakukan di OPPO CPH2365 Android 13 dengan pembayaran QRIS sungguhan — penangkapan, penyaringan, penyimpanan, pengiriman, retry, dan kegagalan autentikasi |

---

## 6. Tabel kode → tindakan — [api-contract.md §4.6](../api-contract.md)

Baris `429`, `5xx`, dan `timeout` menggambarkan perilaku **HP**, bukan backend, sehingga tetap `PENDING` sampai M4.

| Kondisi | Tindakan yang diharapkan | Status | Bukti |
|---|---|---|---|
| `200 accepted` | → `SENT` | `PASS` | `TestCallbackFirstTimeReturnsAccepted`, e2e no. 1, dan terbukti di perangkat: Pembayaran QRIS Rp3 sungguhan: HP menandai `SENT`, dan baris tersimpan di `gopay_dev` dengan `amount_hint=3`, `title=Pembayaran QRIS statis diterima`, `package_name=com.gojek.gopaymerchant`, serta `raw_payload` utuh. Backend lokal, 2026-09-11 |
| `200 duplicate` | → `SENT`, bukan error | `PASS` | `TestCallbackSecondTimeReturnsDuplicate`, e2e no. 2, dan terbukti di perangkat: kiriman ulang event yang sudah `SENT` menghasilkan `Backend: duplicate` tanpa menambah baris di database. 2026-09-11 |
| `400 invalid_payload` | → `FAILED`, tanpa retry | `PASS` | `TestCallbackRejectsMalformedJSON`, `TestCallbackRejectsInvalidFields` (7 subtest) |
| `401 invalid_signature` | → `FAILED`, peringatan kredensial | `PASS` | `TestAuthRejectsWrongSecret`, `TestAuthRejectsUnknownDevice`, e2e no. 3, dan **terbukti di perangkat**: Device Secret sengaja disalahkan → pembayaran sungguhan langsung `FAILED` dengan `invalid_signature` dan berhenti mencoba; setelah secret diperbaiki, Kirim ulang menghasilkan `SENT` dengan `Backend: accepted` dan baris masuk `gopay_dev` pukul 16:21:53. 2026-09-11 |
| `401 clock_skew` | → `FAILED`, pesan jam meleset + jam server | `PASS` | `TestAuthRejectsClockSkewWithServerTime`, e2e no. 4 (`server_time` disertakan) |
| `403 device_disabled` | → `FAILED`, tanpa retry | `PASS` | `TestAuthRejectsDisabledDevice` |
| `429` | tetap `PENDING`, hormati `Retry-After` | `PENDING` | Backend kita tidak menerapkan rate limiting sehingga `429` tidak dapat dihasilkan darinya; ia hanya mungkin datang dari Caddy di depan. Pemetaannya diuji `ResponseMapperTest`, perilaku di perangkat belum pernah terpicu |
| `5xx` | tetap `PENDING`, retry | `PASS` | Postgres dimatikan sementara server Go tetap hidup → `InsertEvent` gagal → backend membalas `500` → HP menandai `PENDING` dan mencoba lagi. Setelah Postgres dinyalakan, kiriman ulang berhasil `SENT`. Berbeda dari uji jaringan mati: di sini server **menjawab** dengan error, bukan diam. 2026-09-11 |
| timeout / jaringan mati | tetap `PENDING`, retry | `PASS` | Backend ditolak koneksinya (server mati) → `IOException` → `Retry("network")`, event bertahan `PENDING` lalu terkirim saat server hidup. Backend lokal dimatikan, pembayaran sungguhan diterima → event bertahan `PENDING` dengan `lastError=network` dan `attemptCount` bertambah. Backend dinyalakan lagi → status berpindah sendiri ke `SENT` tanpa campur tangan. OPPO CPH2365, 2026-09-11 |

---

## 7. Pemeriksaan keamanan & prasyarat lingkungan — [qa-rules.md §9](qa-rules.md)

| Pemeriksaan | Status | Bukti |
|---|---|---|
| Tidak ada secret ter-hardcode (backend) | `PASS` | `grep -rn 'DEVICE_SECRET_KEY' --include='*.go'` → hanya `os.Getenv` dan teks flag |
| Tidak ada secret ter-hardcode (Android) | `PASS` | `grep -rniE 'secret=\|password=\|token=\|apikey' android/src/main` → hanya `KEY_DEVICE_SECRET = "device_secret"`, nama kunci penyimpanan, bukan nilai |
| Tidak ada secret/tanda tangan di log (backend) | `PASS` | `grep -rn 'slog\.' internal/ cmd/ \| grep -iE 'secret\|signature'` → kosong; e2e no. 8 → 0 kemunculan secret di log server |
| Tidak ada secret/tanda tangan di log (Android) | `PASS` | `grep -rn 'Log\.' android/src/main` → dua panggilan, keduanya tentang status ikatan listener tanpa isi sensitif. Diperiksa ulang setelah Task 8 menambah logging pipeline |
| `hmac.Equal` dipakai, bukan `==` | `PASS` | `internal/auth/hmac.go:38` → `return hmac.Equal([]byte(want), []byte(gotSignature))` |
| Body diverifikasi mentah sebelum decode | `PASS` | `auth_middleware.go:71` `io.ReadAll` → `:90` `auth.Verify(...)`; `json.Unmarshal` baru di `callback.go:50`, setelah middleware |
| Toleransi timestamp ditegakkan di kedua batas | `PASS` | `TestCheckSkewBoundaries` — 7 subtest, termasuk ±299/±300/±301 detik |
| Idempotency memakai constraint database | `PASS` | `migrations/00001_init.sql:13` `event_id TEXT NOT NULL UNIQUE`; `event.go:35` `ON CONFLICT (event_id) DO NOTHING`; `event.go:42` `RowsAffected() == 1` |
| Build production menolak HTTP polos | `NEEDS-DEVICE` | Perlu Caddy di VPS — lihat §2 no. 4 |
| Aturan arah berupa allowlist, bukan blocklist | `PENDING` | Ditegakkan backend di sub-project 3. Entri awal `Pembayaran QRIS statis diterima` sudah tercatat di kontrak API |
| Mode Discovery default mati dan mati sendiri | `PASS` | `SettingsTest.discovery mati secara default` dan `discovery aktif hanya sampai batas waktunya`; `startDiscovery` membatasi 1–10 menit |
| Mode Discovery tidak mengirim apa pun keluar HP | `PASS` | Setelah mode Discovery aktif dan WhatsApp tertangkap, `SELECT count(*) FILTER (WHERE package_name <> 'com.gojek.gopaymerchant')` di `gopay_dev` → **0**. Hanya pembayaran yang pernah terkirim |
| Channel "Promotions and Marketing" `com.gojek.gopaymerchant` masih aktif | `PASS` | Notifikasi pembayaran sungguhan sampai ke listener, yang hanya mungkin bila channel-nya aktif. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| Enkripsi konfigurasi terbukti di perangkat | `PASS` | Device Secret diisi, aplikasi ditutup paksa, dibuka lagi — field menampilkan `tersimpan`, bukan `belum diisi`. Membuktikan `EncryptedSharedPreferences` menulis dan membaca lewat Android Keystore. Diverifikasi Akbar di OPPO CPH2365, 2026-09-11 |
| `monitoredPackages` berisi tepat satu entri | `PASS` | `SettingsTest.daftar package default berisi tepat satu entri merchant`; `EventIdBuilderTest` membuktikan `com.gojek.gopay` dan `com.gojek.gopaymerchant` menghasilkan `event_id` berbeda untuk teks identik |
| Setup ColorOS selesai | `PASS` | Keempat setelan dikerjakan ulang untuk package baru `id.akbarryyan.gopaybridge.dev` pada 2026-09-11: aktivitas latar belakang diizinkan, mulai otomatis aktif, aplikasi dikunci di recent apps, optimasi siaga tidur mati. Ketahanannya baru diuji di M6 |

---

## 8. Penyimpangan dari spec

Sebelas penyimpangan terdaftar di [§9 spec](../superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md). Yang menyentuh backend dan dapat dikonfirmasi sekarang:

| # | Penyimpangan | Konfirmasi |
|---|---|---|
| 6 | `payment.amount` → `amount_hint` | Berlaku. Kolom `amount_hint`, field JSON `amount_hint`, tanpa objek `payment` |
| 9 | `/api/callback/gopay` → `/api/v1/callback/gopay` | Berlaku. Seluruh rute berawalan `/api/v1/` |

Penyimpangan baru yang tidak terdaftar: **tidak ada**.

---

## 9. Regresi

Tidak berlaku — ini siklus QA pertama yang memuat implementasi.

Satu catatan untuk siklus berikutnya: target `test` di `Makefile` **wajib** memakai `-p 1`. Paket `store` dan `httpapi` sama-sama `TRUNCATE` database test yang sama, dan Go menjalankan paket secara paralel secara default — keduanya saling menghapus data di tengah jalan dan gagal secara acak, padahal sendiri-sendiri lulus. Menghapus `-p 1` akan memunculkan kembali kegagalan yang menyesatkan itu.

---

## 9b. Performance — [prd.md §12](../prd.md)

Diukur dari satu pembayaran sungguhan, membandingkan `posted_at` (waktu GoPay memasang notifikasi), `received_at` (saat listener menangkapnya), dan `ingested_at` (saat tersimpan di Postgres):

```
303 ms  notifikasi → ditangkap HP
 89 ms  ditangkap → tersimpan di Postgres
392 ms  TOTAL
```

PRD §12 menuntut "delay seminimal mungkin" tanpa angka pasti. Kurang dari setengah detik dari notifikasi muncul sampai tercatat di database, termasuk penandatanganan HMAC dan perjalanan lewat WiFi, memenuhi maksud itu dengan jelas.

Angka ini diambil di jaringan lokal. Lewat internet ke VPS, bagian kedua akan bertambah sebesar latensi jaringan.

---

## 10. Temuan & tindak lanjut

**Satu cacat ditemukan dan diperbaiki selama M1.** Target `test` di plan awal tidak memakai `-p 1`, sehingga `make test` gagal secara acak begitu paket kedua yang menyentuh database ditambahkan. Gejalanya menyesatkan karena setiap paket lulus bila dijalankan sendirian. Diperbaiki di `Makefile` dan di plan, dengan komentar yang menjelaskan sebabnya agar tidak dihapus orang lain di kemudian hari.

**Satu jebakan operasional yang perlu diingat.** Saat verifikasi manual, proses server dari langkah sebelumnya tidak mati dan tetap memegang port dengan binary lama — `GET /events` membalas 404 padahal rutenya sudah ada, dan `go build` ke path binary yang sedang berjalan gagal dengan *text file busy*. Sejak itu verifikasi manual memakai path binary yang unik.

**Yang menghalangi M1 dinyatakan tuntas:** akses VPS. Empat butir di §2 dan pembuatan device sungguhan tidak dapat dikerjakan dari mesin development.

**Perubahan scope 2026-09-11 membatalkan sebagian temuan lapangan.** Package identifier dan bentuk teks yang dikonfirmasi sebelum M1 berasal dari akun GoPay **pribadi**. Sumber pembayaran kini berpindah sepenuhnya ke **GoPay Merchant**, sehingga Open Question #1, #2, dan #3 kembali terbuka.

Biayanya nol baris kode. Package adalah daftar yang dapat diedit di Settings, dan aturan arah adalah konfigurasi backend — keduanya sengaja dirancang demikian di §2.3 dan §2.4 spec justru untuk kemungkinan seperti ini. Yang berubah hanya data.

Yang tetap berlaku dari pengamatan kemarin adalah bentuknya, bukan nilainya: notification id konstan, `Notification.when` bertahan lintas repost, format nominal `Rp1` tanpa pemisah, dan notifikasi transaksi dapat datang lewat channel promosi. Keempatnya mendasari formula `event_id` dan tidak tersentuh.

**Risiko yang justru hilang:** akun merchant hampir hanya menerima, sehingga notifikasi pembayaran **keluar** dengan nominal sama — bahaya utama yang diuraikan di §2.3 spec — praktis tidak ada lagi.

**Langkah berikutnya:** Task 6 (Room) dan Task 7 (konfigurasi terenkripsi). Keduanya tidak bergantung pada VPS maupun pada sampel notifikasi merchant, jadi dikerjakan sementara §2 butir 5–7 dikumpulkan.

---

## 11. Dashboard admin — sub-project 4

Bukan bagian M1–M6 di atas; dicatat terpisah karena sub-project sendiri.
**Ringkasan dashboard:** `PASS` 13 · `FAIL` 0 · `NEEDS-DEVICE` 7 · `PENDING` 5

### Backend (API admin)

| Butir | Status | Bukti |
|---|---|---|
| Login admin: berhasil menerbitkan cookie HttpOnly | `PASS` | `TestAdminLoginBerhasilMenerbitkanCookie` |
| Login admin: password salah ditolak | `PASS` | `TestAdminLoginPasswordSalahDitolak` |
| Login admin: username tak dikenal disamakan dengan password salah | `PASS` | `TestAdminLoginUsernameTakDikenalDisamakanDenganPasswordSalah` |
| Login admin: dibatasi setelah 5 percobaan gagal / 15 menit | `PASS` | `TestAdminLoginDibatasiSetelahBanyakPercobaanGagal` |
| Endpoint admin menolak tanpa sesi, menerima dengan sesi valid, menolak cookie yang diubah | `PASS` | `TestAdminAksesEndpointTerlindungTanpaSesiDitolak`, `...DenganSesiValid`, `TestAdminAksesDenganCookieDiubahDitolak` |
| `ListDevices` tidak pernah membaca kolom secret | `PASS` | `TestListDevicesTidakMengembalikanSecret` |
| Seluruh test Go termasuk paket baru (`store`, `httpapi`, `auth`) | `PASS` | `make test` → seluruh paket `ok`, diulang dengan `-race` → tetap `ok` |
| `go build ./...` dan `go vet ./...` bersih setelah menambah filter events | `PASS` | Dijalankan langsung, tanpa output error |
| `GET /api/v1/events` (dan `/admin/events`) menerima `source`, `q`, `from`, `to` | `PASS` | `TestEventsFiltersBySource`, `TestEventsRejectsUnknownSource`, `TestEventsFiltersByQuery`, `TestEventsFiltersByDateRange`, `TestEventsRejectsBadDateRange` — `make test` dari Akbar, 12 Sep 2026: seluruh paket `ok` (`auth`, `connector`, `httpapi` 2.755s, `secretbox`, `store` 0.796s) |
| `store.DailyEventCounts`: 14 hari selalu terisi, hari sepi bernilai 0, hari ini terhitung benar | `PENDING` | Test baru ditulis (`TestDailyEventCountsMengisiNolUntukHariSepi`, `TestDailyEventCountsMenghitungHariIni`) untuk grafik tren Overview — perlu `make test` sungguhan sebelum `PASS` |
| `GET /api/v1/admin/overview` selalu mengembalikan `events.daily` berisi 14 titik | `PENDING` | Test baru ditulis (`TestAdminOverviewDailySelaluEmpatBelasHari`) — idem, perlu `make test` |
| `go build ./...` dan `go vet ./...` bersih setelah menambah `DailyEventCounts` | `PASS` | Dijalankan langsung, tanpa output error |

Keluaran lengkap:

```
$ go test ./... -race -count=1 -p 1
ok  .../internal/auth        1.019s
ok  .../internal/connector   1.020s
ok  .../internal/httpapi     28.817s
ok  .../internal/secretbox   1.019s
ok  .../internal/store       8.932s
```

Keluaran di atas direkam sebelum filter events (`source`/`q`/`from`/`to`)
ditambahkan. Diulang lagi oleh Akbar 12 Sep 2026 setelah test barunya masuk —
lihat baris filter events di tabel di atas untuk hasilnya.

### Frontend (Next.js)

| Butir | Status | Bukti |
|---|---|---|
| Type-check bersih | `PASS` | `npx tsc --noEmit` → tanpa error |
| Lint bersih | `PASS` | `npx eslint .` → tanpa warning/error |
| Build produksi seluruh route berhasil (termasuk setelah restyle tabel/sidebar, filter Devices/Events, dan grafik tren Overview) | `PASS` | `npx next build` → 5 route (`/`, `/devices`, `/events`, `/login`, `/_not-found`) prerendered, Proxy terdaftar, 0 error/warning |
| Login sungguhan di browser: form, redirect, cookie diterima | `NEEDS-DEVICE` | Perlu `npm run dev` dan interaksi manual — lihat langkah di bawah |
| Grafik tren Overview: kurva sesuai data sungguhan, animasi menggambar halus, hover/crosshair, tabel alternatif | `NEEDS-DEVICE` | `EventsTrendChart` — hitung ulang manual dari `notification_events` vs yang tampil di grafik |
| Overview/Devices/Events menampilkan data sungguhan dari backend dev | `NEEDS-DEVICE` | idem |
| Toggle aktif/nonaktif device dari UI benar-benar mengubah status di database | `NEEDS-DEVICE` | idem |
| Tampilan mobile (sidebar → Sheet) dapat dipakai di layar sempit | `NEEDS-DEVICE` | idem, perlu DevTools atau HP |
| Filter Devices (pencarian nama/ID + status) menyaring tabel dengan benar | `NEEDS-DEVICE` | Filter di sisi klien atas data yang sudah termuat — perlu dicoba di browser |
| Filter Events (pencarian, sumber, rentang tanggal) memanggil backend dan hasilnya benar | `NEEDS-DEVICE` | Bergantung pada endpoint yang testnya sendiri masih `PENDING` di atas — coba di browser setelah `make test` jalan |
| Halaman "Segera" untuk Webhooks/License/Settings/Logs | `PENDING` | Menunggu sub-project 3 fase 2 dan sistem lisensi — Transactions dan API Keys sudah aktif, lihat §12 |
| Auto-refresh berkala (polling) di Overview/Devices | `PENDING` | Belum diimplementasikan — saat ini hanya memuat ulang saat halaman dibuka |
| Halaman Settings (ganti password admin dari UI, bukan CLI) | `PENDING` | `cmd/admintool` cukup untuk MVP |

### Langkah verifikasi `NEEDS-DEVICE` di atas

```bash
cd backend && make run-dev        # jika belum jalan
cd backend && make dev-admin      # buat akun admin, sekali saja

cd dashboard
npm install
cp .env.local.example .env.local
npm run dev
```

Buka `http://localhost:3000`, seharusnya diarahkan ke `/login`. Masuk dengan
akun yang baru dibuat. Periksa: Overview menampilkan angka device/event yang
benar, Devices menampilkan device dari `gopay_dev` dengan status yang sesuai
`heartbeat_at`-nya, tombol Aktifkan/Nonaktifkan benar-benar mengubah kolom
`enabled` di database, Events menampilkan pembayaran yang sudah masuk. Persempit
lebar browser di bawah 768px dan pastikan sidebar berubah jadi tombol menu.

Untuk filter yang baru ditambahkan: di Devices, ketik sebagian nama/ID di
kotak pencarian dan pastikan tabel menyaring baris yang cocok saja, lalu buka
dropdown Status dan pilih salah satu (harus muncul tanda centang di baris yang
sedang aktif, sesuai gaya dropdown di gambar referensi). Di Events, coba
pencarian (device/judul), dropdown Sumber, dan rentang tanggal satu per satu
— tiap kali filter berubah, halaman harus kembali ke offset awal dan hasilnya
konsisten dengan yang benar-benar ada di `notification_events`.

## 12. Invoice, nominal unik, matching, API key — sub-project 3 fase 1

Spec:
[`docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md`](../superpowers/specs/2026-09-12-invoice-nominal-matching-design.md).
**Ringkasan:** `PASS` 23 · `FAIL` 0 · `NEEDS-DEVICE` 0 · `PENDING` 0

Diverifikasi lewat `make test` Akbar, 12 Sep 2026 (setelah dua bug test
ditemukan dan diperbaiki — lihat riwayat commit — dan diulang sekali lagi):

```
$ make test
ok  .../internal/auth        0.003s
ok  .../internal/connector   0.004s
ok  .../internal/httpapi     8.383s
ok  .../internal/secretbox   0.003s
ok  .../internal/store       2.461s
```

### Backend

| Butir | Status | Bukti (test yang menguji) |
|---|---|---|
| Alokasi nominal unik: rentang offset benar, masa berlaku 15 menit | `PASS` | `TestCreateInvoiceMengalokasikanNominalUnik` |
| `external_ref` sama + amount sama → invoice lama dikembalikan (idempotent) | `PASS` | `TestCreateInvoiceExternalRefSamaAmountSamaIdempotent` |
| `external_ref` sama + amount beda → ditolak | `PASS` | `TestCreateInvoiceExternalRefSamaAmountBedaDitolak` |
| Alokasi tidak pernah menabrak nominal invoice PENDING lain | `PASS` | `TestCreateInvoiceMenghindariTabrakanNominal` |
| Invoice kedaluwarsa benar-benar dituliskan EXPIRED sebelum alokasi berikutnya | `PASS` | `TestCreateInvoiceMenulisTransisiExpiredSebelumAlokasi` |
| Matching: amount cocok → PAID + matched_event_id + paid_at | `PASS` | `TestMatchEventMenandaiInvoicePaid` |
| Matching: amount tidak cocok → invoice tidak berubah | `PASS` | `TestMatchEventTidakCocokTidakMengubahApaPun` |
| Matching: amount nil dilewati, bukan error | `PASS` | `TestMatchEventAmountNilDilewati` |
| Matching: race dua event amount sama, hanya satu menang | `PASS` | `TestMatchEventRaceHanyaSatuYangMenang` (dengan `-race`) |
| `ListInvoices` filter status dan pencarian external_ref | `PASS` | `TestListInvoicesFilterStatusDanQuery` |
| API key: hash tersimpan, bukan plaintext; verifikasi benar/salah/dicabut | `PASS` | `TestVerifyAPIKeyBenar`, `...Salah...`, `...SudahDicabut...` |
| API key: revoke idempotent, revoke yang tidak ada → error | `PASS` | `TestRevokeAPIKeyIdempotent`, `TestRevokeAPIKeyTidakDitemukan` |
| `ListAPIKeys` tidak pernah membocorkan hash | `PASS` | `TestListAPIKeysTidakMembocorkanHash` |
| `POST /invoices`: berhasil, idempotent (200 vs 201), konflik (409), key salah/dicabut (401), amount invalid (400) | `PASS` | `TestCreateInvoiceBerhasil`, `...AmountSamaMengembalikan200`, `...AmountBeda409`, `...TanpaAPIKeyDitolak`, `...APIKeySalahDitolak`, `...AmountNolAtauNegatifDitolak` |
| `GET /invoices/{id}`: ditemukan dan tidak ditemukan | `PASS` | `TestGetInvoiceBerhasil`, `TestGetInvoiceTidakDitemukan` |
| Integrasi ujung-ke-ujung: invoice dibuat lewat API, event masuk lewat `POST /events`, matching otomatis tanpa langkah tambahan | `PASS` | `TestInvoiceCocokLewatCallback` |
| `GET /admin/invoices`: data sungguhan, filter status, status tak dikenal ditolak, perlu sesi | `PASS` | `TestAdminInvoicesMenampilkanYangSungguhanAda`, `...FilterStatus`, `...StatusTakDikenalDitolak`, `...MemerlukanSesi` |
| `POST/GET/PATCH /admin/api-keys`: create balas key mentah sekali, list tidak membocorkannya, revoke idempotent, perlu sesi | `PASS` | `TestAdminCreateAPIKeyMengembalikanKeyMentahSekali`, `TestAdminListAPIKeysTidakMenyertakanKeyMentah`, `TestAdminRevokeAPIKeyMenolakKeyBerikutnya`, `...TidakDitemukan`, `TestAdminAPIKeysMemerlukanSesi` |
| `go build ./...`, `go vet ./...`, `gofmt -l .` bersih | `PASS` | Dijalankan langsung, tanpa output error/diff |

### Frontend (Next.js)

| Butir | Status | Bukti |
|---|---|---|
| Type-check, lint, build produksi bersih (termasuk 2 route baru: `/transactions`, `/api-keys`) | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` → 7 route, 0 error/warning |
| Halaman Transactions: filter (pencarian/status/rentang tanggal) memanggil `GET /admin/invoices` dan hasilnya benar | `PASS` | Akbar, 12 Sep 2026: 3 invoice dibuat lewat `POST /invoices` (`ORDER-001/002/003`), satu diubah manual jadi `EXPIRED` untuk uji tampilan; pencarian, dropdown Status, dan rentang tanggal semua menyaring dengan benar |
| Halaman API Keys: buat key baru menampilkan key mentah sekali, dashboard tidak pernah menampilkannya lagi setelah ditutup | `PASS` | Akbar, 12 Sep 2026: key `test-pertama` dibuat, key mentah tampil sekali; response `GET /admin/api-keys` ditempel — tidak membawa field `key`/`key_hash` sama sekali |
| Cabut API key di UI benar-benar membuat key itu ditolak `POST /invoices` berikutnya | `PASS` | Akbar, 12 Sep 2026: `curl` sebelum cabut → `201`; dicabut lewat dashboard; `curl` sesudahnya → `401 unauthenticated` ("API key tidak valid"), ditempel apa adanya |

### Langkah verifikasi

```bash
cd backend
make run-dev             # kalau belum jalan

cd ../dashboard
npm run dev
```

Buka `/api-keys`, buat satu key, salin key mentahnya. Dari terminal lain:

```bash
curl -X POST http://localhost:8090/api/v1/invoices \
  -H "Authorization: Bearer <key mentah>" -H "Content-Type: application/json" \
  -d '{"external_ref":"TEST-1","amount":50000}'
```

Catat `unique_amount` yang dikembalikan. Kirim event lewat perangkat (atau
`POST /api/v1/events` bertanda tangan HMAC) dengan `amount_hint` yang sama,
lalu buka `/transactions` di dashboard — invoice itu harus berubah jadi
`PAID` dengan `matched_event_id` terisi, tanpa langkah manual apa pun.

## 13. Webhook delivery — sub-project 3 fase 2

Spec:
[`docs/superpowers/specs/2026-09-13-webhook-delivery-design.md`](../superpowers/specs/2026-09-13-webhook-delivery-design.md).
**Ringkasan:** `PASS` 22 · `FAIL` 0 · `NEEDS-DEVICE` 0 · `PENDING` 0

Diverifikasi lewat `make test` Akbar, 13 Sep 2026 — seluruh paket `ok`,
setelah dua putaran perbaikan (lihat catatan di bawah tabel).

### Backend

| Butir | Status | Bukti (test yang menguji) |
|---|---|---|
| Enkripsi/dekripsi secret webhook bulat kembali; endpoint tak ditemukan | `PASS` | `TestCreateAndGetWebhookEndpointSecretBulatKembali`, `TestGetWebhookEndpointTidakDitemukan` |
| `ListWebhookEndpoints` ringkasan percobaan terakhir terisi benar | `PASS` | `TestListWebhookEndpointsRingkasanPercobaanTerakhir` |
| Enable/disable endpoint, termasuk yang tidak ada | `PASS` | `TestSetWebhookEndpointEnabled`, `...TidakDitemukan` |
| Delete endpoint meng-cascade riwayat deliveries; delete yang tidak ada | `PASS` | `TestDeleteWebhookEndpointCascadeDeliveries`, `...TidakDitemukan` |
| Enqueue hanya ke endpoint yang enabled dan berlangganan event itu | `PASS` | `TestEnqueueWebhookDeliveriesHanyaEndpointYangBerlangganan`, `...MelewatiEndpointNonaktif` |
| Due deliveries tidak mengambil yang belum jatuh tempo, dan mengklaimnya secara atomik | `PASS` | `TestDueWebhookDeliveriesTidakMengambilYangBelumJatuhTempo` |
| Record sukses berhenti diambil due lagi | `PASS` | `TestRecordDeliverySuccess` |
| Backoff 1→2→4→8→16 menit benar per percobaan, menyerah FAILED permanen di percobaan ke-5 | `PASS` | `TestRecordDeliveryFailureBackoffDanMenyerah` |
| Invoice yang kedaluwarsa memicu invoice.expired tepat sekali, tidak dobel di panggilan berikutnya | `PASS` | `TestExpireInvoicesAndListNewlyExpiredHanyaSekaliPerInvoice` |
| Test delivery tidak pernah RETRYING (langsung DELIVERED/FAILED) | `PASS` | `TestEnqueueAndRecordTestDeliveryTidakPernahRetrying` |
| `POST/GET/PATCH/DELETE /admin/webhooks`: create balas secret sekali, validasi, list tidak membocorkan secret, enable/disable, delete | `PASS` | `TestAdminCreateWebhookMengembalikanSecretSekali`, `TestAdminCreateWebhookValidasi`, `TestAdminListWebhooksTidakMenyertakanSecret`, `TestAdminSetWebhookEnabled`, `...TidakDitemukan`, `TestAdminDeleteWebhook` |
| `POST .../test` mengirim ke endpoint sungguhan (httptest.Server), ditandatangani, tercatat sebagai delivery tanpa invoice_id; endpoint tak ditemukan | `PASS` | `TestAdminTestWebhookMengirimKeEndpointSungguhan`, `...EndpointTidakDitemukan` |
| Seluruh endpoint `/admin/webhooks*` perlu sesi | `PASS` | `TestAdminWebhooksMemerlukanSesi` |
| Integrasi ujung-ke-ujung: invoice.paid terkirim otomatis lewat `POST /events`, tanpa langkah tambahan | `PASS` | `TestWebhookInvoicePaidTerpicuOtomatisLewatCallback` |
| Integrasi: invoice.expired terkirim tepat sekali lewat `ProcessDueWebhooks` | `PASS` | `TestProcessDueWebhooksMengirimInvoiceExpiredTepatSekali` |
| Integrasi: retry sungguhan — server palsu balas 500 lalu 200, terkirim ulang setelah backoff lewat | `PASS` | `TestProcessDueWebhooksRetrySampaiBerhasil` |
| `go build ./...`, `go vet ./...`, `gofmt -l .` bersih | `PASS` | Dijalankan langsung, tanpa output error/diff |

**Dua putaran perbaikan sebelum lulus** (dicatat karena salah satunya bug
produksi sungguhan, bukan cuma bug test):

1. `DueWebhookDeliveries` awalnya `SELECT` polos tanpa mengklaim baris —
   dua pemroses yang jalan bersamaan (ticker 1 menit dan goroutine percobaan
   pertama yang dipicu `invoice.paid`) bisa mengambil delivery yang sama dan
   mengirimnya dua kali ke merchant. Diperbaiki jadi `UPDATE ... RETURNING`
   atomik (pola yang sama dengan `MatchEvent`) — bug produksi sungguhan,
   ditemukan lewat `make test` yang gagal, bukan cuma bug test.
2. Test `TestProcessDueWebhooksRetrySampaiBerhasil` awalnya memberi sinyal
   "percobaan selesai" dari server palsu tepat setelah menerima request —
   padahal goroutine pengirim masih perlu menuliskan hasilnya ke database
   setelah itu. Diperbaiki: test menunggu (polling) sampai status delivery
   benar-benar berubah di database, bukan mengandalkan channel yang
   memberi sinyal terlalu dini.

### Frontend (Next.js)

| Butir | Status | Bukti |
|---|---|---|
| Type-check, lint, build produksi bersih (route baru: `/webhooks`) | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` → 8 route, 0 error/warning |
| Halaman Webhooks: buat endpoint, secret tampil sekali, tidak pernah lagi setelahnya | `PASS` | Akbar, 13 Sep 2026: dibuat via webhook.site, secret tampil sekali, `GET /admin/webhooks` diperiksa lewat DevTools Network — tidak membawa field `secret` |
| Tombol Test mengirim ke endpoint sungguhan dan menampilkan hasil (toast) | `PASS` | Akbar, 13 Sep 2026: endpoint valid → toast sukses + request diterima di webhook.site; endpoint tidak ada → toast gagal, hasilnya beda |
| Baris diperluas menampilkan riwayat pengiriman yang sesuai dengan tabel `webhook_deliveries` | `PASS` | Akbar, 13 Sep 2026: dicocokkan dengan `SELECT event, status, http_status, duration_ms FROM webhook_deliveries` di DBeaver |
| Enable/disable dan hapus webhook dari UI benar-benar mengubah/menghapus baris di database | `PASS` | Akbar, 13 Sep 2026: `enabled` berubah di `webhook_endpoints`; delete meng-cascade — baris endpoint dan seluruh deliveries-nya hilang |

### Langkah verifikasi

```bash
cd backend
make run-dev             # kalau belum jalan

cd ../dashboard
npm run dev
```

Buka `/webhooks`, buat satu endpoint (bisa arahkan ke
[webhook.site](https://webhook.site) atau server lokal sendiri untuk
melihat payload yang diterima), klik **Test** dan pastikan payload
`{"event":"test",...}` diterima dengan header `X-Webhook-Signature` yang
valid (hitung ulang HMAC-SHA256 dengan secret yang ditampilkan saat dibuat,
harus cocok). Lalu buat invoice sungguhan (§12) dan bayar dengan nominal
uniknya — begitu invoice jadi `PAID`, baris pengiriman baru harus muncul di
riwayat webhook dalam beberapa detik tanpa refresh manual berkali-kali.

## 14. Konsol pengecualian — sub-project 3 fase 4 (terakhir)

Spec:
[`docs/superpowers/specs/2026-09-13-exception-console-design.md`](../superpowers/specs/2026-09-13-exception-console-design.md).
**Ringkasan:** `PASS` 18 · `FAIL` 0 · `NEEDS-DEVICE` 0 · `PENDING` 0

Diverifikasi lewat `make test` Akbar, 13 Sep 2026 — seluruh paket `ok`.

### Backend

| Butir | Status | Bukti (test yang menguji) |
|---|---|---|
| Daftar exception mengecualikan event yang sudah cocok invoice manapun | `PASS` | `TestListExceptionsMengecualikanYangSudahCocok` |
| Daftar exception mengecualikan event yang sudah di-dismiss | `PASS` | `TestListExceptionsMengecualikanYangSudahDismiss` |
| Daftar exception menampilkan event yang belum cocok | `PASS` | `TestListExceptionsMenampilkanYangBelumCocok` |
| Daftar exception tidak pernah menampilkan event dengan amount_hint nil | `PASS` | `TestListExceptionsMengecualikanAmountHintNil` |
| Cocok manual berhasil ke invoice PENDING maupun EXPIRED (bayar telat) | `PASS` | `TestManualMatchEventBerhasilKeInvoicePending`, `...KeInvoiceExpired` |
| Cocok manual ditolak ke invoice yang sudah PAID | `PASS` | `TestManualMatchEventGagalKeInvoicePaid` |
| Cocok manual ditolak kalau event sudah dipakai invoice lain (constraint `invoices_matched_event_id_idx` baru) | `PASS` | `TestManualMatchEventGagalEventSudahDipakaiInvoiceLain` |
| Race dua admin mencocokkan event yang sama ke invoice berbeda, hanya satu menang | `PASS` | `TestManualMatchEventRaceHanyaSatuYangMenang` (dengan `-race`) |
| Dismiss berhasil; dismiss dua kali untuk event yang sama ditolak; event yang tidak ada ditolak | `PASS` | `TestDismissEventBerhasil`, `TestDismissEventDuaKaliDitolak`, `TestDismissEventTidakDitemukan` |
| `GET /admin/exceptions` menampilkan event tak cocok; perlu sesi | `PASS` | `TestAdminExceptionsMenampilkanEventTakCocok`, `TestAdminExceptionsMemerlukanSesi` |
| Integrasi ujung-ke-ujung: event masuk tak cocok → muncul di exceptions → dicocokkan manual → invoice PAID + webhook invoice.paid terkirim → event hilang dari exceptions | `PASS` | `TestAdminMatchExceptionUjungKeUjung` |
| `POST .../match` ditolak (409) kalau invoice sudah PAID | `PASS` | `TestAdminMatchExceptionInvoiceSudahPaid` |
| `POST .../dismiss` berhasil sekali, ditolak (409) kalau diulang; event tidak ada → 404 | `PASS` | `TestAdminDismissExceptionBerhasilDanTidakBisaDuaKali`, `TestAdminDismissExceptionTidakDitemukan` |
| `go build ./...`, `go vet ./...`, `gofmt -l .` bersih | `PASS` | Dijalankan langsung, tanpa output error/diff |

### Frontend (Next.js)

| Butir | Status | Bukti |
|---|---|---|
| Type-check, lint, build produksi bersih (route baru: `/exceptions`) | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` → 9 route, 0 error/warning |
| Halaman Exceptions menampilkan event tak cocok yang sesuai dengan query backend | `PASS` | Akbar, 13 Sep 2026: event dengan `amount_hint` sengaja beda dari `unique_amount` invoice muncul di `/exceptions` |
| Dialog Cocokkan: pencarian invoice PENDING/EXPIRED bekerja, memilih lalu konfirmasi benar-benar mengubah invoice jadi PAID dan mengirim webhook | `PASS` | Akbar, 13 Sep 2026: invoice ditemukan lewat pencarian, dicocokkan, berubah `PAID` di `/transactions`, webhook `invoice.paid` tercatat di `/webhooks`, event hilang dari `/exceptions` |
| Dialog Abaikan: catatan tersimpan, event hilang dari daftar setelahnya | `PASS` | Akbar, 13 Sep 2026: dicek lewat `SELECT * FROM event_reviews` — catatan cocok dengan yang diketik, event hilang dari daftar |

### Langkah verifikasi

```bash
cd backend
make run-dev             # kalau belum jalan

cd ../dashboard
npm run dev
```

Buat satu invoice lewat `POST /invoices` (§12), lalu kirim event dengan
nominal yang **sengaja berbeda** dari `unique_amount`-nya (simulasikan
salah ketik). Event itu tidak akan pernah cocok otomatis — buka
`/exceptions`, event itu harus muncul di daftar. Coba **Cocokkan** ke
invoice yang tadi dibuat: invoice harus berubah `PAID` di `/transactions`,
dan kalau ada webhook terdaftar untuk `invoice.paid`, riwayatnya harus
bertambah satu di `/webhooks`. Buat event lain (nominal apa saja yang tidak
cocok invoice manapun), coba **Abaikan** dengan catatan — event itu harus
hilang dari daftar, dan baris di `event_reviews` harus muncul di DBeaver
dengan catatan yang sama.
