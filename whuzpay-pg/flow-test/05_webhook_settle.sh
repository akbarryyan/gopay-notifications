#!/usr/bin/env bash
# Simulates the Cashi PAYMENT_SETTLED webhook. Only works for payments
# created with a *production* API key (sandbox rejects all webhooks).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

: "${CASHI_SECRET_KEY:?CASHI_SECRET_KEY not found — expected it in back/.env, or set it manually in flow-test/.env}"

environment="$(state_read_optional environment sandbox)"
if [ "$environment" != "production" ]; then
  fail "payment was created with a '$environment' API key; webhook only works for 'production' (sandbox always rejects webhooks)"
fi

provider_reference="$(state_read provider_reference)"
[ -n "$provider_reference" ] && [ "$provider_reference" != "null" ] || fail "no provider_reference recorded — did 04_create_payment.sh succeed?"

log "Simulating PAYMENT_SETTLED webhook for order_id=$provider_reference"
body="$(jq -nc --arg order_id "$provider_reference" '{event:"PAYMENT_SETTLED",data:{order_id:$order_id,status:"SETTLED"}}')"
signature="$(printf '%s' "$body" | openssl dgst -sha256 -hmac "$CASHI_SECRET_KEY" | awk '{print $2}')"

resp="$(req POST /api/v1/provider-webhooks/cashi "$body" -H "x-gateway-signature: $signature")"
echo "$resp" | jq .
ok "webhook delivered"
