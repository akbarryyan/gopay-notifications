# Design Spec — Webhook Delivery

**Tanggal:** 2026-09-13
**Cakupan:** Sub-project 3, fase 2 dari 4 (webhook delivery ke website merchant). Fase 1 (invoice + nominal unik + matching + API key) sudah selesai — lihat [2026-09-12-invoice-nominal-matching-design.md](2026-09-12-invoice-nominal-matching-design.md). Fase 4 (konsol pengecualian) **tidak** termasuk di sini.
**Status:** Disetujui, siap masuk implementasi

---

## 1. Konteks

Sejak fase 1, merchant hanya bisa tahu status invoice lewat polling `GET /invoices/{id}`. Fase ini menambahkan jalur push: begitu invoice berubah status (`PAID` atau `EXPIRED`), backend mengirim `POST` ke endpoint yang didaftarkan merchant lewat dashboard, ditandatangani supaya merchant bisa memverifikasi itu benar dari kita.

Keputusan dari sesi brainstorming:

| Keputusan | Pilihan |
|---|---|
| Jumlah endpoint per instalasi | Banyak, tabel `webhook_endpoints` (mirip `api_keys`) — merchant bisa punya "Production" + "Staging" sekaligus |
| Event yang dikirim | `invoice.paid` dan `invoice.expired` — bukan 4 event di `dashboard-spec.md` §23, karena hanya dua ini yang benar-benar ada di model data kita |
| Mekanisme aktif (deteksi expired + retry) | Satu goroutine `time.Ticker` di proses `cmd/server` yang sama, interval ~1 menit — bukan proses/worker terpisah |
| Retry | Maks 5 percobaan, backoff 1→2→4→8→16 menit, lalu `FAILED` permanen |
| Signing | HMAC-SHA256 hex di header `X-Webhook-Signature`, atas raw body — pola yang sama dengan HMAC device |
| Tombol Test | Ya — payload event `test`, tidak menyentuh tabel `invoices` sungguhan |
| Edit endpoint | Tidak ada — ganti URL/events berarti hapus lalu buat baru, konsisten dengan API key (tidak ada un-revoke) |
| Halaman detail deliveries | Baris diperluas di tabel Webhooks (pola yang sama dengan Events/Transactions), bukan route `/webhooks/:id/deliveries` terpisah seperti `dashboard-spec.md` §24 |

---

## 2. Model Data

### 2.1 `webhook_endpoints`

```sql
CREATE TABLE webhook_endpoints (
    id                text PRIMARY KEY,      -- "wh_" + hex(16 byte random)
    name              text NOT NULL,
    url               text NOT NULL,
    secret_encrypted  bytea NOT NULL,        -- lihat §2.3 — dienkripsi, BUKAN di-hash
    events            text[] NOT NULL,       -- subset {'invoice.paid','invoice.expired'}
    enabled           boolean NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now()
);
```

### 2.2 `webhook_deliveries`

```sql
CREATE TABLE webhook_deliveries (
    id               text PRIMARY KEY,     -- "whd_" + hex(16 byte random)
    endpoint_id      text NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event            text NOT NULL,        -- 'invoice.paid' | 'invoice.expired' | 'test'
    invoice_id       text NULL REFERENCES invoices(id),  -- NULL untuk event 'test'
    payload          jsonb NOT NULL,
    status           text NOT NULL DEFAULT 'PENDING',    -- PENDING|RETRYING|DELIVERED|FAILED
    attempt          int NOT NULL DEFAULT 0,
    next_attempt_at  timestamptz NULL,
    http_status      int NULL,
    duration_ms      int NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    delivered_at     timestamptz NULL
);

CREATE INDEX webhook_deliveries_due_idx
    ON webhook_deliveries (next_attempt_at)
    WHERE status IN ('PENDING', 'RETRYING');
```

### 2.3 Kenapa `secret_encrypted`, bukan hash seperti API key

API key (fase 1) di-hash SHA-256 satu-arah karena sistem hanya perlu **membandingkan** — merchant yang menyimpan key mentahnya, kita cukup verifikasi. Webhook secret sebaliknya: kitalah yang harus **memakainya lagi**, setiap kali menandatangani payload keluar ke merchant. Hash satu-arah tidak bisa dibalik, jadi tidak berlaku di sini.

Dienkripsi (bukan disimpan polos) memakai `internal/secretbox` yang sudah dipakai untuk secret device — kunci enkripsi baru, `WEBHOOK_SECRET_KEY`, terpisah dari `DEVICE_SECRET_KEY` dan `ADMIN_SESSION_KEY` (pola yang sama: tiga kunci berbeda untuk tiga tujuan berbeda, jangan dipakai ulang). Ditampilkan ke pengguna **hanya sekali**, tepat setelah dibuat, sama seperti API key — bedanya cuma di penyimpanan internal (dienkripsi vs di-hash), bukan di perilaku yang terlihat pengguna.

