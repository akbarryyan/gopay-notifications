# Vendor Dashboard

Dashboard vendor untuk mengelola License Server — **cuma dipakai Akbar**,
sama sekali terpisah dari `dashboard/` (dashboard customer di tiap
instalasi). Next.js 16 (App Router) + Tailwind CSS v4 + shadcn/ui (base-ui).

Kelola Customers, Licenses, Installations, dan Audit Log. Rancangan penuh:
`docs/superpowers/specs/2026-09-13-online-license-platform-design.md`
(mengikuti `docs/license-spec.md`, spec bisnis otoritatif). Cara deploy ada
di `backend/deploy/README.md` §"Lisensi".

**Kenapa app terpisah, bukan bagian dari `dashboard/`:** dashboard customer
dan dashboard vendor melihat database yang berbeda total (`gopay` vs
`gopay_license`), punya sesi login yang berbeda (`admin_session` vs
`vendor_session`), dan tidak boleh pernah tertukar — mencampurnya berarti
satu bug bisa membocorkan data customer lain ke customer, atau sebaliknya.

## Arsitektur singkat

Sama polanya dengan `dashboard/` — satu domain, satu origin, tanpa CORS:

```text
Produksi (di belakang Caddy, di subdomain vendor mis. license.whuzpay.com)
  /api/*   → License Server (Go)
  /*       → Next.js (halaman ini)

Dev (tanpa Caddy)
  next.config.ts me-rewrite /api/* → LICENSE_SERVER_URL (default http://localhost:8095)
```

Autentikasi lewat cookie sesi HttpOnly `vendor_session`, diterbitkan
License Server (`POST /api/v1/admin/login`). `src/proxy.ts` hanya memeriksa
**keberadaan** cookie untuk mencegah kedipan halaman kosong — validitas
sesi sesungguhnya selalu diputuskan License Server lewat `requireAdmin`
pada tiap panggilan API.

## Setup

```bash
cd vendor-dashboard
npm install
cp .env.local.example .env.local   # isi LICENSE_SERVER_URL bila bukan di :8095
```

License Server harus sudah jalan (lihat `backend/deploy/README.md`
§"Lisensi" bagian A, atau untuk dev lokal: `go run ./cmd/licenseserver`
dengan `DATABASE_URL` ke `gopay_license` lokal) dan sudah punya akun admin:

```bash
cd ../backend
go run ./cmd/licenseserver -create-admin akbar
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
│   ├── api.ts                  # Klien fetch ke /api/v1/admin/* License Server
│   ├── format.ts                # Format tanggal/waktu
│   └── use-api-data.ts          # Hook ambil-data: loading/error/401-redirect
├── components/ui/               # shadcn/ui, jangan diedit manual — re-add via CLI
└── app/
    ├── login/page.tsx
    └── (dashboard)/              # Route group berbagi layout (nav atas)
        ├── layout.tsx
        ├── page.tsx              # Customers: daftar + buat baru
        ├── customers/[id]/page.tsx   # Detail customer: daftar + buat license
        ├── licenses/[id]/page.tsx    # Detail license: installations, renew/suspend/revoke/reset
        └── audit-log/page.tsx
```

Aktivasi license customer **tidak lewat form di dashboard ini** — customer
mengisi `LICENSE_KEY` di `.env` instalasinya sendiri lalu restart backend,
konsisten dengan pola seluruh kunci lain di proyek ini. Dashboard ini hanya
menerbitkan/memperpanjang/mencabut license dan melihat status installation,
tidak pernah memicu aktivasi langsung.

## Catatan Next.js 16

`middleware.ts` sudah deprecated, diganti `proxy.ts` — lihat file itu
langsung, bukan mencarinya dengan nama lama. `cookies()` bersifat async.
Detail lain ada di `node_modules/next/dist/docs/` bila versi berubah lagi.
