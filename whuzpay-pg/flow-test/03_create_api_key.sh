#!/usr/bin/env bash
# Usage: ./03_create_api_key.sh [sandbox|production]
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

ENVIRONMENT="${1:-sandbox}"
case "$ENVIRONMENT" in
  sandbox|production) ;;
  *) fail "environment must be 'sandbox' or 'production', got '$ENVIRONMENT'" ;;
esac

: "${MERCHANT_ID:?set MERCHANT_ID in .env}"
: "${ADMIN_PASSWORD:?set ADMIN_PASSWORD in .env}"
admin_token="$(state_read admin_token)"

log "Minting $ENVIRONMENT API key for merchant $MERCHANT_ID"
body="$(jq -n --arg env "$ENVIRONMENT" --arg pw "$ADMIN_PASSWORD" '{environment:$env,password:$pw}')"
resp="$(req PUT "/api/v1/admin/merchants/$MERCHANT_ID/api-keys" "$body" -H "Authorization: Bearer $admin_token")"

secret="$(echo "$resp" | jq -r '.secret')"
[ -n "$secret" ] && [ "$secret" != "null" ] || fail "no secret in response: $resp"

state_write api_key "$secret"
state_write environment "$ENVIRONMENT"
ok "$ENVIRONMENT API key saved to .state/api_key"
