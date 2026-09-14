# Design Spec — QRIS Statis per Account

**Tanggal:** 2026-09-15
**Cakupan:** Sub-project 1 dari 2 rencana migrasi whuzpay-pg (payment gateway aggregator, `whuzpay-pg/`) dari provider Cashi ke gopay-notifications. Sub-project ini murni di gopay-notifications sendiri (backend + Customer Dashboard) — belum menyentuh `whuzpay-pg/` sama sekali. Sub-project 2 (provider adapter "gopay" + webhook receiver + UI pengaturan provider di `whuzpay-pg/`, plus pencabutan Cashi) dikerjakan setelah ini selesai, spec terpisah.
**Status:** Disetujui, siap masuk implementation plan

---

## 1. Konteks

`whuzpay-pg` (payment gateway aggregator multi-tenant, folder terpisah yang baru dilebur ke repo ini) saat ini merutekan seluruh pembayaran lewat Cashi — provider QRIS sungguhan yang bisa generate QR dinamis untuk nominal berapa pun lewat API, untuk merchant siapa saja, karena Cashi yang punya izin/kontrak ke jaringan QRIS.

gopay-notifications **bukan provider semacam itu**. Tiap `account` di sini merepresentasikan satu HP Android yang benar-benar terpasang GoPay Merchant fisik dengan QRIS statis miliknya sendiri — sistem ini tidak pernah generate QR baru dan tidak pernah memegang uang; ia cuma mendeteksi notifikasi pembayaran masuk lalu mencocokkan nominal unik ke invoice (lihat spec fase 1, `2026-09-12-invoice-nominal-matching-design.md`).

Keputusan dari sesi brainstorming (chat, 2026-09-15):

| Keputusan | Pilihan |
|---|---|
| Model dana | *Bring-your-own-device* — tiap merchant whuzpay-pg yang pilih GoPay wajib punya HP+GoPay Merchant+QRIS sendiri, bikin account gopay-notifications sendiri (signup + pairing HP, dua-duanya sudah ada). Uang tidak pernah singgah di akun vendor gopay-notifications. |
| Siapa upload QRIS | Merchant whuzpay-pg sendiri, lewat Customer Dashboard gopay-notifications miliknya — bukan vendor yang upload-kan manual. |
| Cakupan rollout Cashi (sub-project 2, dicatat di sini biar konteksnya utuh) | Ganti total, Cashi dicabut — bukan coexist. Aman dilakukan karena `whuzpay-pg` belum live/production (data merchant yang ada sekarang cuma seed demo). |
| Pemisahan database | Database `gopay` (gopay-notifications) dan database `whuzpay-pg` **tetap terpisah total** — tidak ada foreign key atau query lintas database. Integrasi murni lewat HTTP API + API key, persis pola integrasi Cashi yang lama. |

Yang tidak ada sebelumnya dan jadi gap: gopay-notifications tidak pernah tahu **bentuk** QRIS-nya, cuma nominalnya. Integrator (whuzpay-pg, atau siapa pun yang integrasi lewat API) harus bisa menampilkan QR itu ke pembeli — sebelum spec ini mereka harus hosting sendiri gambarnya secara terpisah dan menghubungkannya manual ke tiap invoice. Fitur ini berguna untuk SEMUA customer API gopay-notifications, bukan cuma whuzpay-pg.

---

## 2. Model Data

Satu tabel baru, relasi 1:1 murni dengan `accounts` — tidak perlu riwayat/versi, upload baru langsung menimpa yang lama:

