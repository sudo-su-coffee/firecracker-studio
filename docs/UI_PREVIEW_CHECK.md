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

## Four-tab Docker Desktop-style inspection

The final preview at `http://127.0.0.1:4173` shows exactly four primary sidebar areas: Dashboard, Images, Containers / MicroVMs, and Convert / Build. Account switching and `+ Account` are visible in the top bar. The local/remote server selector and `+ Server` action are also visible in the top bar. The server connection dialog performs a health check before adding a worker profile and supports local or remote type, URL, username, and bearer token fields.

Four-tab screenshot: `/home/ubuntu/screenshots/127_0_0_1_2026-08-18_06-22-36_9020.webp`

Advanced operations such as terminal, logs, file management, SSH, snapshots, and restore are now intended to live inside the selected microVM detail view rather than as primary navigation tabs. The current UI still contains legacy unreachable templates for those views and they should be consolidated into the microVM detail panel in the next cleanup pass.

## Shadcn-inspired local-first preview

The latest preview uses a neutral zinc/black shadcn-inspired theme with restrained borders, focused inputs, accessible contrast, compact status badges, and clear card hierarchy. The dashboard visibly includes the first-run local worker setup card, health-check action, server manager action, account selector, and local worker selector.

Latest screenshot: `/home/ubuntu/screenshots/127_0_0_1_2026-08-18_06-28-16_2868.webp`

The local API health check returned `{"service":"firecracker-studio","status":"ok"}` and the managed base-image endpoint returned the catalog successfully. A true remote server was not available in the sandbox, but the native Go bridge now stores each remote profile's bearer token in memory and uses it for health checks and server switching.
