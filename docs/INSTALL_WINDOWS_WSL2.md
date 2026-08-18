# Firecracker Studio Installation

Firecracker Studio is now a **single Go web server with an embedded Vue UI**. It does not require a separate desktop shell or browser runtime installation. The same binary runs on Linux, WSL2, Windows, and remote Linux hosts; Firecracker execution itself remains on Linux/WSL2 where KVM is available.

## Build the pure Go web binary

From the repository root:

```bash
npm ci --prefix frontend
npm run build --prefix frontend
rm -rf internal/web/dist
mkdir -p internal/web/dist
cp -R frontend/dist/. internal/web/dist/
go build -trimpath -ldflags '-s -w' -o firecracker-studio ./cmd/firecracker-studio
```

Start the unified application:

```bash
FIRECRACKER_STUDIO_LISTEN=127.0.0.1:7822 ./firecracker-studio
```

Open the Vue UI at:

```text
http://127.0.0.1:7822
```

The API is served by the same process under `/api/v1`. The health endpoint is:

```text
http://127.0.0.1:7822/api/v1/health
```

GitHub Actions now builds the pure Go web binaries from **Actions → Firecracker Studio Web Release → Run workflow**. The release assets are standalone binaries, not installers.

## End-user Windows requirements

The user should install WSL2 with Ubuntu once, then install Firecracker Studio. On an elevated PowerShell terminal:

```powershell
wsl --install -d Ubuntu
wsl --set-default-version 2
wsl --update
```

Restart Windows when requested. On the first Ubuntu launch, create the Linux username and password. Verify the distribution with:

```powershell
wsl --list --verbose
```

The distribution must show version `2`.

## What the installer does

The Windows installer installs the Firecracker Studio desktop application. From the dashboard, **Install Firecracker** downloads and verifies the official `firecracker` and `jailer` binaries. On Windows, the download is placed inside the selected WSL2 Ubuntu user environment. On Linux, the binaries are placed in the Studio-managed runtime directory.

Studio does not silently change BIOS settings, force a restart, create privileged TAP devices, modify firewall policy, or change Linux permissions. It reports those requirements clearly so the user can approve and complete them.

## WSL2/Linux readiness

Inside Ubuntu or native Linux, install the basic host utilities:

```bash
sudo apt-get update
sudo apt-get install -y curl ca-certificates iproute2 iptables util-linux acl e2fsprogs
```

After using **Install Firecracker** in Studio, verify the runtime from the dashboard or check the managed files:

```bash
find ~/.config/firecracker-studio/runtime/bin -maxdepth 1 -type f -executable -print
command -v firecracker || true
command -v jailer || true
ls -l /dev/kvm || true
[ -r /dev/kvm ] && [ -w /dev/kvm ] && echo 'KVM access: OK' || echo 'KVM access: FAIL'
ip link show
```

Firecracker also requires a compatible uncompressed kernel image and an ext4 rootfs image. Studio’s managed catalog is responsible for downloading and verifying those artifacts before a microVM can boot.

If the KVM check fails, add the Linux user to the KVM group and start a new shell:

```bash
sudo usermod -aG kvm "$USER"
newgrp kvm
```

On WSL2, restart the distribution from an elevated PowerShell window when required:

```powershell
wsl --shutdown
```

## Linux and WSL2 one-command runtime bootstrap

For a native Linux host or WSL2 Ubuntu, the repository is currently private, so the recommended authenticated one-line bootstrap is:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-runtime.sh -H 'Accept: application/vnd.github.raw' | bash
```

If the repository is made public later, the token-free raw form will be:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-runtime.sh | bash
```

For a cloned repository, run:

```bash
bash scripts/install-runtime.sh
```

The script installs the required host utilities, downloads the official Firecracker release, verifies its checksum, installs both `firecracker` and `jailer`, prepares the Studio runtime directories, and checks KVM access. It does not change BIOS/UEFI settings or bypass Linux privilege boundaries. Reopen the Linux shell or run `newgrp kvm` if the script adds your user to the KVM group.

After the script, launch Firecracker Studio and select **Check local runtime**. The same runtime layout works on native Linux and WSL2.

## Local and remote connections

Local microVM execution is managed directly by Firecracker Studio through the official Unix-socket API. No URL, username, password, or bearer token is required for local runtime operation. The local socket is not a TCP port and must not be entered in the remote worker URL field.

A remote Firecracker worker is optional. To add one, open **Servers → Add remote worker** and provide the worker’s reachable HTTPS URL. Use the username and bearer token issued by that remote worker. The health check must succeed before Studio allows the profile to become active. Never expose an unauthenticated Firecracker management API to the public internet.

## Recommended first-run flow

1. Install and launch WSL2 Ubuntu manually on Windows, or prepare native Linux.
2. Install Firecracker Studio, select **Install Firecracker**, and prepare KVM/TAP access in WSL2 or Linux.
3. Install and launch Firecracker Studio.
4. Open **Dashboard → Install Firecracker**. For a separately managed local or remote Firecracker worker, open **Servers → Add remote worker** and enter its HTTP or HTTPS URL.
5. Select **Check health and add**. Studio switches to the worker only after `GET /health` succeeds.
6. Import a Docker/OCI image or open a Compose file.
7. Studio sends conversion and lifecycle requests to the selected worker; the worker builds artifacts through BuildKit and starts isolated Firecracker microVMs.

## Important distinction

The Windows binary is the Firecracker Studio web server. Open its local URL in a browser. On Windows, the Firecracker runtime is installed inside WSL2 Ubuntu; on native Linux, it is installed directly into the Studio runtime directory. Remote Linux workers are optional and can be added through the server manager with their authenticated HTTP or HTTPS URL.

## References

[3]: https://learn.microsoft.com/en-us/windows/wsl/install "Microsoft WSL installation"
[4]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md "Firecracker getting started"
