# Firecracker Studio Vue Screen and API Coverage Report

- Backend route registrations: **58**
- Vue API-client exports: **51**
- Routed screen records: **26**
- Concrete Vue view files: **26**
- Router component mappings: **25**
- Client helpers referenced by the shared real view engine: **47**

> A backend API does not require one separate screen per endpoint. Related endpoints are intentionally grouped into resource screens and action controls. For example, a workload detail screen groups lifecycle, deletion, terminal, and inspection APIs.

## Routed screens

| Route | Component | Main capability group |
|---|---|---|
| `overview` | `OverviewView.vue` | Routed Vue screen |
| `security` | `SecurityView.vue` | Routed Vue screen |
| `metrics` | `MetricsView.vue` | Routed Vue screen |
| `activity` | `ActivityView.vue` | Routed Vue screen |
| `readiness` | `ReadinessView.vue` | Routed Vue screen |
| `images` | `ImagesView.vue` | Routed Vue screen |
| `imageDetail` | `ImageDetailView.vue` | Routed Vue screen |
| `imageCatalog` | `ImageCatalogView.vue` | Routed Vue screen |
| `operations` | `OperationsView.vue` | Routed Vue screen |
| `operationDetail` | `OperationDetailView.vue` | Routed Vue screen |
| `workloads` | `WorkloadsView.vue` | Routed Vue screen |
| `workloadDetail` | `WorkloadDetailView.vue` | Routed Vue screen |
| `workloadProcess` | `WorkloadProcessView.vue` | Routed Vue screen |
| `workloadConfig` | `WorkloadConfigView.vue` | Routed Vue screen |
| `workloadCpu` | `WorkloadCpuConfigView.vue` | Routed Vue screen |
| `workloadBoot` | `WorkloadBootSourceView.vue` | Routed Vue screen |
| `workloadSmt` | `WorkloadSmtView.vue` | Routed Vue screen |
| `workloadConstraints` | `WorkloadConstraintsView.vue` | Routed Vue screen |
| `workloadObservability` | `WorkloadObservabilityView.vue` | Routed Vue screen |
| `workloadVsock` | `WorkloadVsockView.vue` | Routed Vue screen |
| `workloadSnapshots` | `WorkloadSnapshotsView.vue` | Routed Vue screen |
| `imagesKernels` | `KernelCatalogView.vue` | Routed Vue screen |
| `storage` | `StorageView.vue` | Routed Vue screen |
| `importSource` | `ImageImportView.vue` | Routed Vue screen |
| `deploymentYaml` | `DeploymentYamlView.vue` | Routed Vue screen |

## API/client/view interpretation

The current implementation has one shared real API-backed view engine, `RouteView.vue`, plus concrete wrapper components for every routed screen. Each wrapper resolves through the router and the shared engine invokes the relevant API client functions using route parameters and form actions.

## Client helpers currently used by the shared view engine

- `BaseImages`
- `CloneImage`
- `Convert`
- `CreateVM`
- `CreateVolume`
- `DeleteImage`
- `DeleteKernel`
- `DeleteVM`
- `DeleteVolume`
- `GuestCommand`
- `Health`
- `ImageDetail`
- `ImageStorageStats`
- `ImageTemplates`
- `Images`
- `Kernels`
- `Login`
- `Logs`
- `Metrics`
- `OperationDetail`
- `Operations`
- `PruneImages`
- `RegisterImage`
- `RegisterKernel`
- `ResolveGitHub`
- `RuntimeStatus`
- `SnapshotCreate`
- `SnapshotCreateAlias`
- `SnapshotDelete`
- `SnapshotRestore`
- `SnapshotRestoreAlias`
- `SystemInfo`
- `SystemStats`
- `VMAction`
- `VMBootSource`
- `VMCPUConfig`
- `VMConfig`
- `VMConstraints`
- `VMDetail`
- `VMLogs`
- `VMMetrics`
- `VMProcess`
- `VMSMT`
- `VMSock`
- `VMs`
- `ValidateYAML`
- `Volumes`

## Client helpers not directly referenced by the shared view engine

- `AuthStatus`
- `Logout`
- `MetricsStream`
- `SetAuthToken`
