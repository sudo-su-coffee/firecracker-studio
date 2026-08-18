# Firecracker Studio Implementation Plan

## Goal

Build a Docker Desktop-like cross-platform application that makes Firecracker microVMs simple to install, convert, run, inspect, and connect to remotely. The application must remain self-contained while producing verified Firecracker artifacts for local or remote execution.

## Phase 0: Repository and architecture foundation

Create the Go web server with embedded Vue application, define package boundaries, establish Go formatting and tests, document supported host capabilities, and define the local controller API. The UI must be able to launch even when KVM or Firecracker is unavailable.

## Phase 1: Worker protocol and capability discovery

Define a versioned worker API for local and remote execution. It should support worker identity, version, architecture, Firecracker version, kernel profiles, KVM/TAP/cgroup capability status, storage capacity, supported operations, and health. The protocol must use authenticated TLS for remote connections and a protected local IPC path for local operation.

## Phase 2: Artifact and image management

Support local OCI layouts, Docker archives, registry references, manifests, indexes, configs, layers, digests, whiteouts, and image metadata. Add content-addressed storage, cache management, cleanup, import/export, and artifact verification. Keep conversion independent from any external control plane.

## Phase 3: Docker/OCI-to-Firecracker conversion

Implement secure layer extraction, architecture checks, base profile selection, kernel compatibility, rootfs generation, runtime metadata, boot contract, checksums, and atomic publication. Start with a tested Linux/Alpine path, then add Debian, Ubuntu, and custom operator profiles. Images requiring unsupported container-only features must fail with diagnostics.

## Phase 4: MicroVM lifecycle

Implement worker-side start, stop, restart, delete, logs, port mappings, resource configuration, snapshots, restore, and status. Use the official Firecracker Unix-socket API. Keep privileged operations in the supervisor and expose a narrow typed API to the web controller.

## Phase 5: Web experience

Build the Vue UI with a modern component library. Provide Dashboard, Images, Convert, MicroVMs, Workers, Snapshots, Logs, Settings, and Diagnostics views. The primary user journey is Import image → Convert → Choose worker → Run → View logs → Map port → Snapshot/restore.

## Phase 6: Remote worker connectivity

Add worker profiles, secure certificate/token storage, connect/disconnect, capability refresh, latency/status display, remote artifact transfer or registry pull, remote lifecycle operations, and explicit trust/revocation controls. The remote API must never tunnel an unauthenticated Firecracker Unix socket.

## Phase 7: Packaging and updates

Package the Go web server for Windows and Linux. On Windows, provision or connect to a dedicated WSL2 Linux backend. On Linux, provide native supervisor installation. Add signed update metadata, rollback of application updates, uninstall/reset, logs export, and diagnostic bundle generation.

## Phase 8: Validation

Test OCI parsing and secure extraction with malformed archives, whiteouts, symlink traversal, special files, oversized layers, unsupported architectures, and invalid configs. Test worker protocol authentication, authorization, replay protection, and capability reporting. On a real Linux/KVM host, boot converted Alpine, Debian, Ubuntu, MySQL, PostgreSQL, and custom application artifacts. On Windows, test WSL2 capability detection and remote-worker operation separately from native local execution.

## Release milestones

| Release | Scope |
|---|---|
| 0.1 | Go web server and Vue UI, Vue dashboard, local controller, diagnostics, mocked worker |
| 0.2 | Real worker protocol, local Linux supervisor, image inspection and artifact library |
| 0.3 | Docker/OCI conversion and real Firecracker start/stop on Linux |
| 0.4 | Windows WSL2 backend and remote worker connectivity |
| 0.5 | Snapshots, port mappings, logs, exports, signed packaging, and recovery |
| 1.0 | Stable cross-platform web product with compatibility matrix and local/remote Firecracker execution |

## Design rules

The web UI must never run privileged Firecracker operations directly. The converter must not depend on an external platform. Remote workers must be independently deployable. Every operation must have an explicit state, timeout, cancellation behavior, and useful error. Every artifact must be content-addressed and verified before execution.

## Production installation and release roadmap

