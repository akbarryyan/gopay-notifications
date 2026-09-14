#!/usr/bin/env bash
# Pemantau server produksi: memeriksa kesehatan server lalu mengabari vendor
# (Akbar) lewat Telegram SAAT STATUS BERUBAH -- sekali ketika ada masalah,
# diulang tiap REPEAT_HOURS selama belum beres, dan sekali saat pulih.
# Dijalankan gopay-monitor.timer setiap 5 menit.
#
# Yang diperiksa:
#   - service systemd (backend, dua dashboard, caddy, postgresql) aktif
#   - Postgres menerima koneksi
#   - URL publik menjawab lewat HTTPS (membuktikan Caddy + sertifikat + app)
#   - sisa masa berlaku sertifikat HTTPS (Caddy memperpanjang ~30 hari
#     sebelum habis, jadi sisa < CERT_WARN_DAYS berarti perpanjangan gagal)
#   - pemakaian disk dan memori
#   - backup harian terakhir tidak basi
#
# BATASAN: skrip ini berjalan DI server yang dipantau. Kalau VPS-nya sendiri
# mati atau jaringannya putus, skrip ini ikut mati dan tidak bisa mengabari
# siapa pun. Pasangkan dengan pemantau dari luar (UptimeRobot) -- lihat
# backend/deploy/README.md §"Monitoring dan peringatan".
#
# Konfigurasi di /etc/default/gopay-monitor. Wajib: TELEGRAM_BOT_TOKEN,
# TELEGRAM_CHAT_ID. Tanpa keduanya, hasil cuma ditulis ke log (journal).
set -uo pipefail

SERVICES="${SERVICES-gopay-ingestion gopay-dashboard gopay-vendor-dashboard caddy postgresql}"
URLS="${URLS-https://whuzpay.com/api/v1/health https://whuzpay.com/login https://vendor.whuzpay.com/login}"
CERT_DOMAINS="${CERT_DOMAINS-whuzpay.com vendor.whuzpay.com}"
CERT_WARN_DAYS="${CERT_WARN_DAYS:-14}"
DISK_PATHS="${DISK_PATHS-/}"
DISK_WARN_PERCENT="${DISK_WARN_PERCENT:-85}"
MEM_WARN_PERCENT="${MEM_WARN_PERCENT:-92}"
BACKUP_DIR="${BACKUP_DIR-/var/backups/gopay}"
BACKUP_MAX_AGE_HOURS="${BACKUP_MAX_AGE_HOURS:-26}"
CHECK_POSTGRES="${CHECK_POSTGRES:-1}"
REPEAT_HOURS="${REPEAT_HOURS:-6}"
STATE_FILE="${STATE_FILE:-/var/lib/gopay-monitor/state}"
SERVER_NAME="${SERVER_NAME:-$(hostname)}"
TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
TELEGRAM_CHAT_ID="${TELEGRAM_CHAT_ID:-}"
# DRY_RUN=1: tulis pesan ke stdout, jangan kirim ke Telegram.
DRY_RUN="${DRY_RUN:-0}"

now="$(date +%s)"
declare -A result=()   # nama pemeriksaan -> "" (sehat) atau pesan masalah

check() { result["$1"]="$2"; }

# --- Pemeriksaan -----------------------------------------------------------

for svc in $SERVICES; do
  if systemctl is-active --quiet "$svc"; then
    check "service:$svc" ""
  else
    check "service:$svc" "Service $svc tidak berjalan ($(systemctl is-active "$svc" 2>/dev/null))"
  fi
done

if [[ "$CHECK_POSTGRES" == "1" ]]; then
  if pg_isready -q -t 5 2>/dev/null; then
    check "postgres" ""
  else
    check "postgres" "PostgreSQL tidak menerima koneksi"
  fi
fi

for url in $URLS; do
  code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$url" || true)"
  if [[ "$code" =~ ^(2|3)[0-9][0-9]$ ]]; then
    check "url:$url" ""
  else
    [[ "$code" == "000" ]] && code="tidak terhubung/timeout"
    check "url:$url" "$url menjawab $code"
  fi
done

for domain in $CERT_DOMAINS; do
  end="$(echo | timeout 15 openssl s_client -servername "$domain" -connect "$domain:443" 2>/dev/null \
    | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)"
  if [[ -z "$end" ]]; then
    check "cert:$domain" "Sertifikat HTTPS $domain tidak dapat dibaca"
    continue
  fi
  days=$(( ($(date -d "$end" +%s) - now) / 86400 ))
  if (( days < CERT_WARN_DAYS )); then
    check "cert:$domain" "Sertifikat HTTPS $domain tinggal $days hari (perpanjangan otomatis Caddy kemungkinan gagal)"
  else
    check "cert:$domain" ""
  fi
done

for path in $DISK_PATHS; do
  used="$(df --output=pcent "$path" 2>/dev/null | tail -1 | tr -dc '0-9')"
  if [[ -z "$used" ]]; then
    check "disk:$path" "Pemakaian disk $path tidak dapat dibaca"
  elif (( used >= DISK_WARN_PERCENT )); then
    check "disk:$path" "Disk $path terpakai $used% (batas $DISK_WARN_PERCENT%)"
  else
    check "disk:$path" ""
  fi
done

