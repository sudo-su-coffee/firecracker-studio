# Firecracker Studio v2.0.1 Final Release Checklist

**Release status:** Ready for publication after repository validation. This release is the stabilized follow-up to v2.0.0 and includes the final Go runtime, configuration, persistence, networking, notification, installer, and operational Vue corrections.

## 1. Final release scope

Firecracker Studio v2.0.1 is a browser-first control plane for a single host running Firecracker microVMs. The Go server owns the Firecracker Unix-socket lifecycle, durable resource state, image catalog, conversion operations, TAP networking, vsock guest-agent access, authentication, notifications, and the embedded Vue dashboard. The Vue application exposes operational resource screens rather than a route-map demonstration.

| Area | Final status |
|---|---|
| Go API specification | 58 registered routes |
| Vue API integration | 58 real backend call paths |
| Vue browser navigation | 26 routed screens with concrete view components |
| Firecracker API socket | Configured per VM under the artifact directory |
| Guest vsock terminal | Configurable guest CID and agent port; live commands use the VM vsock socket |
| VM configuration | Persisted and reconciled to running or paused VMs where Firecracker permits the update |
| Resource persistence | Defaults to `<state_dir>/resources.json` and supports explicit environment override |
| Image pruning | Real failed-image cleanup with explicit retention period and opt-in artifact deletion |
| Operation retention | Automatic cleanup of completed and failed operations older than `operation_retention` |
| Runtime installation | Configurable runtime root and download timeout |
| Notifications | VM, snapshot, guest-command, and conversion failure notifications through configured SMTP |
| Reverse proxy | `public_url`, secure cookies, CORS origin, credentials, and preflight methods are wired |
| Installer | Systemd service can receive `config.toml` and public URL settings |

## 2. Configuration required for deployment

Copy `config.example.toml` to the configured system location and protect it with mode `0600`. The duration fields accept human-readable values such as `30s`, `10m`, `30m`, and `24h`. The application now parses those documented values directly.

```toml
app_name = "firecracker-studio"
listen = "127.0.0.1:7822"
public_url = "https://studio.example.com"
state_dir = "/var/lib/firecracker-studio"
artifact_dir = "/var/lib/firecracker-studio/artifacts"
runtime_root = "/var/lib/firecracker-studio/runtime"
runtime_download_timeout = "10m"
operation_retention = "24h"
worker_timeout = "30s"
operation_timeout = "30m"
firecracker_api_timeout = "30s"
operation_workers = 2
guest_agent_port = 5000
guest_agent_cid = 3
network_cidr = "172.16.0.0/16"

[admin]
username = "admin"
password_hash = "REPLACE_WITH_BCRYPT_HASH"
email = "admin@example.com"

[notifications]
enabled = true
smtp_host = "smtp.example.com"
smtp_port = 587
smtp_username = "studio@example.com"
smtp_password_file = "/etc/firecracker-studio/smtp-password"
from = "studio@example.com"
recipients = ["admin@example.com"]
```

For remote access, keep `listen` private and terminate TLS at a reverse proxy. Set `public_url` to the externally visible HTTPS origin. If the proxy origin differs from `public_url`, set `FIRECRACKER_STUDIO_CORS_ORIGIN`. The proxy must forward cookies, authorization headers, WebSocket or streaming connections where used, and all HTTP methods used by the UI.

## 3. Runtime and guest requirements

The target host must provide KVM access, the Firecracker and jailer binaries, a readable kernel and rootfs artifact, permission to create TAP devices, permission to configure forwarding and iptables rules, and sufficient access to the configured state, artifact, and runtime directories. Guest terminal operations additionally require a guest agent that listens on the configured vsock guest-agent port and understands the `CONNECT <port>` handshake used by the server.

VM creation configures the Firecracker API Unix socket, machine configuration, boot source, rootfs drive, TAP interface, optional port mappings, and vsock device. Lifecycle and snapshot operations are sent through the same Firecracker API client over the per-VM Unix socket. The server never expects the browser to access Firecracker sockets directly.

## 4. Validation completed

| Validation | Result |
|---|---|
| Vue production build | Passed; Vite transformed 54 modules |
| Embedded Vue distribution | Rebuilt and synchronized into `internal/web/dist` |
| Go unit/package tests | Passed: `go test ./...` |
| Go static analysis | Passed: `go vet ./...` |
| Go production binary | Built successfully |
| Installer syntax | `bash -n scripts/install-server.sh` and `bash -n scripts/install-runtime.sh` passed |
| Router syntax | `node --check frontend/src/router.js` passed |
| Backend route count | 58 registrations verified from `internal/api/server.go` |
| Config duration regression | Passed for `48h`, `45s`, `1h`, `20s`, and `5m` |
| Local control-plane startup | Passed with temporary config.toml |
| Health endpoint | Returned service status and v2.0.1 |
| System-info endpoint | Returned configured runtime paths and readiness information |
| Domain CORS preflight | Returned 204 with origin, credentials, and GET/POST/PUT/PATCH/DELETE/OPTIONS headers |

The local smoke test deliberately did not start Firecracker or mutate host networking. It verified the Go control plane, configuration parser, HTTP server, health path, system-info path, and reverse-proxy preflight behavior only.

## 5. Host acceptance checklist

The following items require execution on the real deployment server and cannot be proven by a repository-only sandbox build:

- Confirm `/dev/kvm` is readable and writable by the service account or service capability set.
- Confirm `firecracker` and `jailer` are executable from `runtime_root/bin` or the configured runtime installation path.
- Confirm the configured kernel and rootfs artifacts exist and are readable by the service account.
- Confirm `ip`, `iptables`, and `sysctl` are installed and that the service can create TAP devices, enable forwarding, and install/remove NAT and forwarding rules.
- Confirm a test VM can be created, started, paused, resumed, stopped, and deleted through the browser UI.
- Confirm a guest agent accepts the configured vsock CID, socket, port, and command protocol.
- Confirm snapshot create, restore, alias routes, and deletion work against a running Firecracker VM.
- Confirm SMTP delivery, sender identity, recipient routing, and secret-file permissions.
- Confirm the reverse proxy preserves cookies, authorization headers, CORS origin, streaming metrics, and direct history-mode routes.
- Confirm backup and restoration of `state_dir`, `artifact_dir`, runtime artifacts, and configuration secrets.

## 6. Final release decision

The repository-level implementation is complete and validated. No additional code gap remains in the audited Go-to-Firecracker socket path, configuration parser, embedded Vue bundle, API-to-UI mapping, persistence default, installer service wiring, operation retention, failed-image cleanup, or failure-notification coverage. The remaining acceptance items are host and infrastructure conditions rather than missing repository functionality.
