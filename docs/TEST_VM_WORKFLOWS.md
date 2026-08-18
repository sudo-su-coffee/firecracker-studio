# Firecracker Studio test workflows

These scripts exercise the Firecracker Studio Go API, not the Firecracker Unix socket directly. They are intended for validating the web server and API contracts before attaching a bootable kernel and rootfs.

## Verify the server

```bash
./scripts/test-vm-workflows.sh health
```

The default server URL is `http://127.0.0.1:7822`. For another address:

```bash
STUDIO_URL=http://192.0.2.10:7822 ./scripts/test-vm-workflows.sh health
```

## Create an empty microVM record

```bash
./scripts/test-vm-workflows.sh empty
```

This creates an API-managed VM with one vCPU, 512 MiB of memory, and a synthetic test artifact digest. It validates VM creation and socket allocation. It does not boot a guest because an empty VM has no kernel or rootfs.

To request a start anyway and observe the expected boot-source error:

```bash
START_VM=1 ./scripts/test-vm-workflows.sh empty
```

The script can also request a stop after creation:

```bash
STOP_VM=1 ./scripts/test-vm-workflows.sh empty
```

## Queue an Alpine conversion

```bash
./scripts/test-vm-workflows.sh alpine
```

This submits `alpine:3.20` with the Alpine base profile to the conversion API. The operation response contains the conversion identifier, which can be inspected through `/api/v1/operations/:id`.

## Queue a PostgreSQL conversion

```bash
./scripts/test-vm-workflows.sh postgres
```

This submits `postgres:16-alpine` using the Alpine guest base. This is a conversion test only. A production PostgreSQL microVM also needs persistent storage, an explicit data directory, a network policy, credentials, backups, and a readiness check. The script intentionally does not initialize or expose a database.

## Run the complete API smoke workflow

```bash
./scripts/test-vm-workflows.sh all
```

The complete workflow checks health, creates an empty VM, and queues Alpine and PostgreSQL conversions. Set `VCPUS` and `MEMORY_MIB` to change the empty VM request:

```bash
VCPUS=2 MEMORY_MIB=1024 ./scripts/test-vm-workflows.sh empty
```
