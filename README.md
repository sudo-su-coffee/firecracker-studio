# Firecracker Studio

Firecracker Studio is a **single Go web server with an embedded Vue UI** for operating official Firecracker microVMs. It provides a Docker Desktop-like experience through a browser while keeping KVM, jailer, Firecracker, Unix sockets, kernels, rootfs images, volumes, and snapshots inside a Linux or WSL2 runtime.

The product does not modify Firecracker and does not require Docker or containerd at runtime. The Go backend owns the control API and privileged runtime boundary; the Vue application is the user interface.

## Architecture

```text
Browser Vue UI
   -> same-origin /api/v1
      -> Go Firecracker Studio server
         -> runtime installer and diagnostics
         -> image conversion and artifact catalog
         -> local Unix-socket supervisor
         -> official Firecracker + jailer + KVM/TAP
```

The browser never opens `/dev/kvm`, executes `jailer`, or talks to a raw Firecracker socket. Those operations remain inside the Go process running on native Linux or WSL2 Ubuntu.

## Installation and local use

Firecracker Studio uses two independent installation layers. Install the official Firecracker runtime first, then install the Go web server and embedded Vue UI.

### 1. Install the Firecracker runtime

For an authenticated private repository:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-runtime.sh -H 'Accept: application/vnd.github.raw' | bash
```

For a public repository:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-runtime.sh | bash
```

### 2. Install the Go server and web UI

For an authenticated private repository:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-server.sh -H 'Accept: application/vnd.github.raw' | bash
```

For a public repository:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-server.sh | bash
```

The server installer downloads and verifies the published `FirecrackerStudio-linux-amd64` release binary. It does not clone the repository or build Go/Vue locally. The server listens on `127.0.0.1:7822` by default. For the full Ubuntu/systemd flow, release asset requirements, remote binding guidance, updates, and verification, see [`docs/UBUNTU_SERVER_INSTALL.md`](docs/UBUNTU_SERVER_INSTALL.md).

### Run locally from a checkout

Build and run the unified web server:

```bash
npm ci --prefix frontend
npm run build --prefix frontend
rm -rf internal/web/dist
mkdir -p internal/web/dist
cp -R frontend/dist/. internal/web/dist/
go build -trimpath -ldflags '-s -w' -o firecracker-studio ./cmd/firecracker-studio
FIRECRACKER_STUDIO_LISTEN=127.0.0.1:7822 ./firecracker-studio
```

Open the UI at [http://127.0.0.1:7822](http://127.0.0.1:7822). The health endpoint is [http://127.0.0.1:7822/api/v1/health](http://127.0.0.1:7822/api/v1/health).

## Development

Start the Go runtime API:

```bash
go run ./cmd/firecracker-studio
```

Start the Vue development server in another terminal:

```bash
npm run dev --prefix frontend
```

Vite proxies `/api/v1` to the Go server at `http://127.0.0.1:7822`. Set `FIRECRACKER_STUDIO_API` to use a different local API address.

## Windows and WSL2

On Windows, run the Go server inside WSL2 Ubuntu and open the WSL2 listener from the Windows browser. The official Firecracker and jailer binaries are installed inside WSL2; Firecracker itself remains a Linux process because it requires KVM.

The same web binary can run on native Linux. Remote Linux workers are optional and use an authenticated HTTPS management API. A raw Firecracker Unix socket is never exposed directly to a browser or public network.

## Core capabilities

The initial product surface includes OCI/Docker image conversion, managed Firecracker base images, artifact verification, local and remote runtime status, microVM creation and lifecycle actions, snapshots, logs, volumes, isolated networking groups, diagnostics, and Docker Compose-like multi-service workflows.

The system must reject images that require unsupported kernel features, privileged container behavior, host mounts, Docker sockets, unsupported devices, or incompatible architecture assumptions. Diagnostics should explain why an image or workload cannot boot rather than presenting a false success state.

## Repository layout

```text
cmd/firecracker-studio/  single Go web-server entrypoint
frontend/                Vue application and browser development tooling
internal/api/            authenticated runtime HTTP API
internal/converter/      OCI/Docker image conversion
internal/images/         managed base-image catalog
internal/operations/     conversion job queue and operation state
internal/runtime/        Firecracker/jailer installation and readiness
internal/web/            embedded Vue assets and SPA serving
internal/worker/         Firecracker lifecycle and Unix-socket service
docs/                    architecture, installation, release, and workflow guides
scripts/                 Linux/WSL2 bootstrap scripts
```

## Current boundary

The unified web binary and browser API are implemented. The next runtime milestone is the local Firecracker supervisor: it must start Firecracker through jailer, allocate a Unix socket and artifact directory per VM, apply the kernel/rootfs/network configuration, and expose real lifecycle state through `/api/v1`. Until that supervisor is complete, the API and UI should report missing runtime artifacts or unavailable local execution explicitly.

## License

The repository is currently private while the architecture is stabilized. The intended release model is fully open source under a permissive license after the foundation is reviewed.
