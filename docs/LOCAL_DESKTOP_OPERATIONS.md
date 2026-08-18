# Local Desktop Operations

Firecracker Studio is a local-desktop-first application. The Wails/Vue UI runs on the user’s desktop, while Firecracker execution runs in a managed Linux worker: directly on Linux or inside the application-managed WSL2 Ubuntu backend on Windows. The UI should guide first-time users through worker readiness instead of asking them to understand Firecracker internals.

## First-run worker setup

The first-run flow checks the Firecracker binary, Linux KVM access through `/dev/kvm`, TAP or equivalent networking capability, cgroup/resource permissions, kernel availability, rootfs storage, and the worker API health endpoint. A worker becomes selectable only after the health check passes. If a check fails, the UI shows the exact repair command or host requirement and keeps conversion/import features available without pretending that local execution is ready.

## Local networking

Each microVM receives an isolated network interface connected to a host-managed TAP device. The worker owns the guest MAC address, private address allocation, forwarding/NAT policy, port forwarding, and local hostname mapping. The desktop UI should expose published ports and connection URLs, while low-level TAP and firewall configuration remains in the worker service. Remote workers expose the same logical networking model through their authenticated API.

## Images and storage

Converted image artifacts are content-addressed and stored under the local Firecracker Studio data directory. A base image contains a compatible kernel profile, guest init contract, and rootfs. An application artifact contains the converted Docker/OCI filesystem and runtime metadata. The application must verify digests before use and must never mutate a shared base image in place.

## Persistence and volumes

MicroVM rootfs storage is ephemeral by default. A user must explicitly attach a named volume for persistent data. Named volumes are host-side directories or block files managed by the worker, with ownership and mount policy recorded in metadata. PostgreSQL, Redis, and similar stateful workloads should use attached volumes or an external database, not rely on a disposable rootfs.

## Idle stop and autoscaling

The local desktop defaults to an opt-in idle-stop policy that can stop an inactive microVM after a configurable period while preserving its artifact and attached volumes. Local autoscaling is intentionally bounded by available host CPU and memory. The UI should offer desired replicas and an idle-stop policy, but should explain that fleet autoscaling and rescheduling require a remote or multi-worker control plane.

## PostgreSQL

For the local desktop MVP, PostgreSQL should not be silently installed inside every application microVM. Firecracker Studio should support an explicit PostgreSQL image plus a named persistent volume for development, and an external PostgreSQL connection for production-like use. The desktop app’s own metadata can remain local and lightweight until a durable control-plane database is required. When PostgreSQL is launched as a user workload, health checks, volume attachment, credentials, ports, and backup/restore must be explicit.

## Remote servers

Remote server profiles contain a display name, local/remote type, worker URL, username label, and a token stored by the native desktop layer. Adding a server performs a health check before it is selectable. Switching servers changes the active API base URL, refreshes images and microVMs, and never sends operations to an unhealthy worker.
