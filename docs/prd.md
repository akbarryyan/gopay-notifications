# Product Requirements Document (PRD)

## 1. Informasi Produk

**Nama Produk:** GoPay Notification Bridge
**Platform:** Android
**Jenis Produk:** Utility / Notification Listener
**Status:** Draft

---

## 2. Ringkasan Produk

GoPay Notification Bridge adalah aplikasi Android yang berfungsi sebagai penghubung antara notifikasi transaksi GoPay yang diterima pada perangkat Android dengan backend website.

Aplikasi akan memantau notifikasi Android, mendeteksi notifikasi yang berasal dari aplikasi GoPay, mengambil informasi transaksi yang tersedia pada notifikasi, kemudian mengirimkan data tersebut ke backend melalui HTTP/HTTPS API.

Tujuan utama aplikasi adalah memungkinkan sistem website menerima informasi transaksi berdasarkan notifikasi GoPay secara otomatis tanpa perlu melakukan input manual.

Aplikasi ini **tidak mengakses atau memodifikasi aplikasi GoPay secara langsung**. Aplikasi hanya memanfaatkan mekanisme Notification Listener yang disediakan oleh Android untuk membaca notifikasi yang telah muncul pada perangkat.

---

## 3. Latar Belakang

Website yang membutuhkan konfirmasi pembayaran tertentu dapat mengalami keterbatasan apabila tidak memiliki mekanisme callback langsung dari sumber pembayaran.

Dengan memanfaatkan Android Notification Listener, perangkat Android dapat digunakan sebagai notification bridge:

```text
GoPay
  ↓
Notifikasi Android
  ↓
GoPay Notification Bridge
  ↓
Backend API
  ↓
Website
```

Dengan pendekatan ini, informasi yang tersedia pada notifikasi dapat diteruskan secara otomatis ke backend untuk diproses lebih lanjut.

---

## 4. Problem Statement

Sistem website membutuhkan cara untuk menerima informasi transaksi GoPay secara otomatis dari perangkat Android.

Permasalahan yang ingin diselesaikan:

1. Tidak ada proses otomatis untuk meneruskan notifikasi GoPay ke backend.
2. Input transaksi berdasarkan notifikasi masih membutuhkan proses manual.
3. Website membutuhkan event ketika terdapat notifikasi transaksi baru.
4. Sistem harus mampu mengidentifikasi bahwa notifikasi berasal dari aplikasi GoPay.
5. Notifikasi yang sama tidak boleh menyebabkan transaksi diproses berkali-kali.

---

## 5. Tujuan Produk

### Primary Goals

1. Mendeteksi notifikasi yang berasal dari GoPay.
2. Mengambil informasi transaksi yang tersedia pada notifikasi.
3. Mengirimkan informasi tersebut ke backend secara otomatis.
4. Menyediakan mekanisme autentikasi antara Android app dan backend.
5. Mencegah event yang sama dikirim atau diproses berulang kali.
6. Menyediakan status sederhana mengenai kondisi listener dan koneksi backend.

### Secondary Goals

1. Menyediakan log transaksi/notifikasi yang telah diterima.
2. Menyediakan konfigurasi endpoint backend.
3. Menyediakan mekanisme retry ketika pengiriman gagal.
4. Memudahkan debugging ketika terjadi masalah komunikasi dengan backend.

---

## 6. Non-Goals

Produk ini **tidak bertujuan untuk**:

1. Mengakses data internal aplikasi GoPay.
2. Mengakses database atau API internal GoPay.
3. Mengotomatisasi login ke aplikasi GoPay.
4. Melakukan transaksi GoPay secara otomatis.
5. Mengirim pembayaran melalui GoPay.
6. Mengubah atau mengontrol aplikasi GoPay.
7. Membaca seluruh notifikasi perangkat untuk kemudian dikirim ke backend.
8. Menjamin bahwa seluruh detail transaksi tersedia apabila informasi tersebut tidak diberikan oleh notifikasi Android.

---

## 7. Target User

### Primary User

Pemilik sistem/backend website yang membutuhkan notification bridge untuk menerima event transaksi GoPay.

### Secondary User

Operator atau administrator yang memasang dan mengonfigurasi aplikasi Android pada perangkat yang digunakan sebagai notification bridge.

---

# 8. User Stories

## 8.1 Setup

**Sebagai pengguna**, saya ingin mengetahui apakah Notification Access sudah diberikan agar saya dapat memastikan aplikasi dapat bekerja.

**Sebagai pengguna**, saya ingin dapat membuka halaman pengaturan Notification Access Android agar saya dapat memberikan permission kepada aplikasi.

