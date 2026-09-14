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

**Mau update server setelah `git push`?** Langsung ke
[§"Update rutin setelah ada perubahan kode"](#update-rutin-setelah-ada-perubahan-kode).
Bagian "Sekali di awal" cuma untuk memasang VPS baru dari nol.

Server produksi sekarang: `whuzpay.com` + `vendor.whuzpay.com`, AWS EC2
region Stockholm (`eu-north-1`), login `ssh -i ~/vps-aws-trial.pem
ubuntu@13.60.252.148`.


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

### Jebakan yang pernah terjadi saat memasang VPS baru

Semuanya terjadi saat pindah ke AWS pada 2026-09-14:

- **Sertifikat HTTPS gagal dengan `Timeout during connect (likely firewall
  problem)`.** Security Group AWS belum membuka port **80** dan **443**.
  Buka lewat EC2 → Instances → tab Security → Security group → Edit inbound
  rules, dengan Source `0.0.0.0/0` dan `::/0`. Caddy mencoba ulang sendiri,
  atau jalankan `sudo systemctl restart caddy` supaya langsung dicoba.
- **`caddy validate` gagal dengan `base64-decoding password: illegal base64
  data`.** Hash dari `caddy hash-password` (`$2a$14$...`) harus ditulis dalam
  bentuk base64 di Caddyfile:
  `printf '%s' "$HASH" | base64 -w0`. Hasilnya diawali `JDJh`.
- **`systemctl reload caddy` gagal padahal `caddy validate` sukses.**
  `sudo caddy validate` ikut membuat file log di `/var/log/caddy/` sebagai
  root, sehingga service Caddy (user `caddy`) tidak bisa menulisnya.
  Perbaikannya `sudo chown -R caddy:caddy /var/log/caddy`, lalu reload lagi.
- **`vendor.whuzpay.com` tidak bisa diakses.** Record DNS `A` untuk
  subdomain `vendor` harus dibuat sendiri; tidak ikut record domain utama.
- **IP publik EC2 berubah setelah instance di-stop/start.** Pasang Elastic
  IP supaya DNS tidak salah alamat.
- **Password database dengan simbol `@ : / ? # %` merusak `DATABASE_URL`.**
  Pakai huruf dan angka saja.
- **`.env` bisa dibuat langsung di VPS** tanpa Go, karena format
  `openssl rand -base64 32` sama dengan `devicetool -genkey`. Simpan salinan
  `.env` di password manager: kunci-kuncinya wajib dibawa kalau pindah VPS
  lagi.
- **AWS memblokir port 25 keluar.** Untuk SMTP di Vendor Dashboard, pakai
  port **587** atau **465**.

## Update rutin setelah ada perubahan kode

Semua langkah dijalankan **dari satu terminal di laptop**. Perintah ke VPS
dikirim lewat `ssh`, jadi tidak perlu membuka sesi SSH terpisah. Binary dan
dashboard dibangun di laptop, bukan di VPS: VPS tidak pernah butuh Go atau
`npm install`, dan RAM instance kecil tidak cukup untuk `next build`.

### Update bagian mana?

Cukup bagian yang berubah sejak update terakhir. Cek dengan `git log` atau
`git diff --stat <commit-terakhir-yang-di-deploy>..HEAD`.

| Yang berubah | Jalankan |
|---|---|
| `backend/` (Go) | U1 Backend |
| `backend/migrations/` (file `.sql` baru) | U1 Backend, **termasuk langkah migrasi** |
| `dashboard/` | U2 Customer Dashboard |
| `vendor-dashboard/` | U3 Vendor Dashboard |
| `backend/deploy/*.service`, `*.timer`, `gopay-backup.sh` | U4 File deploy |
| `backend/.env.example` (variabel baru) | Tambahkan variabelnya ke `.env` VPS **sebelum** U1, lihat U5 |

Kalau ragu, jalankan U1, U2, dan U3 semuanya. Menjalankan update untuk bagian
yang tidak berubah aman, hanya makan waktu.

### U0. Persiapan (setiap buka terminal baru)

```bash
export KEY=~/vps-aws-trial.pem
export NEW=ubuntu@13.60.252.148
export REPO=~/Kerjaan/personal/gopay-notifications

cd "$REPO" && git pull
ssh -i "$KEY" "$NEW" 'uname -m'
```

Lalu jalankan **salah satu**, sesuai jawaban `uname -m`:

```bash
export GOARCH=amd64    # jawaban x86_64
export GOARCH=arm64    # jawaban aarch64
```

Variabel ini hilang kalau terminal ditutup. Ulangi U0 setiap membuka
terminal baru.

### U1. Backend

**Build dan upload:**

```bash
cd "$REPO/backend"
mkdir -p /tmp/deploy
GOOS=linux go build -o /tmp/deploy/server ./cmd/server
scp -i "$KEY" /tmp/deploy/server "$NEW":/tmp/server
```

Hasil yang benar: satu baris progres `server ... 100%`.

**Migrasi database** (wajib kalau ada file baru di `backend/migrations/`,
aman dijalankan walau tidak ada). Port Postgres VPS tidak dibuka ke
internet, jadi `goose` di laptop tersambung lewat terowongan SSH: port
`15432` laptop diteruskan ke `5432` VPS.

Ganti `PASSWORD_DB` dengan password di baris `DATABASE_URL` pada `.env` VPS
(salinannya ada di `~/whuzpay-production.env` kalau disimpan waktu pasang):

```bash
ssh -i "$KEY" -f -N -L 15432:127.0.0.1:5432 "$NEW"
~/go/bin/goose -dir migrations postgres "postgres://gopay:PASSWORD_DB@127.0.0.1:15432/gopay?sslmode=disable" up
pkill -f "15432:127.0.0.1:5432"
```

Hasil yang benar diakhiri dengan
`goose: successfully migrated database to version: <nomor migrasi terakhir>`,
atau `no migrations to run` kalau tidak ada migrasi baru.

Migrasi dijalankan **sebelum** binary baru dinyalakan: kode baru mungkin
membutuhkan kolom atau tabel baru, sedangkan kode lama tetap jalan dengan
skema yang lebih baru karena migrasi di repo ini selalu menambah, tidak
menghapus.

**Pasang dan restart.** Binary lama disimpan sebagai `server.prev` untuk
rollback:

```bash
ssh -i "$KEY" "$NEW" 'cd /opt/gopay-ingestion \
  && sudo cp server server.prev \
  && sudo mv /tmp/server server \
  && sudo chown gopay:gopay server && sudo chmod 755 server \
  && sudo systemctl restart gopay-ingestion \
  && sleep 3 && systemctl is-active gopay-ingestion'
```

Hasil yang benar: `active`.

Kalau `cmd/devicetool` atau `cmd/admintool` juga berubah, build dan upload
dengan pola yang sama (`go build -o /tmp/deploy/admintool ./cmd/admintool`,
`scp`, lalu `sudo mv` ke `/opt/gopay-ingestion/`). Keduanya bukan service,
jadi tidak perlu restart.

### U2. Customer Dashboard

`output: "standalone"` di `next.config.ts` membuat `next build` menghasilkan
server Node yang berdiri sendiri di `.next/standalone`, lengkap dengan
`node_modules` yang dipakai. Aset statis (`public/`, `.next/static`) tidak
ikut otomatis, jadi disalin manual.

```bash
cd "$REPO/dashboard"
npm ci
npx next build
cp -r public .next/standalone/
cp -r .next/static .next/standalone/.next/
ls .next/standalone/server.js
```

`next build` harus diakhiri tabel daftar route tanpa `Error`, dan `ls` harus
menjawab `.next/standalone/server.js`. Peringatan `npm warn deprecated` dan
`allow-scripts` dari `npm ci` aman diabaikan.

**Upload** sebagai satu arsip. Jangan pakai `scp -r`: `.next/standalone`
berisi ribuan file kecil yang dikirim satu per satu, dan ke region Stockholm
bisa makan belasan menit.

```bash
tar -C .next/standalone -czf - . | ssh -i "$KEY" "$NEW" 'rm -rf /tmp/dashboard && mkdir -p /tmp/dashboard && tar -xzf - -C /tmp/dashboard'
```

Tidak ada output; selesai saat prompt muncul lagi.

**Pasang dan restart.** Versi lama disimpan sebagai `dashboard.prev`:

```bash
ssh -i "$KEY" "$NEW" 'cd /opt/gopay-ingestion \
  && sudo rm -rf dashboard.prev \
  && sudo mv dashboard dashboard.prev \
  && sudo mv /tmp/dashboard dashboard \
  && sudo chown -R gopay:gopay dashboard \
  && sudo systemctl restart gopay-dashboard \
  && sleep 3 && systemctl is-active gopay-dashboard'
```

Hasil yang benar: `active`.

Dashboard produksi tidak butuh `.env.local` atau `BACKEND_URL`; itu hanya
untuk `next dev` di laptop. Di produksi, browser memanggil `/api/*` dan
Caddy yang meneruskannya ke backend.

### U3. Vendor Dashboard

Sama dengan U2, tapi **tanpa** `cp -r public`, karena `vendor-dashboard/`
tidak punya folder `public/` (kalau dijalankan, muncul
`cannot stat 'public'`, dan itu tidak apa-apa).

```bash
cd "$REPO/vendor-dashboard"
npm ci
npx next build
cp -r .next/static .next/standalone/.next/
ls .next/standalone/server.js

tar -C .next/standalone -czf - . | ssh -i "$KEY" "$NEW" 'rm -rf /tmp/vendor-dashboard && mkdir -p /tmp/vendor-dashboard && tar -xzf - -C /tmp/vendor-dashboard'

ssh -i "$KEY" "$NEW" 'cd /opt/gopay-ingestion \
  && sudo rm -rf vendor-dashboard.prev \
  && sudo mv vendor-dashboard vendor-dashboard.prev \
  && sudo mv /tmp/vendor-dashboard vendor-dashboard \
  && sudo chown -R gopay:gopay vendor-dashboard \
  && sudo systemctl restart gopay-vendor-dashboard \
  && sleep 3 && systemctl is-active gopay-vendor-dashboard'
```

Hasil yang benar: `active`.

### U4. File deploy (unit systemd / skrip backup)

Hanya kalau file di `backend/deploy/` berubah. Contoh untuk unit backend:

```bash
cd "$REPO/backend"
scp -i "$KEY" deploy/gopay-ingestion.service "$NEW":/tmp/
ssh -i "$KEY" "$NEW" 'sudo mv /tmp/gopay-ingestion.service /etc/systemd/system/ \
  && sudo systemctl daemon-reload \
  && sudo systemctl restart gopay-ingestion \
  && systemctl is-active gopay-ingestion'
```

Untuk skrip backup:

```bash
scp -i "$KEY" deploy/gopay-backup.sh "$NEW":/tmp/
ssh -i "$KEY" "$NEW" 'sudo install -m 755 /tmp/gopay-backup.sh /usr/local/bin/gopay-backup && rm /tmp/gopay-backup.sh'
```

### U5. Variabel `.env` baru

Kalau `backend/.env.example` mendapat variabel baru, backend baru bisa gagal
start tanpa variabel itu. Tambahkan dulu **sebelum** U1:

```bash
ssh -i "$KEY" -t "$NEW" 'sudo nano /opt/gopay-ingestion/.env'
```

Tambahkan barisnya, simpan dengan `Ctrl+O` lalu `Enter`, dan keluar dengan
`Ctrl+X`. Kunci baru (`..._KEY`) dibuat dengan `openssl rand -base64 32`.
**Jangan pernah mengubah kunci yang sudah ada**, dan perbarui juga salinan
`.env` di password manager.

### U6. Cek setelah update

```bash
curl -s https://whuzpay.com/api/v1/health; echo
curl -s -o /dev/null -w 'dashboard: %{http_code}\n' https://whuzpay.com/login
curl -s -o /dev/null -w 'vendor:    %{http_code}\n' https://vendor.whuzpay.com/login
```

Hasil yang benar: `{"status":"ok",...}`, `dashboard: 200`, `vendor: 200`.
Setelah itu buka halaman yang berubah di browser.

### Rollback: kembali ke versi sebelumnya

Kalau sebuah service tidak `active` atau versi baru bermasalah, kembalikan
dulu supaya situs jalan lagi, baru cari penyebabnya lewat log:

```bash
# Backend
ssh -i "$KEY" "$NEW" 'cd /opt/gopay-ingestion && sudo mv server.prev server && sudo systemctl restart gopay-ingestion; sudo journalctl -u gopay-ingestion -n 30 --no-pager'

# Customer Dashboard
ssh -i "$KEY" "$NEW" 'cd /opt/gopay-ingestion && sudo rm -rf dashboard && sudo mv dashboard.prev dashboard && sudo systemctl restart gopay-dashboard; sudo journalctl -u gopay-dashboard -n 30 --no-pager'

# Vendor Dashboard
ssh -i "$KEY" "$NEW" 'cd /opt/gopay-ingestion && sudo rm -rf vendor-dashboard && sudo mv vendor-dashboard.prev vendor-dashboard && sudo systemctl restart gopay-vendor-dashboard; sudo journalctl -u gopay-vendor-dashboard -n 30 --no-pager'
```

Migrasi database **tidak** ikut dibatalkan oleh rollback. Itu memang tidak
perlu, karena kode lama tetap jalan dengan skema yang lebih baru. Jangan
menjalankan `goose down` di produksi tanpa backup, karena perintah itu bisa
menghapus data.

### Kalau ada error

| Gejala | Penyebab | Solusi |
|---|---|---|
| `bind: Address already in use` saat membuka terowongan | Terowongan lama masih hidup | `pkill -f "15432:127.0.0.1:5432"`, ulangi |
| `password authentication failed for user "gopay"` | `PASSWORD_DB` salah | Lihat baris `DATABASE_URL` di `.env` VPS: `ssh -i "$KEY" "$NEW" 'sudo grep DATABASE_URL /opt/gopay-ingestion/.env'` |
| `KEY=`/`NEW=` kosong, atau `ssh: Could not resolve hostname` | Variabel U0 hilang | Ulangi U0 |
| `ssh: connect to host ... Connection timed out` | IP internet laptop berubah, sedangkan Security Group membatasi SSH ke "My IP" | AWS Console → EC2 → Security Group → Edit inbound rules → rule SSH → Source **My IP** → Save |
| Backend tidak `active`, log berisi `config: ... wajib diisi` | Ada variabel `.env` baru yang belum ditambahkan | Rollback, lakukan U5, ulangi U1 |
| Log berisi `status=203/EXEC` atau `exec format error` | `GOARCH` salah | Rollback, cek `uname -m` di U0, ulangi U1 |
| Dashboard tidak `active`, log berisi `Cannot find module .../server.js` | Upload atau build belum lengkap | Rollback, ulangi U2/U3 dari `npx next build` |
| `scp -r` sangat lambat | Ribuan file kecil dikirim satu per satu | Pakai perintah `tar ... \| ssh` di U2/U3 |

## Backup database otomatis

`deploy/gopay-backup.sh` + `gopay-backup.service` + `gopay-backup.timer`:
`pg_dump` setiap hari pukul 02:00 WIB ke `/var/backups/gopay`, diverifikasi
dengan `pg_restore --list`, backup lokal lebih dari 14 hari dihapus, dan
opsional diunggah ke S3.

**Backup lokal saja tidak cukup** — kalau instance atau disknya hilang,
backup ikut hilang. Isi juga bagian S3 di bawah.

`.env` **sengaja tidak ikut dibackup**. Kunci enkripsi yang disimpan
bersama dump membuat siapa pun yang mendapat backup bisa membuka secret
device, webhook, dan SMTP sekaligus. Simpan `.env` di password manager.

### Sekali di awal

Dari laptop, di folder `backend/`:

```bash
scp -i "$KEY" deploy/gopay-backup.sh deploy/gopay-backup.service deploy/gopay-backup.timer "$NEW":/tmp/
```

Di VPS:

```bash
sudo install -m 755 /tmp/gopay-backup.sh /usr/local/bin/gopay-backup
sudo install -d -o postgres -g postgres -m 700 /var/backups/gopay
sudo mv /tmp/gopay-backup.service /tmp/gopay-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload

# Uji sekali sekarang, jangan tunggu jadwal
sudo systemctl start gopay-backup.service
sudo journalctl -u gopay-backup -n 20 --no-pager     # harus ada "backup: selesai"
sudo ls -lh /var/backups/gopay

# Aktifkan jadwal harian
sudo systemctl enable --now gopay-backup.timer
systemctl list-timers gopay-backup.timer             # kolom NEXT = jadwal berikutnya
```

### Salinan di luar server (S3)

1. **Bucket.** AWS Console → S3 → Create bucket, region sama dengan
   instance, *Block all public access* tetap **aktif**.
2. **Retensi di S3.** Bucket → Management → Create lifecycle rule →
   prefix `gopay/` → *Expire current versions of objects* setelah 30 hari.
3. **Izin untuk instance.** IAM → Roles → Create role → *AWS service: EC2*
   → buat inline policy di bawah (ganti `NAMA_BUCKET`) → beri nama mis.
   `gopay-backup-writer`. Lalu EC2 → Instances → pilih instance → Actions →
   Security → **Modify IAM role** → pilih role tadi.

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{
       "Effect": "Allow",
       "Action": "s3:PutObject",
       "Resource": "arn:aws:s3:::NAMA_BUCKET/gopay/*"
     }]
   }
   ```

   Sengaja cuma `PutObject`, tanpa izin hapus atau baca: server yang
   dibobol tidak boleh bisa menghapus atau mengunduh backup di S3.
4. **AWS CLI** (installer resmi, masuk ke `/usr/local/bin` yang terbaca
   systemd — versi snap tidak):

   ```bash
   sudo apt install -y unzip
   curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" -o /tmp/awscliv2.zip
   unzip -q /tmp/awscliv2.zip -d /tmp && sudo /tmp/aws/install && rm -rf /tmp/aws /tmp/awscliv2.zip
   aws --version
   ```

5. **Konfigurasi dan uji:**

   ```bash
   echo 'BACKUP_S3_URI=s3://NAMA_BUCKET/gopay' | sudo tee /etc/default/gopay-backup
   sudo systemctl start gopay-backup.service
   sudo journalctl -u gopay-backup -n 20 --no-pager   # harus ada "backup: unggah ke s3://..."
   ```

   Pastikan filenya muncul di S3 Console → bucket → folder `gopay/`.

### Mengembalikan dari backup

```bash
sudo systemctl stop gopay-ingestion
sudo -u postgres dropdb gopay
sudo -u postgres createdb gopay --owner gopay
sudo -u postgres pg_restore --no-owner --role=gopay -d gopay /var/backups/gopay/gopay-YYYYMMDDTHHMMSSZ.dump
sudo systemctl start gopay-ingestion
```

`.env` yang dipakai harus berisi kunci yang **sama** dengan saat backup
dibuat. Kalau restore ke VPS baru, jalankan migrasi (`goose up`) setelah
restore bila kode yang di-deploy lebih baru dari backup.

Backup yang tidak pernah dicoba di-restore belum terbukti bisa dipakai —
coba restore ke database uji (`createdb gopay_restore_test`) sesekali.

## Monitoring dan peringatan

Dua lapis, karena masing-masing menutupi celah yang tidak bisa ditutup yang
lain:

| | `gopay-monitor` (di VPS) | UptimeRobot (dari luar) |
|---|---|---|
| Service mati (backend, dashboard, Caddy, Postgres) | ✅ | Sebagian (terlihat dari URL yang tidak menjawab) |
| Disk / memori hampir penuh | ✅ | ❌ |
| Sertifikat HTTPS gagal diperpanjang | ✅ | ✅ (paket gratis: peringatan SSL) |
| Backup harian tidak jalan | ✅ | ❌ |
| **VPS mati total / jaringan putus** | ❌ (ikut mati) | ✅ |

Pasang **keduanya**.

### A. `gopay-monitor` — peringatan ke Telegram kamu

`deploy/gopay-monitor.sh` + `gopay-monitor.service` + `gopay-monitor.timer`
berjalan setiap 5 menit dan hanya mengirim pesan **saat status berubah**:
sekali ketika masalah muncul, diulang setiap 6 jam selama belum beres, dan
sekali saat pulih. Tidak ada spam setiap 5 menit.

Yang diperiksa: service `gopay-ingestion`, `gopay-dashboard`,
`gopay-vendor-dashboard`, `caddy`, `postgresql`; Postgres menerima koneksi;
`https://whuzpay.com/api/v1/health`, `/login`, dan
`https://vendor.whuzpay.com/login` menjawab; sisa masa berlaku sertifikat
kedua domain (peringatan kalau < 14 hari); disk `/` < 85%; memori < 92%;
dan backup terakhir < 26 jam.

