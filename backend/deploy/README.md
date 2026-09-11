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

1. Pasang PostgreSQL 16 dan Caddy.
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
   `DEVICE_SECRET_KEY` dihasilkan dengan `go run ./cmd/devicetool -genkey`.

   ```bash
   sudo chmod 600 /opt/gopay-ingestion/.env
   sudo chown gopay:gopay /opt/gopay-ingestion/.env
   ```

5. Pasang unit systemd:

   ```bash
   sudo cp deploy/gopay-ingestion.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable gopay-ingestion
   ```

6. Isi `Caddyfile` dengan domain sungguhan dan hash basic auth
   (`caddy hash-password`), lalu salin ke `/etc/caddy/Caddyfile` dan
   `sudo systemctl reload caddy`.

## Tiap rilis

```bash
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o devicetool ./cmd/devicetool
scp server devicetool VPS:/tmp/
ssh VPS 'sudo systemctl stop gopay-ingestion \
  && sudo mv /tmp/server /tmp/devicetool /opt/gopay-ingestion/ \
  && sudo chown gopay:gopay /opt/gopay-ingestion/server /opt/gopay-ingestion/devicetool \
  && sudo chmod 755 /opt/gopay-ingestion/server /opt/gopay-ingestion/devicetool \
  && sudo systemctl start gopay-ingestion'
```

Migrasi dijalankan terpisah:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

## Membuat device untuk HP

Di VPS:

```bash
cd /opt/gopay-ingestion
sudo -u gopay env $(cat .env | xargs) ./devicetool -name "HP GoPay Utama"
```

Salin `Device ID` dan `Device Secret` ke Settings aplikasi Android.
Secret tidak akan ditampilkan lagi.

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
