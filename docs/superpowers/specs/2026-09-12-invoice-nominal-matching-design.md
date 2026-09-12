# Design Spec — Invoice, Nominal Unik, Matching Engine

**Tanggal:** 2026-09-12
**Cakupan:** Sub-project 3, fase 1 dari 4 (invoice + nominal unik + matching + API key). Webhook delivery (fase 2), pembatasan API key ke pola CLI-vs-dashboard yang lebih halus (sudah diputuskan masuk fase ini, lihat §2), dan konsol pengecualian pembayaran tak cocok (fase 4) **tidak** termasuk di sini.
**Status:** Disetujui, siap masuk implementasi

---

## 1. Konteks & Pemecahan Sub-project 3

Sub-project 3 ("Gateway": invoice, nominal unik, matching, webhook merchant, admin — lihat [ingestion spec §1](2026-09-10-ingestion-and-android-bridge-design.md)) terlalu besar untuk satu spec, persis seperti sub-project 1+2 dulu. Dipecah jadi empat fase:

| # | Fase | Isi | Status |
|---|---|---|---|
| 1 | **Invoice + nominal unik + matching + API key** | Endpoint create/get invoice, alokasi nominal unik, matching otomatis, tabel API key + CRUD-nya (CLI dan dashboard) | **Spec ini** |
| 2 | Webhook delivery | Kirim balik status invoice ke website merchant | Ditunda |
| 3 | ~~API key management~~ | Ditarik maju ke fase 1 (lihat §2) | Selesai lebih awal |
| 4 | Konsol pengecualian | UI untuk pembayaran yang tidak cocok invoice manapun | Ditunda |

Keputusan yang sudah diambil di spec sub-project 1+2 dan tetap berlaku di sini, tidak dibahas ulang:

