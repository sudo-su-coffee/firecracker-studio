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

## What the installer can automate

On first launch, Firecracker Studio should automatically detect WSL, locate or create the configured Ubuntu distribution, check whether `/dev/kvm` exists and is readable/writable, check the Firecracker and `jailer` binaries, check the Linux kernel and base rootfs catalog, check TAP/network capability, create its data directory, start the worker, and call the worker health endpoint. The UI should display a repair action for every failed check.

The installer can safely install the Windows application, create shortcuts, store the selected WSL distribution name, and register the application’s local configuration. It can also invoke `wsl.exe` to inspect status. It should not silently modify firewall rules, kernel files, privileged network interfaces, or user credentials.

## What still requires explicit approval or manual work

WSL installation, Windows restart, BIOS/UEFI virtualization, Hyper-V/Virtual Machine Platform availability, `/dev/kvm` exposure, TAP creation, firewall changes, and privileged Linux group changes can require administrator approval or a host restart. The application must show the exact command and ask the user to approve it instead of pretending that the `.exe` can guarantee success on every Windows host.

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

1. Install and launch Firecracker Studio.
2. Choose **Local WSL2** on Windows or **Local Linux** on Linux.
3. Let Studio run non-destructive checks.
4. Approve only the listed privileged repairs, such as creating a TAP interface or granting `/dev/kvm` access.
5. Studio downloads the selected Alpine default base, kernel, and rootfs artifacts and verifies their digests.
6. Import a Docker/OCI image or open a Compose file.
7. Studio builds artifacts through BuildKit without requiring Docker Engine, then starts isolated Firecracker microVMs.

## Important distinction

The `.exe` is the desktop control plane. It does not turn Windows itself into a Firecracker host. On Windows, Firecracker runs inside the Linux environment provided by WSL2. On native Linux, the application can connect directly to the local worker. Remote Linux workers can be added through the server manager after their authenticated health endpoint is available.

## References

[1]: https://wails.io/docs/guides/windows-installer/ "Wails NSIS installer"
[2]: https://wails.io/docs/gettingstarted/installation "Wails installation requirements"
[3]: https://learn.microsoft.com/en-us/windows/wsl/install "Microsoft WSL installation"
[4]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md "Firecracker getting started"
