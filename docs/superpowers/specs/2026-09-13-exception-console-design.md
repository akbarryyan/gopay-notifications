# Design Spec — Konsol Pengecualian

**Tanggal:** 2026-09-13
**Cakupan:** Sub-project 3, fase 4 dari 4 (terakhir). Fase 1 (invoice + nominal unik + matching + API key) dan fase 2 (webhook delivery) sudah selesai.
**Status:** Disetujui, siap masuk implementasi

---

## 1. Konteks

Sejak fase 1, event dengan nominal yang tidak cocok invoice manapun (salah ketik nominal oleh customer, atau bayar setelah invoice kedaluwarsa) tetap tersimpan sebagai baris biasa di `notification_events` — tidak ditandai atau ditampilkan secara khusus (lihat [fase 1 §1](2026-09-12-invoice-nominal-matching-design.md)). Ini masalah nyata: uang **sudah masuk** ke akun GoPay merchant, tapi sistem tidak tahu itu bayar untuk order yang mana. Fase ini membangun tempat admin merekonsiliasi kasus itu secara manual.

Keputusan dari sesi brainstorming:

| Keputusan | Pilihan |
|---|---|
| Fokus konsol | Event yang **masuk** tapi tak cocok invoice manapun — bukan invoice yang expired tanpa dibayar (itu sudah kelihatan di Transactions, dan tidak ada uang yang perlu direkonsiliasi di situ) |
| Aksi admin | Cocokkan manual ke invoice tertentu, atau abaikan dengan catatan opsional — dua ini saja |
| Invoice yang bisa jadi target cocok manual | `PENDING` dan `EXPIRED` (customer bayar telat adalah kasus paling umum yang justru butuh konsol ini) — bukan `PAID` |

---

## 2. Model Data

**Tidak perlu tabel status untuk "exception"** — daftar dihitung lewat query, bukan disimpan. Sebuah event dianggap exception selama tiga syarat sekaligus: `amount_hint` terisi, tidak direferensikan invoice manapun sebagai `matched_event_id`, dan belum pernah di-dismiss. Begitu event itu cocok (otomatis atau manual), ia otomatis hilang dari daftar karena sudah muncul sebagai `matched_event_id` sebuah invoice — tidak ada state ganda yang bisa tidak sinkron.

**Satu tabel baru** — cuma untuk keputusan "diabaikan", satu-satunya hal di sini yang benar-benar keputusan manual, tidak bisa diturunkan dari data lain:

```sql
CREATE TABLE event_reviews (
    event_id     TEXT PRIMARY KEY REFERENCES notification_events(event_id),
    dismissed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    note         TEXT NULL
);
```

Tidak ada "undo" — begitu di-dismiss, tetap di-dismiss (konsisten dengan API key/webhook: tidak ada un-revoke). `event_id` sebagai primary key otomatis menolak percobaan dismiss dua kali untuk event yang sama.

**Perbaikan yang wajib masuk fase ini:** `invoices.matched_event_id` dari migrasi fase 1 tidak punya unique constraint. Itu aman selama jalur cocok hanya otomatis (nominal memang unik), tapi begitu ada jalur **manual**, admin bisa saja keliru mencocokkan event yang sama ke dua invoice berbeda — satu pembayaran dobel terhitung. Ditambahkan:

```sql
CREATE UNIQUE INDEX invoices_matched_event_id_idx
    ON invoices (matched_event_id) WHERE matched_event_id IS NOT NULL;
```

---

## 3. Backend

### 3.1 Daftar exception

```sql
SELECT e.event_id, e.device_id, e.source, e.package_name, e.title, e.body_text,
       e.big_text, e.amount_hint, e.posted_at, e.received_at, e.raw_payload
FROM notification_events e
WHERE e.amount_hint IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM invoices i WHERE i.matched_event_id = e.event_id)
  AND NOT EXISTS (SELECT 1 FROM event_reviews r WHERE r.event_id = e.event_id)
  -- + filter q (device_id/title) dan rentang tanggal received_at, pola sama
  --   dengan store.EventFilter yang sudah ada
ORDER BY e.received_at DESC
LIMIT $1 OFFSET $2
```

Bentuk baris identik dengan `eventJSON` yang sudah ada (`GET /admin/events`) — tidak perlu tipe respons baru di sisi API.

### 3.2 Cocokkan manual

Atomik, pola yang sama dengan `MatchEvent` (fase 1), tapi dikunci oleh `id` invoice pilihan admin, bukan oleh nominal:

```sql
UPDATE invoices SET status = 'PAID', matched_event_id = $1, paid_at = $2
WHERE id = $3 AND status IN ('PENDING', 'EXPIRED')
RETURNING id
```

- Tidak ketemu baris → invoice itu sudah `PAID` (mungkin baru saja dicocokkan otomatis oleh event lain, atau oleh admin lain) → `409`.
- Kena `invoices_matched_event_id_idx` → event ini sudah dipakai invoice lain → `409` dengan pesan berbeda.
- Berhasil → **`triggerInvoiceWebhook(store.WebhookEventInvoicePaid, invoiceID)`** dipanggil — helper yang sama persis dengan yang dipakai `handleCallback` di fase 2, bukan logic baru. Dari sudut pandang merchant, tidak ada beda antara invoice yang cocok otomatis dan yang dicocokkan admin — keduanya memicu webhook `invoice.paid` yang sama.