- **Nominal unik tetap satu-satunya cara matching** — tidak ada nomor referensi transaksi di notifikasi GoPay Merchant, hanya nominal dan waktu ([ingestion spec §9](2026-09-10-ingestion-and-android-bridge-design.md#9-penyimpangan-dari-spec), Open Question #8–9).
- **QRIS statis berarti customer mengetik nominalnya sendiri** — salah ketik adalah risiko nyata yang perlu jalur penanganan manual (fase 4), bukan sesuatu yang bisa "ditebak" sistem.
- **Postgres dipilih sejak sub-project 1 justru untuk kebutuhan ini** — alokasi nominal unik yang aman dari race condition.

---

## 2. Keputusan dari Sesi Brainstorming

Diputuskan bersama Akbar sebelum spec ini ditulis:

| Keputusan | Pilihan |
|---|---|
| Auth endpoint invoice (dipanggil website merchant) | Tabel `api_keys` sungguhan (bukan satu key statis di `.env`) — pengelolaan multi-key ditarik maju dari fase 3 |
| Kelola API key | Dashboard (bukan CLI-only seperti device/admin) |
| Range offset nominal unik | 1–999 (Rp50.000 → Rp50.001..Rp50.999) |
| Masa berlaku invoice | 15 menit tetap (bukan dari env var — cukup sederhana untuk fase ini, bisa dipindah ke setting kalau nanti benar-benar dibutuhkan) |
| Referensi order merchant (`external_ref`) | Wajib diisi saat create invoice |
| Endpoint cek status invoice (sebelum webhook ada) | Ya, `GET /invoices/{id}`, dibangun di fase ini juga |
| Skoping keunikan nominal | **Global**, bukan per-device — invoice tidak perlu tahu device mana yang akan menerima pembayarannya |

---

## 3. Model Data

### 3.1 `invoices`

```sql
CREATE TABLE invoices (
    id                text PRIMARY KEY,       -- "inv_" + hex(16 byte random), pola sama seperti device_id
    external_ref      text NOT NULL,          -- referensi order milik merchant
    requested_amount  bigint NOT NULL,        -- nominal asli, sebelum offset
    unique_amount     bigint NOT NULL,        -- yang harus dibayar customer
    status            text NOT NULL,          -- PENDING | PAID | EXPIRED
    matched_event_id  text NULL REFERENCES notification_events(event_id),
    created_at        timestamptz NOT NULL DEFAULT now(),
    expires_at        timestamptz NOT NULL,   -- created_at + 15 menit
    paid_at           timestamptz NULL
);

CREATE UNIQUE INDEX invoices_external_ref_idx ON invoices (external_ref);
CREATE UNIQUE INDEX invoices_pending_unique_amount_idx
    ON invoices (unique_amount) WHERE status = 'PENDING';
CREATE INDEX invoices_created_at_idx ON invoices (created_at DESC);
```

**`status` adalah kolom nyata, bukan diturunkan seperti `DeviceStatus`.** Ini deviasi sadar dari pola yang sudah ada (`store.StatusOf` menghitung ONLINE/OFFLINE dari `heartbeat_at` tanpa menyimpannya). Alasannya teknis: `invoices_pending_unique_amount_idx` adalah **partial unique index**, dan predicate index di Postgres harus immutable — tidak bisa memakai `now()` di predicate-nya. Karena itu kedaluwarsa harus berupa transisi state yang benar-benar dituliskan, dijalankan lazy (lihat §4.1) tepat sebelum operasi yang butuh kebenaran status terbaru — bukan cron job terpisah, dan bukan pula status yang dihitung ulang tiap kali dibaca.

**Kenapa constraint ini yang mencegah ambiguitas "dua invoice cocok":** selama `invoices_pending_unique_amount_idx` berlaku, dua invoice `PENDING` tidak mungkin berbagi `unique_amount` yang sama. Kasus "sistem harus menolak menebak" dari catatan sub-project 1+2 secara struktural tidak pernah terjadi untuk *alokasi*. Satu-satunya bentuk ketidakcocokan yang tersisa adalah nominal yang **tidak** cocok ke invoice manapun (salah ketik, atau bayar setelah kedaluwarsa) — itu wilayah fase 4, dan di fase ini event seperti itu tetap tersimpan apa adanya di `notification_events`, tidak ditandai apa pun.

### 3.2 `api_keys`

```sql
CREATE TABLE api_keys (
    id          text PRIMARY KEY,   -- "key_" + hex(16 byte random)
    name        text NOT NULL,      -- label bebas, mis. "Website utama"
    key_hash    bytea NOT NULL,     -- SHA-256 dari key mentah
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz NULL
);
```

**SHA-256, bukan bcrypt.** Bcrypt sengaja lambat untuk menahan brute-force password manusia berentropi rendah. API key di sini digenerate backend sendiri lewat `crypto/rand` (entropi tinggi, bukan sesuatu yang bisa ditebak manusia), jadi hash cepat sudah cukup dan tidak memperlambat tiap panggilan API merchant yang bisa jadi frekuensinya tinggi. Key mentah ditampilkan **satu kali** saat dibuat (di response `POST /admin/api-keys` dan sekali lagi di dashboard), lalu hanya hash-nya yang tersimpan — pola yang sama dengan token GitHub/Stripe.

Format ID (`inv_`, `key_` + hex acak) mengikuti pola yang sudah ada di `cmd/devicetool` (`crypto/rand` + `hex.EncodeToString`), bukan ULID atau library baru — konsisten dengan konvensi yang sudah dipakai untuk `device_id`.

---

## 4. Backend

### 4.1 Alokasi nominal unik

`POST /api/v1/invoices`, auth `Authorization: Bearer <api_key>` (lihat §4.3 untuk verifikasi key):

1. **Expire lazy lebih dulu:** `UPDATE invoices SET status='EXPIRED' WHERE status='PENDING' AND expires_at <= now()`. Dijalankan sebelum mencoba alokasi, supaya slot yang sudah kedaluwarsa benar-benar bebas dipakai ulang.
2. Coba offset acak 1–999 di atas `requested_amount`. `INSERT`; kalau kena `invoices_pending_unique_amount_idx` (offset itu sedang dipakai invoice `PENDING` lain untuk `requested_amount` yang sama), ulangi dengan offset baru. Maksimum ~20 percobaan.
3. **Sebelum mengalokasikan apa pun**, cek dulu apakah `external_ref` itu sudah punya invoice: kalau ada dan `requested_amount`-nya **sama**, balas `200` dengan invoice yang sudah ada itu — ini yang membuatnya benar-benar idempotent, retry jaringan dari sisi merchant aman, tidak membuat invoice kedua. Kalau ada tapi `requested_amount`-nya **beda**, itu bukan retry yang aman — balas `409` (conflict sungguhan, `external_ref` yang sama dipakai untuk nominal yang berbeda).
4. Kalau 20 percobaan alokasi tetap gagal (nominal dasar itu sedang sangat padat), balas `503` — bukan dipaksakan dengan offset dobel, sesuai prinsip "jangan menebak" yang sama dipegang di seluruh sistem ini.

### 4.2 Matching

Dipanggil dari `handleCallback`, **setelah** `InsertEvent` sukses menyimpan event baru — bukan proses/worker terpisah, karena matching hanya relevan tepat setelah event itu ada dan kita sudah berada di request handler yang sama.

```go
// Sama seperti langkah 1 alokasi — expire lazy dulu.
UPDATE invoices SET status='EXPIRED' WHERE status='PENDING' AND expires_at <= now()

// Atomik: satu UPDATE...RETURNING, bukan SELECT lalu UPDATE terpisah.
// Pola yang sama dengan ON CONFLICT DO NOTHING di InsertEvent — race dua
// event dengan amount_hint yang sama, nyaris bersamaan, tidak keduanya
// bisa "menang".
UPDATE invoices
   SET status='PAID', matched_event_id=$1, paid_at=$2
 WHERE status='PENDING' AND unique_amount=$3
RETURNING id
```

Dilewati sepenuhnya bila `amount_hint` event itu `nil` (notifikasi tanpa nominal yang berhasil di-parse) — event tetap tersimpan seperti biasa, tanpa upaya matching.

### 4.3 Endpoint

Prefix `/api/v1`, terpisah dari HMAC device dan cookie admin:

```
POST /invoices
  Header: Authorization: Bearer <api_key>
  Body:   { "external_ref": "ORDER-123", "amount": 50000 }
  201:    { "id": "inv_...", "external_ref": "ORDER-123",
            "requested_amount": 50000, "unique_amount": 50317,
            "status": "PENDING", "expires_at": "2026-09-12T15:15:00Z" }
  200:    external_ref sudah pernah dipakai dengan amount yang SAMA — invoice
          yang sudah ada dikembalikan apa adanya (retry aman, bukan error)
  401:    key tidak ada / salah / sudah dicabut
  400:    external_ref kosong, amount <= 0
  409:    external_ref sudah dipakai invoice lain dengan amount yang BEDA
  503:    ruang nominal unik untuk amount ini penuh

GET /invoices/{id}
  Header: Authorization: Bearer <api_key>
  200:    bentuk sama seperti response create, plus matched_event_id/paid_at bila PAID
  401:    key tidak valid
  404:    id tidak ditemukan
```

Verifikasi API key: SHA-256 raw key dari header, `SELECT ... WHERE key_hash = $1 AND revoked_at IS NULL`. Tanggapan untuk key salah dan key valid-tapi-dicabut **disamakan** (`401`, pesan generik) — pola yang sama dengan username-tidak-ditemukan vs password-salah di login admin, supaya tidak membocorkan informasi lewat perbedaan respons.

**Endpoint admin** (dashboard, `requireAdmin`, prefix `/api/v1/admin`):

```
GET /admin/invoices?limit=&offset=&status=&q=&from=&to=
  q cocok sebagian ke external_ref, case-insensitive — pola sama dengan
  filter events (source/q/from/to) yang sudah ada di handleEvents.

POST /admin/api-keys
  Body: { "name": "Website utama" }
  201:  { "id": "key_...", "name": "...", "key": "<mentah, hanya sekali ini>" }

GET /admin/api-keys
  200:  [{ "id", "name", "created_at", "revoked_at" }, ...]  -- tanpa key_hash/key mentah

PATCH /admin/api-keys/{id}
  Tanpa body — selalu mencabut, tidak ada jalan mengaktifkan key yang sudah
  dicabut (kalau butuh key baru, buat yang baru; bukan menghidupkan yang lama).
  200:  revoked_at diisi now() (idempotent — mencabut yang sudah dicabut tetap 200)
  404:  id tidak ditemukan
```

---

## 5. Dashboard

### 5.1 Transactions

Mengganti status "Segera" di sidebar. Mengikuti pola Events yang sudah ada (bukan halaman detail terpisah seperti `/transactions/:id` di `dashboard-spec.md` §19 — ditunda demi konsistensi dan MVP, baris tabel diperluas seperti Events sudah cukup):

- Filter: pencarian (`external_ref`), status (`FilterDropdown`: Pending/Paid/Expired), rentang tanggal (`DateRangeFilter`) — komponen yang sama persis dengan Events.
- Kolom: Referensi, Nominal diminta, Nominal unik, Status (badge warna per status), Dibuat, Dibayar/Kedaluwarsa.
- Baris diperluas: `id` invoice dan `matched_event_id` (tautan sederhana ke info event bila ada).

### 5.2 API Keys

Mengganti status "Segera" di sidebar:

- Daftar: nama, dibuat kapan, status (Aktif/Dicabut). **Tidak pernah** menampilkan key mentah atau hash.
- "Buat key baru" → dialog menampilkan key mentah **satu kali**, tombol salin, peringatan eksplisit "tidak akan ditampilkan lagi setelah ini ditutup".
- "Cabut" per key → `AlertDialog` konfirmasi, pola yang sama dengan toggle aktif/nonaktif di Devices.

---

## 6. Testing

Ditulis di fase implementasi, mengikuti `docs/qa/qa-rules.md` — status `PENDING` sampai `make test` sungguhan dijalankan Akbar (butuh Postgres via `docker compose`, di luar batas kerja Claude).

**`store`:**
- Alokasi nominal unik berhasil; retry saat offset pertama bertabrakan; gagal (503-worthy) setelah percobaan habis pada nominal yang sangat padat.
- Lazy-expire: invoice yang lewat `expires_at` tidak lagi menghalangi alokasi offset yang sama.
- `external_ref` dobel dengan `requested_amount` sama → invoice lama dikembalikan, bukan invoice baru dibuat; `external_ref` dobel dengan `requested_amount` beda → ditolak (unique constraint jadi jalur terakhir, tapi pengecekan aplikasi di §4.1 langkah 3 seharusnya menangkapnya lebih dulu).
- Matching: `amount_hint` cocok → invoice `PAID` dengan `matched_event_id` benar; tidak cocok → tidak ada perubahan; `amount_hint` nil → dilewati; race dua event dengan `amount_hint` sama diuji dengan goroutine + `-race`, hanya satu yang menang (pola yang sama dengan `TestInsertEventIdempotentSaatBersamaan`).
- API key: hash tersimpan bukan plaintext; verifikasi key benar/salah/dicabut.

**`httpapi`:**
- `POST /invoices`: berhasil (201); key tidak ada/salah/dicabut → 401 (pesan disamakan); `external_ref` dobel dengan amount sama → 200 invoice lama, bukan invoice baru; `external_ref` dobel dengan amount beda → 409; `amount` <= 0 → 400.
- `GET /invoices/{id}`: ditemukan, tidak ditemukan, key tidak valid.
- `GET /admin/invoices`: filter status/q/rentang tanggal, auth admin.
- `POST/GET/PATCH /admin/api-keys`: create mengembalikan key mentah sekali, list tidak pernah membocorkannya, revoke idempotent.

**Dashboard:** `tsc --noEmit`, `eslint .`, `next build` seperti biasa (dijalankan Claude sendiri). Interaksi sungguhan halaman Transactions/API Keys di browser tetap `NEEDS-DEVICE`.

---

## 7. Yang Sengaja Tidak Ada di Fase Ini

- **Webhook delivery** (fase 2) — merchant belum dapat notifikasi push saat invoice `PAID`, hanya lewat `GET /invoices/{id}` (polling).
- **Konsol pengecualian** (fase 4) — event dengan nominal yang tidak cocok invoice manapun tidak ditandai atau ditampilkan secara khusus; tetap baris biasa di `notification_events`/halaman Events.
- **Halaman detail invoice terpisah** (`/transactions/:id`) — baris tabel yang diperluas sudah cukup untuk MVP.
- **Refund / status `FAILED`** yang disebut `dashboard-spec.md` §18.2 — di luar cakupan; hanya `PENDING`/`PAID`/`EXPIRED`.
- **Masa berlaku invoice yang bisa diatur** (env var/setting) — 15 menit tetap di kode untuk fase ini.
