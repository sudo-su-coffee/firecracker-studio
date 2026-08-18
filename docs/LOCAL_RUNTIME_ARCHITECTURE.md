# Firecracker Studio Local Runtime Architecture

Firecracker Studio is a Wails desktop application. Its normal execution path is local and privileged: the Go backend runs on native Linux or inside WSL2, launches the official Firecracker and jailer binaries, and communicates with each microVM through a Unix socket. A browser preview cannot directly access `/dev/kvm`, execute `jailer`, or open a Linux Unix socket. The preview is therefore limited to UI-state testing and must never present itself as a running microVM host.

## One-command Linux/WSL2 bootstrap

Inside native Linux or WSL2 Ubuntu, run:

The repository is currently private. Use authenticated GitHub CLI access without placing a personal token in the URL:

```bash
gh api repos/sudo-su-coffee/firecracker-studio/contents/scripts/install-runtime.sh -H 'Accept: application/vnd.github.raw' | bash
```

If the repository becomes public, the token-free raw form is:

```bash
curl -fsSL https://raw.githubusercontent.com/sudo-su-coffee/firecracker-studio/main/scripts/install-runtime.sh | bash
```

For a cloned repository, use:

```bash
cd firecracker-studio
bash scripts/install-runtime.sh
```

The script installs host utilities, downloads the official Firecracker release archive, verifies its `SHA256SUMS` entry, installs both `firecracker` and `jailer`, creates the Studio runtime directories, and performs a best-effort KVM group setup. It does not modify BIOS/UEFI settings or bypass Linux privilege boundaries.

The script prints the managed runtime directory. The default layout is:

```text
~/.config/firecracker-studio/runtime/bin/firecracker
~/.config/firecracker-studio/runtime/bin/jailer
~/.config/firecracker-studio/runtime/images/default/vmlinux
~/.config/firecracker-studio/runtime/images/default/rootfs.ext4
```

## Desktop execution path

When running as a Wails desktop application, Studio should use this sequence:

| Stage | Responsibility |
|---|---|
| Runtime install | Download and verify Firecracker and jailer |
| Host readiness | Check KVM, TAP/networking, kernel, and rootfs |
| VM allocation | Create an isolated socket and runtime directory per microVM |
| VM launch | Start Firecracker with jailer and the socket path |
| Configuration | Send official Firecracker API requests over the Unix socket |
| Lifecycle | Start, stop, snapshot, restore, inspect logs, and clean up |
| UI refresh | Read local runtime state into dynamic Wails/Vue views |

No TCP URL is needed for local operation. A remote worker connection is a separate optional mode and requires an HTTP endpoint that exposes `/health`.

## Browser preview path

A browser preview cannot run Firecracker. It can display the interface, exercise form validation, and show disconnected or simulated state. It must not claim that KVM, a kernel, a rootfs, or a VM is available.

If browser-based control is required for development, use an explicitly started loopback bridge owned by the user. The bridge must bind to `127.0.0.1`, require authentication, and translate HTTP requests to a local Unix socket. It must not be exposed on a LAN or public interface by default. The Wails desktop app should bypass this bridge and use the Go socket client directly.

## Windows and WSL2

On Windows, the Wails application runs on Windows while the Firecracker runtime runs inside the selected WSL2 distribution. The local adapter must execute Linux commands through WSL2 and keep the Unix socket and VM artifacts inside Linux. The UI should show WSL2 distribution, runtime version, KVM status, artifact status, and actionable repair instructions.

Windows localhost forwarding is relevant only to an HTTP bridge or remote worker. It is not required for direct local Unix-socket control.

## Security boundaries

The bootstrap script uses `sudo` only for package installation and KVM group configuration. It does not silently change host firmware, firewall policy, or privileged networking. Firecracker should be launched through jailer for production workloads, with per-VM directories, restrictive permissions, resource limits, and verified kernel/rootfs artifacts.
