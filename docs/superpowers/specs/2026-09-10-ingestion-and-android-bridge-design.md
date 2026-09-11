# Design Spec — Event Ingestion & Android Bridge

**Tanggal:** 2026-09-10
**Cakupan:** Sub-project 1 (Event Ingestion) + Sub-project 2 (Android Bridge)
**Status:** Disetujui, siap masuk tahap implementation plan

---

## 1. Konteks & Pemecahan Scope

Tujuan akhir yang diinginkan adalah **self-hosted payment gateway**: website merchant membuat invoice, customer transfer ke akun GoPay pribadi, sistem mendeteksi transfer masuk dari notifikasi Android, mencocokkannya ke invoice, lalu mengirim webhook balik ke merchant.

Itu terlalu besar untuk satu spec. Dipecah jadi tiga sub-project:

| # | Sub-project | Isi | Status |
|---|---|---|---|
| 1 | **Event ingestion** | Endpoint callback, auth device, penyimpanan event, idempotency | **Spec ini** |
| 2 | **Android bridge** | Notification listener, antrean, pengiriman ber-HMAC, UI | **Spec ini** |
| 3 | **Gateway** | Invoice, nominal unik, matching engine, webhook merchant, admin | Ditunda |

Urutan ini disengaja. Sub-project 1 + 2 membuktikan asumsi paling berisiko di seluruh project: **apakah notifikasi GoPay benar-benar bisa ditangkap, dikirim, dan dibaca dengan andal.** Kalau tidak, seluruh sub-project 3 tidak ada gunanya dibangun.

### Hal yang sudah diketahui akan mendominasi sub-project 3

Dicatat di sini supaya tidak hilang, **bukan** untuk dikerjakan sekarang:

Transfer GoPay pribadi tidak membawa nomor order. Yang tersedia hanya nominal, nama pengirim, dan waktu. Matching praktis hanya bisa bersandar pada **nominal unik** (order Rp25.000 → customer diminta bayar Rp25.317). Konsekuensinya: ruang kode unik terbatas sehingga butuh alokasi anti-tabrakan, invoice wajib punya masa berlaku agar kode bisa didaur ulang, dan bila dua invoice cocok sistem harus menolak dan minta konfirmasi manual alih-alih menebak.

---

## 2. Keputusan Arsitektur Inti

### 2.1 Kotlin memiliki pipeline data, React Native memiliki UI

Ini keputusan yang paling menentukan bentuk seluruh sistem, dan menyimpang dari [detail-project.md §40](../../detail-project.md).

Saat aplikasi ditutup, Android tetap menghidupkan proses untuk mengirim notifikasi ke `NotificationListenerService` — **Kotlin pasti jalan**. Yang tidak ikut hidup adalah JS runtime React Native; instance-nya baru dibuat saat Activity dibuka. Kalau pengiriman HTTP ditulis di TypeScript seperti diusulkan dokumen, event baru terkirim saat user membuka aplikasi — bisa berjam-jam kemudian. Untuk konfirmasi pembayaran, itu setara dengan tidak berfungsi.

| Layer | Tanggung jawab |
|---|---|
| **Kotlin** | Listener → filter package → simpan ke DB → WorkManager → POST + retry |
| **TypeScript** | Dashboard, History, Settings, Debug, format tampilan, navigasi |

**Konsekuensi:** native adalah satu-satunya pemilik database. TypeScript tidak pernah membuka file DB sendiri (dua penulis ke file yang sama adalah sumber bug), melainkan membaca lewat native module.

Alternatif yang ditolak: (b) native hanya persist, RN yang kirim saat dibuka — delay tidak dapat diterima. (c) headless JS task dibangunkan native tiap notifikasi — setia pada semangat dokumen, tapi menuntut startup JS runtime ~1–2 detik per notifikasi dan jauh lebih rapuh di HP Xiaomi/Oppo/Vivo yang agresif membunuh background process.

### 2.2 Parsing nominal bersifat display-only

Kotlin melakukan regex sederhana untuk `Rp25.000` / `Rp 25.000` / `Rp25,000`. Gagal → `null`. **Tidak pernah menebak** — angka yang tidak jelas didahului `Rp` bukan nominal.

Payload yang dikirim **selalu memuat `title` + `text` mentah**. Backend melakukan ekstraksi nominalnya sendiri dari teks mentah dan memperlakukan nilai dari HP sebagai petunjuk belaka.

Alasan: kalau parser hanya ada di Kotlin, setiap perubahan format notifikasi GoPay menuntut build ulang APK dan pasang ulang ke HP. Dengan parsing otoritatif di backend, perubahan format cukup di-deploy. Parser di HP boleh salah tanpa merusak apa pun — paling buruk angka di layar sempat keliru.

### 2.3 Arah transaksi ditentukan backend, bukan HP

