# Go-to-Vue Integration Verification

## Executive result

The Firecracker Studio backend and Vue frontend are **code-level integrated and synchronized**. The verification found 58 Go API registrations, 58 router metadata mappings, a 51-helper Vue API client, 26 browser routes, and a rebuilt embedded Vue bundle. Frontend production build, Go tests, and Go vet all passed.

The browser smoke checks confirmed that the operational Overview route and nested workload/vsock route render correctly and issue real requests. The standalone preview returns `HTTP 500` because the Go control API is not running in the preview session; this is a real backend-unavailable response, not simulated data.

## Verification matrix

| Integration layer | Result | Evidence |
|---|---:|---|
| Go backend route registrations | **58 / 58** | `internal/api/server.go` route audit |
| Router endpoint metadata | **58 / 58** | `frontend/src/router.js`; no missing or extra normalized routes |
| Vue API-client exports | **51** | `frontend/src/api.js` |
| Routed browser screens | **26** | `frontend/src/router.js` |
| Concrete Vue view files | **26** | `frontend/src/views/` |
| Real resource/action call paths | **58 / 58** | `RouteView.vue`, `App.vue`, and callback execution paths |
| Frontend production build | Passed | `pnpm build` |
| Go unit/package tests | Passed | `go test ./...` |
| Go static analysis | Passed | `go vet ./...` |
| Embedded bundle present | Passed | `internal/web/dist/index.html` and 28 assets |
| Frontend-to-embedded asset references | Matched | Built `index.html` asset references equal embedded references |
| Browser direct route `/` | Passed | Operational Overview rendered |
| Browser direct nested route `/workloads/demo-vm/vsock` | Passed | Vsock and guest-command controls rendered |

## How the 58 APIs are used

The implementation intentionally groups related endpoints into operational screens rather than creating 58 nearly identical pages. The Workloads screen loads and creates VMs. A workload detail screen handles detail, start, stop, pause, resume, terminal, and delete. Child workload screens handle process inspection, machine configuration, CPU configuration, boot source, SMT, constraints, logs, metrics, vsock, terminal, and snapshots. Image screens handle base images, image catalogs, templates, conversion, GitHub resolution, registration, clone, delete, storage statistics, and prune. Kernel, volume, authentication, readiness, metrics, logs, and operations screens provide their respective live controls.

Four client exports are infrastructure-oriented rather than resource-view functions: `AuthStatus`, `Logout`, `SetAuthToken`, and `MetricsStream`. The first three belong to the application session shell, while `MetricsStream` is used by the metrics screen for live SSE updates. List helpers such as `VMs`, `Kernels`, `Volumes`, and `PruneImages` may appear as callback arguments to the shared `run`/`mutate` functions; those callbacks are executed by the view and are not mock declarations.

## Runtime boundary

This verification does not claim that Firecracker can start in the sandbox preview. Real lifecycle success requires the deployed Go binary, configured admin session, Firecracker and jailer binaries, KVM, kernel/rootfs artifacts, TAP permissions, and any guest-agent/vsock configuration on the target host. Once those are present, the Vue screens call the real HTTP endpoints and render their returned success or error states.

The verification changes are on `main` at commit `940c746`.
