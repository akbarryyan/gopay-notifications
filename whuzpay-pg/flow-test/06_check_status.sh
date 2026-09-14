#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

api_key="$(state_read api_key)"
payment_id="$(state_read payment_id)"

log "Checking status of payment $payment_id"
resp="$(req GET "/api/v1/payments/$payment_id/status" "" -H "X-API-Key: $api_key")"
echo "$resp" | jq .

status="$(echo "$resp" | jq -r '.status')"
ok "status: $status"
