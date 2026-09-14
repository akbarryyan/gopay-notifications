# flow-test

Kumpulan script `curl` buat nyoba flow WhuzPay (login → create payment →
webhook → cek status) langsung ke backend lokal, tanpa nyentuh kode di
`back/` atau `front/`.

## Prasyarat

- `jq` dan `openssl` sudah terpasang.
- Backend `back/` sudah jalan lokal, sudah migrate + seed:
  ```
  cd back
  make db-create   # sekali saja
  make migrate
  make seed
  make run          # http://localhost:8080
  ```

## Setup (sekali saja)

```
cd flow-test
cp .env.example .env
chmod +x *.sh lib/*.sh
```

Nilai default di `.env.example` sudah cocok untuk setup lokal standar
(admin seed + merchant demo). `CASHI_SECRET_KEY` **tidak perlu diisi manual**
— otomatis dibaca dari `back/.env` tiap script jalan, jadi selalu sinkron
walau di-rotate. Isi `CASHI_SECRET_KEY` di `flow-test/.env` hanya kalau mau
override (misal testing ke backend remote/staging, bukan lokal).

## Pakai

Flow dasar, sandbox — create payment + cek status (status akan tetap
`pending`, karena payment sandbox memang tidak bisa di-webhook):

```
./run_all.sh
```

Flow penuh sampai `paid`, production — create payment via Cashi asli,
kirim webhook simulasi, verifikasi status jadi `paid`:

```
./run_all.sh production
```

> Mode production bikin request asli dari server `back/` ke Cashi API,
> butuh `CASHI_API_KEY`/`CASHI_SECRET_KEY` valid di `back/.env`.

## Jalankan step satuan

State (token, api key, payment id) disimpan di `.state/`, jadi step
bisa diulang satu-satu:

```
./02_admin_login.sh
./03_create_api_key.sh sandbox   # atau production
./04_create_payment.sh
./05_webhook_settle.sh           # hanya untuk mode production
./06_check_status.sh
```

Reset state: `rm -rf .state/*`
