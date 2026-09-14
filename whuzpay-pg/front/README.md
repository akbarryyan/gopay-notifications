# WhuzPay — Frontend

Dashboard admin, dashboard merchant, landing page, dan halaman checkout
publik untuk platform WhuzPay. Bagian dari workspace
[WhuzPay](../README.md); lihat README root untuk gambaran keseluruhan
sistem (backend + frontend).

## Tech Stack

- **Framework**: Next.js 16 (App Router), React 19, TypeScript
- **Styling**: Tailwind CSS v4 + shadcn/radix-ui, `tw-animate-css`
- **Chart**: Recharts
- **UI utilities**: `class-variance-authority`, `clsx` + `tailwind-merge`, `vaul` (drawer), `react-hot-toast`
- Tidak ada state-management library (Redux/Zustand) — auth & preferensi
  environment disimpan langsung di `localStorage`/`sessionStorage` lewat
  helper di `lib/*.ts`
- Tidak ada test runner terpasang (hanya `eslint`)

> Catatan dari `AGENTS.md`: versi Next.js di project ini punya breaking
> changes dari versi yang mungkin kamu kenal — cek
> `node_modules/next/dist/docs/` sebelum menulis kode yang bergantung pada
> API Next.js tertentu.

## Project Structure

```
front/
├── app/
│   ├── admin/                    # Panel admin
│   │   ├── login/                 # Login admin (publik)
│   │   └── (panel)/                # Halaman terproteksi: dashboard, payments,
│   │                                # merchants, providers, routing,
│   │                                # reconciliation, callbacks, logs, settings
│   ├── dashboard/                 # Dashboard merchant (terproteksi)
│   │   ├── payment-links/          # CRUD payment link
│   │   ├── payments/, api-keys/, webhooks/, reports/, settings/, profile/
│   ├── components/
│   │   ├── admin/                  # Komponen khusus panel admin
│   │   ├── merchant/               # Komponen khusus dashboard merchant
│   │   ├── landing/                 # Komponen landing page
│   │   └── providers/               # Provider React (mis. ToasterProvider)
│   ├── l/[slug]/                  # Redirect pendek payment link (publik)
│   ├── pay/[reference]/           # Halaman checkout publik (publik)
│   ├── login/, register/           # Auth merchant (publik)
│   └── page.tsx                    # Landing page
├── components/ui/                 # Komponen shadcn/radix generik (button, dialog, dst)
└── lib/
    ├── admin-api.ts, admin-auth.ts       # Client API + session admin
    ├── merchant-api.ts, merchant-auth.ts # Client API + session merchant
    ├── use-merchant-environment.ts        # Hook sandbox/production
    └── utils.ts
```

## Setup

```bash
npm install
cp .env.example .env.local   # kalau tersedia, atau set manual di bawah
npm run dev
```

Buka [http://localhost:3000](http://localhost:3000). Backend ([`../back`](../back))
harus berjalan di `http://localhost:8080` (default) sebelum login/API call
apa pun berfungsi.

### Environment Variables

| Var | Default | Keterangan |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Base URL backend API |

## Authentication

Dua sesi JWT independen, masing-masing disimpan terpisah di
`localStorage`/`sessionStorage` (tergantung "remember me"):

| Sesi | Helper | Endpoint login |
|---|---|---|
| Admin | `lib/admin-auth.ts` | `POST /api/v1/auth/admin/login` |
| Merchant | `lib/merchant-auth.ts` | `POST /api/v1/auth/login` |

Token dikirim sebagai `Authorization: Bearer <token>` di setiap request API
via helper `authHeaders()` masing-masing. `AdminAuthGate` dan
`MerchantAuthGate` (di `app/components/{admin,merchant}/`) membungkus
layout halaman terproteksi dan redirect ke halaman login kalau token tidak
ada/invalid — **bukan** middleware Next.js di level route.

Environment merchant (sandbox/production) disimpan terpisah
(`lib/use-merchant-environment.ts`) dan dipertahankan lintas login/logout.

## Alur Request API

Semua pemanggilan API dilakukan langsung dengan `fetch` browser-native ke
`NEXT_PUBLIC_API_URL` (tidak ada API route Next.js sebagai proxy, tidak ada
axios/react-query) — lihat `lib/admin-api.ts` dan `lib/merchant-api.ts`
untuk pola standarnya: build headers auth → `fetch` → parse JSON → lempar
`Error` kalau `!res.ok`.

## Scripts

```bash
npm run dev     # dev server (hot reload)
npm run build   # production build
npm run start   # jalankan hasil build
npm run lint    # eslint
```

## Testing

Belum ada test runner (unit/e2e) terpasang di project ini. Verifikasi
perubahan UI dilakukan manual lewat browser.

## Area Sensitif

Lihat [README root](../README.md) untuk daftar area berisiko tinggi lintas
sistem. Spesifik di frontend:

- `lib/*-auth.ts` — logic penyimpanan & pembersihan token sesi; salah ubah
  bisa membuat sesi bocor lintas tab atau tidak ter-clear saat logout.
- `AdminAuthGate` / `MerchantAuthGate` — satu-satunya lapisan proteksi
  route di client; tidak ada middleware server-side yang memblokir akses.
