#!/usr/bin/env bash
set -Eeuo pipefail

STUDIO_URL="${STUDIO_URL:-http://127.0.0.1:7822}"
SOURCE="${1:-alpine:3.20}"
BASE_PROFILE="${2:-alpine}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1800}"

command -v curl >/dev/null 2>&1 || { echo 'curl is required' >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo 'python3 is required' >&2; exit 1; }

post_data=$(printf '{"source":"%s","sourceType":"oci","baseProfile":"%s","architecture":"native"}' "$SOURCE" "$BASE_PROFILE")
response=$(curl --fail --silent --show-error --max-time 30 -X POST "$STUDIO_URL/api/v1/conversions" -H 'Content-Type: application/json' --data "$post_data")
operation_id=$(printf '%s' "$response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
printf 'queued operation=%s source=%s base=%s\n' "$operation_id" "$SOURCE" "$BASE_PROFILE"

started=$(date +%s)
while :; do
  operation=$(curl --fail --silent --show-error --max-time 30 "$STUDIO_URL/api/v1/operations/$operation_id")
  state=$(printf '%s' "$operation" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("state", ""))')
  printf 'state=%s\n' "$state"
  case "$state" in
    succeeded)
      printf '%s\n' "$operation" | python3 -m json.tool
      printf '\nUse the returned artifact.rootfs with the kernel at the configured runtime default path.\n'
      exit 0
      ;;
    failed)
      printf '%s\n' "$operation" | python3 -m json.tool >&2
      exit 1
      ;;
  esac
  if [ $(( $(date +%s) - started )) -ge "$TIMEOUT_SECONDS" ]; then
    echo "conversion timed out after ${TIMEOUT_SECONDS}s" >&2
    exit 1
  fi
  sleep 2
done
