# QA Report

**Milestone terakhir diperiksa:** M1 selesai · M2 sebagian (aplikasi terpasang, Notification Access aktif)
**Tanggal:** 2026-09-10
**Ringkasan:** `PASS` 28 · `FAIL` 0 · `BLOCKED` 0 · `NEEDS-DEVICE` 3 · `PENDING` 39

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

Butir 7 menentukan bentuk sub-project 3: referensi transaksi membuat matching eksak dan menggugurkan seluruh rencana nominal unik.

---

## 3. Functional Requirements — [prd.md §11](../prd.md)

| # | Requirement | Milestone | Status | Bukti |
|---|---|---|---|---|
| FR-01 | Notification Access terdeteksi | M2 | `PASS` | Setelah izin diberikan di HP, layar menampilkan `Notification Access: aktif` dan `Listener terikat: ya` — dilaporkan Akbar 2026-09-10 |
| FR-02 | Notification Listener menerima event | M2 | `PENDING` | — |
| FR-03 | Hanya memproses event GoPay | M3 | `PENDING` | — |
| FR-04 | Parsing nominal dan `event_id` (unit) | M3 | `PASS` | `./gradlew :gopay-listener:testDebugUnitTest` → BUILD SUCCESSFUL; `build/test-results/testDebugUnitTest/*.xml` mencatat `tests=20 failures=0 errors=0 skipped=0` (AmountParserTest 8, EventIdBuilderTest 5, SignerTest 7) |
| FR-04 | Notifikasi jadi event terstruktur (pipeline) | M3 | `PENDING` | Butuh Task 8 |
| FR-05 | Kirim event via HTTPS | M4 | `PENDING` | Sisi penerima siap; pengirim belum ada |
| FR-06 | Autentikasi request — sisi backend | M1 | `PASS` | `TestVerifyRejectsModifiedBody`, `TestVerifyRejectsWrongSecret`, `TestAuthRejectsWrongSecret`, `TestAuthRejectsUnknownDevice`, `TestAuthRejectsDisabledDevice`, `TestAuthRejectsMissingHeaders` (3 subtest), e2e no. 3 |
| FR-06 | Autentikasi request — sisi Android | M4 | `PENDING` | — |
| FR-07 | Retry untuk error yang dapat dipulihkan | M4 | `PENDING` | — |
| FR-08 | Pencegahan duplikat — sisi backend | M1 | `PASS` | `TestInsertEventConcurrentSameIDInsertsOnce` (8 goroutine, `-count=20 -race`), `TestCallbackSecondTimeReturnsDuplicate`, e2e no. 2 dan 5 |
| FR-08 | Pencegahan duplikat — sisi Android | M3 | `PENDING` | — |
| FR-09 | Pencatatan status event | M3 | `PENDING` | — |
| FR-10 | Indikator listener aktif | M5 | `PENDING` | — |

---

## 4. Acceptance Criteria — [prd.md §16](../prd.md)

| Kriteria | Milestone | Status | Bukti |
|---|---|---|---|
| User dapat memberikan Notification Access | M2 | `PENDING` | — |
| Aplikasi mendeteksi notifikasi baru | M2 | `PENDING` | — |
| Membedakan GoPay dari aplikasi lain | M3 | `PENDING` | — |
| Notifikasi jadi event terstruktur | M3 | `PENDING` | — |
| Event terkirim via HTTPS | M4 | `PENDING` | — |
| Backend mengidentifikasi perangkat pengirim | M1 | `PASS` | `TestAuthAcceptsValidSignature`, `TestTouchDeviceSetsLastSeenAt`, e2e no. 6 (`last_seen_at` = `t`) |
| Event sama tidak diproses dua kali | M1 | `PASS` | `TestInsertEventConcurrentSameIDInsertsOnce`, e2e no. 5 (tepat 1 baris setelah 2 kiriman identik) |
| Event terkirim setelah koneksi normal kembali | M4 | `PENDING` | — |
| User melihat status listener | M5 | `PENDING` | — |
| User melihat status komunikasi backend | M5 | `PENDING` | — |
| User melihat event/history dasar | M5 | `PENDING` | — |
| Tidak memproses notifikasi selain GoPay | M3 | `PENDING` | — |

---

## 5. Definition of Done — [detail-project.md §42](../detail-project.md)

