# Firecracker Studio UI Preview Check

Date: 2026-08-18

The browser preview was checked at `http://127.0.0.1:4173`. The first preview exposed an expected native-runtime issue because Wails bindings are not available in an ordinary browser. The Vue API adapter was then updated with browser-safe preview fallbacks while retaining native Wails calls. The second preview rendered successfully without the runtime error.

The visible dashboard includes the worker connection URL, connection status, OCI-to-microVM conversion controls, guest-base selection, vCPU and memory inputs, microVM controls, recent operations, imported image library, and six managed base-image catalog profiles.

The six catalog profiles are Alpine 3.24.1 for x86_64 and aarch64, Debian 12 for x86_64 and aarch64, and Ubuntu 22.04 for x86_64 and aarch64. Their current status is `catalog`, meaning metadata is present but the kernel/rootfs artifacts have not yet been downloaded and verified on this host.

Preview screenshot: `/home/ubuntu/screenshots/127_0_0_1_2026-08-18_06-14-17_6332.webp`

This preview confirms the UI composition only. Native Wails execution and Firecracker execution still require the appropriate host backend and worker API.

## Sidebar workspace inspection

The redesigned preview at `http://127.0.0.1:4173` now renders as a Wails/Vue desktop-style control center with navigation for Overview, MicroVMs, Images, Marketplace, Live logs, Terminal, File manager, SSH & access, Snapshots, and Settings. The overview shows worker connection, Firecracker readiness checks, image conversion, workload metrics, and recent microVMs. The browser-safe adapter reports a preview worker state and displays the six managed catalog profiles without requiring native Wails bindings.

Redesigned screenshot: `/home/ubuntu/screenshots/127_0_0_1_2026-08-18_06-17-58_9668.webp`