Akun GoPay pribadi menerima campuran notifikasi dari package yang sama:

```
Uang masuk        "Kamu menerima Rp25.000 dari ..."   ← yang dicari
Uang keluar       "Pembayaran Rp25.000 berhasil"      ← BAHAYA
Promo/cashback    "Cashback Rp10.000 menantimu!"      ← noise
```

Baris kedua berbahaya: membayar Rp25.000 ke merchant juga memicu notifikasi berisi Rp25.000. Bila diteruskan mentah dan backend hanya mencocokkan nominal, order Rp25.000 yang `PENDING` bisa ditandai PAID padahal tidak ada uang masuk.

**Keputusan:** Android meneruskan notifikasi dari package GoPay apa adanya, tanpa menyaring berdasarkan arah. Backend menentukan arah dari teks mentah dan hanya memproses yang jelas uang masuk; teks ambigu ditolak, bukan ditebak.

Untuk mengurangi noise promo, Settings menyediakan **daftar kata-diabaikan** yang dapat diedit tanpa rebuild. Default konservatif — bila ragu, tetap kirim. Lebih baik backend menolak sepuluh promo daripada satu transfer asli tersaring diam-diam di HP.

**Aturan arah di backend berupa allowlist judul, bukan blocklist.** Hanya judul yang terdaftar yang boleh dicocokkan ke pembayaran; apa pun selain itu disimpan tetapi tidak pernah dianggap uang masuk. Konsekuensinya notifikasi yang belum pernah dilihat tertolak otomatis — kegagalannya bersifat aman: pembayaran bisa terlewat, tetapi tidak pernah salah dianggap lunas.

Celah "terlewat" ditutup oleh visibilitas: setiap event tetap tersimpan dan terlihat di `GET /events`. Bila suatu saat muncul transfer masuk berjudul lain (dari bank, dari QRIS), event-nya terlihat sebagai tidak dikenali dan judulnya tinggal ditambahkan. Allowlist adalah **konfigurasi, bukan kode** — menambah judul tidak menuntut deploy.

Karena keamanan sudah dijamin allowlist di backend, daftar kata-diabaikan di HP berfungsi murni sebagai pengurang noise dan **default-nya kosong**.

**Catatan privasi:** notifikasi transfer pribadi memuat nama pengirim, dan nama itu ikut terkirim ke backend. Diperlukan untuk matching, tetapi harus tercermin dalam kebijakan retensi.

### 2.4 Package GoPay tidak di-hardcode

Package sudah terkonfirmasi di perangkat target sebagai **`com.gojek.gopaymerchant`** (lihat §2.6.2), tetapi tetap **tidak di-hardcode**. Salah satu huruf saja membuat aplikasi tidak pernah menerima satu event pun, dengan gejala yang identik dengan Notification Access belum aktif — dan nama package dapat berubah antar versi aplikasi.

- Package yang dipantau disimpan sebagai **daftar**, dapat diedit di Settings tanpa rebuild.
- **Mode Discovery** di layar Debug: default mati, mati otomatis setelah 10 menit, mencatat *hanya* nama package dan judul ke penyimpanan lokal, **tidak pernah mengirim apa pun keluar HP**.

Mode Discovery tetap dibangun meski package sudah diketahui: ia yang akan dipakai bila GoPay mengganti package di masa depan, dan untuk memeriksa apa yang sebenarnya sampai ke listener saat sesuatu tidak bekerja.

### 2.5 Auth memakai HMAC signature

Endpoint ini yang nantinya menentukan order ditandai lunas — siapa pun yang bisa mengirim request palsu mendapat barang gratis.

Bearer token ditolak karena token ikut terkirim di setiap request; satu kebocoran (log server, proxy, HP hilang) berarti penyerang bisa memalsukan pembayaran tanpa batas waktu. Dengan HMAC, secret tidak pernah ikut terkirim, request yang tersadap tidak bisa dipakai ulang, dan penyerang harus benar-benar menguasai isi HP.

Yang tidak dilindungi: HP yang jatuh ke tangan orang dan di-root. Penangkalnya bukan kriptografi melainkan kemampuan **mencabut device** dari sisi backend.

Distribusi secret: di-generate di backend, ditempel manual sekali di Settings, disimpan dengan `expo-secure-store`. Tidak perlu alur pairing untuk satu device.

### 2.6 Data lapangan (2026-09-10, **sebagian usang**)