**1. Siapkan chat Telegram kamu**

1. Buka bot Telegram yang sama dengan yang diisi di Vendor Dashboard →
   Settings, lalu tekan **Start**. Bot dilarang mengirim pesan duluan ke
   orang yang belum menekan Start. Bot juga boleh berbeda; untuk kirim pesan,
   satu token bisa dipakai bersamaan oleh backend dan skrip ini.
2. Kirim pesan apa saja ke **@userinfobot** di Telegram. Angka **Id** yang
   dibalas adalah `TELEGRAM_CHAT_ID` kamu.

**2. Pasang di VPS** (dari laptop, setelah U0 di §"Update rutin"):

```bash
cd "$REPO/backend"
scp -i "$KEY" deploy/gopay-monitor.sh deploy/gopay-monitor.service deploy/gopay-monitor.timer "$NEW":/tmp/
ssh -i "$KEY" "$NEW" 'sudo install -m 755 /tmp/gopay-monitor.sh /usr/local/bin/gopay-monitor \
  && sudo mv /tmp/gopay-monitor.service /tmp/gopay-monitor.timer /etc/systemd/system/ \
  && sudo systemctl daemon-reload'
```

**3. Isi token dan chat id** (terminal SSH VPS):

```bash
ssh -i ~/vps-aws-trial.pem ubuntu@13.60.252.148
sudo nano /etc/default/gopay-monitor
```