mem_used="$(awk '/MemTotal/{t=$2} /MemAvailable/{a=$2} END{if(t>0) printf "%d", (t-a)*100/t}' /proc/meminfo)"
if [[ -n "$mem_used" ]] && (( mem_used >= MEM_WARN_PERCENT )); then
  check "memory" "Memori terpakai $mem_used% (batas $MEM_WARN_PERCENT%)"
else
  check "memory" ""
fi

if [[ -n "$BACKUP_DIR" ]]; then
  if [[ ! -d "$BACKUP_DIR" ]]; then
    check "backup" "Folder backup $BACKUP_DIR tidak ada -- backup harian belum dipasang?"
  else
    latest="$(find "$BACKUP_DIR" -maxdepth 1 -name '*.dump' -printf '%T@\n' 2>/dev/null | sort -n | tail -1)"
    if [[ -z "$latest" ]]; then
      check "backup" "Belum ada file backup di $BACKUP_DIR"
    else
      age_h=$(( (now - ${latest%.*}) / 3600 ))
      if (( age_h > BACKUP_MAX_AGE_HOURS )); then
        check "backup" "Backup terakhir sudah $age_h jam lalu (batas $BACKUP_MAX_AGE_HOURS jam)"
      else
        check "backup" ""
      fi
    fi
  fi
fi

# --- Bandingkan dengan status sebelumnya -----------------------------------
# Format state: <nama>\t<sejak>\t<terakhir dikabari>\t<pesan>  (cuma yang bermasalah)

declare -A since=() notified=() last_msg=()
if [[ -f "$STATE_FILE" ]]; then
  while IFS=$'\t' read -r name s n m; do
    [[ -n "$name" ]] || continue
    since["$name"]="$s"; notified["$name"]="$n"; last_msg["$name"]="$m"
  done < "$STATE_FILE"
fi

new_problems=() still_problems=() recovered=()
declare -A next_since=() next_notified=() next_msg=()

for name in "${!result[@]}"; do
  msg="${result[$name]}"
  # Nilai dipindah ke variabel biasa dulu: kunci seperti "url:https://..."
  # tidak aman dipakai langsung di dalam (( )).
  prev_since="${since[$name]:-}"
  prev_notified="${notified[$name]:-0}"
  if [[ -z "$msg" ]]; then
    if [[ -n "$prev_since" ]]; then
      mins=$(( (now - prev_since) / 60 ))
      recovered+=("✅ Sudah pulih setelah $mins menit. Sebelumnya: ${last_msg[$name]:-$name}")
    fi
    continue
  fi
  echo "masalah: $msg"
  next_msg["$name"]="$msg"
  if [[ -z "$prev_since" ]]; then
    new_problems+=("🔴 $msg")
    next_since["$name"]="$now"; next_notified["$name"]="$now"
  elif (( now - prev_notified >= REPEAT_HOURS * 3600 )); then
    hours=$(( (now - prev_since) / 3600 ))
    still_problems+=("🟠 Masih bermasalah sejak $hours jam lalu: $msg")
    next_since["$name"]="$prev_since"; next_notified["$name"]="$now"
  else
    next_since["$name"]="$prev_since"; next_notified["$name"]="$prev_notified"
  fi
done

# Pemeriksaan yang dihapus dari konfigurasi dianggap selesai, bukan pulih.

write_state() {
  mkdir -p "$(dirname "$STATE_FILE")"
  local tmp="$STATE_FILE.tmp"
  : > "$tmp"
  for name in "${!next_since[@]}"; do
    printf '%s\t%s\t%s\t%s\n' "$name" "${next_since[$name]}" "${next_notified[$name]}" "${next_msg[$name]}" >> "$tmp"
  done
  mv "$tmp" "$STATE_FILE"
}

lines=("${new_problems[@]}" "${still_problems[@]}" "${recovered[@]}")
if (( ${#lines[@]} == 0 )); then
  write_state
  echo "monitor: tidak ada perubahan (${#next_since[@]} masalah aktif)"
  exit 0
fi

text="[$SERVER_NAME] Payment Bridge"$'\n'
for l in "${lines[@]}"; do text+="$l"$'\n'; done

if [[ "$DRY_RUN" == "1" ]]; then
  printf '%s' "$text"
  write_state
  exit 0
fi

if [[ -z "$TELEGRAM_BOT_TOKEN" || -z "$TELEGRAM_CHAT_ID" ]]; then
  printf 'monitor: TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID belum diisi, pesan tidak dikirim:\n%s' "$text"
  write_state
  exit 0
fi

# Token lewat config curl dari stdin, bukan argumen: argumen proses bisa
# dibaca user lain lewat /proc/<pid>/cmdline.
if printf 'url = "https://api.telegram.org/bot%s/sendMessage"\n' "$TELEGRAM_BOT_TOKEN" \
  | curl -s --max-time 15 -o /dev/null -w '%{http_code}' -K - \
      --data-urlencode "chat_id=$TELEGRAM_CHAT_ID" --data-urlencode "text=$text" \
  | grep -q '^200$'; then
  echo "monitor: peringatan terkirim"
  write_state
else
  # State TIDAK disimpan: putaran berikutnya mencoba mengabari lagi.
  echo "monitor: gagal mengirim peringatan ke Telegram" >&2
  printf '%s' "$text" >&2
  exit 1
fi
