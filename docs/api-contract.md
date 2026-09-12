# API Contract — GoPay Notification Bridge

**Versi:** v1
**Status:** Disetujui
**Berlaku untuk:** Sub-project 1 (Event Ingestion) + Sub-project 2 (Android Bridge)

Dokumen ini adalah **sumber kebenaran bersama** antara backend Go dan aplikasi Android. Perubahan pada dokumen ini dan penyesuaian kedua sisi harus berada dalam satu commit.

---

## 1. Base URL

| Environment | URL |
|---|---|
| Production | `https://<domain>/api/v1` |
| Development | `http://<ip-lan>:8080/api/v1` |

Production **wajib** HTTPS. Development boleh HTTP polos, tetapi Android 9+ memblokirnya secara default sehingga build development memuat network security config yang mengizinkan hanya alamat development tertentu. Build production tetap HTTPS-only.

Kredensial production tidak boleh dipakai di environment development.

---

## 2. Daftar Endpoint

| Method | Path | Auth | Kegunaan |
|---|---|---|---|
| `POST` | `/events` | HMAC | Kirim event notifikasi (semua sumber) |
| `GET` | `/health` | — | Status backend + jam server |
| `GET` | `/device/me` | HMAC | Tombol *Test Connection*, sekaligus status device |
| `POST` | `/devices/heartbeat` | HMAC | Laporan kondisi berkala dari perangkat |
| `GET` | `/events` | Basic auth (Caddy) | Melihat event masuk saat verifikasi |
| `GET` | `/sources` | — | Daftar sumber pembayaran yang dikenal build ini |

---

## 3. Autentikasi HMAC

### 3.1 Header wajib

```http
X-Device-Id: dev_01ABC
X-Timestamp: 1789036200
X-Signature: 9f2a...
X-App-Version: 1.4.2
Content-Type: application/json
```

`X-Timestamp` adalah Unix epoch **detik**.

`X-App-Version` bersifat informatif dan **tidak** ikut ditandatangani — backend
harus mengetahui versinya untuk memverifikasi, sehingga ia mustahil menjadi
bagian tanda tangan. Aplikasi lama tidak mengirimnya, dan itu bukan alasan
menolak request. Backend mencatat nilai terakhir per device dan memaparkannya
di `GET /device/me`; nilai kosong tidak menimpa versi yang sudah diketahui.

### 3.2 Pembentukan tanda tangan

```
signing_string = device_id + "\n" + timestamp + "\n" + sha256_hex(raw_body)
signature      = hex( hmac_sha256(device_secret, signing_string) )
```

Untuk request tanpa body (`GET`), `raw_body` adalah string kosong; `sha256_hex("")` tetap dihitung dan tetap dimasukkan.

### 3.3 Aturan implementasi yang tidak boleh dilanggar

**Yang ditandatangani adalah byte mentah body, bukan hasil decode JSON.**

Di sisi Go, body harus dibaca mentah dan diverifikasi lebih dulu, baru di-decode. Bila urutannya terbalik — decode dulu lalu re-encode untuk verifikasi — tanda tangan akan gagal secara acak karena urutan field dan spasi berubah. Ini kesalahan klasik yang menghabiskan berjam-jam.

**Perbandingan tanda tangan wajib memakai `hmac.Equal`, bukan `==`.**

**Toleransi timestamp ±300 detik.** Di luar itu ditolak sebagai `clock_skew`.

### 3.4 Penyimpanan secret

Verifikasi HMAC menuntut server memegang secret yang sebenarnya, bukan hash-nya, sehingga tidak bisa disimpan seperti password. Secret dienkripsi di kolom database dengan kunci dari environment variable server.

Di sisi Android, secret disimpan dengan `expo-secure-store` dan **tidak pernah** ditulis ke log.

---

## 4. `POST /events`

### 4.1 Request

