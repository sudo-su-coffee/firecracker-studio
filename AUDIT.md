# Firecracker Studio audit

## Current state

The selected repository is a Go control plane with an embedded Vue frontend. It already includes runtime installation/status checks, OCI conversion packages, a Firecracker Unix-socket client, asynchronous image conversion operations, a local web API, and documentation for WSL2/server workflows.

## Confirmed implementation observations

- `internal/firecracker/client.go` currently covers machine-config, boot-source, drives, InstanceStart, SendCtrlAltDel, snapshot create, and snapshot load.
- The client currently lacks explicit `/vm` state operations for Pause/Resume and does not expose richer GET/list/metrics endpoints.
- `internal/runtime/manager.go` checks Firecracker, jailer, `/dev/kvm`, TAP tooling, kernel, and rootfs status; it downloads a pinned Firecracker release with checksums.
- The runtime status check on Linux labels `/dev/kvm` ready based on device permissions but does not yet validate KVM ioctls or distinguish host capability from permissions in detail.
- `internal/operations/manager.go` uses an in-memory queue and in-memory operation map; conversion operations are not durable across process restart.
- The repository uses external Go dependencies, so the earlier claim of zero third-party dependencies is no longer true for the selected repository.
- The frontend is already Vue-based (`frontend/src/App.vue`) rather than only a single-file vanilla UI.
- Official Firecracker documentation confirms that a rootfs is a filesystem image containing an init such as `/sbin/init`; Docker/OCI images are source material for constructing rootfs images, not directly bootable Firecracker disks. [1] [2]
- Official snapshot documentation confirms Pause/Resume/CreateSnapshot are post-boot operations, LoadSnapshot is pre-boot, snapshots contain guest memory and VM/device state while disk files remain separately managed, and network/vsock connections may not survive restore. [1]

## Immediate risks to resolve

1. The repo’s implementation and the pasted prior response disagree on dependency model and frontend status.
2. Firecracker API coverage is incomplete for the user’s requested real-world lifecycle: pause, resume, VM state, process liveness/reconciliation, and richer status/error reporting.
3. In-memory operations are unsuitable for a long-running server because restart loses queued/running/completed state.
4. KVM preflight must be explicit and actionable before VM start, not merely a filesystem permission check.
5. Docker/OCI conversion must be clearly separated from VM boot artifacts, with an explicit builder boundary for future BuildKit/AOCI support.
6. Snapshot storage needs metadata, checksums, disk references, retention/deletion semantics, and backup/restore documentation.

## Sources

[1]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/snapshotting/snapshot-support.md "Firecracker snapshot support"
[2]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/rootfs-and-kernel-setup.md "Firecracker rootfs and kernel setup"
