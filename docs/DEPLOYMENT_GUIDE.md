# Firecracker Studio User Manual and Deployment Guide

**Version:** 1.4.x repository line  
**Audience:** Operators deploying Firecracker Studio on a Linux server with KVM  
**Author:** Manus AI

## 1. What Firecracker Studio provides

Firecracker Studio is a browser-operated control plane for managing Firecracker microVM workloads on a Linux host. The Go service owns privileged operations and communicates with each Firecracker process over a private Unix socket. The Vue application is embedded into the Go binary and is served from the same origin as the API.

The normal operator workflow is:

```text
Install the Studio server
→ install the Firecracker runtime
→ configure config.toml
→ expose the web service through HTTPS
→ sign in once in the browser
→ import an image or GitHub source
→ run and manage a microVM
```

Firecracker itself is a VMM, not a container runtime. It does not provide an SSH server or a shell inside the guest. Guest terminal access therefore depends on a compatible guest agent over vsock or another supported guest-console path. Firecracker networking also requires host-side configuration around TAP devices, routing, and port exposure; the VMM does not automatically configure a complete host network for an application.[1] [2]

> **Release-scope note.** The repository includes the browser shell, Go API, embedded frontend, image catalog, lifecycle foundations, resource registries, TAP/vsock integration points, and the documented 58-route API surface. Some advanced routes remain capability surfaces or compatibility handlers rather than universal implementations—for example, complete ballooning, live drive mutation, limiter configuration, full snapshot disk orchestration, and per-VM metrics/log routing. Verify those capabilities on the target host and Firecracker version before relying on them in production.[3]

## 2. Supported deployment model

Firecracker Studio is designed for a **single Linux server** that hosts the control plane and the microVMs. The server should be reachable through a private network or a public domain protected by HTTPS and an upstream access policy.

| Component | Responsibility |
|---|---|
| Linux host | Runs the Studio service, Firecracker, jailer, KVM, TAP networking, state files, image artifacts, and optional persistent volumes. |
| Firecracker Studio Go service | Serves the Vue UI, authenticates operators, persists control-plane state, manages operations, and calls Firecracker over per-VM Unix sockets. |
| Embedded Vue UI | Provides dashboard, workload, image/import, readiness, logs, storage, snapshots, settings, and terminal views. |
| Firecracker | Runs each microVM and exposes its control API through a private Unix socket. |
| Guest image | Supplies the kernel/root filesystem and, if terminal access is required, a compatible Studio guest agent. |
| Reverse proxy | Terminates TLS and forwards requests to the Studio listener. This is recommended for any remote deployment. |

The current installer separates the two concerns deliberately: `install-server.sh` installs the prebuilt Studio server, while `install-runtime.sh` installs the Firecracker/jailer runtime artifacts.[4] [5]

## 3. Host prerequisites

Use a supported 64-bit Linux server with hardware virtualization enabled. The host must expose `/dev/kvm` to the service account or the service must run with the privileges required by the selected Firecracker/jailer configuration. The runtime installation also needs permission to install or locate Firecracker and jailer binaries.

Before installation, confirm the basic host state:

```bash
uname -a
command -v curl
command -v tar
ls -l /dev/kvm
```

A missing `/dev/kvm` normally means that virtualization is disabled in firmware, unavailable from the parent virtual machine, or not loaded by the host kernel. Firecracker networking additionally requires the privileges needed to create TAP devices and configure host routing or NAT.[2]

For a remote installation, prepare the following before opening the service publicly:

| Requirement | Recommended value |
|---|---|
| DNS | `studio.example.com` points to the server address. |
| TLS | HTTPS terminated by a reverse proxy or load balancer. |
| Firewall | Only expose HTTPS; keep the Studio backend listener private. |
| Service account | Dedicated account with the minimum permissions required by the runtime setup. |
| State directory | A persistent filesystem with sufficient space for image artifacts, resource state, VM state, and optional volumes. |
| Backups | Back up the state file, image catalog metadata, imported artifacts, snapshots, and persistent volumes according to their importance. |