```sql
-- Migrasi 00018_account_qris_images.sql

CREATE TABLE account_qris_images (
    account_id   TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    image_data   BYTEA NOT NULL,
    content_type TEXT NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`account_id` sebagai primary key (bukan `id` sendiri + unique index) karena tidak ada kebutuhan mengidentifikasi baris ini selain lewat account-nya.

**Kenapa bytea di Postgres, bukan filesystem/S3:** gambar QRIS kecil (~20-60KB, PNG/JPEG). Backup produksi sekarang cuma `pg_dump` (`backend/deploy/gopay-backup.sh`) — menyimpan di filesystem berarti butuh jalur backup terpisah yang belum ada, dan menambah state yang harus disinkronkan manual antara VPS dan dump database. Konsisten dengan gaya infra proyek ini secara keseluruhan (tanpa S3/Docker/Redis).

**Kenapa tabel baru, bukan kolom baru di `accounts`:** `accounts` punya banyak titik `accountSelectCols`/`scanAccount()` yang dipakai luas di seluruh backend (login, overview, vendor accounts, dll) — menambah kolom bytea di sana berisiko menaikkan biaya query yang tidak butuh gambar ini sama sekali di hampir semua pemanggilan. Tabel terpisah = di-query hanya saat benar-benar dibutuhkan (create invoice, endpoint self-service), pola yang sama dengan `plans`/`api_keys`/`webhook_endpoints`.

---

## 3. Backend

### 3.1 Endpoint self-service (sesi customer, `requireAdmin`)

```
PUT    /api/v1/admin/account/qris-image
GET    /api/v1/admin/account/qris-image
DELETE /api/v1/admin/account/qris-image
```

**`PUT`** — body:

```json
{ "image_base64": "iVBORw0KGgoAAAANSU...", "content_type": "image/png" }
```

Validasi, berurutan:

1. `image_base64` dan `content_type` wajib diisi.
2. Decode base64 — gagal decode → `400 invalid_payload`.
3. Ukuran hasil decode ≤ 300KB (`maxQRISImageBytes`) → lebih besar → `400 image_too_large`.
4. `http.DetectContentType` atas byte hasil decode HARUS `image/png` atau `image/jpeg` — **field `content_type` dari klien tidak pernah dipercaya untuk validasi**, hanya dipakai sebagai default `Content-Type` header saat `GET`. Tidak cocok → `400 unsupported_image_type`.
5. `UPSERT` ke `account_qris_images` (`INSERT ... ON CONFLICT (account_id) DO UPDATE`).
6. Catat ke `account_activity_log` lewat `logActivity` — action baru `qris_image_updated` (menambah daftar 8 jenis aktivitas yang sudah ada di migrasi 00016, jadi 9).

Body decoder endpoint ini pakai konstanta baru `maxQRISImageBodyBytes = 512 << 10` (512 KiB) — **bukan** `maxBodyBytes` (64 KiB) yang dipakai hampir semua endpoint lain di `auth_middleware.go`. Base64 dari gambar 300KB jadi ~400KB, sudah melebihi limit global itu; endpoint-endpoint lain tidak berubah.

**`GET`** — mengembalikan gambar tersimpan langsung sebagai body dengan `Content-Type` dari kolom `content_type` (bukan dibungkus JSON) — dipakai `<img src="/api/v1/admin/account/qris-image">` langsung di Settings untuk preview, tanpa perlu decode base64 di frontend. 404 `not_found` kalau belum pernah upload.

**`DELETE`** — idempotent (200 walau belum pernah ada), catat `qris_image_removed` ke activity log.

### 3.2 `GET /api/v1/admin/account` (profil, sudah ada)

Tambah field `qris_image_configured: bool` — dihitung dari `EXISTS(SELECT 1 FROM account_qris_images WHERE account_id = ...)`. Dipakai Settings tahu status tanpa perlu fetch gambar penuh dulu.

### 3.3 `POST /api/v1/invoices` (API key — perubahan perilaku, WAJIB update API Docs)

`invoiceJSON` dapat field baru:

```json
{
  "id": "inv_...",
  "...": "field lain tidak berubah",
  "qris_image": "data:image/png;base64,iVBORw0KGgo..."
}
```

Kalau account **belum** upload QRIS: invoice **tidak dibuat sama sekali**, ditolak `409 qris_not_configured`, pesan "QRIS belum diatur -- upload di halaman Settings dulu". Ini best-effort gagal cepat: invoice tanpa cara bayar tidak berguna buat integrator, lebih baik gagal jelas di titik pembuatan daripada integrator dapat invoice dengan `qris_image: null` dan baru sadar belakangan.

`GET /api/v1/invoices/{id}` (polling status, sudah ada) **tidak berubah** — tidak perlu ikut mengembalikan gambar tiap polling; integrator sudah dapat itu sekali dari response create.

### 3.4 Store — `internal/store/qris_image.go` (baru)

```go
var ErrQRISImageNotFound = errors.New("store: qris image tidak ditemukan")

type QRISImage struct {
    AccountID   string
    ImageData   []byte
    ContentType string
    UpdatedAt   time.Time
}

