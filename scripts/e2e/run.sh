#!/usr/bin/env bash
set -euo pipefail

# E2E validation script (curl-based).
# Usage:
#   ALLOW_PUBLIC_REGISTER=true ./scripts/e2e/run.sh
# Notes:
# - API must be running locally.
# - Requires ADMIN_USERNAME/ADMIN_PASSWORD/ADMIN_EMAIL (loaded from services/api/.env).
# - Optional: set API_BASE_URL or VITE_API_BASE_URL to target another host.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ROOT_ENV_FILE="$ROOT_DIR/.env"
BACKEND_ENV_FILE="$ROOT_DIR/services/api/.env"
FRONTEND_ENV_FILE="$ROOT_DIR/frontend/.env"

if [[ -f "$ROOT_ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$ROOT_ENV_FILE"
  set +a
fi
if [[ -f "$BACKEND_ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$BACKEND_ENV_FILE"
  set +a
fi
if [[ -f "$FRONTEND_ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$FRONTEND_ENV_FILE"
  set +a
fi

API_BASE="${API_BASE_URL:-${VITE_API_BASE_URL:-http://localhost:8080}}"
ADMIN_USERNAME="${ADMIN_USERNAME:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-}"

if [[ -z "$ADMIN_USERNAME" || -z "$ADMIN_PASSWORD" || -z "$ADMIN_EMAIL" ]]; then
  echo "Missing ADMIN_USERNAME/ADMIN_PASSWORD/ADMIN_EMAIL in env."
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
user_cookie="$tmp_dir/user.cookies"
admin_cookie="$tmp_dir/admin.cookies"

http_status=""
body=""

say() {
  printf "\n==> %s\n" "$*"
}

fail() {
  echo "FAIL: $*"
  if [[ -n "$body" ]]; then
    echo "Body: $body"
  fi
  exit 1
}

request() {
  local method="$1"
  local url="$2"
  local data="${3:-}"
  local cookiejar="${4:-}"
  local header="${5:-}"

  local args=(-sS -w '\n%{http_code}')
  if [[ -n "$cookiejar" ]]; then
    args+=(-b "$cookiejar" -c "$cookiejar")
  fi
  if [[ -n "$header" ]]; then
    args+=(-H "$header")
  fi
  if [[ -n "$data" ]]; then
    args+=(-H "Content-Type: application/json" -d "$data")
  fi
  args+=(-X "$method" "$url")

  local resp
  resp="$(curl "${args[@]}")"
  http_status="${resp##*$'\n'}"
  body="${resp%$'\n'*}"
}

extract() {
  local key="$1"
  echo "$body" | sed -n "s/.*\"$key\":\"\\([^\"]*\\)\".*/\\1/p"
}

expect_status() {
  local expected="$1"
  if [[ "$http_status" != "$expected" ]]; then
    fail "expected status $expected, got $http_status"
  fi
}

expect_code() {
  local expected="$1"
  local got
  got="$(extract code)"
  if [[ "$got" != "$expected" ]]; then
    fail "expected code $expected, got ${got:-<empty>}"
  fi
}

gen_key() {
  if command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr '[:upper:]' '[:lower:]'
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 16
    return
  fi
  date +%s
}

USER_SUFFIX="$(date +%s)"
USER_NAME="e2e-user-$USER_SUFFIX"
USER_EMAIL="e2e-$USER_SUFFIX@example.com"
USER_PASSWORD="secret"

say "Auth: register"
request "POST" "$API_BASE/auth/register" \
  "{\"username\":\"$USER_NAME\",\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}" \
  "$user_cookie"
if [[ "$http_status" == "403" ]]; then
  expect_code "registration_disabled"
  fail "public registration disabled; set ALLOW_PUBLIC_REGISTER=true"
fi
expect_status "200"

say "Auth: login"
request "POST" "$API_BASE/auth/login" \
  "{\"identifier\":\"$USER_NAME\",\"password\":\"$USER_PASSWORD\"}" \
  "$user_cookie"
expect_status "200"

say "Auth: me"
request "GET" "$API_BASE/me" "" "$user_cookie"
expect_status "200"

say "Admin: login"
request "POST" "$API_BASE/auth/login" \
  "{\"identifier\":\"$ADMIN_USERNAME\",\"password\":\"$ADMIN_PASSWORD\"}" \
  "$admin_cookie"
expect_status "200"

say "Admin: create event"
request "POST" "$API_BASE/admin/events" \
  "{\"name\":\"E2E Event $USER_SUFFIX\",\"starts_at\":\"2030-01-01T10:00:00Z\"}" \
  "$admin_cookie"
expect_status "201"
EVENT_ID="$(extract id)"
if [[ -z "$EVENT_ID" ]]; then
  fail "missing event id"
fi

say "Admin: create zone"
request "POST" "$API_BASE/admin/events/$EVENT_ID/zones" \
  "{\"name\":\"Zone A\",\"capacity\":10}" \
  "$admin_cookie"
expect_status "201"
ZONE_ID="$(extract id)"
if [[ -z "$ZONE_ID" ]]; then
  fail "missing zone id"
fi

say "Holds: create"
IDEMPOTENCY_KEY="$(gen_key)"
request "POST" "$API_BASE/holds" \
  "{\"event_id\":\"$EVENT_ID\",\"zone_id\":\"$ZONE_ID\",\"quantity\":2,\"idempotency_key\":\"$IDEMPOTENCY_KEY\"}" \
  "$user_cookie"
expect_status "201"
HOLD_ID="$(extract id)"
if [[ -z "$HOLD_ID" ]]; then
  fail "missing hold id"
fi

say "Holds: idempotent retry"
request "POST" "$API_BASE/holds" \
  "{\"event_id\":\"$EVENT_ID\",\"zone_id\":\"$ZONE_ID\",\"quantity\":2,\"idempotency_key\":\"$IDEMPOTENCY_KEY\"}" \
  "$user_cookie"
expect_status "201"
if [[ "$(extract id)" != "$HOLD_ID" ]]; then
  fail "expected same hold id on idempotent retry"
fi

say "Holds: idempotency conflict"
request "POST" "$API_BASE/holds" \
  "{\"event_id\":\"$EVENT_ID\",\"zone_id\":\"$ZONE_ID\",\"quantity\":3,\"idempotency_key\":\"$IDEMPOTENCY_KEY\"}" \
  "$user_cookie"
expect_status "409"
expect_code "idempotency_conflict"

say "Confirm: create order"
CONFIRM_KEY="$(gen_key)"
request "POST" "$API_BASE/holds/$HOLD_ID/confirm" "" "$user_cookie" "Idempotency-Key: $CONFIRM_KEY"
expect_status "201"
ORDER_ID="$(extract id)"
if [[ -z "$ORDER_ID" ]]; then
  fail "missing order id"
fi

say "Confirm: idempotent retry"
request "POST" "$API_BASE/holds/$HOLD_ID/confirm" "" "$user_cookie" "Idempotency-Key: $CONFIRM_KEY"
expect_status "200"

say "Confirm: conflict with different key"
request "POST" "$API_BASE/holds/$HOLD_ID/confirm" "" "$user_cookie" "Idempotency-Key: $(gen_key)"
expect_status "409"
expect_code "hold_already_confirmed"

say "Cancel: create active hold"
ACTIVE_KEY="$(gen_key)"
request "POST" "$API_BASE/holds" \
  "{\"event_id\":\"$EVENT_ID\",\"zone_id\":\"$ZONE_ID\",\"quantity\":1,\"idempotency_key\":\"$ACTIVE_KEY\"}" \
  "$user_cookie"
expect_status "201"
HOLD_ACTIVE_ID="$(extract id)"
if [[ -z "$HOLD_ACTIVE_ID" ]]; then
  fail "missing active hold id"
fi

say "Admin: cancel event"
request "POST" "$API_BASE/admin/events/$EVENT_ID/cancel" "" "$admin_cookie"
expect_status "200"

say "Admin: list orders (should be empty)"
request "GET" "$API_BASE/admin/events/$EVENT_ID/zones/$ZONE_ID/orders" "" "$admin_cookie"
expect_status "200"
if [[ "$(echo "$body" | tr -d '[:space:]')" != "[]" ]]; then
  fail "expected empty orders list after cancellation"
fi

say "Admin: list active holds (should be empty)"
request "GET" "$API_BASE/admin/events/$EVENT_ID/zones/$ZONE_ID/holds" "" "$admin_cookie"
expect_status "200"
if [[ "$(echo "$body" | tr -d '[:space:]')" != "[]" ]]; then
  fail "expected empty active holds list after cancellation"
fi

say "Confirm: event cancelled"
request "POST" "$API_BASE/holds/$HOLD_ACTIVE_ID/confirm" "" "$user_cookie" "Idempotency-Key: $(gen_key)"
expect_status "409"
expect_code "event_cancelled"

say "Admin access control: normal user forbidden"
request "GET" "$API_BASE/admin/events" "" "$user_cookie"
expect_status "403"

say "Admin access control: admin allowed"
request "GET" "$API_BASE/admin/events" "" "$admin_cookie"
expect_status "200"

say "Auth: logout"
request "POST" "$API_BASE/auth/logout" "" "$user_cookie"
expect_status "200"

echo
echo "E2E OK"
