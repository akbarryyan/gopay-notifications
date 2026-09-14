#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

: "${ADMIN_EMAIL:?set ADMIN_EMAIL in .env}"
: "${ADMIN_PASSWORD:?set ADMIN_PASSWORD in .env}"

log "Admin login as $ADMIN_EMAIL"
body="$(jq -n --arg email "$ADMIN_EMAIL" --arg password "$ADMIN_PASSWORD" '{email:$email,password:$password}')"
resp="$(req POST /api/v1/auth/admin/login "$body")"

token="$(echo "$resp" | jq -r '.token')"
[ -n "$token" ] && [ "$token" != "null" ] || fail "no token in response: $resp"

state_write admin_token "$token"
ok "admin token saved to .state/admin_token"