---

## 8.2 Notification Monitoring

**Sebagai pengguna**, saya ingin aplikasi hanya memproses notifikasi dari GoPay agar notifikasi aplikasi lain tidak ikut dikirim ke backend.

**Sebagai pengguna**, saya ingin aplikasi mendeteksi notifikasi GoPay secara otomatis tanpa perlu menekan tombol setiap kali ada transaksi.

---

## 8.3 Notification Processing

**Sebagai pengguna**, saya ingin informasi yang tersedia pada notifikasi GoPay diproses menjadi data terstruktur agar dapat digunakan oleh backend.

Contoh informasi yang dapat diproses:

- Judul notifikasi
- Isi notifikasi
- Nama aplikasi/package
- Waktu notifikasi diterima
- Informasi nominal jika tersedia
- Informasi transaksi lain jika tersedia

---

## 8.4 Backend Integration

**Sebagai pengguna**, saya ingin aplikasi mengirimkan event notifikasi ke backend secara otomatis.

**Sebagai pengguna**, saya ingin aplikasi memberikan informasi ketika pengiriman ke backend berhasil.

**Sebagai pengguna**, saya ingin aplikasi melakukan retry ketika backend sementara tidak dapat diakses.

---

## 8.5 Transaction Safety

**Sebagai pengguna**, saya ingin notifikasi yang sama tidak diproses dua kali agar status transaksi di website tidak mengalami duplikasi.

**Sebagai pengguna**, saya ingin backend dapat memvalidasi request dari perangkat Android sebelum memproses event.

---

# 9. Product Scope

## 9.1 Notification Listener

Aplikasi harus menggunakan Android Notification Listener untuk menerima event notifikasi.

Listener harus:

- Berjalan sebagai service.
- Mendeteksi notifikasi baru.
- Mengidentifikasi aplikasi sumber notifikasi.
- Memfilter hanya notifikasi yang berasal dari GoPay.
- Mengambil informasi yang tersedia pada notification payload.

---

## 9.2 GoPay Notification Filter

Aplikasi hanya memproses notifikasi yang berasal dari package aplikasi GoPay yang telah dikonfigurasi.

Notifikasi dari aplikasi lain harus diabaikan.

Contoh:

```text
GoPay       → PROCESS
WhatsApp    → IGNORE
Gmail       → IGNORE
Instagram   → IGNORE
Bank        → IGNORE
```

Identifikasi aplikasi harus menggunakan package identifier yang sesuai dengan aplikasi GoPay yang terpasang pada perangkat, bukan hanya berdasarkan nama aplikasi yang tampil kepada pengguna.

---

## 9.3 Notification Data

Setiap notifikasi GoPay yang diproses harus menghasilkan event terstruktur.

Contoh konsep data:

```json
{
  "source": "gopay",
  "title": "Pembayaran diterima",
  "message": "Rp25.000",
  "received_at": "2026-09-10T19:30:00+07:00"
}
```

Struktur final dapat berubah berdasarkan kebutuhan backend dan informasi yang benar-benar tersedia dari notifikasi GoPay.

---

## 9.4 Backend Callback

Aplikasi harus memiliki kemampuan mengirim event ke backend melalui API.

Konsep komunikasi:

```text
Android
   ↓
HTTP/HTTPS POST
   ↓
Backend Callback Endpoint
```

Backend bertanggung jawab untuk:

- Memvalidasi request.
- Memvalidasi perangkat.
- Memeriksa duplicate event.
- Memproses data transaksi.
- Menentukan apakah event dapat digunakan untuk memperbarui transaksi.

Android tidak boleh menjadi sumber keputusan final mengenai status pembayaran.

---

## 9.5 Authentication

Komunikasi Android dengan backend harus menggunakan mekanisme autentikasi.

Tujuannya:

- Mencegah request sembarang masuk ke callback endpoint.
- Mengidentifikasi perangkat yang mengirim event.
- Membatasi penggunaan endpoint.
- Menjaga integritas komunikasi.

Detail mekanisme authentication ditentukan pada dokumen `detail-project.md`.

---

## 9.6 Retry Mechanism

Jika request gagal karena masalah jaringan atau backend tidak tersedia, aplikasi harus memiliki mekanisme retry.

Contoh:

```text
Notification diterima
        ↓
Kirim ke backend
        ↓
    FAILED
        ↓
Simpan event sementara
        ↓
Retry
        ↓
    SUCCESS
```

Retry tidak boleh menyebabkan event diproses dua kali oleh backend.

---

## 9.7 Event History / Log

