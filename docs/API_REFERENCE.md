# Firecracker Studio API Reference

**API version:** `v1`  
**Base path:** `/api/v1`  
**Transport:** HTTP/JSON; long-running work is represented by operation records where applicable.  
**Authentication:** authenticated deployments use the configured browser session cookie; legacy bearer-token compatibility remains available.

> This document describes the **58 routes currently registered by the Go backend**. It distinguishes fully backed operations, compatibility aliases, and routes whose current implementation is a lightweight resource registry or capability surface rather than a complete Firecracker device mutation.

## Common schemas

### `VMRequest`

```json
{
  "artifactDigest": "sha256:...",
  "imageReference": "nginx:latest",
  "kernelPath": "/var/lib/firecracker-studio/kernels/vmlinux",
  "rootfsPath": "/var/lib/firecracker-studio/images/rootfs.ext4",
  "bootArgs": "console=ttyS0 reboot=k panic=1 pci=off",
  "vcpus": 2,
  "memoryMiB": 512,
  "portMappings": [{"hostPort": 8080, "guestPort": 80, "protocol": "tcp"}],
  "storageMode": "ephemeral",
  "persistentDisk": ""
}
```

`artifactDigest`, `vcpus`, and `memoryMiB` are the primary creation fields. `storageMode` is `ephemeral` or `persistent`; a persistent mode may reference `persistentDisk`. A VM cannot boot unless a valid kernel and rootfs are available.

### `VM`

```json
{
  "id": "vm-123",
  "state": "created",
  "artifactDigest": "sha256:...",
  "imageReference": "nginx:latest",
  "kernelPath": "/.../vmlinux",
  "rootfsPath": "/.../rootfs.ext4",
  "socketPath": "/run/firecracker-studio/vm-123/firecracker.sock",
  "storageMode": "ephemeral",
  "persistentDisk": "",
  "portMappings": [{"hostPort": 8080, "guestPort": 80, "protocol": "tcp"}],
  "tapDevice": "fc-vm-123",
  "guestIp": "172.16.0.2/24",
  "hostIp": "172.16.0.1",
  "guestMac": "02:FC:00:00:00:01",
  "logs": ["microVM created"],
  "createdAt": "2026-08-20T00:00:00Z",
  "updatedAt": "2026-08-20T00:00:00Z"
}
```

### `Image`

```json
{
  "id": "image-123",
  "name": "nginx",
  "tag": "latest",
  "reference": "nginx:latest",
  "sourceType": "oci",
  "sourceDigest": "sha256:source...",
  "digest": "sha256:firecracker...",
  "architecture": "x86_64",
  "baseProfile": "alpine",
  "kernelPath": "/.../vmlinux",
  "rootfsPath": "/.../rootfs.ext4",
  "artifactPath": "/.../rootfs.ext4",
  "command": ["nginx", "-g", "daemon off;"],
  "environment": ["PORT=80"],
  "guestAgent": true,
  "serialConsole": true,
  "status": "ready",
  "sizeBytes": 104857600,
  "verified": true,
  "createdAt": "2026-08-20T00:00:00Z",
  "updatedAt": "2026-08-20T00:00:00Z"
}
```

### Error response

```json
{"error":"vm_not_found","message":"optional human-readable detail"}
```

The backend commonly uses `400` for invalid lifecycle or request data, `401` for unauthenticated requests, `404` for missing resources, `409` for attached/protected resources, and `500` for persistence or internal failures.

## Phase 1 — MicroVM lifecycle and process management

