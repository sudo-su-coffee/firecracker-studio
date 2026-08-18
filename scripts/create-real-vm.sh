#!/usr/bin/env bash
set -Eeuo pipefail

# Create and boot a real kernel/rootfs-backed workload through Firecracker Studio.
# Usage: ./scripts/create-real-vm.sh [kernel] [rootfs]

STUDIO_URL="${STUDIO_URL:-http://127.0.0.1:7822}"
KERNEL_PATH="${1:-${FIRECRACKER_KERNEL:-$HOME/.config/firecracker-studio/runtime/images/default/vmlinux}}"
ROOTFS_PATH="${2:-${FIRECRACKER_ROOTFS:-$HOME/.config/firecracker-studio/runtime/images/default/rootfs.ext4}}"
IMAGE_NAME="${FIRECRACKER_TEST_IMAGE:-firecracker-hello-real-test}"
VCPUS="${FIRECRACKER_VCPUS:-1}"
MEMORY_MIB="${FIRECRACKER_MEMORY_MIB:-512}"

fatal() { printf '[firecracker-studio-real-vm] ERROR: %s\n' "$*" >&2; exit 1; }
log() { printf '[firecracker-studio-real-vm] %s\n' "$*"; }
command -v curl >/dev/null 2>&1 || fatal 'curl is required'
command -v python3 >/dev/null 2>&1 || fatal 'python3 is required'
[ -f "$KERNEL_PATH" ] || fatal "kernel not found: $KERNEL_PATH"
[ -f "$ROOTFS_PATH" ] || fatal "rootfs not found: $ROOTFS_PATH"

curl --fail --silent --show-error --max-time 15 "$STUDIO_URL/api/v1/health" >/dev/null || fatal "Studio is not reachable at $STUDIO_URL"

payload=$(python3 - "$KERNEL_PATH" "$ROOTFS_PATH" "$IMAGE_NAME" "$VCPUS" "$MEMORY_MIB" <<'PY'
import json, sys
kernel, rootfs, image, vcpus, memory = sys.argv[1:]
print(json.dumps({
  "artifactDigest": "file://" + rootfs,
  "imageReference": image,
  "kernelPath": kernel,
  "rootfsPath": rootfs,
  "bootArgs": "console=ttyS0 reboot=k panic=1 pci=off",
  "vcpus": int(vcpus),
  "memoryMiB": int(memory),
}))
PY
)

log "creating real workload from $KERNEL_PATH and $ROOTFS_PATH"
created=$(curl --fail --silent --show-error --max-time 30 -X POST "$STUDIO_URL/api/v1/vms" -H 'Content-Type: application/json' --data "$payload")
printf '%s\n' "$created" | python3 -m json.tool
vm_id=$(printf '%s' "$created" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
[ -n "$vm_id" ] || fatal 'Studio did not return a VM id'

log "starting workload $vm_id through its Firecracker Unix socket"
started=$(curl --fail --silent --show-error --max-time 30 -X POST "$STUDIO_URL/api/v1/vms/$vm_id/start" -H 'Content-Type: application/json' --data '{}')
printf '%s\n' "$started" | python3 -m json.tool

log "workload is visible in the Web UI: $STUDIO_URL"
log "open Workloads and select $vm_id"