---

## 3. Backend

### 3.1 Pengiriman

```go
func (a *API) sendWebhook(ctx context.Context, d store.WebhookDelivery, endpoint store.WebhookEndpoint) store.DeliveryResult
```

- Body: JSON `{"event": ..., "invoice": <invoiceJSON atau nil untuk event test>, "sent_at": ...}`.
- Header `X-Webhook-Event: <event>`, `X-Webhook-Signature: <hex(HMAC-SHA256(secret, raw body))>`.
- Timeout klien HTTP 10 detik. Status 2xx = sukses; selain itu (termasuk timeout/connection refused) = gagal.
- Hasil (status HTTP, durasi, sukses/gagal) dikembalikan ke pemanggil untuk dituliskan ke `webhook_deliveries` — fungsi ini sendiri tidak menyentuh database, supaya bisa diuji terpisah dari logic retry/state.

### 3.2 Trigger `invoice.paid`

Di `handleCallback`, tepat setelah `MatchEvent` mengembalikan `matched = true`:

```go
if matched {
    a.enqueueWebhooks(r.Context(), "invoice.paid", invoiceID)
}
```

`enqueueWebhooks` menyisipkan satu baris `webhook_deliveries` per `webhook_endpoints` yang `enabled` dan `events` mengandung event itu — status `PENDING`, `next_attempt_at = now()` (diisi eksplisit saat insert, bukan dibiarkan NULL, supaya baris itu langsung "jatuh tempo" tanpa menunggu putaran worker berikutnya). Lalu percobaan pertama dijalankan **di goroutine terpisah** lewat `a.attemptDelivery(ctx, delivery, endpoint, now)` — helper yang sama juga dipakai worker berkala (§3.3 langkah 4) untuk retry, supaya logic "kirim lalu tuliskan hasilnya" tidak terduplikasi. Ini supaya respons ke device (`callbackResponse`) tidak menunggu server merchant yang mungkin lambat/timeout — kegagalan pengiriman pertama bukan kegagalan bagi device; device sudah menerima `200 OK` untuk event-nya sendiri terlepas dari nasib webhook.

### 3.3 Worker berkala

```go
// ProcessDueWebhooks satu putaran: expire invoice yang lewat waktu (memicu
// invoice.expired), lalu proses seluruh delivery yang next_attempt_at-nya
// sudah lewat. Dipanggil manual dengan now palsu di test, dipanggil oleh
// time.Ticker di cmd/server untuk yang sungguhan.
func (a *API) ProcessDueWebhooks(ctx context.Context, now time.Time) error
```

Langkah:
1. `store.ExpireInvoicesAndListNewlyExpired(ctx, now)` — varian `expireStaleInvoices` (fase 1) yang **mengembalikan** ID invoice yang baru saja berpindah ke `EXPIRED` di panggilan ini (bukan yang sudah `EXPIRED` dari sebelumnya) lewat `UPDATE ... RETURNING id`, supaya `invoice.expired` terkirim tepat sekali per invoice, bukan berulang tiap worker jalan.
2. Untuk tiap ID itu, `enqueueWebhooks(ctx, "invoice.expired", id)`.
3. `store.DueWebhookDeliveries(ctx, now)` — baris `PENDING`/`RETRYING` dengan `next_attempt_at <= now` (`PENDING` yang belum pernah dicoba punya `next_attempt_at = created_at`, jadi langsung due).
4. Untuk tiap baris, `attemptDelivery` mengirim lalu menuliskan hasilnya: sukses → `DELIVERED` + `delivered_at`; gagal & `attempt < 5` → `RETRYING`, `attempt += 1`, `next_attempt_at = now + backoff[attempt]` (`backoff = [1, 2, 4, 8, 16] menit`); gagal & `attempt = 5` → `FAILED`, `next_attempt_at = NULL` (tidak diambil `DueWebhookDeliveries` lagi — predicate index di §2.2 tetap match karena masih memfilter dari `status`, tapi `next_attempt_at <= now` di query tidak akan pernah true untuk NULL).

`cmd/server` membungkus ini dengan `time.NewTicker(1 * time.Minute)` di goroutine yang berhenti bersih saat context dibatalkan (pola yang sama dengan graceful shutdown yang sudah ada).

### 3.4 Endpoint

Seluruhnya `requireAdmin`, dashboard-only:

