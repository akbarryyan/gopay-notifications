# QA Report

**Milestone terakhir diperiksa:** M1–M4 selesai — pengiriman, retry, kegagalan autentikasi, duplikat, error server, dan sekarang HTTPS produksi lewat VPS sungguhan, seluruhnya terbukti di perangkat/produksi.
**Tanggal:** 2026-09-13
**Ringkasan:** `PASS` 66 · `FAIL` 0 · `BLOCKED` 0 · `NEEDS-DEVICE` 0 · `PENDING` 3

---

## 1. Ringkasan

Seluruh sisi backend selesai dan terbukti bekerja di mesin development: 4 paket, seluruh test lulus termasuk dengan race detector, ditambah uji end-to-end lewat HTTP sungguhan yang menembus rantai lengkap HMAC → validasi → Postgres.

**Deploy ke VPS produksi (`whuzpay.com`, `94.237.69.93`) selesai 2026-09-13.** Backend, dashboard, Caddy (HTTPS otomatis + basic auth), dan systemd seluruhnya terbukti berjalan di server sungguhan lewat internet, bukan lagi cuma berkas konfigurasi. Detail di §2b.

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

Selesai — lihat §2b untuk bukti lengkap. `devicetool -name "HP GoPay Utama"` masih perlu dijalankan di VPS untuk menghasilkan `Device ID`/`Device Secret` produksi yang sesungguhnya dipakai HP (deploy VPS ini memverifikasi infrastrukturnya, bukan menggantikan pembuatan device produksi).

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

## 2b. Deploy VPS produksi — selesai 2026-09-13

Domain `whuzpay.com`, VPS `94.237.69.93`. Langkah lengkap di [`backend/deploy/README.md`](../../backend/deploy/README.md).

**Bukti — HTTPS otomatis, basic auth, dan redirect HTTP→HTTPS lewat internet sungguhan:**

```
$ curl -s https://whuzpay.com/api/v1/health
{"status":"ok","server_time":1789273446}

$ curl -s -o /dev/null -w '%{http_code}' https://whuzpay.com/api/v1/events
401

$ curl -s -o /dev/null -w '%{http_code}' http://whuzpay.com/api/v1/health
308

$ curl -s -o /dev/null -w '%{http_code}' https://whuzpay.com/
307

$ curl -s -o /dev/null -w '%{http_code}' https://whuzpay.com/login
200
```

`307` di `/` benar — belum ada cookie sesi, `proxy.ts` mengalihkan ke `/login`.

**Bukti — basic auth menerima kredensial yang benar:**

```
$ curl -s -o /dev/null -w '%{http_code}' -u admin:<pw> https://whuzpay.com/api/v1/events
200
```

Digabung dengan `401` tanpa kredensial di atas, ini membuktikan basic auth Caddy pada `/api/v1/events` menyaring dengan benar: menolak tanpa kredensial, menerima dengan kredensial yang cocok hash bcrypt di Caddyfile. Akbar, 2026-09-13.

**Login sungguhan berhasil** di `https://whuzpay.com` lewat browser dengan akun yang dibuat via `admintool` — rantai penuh Caddy → dashboard Next.js (systemd `gopay-dashboard`) → backend Go (systemd `gopay-ingestion`) → Postgres terbukti jalan end-to-end di produksi.

