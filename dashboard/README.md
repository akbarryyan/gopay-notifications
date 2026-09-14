# Dashboard

Dashboard customer Payment Notification Bridge — hosted multi-tenant sejak
pivot arsitektur 2026-09-13 (lihat root `CLAUDE.md` "Sistem akun
multi-tenant"). Satu deployment melayani seluruh customer sekaligus, data
dipisah lewat `account_id` per akun, bukan lewat instalasi terpisah lagi.
Next.js 16 (App Router) + Tailwind CSS v4 + shadcn/ui (base-ui).

`/` adalah landing page publik dan `/register` form signup swalayan (plan
Starter, trial 3 hari, langsung aktif tanpa campur tangan vendor) — spec
`docs/superpowers/specs/2026-09-13-landing-signup-design.md`. Keduanya di
luar route group `(dashboard)`, dikecualikan dari gerbang sesi di
`proxy.ts`. Overview (halaman berkebutuhan sesi) ada di `/overview`.

Hanya mencakup halaman yang datanya benar-benar ada: **Overview**, **Devices**,
**Events**, **Transactions**, **API Keys**, **Webhooks**, **Exceptions**,
**License**. Settings dan Logs ditampilkan di sidebar sebagai "Segera" —
bukan link aktif. Lihat `docs/dashboard-spec.md` untuk rancangan penuh,
`docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md` untuk
rancangan Transactions/API Keys,
`docs/superpowers/specs/2026-09-13-webhook-delivery-design.md` untuk
rancangan Webhooks,
`docs/superpowers/specs/2026-09-13-exception-console-design.md` untuk
rancangan Exceptions, dan
`docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md` untuk
rancangan License (halaman ini belum ikut disesuaikan penuh ke swalayan
tambah device — lihat spec §7, ditunda ke sub-project terpisah).

**Untuk siapa dashboard ini:** customer, bukan vendor. Satu akun = satu
login = satu customer, tapi SEMUA customer berbagi deployment backend yang
sama sekarang (bukan lagi satu instalasi per customer) — akun dibuat vendor
lewat Vendor Dashboard, bukan lagi lewat CLI di server customer sendiri.
Karena itu, **jangan menyebut istilah arsitektur internal ("backend",
"database", nama service, dsb) di teks yang tampil ke pengguna** — field
API boleh tetap bernama begitu, tapi label/copy di halaman diringkas jadi
bahasa netral (mis. "Status Layanan" alih-alih "Backend: operational").
Detail lengkap ada di root `CLAUDE.md` bagian "Pemecahan scope".

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
sudah ada account customer untuk login. Dua cara: (1) daftar sendiri lewat
`/register` (plan Starter, trial 3 hari, langsung aktif) — cara tercepat
untuk dev lokal; atau (2) dibuat lewat Vendor Dashboard
(`vendor-dashboard/`, cuma dipakai Akbar, plan bebas dipilih) — **bukan
lagi** `make dev-admin` (itu sekarang membuat akun vendor, dipakai login
ke Vendor Dashboard itu sendiri, bukan akun customer). Lihat
`vendor-dashboard/README.md`. Untuk data dev yang lebih banyak sekaligus
(banyak account + device/invoice/event dsb), pakai `cmd/seedtool` di
`backend/` — lihat root `CLAUDE.md`.

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
├── proxy.ts                    # Gerbang navigasi (bukan gerbang keamanan) --
│                                # mengecualikan /, /register, /login
├── lib/
│   ├── api.ts                  # Klien fetch ke /api/v1/admin/* + signup()
│   ├── format.ts                # Format nominal & waktu, selaras dengan mobile/
│   └── use-api-data.ts          # Hook ambil-data: loading/error/401-redirect
├── components/
│   ├── ui/                      # shadcn/ui, jangan diedit manual — re-add via CLI
│   ├── auth/                     # Panel kiri login/register (mockup dashboard, ilustratif)
│   ├── landing/                  # Section landing page publik (navbar, hero, pricing, dst)
│   └── dashboard/                 # Sidebar, AppShell, StatCard, FilterDropdown,
│                                  # DateRangeFilter, EventsTrendChart (Recharts, biaxial)
└── app/
    ├── page.tsx                  # Landing page PUBLIK (bukan Overview lagi)
    ├── register/page.tsx         # Signup swalayan
    ├── login/page.tsx
    └── (dashboard)/              # Route group berbagi AppShell (sidebar), butuh sesi
        ├── layout.tsx
        ├── overview/page.tsx     # Overview (dulu di "/")
        ├── devices/page.tsx
        ├── events/page.tsx
        ├── transactions/page.tsx
        ├── api-keys/page.tsx
        ├── webhooks/page.tsx
        ├── exceptions/page.tsx
        └── license/page.tsx
```

## Catatan Next.js 16

`middleware.ts` sudah deprecated, diganti `proxy.ts` — lihat file itu
langsung, bukan mencarinya dengan nama lama. `cookies()` bersifat async.
Detail lain ada di `node_modules/next/dist/docs/` bila versi berubah lagi.
