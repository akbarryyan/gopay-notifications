# Landing Page Redesign — Design

**Tanggal:** 2026-08-18
**Ruang lingkup:** `front/app/page.tsx` dan `front/app/components/landing/*`
**Status:** disetujui, siap masuk rencana implementasi

## Masalah

Landing page sekarang terdiri dari 10 komponen (~800 baris) yang strukturnya
sudah rapi, tapi hasil visualnya datar dan isinya tidak mencerminkan produk.

Temuan konkret:

1. **Hierarki hilang.** `h1` hero dan `h2` section sama-sama `text-4xl`, jadi
   hero tidak terbaca sebagai puncak halaman.
2. **Tidak ada ritme.** Lima section berlatar putih beruntun (Hero →
   PartnerLogos → Solutions → Testimonial → ServicePillars) dengan `py-20`
   seragam; batas antar-bagian tidak terbaca.
3. **Tipografi melawan keterbacaan.** Heading panjang di-`uppercase` sekaligus
   diberi `tracking-[-0.04em]` dan `leading-[0.98]` — kapital dengan tracking
   negatif serapat itu justru lebih sulit dibaca, dan barisnya saling menempel.
4. **Detail yang terlihat belum jadi.** `Solutions` memakai `rounded-2xl`
   bersama `clipPath` sehingga sudut membulatnya terpotong; tag QRIS/Checkout/API
   di-render sebagai gambar `placehold.co` padahal cukup badge teks;
   `MediaFeature` menampilkan placeholder "Foto+Kantor" mentah; panah di
   `TestimonialCarousel` tidak berfungsi (tidak ada state) dan isinya hanya satu
   entri.
5. **Tidak ada `focus-visible` di satu pun link/tombol** — navigasi keyboard
   tidak terlihat sama sekali.
6. **Produk tidak pernah ditampilkan.** Tidak ada dashboard, tidak ada halaman
   checkout, tidak ada contoh integrasi — padahal itu yang dijual.

## Keputusan

| Pertanyaan | Keputusan |
|---|---|
| Seberapa jauh perubahannya | Susun ulang isi, bukan sekadar polish CSS |
| Audiens | Dua jalur terpisah: hero untuk pengambil keputusan bisnis, satu section teknis khusus untuk developer |
| Social proof | Slot logo dan testimonial dipertahankan, diisi aset netral buatan sendiri (tidak lagi hotlink `duitku.com`) |
| Susunan | Pendekatan A — dua jalur berurutan (produk dulu, lalu integrasi), bukan tab |

Pendekatan tab ("Untuk bisnis" / "Untuk developer") ditolak karena
menyembunyikan separuh isi di balik klik dan membuatnya tidak terbaca mesin
pencari. Pendekatan linear teknis ditolak karena pembaca non-teknis kehilangan
alur.

## Fondasi Visual

Diterapkan ke seluruh section. Ini akar perbaikannya — masing-masing section di
bawah hanya menerapkan aturan ini.

### Skala tipografi

| Peran | Kelas |
|---|---|
| h1 (hero) | `text-4xl sm:text-5xl lg:text-6xl font-extrabold leading-tight tracking-tight` |
| h2 (judul section) | `text-3xl sm:text-4xl font-extrabold leading-tight tracking-tight` |
| h3 (judul kartu) | `text-xl font-bold leading-snug` |
| body | `text-base sm:text-lg leading-relaxed text-slate-600` |
| eyebrow | `text-sm font-semibold uppercase tracking-wider text-brand-navy-light` |

Aturan yang mengikat:

- `uppercase` hanya untuk eyebrow dan label pendek (badge, tombol), selalu
  dengan tracking **positif**.
- Heading kalimat panjang memakai sentence case.
- `leading` heading minimal `leading-tight` (~1.15). Tidak ada lagi
  `leading-[0.98]`.
- Blok teks terpusat dibatasi `max-w-3xl` agar panjang baris tetap nyaman.

