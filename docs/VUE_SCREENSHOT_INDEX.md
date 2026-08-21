# Firecracker Studio Vue Screen Screenshot Index

The current Vue implementation groups related backend operations into resource-oriented screens. There is not a one-screen-per-API rule; each screen can invoke multiple related endpoints through the shared API-backed view engine.

| Screen | Browser path | Screenshot |
|---|---|---|
| Overview | `/` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-42-11_1323.webp` |
| Images & Builds | `/images` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-42-44_1140.webp` |
| Import Image | `/images/import` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-42-52_4663.webp` |
| Workloads | `/workloads` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-43-02_1374.webp` |
| Workload Detail | `/workloads/demo-vm` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-43-37_5215.webp` |
| Machine Configuration | `/workloads/demo-vm/config` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-43-47_5913.webp` |
| Vsock & Terminal | `/workloads/demo-vm/vsock` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-43-57_1794.webp` |
| Workload Snapshots | `/workloads/demo-vm/snapshots` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-44-07_1994.webp` |
| Kernel Catalog | `/kernels` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-44-15_1453.webp` |
| Storage | `/storage` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-44-25_1704.webp` |
| Security & Sessions | `/security` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-44-35_9645.webp` |
| Operations | `/operations` | `/home/ubuntu/screenshots/localhost_2026-08-21_02-44-49_5988.webp` |

All screenshots show the real routed interface. The displayed `HTTP 500` state is expected in the screenshot environment because the Go control API is not running; the views are making real HTTP requests rather than rendering fake success data.
