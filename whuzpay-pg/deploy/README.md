# Deploy whuzpay-pg ke VPS

whuzpay-pg (payment gateway aggregator, provider "gopay" → memanggil
gopay-notifications yang sudah jalan di VPS yang sama) dideploy ke **VPS
yang sama** dengan gopay-notifications (`whuzpay.com`/`vendor.whuzpay.com`).
Domain: **`pg.whuzpay.com`**.

Postgres, Caddy, Node.js 20 LTS, dan akses SSH **sudah terpasang** dari
setup gopay-notifications (`backend/deploy/README.md` §"Sekali di awal")
— tidak perlu dipasang ulang. Panduan ini cuma menambahkan bagian yang
baru: role Postgres, user sistem, direktori, unit systemd, dan blok
Caddy untuk `pg.whuzpay.com`.

| | whuzpay-pg |
|---|---|
| Domain | `pg.whuzpay.com` |
| Port backend | `127.0.0.1:8082` |
| Port frontend | `127.0.0.1:3020` |
| Direktori | `/opt/whuzpay-pg` (`/opt/whuzpay-pg/front` untuk frontend) |
| User sistem | `whuzpay-pg` |
| Database | `pg_aggregator` (role Postgres `whuzpay_pg`, **beda** dari `gopay`/`gopay_uat`) |
| `GOPAY_BASE_URL` | `http://127.0.0.1:8080` — loopback ke backend gopay-notifications **yang sama**, bukan lewat `https://whuzpay.com` publik (satu mesin, tidak perlu round-trip keluar) |

