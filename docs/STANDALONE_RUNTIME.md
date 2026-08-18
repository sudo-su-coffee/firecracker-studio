# Firecracker Studio Standalone Runtime

Firecracker Studio manages a local Firecracker runtime directly. Remote worker profiles are an optional separate feature for connecting to another Linux host.

## What Studio installs

When the user selects **Install Firecracker**, the Wails desktop application downloads the official Firecracker release archive for the host architecture, verifies the pinned SHA256 digest, extracts both `firecracker` and `jailer`, and stores them under the Studio runtime directory.

| Platform | Runtime location |
|---|---|
| Linux | `$XDG_CONFIG_HOME/FirecrackerStudio/runtime/bin` or the platform user config directory |
| Windows | The selected WSL2 Ubuntu user’s `~/.config/firecracker-studio/runtime/bin` |

The installer does not install Docker, containerd, or any unrelated container runtime. OCI conversion remains a BuildKit-based Studio capability, while Firecracker execution uses the official Unix-socket API.

## Windows behavior

On Windows, Firecracker requires a Linux environment. Studio invokes WSL2 Ubuntu for the local runtime installation and readiness check. WSL2, CPU virtualization, `/dev/kvm` access, and guest networking still depend on the host configuration. Studio reports missing permissions or artifacts; it does not silently modify BIOS settings or privileged network configuration.

## Readiness states

The dashboard reports Firecracker, jailer, KVM, TAP/networking, kernel, and rootfs independently. Installation of the two binaries does not imply that a microVM can boot. A compatible guest kernel and ext4 rootfs must be present before a real workload can start.

## Optional remote workers

A remote worker is added only through **Add remote worker**. The profile stores a display name, URL, optional username label, and bearer token in the native Go layer. Server profiles are persisted in a per-user `servers.json` file with restrictive permissions; the token is excluded from Wails-returned JSON. Remote health checking never starts or installs the remote worker.

## Current implementation boundary

The standalone binary installer and readiness checks are implemented. The next integration step is to bind the local runtime manager to the existing `internal/worker.Service` so Convert, Create, Start, Stop, Snapshot, and Restore use the installed binary and local socket directory when no remote worker is selected. Until that binding is complete, the UI should show runtime readiness accurately and avoid claiming that a local microVM has booted.
