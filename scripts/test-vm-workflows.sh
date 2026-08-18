#!/usr/bin/env bash
set -Eeuo pipefail

# Firecracker Studio API workflow smoke tests.
# This script talks to the Go web server; it never starts Firecracker directly.
# Usage:
#   ./scripts/test-vm-workflows.sh health
#   ./scripts/test-vm-workflows.sh empty
#   START_VM=1 ./scripts/test-vm-workflows.sh empty
#   ./scripts/test-vm-workflows.sh alpine
#   ./scripts/test-vm-workflows.sh postgres
#   ./scripts/test-vm-workflows.sh all

STUDIO_URL="${STUDIO_URL:-http://127.0.0.1:7822}"
VCPUS="${VCPUS:-1}"
MEMORY_MIB="${MEMORY_MIB:-512}"

log() { printf '[firecracker-studio-test] %s\n' "$*"; }
fatal() { printf '[firecracker-studio-test] ERROR: %s\n' "$*" >&2; exit 1; }

require_tools() {
  command -v curl >/dev/null 2>&1 || fatal 'curl is required'
  command -v python3 >/dev/null 2>&1 || fatal 'python3 is required to parse API responses'
}

api() {
  curl --fail --silent --show-error --max-time "${CURL_TIMEOUT:-30}" "$@"
}

json_value() {
  local key="$1"
  python3 -c 'import json,sys; print(json.load(sys.stdin).get(sys.argv[1], ""))' "$key"
}

health() {
  local response
  response="$(api "$STUDIO_URL/api/v1/health")"
  printf '%s\n' "$response"
  printf '%s\n' "$response" | python3 -c 'import json,sys; d=json.load(sys.stdin); raise SystemExit(0 if d.get("status") == "ok" else 1)' \
    || fatal 'Studio health check did not return status=ok'
}

create_empty() {
  local digest="empty-test-$(date -u +%Y%m%dT%H%M%SZ)"
  log "Creating an empty API-managed microVM with artifact digest ${digest}"
  local response id
  response="$(api -X POST "$STUDIO_URL/api/v1/vms" \
    -H 'Content-Type: application/json' \
    --data "{\"artifactDigest\":\"${digest}\",\"vcpus\":${VCPUS},\"memoryMiB\":${MEMORY_MIB}}")"
  printf '%s\n' "$response"
  id="$(printf '%s' "$response" | json_value id)"
  [ -n "$id" ] || fatal 'API did not return a VM id'
  log "Created VM ${id}"

  if [ "${START_VM:-0}" = '1' ]; then
    log "Requesting start for ${id}"
    api -X POST "$STUDIO_URL/api/v1/vms/${id}/start" -H 'Content-Type: application/json' --data '{}' || \
      fatal 'empty VM start failed; an empty VM has no kernel/rootfs boot source by design'
  fi

  if [ "${STOP_VM:-0}" = '1' ]; then
    log "Requesting stop for ${id}"
    api -X POST "$STUDIO_URL/api/v1/vms/${id}/stop" -H 'Content-Type: application/json' --data '{}'
  fi
}

queue_conversion() {
  local name="$1" reference="$2" base="$3"
  log "Queueing ${name} conversion from ${reference} using ${base} base"
  api -X POST "$STUDIO_URL/api/v1/conversions" \
    -H 'Content-Type: application/json' \
    --data "{\"source\":\"${reference}\",\"sourceType\":\"oci\",\"baseProfile\":\"${base}\",\"architecture\":\"native\"}"
  printf '\n'
}

main() {
  require_tools
  local command="${1:-all}"
  case "$command" in
    health) health ;;
    empty) health; create_empty ;;
    alpine) health; queue_conversion 'Alpine sample' 'alpine:3.20' 'alpine' ;;
    postgres) health; queue_conversion 'PostgreSQL sample' 'postgres:16-alpine' 'alpine' ;;
    all)
      health
      create_empty
      queue_conversion 'Alpine sample' 'alpine:3.20' 'alpine'
      queue_conversion 'PostgreSQL sample' 'postgres:16-alpine' 'alpine'
      ;;
    *)
      fatal "unknown command ${command}; use health, empty, alpine, postgres, or all"
      ;;
  esac
}

main "$@"
