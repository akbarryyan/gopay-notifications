#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

api_key="$(state_read api_key)"
amount="${AMOUNT:-10000}"

log "Creating payment (amount=$amount)"
body="$(jq -n --argjson amount "$amount" '{
  amount: $amount,
  payment_method: "qris",
  description: "flow-test payment",
  expires_in_minutes: 30
}')"
resp="$(req POST /api/v1/payments "$body" -H "X-API-Key: $api_key")"
echo "$resp" | jq .

id="$(echo "$resp" | jq -r '.id')"
reference="$(echo "$resp" | jq -r '.reference')"
provider_reference="$(echo "$resp" | jq -r '.provider_reference')"
[ -n "$id" ] && [ "$id" != "null" ] || fail "no id in response: $resp"

state_write payment_id "$id"
state_write payment_reference "$reference"
state_write provider_reference "$provider_reference"
ok "payment $reference created (id=$id)"
