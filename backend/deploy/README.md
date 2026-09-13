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
   `DEVICE_SECRET_KEY`, `ADMIN_SESSION_KEY`, dan `WEBHOOK_SECRET_KEY`
   sama-sama dihasilkan dengan `go run ./cmd/devicetool -genkey` — jalankan
   tiga kali untuk tiga nilai yang berbeda, jangan memakai hasil yang sama
   untuk lebih dari satu. `LICENSE_KEY` boleh dikosongkan dulu di langkah
   ini — instalasi tetap menyala tanpanya, cuma `402` di endpoint
   device/admin/API key sampai lisensi diaktifkan (lihat §"Lisensi" di
   bawah).

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

## Membuat akun admin dashboard

Di VPS, interaktif — akan meminta password diketik dua kali tanpa ditampilkan:

```bash
cd /opt/gopay-ingestion
sudo -u gopay env $(cat .env | xargs) ./admintool -username admin
```

Password dapat diganti kapan saja dengan menjalankan perintah yang sama lagi.
Secret tidak akan ditampilkan lagi.

## Lisensi

Rancangan lengkap di
[`docs/superpowers/specs/2026-09-13-online-license-platform-design.md`](../../docs/superpowers/specs/2026-09-13-online-license-platform-design.md)
(mengikuti [`docs/license-spec.md`](../../docs/license-spec.md), spec bisnis
otoritatif). Model lama berbasis file lisensi offline yang ditandatangani
manual lewat `licensetool` (spec
[`2026-09-13-license-system-design.md`](../../docs/superpowers/specs/2026-09-13-license-system-design.md),
SUPERSEDED) sudah **diganti total**, bukan sekadar diperluas — `licensetool`
sudah dihapus dari repo.

Sekarang ada dua License Server dan Vendor Dashboard yang **hanya di-deploy
sekali, milik vendor (Akbar), terpisah total dari instalasi customer
manapun** — termasuk dari instalasi Akbar sendiri sebagai customer pertama
di `whuzpay.com`. Tanpa lisensi aktif/akan-berakhir, backend customer tetap
menyala tapi `requireLicense` menolak `402` seluruh endpoint device/admin/API
key — hanya login dan halaman `/license` dashboard (read-only) yang tetap
bisa dibuka.

### A. Deploy License Server + Vendor Dashboard (sekali, infra vendor)

Bedanya dari instalasi customer di atas: user sistem, direktori, database,
dan domain sendiri — jangan dicampur dengan `/opt/gopay-ingestion`.

1. User sistem dan direktori:

   ```bash
   sudo useradd --system --home /opt/gopay-license --shell /usr/sbin/nologin gopay-license
   sudo mkdir -p /opt/gopay-license
   sudo chown gopay-license:gopay-license /opt/gopay-license
   ```

2. Database Postgres terpisah (`gopay_license`, bukan `gopay`/`gopay_uat`):

   ```bash
   sudo -u postgres createuser gopay_license --pwprompt
   sudo -u postgres createdb gopay_license --owner gopay_license
   goose -dir migrations-license postgres "$DATABASE_URL" up
   ```

3. Hasilkan `ADMIN_SESSION_KEY` (sesi dashboard vendor) dan pasangan kunci
   penanda tangan Ed25519 — **pasangan ini beda dari tiga kunci backend
   customer manapun**, dan private key-nya HANYA pernah ada di `.env`
   License Server, tidak pernah di laptop atau di-commit:

   ```bash
   go run ./cmd/licenseserver -genkey
   # cetak: ADMIN_SESSION_KEY, LICENSE_SIGNING_PRIVATE_KEY, LICENSE_SIGNING_PUBLIC_KEY
   ```

   Tempel `LICENSE_SIGNING_PUBLIC_KEY` ke konstanta
   `licensePublicKeyBase64` di `internal/licensecheck/license.go`, commit —
   **seluruh instalasi customer memakai public key yang sama ini** untuk
   memverifikasi tanda tangan state lisensi lokalnya. Kalau public key
   berubah (rotasi darurat), setiap instalasi customer wajib ikut rebuild
   dengan binary baru.

4. Isi `/opt/gopay-license/.env`:

   ```env
   DATABASE_URL=postgres://gopay_license:GANTI_PASSWORD@localhost:5432/gopay_license?sslmode=disable
   LISTEN_ADDR=127.0.0.1:8095
   ADMIN_SESSION_KEY=<dari -genkey>
   LICENSE_SIGNING_PRIVATE_KEY=<dari -genkey>
   ```

5. Pasang unit systemd:

   ```bash
   sudo cp deploy/gopay-licenseserver.service /etc/systemd/system/
   sudo cp deploy/gopay-vendor-dashboard.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable gopay-licenseserver gopay-vendor-dashboard
   ```