Firecracker Studio should provide one simple installation path per operating system while keeping the runtime architecture explicit. The web UI, local controller, Firecracker supervisor, compatible kernel profiles, base artifacts, diagnostics, and update service should be delivered as one product. Firecracker itself remains unmodified; the installer provisions and manages the official runtime and its host integration.

| Platform | Primary distribution | User-facing installation |
|---|---|---|
| Windows | Standalone Go web binary, with managed WSL2 Linux runtime | Run the Go binary and open its local browser URL; optional PowerShell bootstrap command for automation |
| macOS | Signed and notarized `.dmg` plus `.pkg` or Homebrew cask later | Download, install, approve required system permissions, then use remote worker or supported local backend |
| Linux | `.deb`, `.rpm`, AppImage, and compressed `.tar.gz`; `install.sh` for convenience | Package-manager install for production, `install.sh` for quick setup, archive for portable/manual installs |

The installer must check operating-system version, CPU architecture, hardware virtualization, disk space, memory, KVM or WSL2 availability, TAP/network capability, required permissions, and compatible kernel/base artifacts. It must distinguish automatic setup from administrator actions. A user must be able to install the UI even when local Firecracker execution is unavailable and then connect to a remote worker.

### v0.1.0 — web foundation

Deliver the Go web server with embedded Vue UI, application settings, dashboard, host capability diagnostics, version display, local state storage, mocked worker connection, and stable project structure. Publish reproducible web-server binaries for Windows and Linux, with a remote-worker path for other hosts where local KVM is unavailable. This release establishes the update channel and crash/log export format.

### v0.2.0 — worker and image foundation

Add the real local controller and versioned worker protocol, local Linux supervisor, remote worker profiles, secure connection testing, OCI/Docker image pull or archive import, image inspection, content-addressed artifact storage, and conversion progress reporting. Windows should support the managed WSL2 backend path and remote workers.

### v0.3.0 — first real Firecracker workloads

Add Docker/OCI-to-Firecracker conversion, managed kernel/base profiles, artifact verification, Firecracker start/stop/restart/delete, resource configuration, port forwarding, console logs, and diagnostics on a real Linux/KVM worker. Windows can use WSL2 only where capability checks pass; otherwise remote-worker mode remains supported.

### v0.4.0 — production operations

Add snapshots and restore, artifact export/import, remote artifact transfer or registry pull, worker health, reconnect behavior, operation cancellation, cleanup policies, structured logs, signed update packages, installer repair/reset, and recovery after supervisor or desktop restart. Add the first compatibility matrix for Alpine, Debian, Ubuntu, MySQL, PostgreSQL, and custom application images.

### v0.5.0 — cross-platform release candidate

Stabilize the UI and protocol, add signed web-server binaries, signed and notarized macOS artifacts, Linux package repositories and archives, automatic update checks, rollback of failed updates, secure credential storage, worker certificate pinning, release telemetry opt-in, and complete end-to-end acceptance tests. Mark unsupported host capabilities clearly and document all limitations.

### Final 1.0 release

Release only after the Linux/KVM runtime, Windows WSL2 backend, remote-worker mode, image conversion, snapshots, lifecycle operations, packaging, update flow, security boundaries, and supported image matrix pass acceptance testing. The final release should provide a stable `firecracker-studio install` bootstrap path where appropriate, downloadable web-server binaries, reproducible release artifacts, checksums, signatures, release notes, upgrade and uninstall paths, and a documented support matrix.

### One-line installation model

The preferred experience is a small bootstrap command that detects the platform and downloads the signed installer or package. Examples are illustrative and should not be published until the release endpoints are stable:

```text
Windows PowerShell:  irm https://get.firecracker.studio/install.ps1 | iex
Linux/macOS shell:  curl -fsSL https://get.firecracker.studio/install.sh | sh
```

For security, the bootstrap script must verify the downloaded artifact’s checksum and signature, show the selected version, support a pinned version, and provide an offline/manual installation alternative. Production users should also be able to download installers directly from GitHub Releases.

### Release engineering

Every version should be built from a tagged commit by CI for the supported OS and architecture matrix. CI should produce checksums, signatures, SBOM metadata, provenance, installer artifacts, and a release manifest. The repository should not commit Firecracker binaries, large kernels, or base rootfs images directly; those should be downloaded from verified release assets or managed artifact storage according to the license and distribution policy.
