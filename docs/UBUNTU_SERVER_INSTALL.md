# Ubuntu Server Installation

Firecracker Studio is installed in two independent layers. The **runtime installer** installs Firecracker, jailer, and host utilities. The **server installer** installs the Go web server with the embedded Vue UI. Keeping these layers separate makes it possible to update the web control plane without replacing the Firecracker runtime.

## Prerequisites

The server should be Ubuntu 22.04 or newer with `sudo`, network access, and a user that can access the private GitHub repository. Authenticate GitHub CLI once if the repository is private:

```bash
gh auth login
```

## Command 1: install the Firecracker runtime

This installs only the official Firecracker and jailer binaries, verifies the release checksum, prepares the runtime directories, and checks KVM permissions:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-runtime.sh -H 'Accept: application/vnd.github.raw' | bash
```

If the repository becomes public, the equivalent token-free command is:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-runtime.sh | bash
```

The runtime is installed under:

```text
~/.config/firecracker-studio/runtime/bin/firecracker
~/.config/firecracker-studio/runtime/bin/jailer
~/.config/firecracker-studio/runtime/images/default/
```

The runtime installer does not install the Go server or Vue UI.

## Command 2: install the Go backend and web UI

The server installer downloads a prebuilt Linux AMD64 Firecracker Studio binary from GitHub Releases, verifies its SHA256 checksum, installs the binary, and starts a systemd service when systemd is available. It does not clone the repository and does not install Go, Node.js, npm, or frontend build dependencies.

The selected release must contain these Linux assets:

```text
FirecrackerStudio-linux-amd64
SHA256SUMS-linux-amd64.txt
```

If the selected release does not contain a Linux asset, the installer stops instead of compiling locally. Run the release workflow with `publish: true` before installing a new version.

For an authenticated private repository:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-server.sh -H 'Accept: application/vnd.github.raw' | bash
```

For a public repository:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-server.sh | bash
```

The server installer uses these defaults:

| Setting | Default |
|---|---|
| Binary | `/usr/local/bin/firecracker-studio` |
| Install directory | `/opt/firecracker-studio` |
| Listen address | `127.0.0.1:7822` |
| Service | `firecracker-studio.service` |
| Browser URL | `http://127.0.0.1:7822` |
| API health | `http://127.0.0.1:7822/api/v1/health` |

For a server that should be reachable from another machine, explicitly bind to all interfaces:

```bash
FIRECRACKER_STUDIO_LISTEN=0.0.0.0:7822 \
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-server.sh \
  -H 'Accept: application/vnd.github.raw' | bash
```

Then allow the port through the firewall only from trusted networks:

```bash
sudo ufw allow from TRUSTED_CLIENT_IP to any port 7822 proto tcp
```

Do not expose the unauthenticated development server directly to the public internet. Put it behind HTTPS authentication or a private network before remote use.

## Verify the installation

```bash
systemctl status firecracker-studio --no-pager
curl -fsS http://127.0.0.1:7822/api/v1/health
```

Open the web UI in a browser:

```text
http://SERVER_IP:7822
```

The server installer does not install Firecracker, jailer, kernels, rootfs images, or KVM. Those belong to Command 1 and the runtime artifact catalog. A running web server alone is not proof that a microVM can boot.

## Updating only one layer

Update the web server without changing Firecracker by selecting a published release version:

```bash
FIRECRACKER_STUDIO_VERSION=v1.0.3 \
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-server.sh \
  -H 'Accept: application/vnd.github.raw' | bash
```

If `FIRECRACKER_STUDIO_VERSION` is omitted, the installer resolves the latest GitHub Release. Update the Firecracker runtime separately by rerunning `scripts/install-runtime.sh` with the desired pinned `FIRECRACKER_VERSION`.
