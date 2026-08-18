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

The server installer installs Go and Node build tools, obtains the Firecracker Studio source, builds the Vue production assets, embeds them into one Go binary, installs the binary, and starts a systemd service when systemd is available.

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
| Source checkout | `~/src/firecracker-studio` |
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

Update the web server without changing Firecracker:

```bash
sudo systemctl stop firecracker-studio
cd ~/src/firecracker-studio
git pull --ff-only
npm ci --prefix frontend
npm run build --prefix frontend
rm -rf internal/web/dist
mkdir -p internal/web/dist
cp -R frontend/dist/. internal/web/dist/
go build -trimpath -ldflags '-s -w' -o /tmp/firecracker-studio ./cmd/firecracker-studio
sudo install -m 0755 /tmp/firecracker-studio /usr/local/bin/firecracker-studio
sudo systemctl start firecracker-studio
```

Update the Firecracker runtime separately by rerunning `scripts/install-runtime.sh` with the desired pinned `FIRECRACKER_VERSION`.