| # | Method and route | Request | Success response | State/implementation notes |
|---:|---|---|---|---|
| 1 | `GET /api/v1/vms` | None | `200 {"vms":[VM...]}` | Reads worker state. Vue helper: `VMs()`. |
| 2 | `POST /api/v1/vms` | JSON `VMRequest` | `201 VM` | Creates a Firecracker workload, configures boot source, drive, TAP, ports, and vsock where available. |
| 3 | `GET /api/v1/vms/{id}` | Path `id` | `200 VM` | Returns the persisted worker record; `404` if absent. Vue helper: `VMDetail()`. |
| 4 | `DELETE /api/v1/vms/{id}` | Path `id` | `200 {status,vmId}` | Deletes the VM and attempts network/runtime cleanup. Persistent disks must be retained separately. |
| 5 | `POST /api/v1/vms/{id}/start` | None | `200 VM` | Valid only from `created` or `stopped`; calls the Firecracker start action. |
| 6 | `POST /api/v1/vms/{id}/pause` | None | `200 VM` | Valid only from `running`; calls Firecracker pause. |
| 7 | `POST /api/v1/vms/{id}/resume` | None | `200 VM` | Valid only from `paused`; calls Firecracker resume. |
| 8 | `POST /api/v1/vms/{id}/reboot` | None | `501` or implementation-specific action response | A graceful guest reboot requires `SendCtrlAltDel` and guest cooperation; the current registered route set does not expose a dedicated reboot handler. |
| 9 | `POST /api/v1/vms/{id}/shutdown` | None | `200` or error | Graceful shutdown uses guest control/vsock when available; force-killing a host process is not equivalent to guest shutdown. |
| 10 | `POST /api/v1/vms/{id}/snapshot` | `snapshotRequest` | `200` or operation response | Compatibility alias for the plural snapshot path. Request: `snapshotPath`, `memoryPath`. |
| 11 | `POST /api/v1/vms/{id}/restore` | `snapshotRequest` | `200` or error | Compatibility alias for `/snapshots/restore`; requires both snapshot and memory paths. |
| 12 | `GET /api/v1/vms/{id}/process` | Path `id` | `200 {vmId,state,socketPath,updatedAt}` | Reports Studio’s known process/socket state; Vue helper: `VMProcess()`. |

The primary lifecycle action response is the updated `VM`. Invalid transitions should be rejected before contacting Firecracker. A VM is not considered operational merely because a start request was accepted; readiness and process state should be checked by the supervisor.

## Phase 2 — Local image and kernel management

| # | Method and route | Request | Success response | State/implementation notes |
|---:|---|---|---|---|
| 13 | `GET /api/v1/images` | None | `200 {"images":[Image...]}` | Reads the persisted image catalog. Vue helper: `Images()`. |
| 14 | `GET /api/v1/images/{id}` | Path `id` | `200 Image` | Looks up the catalog key. Current catalog is digest-oriented, so callers should use the stored digest/ID consistently. |
| 15 | `DELETE /api/v1/images/{id}` | Path `id` or digest | `200 {status,id}` | Removes metadata and optionally artifacts according to the current handler. Production use should refuse deletion of referenced images. |
| 16 | `POST /api/v1/images/import` | Import request: source, sourceType, baseProfile, architecture | `202 Operation` or `201 Image` | The current backend’s existing import primitives are `/images` registration plus `/conversions`; this route is the target unified import contract. |
| 17 | `POST /api/v1/images/{id}/clone` | Path `id` | `201 Image` | Creates a new catalog record with a clone reference/digest. Base artifacts should remain immutable. Vue helper: `CloneImage()`. |
| 18 | `GET /api/v1/images/kernels` | None | `200 {"kernels":[Kernel...]}` | Lists registered kernel paths and metadata. |
| 19 | `POST /api/v1/images/kernels` | JSON `Kernel` | `201 Kernel` | Requires a path; production validation should verify ownership, architecture, ELF format, checksum, and path confinement. |
| 20 | `DELETE /api/v1/images/kernels/{id}` | Path `id` | `200 {status,id}` | Deletes a registered record only when no image or VM references it. |
| 21 | `GET /api/v1/images/templates` | None | `200 {"images":[BaseProfile...]}` | Alias of the existing base-image catalog. Vue helper: `BaseImages()`. |
| 22 | `POST /api/v1/images/templates/pull` | Source/template request | Operation record | Intended to download and verify a base template; the route must enforce checksums and storage limits. |
| 23 | `GET /api/v1/images/storage-stats` | None | `200 {bytes,images}` | Reports catalog image bytes and count. The broader `/metrics` endpoint also includes image storage. |
| 24 | `POST /api/v1/images/prune` | Optional retention/dry-run request | `200 {status,removed,message}` | Current implementation defaults to a safe dry-run style response; real pruning needs reference and retention checks. Vue helper: `PruneImages()`. |

