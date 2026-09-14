# Deploy ke VPS

Dua lingkungan berjalan di VPS yang sama, terpisah penuh:

| | Produksi | UAT |
|---|---|---|
| Domain | `GANTI-DOMAIN.com` | `uat.GANTI-DOMAIN.com` |
| Port | `127.0.0.1:8080` | `127.0.0.1:8081` |
| Direktori | `/opt/gopay-ingestion` | `/opt/gopay-ingestion-uat` |
| User sistem | `gopay` | `gopay-uat` |
| Database | `gopay` | `gopay_uat` |
| `DEVICE_SECRET_KEY` | sendiri | **berbeda**, jangan dipakai ulang |
| Aplikasi Android | `id.akbarryyan.gopaybridge` | `id.akbarryyan.gopaybridge.uat` |

Device terdaftar **per database**. Karena itu HP UAT yang salah diarahkan ke
backend produksi akan ditolak `401 invalid_signature` — secret-nya tidak ada di
tabel `devices` produksi. Gagalnya nyaring, bukan diam-diam, dan itu properti
keamanan yang paling penting dari pemisahan ini.

Memakai `DEVICE_SECRET_KEY` yang sama di kedua lingkungan menghapus jaminan itu.
Jangan.

Panduan di bawah ditulis untuk produksi. Untuk UAT, ganti setiap nama sesuai
tabel di atas dan pakai `.env.uat.example` serta
`deploy/gopay-ingestion-uat.service`.


## Sekali di awal

1. Pasang PostgreSQL 16, Caddy, dan **Node.js 20 LTS** (dipakai untuk
   menjalankan dashboard — Next.js 16 butuh Node 20+):

   ```bash
   curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
   sudo apt-get install -y nodejs
   node -v   # pastikan v20.x atau lebih baru
   ```

   **Jangan pakai Caddy dari repo Ubuntu/Debian bawaan (`apt install caddy`
   polos).** Versinya sering tertinggal jauh — `2.6.2` dari repo distro
   tidak mengenali directive `basic_auth` di Caddyfile ini sama sekali
   (gagal `unrecognized directive: basic_auth`). Pasang dari repo resmi:

   ```bash
   sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
   curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
   curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
   sudo apt update
   sudo apt install -y caddy
   ```

   Kalau domain sudah dikelola Cloudflare, matikan proxy ("awan oranye" →
   abu-abu / "DNS only") pada record `A`-nya sebelum lanjut. Dengan proxy
   aktif, `dig` domain akan menunjuk ke IP Cloudflare, bukan IP VPS ini, dan
   Caddy gagal menerbitkan sertifikat HTTPS otomatis karena tantangan ACME
   (HTTP-01) mendarat di edge Cloudflare, bukan di origin.

2. Buat user sistem dan direktori:

   ```bash
   sudo useradd --system --home /opt/gopay-ingestion --shell /usr/sbin/nologin gopay
   sudo mkdir -p /opt/gopay-ingestion
   sudo chown gopay:gopay /opt/gopay-ingestion
   ```

3. Buat database dan user Postgres:

   ```bash
   sudo -u postgres createuser gopay --pwprompt
   sudo -u postgres createdb gopay --owner gopay
   ```

4. Salin `.env.example` ke `/opt/gopay-ingestion/.env`, isi seluruh nilainya.
   `DEVICE_SECRET_KEY`, `ADMIN_SESSION_KEY`, `WEBHOOK_SECRET_KEY`,
   `VENDOR_SESSION_KEY`, dan `SETTINGS_SECRET_KEY` sama-sama dihasilkan
   dengan `go run ./cmd/devicetool -genkey` — jalankan lima kali untuk lima
   nilai yang berbeda, jangan memakai hasil yang sama untuk lebih dari
   satu. **Saat pindah VPS, kelima kunci wajib disalin persis dari server
   lama** — kunci baru membuat secret device, secret webhook, dan password
   SMTP yang tersimpan di database tidak bisa didekripsi lagi. `VENDOR_SESSION_KEY` dipakai endpoint `/api/v1/vendor/*`
   (Vendor Dashboard, lihat §"Vendor Dashboard & account customer" di
   bawah) — wajib diisi walau instalasi ini tidak menjalankan Vendor
   Dashboard-nya sendiri.

   ```bash
   sudo chmod 600 /opt/gopay-ingestion/.env
   sudo chown gopay:gopay /opt/gopay-ingestion/.env
   ```

