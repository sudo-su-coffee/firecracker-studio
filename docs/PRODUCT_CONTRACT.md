# Firecracker Studio browser-first product contract

## Product promise

Firecracker Studio is a remote desktop-style web console for running compatible application images as isolated Firecracker microVMs. A user installs Studio on a Linux/KVM server, opens the Studio domain, imports an image from Docker Hub or GitHub, and runs it as a microVM without operating Firecracker sockets, TAP devices, kernel files, or rootfs paths directly.

The primary user workflow is browser-only:

```text
Sign in → Import image → Image becomes Ready → Run as MicroVM
→ Observe health/logs → Open guest terminal when supported
→ Stop/restart/snapshot/recover → Receive failure notifications
```

The CLI is not a prerequisite for this workflow and remains a later automation interface.

## User-facing vocabulary

| Internal concept | UI term |
|---|---|
| OCI/Docker-to-guest materialization | Import image / Prepare image |
| Kernel + rootfs + manifest | Firecracker image |
| Conversion job | Image import operation |
| Firecracker process | MicroVM |
| Host-backed block file | Disk or volume |
| Guest agent over vsock | Guest terminal |
| Firecracker API Unix socket | Runtime socket (not exposed to users) |

## Supported sources

The first release exposes three source paths:

1. **Docker Hub image:** a public image reference such as `alpine:3.20` or `nginx@sha256:...`.
2. **GitHub repository:** a public repository containing a supported Dockerfile, guest artifact pair, or Studio-compatible build description.
3. **GitHub YAML:** a single Studio/Docker-style YAML file that references one image or repository and declares the minimal command, ports, environment, and storage mode.

Sources are pinned to an immutable digest, commit, or release where possible. The UI shows the resolved source before starting preparation.

## Stored Firecracker image

A stored image is immutable after it becomes Ready. It is stored below the configured artifact root and has a catalog record containing:

- image name and tag;
- original source type and reference;
- immutable source digest or commit;
- architecture;
- kernel path and checksum;
- rootfs path and checksum;
- startup command and environment metadata;
- guest-agent and serial-console capabilities;
- total size and creation time;
- preparation status and error details.

The image catalog is a private local registry for Firecracker-native images. It is not a Docker registry and does not expose raw host paths to the browser.

## MicroVM lifecycle

The control plane recognizes these states:

```text
created → preparing → ready → starting → running
running → stopping → stopped
running → pausing → paused → resuming → running
running → failed → recovering → starting
stopped → deleted
```

A VM is not displayed as healthy merely because a start request was accepted. It becomes `running` only after the Firecracker process, guest boot, and configured readiness checks succeed.

## Networking

Every network-enabled microVM receives a Studio-managed TAP device and a private guest address. The first networking mode is host-managed NAT with explicit host-port mappings. Studio owns creation, routing, firewall rules, teardown, and collision validation. Direct public exposure requires an explicit domain/port action and an exposure warning.

Firecracker provides the virtual network device, but Studio provides the host networking integration and the guest-side address configuration.

## Guest terminal

Firecracker does not include SSH or a shell. Studio supports a guest terminal only when the image has a compatible guest agent over virtio-vsock or a supported serial-console configuration. The terminal must provide command text, stdout, stderr, exit code, timeout, cancellation, and an audit record.

SSH inside a guest is an optional compatibility path, not the default. SSH to the Studio host is never part of the normal operator workflow.

## Storage modes

Every VM explicitly chooses one storage mode:

| Mode | Semantics | Default use |
|---|---|---|
| `ephemeral` | A disposable writable disk is created for the VM and deleted with it. | Stateless code, frontend, API, workers, preview deployments. |
| `persistent` | A separately owned data disk survives VM deletion when retained. | Databases and stateful services. |
| `snapshot` | A snapshot stores VM memory/state plus references to required disks. | Fast restore and recovery; not a replacement for volume policy. |

The base Firecracker image is read-only from the workload’s perspective. Persistent data is never deleted implicitly by removing a VM. Snapshot restore must validate Firecracker version, architecture, vCPU/memory/device configuration, kernel/rootfs identity, disk references, and available snapshot files.

## Minimum supported Firecracker features

The browser must expose boot source, root drive, optional data drive, vCPU and memory configuration, VM start/stop, pause/resume where supported, readiness, TAP networking, host-port mapping, health checks, logs, metrics, lifecycle events, guest terminal capability, snapshot create/load, and bounded crash recovery.

Unsupported or image-dependent behavior must be displayed as unavailable or unsupported, never simulated.

## Explicit non-goals for the minimal release

The minimal release does not promise arbitrary Docker compatibility, multi-host scheduling, multi-region placement, full Compose semantics, automatic database clustering, a public image registry, or a universal terminal for every guest image.
