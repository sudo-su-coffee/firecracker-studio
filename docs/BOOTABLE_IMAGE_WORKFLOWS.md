# Bootable Alpine and PostgreSQL microVM images

Firecracker does not boot a Docker image directly. A bootable microVM needs an uncompressed Linux kernel and a filesystem image, normally ext4. Firecracker Studio extracts OCI layers into a rootfs directory, writes a small `/sbin/init` that executes the image entrypoint and command, and creates an ext4 image with `mkfs.ext4 -d`.

## Prepare the kernel

Pull the official Hello World image from the Firecracker Studio Images page, or use the API:

```bash
curl -fsS http://127.0.0.1:7822/api/v1/marketplace/images
curl -fsS -X POST \
  http://127.0.0.1:7822/api/v1/marketplace/images/firecracker-hello-x86_64/pull
```

The verified kernel is stored below:

```text
~/.config/FirecrackerStudio/runtime/images/firecracker-hello-x86_64/vmlinux
```

To make it the default conversion kernel:

```bash
mkdir -p ~/.config/FirecrackerStudio/runtime/images/default
cp ~/.config/FirecrackerStudio/runtime/images/firecracker-hello-x86_64/vmlinux \
  ~/.config/FirecrackerStudio/runtime/images/default/vmlinux
```

The official Hello World rootfs is useful for validating the runtime. It is not a general Alpine or PostgreSQL rootfs.

## Convert Alpine

Start the server and run:

```bash
./scripts/convert-bootable-image.sh alpine:3.20 alpine
```

The converter pulls the OCI image, extracts its layers, applies whiteouts, writes `/sbin/init`, creates an ext4 rootfs, and returns an operation containing:

```json
{
  "artifact": {
    "digest": "sha256:...",
    "kernel": ".../default/vmlinux",
    "rootfs": ".../rootfs.ext4",
    "manifest": ".../manifest.json"
  }
}
```

## Convert PostgreSQL

```bash
./scripts/convert-bootable-image.sh postgres:16-alpine alpine
```

This creates a bootable filesystem artifact from the PostgreSQL OCI layers and preserves the image entrypoint metadata. PostgreSQL still requires an explicit writable data volume, environment variables such as `POSTGRES_PASSWORD`, network configuration, and a readiness check before it is suitable for a real service.

A PostgreSQL rootfs should not be treated as persistent merely because it is bootable. Store the database data directory on a separate persistent block image or volume and back it up independently.

## Inspect build output

The Convert page now contains a **BUILD OUTPUT** panel. It shows the latest operation, source image, queued/running/succeeded/failed state, operation log lines, digest, rootfs path, and conversion error. The same information is available through:

```bash
curl -fsS http://127.0.0.1:7822/api/v1/operations
curl -fsS http://127.0.0.1:7822/api/v1/operations/<operation-id>
```

## Important boot boundary

The rootfs and kernel artifacts are now generated or downloaded in the correct formats, but a complete guest boot still requires the Firecracker supervisor to launch a process, configure `/boot-source` and `/drives/rootfs` over its Unix socket, attach networking, and then issue `InstanceStart`. The current API conversion flow is therefore a real artifact builder, while full unattended boot supervision remains a separate runtime milestone.