| Butir | Milestone | Status | Bukti |
|---|---|---|---|
| Aplikasi berjalan di Android | M2 | `PASS` | `adb shell pm list packages` → `package:id.manjo.gopaybridge.dev`; aplikasi terbuka di OPPO CPH2365 |
| TypeScript sebagai application language | M2 | `PASS` | `npx tsc --noEmit` bersih; `App.tsx` dan `modules/gopay-listener/index.ts` |
| Kotlin untuk Notification Listener | M2 | `PASS` | `adb shell dumpsys package id.manjo.gopaybridge.dev` menampilkan `expo.modules.gopaylistener.GoPayListenerService` dengan permission `BIND_NOTIFICATION_LISTENER_SERVICE` dan action `android.service.notification.NotificationListenerService` |
| Notification Access dapat diaktifkan | M2 | `PASS` | Tombol membuka `Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS`; izin diberikan dan status berubah jadi aktif |
| Listener berjalan di background | M6 | `PENDING` | — |
| Notifikasi GoPay terdeteksi | M2 | `PENDING` | — |
| Notifikasi aplikasi lain diabaikan | M3 | `PENDING` | — |
| Data notification dapat diekstrak | M3 | `PENDING` | — |
| Event ID dibuat | M3 | `PENDING` | — |
| Event tersimpan lokal di HP | M3 | `PENDING` | — |
| Event tersimpan di backend | M1 | `PASS` | `TestCallbackStoresRawPayloadVerbatim`, e2e no. 7 (`GET /events` mengembalikan event) |
| Event dapat dikirim ke backend | M4 | `PENDING` | — |
| HTTPS untuk production | M1 | `NEEDS-DEVICE` | `Caddyfile` ditulis, build silang OK; sertifikat belum diverifikasi — lihat §2 no. 1 |
| Authentication ditegakkan backend | M1 | `PASS` | Sama dengan FR-06 sisi backend |
| Authentication dikirim Android | M4 | `PENDING` | — |
| Retry mechanism berjalan | M4 | `PENDING` | — |
| Duplicate event ditangani backend | M1 | `PASS` | Sama dengan FR-08 sisi backend |
| Duplicate event ditangani Android | M3 | `PENDING` | — |
| Dashboard menampilkan listener status | M5 | `PENDING` | — |
| Dashboard menampilkan backend status | M5 | `PENDING` | — |
| History event tersedia | M5 | `PENDING` | — |
| Device ID tersedia | M1 | `PASS` | `devicetool -name "HP GoPay Utama"` → `Device ID : dev_18576e55cfa02f73` |
| Credential tidak hardcoded | M1 | `PASS` | `grep -rn 'DEVICE_SECRET_KEY' --include='*.go'` hanya menemukan pembacaan `os.Getenv` di `internal/config/config.go:33` dan teks bantuan flag |
| Menangani network failure | M4 | `PENDING` | — |
| Dapat diuji saat UI tidak terbuka | M6 | `PENDING` | — |
| Diuji pada physical Android device | M6 | `PENDING` | — |

---

## 6. Tabel kode → tindakan — [api-contract.md §4.6](../api-contract.md)

Baris `429`, `5xx`, dan `timeout` menggambarkan perilaku **HP**, bukan backend, sehingga tetap `PENDING` sampai M4.

| Kondisi | Tindakan yang diharapkan | Status | Bukti |
|---|---|---|---|
| `200 accepted` | → `SENT` | `PASS` | `TestCallbackFirstTimeReturnsAccepted`, e2e no. 1 |
| `200 duplicate` | → `SENT`, bukan error | `PASS` | `TestCallbackSecondTimeReturnsDuplicate`, e2e no. 2 |
| `400 invalid_payload` | → `FAILED`, tanpa retry | `PASS` | `TestCallbackRejectsMalformedJSON`, `TestCallbackRejectsInvalidFields` (7 subtest) |
| `401 invalid_signature` | → `FAILED`, peringatan kredensial | `PASS` | `TestAuthRejectsWrongSecret`, `TestAuthRejectsUnknownDevice`, e2e no. 3 |
| `401 clock_skew` | → `FAILED`, pesan jam meleset + jam server | `PASS` | `TestAuthRejectsClockSkewWithServerTime`, e2e no. 4 (`server_time` disertakan) |
| `403 device_disabled` | → `FAILED`, tanpa retry | `PASS` | `TestAuthRejectsDisabledDevice` |
| `429` | tetap `PENDING`, hormati `Retry-After` | `PENDING` | Perilaku sisi HP, M4 |
| `5xx` | tetap `PENDING`, retry | `PENDING` | Perilaku sisi HP, M4 |
| timeout / jaringan mati | tetap `PENDING`, retry | `PENDING` | Perilaku sisi HP, M4 |

---

## 7. Pemeriksaan keamanan & prasyarat lingkungan — [qa-rules.md §9](qa-rules.md)

