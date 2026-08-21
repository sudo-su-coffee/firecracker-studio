# Firecracker Studio

Firecracker Studio is a **small Go web server with an embedded Vue UI for controlling Firecracker microVMs**. It is the simple browser experience for Firecracker: install the official runtime separately, start the Studio server, open a browser, and manage workloads without operating Firecracker’s Unix sockets directly.

Firecracker Studio is **not Porter**, a cloud SaaS, a multi-node scheduler, Docker, containerd, or a replacement for the Firecracker runtime. It does not modify Firecracker. The Go process is the narrow privileged boundary between the browser and the official Firecracker API.

> **Firecracker Studio is the Docker Desktop-style UI for Firecracker microVMs running on your own Linux host, VPS, or home lab.**

## Architecture

```text
Browser Vue UI
   -> same-origin /api/v1
      -> Go Firecracker Studio server
         -> installed Firecracker and jailer runtime
         -> local Unix-socket supervisor
            -> official Firecracker + KVM + host integration
```

The browser never opens `/dev/kvm`, executes `jailer`, or talks to a raw Firecracker socket. Those operations remain inside the Go server on native Linux or inside WSL2 Ubuntu.

## Installation model

Firecracker Studio intentionally uses **two separate installation layers**. The runtime installer is optional product infrastructure and may be replaced by the user’s own installation process. Studio itself is only the web control server and embedded UI.

### 1. Install the official Firecracker runtime separately

For the project-provided runtime bootstrap on a private repository:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-runtime.sh -H 'Accept: application/vnd.github.raw' | bash
```

For a public repository:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-runtime.sh | bash
```

This step installs the official Firecracker and jailer binaries and prepares runtime directories. Users may instead install a compatible Firecracker runtime by another method.

### 2. Install and run Firecracker Studio

For an authenticated private repository:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-server.sh -H 'Accept: application/vnd.github.raw' | bash
```

For a public repository:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-server.sh | bash
```

The server installer downloads and verifies the published Go binary. It does not install Firecracker, clone the repository, or build the frontend locally. By default Studio listens on `127.0.0.1:7822`.

Open [http://127.0.0.1:7822](http://127.0.0.1:7822) after starting the service. The health endpoint is [http://127.0.0.1:7822/api/v1/health](http://127.0.0.1:7822/api/v1/health), and the readiness endpoint is [http://127.0.0.1:7822/api/v1/readiness](http://127.0.0.1:7822/api/v1/readiness).

### Exposing Studio on a VPS

Studio is loopback-only by default. To intentionally bind it to a non-loopback address, set a bearer token:

```bash
FIRECRACKER_STUDIO_LISTEN=0.0.0.0:7822 \
FIRECRACKER_STUDIO_TOKEN='replace-with-a-long-random-token' \
firecracker-studio
```

The health endpoint remains readable for availability checks. Other API requests require `Authorization: Bearer <token>`. Put an exposed installation behind HTTPS and an appropriate firewall or reverse proxy. Do not expose a raw Firecracker socket.

## What v2.0.1 provides

The v2.0.1 major release provides a browser-first remote control surface for an already-installed Firecracker runtime. It includes host/runtime readiness, Docker Hub and GitHub image import, OCI artifact conversion, GitHub deployment YAML validation, artifact cataloging, microVM creation, start, stop, pause, resume, snapshots, persistent and ephemeral storage, TAP port mappings, vsock guest terminal access, lifecycle events, live host metrics, durable resource state, and explicit workload deletion and cleanup. It keeps the runtime installer separate and does not require Docker or containerd at runtime.

The first-run experience is intentionally honest. Missing Firecracker binaries, jailer, KVM permissions, kernel/rootfs assets, or host utilities appear as readiness information rather than a false “ready” state. A workload is not presented as healthy merely because a browser button was clicked.

Networking, guest application stdout, snapshots, remote workers, and broad image compatibility remain supported only where the installed runtime and host configuration actually provide them. The UI labels lifecycle events separately from guest logs. Unsupported operations should be treated as unavailable rather than simulated.

## Build from a checkout

```bash
npm ci --prefix frontend
npm run build --prefix frontend
rm -rf internal/web/dist
mkdir -p internal/web/dist
cp -R frontend/dist/. internal/web/dist/
go build -trimpath -ldflags '-s -w' -o firecracker-studio ./cmd/firecracker-studio
FIRECRACKER_STUDIO_LISTEN=127.0.0.1:7822 ./firecracker-studio
```

## Development

Start the Go API:

```bash
go run ./cmd/firecracker-studio
```

Start the Vue development server in another terminal:

```bash
npm run dev --prefix frontend
```

Vite proxies `/api/v1` to `http://127.0.0.1:7822`. Set `VITE_FIRECRACKER_API_URL` only when intentionally connecting the UI to another Studio server.

## Windows and WSL2

On Windows, run the Go server inside WSL2 Ubuntu and open the WSL2 listener from the Windows browser. Firecracker remains a Linux process because it requires KVM. Native Windows execution is not claimed by the Studio binary alone.

## Repository layout

```text
cmd/firecracker-studio/  Go web-server entrypoint
frontend/                Vue application and browser development tooling
internal/api/            narrow HTTP control API and readiness checks
internal/converter/      OCI/Docker image conversion
internal/images/         local artifact and base-image catalog
internal/operations/     conversion job queue and operation state
internal/runtime/        installed-runtime status and optional bootstrap logic
internal/web/            embedded Vue assets and SPA serving
internal/worker/         Firecracker lifecycle and Unix-socket service
docs/                    installation, architecture, and workflow guides
scripts/                 separate runtime/server installation helpers
```

## Current boundary

The unified Go web binary and browser API are implemented for v2.0.1 as a **single-host remote management platform** focused on making an already-installed Firecracker runtime approachable through the Vue dashboard. It does not claim to be a cloud deployment platform or complete container compatibility layer. Runtime behavior still depends on host KVM permissions, Firecracker and jailer availability, kernel/rootfs assets, guest-agent support for terminal operations, and the configured network and storage environment.

## License

The repository is currently private while the architecture is stabilized. The intended release model is fully open source under a permissive license after the foundation is reviewed.