> **Perubahan scope 2026-09-11.** Sumber pembayaran berpindah dari **akun GoPay
> pribadi** ke **GoPay Merchant**, sepenuhnya — akun pribadi tidak lagi dipakai.
> Seluruh data di bagian ini diambil dari akun pribadi dan karena itu **tidak
> lagi berlaku sebagai nilai produksi**: package `com.gojek.gopay` dan judul
> `Transfer masuk` keduanya harus ditemukan ulang di aplikasi merchant.
>
> Yang **tetap berlaku**: pelajaran bentuknya, bukan nilainya. GoPay memakai
> notification id konstan, `Notification.when` bertahan lintas repost, format
> nominal `Rp1` tanpa pemisah, dan notifikasi transaksi dapat datang lewat
> channel promosi. Keempatnya mendasari §4.1 dan tidak berubah.
>
> Dampak ke kode: **tidak ada.** Package adalah daftar yang dapat diedit di
> Settings, dan aturan arah adalah konfigurasi backend — keduanya sengaja
> dirancang begitu justru untuk perubahan seperti ini.

### 2.6.1 Data akun pribadi (arsip, tidak dipakai produksi)

Diambil dari perangkat target dengan `adb shell dumpsys notification --noredact` atas satu transfer masuk sungguhan.

| | |
|---|---|
| Package | `com.gojek.gopay` |
| Title | `Transfer masuk` |
| Text | `Rp1 dari icaangg udah masuk ke GoPay kamu.` |
| BigText | identik dengan text |
| Notification id / tag | `-1` / `null` — **konstan untuk semua notifikasi GoPay** |
| Channel | `promotional_notifications` ("Promotions and Marketing") |
| Perangkat | **OPPO CPH2365, Android 13, ColorOS** |

Empat hal yang mengikat desain:

**Judul `Transfer masuk` adalah pembeda arah yang bersih.** Ia menjadi entri pertama allowlist backend.

**Format nominal `Rp1`** — tanpa spasi, tanpa pemisah ribuan untuk angka kecil. Parser wajib menangani `Rp1` sekaligus `Rp25.000`.

**Channel-nya adalah channel promosi.** Notifikasi transaksi dan notifikasi promo berbagi channel yang sama, sehingga channel tidak dapat dipakai sebagai penyaring. Lebih penting: **channel "Promotions and Marketing" tidak boleh dimatikan di HP** — mematikannya ikut mematikan notifikasi transfer masuk, dan seluruh sistem berhenti bekerja tanpa gejala yang jelas. Ini masuk pemeriksaan QA.

**Id notifikasi konstan** — dasar perubahan formula `event_id` di §4.1.

Sampel notifikasi uang keluar dan promo sengaja **tidak dikumpulkan**. Dengan pendekatan allowlist, keduanya tidak perlu dikenali — cukup tidak cocok dengan allowlist, dan itu terjadi dengan sendirinya.

### 2.6.2 Data GoPay Merchant terkonfirmasi (2026-09-11)

Diambil dari perangkat target atas satu pembayaran QRIS sungguhan sebesar Rp1.

| | |
|---|---|
| Package | **`com.gojek.gopaymerchant`** |
| Title | `Pembayaran QRIS statis diterima` |
| Text | `Rp 1 di AKBAR RAYYAN AL GHIFARI, Digital & Kreatif.` |
| BigText | identik dengan text |
| Notification id / tag | `-1` / `null` — konstan, sama seperti aplikasi pribadi |
| Channel | `promotional_notifications` |
| Nomor referensi transaksi | **tidak ada** |

#### Bahaya: satu pembayaran menghasilkan dua notifikasi

Pembayaran yang sama dilaporkan oleh **dua aplikasi sekaligus**:

```
pkg=com.gojek.gopaymerchant  id=-1  "Pembayaran QRIS statis diterima"
pkg=com.gojek.gopay          id=3   "Pembayaran QRIS statis diterima"
```

Judul dan teksnya identik, hanya berbeda pada entitas HTML (`&` versus `&amp;`).

Bila keduanya dipantau, satu pembayaran menghasilkan **dua `event_id` berbeda**,
karena `packageName` ikut menjadi bahan hash di §4.1. Backend menerima keduanya
sebagai `accepted`, bukan `duplicate`, dan satu pembayaran terhitung dua kali.
Idempotency tidak menolong: ia dirancang menangkap notifikasi yang sama dari
sumber yang sama, bukan dua aplikasi yang melaporkan kejadian yang sama.

**Karena itu `monitoredPackages` wajib berisi tepat satu entri:
`com.gojek.gopaymerchant`.** Menambahkan `com.gojek.gopay` ke daftar akan
menyebabkan pembayaran ganda, bukan sekadar noise.

#### Konsekuensi lain

**Tidak ada nomor referensi transaksi.** Nominal unik karena itu tetap menjadi
satu-satunya jalur matching yang praktis di sub-project 3, sama seperti saat
sumbernya masih akun pribadi.

**QRIS statis** berarti pembayar mengetik sendiri nominalnya. Itu justru yang
membuat nominal unik dapat bekerja, tetapi juga membuka kemungkinan salah ketik:
uang masuk tanpa cocok dengan invoice mana pun, dan butuh penanganan manual.