```json
{
  "event_id": "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c",
  "device_id": "dev_01ABC",
  "source": "gopay",
  "notification": {
    "package_name": "com.gojek.gopaymerchant",
    "title": "Transfer masuk",
    "text": "Rp25.000 dari icaangg udah masuk ke GoPay kamu.",
    "big_text": "Rp25.000 dari icaangg udah masuk ke GoPay kamu.",
    "posted_at": 1789036200000
  },
  "amount_hint": 25000,
  "received_at": "2026-09-10T19:30:00+07:00"
}
```

| Field | Tipe | Wajib | Catatan |
|---|---|---|---|
| `event_id` | string | ya | Deterministik, lihat §4.2 |
| `device_id` | string | ya | Harus sama dengan header `X-Device-Id` |
| `source` | string | ya | Divalidasi terhadap registry connector di backend. Saat ini hanya `"gopay"`; sumber tak dikenal ditolak `invalid_payload` |
| `notification.package_name` | string | ya | |
| `notification.title` | string \| null | ya | Mentah, apa adanya |
| `notification.text` | string \| null | ya | Mentah, apa adanya |
| `notification.big_text` | string \| null | ya | Sering null |
| `notification.posted_at` | number | ya | Unix epoch **milidetik**. Berisi `Notification.when`; bila 0, berisi `sbn.postTime` |
| `amount_hint` | number \| null | ya | **Petunjuk display-only**, lihat §4.3 |
| `received_at` | string | ya | ISO 8601 dengan offset zona waktu |

Seluruh field notifikasi bersifat opsional isinya (boleh `null`) tetapi **wajib hadir** sebagai key.

### 4.2 Pembentukan `event_id`

```
event_id = "evt_" + sha256( packageName | title | text | when )[:32]
```

`when` adalah `Notification.when`; bila 0, dipakai `sbn.postTime`.

Deterministik, tanpa random. Android memanggil `onNotificationPosted` berkali-kali untuk notifikasi yang sama; hash membuat pengulangan itu menghasilkan id identik sehingga tersaring sendiri.

`notificationKey` dan `postTime` sengaja **tidak** dipakai: GoPay memakai id `-1` tanpa tag sehingga key-nya konstan dan tidak menyumbang apa pun, sementara `postTime` berubah tiap notifikasi di-posting ulang sehingga justru memecah satu transfer menjadi beberapa event.

### 4.3 Aturan arah transaksi (backend)

Backend memakai **allowlist judul**, bukan blocklist:

```
title dalam allowlist   →  boleh dicocokkan ke pembayaran
apa pun selain itu      →  disimpan, tidak pernah dianggap uang masuk
```

Allowlist awal berisi satu entri, dikonfirmasi dari perangkat target
2026-09-11 atas pembayaran QRIS sungguhan:

```
"Pembayaran QRIS statis diterima"
```

Entri `"Transfer masuk"` yang sempat tercatat berasal dari akun GoPay pribadi
yang tidak lagi dipakai, dan **tidak boleh** dimasukkan.

Judul untuk QRIS **dinamis** kemungkinan berbeda dan belum pernah teramati.
Bila suatu saat merchant memakainya, pembayaran akan tertolak diam-diam sampai
judul barunya ditambahkan — dan itu akan terlihat di `GET /events` sebagai event
yang tidak dikenali.

Sumber pembayaran adalah `com.gojek.gopaymerchant`. Perangkat **tidak boleh**
memantau `com.gojek.gopay` sekaligus: kedua aplikasi melaporkan pembayaran yang
sama dengan teks identik, sehingga satu pembayaran akan menghasilkan dua
`event_id` berbeda dan terhitung dua kali.

### Kenapa nama connector tidak ada di URL

Rutenya `POST /events`, bukan `POST /callback/gopay`. Payload untuk setiap
sumber identik dan pembedanya hanya field `source`, jadi menambah DANA atau OVO
tidak boleh berarti menambah rute, handler, dan dokumen sekaligus.

Daftar sumber yang dikenal sebuah build dibaca dari `GET /sources`, sehingga
dashboard tidak perlu menanam daftar connector di sisi klien.

Allowlist adalah **konfigurasi, bukan kode** — menambah judul tidak menuntut deploy.

