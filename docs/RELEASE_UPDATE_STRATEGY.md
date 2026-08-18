# Firecracker Studio Update Strategy

## Go web-server binaries

Firecracker Studio is distributed as a versioned Go web-server binary with embedded Vue assets. Each release should contain checksummed binaries for supported operating systems and architectures, together with release notes and a machine-readable manifest.

The application update is user-approved. The user downloads the new binary, verifies its checksum, stops the old process, replaces the binary, and starts the new version. Runtime state, images, volumes, snapshots, and Firecracker artifacts must remain outside the binary in the Studio state directory.

## Recommended update flow

| Stage | Behavior |
|---|---|
| Check | Query the latest stable GitHub Release metadata manually or from the Settings page |
| Notify | Show the available version, release notes, compatibility requirements, and download size |
| Approve | User chooses **Download update** |
| Verify | Check HTTPS download and SHA256 against the release checksum file |
| Stop | Stop the Go web server gracefully |
| Replace | Atomically replace the binary while retaining the state directory |
| Recover | Keep the previous binary available until the new process passes its health check |

The update mechanism must never execute an unverified download or silently replace a running runtime binary.

## Runtime updates

The Go web server and local Firecracker runtime are separate update concerns. Updating the web binary must not silently replace Firecracker, jailer, kernels, rootfs images, volumes, snapshots, or running microVMs. A runtime update should be explicit, display compatibility and restart impact, verify its checksum, and preserve rollback state.

For remote Firecracker workers, Studio should show the worker version and compatibility range. It should warn when the web UI and worker are incompatible but must not force an update. A remote worker is updated independently by its operator.

For local Linux/WSL2 hosts, the runtime bootstrap can be rerun with a pinned `FIRECRACKER_VERSION` after the user approves the operation:

```bash
FIRECRACKER_VERSION=v1.16.1 bash scripts/install-runtime.sh
```

## Signing and release artifacts

Before broad distribution, binaries should be signed for each target platform and accompanied by SBOM and provenance metadata. Signing credentials must remain in CI secrets and must never be committed to the repository.

## Release channels

The stable channel should use tags such as `v1.1.0`. A preview channel can use prereleases such as `v1.2.0-rc.1`, but preview releases must never replace the stable update path. Release notes should distinguish web-server changes, Vue UI changes, runtime changes, Firecracker compatibility changes, and known host limitations.

## Current release model

The active release workflow is `.github/workflows/release-web.yml`. It builds the Vue assets, embeds them into the Go binary, and produces Linux AMD64 and Windows AMD64 web-server binaries. The Windows binary serves the browser UI; Firecracker execution remains in WSL2 or a remote Linux worker.