**Format nominal `Rp 1`** — dengan spasi. Sudah ditangani `AmountParser`.

**Aplikasi merchant juga memasang notifikasi tanpa teks sama sekali** — title, text, dan bigText ketiganya `null`. Teramati di perangkat 2026-09-11; hampir pasti notifikasi ringkasan grup yang dipasang Android berdampingan dengan yang asli. Notifikasi semacam itu dilewati tanpa disimpan: ia tidak mungkin memuat pembayaran, dan menyimpannya hanya mengotori Riwayat serta menghabiskan satu request ke backend. Mode Discovery tetap menangkapnya bila suatu saat perlu diperiksa.

**Tidak ada PII pihak ketiga.** Teks memuat nama merchant, bukan nama pembayar.
Ini lebih baik daripada akun pribadi yang menyertakan nama pengirim, dan
meringankan kewajiban retensi di §4.5.

**Risiko yang hilang.** Akun merchant hampir hanya menerima, sehingga notifikasi
pembayaran **keluar** dengan nominal sama — bahaya utama di §2.3 — praktis tidak
ada lagi.

---

## 3. Tumpukan Teknologi

### 3.1 Repo

Satu repo untuk keduanya. Bila terpisah, format payload akan menyimpang diam-diam dan baru ketahuan saat event ditolak di lapangan.

```
gopay-notifications/
├── CLAUDE.md
├── docs/
│   ├── prd.md, detail-project.md
│   ├── api-contract.md              ← sumber kebenaran bersama
│   ├── qa/
│   │   ├── qa-rules.md
│   │   └── qa-report.md
│   └── superpowers/specs/
├── backend/                          ← Go
└── mobile/                           ← Expo
```

### 3.2 Backend Go

Dependensi ditekan seminimal mungkin — layanan ini akan jalan bertahun-tahun tanpa banyak disentuh.

| Bagian | Pilihan | Alasan |
|---|---|---|
| Routing | `net/http` bawaan (Go 1.22+) | `ServeMux` sudah mendukung pola `POST /path` |
| Database | **PostgreSQL** via `pgx` | lihat di bawah |
| Migrasi | `goose` | file SQL biasa |
| Reverse proxy | **Caddy** di VPS | Let's Encrypt otomatis |

Postgres dipilih sejak awal meski SQLite cukup untuk tahap ini, karena sub-project 3 membutuhkan alokasi nominal unik yang aman dari race condition (`SELECT ... FOR UPDATE`). Pindah database di tengah jalan jauh lebih mahal daripada memasang Postgres sekarang.

### 3.3 Android

| Bagian | Pilihan |
|---|---|
| Framework | **Expo** dengan prebuild + development build |
| Kode native | **Expo local module** (`create-expo-module --local`) |
| Manifest | **Config plugin** — `android/` di-generate ulang tiap prebuild |
| Database | **Room** (dimiliki native) |
| Pengiriman | **WorkManager** + OkHttp |
| Secret | **EncryptedSharedPreferences** (native) |
| Navigasi | **React Navigation** |

Expo Go tidak dapat dipakai sama sekali — ia berisi kumpulan native module tetap sehingga Kotlin buatan sendiri tidak akan masuk. Yang dipakai adalah development build: aplikasi sendiri berisi Kotlin sendiri, tetap dengan hot reload.

`expo-secure-store` **tidak** dipakai meski disebut di [detail-project.md §3.3](../../detail-project.md): ia hanya dapat dibaca dari JavaScript, sementara `EventUploadWorker` membaca `device_secret` justru ketika runtime JS sudah mati. Secret disimpan di `EncryptedSharedPreferences` di sisi native, yang memakai Android Keystore yang sama di baliknya — mekanisme perlindungannya identik, pemiliknya saja yang berpindah. Ini konsekuensi langsung dari §2.1.

**Library di dokumen yang gugur:**

| Library | Alasan |
|---|---|
| MMKV | Konfigurasi harus terbaca WorkManager saat app tertutup — MMKV dari sisi JS tidak bisa |
| expo-sqlite | Native memiliki database; dua penulis ke satu file adalah sumber bug |
| Axios | Seluruh jaringan terjadi di Kotlin (OkHttp), termasuk *Test Connection* — agar penandatanganan HMAC hanya punya satu implementasi |
| expo-secure-store | Tidak terbaca oleh WorkManager saat JS mati; diganti EncryptedSharedPreferences native |

**WorkManager** dipilih karena menjamin eksekusi walau proses mati, punya constraint `NetworkType.CONNECTED` (tidak perlu polling — Android yang membangunkan saat jaringan pulih), punya exponential backoff bawaan, dan memulihkan antreannya sendiri setelah device restart.

---

## 4. Model Data

### 4.1 Pembentukan `event_id`