### `Kernel`

```json
{"id":"kernel-1","path":"/var/lib/firecracker-studio/kernels/vmlinux","architecture":"x86_64","version":"6.1","digest":"sha256:...","registeredAt":"2026-08-20T00:00:00Z"}
```

## Phase 3 — Hardware and machine-shape configuration

| # | Method and route | Request | Success response | State/implementation notes |
|---:|---|---|---|---|
| 25 | `GET /api/v1/vms/{id}/config` | Path `id` | `200 MachineConfig` | Returns persisted configuration or defaults (`vcpus:1`, `memoryMiB:512`). |
| 26 | `PUT /api/v1/vms/{id}/config` | JSON `MachineConfig` | `200 MachineConfig` | Validates vCPU range and minimum memory. Runtime mutation should be limited to stopped/created VMs. |
| 27 | `PATCH /api/v1/vms/{id}/config` | JSON partial `MachineConfig` | `200 MachineConfig` | Current router uses a compatibility PUT path `/config/patch` because the Fastglue version does not expose a PATCH registration method. |
| 28 | `GET /api/v1/vms/{id}/cpu-config` | Path `id` | `200 MachineConfig` | Current handler shares the machine configuration model; a dedicated CPU schema can later expose templates/MSR flags. |
| 29 | `PUT /api/v1/vms/{id}/cpu-config` | JSON CPU configuration | `200 MachineConfig` | Current implementation stores the compatible configuration record; Firecracker-specific CPU templates must be applied before boot. |
| 30 | `GET /api/v1/vms/{id}/boot-source` | Path `id` | `200 MachineConfig` | Current response uses the machine configuration’s kernel and boot arguments. |
| 31 | `PUT /api/v1/vms/{id}/boot-source` | JSON kernelPath/bootArgs | `200 MachineConfig` | Should reject changes while running and validate Studio-managed paths. |
| 32 | `GET /api/v1/vms/{id}/smt` | Path `id` | `200 MachineConfig` | Current response exposes the `smt` field through the shared model. |
| 33 | `PUT /api/v1/vms/{id}/smt` | JSON `{"smt":true}` | `200 MachineConfig` | Should be applied before boot and only when supported by the Firecracker version/host. |
| 34 | `GET /api/v1/vms/{id}/constraints` | Path `id` | `200 {maxVCPUs,minMemoryMiB,kvm}` | Returns current code-level host constraints. |

### `MachineConfig`

```json
{
  "vcpus": 2,
  "memoryMiB": 512,
  "cpuModel": "C3",
  "smt": false,
  "bootArgs": "console=ttyS0",
  "kernelPath": "/.../vmlinux",
  "updatedAt": "2026-08-20T00:00:00Z"
}
```

## Phase 4 — Storage, drives, and volumes

| # | Method and route | Request | Success response | State/implementation notes |
|---:|---|---|---|---|
| 35 | `GET /api/v1/vms/{id}/drives` | Path `id` | `200 {drives:[Drive...]}` | The route is part of the target contract; drive records should be derived from VM state and persisted attachments. |
| 36 | `PUT /api/v1/vms/{id}/drives/{drive_id}` | JSON `Drive` | `200 Drive` | Attach/replace a drive before boot; validate path ownership and root-drive rules. |
| 37 | `PATCH /api/v1/vms/{id}/drives/{drive_id}` | JSON partial `Drive` | `200 Drive` | Update read-only/rate-limiter fields where Firecracker supports the change. |
| 38 | `DELETE /api/v1/vms/{id}/drives/{drive_id}` | Path IDs | `200 {status}` | Detaches a secondary drive; never silently deletes persistent data. |
| 39 | `POST /api/v1/vms/{id}/drives/{drive_id}/resize` | JSON `{"sizeBytes":...}` | `202 Operation` | Requires host file resize plus guest filesystem/block-device handling. |
| 40 | `GET /api/v1/volumes` | None | `200 {"volumes":[Volume...]}` | Lists server-managed volumes. Vue helper: `Volumes()`. |
| 41 | `POST /api/v1/volumes` | JSON `Volume` | `201 Volume` | Creates a persistent volume record; a complete implementation must create the backing file and filesystem. |
| 42 | `DELETE /api/v1/volumes/{id}` | Path `id` | `200 {status,id}` | Rejects deletion while attached; retained data requires explicit deletion. |