### Ritme dan latar

- Section utama: `py-20 sm:py-28`. Strip logo: `py-12`.
- Pergantian latar yang menandai batas section, bukan garis pemisah:

  | Section | Latar |
  |---|---|
  | Hero | gradient `brand-navy` |
  | Logo strip | `white` |
  | Produk | `slate-50` |
  | Cara Kerja | `brand-navy-dark` |
  | Fitur | `white` |
  | Testimonial | `slate-50` |
  | FAQ | `sky-50` |
  | CTA penutup | `brand-navy` |
  | Footer | `brand-navy-dark` (tidak berubah) |
- `Container` tetap `max-w-7xl px-6 lg:px-8` (tidak berubah).

### Aksesibilitas

- Setiap link dan tombol mendapat
  `focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-yellow focus-visible:ring-offset-2`.
- Animasi menghormati `motion-reduce` (marquee sudah, carousel menyusul).
- Carousel dapat dioperasikan dengan keyboard dan mengumumkan slide aktif.

### Komponen `Section` baru

Delapan komponen saat ini menulis ulang pola "eyebrow + judul + subjudul"
dengan ukuran yang berbeda-beda — itu sumber inkonsistensinya. `Section`
menerima `eyebrow`, `title`, `subtitle`, `background` (`white` | `slate` |
`navy` | `sky`), dan `align`, lalu merender heading dengan skala di atas.
Section yang butuh tata letak khusus tetap boleh menyusun isinya sendiri di
dalam `Section`.

## Susunan Section

### 1. Navbar (`Navbar.tsx`, disunting)

Transparan saat di puncak halaman, berubah jadi latar solid `brand-navy` +
shadow setelah di-scroll (listener `scroll` dengan `passive: true`). Menu
mobile dirapikan: transisi buka/tutup, jarak sentuh diperbesar, tertutup saat
salah satu item diklik. Tombol kanan jadi CTA "Coba Sandbox" di samping tautan
Login.

### 2. Hero (`Hero.tsx`, disunting)

- Headline ke skala h1 penuh.
- `clipPath: polygon(0 0, 100% 0, 100% 58%, 0 90%)` diganti bentuk lengkung
  yang tidak memotong mockup di layar sedang.
- Baris mikro-trust di bawah CTA: "Sandbox gratis · tanpa kartu kredit ·
  integrasi dalam hitungan menit".
- `DashboardMockup` dipindah ke berkas sendiri dan dibuat responsif — posisi
  `absolute` sekarang bisa meluber di viewport sempit.

### 3. Logo strip (`LogoStrip.tsx`, menggantikan `PartnerLogos.tsx`)

Wordmark SVG netral buatan sendiri, nama perusahaan generik, tanpa permintaan
jaringan ke host luar. Marquee dipertahankan, ditambah mask gradient di tepi
kiri-kanan agar logo memudar alih-alih terpotong. Entri `duitku.com` dihapus
dari `next.config.ts`.

### 4. Produk (`ProductShowcase.tsx`, baru)

Dua baris bergantian kiri-kanan:

1. **Dashboard merchant** — monitoring transaksi real-time, status, riwayat.
2. **Halaman checkout QRIS** — preview `/pay/{reference}` yang dilihat pembayar:
   QR, nominal, hitung mundur kedaluwarsa.

Kedua visual dibuat sebagai markup HTML/SVG sendiri (mengikuti pola
`DashboardMockup` yang sudah ada), bukan `placehold.co`.

### 5. Cara Kerja (`HowItWorks.tsx` + `CodeBlock.tsx`, baru — menggantikan `MediaFeature.tsx`)

Berlatar navy. Tiga langkah bernomor:

1. Buat API key di dashboard (pilih environment sandbox).
2. `POST /api/v1/payments` dengan header `X-API-Key` → dapat QRIS.
3. Terima webhook saat pembayaran lunas → status jadi `paid`.

