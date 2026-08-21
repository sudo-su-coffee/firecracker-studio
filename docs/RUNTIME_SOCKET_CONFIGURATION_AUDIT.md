# Runtime, Socket, and Configuration Audit

## Verified end-to-end path

The Go worker creates one private runtime directory per VM under `artifact_dir/<vm-id>/`, allocates `firecracker.sock`, and launches Firecracker with `--api-sock <that path>`. The Go Firecracker client uses an HTTP transport whose dialer connects to that Unix socket. Machine configuration, boot source, network interface, vsock, drive, start, stop, pause, resume, and snapshot operations therefore use the Firecracker Unix API socket rather than a remote TCP endpoint.

Guest command execution uses the same VM client to connect to `<artifact_dir>/<vm-id>/vsock.sock`, sends `CONNECT <guest_agent_port>`, and communicates with the guest agent. The default guest CID is `3` and the default guest-agent port is `5000`; both are now configurable through `config.toml` and are used by VM creation and guest command execution.

## Configuration now wired

| Setting | Default | Runtime consumer |
|---|---:|---|
| `listen` | `127.0.0.1:7822` | Go HTTP server bind address |
| `public_url` | `http://127.0.0.1:7822` | Secure-cookie decision and CORS origin fallback |
| `state_dir` | platform user config directory | catalog and persisted state defaults |
| `artifact_dir` | `<state_dir>/artifacts` | VM directories, Firecracker Unix sockets, logs, and artifacts |
| `worker_timeout` | `30s` | API-to-worker context deadline |
| `firecracker_api_timeout` | `30s` | Firecracker Unix-socket HTTP client timeout |
| `operation_timeout` | `30m` | Image conversion queue job timeout |
| `operation_workers` | `2` | Conversion queue concurrency |
| `guest_agent_port` | `5000` | Vsock guest-agent handshake port |
| `guest_agent_cid` | `3` | Firecracker vsock device configuration |
| `network_cidr` | `172.16.0.0/16` | Deterministic /30 TAP allocation pool |
| `[notifications]` | disabled | SMTP notifier construction and failure alerts |

The SMTP password-file setting is now loaded at startup when notifications are enabled. The notifier emits asynchronous alerts for VM creation failures, lifecycle failures, snapshot failures, and guest-command failures. The public URL is used as the default CORS origin, with `FIRECRACKER_STUDIO_CORS_ORIGIN` available as an explicit override.

## Host networking behavior

VM creation creates a TAP device, assigns a deterministic host/guest /30 pair, enables IPv4 forwarding, installs MASQUERADE, configures the Firecracker network interface, and installs optional iptables DNAT/FORWARD rules for port mappings. The configured `network_cidr` controls the allocation pool. The target host must provide `ip`, `iptables`, `sysctl`, and sufficient privileges, normally through a system service with `CAP_NET_ADMIN` or root privileges.

## Important remaining gaps

The following items are not claimed as complete:

1. The `PUT /api/v1/vms/{id}/vsock` endpoint currently persists the desired vsock registry state but does not apply a live `PUT /vsocks` request to an already running VM. VM creation does configure the Firecracker vsock device and guest command execution uses the configured port.
2. The machine configuration registry endpoints persist desired state, while the worker creation path applies machine configuration directly. A later edit through the registry endpoint does not yet reconcile into a live Firecracker VM.
3. `operation_retention` is validated and documented but operation cleanup/retention scheduling is not yet implemented.
4. Image prune currently returns an explicit dry-run response and does not delete artifacts.
5. Runtime installation uses a hardcoded runtime root and a ten-minute download client timeout; those are not yet controlled by `config.toml`.
6. TAP readiness is a lightweight prerequisite check and does not prove that the service has permission to create TAP devices or mutate iptables. This must be verified on the deployment host.
7. SMTP lifecycle coverage is failure-oriented. Successful-event notifications and conversion-queue failure notifications require a further event-hook integration if those are required for the production acceptance criteria.

## Verification

The changed Go code passed `gofmt`, `go test ./...`, `go vet ./...`, and a production binary build. This audit does not execute Firecracker or mutate host networking in the sandbox. Real socket, KVM, TAP, iptables, guest-agent, SMTP, and reverse-proxy behavior must be verified on the target server using the deployment checklist.