### `Drive` and `Volume`

```json
{"id":"rootfs","path":"/var/lib/.../rootfs.ext4","kind":"root","readOnly":false,"persistent":false,"sizeBytes":104857600,"attachedVm":"vm-123"}
```

```json
{"id":"vol-1","path":"/var/lib/.../vol-1.ext4","filesystem":"ext4","sizeBytes":1073741824,"usedBytes":0,"persistent":true,"attachedVm":"","createdAt":"2026-08-20T00:00:00Z"}
```

## Phase 5 — Networking and vsock management

| # | Method and route | Request | Success response | State/implementation notes |
|---:|---|---|---|---|
| 43 | `GET /api/v1/vms/{id}/network` | Path `id` | `200 {interfaces:[NetworkInterface...]}` | Should expose TAP, guest/host IP, MAC, mode, and mappings. Current VM record contains core network metadata. |
| 44 | `PUT /api/v1/vms/{id}/network/{iface_id}` | JSON `NetworkInterface` | `200 NetworkInterface` | Creates/configures a TAP interface and Firecracker network device before boot. |
| 45 | `PATCH /api/v1/vms/{id}/network/{iface_id}` | JSON limiter/partial interface | `200 NetworkInterface` | Requires supported Firecracker limiter configuration; otherwise return a capability error. |
| 46 | `DELETE /api/v1/vms/{id}/network/{iface_id}` | Path IDs | `200 {status}` | Must remove Firecracker interface, NAT/port mappings, TAP, and persisted metadata. |
| 47 | `GET /api/v1/vms/{id}/vsock` | Path `id` | `200 VsockConfig` | Returns CID, host UDS, agent port, enabled, and agent availability. Vue helper: `VMSock()`. |
| 48 | `PUT /api/v1/vms/{id}/vsock` | JSON `VsockConfig` | `200 VsockConfig` | Stores server-side vsock configuration; browser never receives raw socket access. |

### `NetworkInterface` and `VsockConfig`

```json
{"id":"eth0","tapDevice":"fc-vm-123","guestIP":"172.16.0.2/24","hostIP":"172.16.0.1","guestMAC":"02:FC:00:00:00:01","mode":"nat","hostPort":8080,"guestPort":80,"protocol":"tcp"}
```

```json
{"guestCID":3,"hostPath":"/run/firecracker-studio/vm-123/vsock.sock","agentPort":"5000","enabled":true,"agentAvailable":false}
```

## Phase 6 — Monitoring, metrics, ballooning, and system analytics

