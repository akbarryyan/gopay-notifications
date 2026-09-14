#!/usr/bin/env bash
# Usage: ./run_all.sh [sandbox|production]   (default: sandbox)
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

MODE="${1:-sandbox}"
case "$MODE" in
  sandbox|production) ;;
  *) echo "usage: $0 [sandbox|production]" >&2; exit 1 ;;
esac

./01_health.sh
./02_admin_login.sh
./03_create_api_key.sh "$MODE"
./04_create_payment.sh

if [ "$MODE" = "production" ]; then
  ./05_webhook_settle.sh
fi

./06_check_status.sh