```
POST /admin/webhooks
  Body:  { name, url, events: ["invoice.paid", "invoice.expired"] }
  201:   { id, name, url, events, enabled, created_at, secret }  -- secret mentah, sekali ini

GET /admin/webhooks
  200:   [{ id, name, url, events, enabled, created_at, last_delivery_at, last_delivery_status }]
         -- tanpa secret; dua field terakhir dihitung dari webhook_deliveries
            (MAX(created_at) + status pada baris itu) untuk ringkasan tabel

PATCH /admin/webhooks/{id}
  Body:  { enabled: bool }
  200:   sukses

DELETE /admin/webhooks/{id}
  200:   sukses (CASCADE menghapus riwayat deliveries-nya juga)

POST /admin/webhooks/{id}/test
  200:   { delivered: bool, http_status, duration_ms }  -- hasil percobaan test seketika,
         BUKAN 202/async — merchant/admin ingin tahu hasilnya langsung

GET /admin/webhooks/{id}/deliveries?limit=&offset=
  200:   { deliveries: [...] }  -- riwayat, terbaru dulu
```

`POST .../test` sengaja sinkron (menunggu hasil, bukan enqueue-lalu-lapor-nanti) — bedanya dengan `invoice.paid` yang sengaja async: di sini pengguna sedang menonton dashboard menunggu jawaban "endpoint-nya benar atau tidak", bukan device yang harus segera dapat respons.

---

## 4. Dashboard

**Webhooks** (mengganti status "Segera" di sidebar, grup "Gateway" — jadi tiga item: Transactions, API Keys, Webhooks):

- Tabel: Nama, URL, Events (badge kecil per event yang dilanggan), Status (Aktif/Nonaktif), Percobaan terakhir (waktu + hasil).
- "Buat webhook baru": Nama, URL, checkbox Events (default keduanya tercentang) → setelah submit, dialog menampilkan secret mentah sekali (pola sama dengan API Keys).
- Per baris: toggle Aktif/Nonaktif (`AlertDialog` konfirmasi seperti Devices), tombol **Test** (toast sukses/gagal + kode HTTP), tombol **Hapus** (`AlertDialog` konfirmasi, memperingatkan riwayat deliveries ikut terhapus).
- Klik baris → diperluas menampilkan riwayat pengiriman (waktu, event, status, kode HTTP, durasi) lewat `GET /admin/webhooks/{id}/deliveries`, pola yang sama dengan expand-row di Events/Transactions.

---

## 5. Testing

Ditulis di fase implementasi, status `PENDING` sampai `make test` sungguhan dijalankan Akbar (butuh Postgres, di luar batas kerja Claude).

**`store`:**
- Enkripsi/dekripsi secret webhook lewat `secretbox` — bukan hash, dites terpisah dari API key.
- `CreateWebhookEndpoint`, `ListWebhookEndpoints` (tidak membocorkan secret), enable/disable, delete (cascade ke deliveries).
- `ExpireInvoicesAndListNewlyExpired`: hanya mengembalikan ID yang baru berpindah PADA PANGGILAN INI, bukan yang sudah EXPIRED dari sebelumnya — dipanggil dua kali berturut-turut pada invoice yang sama harus mengembalikan ID itu di panggilan pertama saja.
- `DueWebhookDeliveries`: mengambil PENDING/RETRYING yang jatuh tempo, tidak mengambil yang FAILED atau yang `next_attempt_at` di masa depan.
- Backoff terhitung benar per percobaan; menyerah tepat di percobaan ke-5 dengan status FAILED permanen.

**`httpapi`:**
- `sendWebhook`: memakai `httptest.Server` sebagai "server merchant" palsu — signature yang dihitung ulang di sisi test harus cocok; server palsu balas 500 lalu 200 pada percobaan berikut → `ProcessDueWebhooks` dua putaran berturut menghasilkan status akhir DELIVERED.
- `invoice.paid` terpicu otomatis dari `handleCallback` tanpa langkah tambahan (pola integrasi ujung-ke-ujung yang sama seperti fase 1).
- `invoice.expired` terpicu oleh `ProcessDueWebhooks` dengan `now` yang dilewati masa berlaku invoice, tepat sekali (panggilan `ProcessDueWebhooks` kedua tidak mengirim ulang).
- `POST .../test` tidak pernah menulis baris ke tabel `invoices`, hanya ke `webhook_deliveries` dengan `invoice_id = NULL`.
- Seluruh endpoint `/admin/webhooks*` menolak tanpa sesi.

**Dashboard:** `tsc --noEmit`, `eslint .`, `next build` seperti biasa (Claude); interaksi sungguhan `NEEDS-DEVICE`.

---

## 6. Yang Sengaja Tidak Ada di Fase Ini

- **Event `payment.received`/`payment.failed`/`payment.updated`** dari `dashboard-spec.md` §23 — tidak ada kejadian nyata yang bersesuaian di model data kita saat ini.
- **Edit endpoint** (ganti URL/events/nama) — hapus lalu buat baru.
- **Halaman detail deliveries terpisah** (`/webhooks/:id/deliveries`) — baris diperluas di tabel sudah cukup.
- **Konsol pengecualian** (fase 4) — tetap terpisah, tidak digabung ke sini.
