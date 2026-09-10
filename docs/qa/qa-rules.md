# Aturan QA

Dokumen ini mengikat. Setiap kali sebuah milestone selesai diimplementasikan, QA dijalankan sesuai aturan di sini dan hasilnya ditulis ke [`qa-report.md`](qa-report.md).

**QA adalah syarat milestone dianggap selesai, bukan langkah opsional.** Milestone tanpa laporan QA yang diperbarui belum selesai, sekalipun seluruh kodenya sudah ditulis dan seluruh testnya lulus.

---

## 1. Kapan QA dijalankan

- Setiap milestone (M1–M6) selesai diimplementasikan.
- Setiap perubahan yang menyentuh kontrak API.
- Setiap perubahan yang menyentuh autentikasi, idempotency, atau logika retry.

Perbaikan kecil yang tidak menyentuh tiga hal di atas tidak memerlukan siklus QA penuh, tetapi tetap wajib menjalankan ulang test otomatis.

---

## 2. Yang diperiksa

QA memeriksa hasil implementasi terhadap **tiga dokumen sumber**, dengan urutan kewenangan bila terjadi perbedaan:

1. [`docs/superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md`](../superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md) — paling berwenang
2. [`docs/api-contract.md`](../api-contract.md)
3. [`docs/prd.md`](../prd.md) dan [`docs/detail-project.md`](../detail-project.md)

Spec lebih berwenang daripada PRD karena spec memuat keputusan yang sengaja menyimpang dari PRD dan sudah disetujui. Penyimpangan itu terdaftar di §9 spec.

Bila ditemukan perbedaan yang **tidak** terdaftar di §9 spec, itu bukan penyimpangan melainkan **cacat** — dan wajib dilaporkan sebagai `FAIL`.

---

## 3. Cakupan tabel requirement

Laporan wajib memuat tabel yang memetakan **setiap** butir berikut ke bukti konkret:

- `FR-01` sampai `FR-10` dari [prd.md §11](../prd.md)
- Seluruh kotak centang [Acceptance Criteria di prd.md §16](../prd.md)
- Seluruh kotak centang [Definition of Done di detail-project.md §42](../detail-project.md)
- Seluruh baris tabel kode → tindakan di [api-contract.md §4.5](../api-contract.md)

Butir yang belum tersentuh milestone berjalan ditandai `PENDING`, **bukan dihilangkan dari tabel**. Yang hilang dari tabel tidak akan pernah teringat lagi.

---

## 4. Status yang boleh dipakai

| Status | Arti | Syarat |
|---|---|---|
| `PASS` | Terbukti bekerja | **Wajib** menyertakan bukti tertempel |
| `FAIL` | Terbukti tidak bekerja | Wajib menyertakan bukti gagal + dugaan penyebab |
| `BLOCKED` | Tertahan hal lain | Wajib menyebutkan apa yang menahan |
| `PENDING` | Di luar cakupan milestone ini | — |
| `NEEDS-DEVICE` | Hanya dapat dijalankan oleh pemilik HP | Wajib menyertakan langkah persis + hasil yang diharapkan |

Tidak ada status lain. Tidak ada "sebagian lulus" — pecah jadi beberapa baris.

---

## 5. Aturan bukti

**`PASS` menuntut bukti yang ditempel, bukan pernyataan.**

Dilarang menandai apa pun `PASS` tanpa menyertakan keluaran perintah yang sebenarnya — nama test yang lulus, potongan output `go test`, output `gradlew test`, response HTTP yang benar-benar diterima. Ini menutup kegagalan yang paling mungkin terjadi: menyimpulkan sesuatu bekerja karena kodenya *terlihat* benar.

Bukti yang sah:

```
go test ./... -run TestHMAC -v
=== RUN   TestHMAC_RejectsModifiedBody
--- PASS: TestHMAC_RejectsModifiedBody (0.00s)
```

Bukti yang **tidak** sah:

- "Sudah diverifikasi berfungsi"
- "Implementasi sesuai spec"
- "Test seharusnya lulus"
- Kutipan kode tanpa hasil eksekusi

---

## 6. `NEEDS-DEVICE`

Sebagian besar hal yang paling mungkin gagal di project ini berada di luar jangkauan otomatisasi: transfer GoPay sungguhan, restart HP, mode pesawat, HP didiamkan semalaman, Notification Access dicabut lalu diberikan lagi, dan perilaku pembunuh proses bawaan OEM.

Butir semacam ini **tidak boleh** ditandai `PASS` berdasarkan penalaran. Tandai `NEEDS-DEVICE` dan sertakan:

1. Langkah persis yang harus dilakukan.
2. Hasil yang diharapkan.
3. Apa yang harus dilaporkan balik.

Setelah pemilik HP menjalankannya dan melaporkan hasil, barulah status diubah menjadi `PASS` atau `FAIL` dengan hasil itu sebagai bukti.

Tanpa pemisahan ini, laporan QA akan terlihat hijau padahal bagian yang paling rawan belum pernah dicoba.

---

## 7. Regresi

Setiap QA menjalankan ulang pemeriksaan seluruh milestone sebelumnya, bukan hanya yang baru. Butir yang sebelumnya `PASS` lalu menjadi `FAIL` ditandai jelas sebagai **regresi**.

---

## 8. Bagian penyimpangan

Laporan wajib memuat satu bagian yang membandingkan implementasi dengan §9 spec:

- Penyimpangan yang **terdaftar dan disengaja** — konfirmasi masih berlaku.
- Penyimpangan **baru yang tidak terdaftar** — laporkan sebagai `FAIL`, lalu putuskan apakah didaftarkan atau diperbaiki.

Tujuannya agar perbedaan antara dokumen dan kenyataan selalu terlihat, dan tidak ada yang mengira itu kelalaian.

---

## 9. Pemeriksaan keamanan wajib

Dijalankan setiap siklus QA, tanpa kecuali, karena endpoint ini menentukan order ditandai lunas:

- [ ] Tidak ada secret ter-hardcode di source code mana pun.
- [ ] Tidak ada secret, token, atau tanda tangan tertulis di log — backend maupun Android.
- [ ] `hmac.Equal` dipakai untuk membandingkan tanda tangan, bukan `==`.
- [ ] Body diverifikasi sebagai byte mentah sebelum di-decode.
- [ ] Toleransi timestamp benar-benar ditegakkan di kedua ujung batas.
- [ ] Idempotency memakai constraint database, bukan `SELECT` lebih dulu.
- [ ] Build production menolak HTTP polos.
- [ ] Mode Discovery default mati dan tidak pernah mengirim apa pun keluar HP.

---

## 10. Format laporan

Satu berkas hidup: [`qa-report.md`](qa-report.md), diperbarui tiap siklus. Riwayat dipegang git, sehingga menumpuk berkas per milestone hanya membuat yang terbaru sulit ditemukan.

Struktur:

```
# QA Report
Milestone terakhir diperiksa · tanggal · ringkasan hitungan status

## 1. Ringkasan
## 2. Butir NEEDS-DEVICE yang menunggu   ← paling atas, ini yang perlu dikerjakan
## 3. Functional Requirements (FR-01..FR-10)
## 4. Acceptance Criteria (prd.md §16)
## 5. Definition of Done (detail-project.md §42)
## 6. Tabel kode → tindakan (api-contract.md §4.5)
## 7. Pemeriksaan keamanan
## 8. Penyimpangan dari spec
## 9. Regresi
## 10. Temuan & tindak lanjut
```

Bagian `NEEDS-DEVICE` sengaja diletakkan dekat atas: itu satu-satunya bagian yang menuntut tindakan dari manusia, dan bagian yang paling mudah terabaikan bila terkubur di bawah.