Dibuat di Kotlin, deterministik, tanpa random:

```
event_id = "evt_" + sha256( packageName | title | text | when )[:32]
```

`when` adalah `Notification.when` — timestamp yang ditetapkan aplikasi. Bila bernilai 0, dipakai `sbn.postTime` sebagai cadangan.

Android memanggil `onNotificationPosted` berkali-kali untuk notifikasi yang sama (saat di-update, saat grup berubah). Dengan UUID acak, satu transfer bisa terkirim tiga kali sebagai tiga event berbeda dan backend tidak punya cara tahu itu satu. Dengan hash, ketiganya menghasilkan id identik dan tersaring sendiri.

**Dua bahan sengaja tidak dipakai**, berdasarkan temuan lapangan di §2.6:

`notificationKey` tidak dipakai karena GoPay memakai id `-1` tanpa tag untuk seluruh notifikasinya, sehingga key-nya identik untuk setiap notifikasi dan tidak menyumbang apa pun ke hash.

`postTime` tidak dipakai sebagai bahan utama karena nilainya **berubah setiap kali notifikasi di-posting ulang**. Memakainya berarti satu transfer yang di-repost GoPay menghasilkan dua `event_id` berbeda dan terkirim dua kali — persis kebalikan dari tujuan formula ini. `when` bertahan lintas repost.

Dua transfer asli dari pengirim yang sama dengan nominal sama tetap aman dibedakan karena `when` beresolusi milidetik.

### 4.2 Tabel `events` di HP (Room)

Indeks unik pada `event_id`, sehingga penyisipan ganda ditolak di level database.

| Kolom | Catatan |
|---|---|
| `event_id` | unik, deterministik |
| `package_name` | |
| `title`, `text`, `big_text` | mentah, apa adanya |
| `amount_hint` | display-only, boleh null |
| `posted_at` | waktu dari Android |
| `received_at` | waktu service menangkap |
| `status` | lihat §4.3 |
| `attempt_count` | |
| `last_error` | pesan singkat, tanpa data sensitif |
| `backend_status` | accepted / duplicate / rejected |
| `sent_at` | |

### 4.3 Status

```
PENDING  →  SENDING  →  SENT
                ↓
             FAILED  →  (kirim ulang manual)

IGNORED     ← tersaring daftar kata-diabaikan, tidak pernah dikirim
```

Lima status, bukan sembilan seperti [detail-project.md §27](../../detail-project.md). `RECEIVED`, `PARSED`, `PERSISTED`, `QUEUED` terjadi dalam satu transaksi tak terpisahkan — menyimpannya terpisah hanya menambah kemungkinan salah tanpa memberi informasi. `PARSE_FAILED` gugur karena teks mentah selalu diteruskan; gagal parsing nominal bukan kegagalan event, hanya `amount_hint` bernilai null. `RETRY_PENDING` diwakili `PENDING` dengan `attempt_count > 0`.

### 4.4 Tabel di backend

**`devices`** — `device_id`, nama, secret, status aktif, `last_seen_at`.

Verifikasi HMAC menuntut server memegang secret yang sebenarnya, bukan hash-nya, sehingga tidak bisa disimpan seperti password. Secret dienkripsi di kolom dengan kunci dari environment variable, agar dump database yang bocor tidak langsung membocorkan secret.

**`notification_events`** — `event_id` dengan unique constraint, `device_id`, teks mentah, `amount_hint`, waktu, dan payload asli disimpan utuh sebagai JSON.

Payload utuh disengaja: saat sub-project 3 membangun aturan matching, bentuk asli notifikasi yang pernah masuk akan dibutuhkan.

Idempotency bersandar pada constraint database, bukan `SELECT` lebih dulu: `INSERT ... ON CONFLICT (event_id) DO NOTHING`, lalu jumlah baris terpengaruh menentukan `accepted` atau `duplicate`. Cara ini benar bahkan bila dua request identik tiba bersamaan.

### 4.5 Retensi & batas retry

- Event `SENT` dan `IGNORED` dihapus otomatis dari HP setelah **30 hari** (teks memuat nama pengirim).
- Event `FAILED` tidak pernah dihapus otomatis.
- Backend menyimpan event permanen sebagai jejak audit.
- WorkManager mencoba dengan backoff sampai **~24 jam**, lalu event ditandai `FAILED` dan berhenti. Backend mati lebih dari sehari adalah masalah yang perlu dilihat langsung, bukan dicoba diam-diam selamanya.

---

## 5. Kontrak API

Detail lengkap ada di [`docs/api-contract.md`](../../api-contract.md). Ringkasnya:

| Method | Path | Auth |
|---|---|---|
| `POST` | `/api/v1/callback/gopay` | HMAC |
| `GET` | `/api/v1/health` | — |
| `GET` | `/api/v1/device/me` | HMAC |
| `GET` | `/api/v1/events` | basic auth (Caddy) |