## 4. Install the server and runtime

Clone the repository if you are installing from source or inspecting the scripts:

```bash
git clone https://github.com/sudo-su-coffee/firecracker-studio.git
cd firecracker-studio
```

### 4.1 Install the Studio server

The server installer downloads a release asset from GitHub and, when systemd is available, creates and starts `firecracker-studio.service`. The default listener is `127.0.0.1:7822`, which is appropriate when a reverse proxy is placed in front of it.

```bash
sudo FIRECRACKER_STUDIO_VERSION=v2.0.0 \
  FIRECRACKER_STUDIO_LISTEN=127.0.0.1:7822 \
  bash scripts/install-server.sh
```

The relevant installer variables are:

| Variable | Default | Purpose |
|---|---|---|
| `FIRECRACKER_STUDIO_REPO` | `sudo-su-coffee/firecracker-studio` | GitHub repository containing release assets. |
| `FIRECRACKER_STUDIO_VERSION` | `latest` | Release version to install. Pin a version for repeatability. |
| `FIRECRACKER_STUDIO_INSTALL_DIR` | `/opt/firecracker-studio` | Server installation directory. |
| `FIRECRACKER_STUDIO_BIN` | `/usr/local/bin/firecracker-studio` | Installed binary path. |
| `FIRECRACKER_STUDIO_LISTEN` | `127.0.0.1:7822` | HTTP listener address. |

After installation, inspect the service:

```bash
sudo systemctl status firecracker-studio
sudo journalctl -u firecracker-studio -f
curl -i http://127.0.0.1:7822/api/v1/health
```

If systemd is not available, start the binary manually:

```bash
FIRECRACKER_STUDIO_LISTEN=127.0.0.1:7822 \
  /usr/local/bin/firecracker-studio
```

### 4.2 Install the Firecracker runtime

Install the runtime separately:

```bash
bash scripts/install-runtime.sh
```

The runtime installer uses `FIRECRACKER_STUDIO_HOME` when set; otherwise it uses `${XDG_CONFIG_HOME:-$HOME/.config}/firecracker-studio` as its installation prefix. Review the script output after installation and confirm that the Firecracker and jailer binaries are available to the Studio service account.

```bash
find "${FIRECRACKER_STUDIO_HOME:-$HOME/.config/firecracker-studio}" -maxdepth 3 -type f -executable -print
command -v firecracker
command -v jailer
```

If the service runs under systemd as a different user, make sure that user can execute the installed binaries and access the required runtime directories.

## 5. Configure `config.toml`

Create a private configuration file from the repository example:

```bash
sudo install -d -m 0750 /etc/firecracker-studio
sudo cp config.example.toml /etc/firecracker-studio/config.toml
sudo chmod 0600 /etc/firecracker-studio/config.toml
sudoedit /etc/firecracker-studio/config.toml
```

The exact configuration fields are defined in `config.example.toml` and the Go configuration package. At minimum, configure the listener, state/artifact locations, administrator identity, and notification settings. Do not place a plaintext production password in a world-readable file. Use the password-hash field supported by the current configuration schema.

A representative configuration shape is:

```toml
[server]
listen = "127.0.0.1:7822"
public_url = "https://studio.example.com"

[admin]
username = "admin"
password_hash = "REPLACE_WITH_BCRYPT_HASH"
email = "admin@example.com"

[storage]
state_dir = "/var/lib/firecracker-studio"
image_dir = "/var/lib/firecracker-studio/images"
resource_state = "/var/lib/firecracker-studio/resources.json"

[notifications]
enabled = true
smtp_host = "smtp.example.com"
smtp_port = 587
smtp_username = "studio@example.com"
smtp_password = "REPLACE_WITH_SECRET"
from = "studio@example.com"
recipient = "admin@example.com"
```

Use the exact key names shipped in the repository’s `config.example.toml`; the example above illustrates the deployment intent and should not replace the checked-in schema. After editing, restart the service and inspect the journal:

```bash
sudo systemctl restart firecracker-studio
sudo journalctl -u firecracker-studio -n 100 --no-pager
```

The resource registry path is controlled by `FIRECRACKER_STUDIO_RESOURCE_STATE` and defaults to `resources.json` when unset. Keep this file on persistent storage because it contains machine configurations, kernel registrations, volumes, and vsock configuration.[6]

## 6. Expose the UI through a domain

Keep the Go listener private and use a reverse proxy for TLS. A minimal Nginx-style configuration is:

```nginx
server {
    listen 443 ssl http2;
    server_name studio.example.com;

    ssl_certificate     /etc/letsencrypt/live/studio.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/studio.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:7822;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 300s;
    }
}
```

The UI and API are same-origin, so do not expose the Firecracker Unix sockets, `/dev/kvm`, image directories, state files, or the internal listener directly to the browser. The browser should communicate only with the HTTPS Studio domain.

## 7. First login and readiness checks

Open the configured domain:

```text
https://studio.example.com
```

Sign in with the administrator account configured in `config.toml`. Then open **Host Readiness** and review KVM, Firecracker, jailer, kernel, rootfs, storage, and network checks before importing an image.

The first operational checks are:

```bash
curl -fsS https://studio.example.com/api/v1/health
curl -fsS https://studio.example.com/api/v1/readiness
```

Do not start workloads until the host readiness view reports that the required runtime prerequisites are available. A successful HTTP health response only proves that the control service is reachable; it does not prove that a microVM can boot.

## 8. Import an image and create a microVM

Open **Images & Builds** and choose one of the supported source forms exposed by the UI:

| Source | Operator action |
|---|---|
| Docker Hub/OCI image | Enter a reference such as `nginx:latest`, review the source details, and choose **Import image**. |
| GitHub repository | Enter the public repository URL. Studio resolves and pins the repository to a commit before preparation. |
| GitHub YAML | Paste or select the supported constrained YAML input, validate it, and then import the referenced workload. |
| Firecracker archive | Import an already prepared Firecracker image artifact when supported by the repository workflow. |

The user-facing operation is **Import image**. Internally, Studio prepares the kernel, root filesystem, and boot metadata required by Firecracker. The imported image should be treated as a stored Firecracker image record with source reference, digest or commit, architecture, artifact paths, checksum, size, and readiness state.

After the import operation succeeds:

1. Open the image detail or completed operation.
2. Choose **Create Workload** or **New MicroVM**.
3. Select the image and configure vCPU, memory, ports, and storage mode.
4. Choose **ephemeral** for stateless applications such as a frontend or disposable API instance.
5. Choose **persistent** and attach a retained volume for stateful workloads such as a database.
6. Create the workload and wait for readiness before exposing it publicly.

## 9. Workload operations

Open **Workloads** and select a microVM. The workload detail panel exposes the supported lifecycle actions:

| Action | Intended state transition |
|---|---|
| Start | Created or stopped → running, after Firecracker configuration succeeds. |
| Stop | Running or paused → stopped, with runtime cleanup according to storage policy. |
| Pause | Running → paused, when supported by the runtime. |
| Resume | Paused → running. |
| Restart | Stop and start sequence; use only when the image and storage policy permit it. |
| Delete | Removes the workload and associated ephemeral resources; retained persistent volumes require separate handling. |
| Snapshot | Captures a supported VM state and its required artifact references. Validate disk and runtime compatibility before relying on restore. |

The Go backend is the authoritative lifecycle controller. The Vue UI should not infer success merely because a button was clicked; wait for the operation status and inspect the resulting VM state.

## 10. Networking and ports

A Firecracker network interface normally connects to a host TAP device. Studio’s network layer allocates the interface metadata, configures the Firecracker network device, applies declared port mappings, and removes host-side resources when the workload is deleted.

A typical port mapping is:

```text
Host port 8080 → Guest port 80/tcp
```