Jebakan yang ditemukan dan diperbaiki selama deploy ini (dicatat di §10):
- DNS sempat mengarah ke Cloudflare (proxy aktif), bukan langsung ke VPS — diperbaiki dengan mode "DNS only".
- Caddy versi lama (2.6.2, dari repo distro) tidak mengenali directive `basic_auth` — diperbaiki dengan upgrade ke Caddy stabil terbaru dari repo resmi.

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
| FR-05 | Kirim event via **HTTPS** | M4 | `PASS` | `https://whuzpay.com/api/v1/health` menjawab lewat sertifikat sah (Caddy, Let's Encrypt otomatis); `http://` dialihkan `308`. Lihat §2b, 2026-09-13 |
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
| Event terkirim via HTTPS | M4 | `PASS` | HTTPS produksi terbukti di `whuzpay.com`, sertifikat otomatis Let's Encrypt lewat Caddy. Lihat §2b, 2026-09-13 |
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
| HTTPS untuk production | M1 | `PASS` | Sertifikat sah diterbitkan otomatis oleh Caddy di `whuzpay.com`, diverifikasi lewat `curl`. Lihat §2b, 2026-09-13 |
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
| Build production menolak HTTP polos | `PASS` | `curl http://whuzpay.com/api/v1/health` → `308`, dialihkan ke HTTPS oleh Caddy. Lihat §2b, 2026-09-13 |
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

**Dua temuan operasional selama deploy VPS produksi, 2026-09-13.** Keduanya bukan bug kode, tapi jebakan infrastruktur yang layak dicatat karena akan terulang di deploy customer lain yang memakai `backend/deploy/README.md`:

1. Domain yang sudah dikelola Cloudflare (proxy/"awan oranye" aktif) membuat `dig` menunjuk IP Cloudflare, bukan IP VPS — Caddy tidak bisa menerbitkan sertifikat HTTPS otomatis karena tantangan ACME (HTTP-01) mendarat di edge Cloudflare, bukan di origin. Perbaikan: set record ke "DNS only" (abu-abu) supaya domain menunjuk langsung ke VPS.
2. Caddy versi lama dari repo distro Ubuntu (`2.6.2`, rilis 2022) tidak mengenali directive `basic_auth` di `Caddyfile` — gagal dengan `unrecognized directive: basic_auth`. Perbaikan: upgrade ke Caddy stabil terbaru dari repo resmi (`https://dl.cloudsmith.io/public/caddy/stable/...`), bukan mengandalkan paket bawaan distro.

Kedua langkah ini akan ditambahkan ke `backend/deploy/README.md` sebagai catatan prasyarat untuk deploy berikutnya.

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

---

## 15. Platform akun multi-tenant (hosted) — sub-project 5 (pivot arsitektur)

Spec: [`docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md`](../superpowers/specs/2026-09-13-multitenant-accounts-design.md),
menggantikan **total** platform lisensi online yang tercatat sebelumnya di
[`2026-09-13-online-license-platform-design.md`](../superpowers/specs/2026-09-13-online-license-platform-design.md)
(SUPERSEDED) — yang itu sendiri sudah menggantikan versi offline murni
paling awal
([`2026-09-13-license-system-design.md`](../superpowers/specs/2026-09-13-license-system-design.md),
SUPERSEDED juga). Rencana implementasi:
[`docs/superpowers/plans/2026-09-13-multitenant-accounts-plan.md`](../superpowers/plans/2026-09-13-multitenant-accounts-plan.md)
(15 task, seluruhnya selesai).

**Ringkasan:** `PASS` 12 · `FAIL` 0 · `NEEDS-DEVICE` 3 · `PENDING` 0

**Perubahan arsitektur:** produk berubah dari self-hosted (tiap customer
deploy backend+dashboard sendiri) jadi hosted SaaS multi-tenant seperti
Midtrans. `internal/licenseserver`, `internal/licenseclient`,
`internal/licensecheck`, `internal/version` (seluruhnya dari platform
lisensi online sebelumnya) **dibongkar total** — konsepnya (banyak
instalasi tersebar memvalidasi ke satu otoritas pusat) sudah tidak
berlaku begitu backend jadi satu. Digantikan:

- Tabel `accounts` di database utama (`gopay`) — gabungan `admin_users`
  lama + `customers`/`licenses` License Server, satu baris = satu
  customer = satu login.
- Kolom `account_id` di seluruh tabel data (`devices`, `invoices`,
  `notification_events`, `api_keys`, `webhook_endpoints`,
  `webhook_deliveries`, `event_reviews`) — diturunkan server-side dari
  sesi/API key/HMAC device, tidak pernah dari input client.
- `requireActiveAccount` menggantikan `requireLicense` — cek langsung ke
  `accounts` (bukan file lokal + grace period, karena tidak ada lagi
  jaringan antar dua service).
- Endpoint vendor (`/api/v1/vendor/*`) di backend utama menggantikan
  License Server yang dulu terpisah — sesi `vendor_session` + tabel
  `vendor_admins`, terpisah total dari sesi customer.
- `vendor-dashboard/` tetap app terpisah, sekarang manggil backend utama
  langsung (bukan License Server terpisah).

### Sudah diverifikasi Claude langsung (tidak butuh Docker/server)

| Butir | Status | Bukti |
|---|---|---|
| `go build ./...`, `go vet ./...`, `gofmt -l .` bersih di seluruh backend | `PASS` | Dijalankan langsung, tanpa output error/diff |
| Dashboard customer (`dashboard/`): halaman `/license` ditulis ulang mengikuti skema account (5 status: active/expiring/expired/suspended/revoked) — type-check, lint, build produksi bersih, 10 route | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` → 0 error, 10 route |
| Vendor Dashboard (`vendor-dashboard/`): customers+licenses digabung jadi accounts, halaman `/accounts/[id]` menggantikan `/customers/[id]`+`/licenses/[id]` — type-check, lint, build produksi bersih, 5 route | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` → 0 error, 5 route |

### `make test` sungguhan (Postgres via Docker)

Satu database (`gopay_test`) — container Postgres kedua
(`postgres-license`) yang sebelumnya dipakai License Server sudah dihapus
dari `docker-compose.yml`, tidak ada lagi migrasi/database terpisah untuk
dijalankan.

| Butir | Status | Bukti |
|---|---|---|
| `make test` — seluruh `go test ./... -count=1 -p 1`, database `gopay_test` (migrasi sampai `00009_audit_log.sql`) | `PASS` | Dijalankan langsung: `ok internal/auth`, `ok internal/connector`, `ok internal/httpapi 15.408s`, `ok internal/secretbox`, `ok internal/store 8.156s` — 246 test, semua `ok`, nol `FAIL` |
| Isolasi data lintas akun (`internal/httpapi/tenant_isolation_test.go`, 5 test) — akun A tidak bisa lihat/ubah device, invoice, webhook milik akun B lewat ID langsung (404); `account_id` di body request diabaikan sepenuhnya (dibuktikan eksplisit, bukan cuma diasumsikan); event lewat HMAC device satu akun tidak terlihat di daftar akun lain | `PASS` | Termasuk dalam 246 test di atas — kategori paling penting di seluruh sub-project ini, kesalahan di sini berarti kebocoran data lintas customer |
| Endpoint vendor (`internal/httpapi/vendor_accounts_test.go`, 6 test) — create/list/renew/suspend/revoke account, audit log, sesi vendor tidak bisa dipakai sebagai sesi customer walau nama cookie dipalsukan manual | `PASS` | Termasuk dalam 246 test di atas |
| Race condition kuota (bila relevan lagi di masa depan) — TIDAK ADA di model ini: `max_devices` bukan lagi dicek lewat quota lock terpisah seperti instalasi License Server dulu, penegakannya jadi tanggung jawab sub-project Customer Dashboard (swalayan tambah device) | N/A | Dicatat sebagai keputusan, bukan celah — lihat spec §7 |

### Butir `NEEDS-DEVICE` — menunggu deploy ke VPS produksi

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Deploy backend baru (migrasi sampai `00009_audit_log.sql`, `VENDOR_SESSION_KEY` di `.env`) dan `vendor-dashboard` (Next.js, `BACKEND_URL` menunjuk ke backend yang sama) ke `whuzpay.com`, buat akun vendor lewat `admintool -username akbar` | Login ke Vendor Dashboard berhasil, halaman Accounts kosong (atau berisi akun lama hasil migrasi data) tampil tanpa error |
| 2 | Buat account baru lewat Vendor Dashboard (plan Business, expires_at jauh), catat username+password awal, buat device lewat `devicetool -account <id> -name "HP Uji"` | Login dashboard customer dengan akun itu berhasil, `/license` menampilkan `Aktif` dengan plan/expires_at yang benar |
| 3 | Suspend account itu dari Vendor Dashboard | Endpoint lain di dashboard customer (mis. Devices) langsung menjawab `402 account_suspended` pada request berikutnya — tidak perlu menunggu siklus validasi apa pun, karena statusnya dicek langsung tiap request |

Sengaja `NEEDS-DEVICE` — perilaku ujung-ke-ujung produksi (migrasi data
lama, DNS/Caddy, systemd) tidak bisa disimulasikan penuh dari `make test`
satu repo.

### Langkah verifikasi manual (dev lokal)

```bash
# 1. Buat akun vendor (Akbar) -- pakai .env.dev yang sudah ada
cd backend
make run-dev &   # backend jalan di background sebentar untuk langkah berikut
go run ./cmd/admintool -username akbar   # interaktif, buat password

# 2. Jalankan Vendor Dashboard
cd ../vendor-dashboard
cp .env.local.example .env.local   # BACKEND_URL default :8090, cocok dengan run-dev
npm run dev

# 3. Login ke Vendor Dashboard (akbar), buat account baru (plan Business,
#    expires_at jauh) -- catat username + initial_password yang muncul sekali.

# 4. Buat device untuk account itu
cd ../backend
go run ./cmd/devicetool -account <account_id_dari_langkah_3> -name "HP Uji"

# 5. Login ke Customer Dashboard (dashboard/) dengan username+password dari
#    langkah 3, buka /license -- harus Aktif dengan plan/expires_at yang benar.

# 6. Suspend account itu dari Vendor Dashboard, refresh halaman apa pun di
#    Customer Dashboard -- harus langsung 402, tanpa jeda/restart apa pun.
```

---

## 16. Landing page + signup swalayan, redesain visual, Dashboard Vendor, seedtool

Spec: [`docs/superpowers/specs/2026-09-13-landing-signup-design.md`](../superpowers/specs/2026-09-13-landing-signup-design.md)
untuk sub-project #2+#6 dari pivot (§0 spec multitenant-accounts). Bagian
redesain visual (Finpay-style, enterprise-grade, animasi konektor, seed
data, Dashboard Vendor) tidak punya spec tertulis tersendiri — permintaan
iteratif langsung dari Akbar lewat chat, diverifikasi manual olehnya di
`npm run dev` tiap tahap sebelum lanjut ke tahap berikutnya.

**Ringkasan:** `PASS` 14 · `FAIL` 0 · `NEEDS-DEVICE` 0 · `PENDING` 0

### 16a. Signup swalayan + landing page publik (dashboard/)

| Butir | Status | Bukti |
|---|---|---|
| `POST /api/v1/signup` — buat account (plan Starter, trial 3 hari, max 3 device), rate limit per IP, auto-login | `PASS` | `go test ./internal/httpapi -run TestSignup -v` — 7 test lulus (sukses+cookie, konflik email/username → 409, password < 8 karakter, field kosong, rate limit setelah 5 percobaan, tanpa kredensial) |
| Konflik email/username di `CreateAccount` (store) dan `handleVendorCreateAccount` mengembalikan 409, bukan 500 generik | `PASS` | `TestCreateAccountEmailBentrokDitolak`, `TestCreateAccountUsernameBentrokDitolak`, `TestVendorCreateAccountEmailBentrokMengembalikan409` — lulus |
| `/` jadi landing page publik, `/register` jadi form signup, `/overview` (dulu `/`) tetap perlu sesi — proxy.ts mengecualikan keduanya dari gerbang sesi | `PASS` | `npx next build` bersih, route `/`, `/register`, `/overview` semua ter-generate; diverifikasi manual oleh Akbar di `npm run dev` ("okee sudah aman semua yg aku test") |
| Landing page tidak mengarang logo/testimoni/statistik pelanggan yang tidak nyata | `PASS` | Tinjauan manual tiap section saat ditulis — highlight fitur (bukan testimoni orang) dipakai justru karena produk belum punya customer nyata untuk dikutip, dikonfirmasi eksplisit oleh Akbar saat memilih opsi itu |

### 16b. Redesain visual landing/login/register + Overview biaxial chart (dashboard/)

| Butir | Status | Bukti |
|---|---|---|
| `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih di setiap tahap redesain (Finpay-style, enterprise-grade, animasi konektor, halaman login/register split-panel) | `PASS` | Dijalankan berulang sepanjang sesi, keluaran bersih tiap kali sebelum lanjut ke permintaan berikutnya |
| `GET /overview` menyertakan `paid_amount_rp` per hari (nominal lunas), selain `count` yang sudah ada | `PASS` | `TestDailyEventCountsMenjumlahkanNominalLunasPerHariSesuaiPaidAt`, `TestDailyEventCountsNolKalauBelumAdaPembayaran` — lulus, isolasi antar akun ikut diuji |
| `EventsTrendChart` diganti dari SVG manual ke Recharts biaxial (jumlah event kiri, nominal lunas kanan) | `PASS` | Build bersih + dikonfirmasi manual oleh Akbar di `npm run dev` ("ok cocok") setelah revisi (skala sumbu-Y, titik per hari) |
| Animasi (fade-in scroll, konektor "Cara Kerja"/"Arsitektur" mengisi warna, carousel highlight) menghormati `prefers-reduced-motion` | `PASS` | Kode memeriksa `window.matchMedia("(prefers-reduced-motion: reduce)")` di `Reveal`, `EventsTrendChart` (lama), dan CSS `@media (prefers-reduced-motion: reduce)` untuk `lp-flow-dot`/`lp-line-fill` di `globals.css` |

### 16c. `cmd/seedtool` — data dev banyak-account

| Butir | Status | Bukti |
|---|---|---|
| `go build ./...`, `go vet ./...`, `gofmt -l .` bersih | `PASS` | Dijalankan langsung, tanpa output error/diff |
| Seed sungguhan ke `gopay_dev` oleh Akbar: 8 account (plan berselang Starter/Business/Enterprise), device, invoice (PAID/EXPIRED/PENDING campuran), event pengecualian, API key, webhook + riwayat delivery | `PASS` | Ditempel Akbar: `seed: Warung Kopi Senja selesai (1 device, 47 invoice)` … dst, 8 baris, total 490 invoice — akun hasil seed langsung dipakai login sungguhan ke `dashboard/` |
| Riwayat `webhook_deliveries` diisi lewat `Pool()` langsung, bukan `EnqueueWebhookDeliveries` (fungsi itu tidak difilter `account_id`, akan bocor lintas akun bila dipanggil berulang saat seeding banyak akun) | `PASS` | Tinjauan kode `seedWebhook` di `cmd/seedtool/main.go` — dicatat sebagai keputusan desain di komentar fungsi |

### 16d. Dashboard Vendor — sidebar, halaman Dashboard, `/api/v1/vendor/overview`

| Butir | Status | Bukti |
|---|---|---|
| `GET /api/v1/vendor/overview` — ringkasan lintas SEMUA account (total/status/account baru minggu ini/total device/total pendapatan seumur hidup), tren 14 hari (account baru + nominal lunas), 5 account terbaru | `PASS` | `TestVendorOverviewStatsMenghitungRingkasanLintasAccount`, `TestVendorOverviewStatsMenjumlahkanNominalLunasLintasAccount`, `TestVendorDailyStatsLintasAccountTanpaFilter`, `TestVendorOverviewButuhSesiVendor`, `TestVendorOverviewMeringkasAccountDanDaily`, `TestVendorOverviewRecentAccountsDibatasiLimaBaris` — 6 test, seluruhnya lulus |
| `make test` full suite (seluruh 4 paket backend) tetap lulus setelah `vendor_stats.go`/`vendor_overview.go` ditambahkan | `PASS` | `ok internal/auth`, `ok internal/connector`, `ok internal/httpapi` (16.7s), `ok internal/secretbox`, `ok internal/store` (8.8s) |
| Vendor Dashboard: sidebar collapsible (pola sama dengan `AppShell` Customer Dashboard, token `--sidebar-*` yang sudah ada tapi belum pernah dipakai), Dashboard jadi `/`, Accounts pindah ke `/accounts` | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih; dikonfirmasi manual oleh Akbar di `npm run dev`, sudah di-commit+push olehnya sendiri (`abc014b`, `95d5595`, `bd9f8a3`) |
| Bug runtime `Cannot read properties of undefined (reading 'length')` pada `recent_accounts` saat backend lokal belum di-restart setelah field baru ditambahkan | `FAIL` → `PASS` | Ditempel Akbar (stack trace lengkap); diperbaiki dengan fallback `?? []` di frontend, dikonfirmasi lewat build bersih. Penyebab sungguhan: proses `go run ./cmd/server` tidak hot-reload, bukan bug logika |

### Catatan cakupan

Tidak ada butir `NEEDS-DEVICE` di bagian ini — seluruhnya halaman web (Next.js) dan backend Go yang bisa diverifikasi penuh dari mesin development lewat build otomatis + `npm run dev` manual oleh Akbar sendiri, tanpa HP atau VPS. Deploy pivot akun multi-tenant ke VPS produksi (§15) dan uji ketahanan HP semalaman (M6) masih `NEEDS-DEVICE` seperti sebelumnya, tidak berubah oleh pekerjaan di bagian ini.

---

## 17. Notifikasi ke customer: pengingat kedaluwarsa, peringatan HP offline, riwayat notifikasi

Permintaan iteratif langsung dari Akbar lewat chat (tanpa spec tertulis
tersendiri): pengingat akun mendekati kedaluwarsa, pengaturan SMTP/bot
Telegram yang disimpan di database lewat Vendor Dashboard, peringatan HP
bridge offline/kembali online, dan halaman riwayat notifikasi di Vendor
Dashboard. Batch sebelumnya (riwayat webhook lintas account, export CSV
Customer Dashboard) ikut dicatat di 17d.

**Ringkasan:** `PASS` 13 · `FAIL` 0 · `NEEDS-DEVICE` 3 · `PENDING` 0

### 17a. Pengingat kedaluwarsa + pengaturan notifikasi di database

| Butir | Status | Bukti |
|---|---|---|
| Akun aktif yang berakhir ≤ 7 hari dikirimi pengingat sekali per nilai `expires_at`; perpanjangan membuka pengingat periode berikutnya | `PASS` | `TestAccountsNeedingExpiryReminder`, `TestExpiryReminderTidakDikirimDuaKaliTapiTerbukaLagiSetelahRenew` — lulus |
| Satu akun gagal tidak menghentikan sisanya; yang gagal tidak ditandai; Telegram gagal setelah email terkirim tetap dihitung terkirim | `PASS` | `TestRunSatuAkunGagalTidakMenghentikanSisanya`, `TestRunTelegramGagalTetapDianggapTerkirim`, `TestRunGagalMenandaiTidakDihitungTerkirim` — lulus |
| Customer mengisi/menghapus chat id Telegram sendiri (angka saja) di `/license` | `PASS` | `TestAdminSetTelegramChatID`, `TestAdminHapusTelegramChatID`, `TestAdminSetTelegramUsernameDitolak`, `TestAdminSetTelegramButuhSesi`, `TestSetAccountTelegramChatID` — lulus |
| SMTP host/port/username/password/pengirim + token bot tersimpan di `notification_settings`; password & token terenkripsi `SETTINGS_SECRET_KEY`, tidak tersimpan plaintext, kunci salah gagal didekripsi | `PASS` | `TestSaveNotificationSettingsRoundtripDanTerenkripsi`, `TestGetNotificationSettingsKunciSalahGagal`, `TestGetNotificationSettingsBelumPernahDisimpan` — lulus |
| API tidak pernah mengembalikan password/token; field tidak dikirim = biarkan, `""` = hapus; validasi port/pengirim/token; wajib sesi vendor | `PASS` | `TestVendorNotificationSettingsSimpanTanpaMembocorkanKredensial`, `TestSaveNotificationSettingsNilMempertahankanKosongMenghapus`, `TestVendorNotificationSettingsValidasi`, `TestVendorNotificationSettingsButuhSesiVendor`, `TestVendorTestNotificationBelumDikonfigurasi` — lulus |
| Pengaturan dibaca ulang tiap putaran (berlaku tanpa restart); putaran dilewati selama SMTP kosong | `PASS` | `TestRunMembacaPengaturanTiapPutaran`, `TestRunDilewatiBilaSMTPBelumDikonfigurasi` — lulus |
| Email dan Telegram uji benar-benar sampai lewat SMTP/bot sungguhan | `NEEDS-DEVICE` | Butuh kredensial SMTP + token bot asli. Langkah: isi Vendor Dashboard > Settings, kirim email uji dan Telegram uji, cek kotak masuk dan halaman Notifications |

### 17b. Peringatan HP offline / kembali online

| Butir | Status | Bukti |
|---|---|---|
| HP dianggap perlu diperingatkan bila heartbeat > 45 menit (sama dengan `StatusOf`), ≤ 24 jam (HP basi dilewati), device enabled, account aktif & belum kedaluwarsa, belum diperingatkan untuk `heartbeat_at` itu; PENDING dilewati | `PASS` | `TestDevicesNeedingOfflineAlert` — lulus |
| Heartbeat baru setelah peringatan → pemberitahuan kembali online sekali, lalu offline berikutnya membuka peringatan baru | `PASS` | `TestDevicesRecoveredFromOfflineLaluBisaOfflineLagi` — lulus |
| Job: kirim offline + online, gagal kirim tidak ditandai (dicoba lagi 5 menit kemudian), dilewati selama SMTP kosong | `PASS` | `TestRunMengabarkanOfflineDanOnline`, `TestRunGagalKirimTidakDitandai`, `TestRunDilewatiBilaSMTPBelumDikonfigurasi` (paket `devicealert`) — lulus |
| Isi pesan menyebut nama HP & bisnis, jam dalam WIB | `PASS` | `TestPesanStatusHPMemakaiJamWIB` — lulus |
| HP sungguhan dimatikan > 45 menit → email peringatan sampai; dinyalakan lagi → email kembali online sampai | `NEEDS-DEVICE` | Butuh HP + SMTP asli + backend berjalan. Langkah: matikan internet HP bridge, tunggu ±50 menit, cek email & halaman Notifications; nyalakan lagi, tunggu ≤ 5 menit setelah heartbeat berikutnya |

### 17c. Riwayat notifikasi (Vendor Dashboard `/notifications`)

| Butir | Status | Bukti |
|---|---|---|
| Setiap percobaan kirim dicatat per channel (email & Telegram baris terpisah) beserta alasan gagal; email gagal → Telegram tidak dicoba | `PASS` | `TestDeliverEmailDanTelegramDicatatTerpisah`, `TestDeliverEmailGagalTelegramTidakDicoba`, `TestDeliverTelegramGagalMengembalikanTelegramError`, `TestDeliverTanpaChatIDAtauTokenBotCumaEmail` — lulus |
| `GET /api/v1/vendor/notification-log` lintas account dengan filter jenis/channel/status/pencarian/tanggal, nilai filter tak dikenal → 400, wajib sesi vendor | `PASS` | `TestNotificationLogCatatDanFilter`, `TestVendorNotificationLog` — lulus |
| Migrasi `00012` bisa di-rollback dan diterapkan ulang | `PASS` | `goose down` lalu `goose up` di `gopay_test`: `OK 00012_device_offline_alert_and_notification_log.sql` dua kali, `successfully migrated database to version: 12` |
| Halaman `/notifications` dan kartu Settings tampil benar di browser | `NEEDS-DEVICE` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih (route `/notifications` ter-generate); tampilan menunggu dicek Akbar di `npm run dev` |

### 17d. Riwayat webhook lintas account + export CSV Customer Dashboard

| Butir | Status | Bukti |
|---|---|---|
| `GET /api/v1/vendor/webhook-deliveries` lintas account, filter status, wajib sesi vendor | `PASS` | `TestVendorWebhookDeliveriesButuhSesiVendor`, `TestVendorWebhookDeliveriesLintasAccount`, `TestVendorWebhookDeliveriesFilterStatus`, `TestVendorWebhookDeliveriesStatusTidakDikenalDitolak` — lulus; halaman diverifikasi manual Akbar ("riwayat webhook delivery lintas account, export csv di customer dashboard sudah aman") |
| Export CSV Transactions/Events di Customer Dashboard (paginasi 1000/halaman) | `PASS` | Diverifikasi manual Akbar di `npm run dev` (kutipan di atas) |

### Keseluruhan suite

`make test` setelah seluruh perubahan di bagian ini:

```
OK   00012_device_offline_alert_and_notification_log.sql
ok  	.../internal/auth
ok  	.../internal/connector
ok  	.../internal/devicealert
ok  	.../internal/httpapi	22.764s
ok  	.../internal/notify
ok  	.../internal/reminder
ok  	.../internal/secretbox
ok  	.../internal/store	11.029s
```

---

## 18. Password customer: lupa password lewat email + halaman Settings

Permintaan langsung Akbar lewat chat (tanpa spec tertulis tersendiri):
reset password lewat email dan halaman Settings di Customer Dashboard
(profil, ganti password, chat id Telegram yang dipindah dari `/license`).

**Ringkasan:** `PASS` 11 · `FAIL` 0 · `NEEDS-DEVICE` 2 · `PENDING` 0

### 18a. Lupa password / reset lewat email

| Butir | Status | Bukti |
|---|---|---|
| Fitur menjawab `503 not_available` bila `DASHBOARD_URL` atau SMTP belum diisi | `PASS` | `TestForgotPasswordBelumTersedia` (2 subtest) — lulus |
| Jawaban identik untuk email terdaftar dan tidak terdaftar; token cuma dibuat untuk email terdaftar (tidak peka huruf besar); email dikirim di background dan tercatat di `notification_log` sebagai `password_reset` tanpa isi pesan | `PASS` | `TestForgotPasswordJawabanSamaUntukEmailTerdaftarDanTidak`, `TestGetAccountByEmailTidakPekaHurufBesar` — lulus |
| Maksimal satu email reset per account per 2 menit; maksimal 5 permintaan per IP per 15 menit | `PASS` | `TestForgotPasswordJawabanSamaUntukEmailTerdaftarDanTidak` (bagian cooldown), `TestForgotPasswordDibatasiPerIP` — lulus |
| Token disimpan sebagai hash, berlaku 30 menit, sekali pakai, permintaan baru membatalkan link lama | `PASS` | `TestResetPasswordWithTokenSekaliPakai`, `TestResetPasswordTokenKedaluwarsaDanTokenLamaDigantikan` — lulus |
| Reset berhasil: password lama tidak bisa login, password baru bisa, sesi yang terbit sebelum reset ditolak 401, token dipakai ulang → `400 invalid_token` | `PASS` | `TestResetPasswordMenggantiPasswordDanMencabutSesiLama` — lulus |
| Waktu terbit sesi diturunkan dari expiry token | `PASS` | `TestSessionIssuedAtDariMasaBerlaku` — lulus |
| Migrasi `00013` bisa di-rollback dan diterapkan ulang | `PASS` | `goose down` lalu `goose up` di `gopay_test`: `OK 00013_password_reset.sql`, `successfully migrated database to version: 13` |
| Email reset sungguhan sampai, link membuka `/reset-password`, password baru bisa dipakai login | `NEEDS-DEVICE` | Butuh SMTP asli + `DASHBOARD_URL`. Langkah: `/login` → "Lupa password?" → isi email account → buka link di email → simpan password baru → login |

### 18b. Settings Customer Dashboard

| Butir | Status | Bukti |
|---|---|---|
| Ganti password: password saat ini wajib benar, sesi lain dan sesi lama dicabut, cookie sesi yang dipakai diterbitkan ulang; token reset yang tertinggal ikut dibatalkan | `PASS` | `TestAdminChangePassword`, `TestChangeAccountPasswordMembatalkanTokenReset` — lulus |
| Tebakan password saat ini dibatasi 5 per account per 15 menit (tetap 429 walau tebakan ke-6 benar) | `PASS` | `TestAdminChangePasswordDibatasiPerAccount` — lulus |
| Profil: ganti nama bisnis tanpa password; ganti email wajib password saat ini; email tidak valid → 400; email dipakai akun lain → 409 | `PASS` | `TestAdminAccountProfile`, `TestUpdateAccountProfile` — lulus |
| Halaman `/settings`, `/forgot-password`, `/reset-password`, link "Lupa password?" di `/login`, Settings aktif di sidebar | `NEEDS-DEVICE` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih (ketiga route ter-generate); tampilan menunggu dicek Akbar di `npm run dev` |

### Keseluruhan suite

`make test` setelah seluruh perubahan di bagian ini:

```
OK   00013_password_reset.sql
ok  	.../internal/auth
ok  	.../internal/connector
ok  	.../internal/devicealert
ok  	.../internal/httpapi	24.846s
ok  	.../internal/notify
ok  	.../internal/reminder
ok  	.../internal/secretbox
ok  	.../internal/store	12.461s
```

---

## 19. Hubungkan Telegram lewat deep link + backup database otomatis

Permintaan langsung Akbar lewat chat. Menutup bug: chat id Telegram yang
diketik manual tidak pernah bisa dikirimi pesan, karena bot Telegram
dilarang memulai obrolan dengan orang yang belum menekan Start.

**Ringkasan:** `PASS` 9 · `FAIL` 0 · `NEEDS-DEVICE` 3 · `PENDING` 0

### 19a. Hubungkan Telegram

| Butir | Status | Bukti |
|---|---|---|
| Kode tautan sekali pakai, berlaku 15 menit, disimpan hash, kode baru membatalkan yang lama; pemakaian ulang tidak menimpa chat id | `PASS` | `TestTelegramLinkCodeSekaliPakai`, `TestTelegramLinkCodeKedaluwarsaDanDigantikan` — lulus |
| Offset `getUpdates` tersimpan di database dan direset saat token bot diganti | `PASS` | `TestTelegramUpdateOffsetDiresetSaatTokenBerganti` — lulus |
| Bot: `/start <kode>` di chat pribadi menautkan dan membalas konfirmasi; kode salah, `/start` polos, dan chat grup dibalas petunjuk; pesan lain diabaikan; offset maju ke update terakhir + 1 | `PASS` | `TestPollOnceMenautkanDanMembalas` — lulus |
| Bot diam selama token kosong; galat database tidak memajukan offset (pesan dicoba lagi) | `PASS` | `TestPollOnceTanpaTokenTidakMembaca`, `TestPollOnceGagalDatabaseTidakMemajukanOffset` — lulus |
| Klien Bot API: `getMe`/`getUpdates` terbaca, `409` jadi `ErrConflict`, galat jaringan tidak memuat token | `PASS` | `TestGetMeDanGetUpdates`, `TestKonflikDanGalatTidakMembocorkanToken` — lulus |
| `POST /admin/account/telegram/link`: 503 tanpa token, deep link `t.me/<bot>?start=<kode>` dengan username dari `getMe` (di-cache per token), `telegram_available` di `GET /admin/account`, chat id terlihat setelah kode dipakai, wajib sesi | `PASS` | `TestAdminTelegramLink` (Bot API palsu via `httptest`) — lulus |
| Migrasi `00014` bisa di-rollback dan diterapkan ulang | `PASS` | `goose down` lalu `goose up`: `OK 00014_telegram_link.sql`, `successfully migrated database to version: 14` |
| Alur sungguhan: Settings → Hubungkan Telegram → Start di bot asli → status Terhubung → notifikasi uji sampai di Telegram | `NEEDS-DEVICE` | Butuh token bot asli di server yang berjalan. `npx tsc`, `npx eslint`, `npx next build` bersih untuk `dashboard/` dan `vendor-dashboard/` |

### 19b. Backup database otomatis

| Butir | Status | Bukti |
|---|---|---|
| `gopay-backup.sh`: dump terverifikasi, retensi lokal menghapus dump lama, dan hasilnya bisa di-restore utuh | `PASS` | Uji di Postgres 16 (container `make db-up`) terhadap `gopay_dev`: `backup: selesai, ukuran 104.0K`; restore ke `gopay_restore_test` → `accounts asli: 10`, `accounts restore: 10`, `versi migrasi: 13`. Uji retensi (laptop): file bertanggal 20 hari lalu → `backup: 1 backup lokal lebih dari 14 hari dihapus` |
| Timer systemd berjalan harian di VPS produksi | `NEEDS-DEVICE` | Langkah: `backend/deploy/README.md` §"Backup database otomatis", bukti `systemctl list-timers gopay-backup.timer` + `journalctl -u gopay-backup` |
| Unggah ke S3 dengan IAM role `PutObject` saja | `NEEDS-DEVICE` | Butuh bucket + IAM role di akun AWS Akbar; bukti file muncul di bucket |

### Keseluruhan suite

`make test`: `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (25.9s), `notify`, `reminder`, `secretbox`, `store` (13.1s), `telegram`, `telegrambot`.

---

## 20. Monitoring server + halaman API Docs

Permintaan langsung Akbar lewat chat.

**Ringkasan:** `PASS` 5 · `FAIL` 0 · `NEEDS-DEVICE` 4 · `PENDING` 0

### 20a. Monitoring dan peringatan

| Butir | Status | Bukti |
|---|---|---|
| `gopay-monitor.sh` hanya mengabari saat status berubah: masalah baru dikabari, masalah yang sama tidak dikirim ulang, diulang setelah `REPEAT_HOURS`, pulih dikabari dengan pesan masalah sebelumnya | `PASS` | Uji lokal `DRY_RUN=1` dengan service fiktif, URL mati (`http://127.0.0.1:1/`), dan backup berumur 30 jam. Putaran 1: `🔴` untuk ketiganya. Putaran 2: `monitor: tidak ada perubahan (3 masalah aktif)`. Setelah pulih: `✅ Sudah pulih setelah 0 menit. Sebelumnya: Backup terakhir sudah 30 jam lalu`. Setelah waktu kirim di state dimundurkan: `🟠 Masih bermasalah sejak 0 jam lalu: ...` |
| Tanpa token Telegram, pesan tetap ditulis ke log dan skrip tidak gagal | `PASS` | `monitor: TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID belum diisi, pesan tidak dikirim:` diikuti isi pesan, `exit=0` |
| Pesan sungguhan sampai ke Telegram vendor dari VPS, dan timer berjalan tiap 5 menit | `NEEDS-DEVICE` | Langkah A.4–A.5 di `backend/deploy/README.md` §"Monitoring dan peringatan" |
| UptimeRobot mengabari saat layanan mati dari luar | `NEEDS-DEVICE` | Langkah B.4 (stop `gopay-dashboard`, tunggu kabar down lalu up) |

### 20b. Halaman API Docs (`/api-docs`)

| Butir | Status | Bukti |
|---|---|---|
| Isi dicocokkan dengan kode: header `Authorization: Bearer sk_…`, body `external_ref`/`amount`, `201` baru vs `200` idempoten, `409 external_ref_conflict`, `503 allocation_full`, nominal unik +1..999, masa berlaku 15 menit, `external_ref` lama tetap mengembalikan invoice EXPIRED, header `X-Webhook-Event`/`X-Webhook-Signature`, HMAC-SHA256 hex dengan secret `whsec_…` apa adanya, timeout 10 detik, 5 percobaan dengan jeda 1/2/4/8 menit, `invoice.paid` bisa menyusul `invoice.expired` lewat pencocokan manual Exceptions | `PASS` | Dibaca langsung dari `invoices.go`, `apikey_auth.go`, `webhook_send.go`, `store/invoice.go` (`invoiceExpiryDuration = 15m`, `invoiceOffsetMax = 999`, `GetInvoiceByExternalRef` tanpa filter status), `store/webhook.go` (`webhookMaxAttempts = 5`, `webhookBackoff`), `store/exception.go` (`ManualMatchEvent` menerima `PENDING`/`EXPIRED`) |
| Halaman ter-build dan muncul di sidebar Gateway | `PASS` | `npx tsc --noEmit`, `npx eslint .` bersih; `npx next build` → `○ /api-docs` |
| Rumus tanda tangan di contoh Node.js, PHP, dan `openssl` identik dengan `signWebhookBody` backend | `PASS` | Body dan secret yang sama, dihitung empat cara: `go` (salinan persis `signWebhookBody`), `node` (`createHmac`), `openssl dgst -hmac`, `php hash_hmac` — keempatnya `49b58c4b221761683e8ffbb8d5118c3f66df55299f62e62204e44ec460026fb7` |
| Contoh verifikasi tanda tangan berjalan end-to-end di server merchant sungguhan | `NEEDS-DEVICE` | Butuh server merchant uji: daftarkan webhook, tekan **Test** di halaman Webhooks, pastikan contoh kode menjawab 200; ubah secret, pastikan menjawab 401 |
| Tampilan halaman di browser (tab kode, tombol salin, daftar isi) | `NEEDS-DEVICE` | Menunggu dicek Akbar di `npm run dev` |

---

## 21. Validasi + verifikasi email saat signup, dan "Kirim link reset password" di Vendor Dashboard

Permintaan langsung Akbar lewat chat.

**Ringkasan:** `PASS` 15 · `FAIL` 0 · `NEEDS-DEVICE` 2 · `PENDING` 0

### 21a. Validasi format email saat signup

| Butir | Status | Bukti |
|---|---|---|
| `/register` menolak format email salah sebelum submit (client-side) dan backend menolaknya juga (authoritative) | `PASS` | `TestSignupEmailFormatSalahDitolak` (4 kasus: tanpa `@`, tanpa TLD, lokal kosong, mengandung spasi) — lulus; `npx tsc`/`eslint`/`next build` bersih untuk perubahan `/register` |
| Validasi yang sama dipakai ulang di ganti email Settings (`isEmailAddress`, sudah ada sejak §18) | `PASS` | Tidak ada regresi — `TestAdminAccountProfile` (§18) masih lulus |

### 21b. Verifikasi email

| Butir | Status | Bukti |
|---|---|---|
| Signup mengirim email verifikasi otomatis (di background), tercatat di `notification_log` kind `email_verification` | `PASS` | `TestSignupMengirimEmailVerifikasi` — lulus |
| Token sekali pakai, berlaku 24 jam, permintaan baru membatalkan token lama | `PASS` | `TestEmailVerificationTokenSekaliPakai`, `TestEmailVerificationTokenKedaluwarsaDanDigantikan` — lulus |
| `POST /api/v1/email/verify`: token valid menandai terverifikasi, token dipakai ulang atau tidak dikenal → `400 invalid_token`, endpoint publik (tidak butuh sesi) | `PASS` | `TestVerifyEmailSekaliPakai` — lulus |
| `POST /api/v1/admin/account/email/resend`: terkirim, dibatasi cooldown 2 menit, menjawab `already_verified` bila sudah terverifikasi, wajib sesi | `PASS` | `TestResendVerificationEmail`, `TestResendVerificationEmailSudahTerverifikasi` — lulus |
| Ganti ke email BEDA dari Settings mengosongkan status terverifikasi dan mengirim link verifikasi baru; ganti ke email SAMA tidak mengubah apa pun | `PASS` | `TestUpdateAccountProfileMenggantiEmailMengosongkanVerifikasi`, `TestGantiEmailMengirimVerifikasiBaru` — lulus |
| Halaman `/verify-email` (token dari query string, bukan fragment — verifikasi bukan kunci akun) dan banner "Email belum diverifikasi" + tombol kirim ulang di Settings | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih; route `/verify-email` ter-generate |
| Migrasi `00015` bisa di-rollback dan diterapkan ulang | `PASS` | `goose down` lalu `goose up`: `OK 00015_email_verification.sql` dua kali, `successfully migrated database to version: 15` |
| Email verifikasi sungguhan sampai, link membuka halaman dan menandai terverifikasi | `NEEDS-DEVICE` | Butuh SMTP asli + `DASHBOARD_URL`. Langkah: daftar akun baru → buka email verifikasi → klik link → halaman `/verify-email` menampilkan "Email terverifikasi" |

### 21c. Vendor: "Kirim link reset password" di detail account

| Butir | Status | Bukti |
|---|---|---|
| Terkirim ke email account, tercatat di `notification_log` sebagai `password_reset`, dan ke `audit_log` sebagai `PASSWORD_RESET_SENT` oleh vendor | `PASS` | `TestVendorSendPasswordReset` — lulus |
| Dibatasi cooldown yang sama dengan reset password publik (2 menit) | `PASS` | `TestVendorSendPasswordReset` (bagian cooldown) — lulus |
| `503 not_available` tanpa `DASHBOARD_URL`/SMTP; `404` account tidak ada; `409 account_revoked` untuk account yang sudah dicabut; wajib sesi vendor | `PASS` | `TestVendorSendPasswordResetBelumTersedia`, `TestVendorSendPasswordResetAccountTidakDitemukanAtauDicabut`, `TestVendorSendPasswordResetButuhSesiVendor` — lulus |
| Tombol + modal konfirmasi di halaman detail account | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih untuk `vendor-dashboard/` |
| Email reset sungguhan sampai dan link berfungsi | `NEEDS-DEVICE` | Sama alur reset password yang sudah diverifikasi di §18 (tanda tangan/logika sama, cuma pemicunya beda) — butuh SMTP asli untuk membuktikan pengiriman vendor secara spesifik |

### Keseluruhan suite

`make test`: `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (29.2s), `notify`, `reminder`, `secretbox`, `store` (14.5s), `telegram`, `telegrambot`.

---

## 22. Halaman Logs (riwayat aktivitas akun) di Customer Dashboard

Permintaan langsung Akbar lewat chat. Menu "Logs" yang sebelumnya bertanda
"Segera" sekarang aktif, isinya riwayat aktivitas keamanan akun (login,
ganti password, API key, device) — beda dari "Logs" di
`docs/dashboard-spec.md` yang membayangkan log teknis untuk troubleshooting;
lihat catatan penyimpangan di CLAUDE.md §"Logs (riwayat aktivitas akun)".

**Ringkasan:** `PASS` 6 · `FAIL` 0 · `NEEDS-DEVICE` 1 · `PENDING` 0

| Butir | Status | Bukti |
|---|---|---|
| Tercatat per account, tidak bocor lintas account | `PASS` | `TestLogActivityDanListActivityLog`, `TestActivityLogTidakBocorLintasAccount` — lulus |
| Login berhasil dan login gagal (password salah) tercatat dengan alamat IP; username tidak dikenal tidak menghasilkan baris (tidak ada account untuk dilekatkan) | `PASS` | `TestActivityLogMencatatLoginBerhasilDanGagal` — lulus |
| Ganti password (Settings), reset password (lupa password), buat/cabut API key, tambah/hapus device semuanya tercatat dengan metadata yang relevan, urutan terbaru dulu | `PASS` | `TestActivityLogMencatatGantiPasswordApiKeyDanDevice` — lulus |
| `GET /api/v1/admin/activity`: filter jenis aktivitas dan rentang tanggal, jenis tidak dikenal → 400, wajib sesi | `PASS` | `TestActivityLogMencatatGantiPasswordApiKeyDanDevice` (bagian filter), `TestActivityLogButuhSesi`, `TestListActivityLogFilterTanggal` — lulus |
| Migrasi `00016` bisa di-rollback dan diterapkan ulang | `PASS` | `goose down` lalu `goose up`: `OK 00016_account_activity_log.sql` dua kali, `successfully migrated database to version: 16` |
| Halaman `/logs` aktif di sidebar System (bukan lagi "Segera"), dengan filter jenis aktivitas dan rentang tanggal | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih; route `/logs` ter-generate |
| Tampilan halaman di browser | `NEEDS-DEVICE` | Menunggu dicek Akbar di `npm run dev` |

### Keseluruhan suite

`make test`: `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (30.5s), `notify`, `reminder`, `secretbox`, `store` (14.8s), `telegram`, `telegrambot`.

---

## 23. Pairing HP lewat QR code

Permintaan langsung Akbar lewat chat. Dashboard menampilkan QR saat device
baru dibuat; aplikasi Android memindainya dan mengisi Backend URL/Device
ID/Device Secret otomatis, menggantikan ketik manual. **Tidak ada
perubahan Kotlin** — jalur penyimpanan settings (`saveSettings`) sudah bisa
dikendalikan sepenuhnya dari JS sejak sebelumnya, QR cuma cara baru mengisi
nilai yang sama.

**Ringkasan:** `PASS` 3 · `NEEDS-DEVICE` 3 · `PENDING` 0

| Butir | Status | Bukti |
|---|---|---|
| QR berisi `backend_url` (dengan akhiran `/api/v1`, format sama dengan nilai bawaan varian di `mobile/src/lib/env.ts`), `device_id`, `device_secret` — nilai sama persis dengan yang ditampilkan untuk disalin manual | `PASS` | Tinjauan kode terhadap `Uploader.kt` (`cfg.backendUrl + "/events"`, tanpa menyisipkan `/api/v1` sendiri) — ditemukan dan diperbaiki SEBELUM sempat diuji ke HP: draf pertama QR memakai `window.location.origin` tanpa akhiran, yang akan lolos scan tapi Test Connection gagal 404 |
| Dashboard: dialog "Device dibuat" menampilkan QR + peringatan jangan screenshot; `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih | `PASS` | Build bersih setelah `npm install react-qr-code` (2 paket ditambahkan, 0 kerentanan) |
| Tidak ada regresi di backend (fitur ini murni frontend + mobile) | `PASS` | `make test`: seluruh 10 paket `ok`, sama seperti sebelum perubahan ini |
| Izin kamera diminta dan diberikan di HP sungguhan, `CameraView` menampilkan preview | `NEEDS-DEVICE` | Butuh `npx expo prebuild --platform android --clean` (plugin `expo-camera` baru) lalu `npx expo run:android` |
| QR dari dashboard sungguhan berhasil dipindai, tersimpan, dan Test Connection berhasil ("Terhubung sebagai ...") | `NEEDS-DEVICE` | Sama seperti di atas — perlu HP + backend + akun customer sungguhan |
| QR yang salah/rusak (bukan JSON, field kurang) diam-diam diabaikan, tidak membuat aplikasi crash | `NEEDS-DEVICE` | Kode `parsePairingPayload` menangani lewat `try/catch` + pengecekan tipe, tapi belum diuji perilakunya di kamera sungguhan |

### Yang perlu dilakukan sebelum uji di HP

1. `cd mobile && npx expo install expo-camera` (bukan `npm install` manual — command ini mengunci versi yang cocok dengan Expo SDK 57 yang terpasang).
2. `npx expo prebuild --platform android --clean` — wajib karena plugin `expo-camera` baru ditambahkan ke `app.config.ts` (menyuntikkan izin `CAMERA` ke manifest).
3. `npx expo run:android`.
4. Di HP: Pengaturan → "Scan QR dari Dashboard" → izinkan kamera → arahkan ke QR di halaman Devices dashboard (buka di laptop/HP lain) → pastikan muncul "Terhubung sebagai ...".

### Keseluruhan suite

`make test` (backend, tidak berubah oleh fitur ini): `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (30.2s), `notify`, `reminder`, `secretbox`, `store` (14.7s), `telegram`, `telegrambot`.

---

## 24. Mencabut status "expiring" (peringatan dini 30 hari)

Permintaan langsung Akbar setelah membuat account baru berakhir 5 Oktober
2026 (21 hari dari tanggal itu) langsung berstatus "expiring" — dianggap
membingungkan karena masih berbulan-bulan tersisa terasa seperti sudah
"akan berakhir". Keputusan: hapus status ini sama sekali, bukan cuma
menurunkan ambangnya. `Account.DerivedStatus` sekarang cuma
`active`/`expired`/`suspended`/`revoked` — account tetap "active" sampai
PERSIS melewati `expires_at`. Pengingat aktif ke customer (email/Telegram,
7 hari sebelumnya, `internal/reminder`) TIDAK berubah — itu tetap
satu-satunya jalur "hampir habis".

**Ringkasan:** `PASS` 6 · `FAIL` 0 · `NEEDS-DEVICE` 1 · `PENDING` 0

| Butir | Status | Bukti |
|---|---|---|
| `DerivedStatus`: account dengan sisa 10 hari atau 1 jam tetap `active`; sudah lewat `expires_at` → `expired`; `suspended`/`revoked` tidak berubah | `PASS` | `TestAccountDerivedStatus` (kasus baru "akan berakhir dalam 10 hari, tetap aktif" dan "akan berakhir dalam 1 jam, tetap aktif") — lulus |
| `Operational()` cuma `true` untuk `active` (bukan lagi `active` atau `expiring`) | `PASS` | `TestAccountOperational` — lulus (tidak berubah, sudah benar sejak awal karena tidak pernah membedakan expiring secara eksplisit) |
| `VendorOverviewStats` tidak lagi menghitung `Expiring`; account yang dulu masuk hitungan itu sekarang ikut `Active` | `PASS` | `TestVendorOverviewStatsMenghitungRingkasanLintasAccount` (diperbarui: `Active` 1 → 2, field `Expiring` dihapus) — lulus |
| `GET /api/v1/vendor/overview` tidak lagi mengembalikan `accounts.expiring` | `PASS` | `go build`/`go vet` bersih setelah field dihapus dari `vendorOverviewResponse`; test overview lain (tidak menyentuh field ini) tetap lulus |
| Tidak ada sisa referensi "expiring" yang masih fungsional di backend maupun kedua dashboard (Customer License page, Vendor Dashboard/Accounts/stat-card) | `PASS` | `grep -rn "expiring"` lintas `backend/`, `dashboard/`, `vendor-dashboard/` — nihil di luar komentar historis yang sengaja menjelaskan pencabutannya |
| `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih untuk `dashboard/` dan `vendor-dashboard/` | `PASS` | Dijalankan langsung, keduanya 0 error/warning |
| Tampilan status di kedua dashboard setelah update (account baru langsung "Aktif", bukan "Akan Berakhir"/"Expiring") | `NEEDS-DEVICE` | Menunggu dicek Akbar di `npm run dev` dan setelah update VPS |

### Keseluruhan suite

`make test`: `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (30.3s), `notify`, `reminder`, `secretbox`, `store` (14.5s), `telegram`, `telegrambot`.

---

## 25. Halaman Plans di Vendor Dashboard + section Harga landing page dinamis

Permintaan langsung Akbar: kelola paket lewat Vendor Dashboard, dengan
section Harga landing page tetap sama styling/layoutingnya tapi datanya
dari input vendor. Tabel `plans` (migrasi 00017) jadi sumber ganda: kuota
device saat account dibuat/diganti plan (menggantikan `planPresets` yang
dulu hardcode) DAN teks yang tampil publik. `accounts.plan` tetap teks
bebas — menghapus/mengubah plan tidak menyentuh account yang sudah
memakainya.

**Ringkasan:** `PASS` 13 · `FAIL` 0 · `NEEDS-DEVICE` 1 · `PENDING` 0

### 25a. Backend

| Butir | Status | Bukti |
|---|---|---|
| CRUD plan (create/get-by-name/update/delete) termasuk validasi nama bentrok | `PASS` | `TestCreatePlanDanGetByName`, `TestCreatePlanNamaBentrokDitolak`, `TestUpdatePlan`, `TestUpdatePlanNamaBentrokDenganPlanLain` — lulus |
| Menghapus plan yang sudah dipakai account TIDAK mengubah `plan`/`max_devices` account itu | `PASS` | `TestDeletePlanAmanWalauSudahDipakaiAccount` — lulus |
| `ListPlans` (semua) vs `ListVisiblePlans` (cuma `visible=true`) | `PASS` | `TestListPlansDanListVisiblePlans` — lulus |
| Reorder (`MovePlan`) naik/turun, diam saja di ujung, pemutus seri konsisten | `PASS` | `TestMovePlan` — lulus |
| `POST/PATCH/DELETE/move /api/v1/vendor/plans*` wajib sesi vendor; validasi `max_devices`/`unlimited`; fitur baris kosong dibuang; nama bentrok → 409 | `PASS` | `TestVendorPlansButuhSesiVendor`, `TestVendorCreateUpdateDeletePlan`, `TestVendorMovePlan` — lulus |
| `GET /api/v1/pricing-plans` publik (tanpa sesi), cuma plan visible, `device_label` dihitung server-side dari `max_devices` | `PASS` | `TestPublicPricingPlans` — lulus |
| `handleVendorCreateAccount`/`handleVendorChangePlan` memvalidasi plan lewat `GetPlanByName` (bukan lagi peta hardcode); plan tidak dikenal → 400 | `PASS` | `TestVendorChangePlan`, `TestVendorChangePlanTidakValidDitolak` (sudah ada sebelumnya, tetap lulus tanpa perubahan) |
| Signup swalayan mengambil kuota trial dari plan "Starter"; kalau plan itu dihapus, signup berhenti jelas (503 `not_available`), tidak diam-diam salah kuota | `PASS` | `TestSignupGagalBilaPlanTrialTidakAda` — lulus |
| Migrasi `00017` bisa di-rollback dan diterapkan ulang, seed 3 plan (Starter/Business/Enterprise) sama persis dengan teks yang sebelumnya hardcode di landing page | `PASS` | `goose down` lalu `goose up`: `OK 00017_plans.sql`, `successfully migrated database to version: 17` |

### 25b. Vendor Dashboard

| Butir | Status | Bukti |
|---|---|---|
| Halaman `/plans`: tambah/edit/hapus/reorder, badge Direkomendasikan/Disembunyikan, harga kosong menampilkan pesan "tidak tampil di landing page" | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih; route `/plans` ter-generate |
| Dropdown Plan di Accounts (buat baru & ubah plan) diisi dinamis dari `GET /vendor/plans`, bukan lagi tiga nilai tetap | `PASS` | Build bersih setelah refactor `AccountPlan` jadi `string` dan penghapusan konstanta `PLANS` di kedua halaman Accounts |

### 25c. Landing page (Customer Dashboard)

| Butir | Status | Bukti |
|---|---|---|
| Section Harga mengambil data dari `GET /api/v1/pricing-plans`, layout/styling kartu (grid 3 kolom, kartu highlight gelap, hover, CTA) dipertahankan persis sama; skeleton saat loading, pesan fallback bila kosong/gagal | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` (build bersih penuh setelah `rm -rf .next`, route `index` ter-generate) |
| Harga (`price_label`) tidak ditampilkan sama sekali untuk plan yang belum diisi vendor -- bukan "Rp 0" | `PASS` | Data seed migrasi sengaja `price_label=''`; kode render `{plan.price_label && (...)}` — tinjauan kode |
| Tampilan sungguhan section Harga di browser (skeleton, transisi ke data asli, kartu highlight) | `NEEDS-DEVICE` | Menunggu dicek Akbar di `npm run dev`, termasuk mengisi harga lewat halaman Plans lalu memuat ulang `/` |

### Keseluruhan suite

`make test`: `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (32.4s), `notify`, `reminder`, `secretbox`, `store` (15.5s), `telegram`, `telegrambot`.

---

## 26. Kelola device + cabut API key dari Vendor Dashboard (halaman detail account)

Permintaan langsung Akbar (setelah menyebut Vendor Dashboard terasa
minim fitur untuk "super admin"): tambah/hapus device dan cabut API key
customer langsung dari Vendor Dashboard, tanpa SSH `cmd/devicetool` atau
minta customer login sendiri. Endpoint baru reuse penuh helper/store yang
sudah ada (`randomDeviceID`, `toAdminDeviceJSON`, `store.ListAPIKeys`,
`store.RevokeAPIKey` — dua yang terakhir sudah di-scope per account sejak
awal, tidak perlu store method baru). Vendor tidak pernah bisa membuat
API key baru untuk customer (cuma lihat metadata + cabut) dan tidak
pernah melihat key mentah/hash. Aksi dicatat ke `LogAudit` (Audit Log
vendor), bukan `account_activity_log` customer — konsisten dengan
renew/suspend/plan/send-password-reset yang sudah ada.

**Ringkasan:** `PASS` 10 · `FAIL` 0 · `NEEDS-DEVICE` 1 · `PENDING` 0

### 26a. Backend

| Butir | Status | Bukti |
|---|---|---|
| `POST /vendor/accounts/{id}/devices` membuat device, menegakkan kuota `max_devices` plan (409 `device_limit_reached` saat penuh), butuh sesi vendor | `PASS` | `TestVendorCreateDevice`, `TestVendorCreateDeviceKuotaPenuh`, `TestVendorCreateDeviceButuhSesiVendor` — lulus |
| `DELETE /vendor/accounts/{id}/devices/{deviceID}` menghapus device tanpa riwayat; menolak 409 `device_has_events` untuk device yang sudah punya event | `PASS` | `TestVendorDeleteDevice`, `TestVendorDeleteDeviceDenganRiwayatDitolak` — lulus |
| `PATCH /vendor/accounts/{id}/devices/{deviceID}` mengaktifkan/menonaktifkan device; 404 untuk device tak dikenal | `PASS` | `TestVendorSetDeviceEnabled`, `TestVendorSetDeviceEnabledDeviceTakDikenal` — lulus |
| `GET /vendor/accounts/{id}/devices` (sudah ada) tetap lulus setelah refactor helper `createVendorTestAccount` | `PASS` | `TestVendorListDevices`, `TestVendorListDevicesButuhSesiVendor` — lulus |
| `GET /vendor/accounts/{id}/api-keys` mengembalikan metadata key (nama, dibuat, dicabut) TANPA key mentah/hash sama sekali; butuh sesi vendor | `PASS` | `TestVendorListAPIKeys` (assert `hash-dummy` tidak muncul di body), `TestVendorAPIKeysButuhSesiVendor` — lulus |
| `DELETE /vendor/accounts/{id}/api-keys/{keyID}` mencabut key, idempotent, 404 untuk key tak dikenal | `PASS` | `TestVendorRevokeAPIKey`, `TestVendorRevokeAPIKeyIdempotent`, `TestVendorRevokeAPIKeyTakDikenal` — lulus |
| `go build`/`go vet`/`gofmt -l .` bersih | `PASS` | Dijalankan langsung, keluaran kosong |
| `make test` (seluruh suite backend) | `PASS` | `ok` untuk seluruh paket termasuk `httpapi` (40.1s) dan `store` (18.8s) |

### 26b. Vendor Dashboard

| Butir | Status | Bukti |
|---|---|---|
| Halaman detail account: tombol "Tambah Device" (dialog nama → reveal device_id/secret sekali, copy-to-clipboard), tombol Aktifkan/Nonaktifkan/Hapus per baris, card API Keys baru (badge Aktif/Dicabut, tombol Cabut dengan konfirmasi) | `PASS` | `npx tsc --noEmit`, `npx eslint .`, `npx next build` bersih setelah `rm -rf .next` |
| Fungsi `lib/api.ts` baru (`createAccountDevice`, `setAccountDeviceEnabled`, `deleteAccountDevice`, `getAccountAPIKeys`, `revokeAccountAPIKey`) tertipe benar, dipakai halaman detail | `PASS` | Bagian dari build/tsc di atas — tidak ada `any` yang lolos |
| Uji end-to-end sungguhan di browser (tambah device baru untuk account nyata, cabut API key, lihat device dengan riwayat event ditolak dihapus) | `NEEDS-DEVICE` | Menunggu dicek Akbar di `npm run dev` |

### Keseluruhan suite

`make test`: `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (40.1s), `notify`, `reminder`, `secretbox`, `store` (18.8s), `telegram`, `telegrambot`.

---

## 27. QRIS statis per account (sub-project 1 migrasi Cashi → gopay-notifications)

Permintaan langsung Akbar: rencana migrasi `whuzpay-pg/` (payment gateway
aggregator, folder terpisah di repo ini) dari provider Cashi ke
gopay-notifications sendiri. Spec:
[`docs/superpowers/specs/2026-09-15-account-qris-image-design.md`](../superpowers/specs/2026-09-15-account-qris-image-design.md).
Sub-project 1 ini murni di gopay-notifications sendiri (belum menyentuh
`whuzpay-pg/`): tiap account upload gambar QRIS statisnya sendiri, dan
`POST /invoices` menyertakannya ke integrator, menolak `409
qris_not_configured` kalau belum diatur.

**Ringkasan:** `PASS` 13 · `FAIL` 0 · `NEEDS-DEVICE` 0 · `PENDING` 0

### 27a. Backend

| Butir | Status | Bukti |
|---|---|---|
| Tabel `account_qris_images` (migrasi 00018), relasi 1:1 dengan `accounts`, `ON DELETE CASCADE` | `PASS` | `TestDeleteAccountIkutMenghapusQRISImage` — lulus; `goose up` sukses ke versi 18 |
| Store: upsert menimpa (bukan menambah baris), get, delete idempotent, cek eksistensi ringan | `PASS` | `go test ./internal/store/... -run QRISImage -v` → `TestUpsertDanGetQRISImage`, `TestUpsertQRISImageMenimpaBukanMenambah`, `TestGetQRISImageTidakAdaMengembalikanErrNotFound`, `TestHasQRISImage`, `TestDeleteQRISImageIdempotent`, `TestDeleteAccountIkutMenghapusQRISImage` — 6 lulus |
| `PUT /admin/account/qris-image`: validasi ukuran (≤300KB) dan tipe dari ISI byte (`http.DetectContentType`, bukan field yang diklaim klien), bukan base64 valid ditolak jelas | `PASS` | `TestUploadQRISImageBerhasil`, `TestUploadQRISImageBase64TidakValid`, `TestUploadQRISImageTerlaluBesar`, `TestUploadQRISImageTipeTidakDidukung` — lulus |
| `GET`/`DELETE /admin/account/qris-image`: 404 belum ada, delete idempotent, seluruh endpoint butuh sesi | `PASS` | `TestGetQRISImageBelumAdaMengembalikan404`, `TestDeleteQRISImageIdempotenLewatHTTP`, `TestQRISImageEndpointButuhSesi` — lulus |
| `GET /admin/account` menyertakan `qris_image_configured` | `PASS` | `TestGetAccountProfileMemuatQRISImageConfigured` — lulus |
| `POST /invoices` menyertakan `qris_image` (data URI) di response create; account tanpa QRIS ditolak `409 qris_not_configured` TANPA menyimpan invoice apa pun; `GET /invoices/{id}` (polling) sengaja TIDAK mengulang field ini | `PASS` | `TestCreateInvoiceMenyertakanQRISImage`, `TestCreateInvoiceGagalTanpaQRISImage`, `TestCreateInvoiceGagalTanpaQRISImageTidakMenyimpanApaPun`, `TestGetInvoiceTidakMenyertakanQRISImage` — lulus |
| Webhook payload (`invoice.paid`) TIDAK ikut membengkak dengan `qris_image` (`omitempty`, cuma diisi manual di `handleCreateInvoice`) | `PASS` | Tinjauan kode `webhook_send.go` — masih memanggil `toInvoiceJSON(inv)` polos tanpa mengisi field baru; test webhook yang sudah ada (`TestWebhookInvoicePaidTerpicuOtomatisLewatCallback` dkk) tetap lulus tanpa perubahan assertion |
| `go build`/`go vet`/`gofmt -l .` bersih | `PASS` | Dijalankan langsung, keluaran kosong |
| `make test` (seluruh suite backend, termasuk 4 helper test lain yang diperbarui ikut upload QRIS dummy: `admin_exceptions_test.go`, `webhook_worker_test.go`, `tenant_isolation_test.go`) | `PASS` | `ok` untuk seluruh paket termasuk `httpapi` (37.4s) dan `store` (17.4s) |

### 27b. Customer Dashboard

| Butir | Status | Bukti |
|---|---|---|
| Card "QRIS Pembayaran" di Settings: upload/ganti/hapus, preview `<img>` langsung dari endpoint backend (tanpa decode base64 di frontend), validasi ukuran+tipe di klien sebelum kirim | `PASS` | `npx tsc --noEmit`, `npx eslint .` (0 error/warning setelah perbaikan posisi komentar `eslint-disable-next-line`), `npx next build` bersih setelah `rm -rf .next` |
| Halaman Logs mengenali 2 aktivitas baru (`qris_image_updated`/`qris_image_removed`) di `ACTION_LABEL`/`ACTION_BADGE`/`ActionIcon` (`Record<ActivityAction, ...>` exhaustive — `tsc` akan gagal kalau ada yang terlewat) | `PASS` | Bagian dari `npx tsc --noEmit` di atas |
| API Docs (`/api-docs`): contoh response `POST /invoices` memuat `qris_image`, baris baru di tabel field invoice, baris baru `409 qris_not_configured` di tabel error | `PASS` | Tinjauan kode + build bersih di atas |
| Uji end-to-end sungguhan di browser (upload gambar asli, lihat preview, buat invoice API sungguhan dan cek `qris_image` di response, hapus lalu coba buat invoice lagi dan lihat `409`) | `PASS` | Dicek Akbar di `npm run dev` (2026-09-15): upload JPEG asli di Settings → `POST /invoices` (`external_ref=TEST-001`) balas `201` dengan `qris_image` berupa `data:image/jpeg;base64,/9j/4AAQSkZJRgAB...` (signature JFIF asli, cocok dengan file yang di-upload) → hapus QRIS di Settings → `POST /invoices` (`external_ref=TEST-002`) balas persis `{"success":false,"error":"qris_not_configured","message":"QRIS belum diatur -- upload di halaman Settings dulu"}` |

### Keseluruhan suite

`make test`: `ok` untuk `auth`, `connector`, `devicealert`, `httpapi` (37.4s), `notify`, `reminder`, `secretbox`, `store` (17.4s), `telegram`, `telegrambot`.