Sifat kegagalannya disengaja: notifikasi yang belum pernah dilihat tertolak otomatis, sehingga pembayaran dapat *terlewat* tetapi tidak pernah *salah dianggap lunas*. Celah "terlewat" ditutup oleh `GET /events`, tempat event tak dikenali tetap terlihat untuk ditinjau.

Format nominal yang wajib ditangani parser backend, dikonfirmasi dari perangkat target: `Rp1`, `Rp25.000`, `Rp 25.000`, `Rp25,000`.

---

### 4.4 Arti `amount_hint`

Namanya sengaja canggung. Ini **bukan** pernyataan bahwa sebuah pembayaran terjadi.

- Backend **wajib** melakukan ekstraksi nominalnya sendiri dari `notification.title` + `notification.text`.
- Backend **wajib** menentukan arah transaksi (uang masuk / uang keluar / bukan transaksi) dari teks mentah.
- `amount_hint` boleh dipakai untuk tampilan dan pembanding, **tidak boleh** menjadi dasar keputusan pembayaran.

Alasannya ada di [spec §2.2](superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md).

### 4.5 Response

**Diterima:**

```json
{ "success": true, "event_id": "evt_3f9a...", "status": "accepted" }
```

**Sudah pernah masuk:**

```json
{ "success": true, "event_id": "evt_3f9a...", "status": "duplicate" }
```

**Ditolak:**

```json
{ "success": false, "error": "clock_skew", "message": "...", "server_time": 1789036500 }
```

### 4.6 Tabel kode → tindakan HP

Tabel ini menentukan perilaku `EventUploadWorker` dan **wajib** diuji langsung terhadapnya.

| Kode | `status` / `error` | Tindakan HP | Retry |
|---|---|---|---|
| `200` | `accepted` | → `SENT` | — |
| `200` | `duplicate` | → `SENT` — bukan error | — |
| `400` | `invalid_payload` | → `FAILED` | tidak |
| `401` | `invalid_signature` | → `FAILED`, tampilkan peringatan kredensial | tidak |
| `401` | `clock_skew` | → `FAILED`, tampilkan pesan **jam HP meleset** + jam server | tidak |
| `403` | `device_disabled` | → `FAILED` | tidak |
| `429` | — | tetap `PENDING`, hormati `Retry-After` | ya |
| `5xx` | — | tetap `PENDING` | ya |
| timeout / jaringan mati | — | tetap `PENDING` | ya |

`duplicate` diperlakukan sebagai sukses. Inilah yang membuat retry aman: bila response hilang di tengah jalan padahal backend sudah menerima, percobaan berikutnya membalas `duplicate` dan event beres — tidak ada pembayaran terhitung dua kali.

`invalid_signature` dan `clock_skew` sama-sama 401 tetapi **wajib** dibedakan. Keduanya menuntut tindakan yang sama sekali berbeda: yang satu berarti secret salah, yang satu berarti jam HP perlu disetel. Tanpa pembedaan ini gejalanya identik dan menghabiskan waktu debugging.

### 4.7 Idempotency di sisi backend

```sql
INSERT INTO notification_events (...) VALUES (...)
ON CONFLICT (event_id) DO NOTHING
```

Jumlah baris terpengaruh menentukan `accepted` (1) atau `duplicate` (0). Tidak boleh memakai `SELECT` lebih dulu lalu `INSERT` — cara itu salah bila dua request identik tiba bersamaan.

### 4.8 Pengiriman

Satu event per request, dikirim berurutan dari antrean. Bukan batch. Bila satu event gagal permanen, event lain di antrean tetap jalan.

---

## 5. `GET /health`

Tanpa autentikasi.

```json
{ "status": "ok", "server_time": 1789036200 }
```

`server_time` disertakan agar HP dapat mendeteksi jamnya sendiri meleset **sebelum** mengirim apa pun, lalu menampilkan pesan yang benar alih-alih membiarkan user mengira kredensialnya salah.

Dipakai Dashboard untuk indikator status backend.

---

## 6. `GET /device/me`

Auth HMAC. Dipakai tombol *Test Connection* di Settings.