**DNS**: record `A` untuk subdomain `pg` mengarah ke IP VPS — kalau sudah
ditambahkan (sesuai pesanmu), **pastikan proxy Cloudflare OFF** ("DNS
only", awan abu-abu) untuk record itu bila domain dikelola Cloudflare,
persis jebakan yang sama dengan `vendor.whuzpay.com` dulu (lihat
`backend/deploy/README.md` §"Jebakan yang pernah terjadi"). Kalau proxy
menyala, Caddy gagal menerbitkan sertifikat HTTPS otomatis.

## Sekali di awal

Dari laptop:

```bash
export KEY=~/vps-aws-trial.pem
export NEW=ubuntu@13.60.252.148
export REPO=~/Kerjaan/personal/gopay-notifications
cd "$REPO" && git pull
ssh -i "$KEY" "$NEW" 'uname -m'   # lalu export GOARCH=amd64 atau arm64 sesuai jawabannya
```

1. **User sistem dan direktori** (SSH ke VPS):

   ```bash
   sudo useradd --system --home /opt/whuzpay-pg --shell /usr/sbin/nologin whuzpay-pg
   sudo mkdir -p /opt/whuzpay-pg/front
   sudo chown -R whuzpay-pg:whuzpay-pg /opt/whuzpay-pg
   ```

2. **Role dan database Postgres** — role terpisah dari `gopay`/`gopay_uat`,
   di instance Postgres **yang sama** (native, bukan Docker — Docker cuma
   dipakai untuk `gopay_test`/`gopay_dev` di laptop):

   ```bash
   sudo -u postgres createuser whuzpay_pg --pwprompt
   sudo -u postgres createdb pg_aggregator --owner whuzpay_pg
   ```

3. **File `.env`** di `/opt/whuzpay-pg/.env` (buat manual dengan `nano`, isi
   persis seperti di bawah, ganti `PASSWORD_DB` dengan password yang barusan
   diketik di `createuser`, dan `JWT_SECRET` dengan hasil
   `openssl rand -base64 32`):

   ```bash
   sudo nano /opt/whuzpay-pg/.env
   ```

   ```
   APP_NAME=whuzpay-pg
   APP_ENV=production
   APP_PORT=8082
   APP_URL=https://pg.whuzpay.com

   FRONTEND_URL=https://pg.whuzpay.com

   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=whuzpay_pg
   DB_PASSWORD=PASSWORD_DB
   DB_NAME=pg_aggregator
   DB_SSLMODE=disable

   REDIS_HOST=localhost
   REDIS_PORT=6379
   REDIS_PASSWORD=

   GOPAY_BASE_URL=http://127.0.0.1:8080

   JWT_SECRET=HASIL_openssl_rand_base64_32

   LOG_LEVEL=info
   ```

   ```bash
   sudo chmod 600 /opt/whuzpay-pg/.env
   sudo chown whuzpay-pg:whuzpay-pg /opt/whuzpay-pg/.env
   ```

   **Password Postgres jangan pakai simbol `@ : / ? # %`** — jebakan yang
   sama dengan gopay-notifications, karena beberapa tool di sini menyusun
   connection string dari nilai-nilai ini secara langsung.

4. **Migrasi database.** Tidak seperti gopay-notifications (`goose`),
   whuzpay-pg pakai `scripts/migrate.sh` sendiri (baca `.env` LOKAL di
   `whuzpay-pg/back/`, bukan `.env` VPS) — jadi migrasi dijalankan lewat
   terowongan SSH yang sama seperti U1 backend gopay-notifications:

   ```bash
   cd "$REPO/whuzpay-pg/back"
   cp .env.example .env   # kalau belum ada .env lokal untuk keperluan migrate.sh ini
   ```

   Edit `.env` LOKAL ini (bukan yang di VPS) supaya `DB_*`-nya menunjuk ke
   terowongan, lalu jalankan:

   ```bash
   ssh -i "$KEY" -f -N -L 15432:127.0.0.1:5432 "$NEW"
   ```

   Isi sementara di `whuzpay-pg/back/.env` (laptop):
   ```
   DB_HOST=localhost
   DB_PORT=15432
   DB_USER=whuzpay_pg
   DB_PASSWORD=PASSWORD_DB
   DB_NAME=pg_aggregator
   DB_SSLMODE=disable
   ```

   ```bash
   ./scripts/migrate.sh
   pkill -f "15432:127.0.0.1:5432"
   ```

   Hasil yang benar: `Migrations completed (applied: <n>, skipped: 0)`.
   **Jangan jalankan `scripts/seed.sh` di produksi** — isinya 50 merchant
   demo + akun admin/merchant dengan password contoh yang sudah publik di
   repo ini (lihat `whuzpay-pg/back/seeds/002_admin_user.sql` dan
   `006_merchant_user.sql`), cuma untuk dev lokal.

5. **Unit systemd:**

   ```bash
   cd "$REPO/whuzpay-pg"
   scp -i "$KEY" deploy/whuzpay-pg-backend.service deploy/whuzpay-pg-frontend.service "$NEW":/tmp/
   ssh -i "$KEY" "$NEW" 'sudo mv /tmp/whuzpay-pg-backend.service /tmp/whuzpay-pg-frontend.service /etc/systemd/system/ \
     && sudo systemctl daemon-reload \
     && sudo systemctl enable whuzpay-pg-backend whuzpay-pg-frontend'
   ```

6. **Blok Caddy** untuk `pg.whuzpay.com` — tambahkan ke `/etc/caddy/Caddyfile`
   yang sudah ada di VPS (jangan timpa blok `whuzpay.com`/`vendor.whuzpay.com`
   yang sudah ada):

   ```bash
   ssh -i "$KEY" -t "$NEW" 'sudo nano /etc/caddy/Caddyfile'
   ```

   Tambahkan di akhir file:

   ```caddyfile
   pg.whuzpay.com {
   	encode zstd gzip

   	@api path /api/*
   	handle @api {
   		reverse_proxy 127.0.0.1:8082
   	}

   	handle {
   		reverse_proxy 127.0.0.1:3020
   	}

   	log {
   		output file /var/log/caddy/pg-whuzpay.log
   		format json
   	}
   }
   ```

   ```bash
   ssh -i "$KEY" "$NEW" 'sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy'
   ```

Sekarang lanjut ke §"Deploy pertama kali" di bawah (build + upload + start) —
langkahnya SAMA dengan "Update rutin" seterusnya.

## Update rutin (dan deploy pertama kali)

Build tetap di laptop, seperti gopay-notifications — VPS tidak pernah
menjalankan `go build` atau `npm install`.

### Backend

```bash
cd "$REPO/whuzpay-pg/back"
GOOS=linux GOARCH=$GOARCH go build -o /tmp/deploy/whuzpay-pg-server ./cmd/api
scp -i "$KEY" /tmp/deploy/whuzpay-pg-server "$NEW":/tmp/server

ssh -i "$KEY" "$NEW" 'cd /opt/whuzpay-pg \
  && ([ -f server ] && sudo cp server server.prev || true) \
  && sudo mv /tmp/server server \
  && sudo chown whuzpay-pg:whuzpay-pg server && sudo chmod 755 server \
  && sudo systemctl restart whuzpay-pg-backend \
  && sleep 3 && systemctl is-active whuzpay-pg-backend'
```

Hasil yang benar: `active`.

### Frontend

```bash
cd "$REPO/whuzpay-pg/front"
npm ci
npx next build
cp -r public .next/standalone/
cp -r .next/static .next/standalone/.next/
ls .next/standalone/server.js

tar -C .next/standalone -czf - . | ssh -i "$KEY" "$NEW" 'rm -rf /tmp/whuzpay-pg-front && mkdir -p /tmp/whuzpay-pg-front && tar -xzf - -C /tmp/whuzpay-pg-front'

ssh -i "$KEY" "$NEW" 'cd /opt/whuzpay-pg \
  && ([ -d front ] && sudo rm -rf front.prev && sudo mv front front.prev || true) \
  && sudo mv /tmp/whuzpay-pg-front front \
  && sudo chown -R whuzpay-pg:whuzpay-pg front \
  && sudo systemctl restart whuzpay-pg-frontend \
  && sleep 3 && systemctl is-active whuzpay-pg-frontend'
```

Hasil yang benar: `active`.

### Migrasi baru

Kalau ada file baru di `whuzpay-pg/back/migrations/`, ulangi langkah 4 di
§"Sekali di awal" (terowongan SSH + `scripts/migrate.sh`) **sebelum**
restart backend.

### Verifikasi

```bash
curl -s https://pg.whuzpay.com/api/v1/health; echo
curl -s -o /dev/null -w 'frontend: %{http_code}\n' https://pg.whuzpay.com/login
```

Hasil yang benar: body health mengandung `"status"` (cek isinya, format
persis lihat `cmd/api/main.go` handler `/health`), `frontend: 200`.

Uji nyambungnya ke gopay-notifications: buat account gopay-notifications
kalau belum ada (Vendor Dashboard), upload QRIS + buat API key + buat
webhook endpoint mengarah ke `https://pg.whuzpay.com/api/v1/provider-webhooks/gopay`
(dari luar VPS sekalipun ini sebenarnya loopback-ke-loopback di satu
mesin — webhook keluar dari gopay-notifications lewat proses worker-nya
sendiri, jadi URL yang didaftarkan harus tetap alamat publik
`pg.whuzpay.com`, BUKAN `127.0.0.1`, supaya Caddy bisa merutekannya balik
ke port `8082`). Lalu login merchant di `https://pg.whuzpay.com/login`,
tempel API key + webhook secret di Settings, buat payment (environment
**production**), scan QR sungguhan.

### Rollback

```bash
# Backend
ssh -i "$KEY" "$NEW" 'cd /opt/whuzpay-pg && sudo mv server.prev server && sudo systemctl restart whuzpay-pg-backend; sudo journalctl -u whuzpay-pg-backend -n 30 --no-pager'

# Frontend
ssh -i "$KEY" "$NEW" 'cd /opt/whuzpay-pg && sudo rm -rf front && sudo mv front.prev front && sudo systemctl restart whuzpay-pg-frontend; sudo journalctl -u whuzpay-pg-frontend -n 30 --no-pager'
```

## Kalau ada error

Sama daftar jebakan dengan `backend/deploy/README.md` §"Kalau ada error"
(terowongan bentrok, `GOARCH` salah, dst) berlaku sama persis di sini.
Tambahan yang spesifik whuzpay-pg:

| Gejala | Penyebab | Solusi |
|---|---|---|
| Backend tidak `active`, log `APP_URL is required`/`FRONTEND_URL is required`/dst | `.env` VPS belum lengkap | `sudo nano /opt/whuzpay-pg/.env`, lengkapi, `sudo systemctl restart whuzpay-pg-backend` |
| Payment gagal dibuat, error "merchant belum mengatur kredensial" padahal sudah diisi | `GOPAY_BASE_URL` salah (masih default `https://whuzpay.com` alih-alih `http://127.0.0.1:8080`, atau port backend gopay-notifications sungguhan beda) | Cek `sudo grep GOPAY_BASE_URL /opt/whuzpay-pg/.env`, cek juga backend gopay-notifications produksi memang di port `8080` |
| Webhook tidak pernah sampai (payment tetap `pending` walau sudah dibayar) | URL webhook yang didaftarkan di gopay-notifications salah (mis. pakai `127.0.0.1` alih-alih `pg.whuzpay.com`, atau lupa akhiran `/api/v1/provider-webhooks/gopay`) | Cek ulang URL di halaman Webhooks gopay-notifications; test manual `curl -X POST https://pg.whuzpay.com/api/v1/provider-webhooks/gopay -d '{}'` harus dijawab (bukan connection refused/timeout) |