The guest must also be configured with an appropriate IP address, route, and service listener. A port mapping alone does not make an application reachable if the guest process is not listening or the host firewall blocks the connection.

Recommended checks:

```bash
ip link show
ip addr show
sudo nft list ruleset
sudo iptables -t nat -S 2>/dev/null || true
```

Do not manually delete a TAP device or NAT rule belonging to a running workload. Use the UI’s delete or network action so persisted metadata and host resources remain consistent.

## 11. Guest terminal and vsock

Firecracker does not provide built-in SSH. The Studio terminal uses a host-side vsock Unix-domain-socket path and a guest agent protocol. A compatible guest image must include the Studio agent and listen on the configured guest agent port.

In the UI, open a workload and use **Guest Terminal**. Enter a command such as:

```bash
uname -a
```

If the terminal reports that the agent is unavailable, verify all of the following:

1. The guest image contains the Studio guest agent.
2. The Firecracker VM has a vsock device configured.
3. The host-side vsock socket exists in the VM runtime directory.
4. The guest agent is listening on the configured port.
5. The workload is running and its agent capability is reported as available.

The browser must never receive direct access to the raw vsock socket. Commands should flow through the authenticated Go API, which can apply timeouts, audit records, output limits, and authorization checks.

## 12. Storage, snapshots, and recovery

Use the following storage policy as the default operational rule:

| Workload type | Recommended storage mode |
|---|---|
| Static frontend, disposable preview, stateless API | Ephemeral. The writable layer may be removed when the workload is deleted. |
| Database, queue, uploaded files, or durable application state | Persistent. Attach a retained volume and back it up independently. |
| Build/test workload | Ephemeral unless test outputs must survive termination. |
| Snapshot-based fast recovery | Persistent or explicitly retained artifacts, with snapshot compatibility verified for the exact runtime configuration. |

The **Storage** page lists persisted volumes and prevents deletion of an attached volume. Back up the volume data separately from the control-plane JSON state. A snapshot is not a replacement for a database backup: it represents a runtime state and references disk artifacts that must remain available.

For recovery, first confirm that the kernel, root filesystem, drive files, VM configuration, and Firecracker version are compatible with the snapshot. Restore into a controlled workload state, inspect logs, and verify network and application health before switching production traffic.

## 13. Logs, metrics, and operations

Use **Activity Logs** for control-plane lifecycle events and the workload detail view for boot logs. The API exposes system and per-VM observability routes, while the Vue dashboard summarizes host and workload activity.

Useful checks include:

```bash
sudo journalctl -u firecracker-studio -f
sudo systemctl status firecracker-studio
curl -fsS https://studio.example.com/api/v1/system/info
curl -fsS https://studio.example.com/api/v1/system/stats
```

Long-running imports, snapshots, restores, and deployments should be followed through the operation status in the UI. If an operation fails, inspect its error field, the Studio journal, the workload boot log, and the Firecracker process log before retrying.

## 14. Email notifications

Configure SMTP settings and an administrator recipient in `config.toml` when lifecycle failure alerts are required. The notification foundation is intended for events such as failed start, failed stop, import failure, snapshot failure, and recovery failure. After configuration, trigger a controlled test event in a non-production environment and verify delivery, sender identity, recipient routing, and the absence of credentials in logs.

If email does not arrive, check:

```bash
sudo journalctl -u firecracker-studio | grep -iE 'smtp|mail|notification|alert'
```

Also verify outbound TCP access to the SMTP host, TLS mode, credentials, and any provider-specific sender restrictions.

## 15. Backups and upgrades

Back up these items before upgrading:

| Item | Why it matters |
|---|---|
| `config.toml` | Administrator and service configuration. Store secrets separately when possible. |
| VM state store | Workload identity and lifecycle metadata. |
| `resources.json` or configured resource state | Machine, kernel, volume, and vsock registries. |
| Image catalog and artifact directory | Imported image records and Firecracker boot artifacts. |
| Persistent volumes | Application data. |
| Snapshots | Recovery points and their referenced disk artifacts. |

Upgrade procedure:

```bash
sudo systemctl stop firecracker-studio
sudo cp -a /var/lib/firecracker-studio /var/backups/firecracker-studio-$(date +%Y%m%d-%H%M%S)
sudo FIRECRACKER_STUDIO_VERSION=vNEXT bash scripts/install-server.sh
bash scripts/install-runtime.sh
sudo systemctl start firecracker-studio
sudo systemctl status firecracker-studio
```

After an upgrade, validate health, readiness, image listing, volume listing, one non-production workload, logs, and any required terminal or snapshot capability before returning traffic.

## 16. Troubleshooting

### The web page does not open

Confirm that the service is listening locally, that the reverse proxy points to the correct address, and that the firewall permits HTTPS:

```bash
sudo systemctl status firecracker-studio
ss -ltnp | grep 7822
curl -i http://127.0.0.1:7822/api/v1/health
```

### The service starts but workloads cannot boot

Open **Host Readiness**, inspect the service journal, confirm `/dev/kvm`, verify the Firecracker and jailer paths, and check that the selected image has a valid kernel and root filesystem. A healthy API is not equivalent to a bootable microVM.

### Image import fails

Confirm the source reference, network access to Docker Hub or GitHub, architecture compatibility, available disk space, and the operation error. For GitHub imports, verify that the repository is public and that the selected branch or commit contains a supported Dockerfile, artifact, or constrained YAML definition.

### The guest terminal is unavailable

The image probably does not contain the Studio guest agent, or the vsock device/agent port is not configured. Firecracker does not supply SSH automatically. Use boot logs and the vsock status route to identify the missing capability.

### A volume cannot be deleted

The backend intentionally rejects deletion while the volume is attached to a VM. Stop or detach the workload through the UI first, then retry deletion.

### A workload loses its data after restart

Check whether it was created with `ephemeral` storage. Ephemeral storage is appropriate for stateless applications and is not a substitute for a persistent volume. Recreate the workload with persistent storage and back up the resulting volume.

### Remote access is unsafe

Do not bind the Studio service publicly without TLS and an access policy. Keep the backend listener private, protect the configuration file, rotate administrator credentials, and restrict the server’s firewall and SSH access.

## 17. Minimal operator checklist

Before declaring a deployment ready, confirm:

```text
[ ] Firecracker Studio server installed
[ ] Firecracker and jailer runtime installed
[ ] /dev/kvm available
[ ] config.toml protected and administrator configured
[ ] Persistent state and image directories selected
[ ] HTTPS domain and reverse proxy configured
[ ] Health and readiness checks pass
[ ] Docker Hub or GitHub image import succeeds
[ ] One test microVM starts and stops
[ ] Port mapping reaches the guest service
[ ] Logs appear in the UI
[ ] Guest terminal works for an agent-enabled image, if required
[ ] Ephemeral and persistent storage behavior understood
[ ] Snapshots tested only with compatible artifacts
[ ] Failure notification tested
[ ] Backup procedure documented
```

## References

[1]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/design.md "Firecracker design documentation"

[2]: https://github.com/firecracker-microvm/firecracker/blob/main/docs/network-setup.md "Firecracker network setup documentation"

[3]: https://github.com/sudo-su-coffee/firecracker-studio "Firecracker Studio repository"

[4]: https://github.com/sudo-su-coffee/firecracker-studio/blob/main/scripts/install-server.sh "Firecracker Studio server installer"

[5]: https://github.com/sudo-su-coffee/firecracker-studio/blob/main/scripts/install-runtime.sh "Firecracker Studio runtime installer"

[6]: https://github.com/sudo-su-coffee/firecracker-studio/blob/main/internal/resources/state.go "Firecracker Studio persisted resource state"

[7]: https://github.com/sudo-su-coffee/firecracker-studio/blob/main/docs/API_REFERENCE.md "Firecracker Studio API reference"

[8]: https://github.com/sudo-su-coffee/firecracker-studio/blob/main/config.example.toml "Firecracker Studio example configuration"