Aplikasi sebaiknya menyediakan riwayat event untuk membantu pengguna mengetahui apakah notification listener dan backend integration berjalan dengan baik.

Informasi minimal:

- Waktu event.
- Judul notifikasi.
- Status pengiriman.
- Waktu pengiriman.
- Response backend secara terbatas.
- Status retry jika ada.

Data sensitif yang tidak diperlukan tidak boleh ditampilkan atau disimpan secara berlebihan.

---

# 10. User Interface Requirements

UI aplikasi harus sederhana karena aplikasi berfungsi sebagai background utility.

## 10.1 Dashboard

Dashboard menampilkan:

```text
GoPay Notification Bridge

Notification Access
● Connected

Backend
● Connected

Listener
● Running

Last Event
10 Sep 2026, 19:30
Rp25.000

Last Sync
Success
```

Dashboard tidak perlu menampilkan desain yang kompleks.

---

## 10.2 Settings

Pengguna dapat mengatur:

- Backend URL.
- Device identifier jika diperlukan.
- Credential/token yang diperlukan.
- Pengaturan retry.
- Pengaturan log.

Credential/token harus disimpan menggunakan mekanisme penyimpanan aman Android.

---

## 10.3 Event History

Halaman history menampilkan event yang pernah diterima.

Contoh:

```text
10 Sep 19:30
Rp25.000
Sent ✓

10 Sep 19:12
Rp50.000
Sent ✓

10 Sep 18:45
Rp100.000
Retrying...
```

---

# 11. Functional Requirements

### FR-01 — Notification Access

Aplikasi harus dapat mendeteksi apakah Notification Access telah diberikan.

### FR-02 — Notification Listener

Aplikasi harus dapat menerima notification event melalui Android Notification Listener.

### FR-03 — GoPay Filtering

Aplikasi harus hanya memproses notification event dari GoPay.

### FR-04 — Notification Parsing

Aplikasi harus dapat mengambil informasi yang tersedia dari notifikasi dan mengubahnya menjadi event terstruktur.

### FR-05 — Backend Request

Aplikasi harus dapat mengirim event ke backend menggunakan HTTPS.

### FR-06 — Authentication

Request ke backend harus memiliki mekanisme autentikasi.

### FR-07 — Retry

Aplikasi harus melakukan retry apabila request gagal karena kondisi yang dapat dipulihkan.

### FR-08 — Duplicate Prevention

Event harus memiliki identifier/idempotency mechanism sehingga event yang sama tidak diproses berkali-kali.

### FR-09 — Event Logging

Aplikasi harus mencatat status event untuk kebutuhan monitoring dan debugging.

### FR-10 — Service Status

Aplikasi harus memberikan indikator apakah notification listener sedang aktif.

---

# 12. Non-Functional Requirements

## Performance

Aplikasi harus dapat memproses notifikasi dengan delay seminimal mungkin setelah notifikasi diterima Android.

## Reliability

Kegagalan koneksi sementara tidak boleh menyebabkan event hilang apabila event tersebut sudah berhasil diterima oleh listener.

## Security

- Komunikasi backend menggunakan HTTPS.
- Credential tidak disimpan sebagai plaintext jika memungkinkan.
- Endpoint callback harus melakukan authentication.
- Data sensitif diminimalkan.
- Log tidak boleh menyimpan informasi sensitif secara berlebihan.

## Privacy

Aplikasi hanya boleh memproses notifikasi yang diperlukan untuk fungsi produk, yaitu notifikasi GoPay.

Notifikasi dari aplikasi lain harus diabaikan.

## Maintainability

Arsitektur aplikasi harus memungkinkan perubahan format notifikasi tanpa harus melakukan perubahan besar pada seluruh sistem.

---

# 13. Error Handling

Aplikasi harus menangani minimal kondisi berikut:

### Notification Access Disabled

```text
Notification Access belum aktif.

[Open Settings]
```

### Backend Unreachable

```text
Backend tidak dapat dihubungi.

Event disimpan dan akan dicoba kembali.
```

### Invalid Backend Response

```text
Backend memberikan response yang tidak valid.

Event ditandai sebagai failed.
```

### Authentication Failed

```text
Authentication gagal.

Periksa konfigurasi device/token.
```

### Unknown Notification Format

Jika notifikasi berasal dari GoPay tetapi formatnya tidak dikenali:

```text
GoPay notification detected
Parsing failed
```

Event dapat dicatat untuk debugging tanpa langsung dianggap sebagai pembayaran berhasil.

---

# 14. Payment Processing Principle