| Pemeriksaan | Status | Bukti |
|---|---|---|
| Tidak ada secret ter-hardcode (backend) | `PASS` | `grep -rn 'DEVICE_SECRET_KEY' --include='*.go'` → hanya `os.Getenv` dan teks flag |
| Tidak ada secret ter-hardcode (Android) | `PENDING` | — |
| Tidak ada secret/tanda tangan di log (backend) | `PASS` | `grep -rn 'slog\.' internal/ cmd/ \| grep -iE 'secret\|signature'` → kosong; e2e no. 8 → 0 kemunculan secret di log server |
| Tidak ada secret/tanda tangan di log (Android) | `PENDING` | — |
| `hmac.Equal` dipakai, bukan `==` | `PASS` | `internal/auth/hmac.go:38` → `return hmac.Equal([]byte(want), []byte(gotSignature))` |
| Body diverifikasi mentah sebelum decode | `PASS` | `auth_middleware.go:71` `io.ReadAll` → `:90` `auth.Verify(...)`; `json.Unmarshal` baru di `callback.go:50`, setelah middleware |
| Toleransi timestamp ditegakkan di kedua batas | `PASS` | `TestCheckSkewBoundaries` — 7 subtest, termasuk ±299/±300/±301 detik |
| Idempotency memakai constraint database | `PASS` | `migrations/00001_init.sql:13` `event_id TEXT NOT NULL UNIQUE`; `event.go:35` `ON CONFLICT (event_id) DO NOTHING`; `event.go:42` `RowsAffected() == 1` |
| Build production menolak HTTP polos | `NEEDS-DEVICE` | Perlu Caddy di VPS — lihat §2 no. 4 |
| Aturan arah berupa allowlist, bukan blocklist | `PENDING` | Sub-project 3; di luar cakupan M1 |
| Mode Discovery default mati, tidak mengirim keluar | `PENDING` | Sisi Android, M2 |
| Channel "Promotions and Marketing" GoPay masih aktif | `NEEDS-DEVICE` | Perlu HP — diperiksa di M2 |
| Setup ColorOS selesai | `PASS` | Keempat setelan dikerjakan Akbar 2026-09-10: aktivitas latar belakang diizinkan, mulai otomatis aktif, aplikasi dikunci di recent apps, optimasi siaga tidur mati. Ketahanannya baru diuji di M6 |

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

## 10. Temuan & tindak lanjut

**Satu cacat ditemukan dan diperbaiki selama M1.** Target `test` di plan awal tidak memakai `-p 1`, sehingga `make test` gagal secara acak begitu paket kedua yang menyentuh database ditambahkan. Gejalanya menyesatkan karena setiap paket lulus bila dijalankan sendirian. Diperbaiki di `Makefile` dan di plan, dengan komentar yang menjelaskan sebabnya agar tidak dihapus orang lain di kemudian hari.

**Satu jebakan operasional yang perlu diingat.** Saat verifikasi manual, proses server dari langkah sebelumnya tidak mati dan tetap memegang port dengan binary lama — `GET /events` membalas 404 padahal rutenya sudah ada, dan `go build` ke path binary yang sedang berjalan gagal dengan *text file busy*. Sejak itu verifikasi manual memakai path binary yang unik.

**Yang menghalangi M1 dinyatakan tuntas:** akses VPS. Empat butir di §2 dan pembuatan device sungguhan tidak dapat dikerjakan dari mesin development.

**Perubahan scope 2026-09-11 membatalkan sebagian temuan lapangan.** Package identifier dan bentuk teks yang dikonfirmasi sebelum M1 berasal dari akun GoPay **pribadi**. Sumber pembayaran kini berpindah sepenuhnya ke **GoPay Merchant**, sehingga Open Question #1, #2, dan #3 kembali terbuka.

Biayanya nol baris kode. Package adalah daftar yang dapat diedit di Settings, dan aturan arah adalah konfigurasi backend — keduanya sengaja dirancang demikian di §2.3 dan §2.4 spec justru untuk kemungkinan seperti ini. Yang berubah hanya data.

Yang tetap berlaku dari pengamatan kemarin adalah bentuknya, bukan nilainya: notification id konstan, `Notification.when` bertahan lintas repost, format nominal `Rp1` tanpa pemisah, dan notifikasi transaksi dapat datang lewat channel promosi. Keempatnya mendasari formula `event_id` dan tidak tersentuh.

**Risiko yang justru hilang:** akun merchant hampir hanya menerima, sehingga notifikasi pembayaran **keluar** dengan nominal sama — bahaya utama yang diuraikan di §2.3 spec — praktis tidak ada lagi.

**Langkah berikutnya:** Task 6 (Room) dan Task 7 (konfigurasi terenkripsi). Keduanya tidak bergantung pada VPS maupun pada sampel notifikasi merchant, jadi dikerjakan sementara §2 butir 5–7 dikumpulkan.
