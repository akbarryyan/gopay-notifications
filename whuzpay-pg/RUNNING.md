# Menjalankan & Mencoba WhuzPay (Lokal)

Panduan praktis untuk menjalankan workspace ini di mesin lokal, lalu
mencobanya lewat dashboard, API, dan script flow-test. Untuk gambaran
arsitektur lihat [README.md](README.md); untuk detail per aplikasi lihat
[back/README.md](back/README.md) dan [front/README.md](front/README.md).

---

## 1. Prasyarat

| Kebutuhan               | Versi    | Dipakai untuk                                |
| ----------------------- | -------- | -------------------------------------------- |
| Go                      | 1.25+    | backend `back/`                              |
| Node.js + npm           | Node 20+ | frontend `front/`                            |
| PostgreSQL              | 14+      | database (dijalankan lokal, bukan container) |
| `psql`                  | —        | dipakai script `make migrate` / `make seed`  |
| `jq`, `openssl`, `curl` | —        | hanya untuk `flow-test/`                     |
| `air` (opsional)        | —        | hot reload backend (`make dev`)              |

Cek cepat:

```bash
go version && node -v && npm -v && psql --version
jq --version && openssl version   # opsional, untuk flow-test
```

---

## 2. Setup Pertama Kali

### Backend

```bash
cd back
cp .env.example .env       # lalu edit sesuai kebutuhan (lihat catatan port di bawah)
go mod download
make db-create             # bikin database pg_aggregator (sekali saja, opsional)
make migrate               # idempoten — file yang sudah applied akan di-skip
make seed                  # admin, merchant demo, 50 merchant, ~200 payment contoh
```

### Frontend

```bash
cd front
npm install
# .env.local: NEXT_PUBLIC_API_URL harus menunjuk ke URL backend
```

### flow-test (opsional)

```bash
cd flow-test
cp .env.example .env
chmod +x *.sh lib/*.sh
```

> **Catatan port.** Default di `.env.example` adalah `APP_PORT=8080`. Di mesin
> ini port 8080 sudah dipakai proses lain, jadi `back/.env` memakai
> **`APP_PORT=8090`**. Tiga tempat ini harus konsisten:
>
> | File               | Key                   | Nilai di workspace ini          |
> | ------------------ | --------------------- | ------------------------------- |
> | `back/.env`        | `APP_PORT`, `APP_URL` | `8090`, `http://localhost:8090` |
> | `front/.env.local` | `NEXT_PUBLIC_API_URL` | `http://localhost:8090`         |
> | `flow-test/.env`   | `BASE_URL`            | `http://localhost:8090`         |
>
> Sisa dokumen ini memakai `8090`. Ganti kalau setup kamu beda.

### Environment variable yang wajib benar

| Var                                 | Kenapa penting                                                                |
| ----------------------------------- | ----------------------------------------------------------------------------- |
| `APP_URL`, `FRONTEND_URL`           | `config.Validate()` menolak start kalau kosong                                |
| `DB_*`                              | koneksi PostgreSQL; dipakai juga oleh `make migrate`/`make seed`              |
| `JWT_SECRET`                        | wajib diganti kalau `APP_ENV=production` (nilai default ditolak)              |
| `CASHI_API_KEY`, `CASHI_SECRET_KEY` | hanya perlu valid untuk payment `production`; sandbox tidak menyentuh network |

---

## 3. Menjalankan

Dua proses, dua terminal:

```bash
# Terminal 1 — backend
cd back && make run          # atau: make dev   (hot reload, butuh air)

# Terminal 2 — frontend
cd front && npm run dev
```

Kalau `make dev` bilang `air not found`:

```bash
go install github.com/air-verse/air@latest   # lalu pastikan $(go env GOPATH)/bin ada di PATH
```

### Verifikasi backend hidup

```bash
curl -s http://localhost:8090/api/v1/health
# {"status":"ok"}
```

Frontend: buka <http://localhost:3000>.

---

## 4. Kredensial Hasil Seed

| Peran    | Email                               | Password       | Halaman login                       |
| -------- | ----------------------------------- | -------------- | ----------------------------------- |
| Admin    | `admin@pg-aggregator.local`         | `Admin123!`    | <http://localhost:3000/admin/login> |
| Merchant | `merchant.demo@pg-aggregator.local` | `Merchant123!` | <http://localhost:3000/login>       |

Merchant demo: `11111111-1111-1111-1111-111111111111`.
Seed juga mengisi 50 merchant lain + ratusan payment historis supaya
dashboard, chart, dan filter tidak kosong.

---

## 5. Peta URL

### Frontend (port 3000)