Aplikasi Android **tidak boleh menentukan bahwa sebuah pembayaran berhasil hanya berdasarkan asumsi dari teks notifikasi**.

Android hanya bertugas sebagai:

```text
Notification Source
       ↓
Event Collector
       ↓
Event Forwarder
```

Sedangkan backend bertugas sebagai:

```text
Event Receiver
       ↓
Validation
       ↓
Duplicate Check
       ↓
Transaction Matching
       ↓
Business Decision
```

Dengan demikian, Android app tetap sederhana dan keputusan bisnis tetap berada di server.

---

# 15. High-Level Flow

```text
┌──────────────┐
│    GoPay     │
└──────┬───────┘
       │
       │ Notification
       ▼
┌──────────────────────┐
│ Android Notification │
│       System         │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────────┐
│ GoPay Notification      │
│ Bridge                  │
│                         │
│ Notification Listener   │
│        ↓                │
│ Filter GoPay            │
│        ↓                │
│ Parse Event             │
│        ↓                │
│ Queue / Retry           │
└───────────┬─────────────┘
            │
            │ HTTPS POST
            ▼
┌─────────────────────────┐
│       Backend API       │
│                         │
│ Authentication          │
│        ↓                │
│ Validation              │
│        ↓                │
│ Idempotency             │
│        ↓                │
│ Transaction Matching    │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│       Website           │
│                         │
│ Update Transaction      │
│ / Payment Status        │
└─────────────────────────┘
```

---

# 16. Acceptance Criteria

Produk dianggap memenuhi MVP apabila:

- [ ] User dapat memberikan Notification Access kepada aplikasi.
- [ ] Aplikasi dapat mendeteksi notifikasi baru.
- [ ] Aplikasi dapat membedakan notifikasi GoPay dengan aplikasi lain.
- [ ] Notifikasi GoPay dapat diubah menjadi event terstruktur.
- [ ] Event dapat dikirim ke backend melalui HTTPS.
- [ ] Backend dapat mengidentifikasi perangkat pengirim.
- [ ] Event yang sama tidak diproses dua kali.
- [ ] Event tetap dapat dikirim setelah koneksi kembali normal.
- [ ] User dapat melihat status listener.
- [ ] User dapat melihat status komunikasi dengan backend.
- [ ] User dapat melihat event/history dasar.
- [ ] Aplikasi tidak memproses notifikasi dari aplikasi selain GoPay.

---

# 17. MVP Scope

Versi MVP hanya mencakup:

1. Notification Access.
2. GoPay Notification Listener.
3. GoPay notification filtering.
4. Basic notification parsing.
5. Backend callback API integration.
6. Basic authentication.
7. Basic retry.
8. Duplicate prevention.
9. Basic event history.
10. Basic dashboard/status.

Fitur seperti analytics, multi-device management, remote configuration, advanced monitoring, dan dashboard web khusus tidak termasuk MVP.

---

# 18. Future Development

Fitur yang dapat dipertimbangkan setelah MVP:

- Multi-device management.
- Remote device monitoring.
- Web dashboard untuk melihat status Android device.
- Advanced notification parser.
- Notification rules.
- Multiple payment sources.
- Advanced retry queue.
- Push notification untuk error.
- Device health monitoring.
- Centralized event monitoring.
- Automatic configuration.
- Webhook forwarding ke beberapa endpoint.

---

# 19. Product Principle

Prinsip utama produk:

> **Android menerima event, backend yang mengambil keputusan.**

Android app harus dibuat sebagai bridge yang ringan, reliable, dan aman. Business logic, validasi transaksi, idempotency, dan keputusan pembayaran harus tetap berada di backend.

---

# 20. Open Questions

Hal-hal berikut perlu ditentukan sebelum masuk tahap technical design:

1. Package identifier GoPay yang akan digunakan untuk filtering.
2. Format notifikasi GoPay yang benar-benar diterima pada perangkat.
3. Informasi apa saja yang tersedia pada notifikasi.
4. Backend menggunakan teknologi apa.
5. Format endpoint callback.
6. Mekanisme authentication Android → backend.
7. Apakah satu Android device atau multiple devices.
8. Bagaimana transaksi dicocokkan dengan event pembayaran.
9. Apakah nominal unik akan digunakan untuk matching.
10. Berapa lama event disimpan di Android.
11. Bagaimana strategi retry.
12. Berapa lama history event disimpan.
13. Apakah aplikasi harus tetap berjalan setelah perangkat restart.
14. Apakah backend membutuhkan heartbeat/status dari Android device.
15. Bagaimana menangani perubahan format notifikasi GoPay.