func (s *Store) UpsertQRISImage(ctx context.Context, accountID string, data []byte, contentType string) error
func (s *Store) GetQRISImage(ctx context.Context, accountID string) (QRISImage, error) // ErrQRISImageNotFound kalau tidak ada
func (s *Store) DeleteQRISImage(ctx context.Context, accountID string) error            // idempotent, tidak error kalau tidak ada
func (s *Store) HasQRISImage(ctx context.Context, accountID string) (bool, error)
```

---

## 4. Customer Dashboard — Settings

Card baru "QRIS Pembayaran" di `dashboard/src/app/(dashboard)/settings/page.tsx`, pola `<section className="rounded-2xl border border-border/60 p-6 shadow-sm">` yang sama dengan card Password/Telegram yang sudah ada di halaman itu.

Isi:

- Kalau `qris_image_configured` true: tampilkan `<img src="/api/v1/admin/account/qris-image">` (browser otomatis kirim cookie sesi, endpoint ini di belakang `requireAdmin`) + tombol "Ganti" dan "Hapus" (dengan `AlertDialog` konfirmasi untuk Hapus, karena langsung memutus pembuatan invoice baru sampai diupload ulang).
- Kalau belum: input file (`accept="image/png,image/jpeg"`) + teks penjelasan singkat kenapa ini dibutuhkan.
- Validasi klien dulu (ukuran ≤ 300KB, tipe) sebelum encode base64 dan `PUT` — UX cepat, backend tetap validasi ulang penuh (klien tidak pernah dipercaya).
- Toast sukses/gagal, pola sama dengan card lain di halaman ini.

---

## 5. Dokumentasi yang WAJIB ikut diperbarui

- `dashboard/src/app/(dashboard)/api-docs/page.tsx` — field `qris_image` baru di contoh response `POST /invoices`, error code `qris_not_configured` di tabel kode error, catatan bahwa field ini `null`-proof karena request ditolak duluan kalau belum ada QRIS (jadi field ini di dokumentasi selalu terisi, tidak perlu integrator menangani `null`).
- `CLAUDE.md` — paragraf baru di bawah bagian invoice/API existing, mendokumentasikan tabel `account_qris_images`, keputusan bytea-di-Postgres, dan keputusan gagal-cepat `qris_not_configured`.
- `docs/qa/qa-report.md` — laporan QA milestone ini.

---

## 6. Testing

**Store (`internal/store/qris_image_test.go`):**
- Upsert lalu Get mengembalikan data yang sama persis (byte-for-byte).
- Upsert kedua menimpa (bukan menambah baris) — cek lewat query count.
- Get untuk account tanpa gambar → `ErrQRISImageNotFound`.
- Delete idempotent — delete dua kali tidak error.
- `HasQRISImage` benar sebelum/sesudah upsert/delete.
- Delete account (CASCADE) ikut menghapus baris `account_qris_images`-nya.

**HTTP (`internal/httpapi/admin_qris_image_test.go`):**
- `PUT` berhasil dengan PNG 1x1 valid → `GET` mengembalikan byte yang sama, `Content-Type` benar.
- `PUT` dengan base64 tidak valid → `400 invalid_payload`.
- `PUT` dengan ukuran > 300KB (setelah decode) → `400 image_too_large`.
- `PUT` dengan `content_type` yang diklaim `image/png` tapi isinya bukan gambar (mis. teks biasa) → `400 unsupported_image_type` (membuktikan validasi pakai `http.DetectContentType`, bukan field klien).
- `GET`/`PUT`/`DELETE` butuh sesi (`401` tanpa cookie).
- `DELETE` lalu `GET` → `404 not_found`.
- Aktivitas tercatat: `qris_image_updated`/`qris_image_removed` muncul di `GET /api/v1/admin/activity`.

**HTTP (`internal/httpapi/invoices_test.go`, tambahan ke test yang sudah ada):**
- `POST /invoices` untuk account tanpa QRIS → `409 qris_not_configured`, dan invoice **tidak** tersimpan (cek lewat `ListInvoices`/query langsung, bukan cuma dari response).
- `POST /invoices` untuk account dengan QRIS → response memuat `qris_image` berupa data URI yang valid (prefix `data:image/png;base64,` atau `data:image/jpeg;base64,`, dan bagian base64-nya decode balik jadi byte yang sama dengan yang di-upload).
- `GET /invoices/{id}` (polling) tidak memuat `qris_image` (memastikan keputusan §3.3 tidak diam-diam berubah).

---

## 7. Yang sengaja di luar cakupan spec ini

- Provider adapter "gopay" di `whuzpay-pg` (sub-project 2, spec terpisah setelah ini selesai).
- Pencabutan Cashi dari `whuzpay-pg` (bagian dari sub-project 2).
- Validasi bahwa gambar yang diupload benar-benar QRIS EMV yang valid (bukan sekadar gambar apa pun) — di luar cakupan, terlalu berat (butuh parsing EMVCo QR) untuk manfaat yang kecil; kalau merchant upload gambar salah, itu kesalahan operasional mereka sendiri yang cepat ketahuan pas pembeli pertama mencoba scan.
- Riwayat/versi gambar QRIS (rollback ke gambar sebelumnya) — YAGNI, upsert-menimpa sudah cukup untuk kasus penggunaan saat ini.