6. Build dan kirim `licenseserver` seperti `server`/`devicetool` di atas,
   lalu `vendor-dashboard` dengan cara yang sama seperti `dashboard/`
   (`output: "standalone"`, `npx next build`, salin `public/`+`.next/static`
   manual):

   ```bash
   GOOS=linux GOARCH=amd64 go build -o licenseserver ./cmd/licenseserver
   scp licenseserver VPS:/opt/gopay-license/
   ssh VPS 'sudo chown gopay-license:gopay-license /opt/gopay-license/licenseserver \
     && sudo chmod 755 /opt/gopay-license/licenseserver'

   cd ../vendor-dashboard
   npm ci && npx next build
   cp -r public .next/standalone/
   cp -r .next/static .next/standalone/.next/
   scp -r .next/standalone/. VPS:/tmp/vendor-dashboard/
   ssh VPS 'sudo rm -rf /opt/gopay-license/vendor-dashboard \
     && sudo mv /tmp/vendor-dashboard /opt/gopay-license/vendor-dashboard \
     && sudo chown -R gopay-license:gopay-license /opt/gopay-license/vendor-dashboard'

   ssh VPS 'sudo systemctl start gopay-licenseserver gopay-vendor-dashboard'
   ```

7. Tambahkan blok baru di `Caddyfile` untuk subdomain vendor (mis.
   `license.whuzpay.com`), sama polanya dengan blok domain customer:
   `/api/*` ke `licenseserver` (port 8095), sisanya ke `vendor-dashboard`
   (port 3010):

   ```caddyfile
   license.whuzpay.com {
   	encode zstd gzip

   	@api path /api/*
   	handle @api {
   		reverse_proxy 127.0.0.1:8095
   	}

   	handle {
   		reverse_proxy 127.0.0.1:3010
   	}

   	log {
   		output file /var/log/caddy/gopay-license.log
   		format json
   	}
   }
   ```

   `sudo systemctl reload caddy` setelahnya.

8. Buat akun admin vendor (interaktif, sekali per admin):

   ```bash
   cd /opt/gopay-license
   sudo -u gopay-license env $(cat .env | xargs) ./licenseserver -create-admin akbar
   ```

   Login di `https://license.whuzpay.com`.

### B. Menerbitkan/memperpanjang lisensi customer (tiap customer, lewat Vendor Dashboard)

Tidak ada lagi CLI atau file `.lic` yang di-scp. Di Vendor Dashboard:

1. Buat customer (nama + kontak) kalau belum ada.
2. Di halaman customer, buat license — pilih plan (Starter/Business/
   Enterprise, menentukan `max_devices`) dan tanggal kedaluwarsa. Dashboard
   menampilkan **License Key sekali saja** saat dibuat (format
   `PB-<PLAN>-XXXX-XXXX-XXXX`) — catat sebelum menutup dialog, tidak bisa
   dilihat ulang (yang tersimpan di database cuma hash-nya).

3. Kirim License Key ke customer. Di sisi customer (`/opt/gopay-ingestion/.env`
   atau `.env.uat`), isi:

   ```env
   LICENSE_KEY=PB-BUSINESS-XXXX-XXXX-XXXX
   ENVIRONMENT=production
   LICENSE_SERVER_URL=https://license.whuzpay.com
   ```

   lalu `sudo systemctl restart gopay-ingestion` (atau `gopay-ingestion-uat`
   untuk `ENVIRONMENT=uat`). Backend mengaktivasi otomatis saat start —
   membuat baris `installations` baru yang terikat ke license ini, lalu
   memvalidasi ulang tiap 24 jam. Tidak ada langkah manual lain di sisi
   customer.

4. Memperpanjang: buka license di Vendor Dashboard, klik Renew, isi tanggal
   kedaluwarsa baru. Tidak perlu mengirim apa pun ke customer atau restart
   backend mereka — validasi 24-jam berikutnya otomatis mengambil tanggal
   baru. Suspend/Revoke bekerja sama: efeknya baru terlihat di customer
   setelah siklus validasi berikutnya (atau setelah grace period 7 hari
   habis, kalau License Server sedang tidak terjangkau customer itu).

5. Reset installation (mis. customer pindah server/reinstall): buka license
   di Vendor Dashboard, reset installation yang lama supaya kuota
   `max_devices` terbuka lagi. Aktivasi berikutnya dari `.env` yang sama
   akan membuat installation baru.

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

Satu pemeriksaan lagi khusus lisensi, setelah `LICENSE_KEY` diisi dan
diaktivasi (lihat §"Lisensi" di atas): buka halaman `/license` di dashboard
(login dulu) dan pastikan statusnya `Aktif` dengan detail customer/plan yang
benar. Kalau `LICENSE_KEY` masih kosong di tahap ini, itu diharapkan —
halaman akan menunjukkan status `Belum Aktif` dan endpoint lain menjawab
`402` sampai lisensinya diaktivasi.

Buka `https://GANTI-DOMAIN.com/login` di browser sungguhan dan coba login dengan
akun yang dibuat lewat `admintool` (§"Membuat akun admin dashboard" di
atas) untuk verifikasi penuh.
