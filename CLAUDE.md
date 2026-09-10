# GoPay Notification Bridge

Bridge notifikasi GoPay dari HP Android ke backend, sebagai fondasi self-hosted payment gateway.

## Dokumen sumber

Urutan kewenangan bila terjadi perbedaan:

1. [`docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md`](docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md) — spec yang disetujui, paling berwenang
2. [`docs/api-contract.md`](docs/api-contract.md) — kontrak antara backend dan Android
3. [`docs/prd.md`](docs/prd.md), [`docs/detail-project.md`](docs/detail-project.md) — dokumen awal

Spec lebih berwenang daripada PRD karena memuat keputusan yang sengaja menyimpang dari PRD dan sudah disetujui. Penyimpangan itu terdaftar di §9 spec — jangan "memperbaiki" implementasi agar kembali sesuai PRD tanpa memeriksa daftar itu lebih dulu.

## QA wajib

**Setiap kali sebuah milestone selesai diimplementasikan, jalankan QA sesuai [`docs/qa/qa-rules.md`](docs/qa/qa-rules.md) dan perbarui [`docs/qa/qa-report.md`](docs/qa/qa-report.md) sebelum menyatakan milestone itu selesai.**

Milestone tanpa laporan QA yang diperbarui belum selesai, sekalipun seluruh kodenya sudah ditulis dan seluruh testnya lulus.

Aturan yang paling mudah dilanggar dan paling penting ditegakkan:

- `PASS` menuntut keluaran perintah yang ditempel, bukan pernyataan bahwa sesuatu berfungsi.
- Hal yang hanya dapat diuji di HP sungguhan ditandai `NEEDS-DEVICE`, tidak pernah `PASS` berdasarkan penalaran.
- Butir requirement yang belum tersentuh ditandai `PENDING`, tidak dihapus dari tabel.

## Pemecahan scope

| # | Sub-project | Status |
|---|---|---|
| 1 | Event ingestion (Go) | sedang dikerjakan |
| 2 | Android bridge (Expo + Kotlin) | sedang dikerjakan |
| 3 | Gateway: invoice, nominal unik, matching, webhook | ditunda |

Jangan membangun apa pun dari sub-project 3 kecuali diminta. Invoice, matching, dan webhook berada di luar cakupan saat ini.

## Keputusan arsitektur yang tidak boleh dilanggar diam-diam

- **Kotlin memiliki pipeline data; TypeScript hanya UI.** JS runtime mati saat aplikasi tertutup, jadi pengiriman HTTP tidak boleh ditulis di TypeScript.
- **Native adalah satu-satunya pemilik database.** TypeScript membaca lewat native module, tidak pernah membuka file DB sendiri.
- **Parsing nominal di HP bersifat display-only.** Backend melakukan ekstraksi otoritatifnya sendiri dari teks mentah.
- **Arah transaksi (uang masuk vs keluar) ditentukan backend, bukan HP.** Salah dalam hal ini berarti order ditandai lunas padahal tidak ada uang masuk.
- **Package GoPay `com.gojek.gopay`** — terkonfirmasi di perangkat, tetapi tetap disimpan sebagai daftar yang dapat diedit, bukan konstanta di kode.
- **Arah transaksi ditentukan allowlist judul, bukan blocklist.** Entri awal: `Transfer masuk`. Yang tidak dikenali ditolak, bukan ditebak.
- **`event_id` dihitung dari `packageName | title | text | when`** — bukan dari `notificationKey` (konstan di GoPay) atau `postTime` (berubah tiap repost).
- **Tanda tangan HMAC dihitung dari byte mentah body**, diverifikasi sebelum decode JSON.

## Perangkat target

ColorOS (OPPO/Realme/OnePlus) — pembunuh background process paling agresif. Setiap keputusan soal ketahanan service harus diuji di sana, tidak boleh diasumsikan dari perilaku Android standar.

Notifikasi transfer masuk GoPay memakai channel **"Promotions and Marketing"**. Jangan pernah menyarankan mematikan channel itu — mematikannya mematikan seluruh sistem tanpa gejala.

## Perintah

Belum ada — `backend/` dan `mobile/` belum dibuat. Bagian ini diisi saat M1 dan M2.
