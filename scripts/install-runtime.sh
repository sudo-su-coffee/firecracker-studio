#!/usr/bin/env bash
set -Eeuo pipefail

# Firecracker Studio unattended Linux/WSL2 runtime bootstrap.
# This script installs host utilities and official Firecracker/jailer binaries.
# It does not change BIOS/UEFI settings and cannot grant KVM access without a
# new login shell when the host uses group-based device permissions.

VERSION="${FIRECRACKER_VERSION:-v1.16.1}"
PREFIX="${FIRECRACKER_STUDIO_HOME:-${XDG_CONFIG_HOME:-$HOME/.config}/firecracker-studio}"
RUNTIME="$PREFIX/runtime"
BIN="$RUNTIME/bin"
IMAGES="$RUNTIME/images/default"

log() { printf '[firecracker-studio] %s\n' "$*"; }
fatal() { printf '[firecracker-studio] ERROR: %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -ne 0 ] || fatal "run this script as your normal Linux user; it uses sudo for system packages"
command -v sudo >/dev/null 2>&1 || fatal "sudo is required"

if command -v apt-get >/dev/null 2>&1; then
  log "Installing Linux host utilities"
  sudo apt-get update -y
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates curl tar gzip coreutils iproute2 iptables acl e2fsprogs squashfs-tools
else
  log "apt-get is unavailable; skipping package installation"
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) RELEASE_ARCH="x86_64" ;;
  aarch64|arm64) RELEASE_ARCH="aarch64" ;;
  *) fatal "unsupported architecture: $ARCH" ;;
esac

mkdir -p "$BIN" "$IMAGES"
chmod 700 "$RUNTIME" "$BIN" "$IMAGES"

BASE="https://github.com/firecracker-microvm/firecracker/releases/download/${VERSION}"
ARCHIVE="firecracker-${VERSION}-${RELEASE_ARCH}.tgz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

log "Downloading official Firecracker ${VERSION} (${RELEASE_ARCH})"
curl --fail --location --silent --show-error --retry 3 \
  "${BASE}/${ARCHIVE}" -o "$TMP/$ARCHIVE"

# GitHub release archives are verified against the release SHA256SUMS asset.
curl --fail --location --silent --show-error --retry 3 \
  "${BASE}/SHA256SUMS" -o "$TMP/SHA256SUMS"
grep "  ${ARCHIVE}$" "$TMP/SHA256SUMS" | (cd "$TMP" && sha256sum -c -)

tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
RELEASE_DIR="$TMP/release-${VERSION}-${RELEASE_ARCH}"
[ -f "$RELEASE_DIR/firecracker-${VERSION}-${RELEASE_ARCH}" ] || fatal "Firecracker binary missing from official archive"
[ -f "$RELEASE_DIR/jailer-${VERSION}-${RELEASE_ARCH}" ] || fatal "jailer binary missing from official archive"

install -m 0700 "$RELEASE_DIR/firecracker-${VERSION}-${RELEASE_ARCH}" "$BIN/firecracker"
install -m 0700 "$RELEASE_DIR/jailer-${VERSION}-${RELEASE_ARCH}" "$BIN/jailer"

# Best-effort KVM group setup. The user must start a new login session for the
# group membership to take effect; no privileged change is silently forced.
if [ -e /dev/kvm ]; then
  if getent group kvm >/dev/null 2>&1; then
    sudo usermod -aG kvm "$USER" || log "Could not add $USER to kvm group"
  fi
fi

log "Firecracker: $BIN/firecracker"
log "jailer:      $BIN/jailer"
"$BIN/firecracker" --version
"$BIN/jailer" --help >/dev/null

if [ -e /dev/kvm ]; then
  if [ -r /dev/kvm ] && [ -w /dev/kvm ]; then
    log "KVM access: ready"
  else
    log "KVM access: permission denied; run 'newgrp kvm' or reopen your Linux session"
  fi
else
  log "KVM device: missing; enable KVM on the Linux/WSL2 host"
fi

log "Runtime installed. Kernel/rootfs artifacts can be placed under: $IMAGES"
printf '%s\n' "$RUNTIME"