Isi dengan dua baris berikut, ganti nilainya, lalu simpan (`Ctrl+O`, `Enter`,
`Ctrl+X`):

```
TELEGRAM_BOT_TOKEN=123456789:ABCdef...
TELEGRAM_CHAT_ID=123456789
```

```bash
sudo chmod 600 /etc/default/gopay-monitor
```

**4. Uji: pastikan pesan benar-benar sampai.** Paksa satu pemeriksaan gagal
dengan batas disk 1%, yang pasti terlampaui, memakai file state terpisah:

```bash
sudo env $(sudo cat /etc/default/gopay-monitor | xargs) DISK_WARN_PERCENT=1 STATE_FILE=/tmp/monitor-uji /usr/local/bin/gopay-monitor
```

Harus muncul `monitor: peringatan terkirim`, dan Telegram kamu menerima pesan
`🔴 Disk / terpakai ...`. Hapus file ujinya: `sudo rm -f /tmp/monitor-uji`.

**5. Aktifkan jadwal:**

```bash
sudo systemctl enable --now gopay-monitor.timer
sudo systemctl start gopay-monitor.service
sudo journalctl -u gopay-monitor -n 20 --no-pager
systemctl list-timers gopay-monitor.timer --no-pager
```

Log yang benar setelah semuanya sehat: `monitor: tidak ada perubahan (0
masalah aktif)`. Kalau ada baris `masalah: ...`, itu masalah sungguhan yang
juga dikirim ke Telegram.

