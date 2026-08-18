# Firecracker Studio Implementation Plan

## Goal

Build a Docker Desktop-like cross-platform application that makes Firecracker microVMs simple to install, convert, run, inspect, and connect to remotely. The application must remain independent of Porter while producing artifacts Porter can consume.

## Phase 0: Repository and architecture foundation

Create the Wails/Vue application shell, define package boundaries, establish Go formatting and tests, document supported host capabilities, and define the local controller API. The UI must be able to launch even when KVM or Firecracker is unavailable.

## Phase 1: Worker protocol and capability discovery

Define a versioned worker API for local and remote execution. It should support worker identity, version, architecture, Firecracker version, kernel profiles, KVM/TAP/cgroup capability status, storage capacity, supported operations, and health. The protocol must use authenticated TLS for remote connections and a protected local IPC path for local operation.

## Phase 2: Artifact and image management

Support local OCI layouts, Docker archives, registry references, manifests, indexes, configs, layers, digests, whiteouts, and image metadata. Add content-addressed storage, cache management, cleanup, import/export, and artifact verification. Keep conversion independent from Porter.

## Phase 3: Docker/OCI-to-Firecracker conversion

Implement secure layer extraction, architecture checks, base profile selection, kernel compatibility, rootfs generation, runtime metadata, boot contract, checksums, and atomic publication. Start with a tested Linux/Alpine path, then add Debian, Ubuntu, and custom operator profiles. Images requiring unsupported container-only features must fail with diagnostics.

## Phase 4: MicroVM lifecycle

Implement worker-side start, stop, restart, delete, logs, port mappings, resource configuration, snapshots, restore, and status. Use the official Firecracker Unix-socket API. Keep privileged operations in the supervisor and expose a narrow typed API to the desktop controller.

## Phase 5: Desktop experience

Build the Vue UI with a modern component library. Provide Dashboard, Images, Convert, MicroVMs, Workers, Snapshots, Logs, Settings, and Diagnostics views. The primary user journey is Import image → Convert → Choose worker → Run → View logs → Map port → Snapshot/restore.

## Phase 6: Remote worker connectivity

Add worker profiles, secure certificate/token storage, connect/disconnect, capability refresh, latency/status display, remote artifact transfer or registry pull, remote lifecycle operations, and explicit trust/revocation controls. The remote API must never tunnel an unauthenticated Firecracker Unix socket.

## Phase 7: Packaging and updates

Package the Wails application for Windows and Linux. On Windows, provision or connect to a dedicated WSL2 Linux backend. On Linux, provide native supervisor installation. Add signed update metadata, rollback of application updates, uninstall/reset, logs export, and diagnostic bundle generation.

## Phase 8: Validation

Test OCI parsing and secure extraction with malformed archives, whiteouts, symlink traversal, special files, oversized layers, unsupported architectures, and invalid configs. Test worker protocol authentication, authorization, replay protection, and capability reporting. On a real Linux/KVM host, boot converted Alpine, Debian, Ubuntu, MySQL, PostgreSQL, and custom application artifacts. On Windows, test WSL2 capability detection and remote-worker operation separately from native local execution.

## Release milestones

| Release | Scope |
|---|---|
| 0.1 | Wails shell, Vue dashboard, local controller, diagnostics, mocked worker |
| 0.2 | Real worker protocol, local Linux supervisor, image inspection and artifact library |
| 0.3 | Docker/OCI conversion and real Firecracker start/stop on Linux |
| 0.4 | Windows WSL2 backend and remote worker connectivity |
| 0.5 | Snapshots, port mappings, logs, exports, signed packaging, and recovery |
| 1.0 | Stable cross-platform desktop product with compatibility matrix and Porter artifact integration |

## Design rules

The desktop UI must never run privileged Firecracker operations directly. The converter must not depend on Porter. Remote workers must be independently deployable. Every operation must have an explicit state, timeout, cancellation behavior, and useful error. Every artifact must be content-addressed and verified before execution.