| URL                   | Isi                                                                                    |
| --------------------- | -------------------------------------------------------------------------------------- |
| `/`                   | landing page                                                                           |
| `/login`, `/register` | auth merchant                                                                          |
| `/dashboard`          | dashboard merchant (payments, api-keys, payment-links, webhooks, reports, settings)    |
| `/admin/login`        | login admin                                                                            |
| `/admin`              | panel admin (merchants, payments, providers, routing, reconciliation, callbacks, logs) |
| `/pay/{reference}`    | **halaman checkout publik** — QR + polling status, tanpa auth                          |
| `/l/{slug}`           | **payment link publik** — pengunjung isi nominal/data lalu di-spawn payment baru       |

### Backend (port 8090, prefix `/api/v1`)

| Endpoint                                                                    | Auth                                                              |
| --------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| `GET /health`                                                               | —                                                                 |
| `POST /auth/login`, `/auth/register`, `/auth/admin/login`                   | — (rate limit 5 req/menit)                                        |
| `/admin/*`                                                                  | JWT admin                                                         |
| `/merchant/*`                                                               | JWT merchant                                                      |
| `POST /payments`, `GET /payments/{id}`, `GET /payments/{id}/status`         | API key merchant (`X-API-Key` atau `Authorization: Bearer <key>`) |
| `GET /public/payments/by-reference/{reference}`                             | —                                                                 |
| `GET /public/payment-links/{slug}`, `POST /public/payment-links/{slug}/pay` | —                                                                 |
| `POST /provider-webhooks/{providerName}`                                    | signature `x-gateway-signature`                                   |

Kontrak lengkap (semua route + schema): [back/docs/openapi.yaml](back/docs/openapi.yaml),
bisa dibuka di <https://editor.swagger.io>.

---

## 6. Skenario Mencoba

### A. Lewat dashboard (paling cepat)

1. Login merchant di <http://localhost:3000/login>.
2. Pastikan switch environment di **header dashboard** ada di posisi
   `sandbox` — switch itu menentukan environment semua data yang ditampilkan
   maupun payment yang dibuat.
3. Buka **Payments → buat payment baru**.
4. Backend memakai adapter mock in-memory — QR digenerate lokal, tidak ada
   panggilan HTTP ke Cashi sama sekali.
5. Buka halaman checkout publik `/pay/{reference}` (reference ada di detail
   payment) untuk melihat tampilan yang dilihat pembayar: QR + polling status.
6. Login admin di `/admin/login` untuk melihat payment yang sama dari sisi
   operator: detail, event, provider routing, callback, logs.

### B. Payment link publik

1. Merchant dashboard → **Payment Links → New**, isi nama + nominal.
2. Buka `/l/{slug}` di tab baru (atau incognito — halaman ini tanpa auth).
3. Setiap checkout di link tersebut men-spawn **payment baru sekali pakai**;
   link-nya sendiri tetap reusable.

### C. API merchant lewat curl (integrasi API-to-API)

Belum ada API key di seed — mint dulu lewat admin API:

```bash
BASE=http://localhost:8090
MERCHANT=11111111-1111-1111-1111-111111111111

# 1. Login admin
TOKEN=$(curl -s -X POST $BASE/api/v1/auth/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@pg-aggregator.local","password":"Admin123!"}' | jq -r .token)

# 2. Mint API key sandbox (password admin dikonfirmasi ulang)
APIKEY=$(curl -s -X PUT $BASE/api/v1/admin/merchants/$MERCHANT/api-keys \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"environment":"sandbox","password":"Admin123!"}' | jq -r .secret)

# 3. Create payment
curl -s -X POST $BASE/api/v1/payments \
  -H "X-API-Key: $APIKEY" -H 'Content-Type: application/json' \
  -d '{"amount":10000,"payment_method":"qris","description":"coba-coba","expires_in_minutes":30}' | jq .

# 4. Cek status (pakai id dari response di atas)
curl -s $BASE/api/v1/payments/<PAYMENT_ID>/status -H "X-API-Key: $APIKEY" | jq .
```

API key hanya ditampilkan **sekali** saat dibuat — simpan nilainya.
Nominal minimum sandbox adalah **2000**; di bawah itu ditolak adapter.
Backend menambahkan suffix unik 1–99 ke nominal (meniru perilaku Cashi),
jadi `amount` di response sedikit berbeda dari yang dikirim — itu normal.

### D. Script flow-test (otomatis, end-to-end)

```bash
cd flow-test
./run_all.sh                 # sandbox: health → login admin → mint key → create payment → cek status
./run_all.sh production      # + kirim webhook simulasi sampai status jadi paid
```

Step satuan (state token/key/payment id disimpan di `.state/`):

```bash
./01_health.sh
./02_admin_login.sh
./03_create_api_key.sh sandbox    # atau production
./04_create_payment.sh
./05_webhook_settle.sh            # khusus mode production
./06_check_status.sh
rm -rf .state/*                   # reset
```

