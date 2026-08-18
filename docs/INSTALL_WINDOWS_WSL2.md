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

## Linux installation

Install the Linux package or archive for the matching release, launch Firecracker Studio, and select **Install Firecracker** from the dashboard. The same flow works on native Linux and does not require a separate control-plane service.

## Local and remote connections

Local microVM execution is managed directly by Firecracker Studio through the official Unix-socket API. No URL, username, password, or bearer token is required for local runtime operation.

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

The `.exe` is the Firecracker Studio desktop application. On Windows, the local runtime is installed inside WSL2 Ubuntu; on native Linux, it is installed directly into the Studio runtime directory. Remote Linux workers are optional and can be added through the server manager with their authenticated HTTP or HTTPS URL.

## References

[1]: https://wails.io/docs/guides/windows-installer/ "Wails NSIS installer"
[2]: https://wails.io/docs/gettingstarted/installation "Wails installation requirements"
[3]: https://learn.microsoft.com/en-us/windows/wsl/install "Microsoft WSL installation"
[4]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md "Firecracker getting started"