```json
{
  "device_id": "dev_01ABC",
  "name": "HP GoPay Utama",
  "enabled": true,
  "last_seen_at": "2026-09-10T19:30:00+07:00"
}
```

Setiap request ber-HMAC yang berhasil memperbarui `last_seen_at`. Tidak ada heartbeat berkala — `last_seen_at` diperbarui dari request yang memang terjadi, sehingga tidak ada network request tambahan yang membebani baterai.

---

## 6b. `POST /devices/heartbeat`

Auth HMAC. Dikirim perangkat setiap **15 menit** — interval minimum WorkManager
untuk periodic work.

### Request

```json
{
  "android_version": "13",
  "listener_connected": true,
  "pending_count": 2,
  "failed_count": 1
}
```

| Field | Tipe | Catatan |
|---|---|---|
| `android_version` | string | Kosong tidak menimpa nilai yang sudah diketahui |
| `listener_connected` | boolean | Status **ikatan** `NotificationListenerService`, bukan status izin |
| `pending_count` | number | Antrean yang belum terkirim di perangkat; tidak boleh negatif |
| `failed_count` | number | Event gagal permanen di perangkat; tidak boleh negatif |

`listener_connected` adalah informasi paling berharga di sini. Izin aktif
tetapi listener tidak terikat adalah gejala service dibunuh OEM, dan laporan
itu **tidak** ditolak — menolaknya berarti membuang justru sinyal yang dicari.
Backend mencatatnya sebagai peringatan di log.

### Response

```json
{ "success": true, "status": "ok", "server_time": 1789036200 }
```

`server_time` disertakan agar perangkat dapat mendeteksi jamnya meleset tanpa
memanggil `/health` terpisah.

### Status device

Diturunkan backend dari `heartbeat_at`, bukan disimpan:

| Status | Arti |
|---|---|
| `PENDING` | Terdaftar, belum pernah mengirim heartbeat |
| `ONLINE` | Heartbeat terakhir dalam **45 menit** |
| `OFFLINE` | Tidak ada heartbeat dalam 45 menit |
| `DISABLED` | Dinonaktifkan dari backend |

Toleransi 45 menit adalah **tiga kali** interval, bukan satu. Android menunda
periodic work saat Doze, dan status yang sering salah adalah status yang
diabaikan orang.

---

## 7. `GET /events`

Dilindungi basic auth di Caddy, bukan HMAC — ini untuk dibuka di browser saat verifikasi M2 dan M4.
Handler yang sama juga dipasang di `GET /admin/events` untuk dashboard, di
belakang `requireAdmin` (cookie sesi) alih-alih basic auth Caddy — itu jalur
yang dipakai halaman Events dashboard sub-project 4.

```
GET /events?limit=50&offset=0&source=gopay&q=dev_01&from=2026-09-01&to=2026-09-30
```

| Query param | Wajib | Arti |
|---|---|---|
| `limit` | tidak (default 50, maks 1000) | jumlah baris |
| `offset` | tidak (default 0) | untuk paginasi |
| `source` | tidak | harus salah satu ID di `internal/connector` (mis. `gopay`); ID tidak dikenal → `400 invalid_payload` |
| `q` | tidak | cocok sebagian ke `device_id` ATAU `title`, tanpa peduli huruf besar/kecil |
| `from`, `to` | tidak | tanggal saja (`YYYY-MM-DD`, UTC), inklusif kedua ujungnya — bukan RFC3339, sengaja selaras dengan `<input type="date">` |

Mengembalikan event terbaru lebih dulu, memuat teks mentah dan payload asli.
Seluruh filter bersifat AND — mengisi lebih dari satu mempersempit hasil,
tidak memperluasnya.

---

## 8. Yang Belum Ada di v1

Ditunda ke sub-project 3, dicatat agar tidak terlupa:

- Pembuatan invoice dan alokasi nominal unik
- Matching event ke invoice
- Webhook ke website merchant (beserta signature dan retry-nya sendiri)
- API key merchant
- Halaman admin

Perubahan yang menyentuh bentuk payload event akan naik ke `/api/v2/`.