**Menyesuaikan batas.** Semua variabel di bagian atas
`deploy/gopay-monitor.sh` bisa ditimpa di `/etc/default/gopay-monitor`,
misalnya `DISK_WARN_PERCENT=90`, `REPEAT_HOURS=12`, atau
`URLS="https://whuzpay.com/api/v1/health"`. Tidak perlu restart; perubahan
berlaku di putaran berikutnya.

### B. UptimeRobot — pemantau dari luar (gratis)

1. Daftar di <https://uptimerobot.com> (paket Free: 50 monitor, interval 5
   menit).
2. **Alert Contacts:** Integrations & API → **Telegram** → ikuti
   petunjuknya, atau cukup pakai email yang terdaftar.
3. **Add New Monitor** tiga kali:

   | Monitor Type | Friendly Name | URL | Catatan |
   |---|---|---|---|
   | Keyword | Backend health | `https://whuzpay.com/api/v1/health` | Keyword `"ok"`, alert kalau keyword **tidak ada** |
   | HTTP(s) | Customer Dashboard | `https://whuzpay.com/login` | |
   | HTTP(s) | Vendor Dashboard | `https://vendor.whuzpay.com/login` | |

   Di setiap monitor, centang alert contact dari langkah 2, dan aktifkan
   **SSL expiry reminder** kalau pilihannya ada.