### 3.3 Abaikan

```sql
INSERT INTO event_reviews (event_id, note) VALUES ($1, $2)
```

Kena `event_id` primary key (sudah pernah di-dismiss) → `409`.

Tidak ada pengecekan silang terhadap `invoices` sebelum dismiss — kalau event
itu sudah keburu cocok (mis. dua admin bertindak nyaris bersamaan, satu
mencocokkan, satu meng-abaikan), baris `event_reviews` tetap tertulis tapi
tidak berpengaruh: query daftar exception (§3.1) sudah mengecualikannya lewat
kondisi "tidak direferensikan invoice manapun" duluan. Baris dismiss yang
jadi berlebih itu tidak berbahaya, cuma tidak pernah dibaca lagi.

### 3.4 Endpoint

Seluruhnya `requireAdmin`:

```
GET /admin/exceptions?limit=&offset=&q=&from=&to=
  200: { exceptions: [eventJSON, ...] }

POST /admin/exceptions/{eventID}/match
  Body: { "invoice_id": "inv_..." }
  200: { success: true }
  404: event atau invoice tidak ditemukan
  409: invoice sudah PAID, atau event sudah dipakai invoice lain

POST /admin/exceptions/{eventID}/dismiss
  Body: { "note": "..." }  -- opsional
  200: { success: true }
  409: event sudah pernah di-dismiss
```

`GET /admin/invoices` (fase 1) diperlonggar: `status` menerima banyak nilai dipisah koma (`status=PENDING,EXPIRED`), dipakai dialog pencocokan manual di dashboard untuk mencari invoice target dalam satu panggilan, bukan dua.

---

## 4. Dashboard

**Exceptions** (halaman baru, sidebar masuk grup "Gateway" — jadi 4 item: Transactions, API Keys, Webhooks, Exceptions):

- Tabel: Waktu diterima, Device, Judul notifikasi, Nominal, Aksi. Filter pencarian + rentang tanggal, pola sama dengan Events.
- Per baris: tombol **Cocokkan** (dialog dengan kotak pencarian invoice — memanggil `GET /admin/invoices?status=PENDING,EXPIRED&q=...`, pilih satu baris hasil pencarian, konfirmasi) dan **Abaikan** (dialog kecil dengan kotak catatan opsional, konfirmasi).
- Berhasil salah satu aksi → toast sukses, baris hilang dari daftar (reload).
- Kosong: "Tidak ada pembayaran yang perlu direkonsiliasi" — kondisi normal justru daftar ini kosong, bukan pesan alarm.

---

## 5. Testing

Ditulis di fase implementasi, status `PENDING` sampai `make test` sungguhan dijalankan Akbar.

**`store`:**
- `ListExceptions`: event dengan `amount_hint` cocok invoice manapun tidak muncul; yang sudah di-dismiss tidak muncul; `amount_hint` nil tidak pernah muncul; filter q/rentang tanggal.
- `ManualMatchEvent`: berhasil ke invoice `PENDING`; berhasil ke invoice `EXPIRED`; gagal ke invoice yang sudah `PAID`; gagal kalau event sudah dipakai invoice lain (constraint baru); race dua pencocokan manual ke invoice berbeda untuk event yang sama, hanya satu menang (`-race`).
- `DismissEvent`: sukses; dismiss dua kali untuk event yang sama ditolak (primary key).

**`httpapi`:**
- `GET /admin/exceptions`: filter q/tanggal, perlu sesi.
- `POST .../match`: berhasil (invoice PAID + webhook invoice.paid terkirim — integrasi ujung-ke-ujung, event masuk tak cocok → muncul di exceptions → dicocokkan → menghilang dari exceptions), invoice tidak ditemukan (404), invoice sudah PAID (409), event sudah dipakai invoice lain (409).
- `POST .../dismiss`: berhasil, event tidak ditemukan (404), sudah pernah di-dismiss (409).
- `GET /admin/invoices?status=PENDING,EXPIRED`: mengembalikan gabungan kedua status dalam satu panggilan.

**Dashboard:** `tsc --noEmit`, `eslint .`, `next build`; interaksi sungguhan di browser `NEEDS-DEVICE`.

---

## 6. Yang Sengaja Tidak Ada di Fase Ini

- **Undo dismiss** — konsisten dengan pola "tidak ada un-revoke" di seluruh sistem ini.
- **Invoice `EXPIRED` yang tidak pernah dibayar** sebagai bagian dari konsol ini — sudah terlihat di Transactions (filter Status=Expired), tidak ada uang yang perlu direkonsiliasi.
- **Notifikasi/alert proaktif** ("ada N exception baru") — konsol ini pasif, admin membukanya sendiri; kalau nanti dibutuhkan, itu perluasan terpisah.