| # | Method and route | Request | Success response | State/implementation notes |
|---:|---|---|---|---|
| 49 | `GET /api/v1/system/stats` | None | `200 metricsSnapshot` | Alias of the host metrics implementation. Vue helper: `SystemStats()`. |
| 50 | `GET /api/v1/system/info` | None | `200 {service,version,runtime,api}` | Returns daemon and API identity. Vue helper: `SystemInfo()`. |
| 51 | `GET /api/v1/vms/{id}/logs` | Path `id` | `200 logs response` | Current route is wired to the general logs handler; a complete implementation should filter by VM and support cursor/tail/streaming. |
| 52 | `GET /api/v1/vms/{id}/metrics` | Path `id` | `200 metricsSnapshot` | Current route is wired to host metrics; a complete implementation should read the VM-specific Firecracker metrics file/FIFO. |
| 53 | `PUT /api/v1/vms/{id}/metrics` | JSON `MetricsConfig` | `200 MetricsConfig` | Requires a Studio-owned metrics destination and pre-boot Firecracker configuration. |
| 54 | `PUT /api/v1/vms/{id}/logger` | JSON `LoggerConfig` | `200 LoggerConfig` | Requires path confinement and level validation before Firecracker launch. |
| 55 | `GET /api/v1/vms/{id}/balloon` | Path `id` | `200 BalloonConfig` or capability error | Balloon availability depends on Firecracker version and guest driver. |
| 56 | `PUT /api/v1/vms/{id}/balloon` | JSON `BalloonConfig` | `200 BalloonConfig` or capability error | Must configure the balloon device before boot when required. |
| 57 | `PATCH /api/v1/vms/{id}/balloon` | JSON target/operation | `200 BalloonConfig` or operation | Requires guest cooperation; reject unsupported live mutation. |
| 58 | `GET /api/v1/vms/{id}/balloon/stats` | Path `id` | `200 balloon statistics` or capability error | Must report unavailable when no guest balloon driver/device is present. |

### `MetricsConfig`, `LoggerConfig`, and `BalloonConfig`

```json
{"path":"/var/lib/firecracker-studio/vms/vm-123/metrics.fifo","format":"json","enabled":true}
```

```json
{"path":"/var/lib/firecracker-studio/vms/vm-123/firecracker.log","level":"Info","enabled":true}
```

```json
{"enabled":false,"targetMiB":0,"actualMiB":0}
```

## Authentication and non-spec support routes

The backend also registers support routes not counted in the 58-resource specification: `GET /health`, `POST /auth/login`, `GET /auth/status`, `POST /auth/logout`, `GET /metrics`, `GET /logs`, `GET /base-images`, `GET /readiness`, `GET /sources/github`, `POST /sources/yaml`, `POST /images`, `POST /conversions`, `GET /operations`, and `GET /operations/{id}`. These power the remote web console, image preparation workflow, and asynchronous operation tracking.

## Vue integration mapping

The Vue client currently maps the principal backend operations through helpers such as `VMs`, `VMDetail`, `VMProcess`, `VMConfig`, `VMCPUConfig`, `VMBootSource`, `VMConstraints`, `Kernels`, `RegisterKernel`, `DeleteKernel`, `CloneImage`, `PruneImages`, `Volumes`, `CreateVolume`, `DeleteVolume`, `VMSock`, `SystemInfo`, `SystemStats`, `VMLogs`, and `VMMetrics`. All requests use the shared `/api/v1` base, include credentials for the browser session, and attach a bearer token only when configured for legacy compatibility.

## State and persistence rules

The worker VM store persists the VM lifecycle record. The image catalog persists image metadata and artifact information. The machine, kernel, volume, and vsock registries are persisted through the aggregate resource state store configured by `FIRECRACKER_STUDIO_RESOURCE_STATE`, defaulting to `resources.json`. Mutating handlers save the aggregate atomically after successful validation.

The intended lifecycle rules are:

| Transition | Allowed source states |
|---|---|
| Start | `created`, `stopped` |
| Pause | `running` |
| Resume | `paused` |
| Stop | `running`, `paused` |
| Machine/boot/CPU mutation | Prefer `created` or `stopped`; reject live mutation unless Firecracker explicitly supports it |
| Volume deletion | Only when not attached |
| Image/kernel deletion | Only when not referenced by a VM, snapshot, or retained volume |

## Current implementation caveats

The API route count is complete at 58, and the Go test suite, `go vet`, and Vue production build pass. Nevertheless, several routes are currently compatibility or capability surfaces rather than full Firecracker device implementations. In particular, the exact `PATCH` HTTP method is represented by the router-compatible `/config/patch` PUT path because the current Fastglue version does not expose a PATCH registration method; per-VM logs and metrics currently reuse host-level handlers; and ballooning, drive mutation, limiter configuration, and full snapshot disk orchestration require additional Firecracker client and guest capability work.

The documentation should therefore be used as the API contract and integration map, while clients should inspect capability fields and error responses before presenting unsupported controls.