> `./run_all.sh production` membuat request **asli** dari backend ke Cashi,
> jadi butuh `CASHI_API_KEY` / `CASHI_SECRET_KEY` yang valid di `back/.env`.
> `CASHI_SECRET_KEY` dibaca otomatis dari `back/.env`, tidak perlu disalin.

### E. Simulasi webhook manual

Webhook divalidasi HMAC-SHA256 atas raw body memakai `CASHI_SECRET_KEY`,
dikirim di header `x-gateway-signature`:

```bash
BODY='{"event":"PAYMENT_SETTLED","data":{"order_id":"<PROVIDER_REFERENCE>","status":"SETTLED"}}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$CASHI_SECRET_KEY" | awk '{print $2}')
curl -s -X POST http://localhost:8090/api/v1/provider-webhooks/cashi \
  -H "x-gateway-signature: $SIG" -H 'Content-Type: application/json' -d "$BODY" | jq .
```

Setelah webhook diterima: status payment berubah jadi `paid` dan callback ke
merchant dipicu (bisa dilihat di admin → Callbacks).

---

## 7. Catatan Sandbox vs Production

|                        | sandbox                                                  | production                 |
| ---------------------- | -------------------------------------------------------- | -------------------------- |
| Provider               | mock in-memory (`internal/provider/sandbox`)             | Cashi (HTTP asli)          |
| Butuh kredensial Cashi | tidak                                                    | ya                         |
| Webhook provider       | **ditolak**                                              | diterima (signature valid) |
| Bisa jadi `paid`?      | tidak lewat alur normal — tetap `pending` sampai expired | ya, lewat webhook          |

Konsekuensi praktis: payment sandbox **memang tidak bisa di-`paid`-kan** lewat
webhook — itu disengaja, bukan bug. Untuk mencoba tampilan status `paid`,
pakai payment hasil seed (sudah ada yang `paid`/`expired`/`failed`/`cancelled`)
atau jalankan `./run_all.sh production`.

Environment ditentukan dari API key yang dipakai (integrasi API-to-API) atau
dari switch environment di header dashboard merchant. Trafik `production`
**tidak pernah** di-route ke adapter sandbox — pengecualian ini di-enforce di
`internal/service/payment_service.go`.

---

## 8. Test

```bash
cd back
go test ./...           # semua test, hermetic — tidak butuh Postgres
go test ./... -race     # + race detector (dipakai di CI)
go vet ./...
```

Frontend:

```bash
cd front
npm run lint
npm run build           # verifikasi build production
```

---

## 9. Troubleshooting

| Gejala                                              | Penyebab & solusi                                                                                                                                                       |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bind: address already in use`                      | port `APP_PORT` sudah dipakai proses lain. Cek `ss -ltnp \| grep 8090`, atau ganti `APP_PORT` (+ `APP_URL`, `NEXT_PUBLIC_API_URL`, `BASE_URL` flow-test).               |
| Backend exit saat start, keluhan config             | `APP_URL` / `FRONTEND_URL` kosong di `.env` — `config.Validate()` menolak start.                                                                                        |
| `.env file not found` dari `make migrate`           | jalankan dari dalam `back/`, dan pastikan `.env` sudah dicopy dari `.env.example`.                                                                                      |
| Migrate/seed gagal connect                          | PostgreSQL belum jalan (`systemctl status postgresql`) atau `DB_*` salah.                                                                                               |
| Frontend jalan tapi semua data kosong / error fetch | `NEXT_PUBLIC_API_URL` tidak cocok dengan port backend. Env `NEXT_PUBLIC_*` dibaca saat build/start — restart `npm run dev` setelah mengubahnya.                         |
| HTTP 429                                            | kena rate limiter: login 5 req/menit, aksi sensitif (ganti password, mint API key, rotate webhook secret) 10 req/menit, endpoint publik 120 req/menit. Tunggu sebentar. |
| Payment sandbox tidak pernah `paid`                 | perilaku yang benar — lihat bagian 7.                                                                                                                                   |
| `sandbox: amount must be at least 2000`             | nominal minimum sandbox adalah 2000.                                                                                                                                    |
| Webhook balas signature invalid                     | body yang di-HMAC harus **persis** byte yang dikirim (jangan re-format JSON-nya), dan `CASHI_SECRET_KEY` harus sama dengan yang di `back/.env`.                         |
| `air not found`                                     | `go install github.com/air-verse/air@latest`, pastikan `$(go env GOPATH)/bin` di PATH — atau pakai `make run` saja.                                                     |

### Reset data

```bash
cd back
make migrate && make seed      # seed idempoten (upsert), aman diulang
```

Reset total: drop database, lalu `make db-create && make migrate && make seed`.
