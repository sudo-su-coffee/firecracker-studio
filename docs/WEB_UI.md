# Firecracker Studio Web UI

Firecracker Studio is a **single Go web server with an embedded Vue application**. The Go process serves the browser UI, exposes the `/api/v1` runtime API, owns Firecracker and jailer integration, and keeps privileged operations on Linux or WSL2. There is no Wails shell in the primary product architecture.

## Build and run

From the repository root:

```bash
npm ci --prefix frontend
npm run build --prefix frontend
rm -rf internal/web/dist
mkdir -p internal/web/dist
cp -R frontend/dist/. internal/web/dist/
go build -trimpath -ldflags '-s -w' -o firecracker-studio ./cmd/firecracker-studio

FIRECRACKER_STUDIO_LISTEN=127.0.0.1:7822 ./firecracker-studio
```

Open:

```text
http://127.0.0.1:7822
```

The same process serves the Vue SPA and API. The API health endpoint is:

```text
http://127.0.0.1:7822/api/v1/health
```

## Development mode

Run the Go runtime API in one terminal:

```bash
go run ./cmd/firecracker-studio
```

Run the Vue development server in another:

```bash
npm run dev --prefix frontend
```

Vite proxies `/api/v1` to `http://127.0.0.1:7822`. A different local API can be selected with:

```bash
FIRECRACKER_STUDIO_API=http://127.0.0.1:7822 npm run dev --prefix frontend
```

The Go API allows only the configured local development origin by default. Set `FIRECRACKER_STUDIO_CORS_ORIGIN` for another trusted origin. Do not use a wildcard origin for a service that controls microVMs.

## Runtime boundary

A browser cannot access KVM, jailer, or Unix sockets. The Go server must run on native Linux or inside WSL2 Ubuntu. It launches or supervises Firecracker and communicates with each microVM through its Unix socket. The browser talks only to the authenticated Go API.

| Layer | Responsibility |
|---|---|
| Vue browser UI | Navigation, forms, images, operations, VM controls, diagnostics |
| Go HTTP server | API, auth boundary, persistence, runtime status, worker orchestration |
| Linux/WSL2 runtime | KVM, TAP, Firecracker, jailer, kernels, rootfs, volumes, sockets |
| Remote worker | Optional HTTPS Firecracker control endpoint |

## Windows and WSL2

On Windows, run the Go web server inside WSL2 for the simplest local setup, then open the WSL2 listener from the Windows browser. The one-command runtime bootstrap is:

Because the repository is currently private, install the runtime with the authenticated GitHub CLI:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-runtime.sh -H 'Accept: application/vnd.github.raw' | bash
```

If the repository becomes public, use the token-free raw URL:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-runtime.sh | bash
```

The bootstrap installs the official Firecracker and jailer binaries, checks KVM access, and prepares the Studio runtime directories. It does not modify BIOS/UEFI settings or bypass privileged host permissions.

## Remote use

For a remote Linux host, run the same Go binary there and put it behind HTTPS authentication. The browser then uses:

```bash
VITE_FIRECRACKER_API_URL=https://firecracker.example.com/api/v1 npm run dev --prefix frontend
```

The remote API must never expose raw Firecracker Unix sockets. It should expose only the narrow authenticated Studio API and enforce TLS, authentication, authorization, request limits, and audit logging.

## Release model

GitHub Actions builds the Vue assets first, copies them into `internal/web/dist`, and compiles one Go binary for each target. The release contains standalone web-server binaries rather than Wails installers. Users run the binary on Linux/WSL2 and open its URL in a browser.