`/health` tanpa auth dan mengembalikan jam server, agar HP dapat mendeteksi jamnya sendiri meleset sebelum mengirim apa pun.

`/events` diperlukan agar M2 dan M4 dapat diverifikasi — cukup JSON yang bisa dibuka di browser. Halaman admin adalah urusan sub-project 3.

---

## 6. Aplikasi Android

### 6.1 Lapisan Kotlin

```
GoPayListenerService.kt   NotificationListenerService — tangkap & saring
AmountParser.kt           regex nominal, display-only, fungsi murni
EventRepository.kt        Room DAO + insert anti-duplikat
EventUploadWorker.kt      HMAC + OkHttp + pemetaan kode response
GopayListenerModule.kt    Expo module — jembatan ke TypeScript
```

`AmountParser` dipisah sebagai fungsi murni tanpa dependensi Android agar dapat diuji JUnit tanpa emulator.

### 6.2 Alur saat notifikasi masuk

```
onNotificationPosted
  → package ada di daftar pantau?     tidak → berhenti, tidak dicatat
  → cocok daftar kata-diabaikan?      ya   → simpan sebagai IGNORED
  → hitung event_id, parse nominal
  → INSERT (abaikan jika event_id sudah ada)
  → antre WorkManager
  → kirim event ke JS bila UI hidup
```

Langkah terakhir hanya kosmetik agar Dashboard bereaksi saat sedang dilihat. Bila UI mati, tidak ada yang hilang.

### 6.3 Yang tidak perlu dibangun

**`BOOT_COMPLETED` receiver** ([detail-project.md §31](../../detail-project.md)) tidak diperlukan: Android mengikat ulang `NotificationListenerService` otomatis setelah booting, dan WorkManager memulihkan antreannya sendiri. Menambahkan receiver menduplikasi kerja sistem.

**Optimasi baterai khusus** ([§32](../../detail-project.md)) terpenuhi tanpa usaha: tidak ada polling. Service pasif menunggu dipanggil sistem; pengiriman dipicu constraint `NetworkType.CONNECTED`.

**Foreground service permanen** tidak dipakai. `NotificationListenerService` diikat sistem sehingga prosesnya sudah berprioritas tinggi; notifikasi permanen menambah gangguan tanpa menambah jaminan. Dapat ditambahkan belakangan tanpa membongkar apa pun bila terbukti perlu.

### 6.4 Ketahanan terhadap OEM

Risiko terbesar bukan Android melainkan lapisan hemat baterai Xiaomi/Oppo/Vivo/Samsung yang membunuh proses agresif dan kadang mencabut Notification Access diam-diam.

1. `onListenerDisconnected` memanggil `requestRebind`.
2. Dashboard menampilkan **status ikatan yang sebenarnya**, bukan sekadar "izin sudah diberikan". Selisih antara keduanya justru gejala HP membunuh service.
3. Saat setup, aplikasi mengarahkan user mematikan optimasi baterai untuk aplikasi ini.
4. Dashboard menampilkan **berapa lama sejak event terakhir**. Bila ColorOS diam-diam membunuh service, gejalanya terlihat sebagai "tidak ada event selama 3 hari" alih-alih tidak terlihat sama sekali.

**Perangkat target adalah OPPO CPH2365 dengan Android 13 (ColorOS)**, salah satu yang paling agresif. Langkah berikut wajib, bukan opsional, dan masuk daftar `NEEDS-DEVICE` di M2:

- Settings → Baterai → Manajemen baterai aplikasi → aplikasi ini → **Izinkan aktivitas latar belakang**, jangan dioptimalkan
- Settings → Apps → Manajemen aplikasi → aplikasi ini → **Izinkan mulai otomatis**
- Di layar recent apps, **kunci** aplikasi agar tidak ikut terhapus saat *clear all*
- Settings → Baterai → **Optimasi siaga tidur** → matikan

Keputusan "tanpa foreground service" tetap berlaku, tetapi M6 yang menentukan apakah ia bertahan. Bila uji ketahanan gagal di perangkat ini, foreground service ditambahkan.

### 6.5 Layar

| Layar | Isi |
|---|---|
| **Dashboard** | Notification Access, status ikatan listener, backend (dari `/health`), event terakhir, jumlah antrean tertunda |
| **History** | Daftar event + status, tombol kirim ulang untuk `FAILED` |
| **Settings** | Backend URL, Device ID, Device Secret (tersamar), daftar package dipantau, daftar kata-diabaikan, Test Connection, tombol optimasi baterai, hapus history |
| **Debug** | Mode Discovery + tombol kirim ulang event lama untuk menguji pipeline tanpa transfer sungguhan |

### 6.6 Keputusan kecil

