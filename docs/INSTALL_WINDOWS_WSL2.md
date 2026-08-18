# Firecracker Studio Installation

## Current release status

Windows v1.0.0 has been published as a GitHub Release. The release contains the NSIS installer, the standalone executable, and SHA256 checksums.

## Windows developer/build command

From a Windows machine with Go, Node.js, Wails, and NSIS installed:

```powershell
$env:PATH += ";$env:USERPROFILE\go\bin"
wails build -clean -platform windows/amd64 -nsis -ldflags "-X main.version=v0.1.0"
```

The generated installer is placed in `build/bin`. Wails uses NSIS to create the installer, and the application itself is a desktop executable rather than a web application.

For a release from GitHub Actions, open **Actions → Windows Release → Run workflow**, enter a version such as `v0.1.0`, and start the workflow. The current end-user command is:

```powershell
irm https://github.com/sudo-su-coffee/firecracker-studio/releases/latest/download/FirecrackerStudioInstaller.exe -OutFile "$env:TEMP\FirecrackerStudioInstaller.exe"; Start-Process "$env:TEMP\FirecrackerStudioInstaller.exe"
```

The published v1.0.0 assets are available from the [GitHub release page](https://github.com/sudo-su-coffee/firecracker-studio/releases/tag/v1.0.0). The SHA256 file should be downloaded and checked before distributing the installer through another channel.

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

## What the installer does not automate

The installer only installs the Firecracker Studio desktop application, creates shortcuts, and stores the application configuration. It does not install WSL2, install Firecracker, start the Porter worker, modify `/dev/kvm`, create TAP interfaces, or change firewall and Linux permissions.

On first launch, the user manually adds a server profile with the worker URL. Studio then performs the health check and allows switching only when the endpoint responds successfully. A connection refusal means that the worker is not running at that URL or that the URL is not reachable from Windows.

## What requires manual work

The user is responsible for WSL installation, the Windows restart, BIOS/UEFI virtualization, Hyper-V/Virtual Machine Platform availability, `/dev/kvm` access, Firecracker and `jailer` installation, kernel/rootfs preparation, TAP networking, firewall rules, and starting the worker process. This separation keeps the desktop app predictable and avoids silently changing privileged host configuration.

## WSL2 backend readiness checks

Inside Ubuntu, run:

```bash
sudo apt-get update
sudo apt-get install -y curl ca-certificates iproute2 iptables util-linux acl e2fsprogs

command -v firecracker || echo 'Firecracker binary is missing'
command -v jailer || echo 'jailer binary is missing'
ls -l /dev/kvm || true
[ -r /dev/kvm ] && [ -w /dev/kvm ] && echo 'KVM access: OK' || echo 'KVM access: FAIL'
ip link show
```

Firecracker requires Linux KVM and read/write access to `/dev/kvm`. It also requires a compatible uncompressed kernel image and an ext4 rootfs image. The Studio catalog should download and verify these artifacts; users should not need to assemble them manually after the managed catalog is complete.

The official Firecracker release binary and `jailer` should be downloaded for the host architecture from the official Firecracker release page, checksum-verified, and installed into the Studio-managed worker directory. Do not download an arbitrary binary from an untrusted mirror.

## Linux one-line bootstrap

For a Linux host where the user wants to install prerequisites and then launch the Studio worker, the planned command is:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-linux.sh | bash
```

This command is not active until `scripts/install-linux.sh` is committed and the repository’s release policy has been finalized. Until then, use the repository’s Linux build workflow and install the official Firecracker artifacts through the Studio catalog.

## Recommended first-run flow

1. Install and launch WSL2 Ubuntu manually on Windows, or prepare native Linux.
2. Install Firecracker and `jailer`, prepare KVM/TAP access, and start the Porter worker manually.
3. Install and launch Firecracker Studio.
4. Open **Servers → Add server**, select **Local**, and enter the worker URL, normally `http://127.0.0.1:7822` when the worker is listening inside the same reachable environment.
5. Select **Check health and add**. Studio switches to the worker only after a successful health response.
6. Import a Docker/OCI image or open a Compose file.
7. Studio sends conversion and lifecycle requests to the selected worker; the worker builds artifacts through BuildKit and starts isolated Firecracker microVMs.

## Important distinction

The `.exe` is the desktop control plane. It does not turn Windows itself into a Firecracker host. On Windows, Firecracker runs inside the Linux environment provided by WSL2. On native Linux, the application can connect directly to the local worker. Remote Linux workers can be added through the server manager after their authenticated health endpoint is available.

## References

[1]: https://wails.io/docs/guides/windows-installer/ "Wails NSIS installer"
[2]: https://wails.io/docs/gettingstarted/installation "Wails installation requirements"
[3]: https://learn.microsoft.com/en-us/windows/wsl/install "Microsoft WSL installation"
[4]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md "Firecracker getting started"
