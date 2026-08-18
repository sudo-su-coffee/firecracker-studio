# Firecracker Studio

Firecracker Studio is a cross-platform desktop application for managing Firecracker microVM workloads through a simple, Docker Desktop-like experience. It is designed for Windows, Linux, and later macOS, with a Vue-based interface and a Go/Wails application shell.

Firecracker Studio does not modify Firecracker. It packages the user experience around the official Firecracker VMM and its Unix-socket API. Users can import Docker or OCI images, convert compatible images into standalone Firecracker artifacts, run and inspect microVMs, manage ports and resources, create snapshots, view logs, and connect the desktop UI to a remote Firecracker worker.

## Why Wails

Wails is the preferred desktop framework because Firecracker Studio’s core services are written in Go. Wails permits a modern Vue frontend while keeping the local controller, artifact manager, worker connector, and security-sensitive orchestration in Go. It avoids placing privileged runtime logic in a browser process and avoids requiring a separate Node.js runtime for the application controller.

The application uses a strict process boundary:

```text
Vue UI in Wails
  -> typed local IPC / authenticated local API
    -> desktop controller
      -> privileged Firecracker supervisor
        -> official Firecracker Unix socket, KVM, TAP, cgroups, and jailer
```

Wails v2 is the conservative MVP target. Wails v3 may be evaluated after its packaging and lifecycle APIs are stable enough for production distribution.

## Windows and Linux operation

On Linux, Firecracker Studio can run the supervisor natively when the host has KVM access, TAP support, cgroups, and the required kernel/rootfs capabilities. On Windows, the UI runs natively and the application manages a dedicated WSL2 Linux backend. The app must detect `/dev/kvm`, network capabilities, virtualization support, and permissions before offering local microVM execution.

When local execution is unavailable, the UI can still inspect images, prepare conversions, manage artifacts, and connect to a remote authenticated Firecracker worker.

## Remote workers

Remote workers are first-class. A worker connection includes a human-readable name, endpoint, transport security configuration, authentication material stored in the operating system credential store, capability discovery, health state, and compatibility information. The desktop application never exposes the Firecracker socket directly over the network. A remote worker agent owns the Firecracker process and exposes a narrow authenticated management API.

## Initial capabilities

The first milestone includes Docker Hub and OCI registry image references, local Docker/OCI archive import, OCI manifest and layer inspection, secure conversion into a standalone Firecracker artifact, artifact verification, local and remote worker discovery, microVM start/stop/restart, resource settings, port mappings, console logs, image and VM lists, snapshot create/restore, export, cleanup, and capability diagnostics.

## Non-goals

Firecracker Studio is not a replacement VMM, does not fork or modify Firecracker, does not require Docker or containerd at runtime, and does not promise that every container image is automatically bootable as a microVM. Images that depend on unsupported kernel features, privileged container behavior, host mounts, Docker sockets, unsupported devices, or architecture-specific assumptions must be rejected with actionable diagnostics.

## Relationship to Porter

Firecracker Studio is an independent local and remote-worker desktop product. Porter is the larger self-hosted cloud platform that can consume the same verified artifacts and schedule them across managed workers. Firecracker Studio must remain useful without Porter.

## Repository layout

```text
app/             Wails application and Go bindings
frontend/        Vue application
internal/        controller, worker protocol, artifact, and security packages
docs/            architecture, plans, protocol, and compatibility documentation
build/           platform packaging configuration
```

## Development status

The project is in architecture and scaffold phase. The first implementation target is the local controller and worker protocol, followed by artifact inspection/conversion and the initial Vue desktop shell.

## License

The repository is currently private while the architecture is stabilized. The intended release model is fully open source under a permissive license after the initial foundation is reviewed.