5. Pasang unit systemd untuk backend **dan** dashboard:

   ```bash
   sudo cp deploy/gopay-ingestion.service /etc/systemd/system/
   sudo cp deploy/gopay-dashboard.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable gopay-ingestion gopay-dashboard
   ```

6. Isi `Caddyfile` dengan domain sungguhan dan hash basic auth
   (`caddy hash-password`), lalu salin ke `/etc/caddy/Caddyfile` dan
   `sudo systemctl reload caddy`. Caddyfile yang sudah disiapkan merutekan
   `/api/*` ke backend (port 8080) dan sisanya ke dashboard (port 3000) —
   satu domain, tanpa CORS, persis seperti yang diasumsikan
   `dashboard/README.md`.

## Tiap rilis — backend

```bash
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o devicetool ./cmd/devicetool
GOOS=linux GOARCH=amd64 go build -o admintool ./cmd/admintool
scp server devicetool admintool VPS:/tmp/
ssh VPS 'sudo systemctl stop gopay-ingestion \
  && sudo mv /tmp/server /tmp/devicetool /tmp/admintool /opt/gopay-ingestion/ \
  && sudo chown gopay:gopay /opt/gopay-ingestion/server /opt/gopay-ingestion/devicetool /opt/gopay-ingestion/admintool \
  && sudo chmod 755 /opt/gopay-ingestion/server /opt/gopay-ingestion/devicetool /opt/gopay-ingestion/admintool \
  && sudo systemctl start gopay-ingestion'
```

Migrasi dijalankan terpisah:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

## Tiap rilis — dashboard

`output: "standalone"` di `dashboard/next.config.ts` membuat `next build`
menghasilkan server Node yang berdiri sendiri di `.next/standalone` —
lengkap dengan subset `node_modules` yang benar-benar dipakai. VPS tidak
pernah perlu `npm install`.

```bash
cd dashboard
npm ci
npx next build

# .next/standalone TIDAK menyertakan aset statis (public/, .next/static)
# secara default — disalin manual ke dalamnya sebelum dikirim.
cp -r public .next/standalone/
cp -r .next/static .next/standalone/.next/

scp -r .next/standalone/. VPS:/tmp/dashboard/
ssh VPS 'sudo systemctl stop gopay-dashboard \
  && sudo rm -rf /opt/gopay-ingestion/dashboard \
  && sudo mv /tmp/dashboard /opt/gopay-ingestion/dashboard \
  && sudo chown -R gopay:gopay /opt/gopay-ingestion/dashboard \
  && sudo systemctl start gopay-dashboard'
```

Dashboard produksi **tidak butuh** `.env.local` atau `BACKEND_URL` sama
sekali — itu cuma dipakai `next dev` di laptop. Di belakang Caddy, dashboard
tidak pernah memanggil backend lewat rewrite-nya sendiri; browser yang
memanggil `/api/*` langsung, dan Caddy yang merutekannya ke backend.

## Membuat device untuk HP

Di VPS:

```bash
cd /opt/gopay-ingestion
sudo -u gopay env $(cat .env | xargs) ./devicetool -name "HP GoPay Utama"
```

Salin `Device ID` dan `Device Secret` ke Settings aplikasi Android.

## Membuat akun vendor