Disertai satu blok kode cURL yang **sesuai kontrak API sungguhan**
(`amount`, `payment_method`, `description`, `expires_in_minutes`) dan badge
"webhook bertanda tangan HMAC". `CodeBlock` statis — tanpa dependensi
syntax highlighter baru; pewarnaan seadanya lewat markup.

### 6. Fitur (`Features.tsx`, melebur `Solutions.tsx` + `ServicePillars.tsx`)

Keduanya sekarang membahas hal yang tumpang tindih, jadi digabung menjadi satu
grid kartu. `clipPath` yang merusak sudut membulat dibuang. Tag berbasis gambar
diganti badge teks.

### 7. Testimonial (`TestimonialCarousel.tsx`, disunting)

Carousel yang benar-benar berfungsi: state indeks, tombol prev/next aktif,
navigasi panah kiri/kanan keyboard, dot indicator, `aria-live` sopan. Tiga
entri dummy netral. Di viewport sempit tombol pindah ke bawah kartu agar kartu
tidak terjepit.

### 8. FAQ (`FAQSection.tsx`, disunting)

Accordion dengan transisi tinggi dan ikon +/− yang berputar. Item memakai
`<button aria-expanded>` yang benar.

### 9. CTA penutup (`CtaRow.tsx`, disunting)

Jadi band navy penuh, bukan putih — penutup halaman yang tegas sebelum footer.

### 10. Footer (`Footer.tsx`, disunting)

Ritme kolom dan jarak disamakan dengan sisa halaman. Tidak ada perubahan isi.

## Perubahan Berkas

**Baru**

- `app/components/landing/Section.tsx`
- `app/components/landing/LogoStrip.tsx`
- `app/components/landing/ProductShowcase.tsx`
- `app/components/landing/HowItWorks.tsx`
- `app/components/landing/CodeBlock.tsx`
- `app/components/landing/Features.tsx`
- `app/components/landing/DashboardMockup.tsx` (dipisah dari `Hero.tsx`)

**Dihapus**
- `app/components/landing/MediaFeature.tsx` (digantikan `HowItWorks`)
- `app/components/landing/PartnerLogos.tsx` (digantikan `LogoStrip`)
- `app/components/landing/Solutions.tsx`, `ServicePillars.tsx` (dilebur ke `Features`)

**Disunting**
- `app/page.tsx` (susunan section)
- `Navbar.tsx`, `Hero.tsx`, `TestimonialCarousel.tsx`, `FAQSection.tsx`,
  `CtaRow.tsx`, `Footer.tsx`
- `next.config.ts` — hapus **kedua** entri `remotePatterns`. Landing page adalah
  satu-satunya pemakai `placehold.co` maupun `duitku.com` di seluruh aplikasi
  (diverifikasi lewat pencarian), jadi setelah redesign tidak ada lagi gambar
  dari host luar.

`Container.tsx` tidak berubah.

## Di Luar Ruang Lingkup

- Dark mode untuk landing page — halaman ini `bg-white` permanen; dashboard
  punya sistem temanya sendiri.
- Penulisan ulang copywriting secara menyeluruh. Teks yang ada dipertahankan
  kecuali section baru, yang butuh teks baru.
- Animasi scroll-reveal.
- Perubahan pada dashboard, panel admin, atau halaman checkout sungguhan.

## Kriteria Selesai

1. `npm run build` dan `npm run lint` lolos tanpa peringatan baru.
2. Tidak ada lagi permintaan gambar ke `duitku.com` dari landing page.
3. Setiap link dan tombol di landing punya indikator fokus yang terlihat saat
   di-Tab.
4. Carousel testimonial bisa dijalankan dengan mouse maupun keyboard, dan
   berpindah slide sungguhan.
5. Tidak ada scroll horizontal pada lebar 320px, 768px, 1280px, dan 1920px.
6. Blok kode di section Cara Kerja cocok dengan kontrak `POST /api/v1/payments`
   yang sebenarnya (lihat `back/docs/openapi.yaml`).
