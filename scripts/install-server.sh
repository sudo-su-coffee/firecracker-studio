#!/usr/bin/env bash
set -Eeuo pipefail

# Firecracker Studio unattended Go web-server installer.
# This script installs only the Go backend and embedded Vue web UI.
# Firecracker/jailer/KVM artifacts are intentionally handled by install-runtime.sh.

VERSION="${FIRECRACKER_STUDIO_VERSION:-main}"
APP_NAME="firecracker-studio"
INSTALL_DIR="${FIRECRACKER_STUDIO_INSTALL_DIR:-/opt/firecracker-studio}"
BIN_PATH="${FIRECRACKER_STUDIO_BIN:-/usr/local/bin/firecracker-studio}"
SOURCE_DIR="${FIRECRACKER_STUDIO_SOURCE_DIR:-$HOME/src/firecracker-studio}"
LISTEN_ADDRESS="${FIRECRACKER_STUDIO_LISTEN:-127.0.0.1:7822}"
REPO="${FIRECRACKER_STUDIO_REPO:-https://github.com/sudo-su-coffee/firecracker-studio.git}"

log() { printf '[firecracker-studio] %s\n' "$*"; }
fatal() { printf '[firecracker-studio] ERROR: %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -ne 0 ] || fatal "run this script as your normal Linux user; it uses sudo for system installation"
command -v sudo >/dev/null 2>&1 || fatal "sudo is required"

if command -v apt-get >/dev/null 2>&1; then
  log "Installing Go web-server build utilities"
  sudo apt-get update -y
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates git golang-go nodejs npm
else
  fatal "apt-get is required for this Ubuntu installer"
fi

if [ ! -d "$SOURCE_DIR/.git" ]; then
  mkdir -p "$(dirname "$SOURCE_DIR")"
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    log "Cloning Firecracker Studio source with GitHub CLI"
    rm -rf "$SOURCE_DIR"
    gh repo clone sudo-su-coffee/firecracker-studio "$SOURCE_DIR" -- --branch "$VERSION"
  elif [ -n "${GITHUB_TOKEN:-}" ]; then
    log "Cloning Firecracker Studio source with GITHUB_TOKEN"
    rm -rf "$SOURCE_DIR"
    git clone --branch "$VERSION" "https://x-access-token:${GITHUB_TOKEN}@github.com/sudo-su-coffee/firecracker-studio.git" "$SOURCE_DIR"
  else
    fatal "source directory is missing; clone the private repository first with gh auth login, or set GITHUB_TOKEN"
  fi
fi

cd "$SOURCE_DIR"
log "Building Vue production assets"
npm ci --prefix frontend
npm run build --prefix frontend
rm -rf internal/web/dist
mkdir -p internal/web/dist
cp -R frontend/dist/. internal/web/dist/

log "Building the single Go web binary"
tmp_binary="$(mktemp)"
trap 'rm -f "$tmp_binary"' EXIT
go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o "$tmp_binary" ./cmd/firecracker-studio

sudo install -d -m 0755 "$INSTALL_DIR"
sudo install -m 0755 "$tmp_binary" "$BIN_PATH"
sudo install -d -m 0755 "$INSTALL_DIR/state"
sudo install -d -m 0755 "$INSTALL_DIR/logs"

if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
  SERVICE_USER="$USER"
  SERVICE_HOME="$HOME"
  sudo tee /etc/systemd/system/firecracker-studio.service >/dev/null <<UNIT
[Unit]
Description=Firecracker Studio Go web server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
Environment=FIRECRACKER_STUDIO_LISTEN=${LISTEN_ADDRESS}
Environment=HOME=${SERVICE_HOME}
ExecStart=${BIN_PATH}
Restart=on-failure
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
UNIT
  sudo systemctl daemon-reload
  sudo systemctl enable --now firecracker-studio.service
  log "systemd service started: firecracker-studio.service"
else
  log "systemd is unavailable; start manually with: FIRECRACKER_STUDIO_LISTEN=${LISTEN_ADDRESS} ${BIN_PATH}"
fi

log "Server binary: $BIN_PATH"
log "Listen address: $LISTEN_ADDRESS"
log "Open the UI at: http://127.0.0.1:${LISTEN_ADDRESS##*:}"
