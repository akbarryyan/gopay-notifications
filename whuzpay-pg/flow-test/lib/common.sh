#!/usr/bin/env bash
# Shared helpers for flow-test scripts. Sourced, not executed directly.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$SCRIPT_DIR/.state"
mkdir -p "$STATE_DIR"

if [ -f "$SCRIPT_DIR/.env" ]; then
  set -a
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/.env"
  set +a
fi

BASE_URL="${BASE_URL:-http://localhost:8080}"

# CASHI_SECRET_KEY belongs to back/ (it's the shared secret the real Cashi
# provider signs webhooks with). Rather than duplicating it into
# flow-test/.env — where it'd silently go stale after a rotation — read it
# straight from back/.env unless the user explicitly overrode it here.
BACK_ENV_FILE="$SCRIPT_DIR/../back/.env"
if [ -z "${CASHI_SECRET_KEY:-}" ] && [ -f "$BACK_ENV_FILE" ]; then
  CASHI_SECRET_KEY="$(grep -E '^CASHI_SECRET_KEY=' "$BACK_ENV_FILE" | tail -n1 | cut -d'=' -f2-)"
  export CASHI_SECRET_KEY
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${CYAN}==>${NC} $*"; }
ok() { echo -e "${GREEN}✓${NC} $*"; }
fail() { echo -e "${RED}✗${NC} $*" >&2; exit 1; }

require_jq() {
  command -v jq >/dev/null 2>&1 || fail "jq is required but not installed"
}

state_write() { printf '%s' "$2" >"$STATE_DIR/$1"; }
state_read() {
  [ -f "$STATE_DIR/$1" ] || fail "missing $STATE_DIR/$1 — run the earlier step first"
  cat "$STATE_DIR/$1"
}
state_read_optional() { [ -f "$STATE_DIR/$1" ] && cat "$STATE_DIR/$1" || echo -n "$2"; }

# req METHOD PATH BODY_JSON_OR_EMPTY [EXTRA_CURL_ARGS...]
# Prints the response body to stdout, fails on non-2xx.
req() {
  require_jq
  local method="$1" path="$2" body="${3:-}"
  shift 3
  local tmp status
  tmp="$(mktemp)"
  local -a curl_args=(-sS -o "$tmp" -w '%{http_code}' -X "$method" "$BASE_URL$path" -H 'Content-Type: application/json')
  if [ -n "$body" ]; then
    curl_args+=(-d "$body")
  fi
  curl_args+=("$@")
  status="$(curl "${curl_args[@]}")"
  local resp
  resp="$(cat "$tmp")"
  rm -f "$tmp"
  if [ "$status" -lt 200 ] || [ "$status" -ge 300 ]; then
    echo "$resp" | jq . >&2 2>/dev/null || echo "$resp" >&2
    fail "$method $path returned HTTP $status"
  fi
  echo "$resp"
}
