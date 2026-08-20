# Firecracker Studio release acceptance

## Supported release target

The release targets a Linux host with KVM, Firecracker, jailer, TAP support, ext4 tooling, and a supported architecture. Studio is installed on the remote host and served through its web UI. The CLI is not required for normal operation.

## Required browser workflow

1. The operator installs the Studio server and Firecracker runtime.
2. The operator opens the configured Studio domain and signs in with the configured admin account.
3. The operator imports a supported Docker Hub image, GitHub repository, or GitHub YAML source.
4. Studio resolves and records an immutable source digest or commit.
5. Studio prepares and stores a Firecracker-native image containing a compatible kernel, rootfs, and manifest.
6. The operator launches a microVM from the stored image.
7. Studio configures per-VM TAP networking and declared host-port mappings.
8. The operator observes real readiness, health, logs, metrics, and lifecycle events.
9. The operator can use a guest terminal only when the image advertises the supported vsock agent or serial capability.
10. The operator can stop, restart, snapshot, restore, recover, and delete the workload while respecting storage policy.
11. Failed import, start, stop, crash, or recovery events create durable operations and configurable email notifications.

## Support matrix

| Area | Minimal supported behavior | Explicit limitation |
|---|---|---|
| Host | Linux x86_64 first; aarch64 only when matching kernel/artifacts are available | No native Windows runtime; WSL2 depends on KVM availability |
| Images | Public Docker Hub images that materialize into a bootable guest; public GitHub sources with supported files | No promise of arbitrary Docker compatibility |
| Networking | One TAP interface per VM, managed NAT, declared TCP/UDP host ports | No multi-queue dataplane or multi-host network fabric |
| Terminal | Studio guest agent over vsock; serial diagnostics fallback | No universal SSH; guest must opt into the capability |
| Storage | Ephemeral writable disk by default; explicitly retained persistent data disk | No automatic database replication or distributed volume layer |
| Snapshots | Full snapshot create/load with disk references and compatibility checks | Network connections may need to reconnect after restore |
| Authentication | Configured admin account, secure cookie session, logout/revocation | Multi-user teams/RBAC can follow the minimal beta |
| Notifications | SMTP alerts for failed operations and host/runtime degradation | No external event bus required for the minimal release |

## Release gates

A release cannot be merged to `main` unless formatting, `go test ./...`, `go vet ./...`, frontend production build, embedded asset verification, installer shell checks, API smoke tests, and a Linux/KVM Firecracker smoke test pass. Any gate that cannot run in the development environment must be marked host-dependent and completed on the target validation host before release tagging.

The release must not claim a feature is complete when the UI only renders a placeholder or when the backend accepts an operation without verifying the resulting Firecracker state.
