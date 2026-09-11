# QA Report

**Milestone terakhir diperiksa:** M1, M2, M3 selesai · **M4 terbukti lewat backend lokal** — pembayaran sungguhan menempuh seluruh rantai sampai tersimpan di Postgres
**Tanggal:** 2026-09-10
**Ringkasan:** `PASS` 60 · `FAIL` 0 · `BLOCKED` 0 · `NEEDS-DEVICE` 2 · `PENDING` 6

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
| Duplicate event ditangani Android | M3 | `PENDING` | — |
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
| `200 duplicate` | → `SENT`, bukan error | `PASS` | `TestCallbackSecondTimeReturnsDuplicate`, e2e no. 2 |
| `400 invalid_payload` | → `FAILED`, tanpa retry | `PASS` | `TestCallbackRejectsMalformedJSON`, `TestCallbackRejectsInvalidFields` (7 subtest) |
| `401 invalid_signature` | → `FAILED`, peringatan kredensial | `PASS` | `TestAuthRejectsWrongSecret`, `TestAuthRejectsUnknownDevice`, e2e no. 3, dan **terbukti di perangkat**: Device Secret sengaja disalahkan → pembayaran sungguhan langsung `FAILED` dengan `invalid_signature` dan berhenti mencoba; setelah secret diperbaiki, Kirim ulang menghasilkan `SENT` dengan `Backend: accepted` dan baris masuk `gopay_dev` pukul 16:21:53. 2026-09-11 |
| `401 clock_skew` | → `FAILED`, pesan jam meleset + jam server | `PASS` | `TestAuthRejectsClockSkewWithServerTime`, e2e no. 4 (`server_time` disertakan) |
| `403 device_disabled` | → `FAILED`, tanpa retry | `PASS` | `TestAuthRejectsDisabledDevice` |
| `429` | tetap `PENDING`, hormati `Retry-After` | `PENDING` | Backend kita tidak menerapkan rate limiting sehingga `429` tidak dapat dihasilkan darinya; ia hanya mungkin datang dari Caddy di depan. Pemetaannya diuji `ResponseMapperTest`, perilaku di perangkat belum pernah terpicu |
| `5xx` | tetap `PENDING`, retry | `PENDING` | Perilaku sisi HP, M4 |
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