- **Bahasa UI:** Indonesia.
- **Tiga lingkungan:** `development`, `uat`, `production`, masing-masing dengan package Android berbeda sehingga dapat terpasang bersamaan. Android menyandera data tiap package di sandbox-nya sendiri, sehingga database event, Device Secret, dan Settings tidak pernah bercampur antar lingkungan.

  UAT dan produksi berjalan di VPS yang sama sebagai dua layanan terpisah, dengan database dan `DEVICE_SECRET_KEY` masing-masing. Karena device terdaftar per database, HP yang salah diarahkan ditolak `401 invalid_signature` — gagalnya nyaring, bukan diam-diam.

  Varian di UI diambil dari package name yang benar-benar terpasang, bukan dari konfigurasi build. Dashboard menampilkan pita warna untuk varian non-produksi: tiga aplikasi dengan tampilan identik di satu HP adalah undangan untuk salah lihat.

  Konsekuensi yang perlu disadari: bila aplikasi UAT dan produksi sama-sama diberi Notification Access, keduanya menangkap setiap pembayaran, sehingga database UAT memuat catatan transaksi sungguhan.

- **Distribusi:** APK dari EAS Build, dipasang manual. Play Store dihindari — kebijakan mereka soal notification listener ketat dan distribusi publik tidak dibutuhkan.
- **Jumlah device:** satu, tetapi skema `device_id` + secret sudah mendukung banyak device tanpa perubahan.
- **Cleartext HTTP:** Android 9+ memblokir HTTP polos. Build **development** menyetel `usesCleartextTraffic: true` lewat `expo-build-properties`, sehingga Metro dan backend yang jalan di laptop dapat dihubungi. Build **production** tidak menyetelnya sama sekali, dan Android sejak targetSdk 28 memblokir cleartext secara default — jadi production tetap HTTPS-only tanpa konfigurasi apa pun.

  Rancangan awal memakai config plugin dengan daftar host development yang disebut satu per satu. Itu dibuang setelah terbukti rapuh: alamat LAN laptop berubah setiap ganti jaringan, dan `base-config cleartextTrafficPermitted="false"` ikut memblokir koneksi ke Metro sehingga aplikasi gagal start. Manfaat membatasi host hanya berlaku di build yang memang tidak pernah dirilis, sementara biayanya menghentikan pekerjaan setiap kali IP berpindah.

---

## 7. Strategi Pengujian

### Go

Verifikasi HMAC (tanda tangan benar, salah, secret tertukar, body dimodifikasi satu byte), toleransi timestamp di kedua ujung batas, dan idempotency.

Idempotency diuji dengan dua request identik dikirim **bersamaan** — harus menghasilkan tepat satu baris, satu `accepted`, satu `duplicate`. Diuji terhadap Postgres asli, bukan mock: perilaku `ON CONFLICT` justru yang sedang diuji, sehingga mengganti database berarti tidak menguji apa-apa.

### Kotlin

`AmountParser` terhadap ragam format di [detail-project.md §10](../../detail-project.md), termasuk yang **harus** menghasilkan `null`: teks tanpa `Rp`, angka yang ternyata nomor referensi, notifikasi promo.

Determinisme `event_id`: input sama → id sama; satu byte berubah → id berbeda.

Pemetaan kode response → tindakan, langsung dari tabel di `api-contract.md`.

### TypeScript

Tidak ada unit test. Lapisan ini murni tampilan; mengujinya berarti menguji React Navigation, bukan menguji produk.

### Di HP sungguhan

Daftar di [detail-project.md §35](../../detail-project.md) dipakai apa adanya, ditambah dua yang paling mungkin gagal: HP didiamkan semalaman lalu diuji lagi (menangkap OEM yang membunuh service saat Doze), dan Notification Access dicabut lalu diberikan lagi.

---

## 8. Milestone

| | Isi |
|---|---|
| **M0** | Repo, `git init`, kontrak API, aturan QA |
| **M1** | Backend Go jalan di VPS: `/health`, `/callback`, HMAC, Postgres, idempotency |
| **M2** | Aplikasi terpasang di HP, Notification Access aktif, setup ColorOS selesai, **listener benar-benar menangkap notifikasi GoPay sungguhan** ← titik penentu |
| **M3** | Room, penangkapan event, `event_id`, parser nominal |
| **M4** | WorkManager + pengiriman ber-HMAC + retry — **transfer sungguhan sampai ke VPS** |
| **M5** | Dashboard, History, Settings, Debug |
| **M6** | Uji ketahanan: restart, mode pesawat, didiamkan berhari-hari |

M2 sengaja mendahului pembangunan pipeline. Isi notifikasi sudah terbukti memadai lewat §2.6, sehingga yang tersisa untuk dibuktikan adalah yang tidak dapat dibuktikan lewat `adb`: bahwa `NotificationListenerService` kita sendiri benar-benar menerima event itu di perangkat ColorOS, dan tetap menerimanya setelah beberapa jam.

