# Firecracker Compose Workflow

Firecracker Studio should accept a familiar Compose file while translating each service into an isolated Firecracker microVM. The Compose file remains the user-facing declaration; the worker produces one immutable artifact and one microVM per service replica.

## Inputs

The source can be a local directory, a local `compose.yaml`, a GitHub URL, or a registry reference. A GitHub source is cloned by the worker using a pinned commit or tag. A local source is copied into an operation workspace. Each service image is then resolved through BuildKit and converted into a Firecracker artifact.

## Compose translation

| Compose concept | Firecracker Studio behavior |
|---|---|
| `services` | One logical workload per service, backed by one or more microVM replicas |
| `image` | Pull OCI/Docker image, inspect config, extract layers, merge with a compatible guest base, and create an immutable artifact |
| `build` | Run BuildKit from the local or Git source, then convert the resulting OCI output |
| `command` and `entrypoint` | Store as guest runtime metadata and launch through the guest init/agent contract |
| `environment` | Inject at boot through a protected runtime configuration, never rewrite the shared image |
| `ports` | Allocate host bindings and expose a stable local connection URL |
| `expose` | Register private service discovery without publishing to the host |
| `depends_on` | Create a startup dependency graph and wait for health before promoting dependents |
| `networks` | Create a worker-managed private network group with service-name discovery |
| `volumes` | Attach named persistent host storage to a specific guest mount path |
| `restart` | Reconcile desired state after guest exit or worker restart |
| `deploy.replicas` | Run multiple isolated microVM replicas when host resources allow |

## Networking

The worker owns private network groups, guest addresses, service discovery, NAT, published ports, and local DNS aliases. A service named `db` should be reachable by other services as `db` inside the private group. Published ports are explicit and are not automatically exposed to the LAN. Remote worker networking uses the same logical model through an authenticated control API.

## Volumes and persistence

The rootfs is ephemeral by default. A named volume is required for PostgreSQL data, Redis data, uploads, or any other state that must survive a microVM replacement. Volumes are not shared accidentally between services. The worker records volume ownership, mount path, size, checksum, backup state, and attach/detach lifecycle.

## PostgreSQL

A Compose file can define PostgreSQL as a normal Firecracker workload with a persistent named volume, private network membership, health checks, credentials, and an optional published port. For production-like use, an external PostgreSQL URL is preferred. Firecracker Studio’s own desktop metadata remains separate from the application’s PostgreSQL data.

## Idle stop and scaling

A local desktop can stop inactive workloads after a user-configured idle period, but it must keep named volumes intact. Local scaling is constrained by the host’s CPU, memory, disk, and KVM capacity. The UI should show the desired replica count and the reason a replica could not be scheduled. Automatic fleet rescheduling belongs to the remote multi-worker mode.

## Example

```yaml
services:
  web:
    build: ./web
    environment:
      PORT: "8080"
    ports:
      - "8080:8080"
    networks: [app]
    depends_on: [db]
    deploy:
      replicas: 2

  db:
    image: postgres:17-alpine
    environment:
      POSTGRES_PASSWORD: local-development-only
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks: [app]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]

networks:
  app: {}

volumes:
  pgdata: {}
```

The implementation must reject unsupported Compose features explicitly rather than silently ignoring them. In particular, privileged container assumptions, host PID/network modes, device passthrough, and kernel-dependent behavior require a compatible Firecracker guest profile or an explicit error.