4. **Uji:** di VPS jalankan `sudo systemctl stop gopay-dashboard`, tunggu ±5
   menit sampai UptimeRobot mengabari "Customer Dashboard down" (dan
   `gopay-monitor` juga), lalu `sudo systemctl start gopay-dashboard` dan
   tunggu kabar "up" atau "pulih".

## Membuat device untuk HP

Di VPS:

```bash
cd /opt/gopay-ingestion
sudo -u gopay env $(sudo cat .env | xargs) ./devicetool -account <account_id> -name "HP GoPay Utama"
```

Salin `Device ID` dan `Device Secret` ke Settings aplikasi Android.

## Membuat akun vendor

`admintool` membuat/mereset akun **vendor** (Akbar), dipakai login ke
Vendor Dashboard — **bukan** akun customer (lihat §"Vendor Dashboard &
account customer" di bawah untuk itu). Di VPS, interaktif — akan meminta
password diketik dua kali tanpa ditampilkan:

```bash
cd /opt/gopay-ingestion
sudo -u gopay env $(sudo cat .env | xargs) ./admintool -username akbar
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

Pola build dan upload-nya ada di §"Update rutin" → **U3. Vendor
Dashboard**. Yang cuma sekali di awal adalah unit systemd, Caddyfile, dan
DNS di bawah ini.

Unit systemd (sekali di awal, mirip `gopay-dashboard.service` tapi port
`3010`):

```bash
sudo cp deploy/gopay-vendor-dashboard.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable gopay-vendor-dashboard
```

Buat record DNS `A` untuk `vendor` yang mengarah ke IP VPS, karena subdomain
tidak ikut otomatis walau record domain utama sudah benar. Lalu tambahkan
blok `Caddyfile` untuk subdomain vendor (mis. `vendor.whuzpay.com`) yang merutekan `/api/*` ke backend **yang sama**
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
4. Customer menambah device sendiri dari Customer Dashboard → **Devices →
   Tambah device**, lalu mengisi Device ID dan Secret yang muncul ke aplikasi
   Android. `devicetool` masih bisa dipakai dari VPS untuk keadaan darurat:

   ```bash
   cd /opt/gopay-ingestion
   sudo -u gopay env $(sudo cat .env | xargs) ./devicetool -account <account_id> -name "HP Toko"
   ```

   `<account_id>` dilihat dari URL halaman detail account di Vendor
   Dashboard (`/accounts/<account_id>`).

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
curl -s -o /dev/null -w '%{http_code}\n' -u admin:<pw> "https://$D/api/v1/events"    # mau 405 (lolos basic auth; rute ini cuma menerima POST dari HP)
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
