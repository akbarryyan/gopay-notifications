# Deploy ke VPS

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

## Verifikasi setelah deploy

```bash
curl -s https://<domain>/api/v1/health
curl -s -o /dev/null -w '%{http_code}\n' https://<domain>/api/v1/events            # mau 401
curl -s -o /dev/null -w '%{http_code}\n' -u admin:<pw> https://<domain>/api/v1/events  # mau 200
curl -s -o /dev/null -w '%{http_code}\n' http://<domain>/api/v1/health             # mau 308
```
