#!/usr/bin/env bash
set -Eeuo pipefail

# Firecracker Studio local demo server helper.
# Usage: ./scripts/run-demo-server.sh start|stop|status|logs

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PORT="${FIRECRACKER_STUDIO_DEMO_PORT:-7822}"
ADDRESS="${FIRECRACKER_STUDIO_DEMO_LISTEN:-127.0.0.1:${PORT}}"
STATE_DIR="${FIRECRACKER_STUDIO_DEMO_STATE:-${TMPDIR:-/tmp}/firecracker-studio-demo}"
PID_FILE="$STATE_DIR/server.pid"
LOG_FILE="$STATE_DIR/server.log"
BINARY="${FIRECRACKER_STUDIO_DEMO_BINARY:-$STATE_DIR/firecracker-studio}"

log() { printf '[firecracker-studio-demo] %s\n' "$*"; }
fatal() { printf '[firecracker-studio-demo] ERROR: %s\n' "$*" >&2; exit 1; }

mkdir -p "$STATE_DIR"

build() {
  if [[ -x "${FIRECRACKER_STUDIO_BIN:-}" ]]; then
    BINARY="$FIRECRACKER_STUDIO_BIN"
    return
  fi
  if [[ -x /usr/local/bin/firecracker-studio ]]; then
    BINARY=/usr/local/bin/firecracker-studio
    return
  fi
  command -v go >/dev/null 2>&1 || fatal "Go is required when no installed Firecracker Studio binary exists"
  log "Building demo server binary"
  (cd "$ROOT_DIR" && go build -trimpath -o "$BINARY" ./cmd/firecracker-studio)
}

running_pid() {
  [[ -s "$PID_FILE" ]] || return 1
  local pid
  pid=$(cat "$PID_FILE")
  kill -0 "$pid" 2>/dev/null || return 1
  printf '%s\n' "$pid"
}

start() {
  if pid=$(running_pid); then
    log "already running: pid=$pid url=http://127.0.0.1:${PORT}"
    return 0
  fi
  build
  rm -f "$PID_FILE"
  log "starting $BINARY on $ADDRESS"
  FIRECRACKER_STUDIO_LISTEN="$ADDRESS" "$BINARY" >"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"
  for _ in {1..40}; do
    if curl --connect-timeout 1 --max-time 2 --fail --silent "http://127.0.0.1:${PORT}/api/v1/health" >/dev/null 2>&1; then
      log "ready: http://127.0.0.1:${PORT}"
      return 0
    fi
    sleep 0.25
  done
  cat "$LOG_FILE" >&2 || true
  fatal "demo server did not become ready"
}

stop() {
  if ! pid=$(running_pid); then
    rm -f "$PID_FILE"
    log "not running"
    return 0
  fi
  log "stopping pid=$pid"
  kill "$pid" 2>/dev/null || true
  for _ in {1..20}; do
    kill -0 "$pid" 2>/dev/null || { rm -f "$PID_FILE"; log "stopped"; return 0; }
    sleep 0.25
  done
  kill -KILL "$pid" 2>/dev/null || true
  rm -f "$PID_FILE"
  log "stopped forcefully"
}

status() {
  if pid=$(running_pid); then
    log "running: pid=$pid url=http://127.0.0.1:${PORT}"
  else
    log "stopped"
    return 1
  fi
}

logs() { [[ -f "$LOG_FILE" ]] && tail -n 100 "$LOG_FILE" || log "no log file"; }

case "${1:-status}" in
  start) start ;;
  stop) stop ;;
  restart) stop || true; start ;;
  status) status ;;
  logs) logs ;;
  *) fatal "usage: $0 start|stop|restart|status|logs" ;;
esac
