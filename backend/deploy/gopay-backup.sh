#!/usr/bin/env bash
# Backup harian database produksi. Dijalankan gopay-backup.service (lewat
# gopay-backup.timer) sebagai user "postgres", jadi pg_dump memakai peer
# auth lewat socket lokal -- tidak ada password database di skrip ini.
#
# Yang dilakukan:
#   1. pg_dump format custom (terkompresi, bisa di-restore per tabel)
#   2. verifikasi dump bisa dibaca pg_restore sebelum dianggap sah
#   3. opsional: unggah ke S3 bila BACKUP_S3_URI diisi
#   4. hapus backup lokal yang lebih tua dari RETENTION_DAYS
#
# Backup lokal saja TIDAK melindungi dari instance/disk yang hilang --
# isi BACKUP_S3_URI untuk salinan di luar server. Konfigurasi di
# /etc/default/gopay-backup (lihat backend/deploy/README.md).
#
# Sengaja TIDAK ikut menyalin /opt/gopay-ingestion/.env: kunci enkripsi yang
# disimpan di tempat yang sama dengan dump membuat siapa pun yang mendapat
# backup bisa membuka secret device/webhook/SMTP sekaligus. Simpan .env di
# password manager, terpisah dari backup.
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/gopay}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
DB_NAME="${DB_NAME:-gopay}"
BACKUP_S3_URI="${BACKUP_S3_URI:-}"

umask 077
mkdir -p "$BACKUP_DIR"

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
final="$BACKUP_DIR/${DB_NAME}-${stamp}.dump"
partial="$final.partial"
trap 'rm -f "$partial"' EXIT

echo "backup: dump $DB_NAME ke $final"
pg_dump --format=custom --no-owner --dbname="$DB_NAME" --file="$partial"

# Dump yang terpotong (disk penuh, koneksi putus) tetap berukuran > 0 --
# satu-satunya cara tahu dump itu sah adalah mencoba membacanya.
pg_restore --list "$partial" > /dev/null
mv "$partial" "$final"
echo "backup: selesai, ukuran $(du -h "$final" | cut -f1)"

if [[ -n "$BACKUP_S3_URI" ]]; then
  echo "backup: unggah ke ${BACKUP_S3_URI%/}/"
  aws s3 cp "$final" "${BACKUP_S3_URI%/}/$(basename "$final")" --only-show-errors
fi

# Retensi cuma untuk salinan lokal. Salinan di S3 diatur lifecycle rule
# bucket-nya sendiri, supaya kredensial server ini tidak perlu izin hapus --
# server yang dibobol tidak boleh bisa ikut menghapus backup di luar server.
deleted="$(find "$BACKUP_DIR" -maxdepth 1 -name "${DB_NAME}-*.dump" -mtime +"$RETENTION_DAYS" -print -delete | wc -l)"
echo "backup: $deleted backup lokal lebih dari $RETENTION_DAYS hari dihapus"