M1 mendahului M2 karena begitu backend hidup, ia sekaligus menjadi alat ukur — setiap event yang dikirim HP dapat langsung dilihat masuk atau tidak.

---

## 9. Penyimpangan dari Dokumen Awal

Seluruhnya sudah disetujui. Dicatat agar perbedaan antara dokumen dan kenyataan selalu terlihat dan tidak dikira kelalaian.

| # | Dokumen | Spec ini | Alasan |
|---|---|---|---|
| 1 | React Native CLI | Expo + prebuild + local module | Permintaan eksplisit |
| 2 | §40 "keep native part small" — RN pegang storage/API/retry | Kotlin pegang seluruh pipeline data | JS runtime mati saat UI tertutup |
| 3 | §3.3 MMKV | Konfigurasi di sisi native | WorkManager harus bisa membacanya saat app tertutup |
| 4 | §3.4 SQLite via RN | Room, dimiliki native | Dua penulis ke satu file adalah sumber bug |
| 5 | §3.2 Axios untuk pengiriman | OkHttp di Kotlin | Pengiriman terjadi di native |
| 5b | §3.3 credential di secure storage Android | EncryptedSharedPreferences native, bukan expo-secure-store | expo-secure-store tidak terbaca WorkManager saat JS mati |
| 6 | §16 `payment.amount` | `amount_hint` di tingkat atas | `payment.amount` terbaca seolah HP menyatakan sebuah pembayaran |
| 7 | §27 sembilan status | Lima status | Empat status menggambarkan momen yang tak dapat diamati terpisah |
| 8 | §31 boot handling | Tanpa `BOOT_COMPLETED` receiver | Sistem sudah melakukannya |
| 9 | §16 `/api/callback/gopay` | `/api/v1/callback/gopay` | Versioning murah sekarang, mahal nanti |
| 10 | §10 parser nominal di HP | Tetap ada, tapi display-only; backend otoritatif | Perubahan format GoPay tidak memaksa rilis APK |
| 11 | §6.6 cleartext dibatasi daftar host development | `usesCleartextTraffic` menyeluruh, hanya di build development | Daftar host putus setiap IP laptop berubah, dan ikut memblokir Metro. Production tetap HTTPS-only lewat default Android |

---

## 10. Open Question yang Terselesaikan

Merujuk [prd.md §20](../../prd.md):

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Package identifier GoPay | **`com.gojek.gopaymerchant`** — terkonfirmasi 2026-09-11, lihat §2.6.2. Wajib satu entri saja |
| 2 | Format notifikasi sebenarnya | **Terkonfirmasi** untuk merchant, lihat §2.6.2 |
| 3 | Informasi yang tersedia pada notifikasi | **Terkonfirmasi**: title, text, bigText, when. Tidak ada nomor referensi transaksi, tidak ada identitas pembayar |
| 4 | Teknologi backend | Go + PostgreSQL + Caddy di VPS |
| 5 | Format endpoint callback | `POST /api/v1/callback/gopay`, lihat `api-contract.md` |
| 6 | Mekanisme authentication | HMAC-SHA256 + toleransi timestamp ±5 menit |
| 7 | Satu atau banyak device | Satu; skema mendukung banyak tanpa perubahan |
| 8 | Cara transaksi dicocokkan | Sub-project 3 |
| 9 | Nominal unik untuk matching | Ya — sub-project 3 |
| 10 | Lama event disimpan di Android | `SENT` 30 hari, `FAILED` permanen |
| 11 | Strategi retry | WorkManager exponential backoff, menyerah setelah ~24 jam |
| 12 | Lama history disimpan | Sama dengan #10 |
| 13 | Tetap jalan setelah restart | Ya, tanpa kode tambahan |
| 14 | Heartbeat dari device | Tidak. `last_seen_at` diperbarui dari request yang memang terjadi |
| 15 | Menangani perubahan format notifikasi | Aturan otoritatif di backend, cukup deploy — tanpa rilis APK |

Seluruh Open Question yang berada dalam cakupan sub-project 1 + 2 kini terjawab. Nomor 8 dan 9 tetap menjadi urusan sub-project 3.

Untuk sub-project 3, pertanyaan penentunya sudah terjawab: **notifikasi merchant tidak memuat nomor referensi transaksi.** Yang tersedia hanya nominal dan nama merchant.

Nominal unik karena itu tetap satu-satunya jalur matching yang praktis. Ditambah fakta bahwa QRIS-nya statis — pembayar mengetik sendiri nominalnya — pendekatan itu memang bisa bekerja, dengan konsekuensi salah ketik harus punya jalur penanganan manual.
