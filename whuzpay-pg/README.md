# WhuzPay — Payment Gateway Aggregator

Platform pembayaran QRIS yang mengagregasi beberapa payment provider di
belakang satu API dan dashboard yang konsisten. Repo ini adalah workspace
berisi dua aplikasi independen, masing-masing dengan git repo sendiri:

```
whuzpay-pg/
├── back/   Go REST API (payment orchestration backend)
└── front/  Next.js dashboard (admin, merchant, checkout publik)
```

> `back/` dan `front/` masing-masing punya `.git` sendiri — bukan git
> submodule, jadi commit harus dilakukan dari dalam folder masing-masing.

## Prinsip Arsitektur

- **Orchestration backend, bukan wrapper provider.** Cashi hanyalah satu
  `PaymentProvider` adapter; provider lain bisa ditambahkan tanpa mengubah
  domain/service inti.
- **Status pembayaran dinormalisasi secara internal** — tiap adapter
  menerjemahkan status providernya sendiri ke status kanonik
  (`pending`/`paid`/`expired`/`failed`/`cancelled`).
- **Sandbox terisolasi dari production.** Setiap merchant punya environment
  `sandbox`/`production`; sandbox memakai adapter mock in-memory yang tidak
  pernah menyentuh network, dan secara eksplisit dikecualikan dari routing
  provider production.

## Komponen

| Komponen | Stack | Peran |
|---|---|---|
| [`back/`](back/README.md) | Go 1.25, Gorilla Mux, PostgreSQL | REST API: auth, payment, payment link, merchant, admin, webhook |
| [`front/`](front/README.md) | Next.js 16 (App Router), React 19, TypeScript, Tailwind v4 | Dashboard admin, dashboard merchant, landing page, halaman checkout publik |

## Alur Level Tinggi

```
Merchant Dashboard / Admin Panel (front/)
        │  fetch + Bearer JWT
        ▼
   REST API (back/) ── /api/v1/*
        │
        ├─ AuthService        → JWT admin & merchant (golang-jwt)
        ├─ PaymentService      → create/get payment, expire, webhook
        ├─ PaymentLinkService  → link pembayaran reusable → spawn Payment baru
        ├─ ProviderRouter      → pilih & failover antar PaymentProvider
        │        ├─ cashi/     (provider QRIS produksi, HTTP real)
        │        └─ sandbox/   (mock in-memory, tanpa network)
        └─ scheduler/          → expire payment, retry callback merchant (in-process ticker)
        │
        ▼
   PostgreSQL (migrations manual, tanpa ORM)
```

Endpoint publik tanpa auth: checkout by-reference, resolve & pay payment
link, provider webhook (divalidasi via signature, bukan session).
Endpoint merchant API-to-API pakai API key (`X-API-Key` /
`Authorization: Bearer <key>`), terpisah dari JWT session dashboard.

## Menjalankan Secara Lokal

Panduan lengkap (setup, kredensial seed, skenario mencoba, troubleshooting)
ada di [RUNNING.md](RUNNING.md). Ringkasnya, backend dan frontend dijalankan
sebagai dua proses terpisah:

```bash
# Terminal 1 — backend (port 8080)
cd back && cp .env.example .env && make migrate && make seed && make run

# Terminal 2 — frontend (port 3000)
cd front && npm install && npm run dev
```

## Area Sensitif (hati-hati saat mengubah)

- **`ProviderRouter` + pengecualian sandbox** (`back/internal/provider/adapter.go`,
  `back/internal/service/payment_service.go`) — mencegah trafik production
  ter-route ke provider mock.
- **Validasi signature webhook** per adapter provider — titik kritis untuk
  mencegah pemalsuan status "paid".
- **Context key middleware** (admin JWT vs merchant JWT vs API key) — salah
  urutan `Use()` di router bisa membuka endpoint tanpa auth.
- `back/internal/handler/` dan `back/internal/repository/` saat ini **tidak
  punya test coverage otomatis**.
