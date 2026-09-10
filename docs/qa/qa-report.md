# QA Report

**Milestone terakhir diperiksa:** M0 — Repo, kontrak API, aturan QA
**Tanggal:** 2026-09-10
**Ringkasan:** `PASS` 0 · `FAIL` 0 · `BLOCKED` 0 · `NEEDS-DEVICE` 0 · `PENDING` 62

---

## 1. Ringkasan

Ini **baseline**. Belum ada satu baris kode implementasi pun — M0 hanya menghasilkan dokumen. Seluruh butir requirement sengaja didaftarkan sekarang dalam status `PENDING` agar tidak ada yang terlupa saat milestone berjalan.

Laporan ini diperbarui setiap milestone selesai, sesuai [`qa-rules.md`](qa-rules.md).

---

## 2. Butir `NEEDS-DEVICE` yang menunggu

Belum ada. Bagian ini akan terisi mulai M2, dan merupakan bagian yang menuntut tindakan langsung dari pemilik HP.

---

## 3. Functional Requirements — [prd.md §11](../prd.md)

| # | Requirement | Milestone | Status | Bukti |
|---|---|---|---|---|
| FR-01 | Notification Access terdeteksi | M2 | `PENDING` | — |
| FR-02 | Notification Listener menerima event | M2 | `PENDING` | — |
| FR-03 | Hanya memproses event GoPay | M3 | `PENDING` | — |
| FR-04 | Parsing jadi event terstruktur | M3 | `PENDING` | — |
| FR-05 | Kirim event via HTTPS | M4 | `PENDING` | — |
| FR-06 | Autentikasi request | M1 + M4 | `PENDING` | — |
| FR-07 | Retry untuk error yang dapat dipulihkan | M4 | `PENDING` | — |
| FR-08 | Pencegahan duplikat / idempotency | M1 + M3 | `PENDING` | — |
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
| Backend mengidentifikasi perangkat pengirim | M1 | `PENDING` | — |
| Event sama tidak diproses dua kali | M1 | `PENDING` | — |
| Event terkirim setelah koneksi normal kembali | M4 | `PENDING` | — |
| User melihat status listener | M5 | `PENDING` | — |
| User melihat status komunikasi backend | M5 | `PENDING` | — |
| User melihat event/history dasar | M5 | `PENDING` | — |
| Tidak memproses notifikasi selain GoPay | M3 | `PENDING` | — |

---

## 5. Definition of Done — [detail-project.md §42](../detail-project.md)

| Butir | Milestone | Status | Bukti |
|---|---|---|---|
| Aplikasi berjalan di Android | M2 | `PENDING` | — |
| TypeScript sebagai application language | M2 | `PENDING` | — |
| Kotlin untuk Notification Listener | M2 | `PENDING` | — |
| Notification Access dapat diaktifkan | M2 | `PENDING` | — |
| Listener berjalan di background | M6 | `PENDING` | — |
| Notifikasi GoPay terdeteksi | M2 | `PENDING` | — |
| Notifikasi aplikasi lain diabaikan | M3 | `PENDING` | — |
| Data notification dapat diekstrak | M3 | `PENDING` | — |
| Event ID dibuat | M3 | `PENDING` | — |
| Event tersimpan lokal | M3 | `PENDING` | — |
| Event dapat dikirim ke backend | M4 | `PENDING` | — |
| HTTPS untuk production | M1 | `PENDING` | — |
| Authentication diterapkan | M1 + M4 | `PENDING` | — |
| Retry mechanism berjalan | M4 | `PENDING` | — |
| Duplicate event ditangani | M1 + M3 | `PENDING` | — |
| Dashboard menampilkan listener status | M5 | `PENDING` | — |
| Dashboard menampilkan backend status | M5 | `PENDING` | — |
| History event tersedia | M5 | `PENDING` | — |
| Device ID tersedia | M1 | `PENDING` | — |
| Credential tidak hardcoded | M1 + M4 | `PENDING` | — |
| Menangani network failure | M4 | `PENDING` | — |
| Dapat diuji saat UI tidak terbuka | M6 | `PENDING` | — |
| Diuji pada physical Android device | M6 | `PENDING` | — |

---

## 6. Tabel kode → tindakan — [api-contract.md §4.5](../api-contract.md)

| Kondisi | Tindakan yang diharapkan | Milestone | Status | Bukti |
|---|---|---|---|---|
| `200 accepted` | → `SENT` | M4 | `PENDING` | — |
| `200 duplicate` | → `SENT`, bukan error | M4 | `PENDING` | — |
| `400 invalid_payload` | → `FAILED`, tanpa retry | M4 | `PENDING` | — |
| `401 invalid_signature` | → `FAILED`, peringatan kredensial | M4 | `PENDING` | — |
| `401 clock_skew` | → `FAILED`, pesan jam meleset + jam server | M4 | `PENDING` | — |
| `403 device_disabled` | → `FAILED`, tanpa retry | M4 | `PENDING` | — |
| `429` | tetap `PENDING`, hormati `Retry-After` | M4 | `PENDING` | — |
| `5xx` | tetap `PENDING`, retry | M4 | `PENDING` | — |
| timeout / jaringan mati | tetap `PENDING`, retry | M4 | `PENDING` | — |

---

## 7. Pemeriksaan keamanan — [qa-rules.md §9](qa-rules.md)

| Pemeriksaan | Status | Bukti |
|---|---|---|
| Tidak ada secret ter-hardcode | `PENDING` | — |
| Tidak ada secret/token/tanda tangan di log | `PENDING` | — |
| `hmac.Equal` dipakai, bukan `==` | `PENDING` | — |
| Body diverifikasi mentah sebelum decode | `PENDING` | — |
| Toleransi timestamp ditegakkan di kedua batas | `PENDING` | — |
| Idempotency memakai constraint database | `PENDING` | — |
| Build production menolak HTTP polos | `PENDING` | — |
| Mode Discovery default mati, tidak mengirim keluar | `PENDING` | — |

---

## 8. Penyimpangan dari spec

Sepuluh penyimpangan terdaftar di [§9 spec](../superpowers/specs/2026-09-10-ingestion-and-android-bridge-design.md) dan seluruhnya disengaja serta disetujui. Belum ada implementasi, sehingga belum ada yang dapat dikonfirmasi masih berlaku.

Penyimpangan baru yang tidak terdaftar: tidak ada.

---

## 9. Regresi

Tidak berlaku — ini baseline.

---

## 10. Temuan & tindak lanjut

Satu hal yang perlu diketahui sejak sekarang: **package identifier GoPay belum diketahui** dan sengaja tidak ditebak. Ia ditemukan di M2 lewat mode Discovery. Sampai saat itu, seluruh butir yang bergantung pada penyaringan GoPay tidak dapat diverifikasi dengan cara apa pun.
