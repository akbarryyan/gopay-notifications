# Dashboard

Dashboard admin untuk satu instalasi self-hosted Payment Notification Bridge.
Next.js 16 (App Router) + Tailwind CSS v4 + shadcn/ui (base-ui).

Hanya mencakup halaman yang datanya benar-benar ada: **Overview**, **Devices**,
**Events**, **Transactions**, **API Keys**. Webhooks, License, Settings, dan
Logs ditampilkan di sidebar sebagai "Segera" — bukan link aktif — karena fase
2 (webhook) dan sistem lisensi belum dibangun. Lihat `docs/dashboard-spec.md`
untuk rancangan penuh dan
`docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md` untuk
rancangan Transactions/API Keys.

**Untuk siapa dashboard ini:** customer (pemilik instalasi), bukan vendor.
Model self-hosted + annual license berarti tiap customer men-deploy backend
dan dashboard-nya sendiri; vendor tidak pernah menyentuh instalasi mereka.
Satu instalasi = satu dashboard = satu customer. Karena itu, **jangan
menyebut istilah arsitektur internal ("backend", "database", nama service,
dsb) di teks yang tampil ke pengguna** — field API boleh tetap bernama
begitu, tapi label/copy di halaman diringkas jadi bahasa netral (mis.
"Status Layanan" alih-alih "Backend: operational"). Detail lengkap ada di
root `CLAUDE.md` bagian "Pemecahan scope".

## Arsitektur singkat

Satu domain, satu origin, tanpa CORS:

```text
Produksi (di belakang Caddy)
  /api/*   → backend Go
  /*       → Next.js (halaman ini)

Dev (tanpa Caddy)
  next.config.ts me-rewrite /api/* → BACKEND_URL (default http://localhost:8090)
```

Autentikasi lewat cookie sesi HttpOnly yang diterbitkan backend
(`POST /api/v1/admin/login`). `src/proxy.ts` hanya memeriksa **keberadaan**
cookie untuk mencegah kedipan halaman kosong sebelum redirect ke `/login` —
validitas sesi yang sesungguhnya selalu diputuskan backend lewat
`requireAdmin` pada setiap panggilan API.

## Setup

```bash
cd dashboard
npm install
cp .env.local.example .env.local   # isi BACKEND_URL bila backend tidak di :8090
```

Backend harus sudah jalan (lihat `backend/CLAUDE.md` / root `CLAUDE.md`) dan
sudah punya akun admin:

```bash
cd ../backend
make dev-admin   # buat/reset password admin, interaktif
```

Jalankan dev server (dijalankan sendiri oleh Akbar, bukan oleh Claude —
lihat pembagian kerja di root `CLAUDE.md`):

```bash
npm run dev
```

Buka `http://localhost:3000`, masuk dengan akun admin yang baru dibuat.

## Verifikasi tanpa dev server

Tiga perintah ini tidak menyalakan server apa pun, aman dijalankan siapa saja:

```bash
npx tsc --noEmit     # perlu `npx next typegen` sekali dulu bila belum pernah
npx eslint .
npx next build       # build produksi penuh, mem-verifikasi seluruh route
```

## Struktur

```text
src/
├── proxy.ts                    # Gerbang navigasi (bukan gerbang keamanan)
├── lib/
│   ├── api.ts                  # Klien fetch ke /api/v1/admin/*
│   ├── format.ts                # Format nominal & waktu, selaras dengan mobile/
│   └── use-api-data.ts          # Hook ambil-data: loading/error/401-redirect
├── components/
│   ├── ui/                      # shadcn/ui, jangan diedit manual — re-add via CLI
│   └── dashboard/                # Sidebar, AppShell, StatCard, FilterDropdown, DateRangeFilter, EventsTrendChart
└── app/
    ├── login/page.tsx
    └── (dashboard)/              # Route group berbagi AppShell (sidebar)
        ├── layout.tsx
        ├── page.tsx              # Overview
        ├── devices/page.tsx
        ├── events/page.tsx
        ├── transactions/page.tsx
        └── api-keys/page.tsx
```

## Catatan Next.js 16

`middleware.ts` sudah deprecated, diganti `proxy.ts` — lihat file itu
langsung, bukan mencarinya dengan nama lama. `cookies()` bersifat async.
Detail lain ada di `node_modules/next/dist/docs/` bila versi berubah lagi.
