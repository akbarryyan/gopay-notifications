#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

log "Checking $BASE_URL/api/v1/health"
resp="$(req GET /api/v1/health "")"
echo "$resp" | jq .
ok "backend is up"