`admintool` membuat/mereset akun **vendor** (Akbar), dipakai login ke
Vendor Dashboard — **bukan** akun customer (lihat §"Vendor Dashboard &
account customer" di bawah untuk itu). Di VPS, interaktif — akan meminta
password diketik dua kali tanpa ditampilkan:

```bash
cd /opt/gopay-ingestion
sudo -u gopay env $(cat .env | xargs) ./admintool -username akbar
```

Password dapat diganti kapan saja dengan menjalankan perintah yang sama lagi.
Secret tidak akan ditampilkan lagi.

## Vendor Dashboard & account customer

**PIVOT ARSITEKTUR (2026-09-13):** produk ini sekarang hosted multi-tenant
— satu backend (instalasi di atas) melayani SEMUA customer sekaligus, data
dipisah lewat `account_id` per baris. Rancangan lengkap:
[`docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md`](../../docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md).

Model lama (License Server + Vendor Dashboard sebagai infra terpisah,
`internal/licensecheck`/`internal/licenseclient`, file `.lic`, aktivasi
lewat `LICENSE_KEY` di `.env`) sudah **dibongkar total** — bukan sekadar
diperluas. Vendor Dashboard sekarang cuma app Next.js tambahan yang
memanggil backend instalasi ini (bukan service terpisah dengan database
sendiri), dan account customer dibuat langsung di database yang sama
lewat endpoint `/api/v1/vendor/*` — tidak ada lagi file lisensi, tidak ada
lagi validasi berkala.

### Deploy Vendor Dashboard

Sama persis polanya seperti `dashboard/` di atas (`output: "standalone"`,
`npx next build`, salin `public/`+`.next/static` manual) — bedanya cuma
port dan nama direktori:

```bash
cd vendor-dashboard
npm ci
npx next build
cp -r public .next/standalone/
cp -r .next/static .next/standalone/.next/

scp -r .next/standalone/. VPS:/tmp/vendor-dashboard/
ssh VPS 'sudo systemctl stop gopay-vendor-dashboard \
  && sudo rm -rf /opt/gopay-ingestion/vendor-dashboard \
  && sudo mv /tmp/vendor-dashboard /opt/gopay-ingestion/vendor-dashboard \
  && sudo chown -R gopay:gopay /opt/gopay-ingestion/vendor-dashboard \
  && sudo systemctl start gopay-vendor-dashboard'
```

Unit systemd (sekali di awal, mirip `gopay-dashboard.service` tapi port
`3010`):

```bash
sudo cp deploy/gopay-vendor-dashboard.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable gopay-vendor-dashboard
```

Tambahkan blok `Caddyfile` untuk subdomain vendor (mis.
`vendor.whuzpay.com`) yang merutekan `/api/*` ke backend **yang sama**
(port 8080, BUKAN service terpisah) dan sisanya ke Vendor Dashboard (port
3010):

```caddyfile
vendor.whuzpay.com {
	encode zstd gzip

	@api path /api/*
	handle @api {
		reverse_proxy 127.0.0.1:8080
	}

	handle {
		reverse_proxy 127.0.0.1:3010
	}

	log {
		output file /var/log/caddy/gopay-vendor.log
		format json
	}
}
```

`sudo systemctl reload caddy` setelahnya. Login pakai akun vendor yang
dibuat lewat `admintool` (lihat §"Membuat akun vendor" di atas).

### Membuat account customer baru

Tidak ada lagi CLI atau file `.lic` yang di-scp. Semuanya lewat Vendor
Dashboard:

1. Login ke Vendor Dashboard, klik "Buat akun" — isi nama bisnis, email,
   username, plan (Starter/Business/Enterprise, menentukan `max_devices`),
   dan tanggal kedaluwarsa.
2. Dashboard menampilkan **password awal sekali saja** saat akun dibuat —
   catat sebelum menutup dialog, tidak bisa dilihat ulang (yang tersimpan
   di database cuma hash-nya). Kirim username+password ke customer lewat
   kanal sendiri.
3. Customer login langsung ke `dashboard/` (backend yang sama, tidak ada
   instalasi terpisah untuk mereka) dengan kredensial itu.
4. Buat device untuk account itu:

   ```bash
   cd /opt/gopay-ingestion
   sudo -u gopay env $(cat .env | xargs) ./devicetool -account <account_id> -name "HP Toko"
   ```

   `<account_id>` dilihat dari URL halaman detail account di Vendor
   Dashboard (`/accounts/<account_id>`). Swalayan tambah device dari
   Customer Dashboard belum ada — ditunda ke sub-project terpisah (spec §7).

5. Memperpanjang/suspend/revoke: dari halaman detail account di Vendor
   Dashboard. Efeknya **langsung** terlihat di request berikutnya customer
   itu (status dicek langsung ke database tiap request, tidak ada lagi
   validasi berkala atau grace period).

## Membangun aplikasi Android per varian

```bash
cd mobile
APP_VARIANT=uat        npx expo prebuild --platform android --clean && npx expo run:android
APP_VARIANT=production npx expo prebuild --platform android --clean && npx expo run:android
```

`--clean` wajib saat berpindah varian: package name berubah, dan `android/`
harus dibuat ulang dari nol.

Ketiga varian dapat terpasang bersamaan dan dibedakan dari namanya. Android
menyandera data tiap package di sandbox-nya sendiri, sehingga database event,
Device Secret, dan Settings UAT tidak pernah bercampur dengan produksi.

**Yang perlu disadari:** bila aplikasi UAT dan produksi sama-sama terpasang dan
keduanya diberi Notification Access, **keduanya menangkap setiap pembayaran**.
Database UAT akan memuat catatan transaksi sungguhan. Untuk menguji itu berguna,
tetapi perlakukan datanya dengan serius yang sama. Bila tidak diinginkan, jangan
beri Notification Access ke aplikasi UAT kecuali sedang menguji.

Backend URL sudah ditanam sebagai nilai awal per varian di `mobile/src/lib/env.ts` —
edit konstanta `DOMAIN` di sana sekali saja. Nilainya hanya diisi bila Settings
masih kosong, sehingga perubahan manual tidak pernah ditimpa.

## Verifikasi setelah deploy

Jalankan untuk **kedua** domain:

```bash
D=GANTI-DOMAIN.com          # lalu ulangi dengan D=uat.GANTI-DOMAIN.com

curl -s "https://$D/api/v1/health"
curl -s -o /dev/null -w '%{http_code}\n' "https://$D/api/v1/events"                  # mau 401
curl -s -o /dev/null -w '%{http_code}\n' -u admin:<pw> "https://$D/api/v1/events"    # mau 200
curl -s -o /dev/null -w '%{http_code}\n' "http://$D/api/v1/health"                   # mau 308
```

Satu pemeriksaan tambahan yang membuktikan pemisahannya nyata: ambil Device ID
dan Secret dari **UAT**, lalu coba kirim ke **produksi**. Harus ditolak
`401 invalid_signature`.

Khusus dashboard (produksi saja, UAT belum dipasang):

```bash
curl -s -o /dev/null -w '%{http_code}\n' "https://GANTI-DOMAIN.com/"        # mau 200 atau 307 (redirect ke /login)
curl -s -o /dev/null -w '%{http_code}\n' "https://GANTI-DOMAIN.com/login"   # mau 200
```

`307` untuk `/` berarti `proxy.ts` benar mengalihkan karena belum ada cookie
sesi — itu tanda dashboard-nya sendiri sudah jalan, bukan error.

Satu pemeriksaan lagi khusus account, setelah sebuah account customer
dibuat lewat Vendor Dashboard (lihat §"Vendor Dashboard & account
customer" di atas): login ke `dashboard/` dengan kredensial account itu,
buka halaman `/license`, pastikan statusnya `Aktif` dengan plan/tanggal
kedaluwarsa yang benar. Belum ada account customer sama sekali di tahap
ini itu wajar — seluruh endpoint device/admin/API key menjawab `402`
sampai account pertama dibuat.

Buka `https://GANTI-DOMAIN.com/login` di browser sungguhan dan coba login dengan
akun customer yang dibuat lewat Vendor Dashboard (§"Vendor Dashboard &
account customer" di
atas) untuk verifikasi penuh.
