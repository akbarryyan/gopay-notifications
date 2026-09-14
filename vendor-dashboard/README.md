# Vendor Dashboard

Dashboard vendor untuk mengelola account customer — **cuma dipakai Akbar**,
sama sekali terpisah dari `dashboard/` (dashboard customer). Next.js 16
(App Router) + Tailwind CSS v4 + shadcn/ui (base-ui).

Kelola Accounts (customer), lihat ringkasan lintas platform di Dashboard,
dan Audit Log. Rancangan penuh:
`docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md`. Sejak
pivot ke hosted multi-tenant (2026-09-13), app ini memanggil endpoint
vendor (`/api/v1/vendor/*`) **di backend utama yang sama** dipakai
`dashboard/` — License Server yang dulu terpisah sudah dibongkar total,
bukan lagi service sendiri.

**Kenapa app terpisah, bukan bagian dari `dashboard/`:** meski sama-sama
memanggil backend yang satu, dashboard customer dan dashboard vendor punya
sesi login yang berbeda (`admin_session` vs `vendor_session`, kunci
tanda tangan berbeda) dan tidak boleh pernah tertukar — mencampurnya
berarti satu bug bisa membocorkan data customer lain ke customer, atau
sebaliknya.

## Arsitektur singkat

Sama polanya dengan `dashboard/` — satu domain, satu origin, tanpa CORS:

```text
Produksi (di belakang Caddy, di subdomain vendor mis. vendor.whuzpay.com)
  /api/*   → backend utama (Go), sama dengan yang dipakai dashboard/
  /*       → Next.js (halaman ini)

Dev (tanpa Caddy)
  next.config.ts me-rewrite /api/* → BACKEND_URL (default http://localhost:8090)
```

Autentikasi lewat cookie sesi HttpOnly `vendor_session`, diterbitkan
backend utama (`POST /api/v1/vendor/login`). `src/proxy.ts` hanya
memeriksa **keberadaan** cookie untuk mencegah kedipan halaman kosong —
validitas sesi sesungguhnya selalu diputuskan backend lewat
`requireVendor` pada tiap panggilan API.

## Setup

```bash
cd vendor-dashboard
npm install
cp .env.local.example .env.local   # isi BACKEND_URL bila backend tidak di :8090
```

Backend harus sudah jalan (lihat `backend/CLAUDE.md` / root `CLAUDE.md`)
dan sudah punya akun vendor:

```bash
cd ../backend
go run ./cmd/admintool -username akbar   # buat/reset akun vendor, interaktif
```

Jalankan dev server (dijalankan sendiri oleh Akbar, bukan oleh Claude —
lihat pembagian kerja di root `CLAUDE.md`):

```bash
npm run dev
```

Buka `http://localhost:3000`, masuk dengan akun vendor yang baru dibuat.

## Verifikasi tanpa dev server

Tiga perintah ini tidak menyalakan server apa pun, aman dijalankan siapa saja:

```bash
npx tsc --noEmit
npx eslint .
npx next build       # build produksi penuh, mem-verifikasi seluruh route
```

## Struktur

```text
src/
├── proxy.ts                    # Gerbang navigasi (bukan gerbang keamanan)
├── lib/
│   ├── api.ts                   # Klien fetch ke /api/v1/vendor/*
│   ├── format.ts                 # Format tanggal/waktu/rupiah
│   ├── account-status.ts         # Warna badge status, dipakai Dashboard + Accounts
│   └── use-api-data.ts           # Hook ambil-data: loading/error/401-redirect
├── components/
│   ├── ui/                       # shadcn/ui, jangan diedit manual — re-add via CLI
│   └── dashboard/                # app-shell (sidebar collapsible), sidebar-nav,
│                                  # nav-items, stat-card, overview-trend-chart
│                                  # -- pola sama dengan components/dashboard/ di dashboard/
└── app/
    ├── login/page.tsx
    └── (dashboard)/              # Route group berbagi AppShell (sidebar)
        ├── layout.tsx
        ├── page.tsx              # Dashboard ("/"): ringkasan lintas account + grafik
        ├── accounts/page.tsx     # Accounts: daftar + buat baru
        ├── accounts/[id]/page.tsx    # Detail account: renew/suspend/revoke
        └── audit-log/page.tsx
```

Pembuatan device untuk account customer **tidak lewat dashboard ini** —
lewat `cmd/devicetool -account <id> -name "..."` di server (swalayan dari
Customer Dashboard belum ada, ditunda ke sub-project terpisah, lihat spec
§7). Dashboard ini cuma menerbitkan/mengelola account, tidak pernah
memicu pembuatan device.

## Catatan Next.js 16

`middleware.ts` sudah deprecated, diganti `proxy.ts` — lihat file itu
langsung, bukan mencarinya dengan nama lama. `cookies()` bersifat async.
Detail lain ada di `node_modules/next/dist/docs/` bila versi berubah lagi.
