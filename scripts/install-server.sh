#!/usr/bin/env bash
set -Eeuo pipefail

# Firecracker Studio unattended server installer.
# This script installs only a prebuilt Go server binary from GitHub Releases.
# Firecracker/jailer/KVM artifacts are handled separately by install-runtime.sh.

REPO="${FIRECRACKER_STUDIO_REPO:-sudo-su-coffee/firecracker-studio}"
VERSION="${FIRECRACKER_STUDIO_VERSION:-latest}"
INSTALL_DIR="${FIRECRACKER_STUDIO_INSTALL_DIR:-/opt/firecracker-studio}"
BIN_PATH="${FIRECRACKER_STUDIO_BIN:-/usr/local/bin/firecracker-studio}"
LISTEN_ADDRESS="${FIRECRACKER_STUDIO_LISTEN:-127.0.0.1:7822}"
CONFIG_PATH="${FIRECRACKER_STUDIO_CONFIG:-/etc/firecracker-studio/config.toml}"
PUBLIC_URL="${FIRECRACKER_STUDIO_PUBLIC_URL:-}"
ASSET_NAME="FirecrackerStudio-linux-amd64"
CHECKSUM_NAME="SHA256SUMS-linux-amd64.txt"

log() { printf '[firecracker-studio] %s\n' "$*"; }
fatal() { printf '[firecracker-studio] ERROR: %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -ne 0 ] || fatal "run this script as your normal Linux user; it uses sudo for system installation"
command -v sudo >/dev/null 2>&1 || fatal "sudo is required"
command -v sha256sum >/dev/null 2>&1 || fatal "sha256sum is required"

if command -v apt-get >/dev/null 2>&1; then
  log "Installing server runtime utilities"
  sudo apt-get update -y
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl coreutils
else
  fatal "apt-get is required for this Ubuntu installer"
fi

CURL_OPTS=(--fail --show-error --location --silent --connect-timeout 15 --max-time 600 --retry 3 --retry-delay 2 --retry-all-errors)
if [ "$VERSION" = "latest" ]; then
  log "Resolving latest published release"
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    VERSION="$(gh release view --repo "$REPO" --json tagName --jq '.tagName')"
  else
    VERSION="$(curl "${CURL_OPTS[@]}" -H 'Accept: application/vnd.github+json' "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | head -n 1)"
  fi
fi
[ -n "$VERSION" ] || fatal "could not determine a GitHub release version"

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

log "Downloading Firecracker Studio release $VERSION"
base_url="https://github.com/${REPO}/releases/download/${VERSION}"
log "Downloading $ASSET_NAME"
curl "${CURL_OPTS[@]}" -o "$work_dir/$ASSET_NAME" "$base_url/$ASSET_NAME"
log "Downloading $CHECKSUM_NAME"
curl "${CURL_OPTS[@]}" -o "$work_dir/$CHECKSUM_NAME" "$base_url/$CHECKSUM_NAME"

[ -s "$work_dir/$ASSET_NAME" ] || fatal "release asset $ASSET_NAME was not downloaded"
[ -s "$work_dir/$CHECKSUM_NAME" ] || fatal "release checksum $CHECKSUM_NAME was not downloaded"

log "Verifying release checksum"
(
  cd "$work_dir"
  sha256sum -c "$CHECKSUM_NAME" --ignore-missing
)

sudo install -d -m 0755 "$INSTALL_DIR" "$INSTALL_DIR/state" "$INSTALL_DIR/logs" "$(dirname "$CONFIG_PATH")"
sudo install -m 0755 "$work_dir/$ASSET_NAME" "$BIN_PATH"

if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
  SERVICE_USER="$USER"
  SERVICE_HOME="$HOME"
  CONFIG_ENV=""
  [ -f "$CONFIG_PATH" ] && CONFIG_ENV="Environment=FIRECRACKER_STUDIO_CONFIG=$CONFIG_PATH"
  PUBLIC_ENV=""
  [ -n "$PUBLIC_URL" ] && PUBLIC_ENV="Environment=FIRECRACKER_STUDIO_PUBLIC_URL=$PUBLIC_URL"
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
${CONFIG_ENV}
${PUBLIC_ENV}
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
  [ -f "$CONFIG_PATH" ] || log "No config.toml found at $CONFIG_PATH; copy config.example.toml there before enabling remote access"
else
  log "systemd is unavailable; start manually with: FIRECRACKER_STUDIO_LISTEN=${LISTEN_ADDRESS} ${BIN_PATH}"
fi

log "Installed release: $VERSION"
log "Server binary: $BIN_PATH"
log "Listen address: $LISTEN_ADDRESS"
log "Open the UI at: http://127.0.0.1:${LISTEN_ADDRESS##*:}"
