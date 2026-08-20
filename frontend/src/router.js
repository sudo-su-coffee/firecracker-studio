import { createRouter, createWebHistory } from 'vue-router'

/**
 * Firecracker Studio route registry.
 *
 * Each route declares the backend capability or capabilities rendered by its
 * view in `meta.api`. The API strings intentionally match the Go route
 * registrations under /api/v1 so the UI coverage can be audited from one file.
 */
const views = {
  overview: () => import('./views/OverviewView.vue'),
  security: () => import('./views/SecurityView.vue'),
  metrics: () => import('./views/MetricsView.vue'),
  activity: () => import('./views/ActivityView.vue'),
  readiness: () => import('./views/ReadinessView.vue'),
  images: () => import('./views/ImagesView.vue'),
  imageDetail: () => import('./views/ImageDetailView.vue'),
  imageCatalog: () => import('./views/ImageCatalogView.vue'),
  operations: () => import('./views/OperationsView.vue'),
  operationDetail: () => import('./views/OperationDetailView.vue'),
  workloads: () => import('./views/WorkloadsView.vue'),
  workloadDetail: () => import('./views/WorkloadDetailView.vue'),
  workloadProcess: () => import('./views/WorkloadProcessView.vue'),
  workloadConfig: () => import('./views/WorkloadConfigView.vue'),
  workloadCpu: () => import('./views/WorkloadCpuConfigView.vue'),
  workloadBoot: () => import('./views/WorkloadBootSourceView.vue'),
  workloadSmt: () => import('./views/WorkloadSmtView.vue'),
  workloadConstraints: () => import('./views/WorkloadConstraintsView.vue'),
  workloadObservability: () => import('./views/WorkloadObservabilityView.vue'),
  workloadVsock: () => import('./views/WorkloadVsockView.vue'),
  workloadSnapshots: () => import('./views/WorkloadSnapshotsView.vue'),
  imagesKernels: () => import('./views/KernelCatalogView.vue'),
  storage: () => import('./views/StorageView.vue'),
  importSource: () => import('./views/ImageImportView.vue'),
  deploymentYaml: () => import('./views/DeploymentYamlView.vue'),
}

const route = (path, name, component, api, extra = {}) => ({
  path,
  name,
  component,
  meta: { api: Array.isArray(api) ? api : [api], ...extra },
})

export const routes = [
  route('/', 'overview', views.overview, [
    'GET /api/v1/health',
    'GET /api/v1/metrics',
    'GET /api/v1/system/stats',
    'GET /api/v1/system/info',
    'GET /api/v1/readiness',
  ], { workspace: 'overview' }),
  route('/security', 'security', views.security, [
    'POST /api/v1/auth/login',
    'GET /api/v1/auth/status',
    'POST /api/v1/auth/logout',
  ], { workspace: 'settings' }),
  route('/metrics', 'metrics', views.metrics, [
    'GET /api/v1/metrics',
    'GET /api/v1/vms/:id/metrics',
  ], { workspace: 'observability' }),
  route('/activity', 'activity', views.activity, [
    'GET /api/v1/logs',
    'GET /api/v1/vms/:id/logs',
  ], { workspace: 'activity' }),
  route('/host-readiness', 'readiness', views.readiness, [
    'GET /api/v1/readiness',
  ], { workspace: 'readiness' }),

  route('/images', 'images', views.images, [
    'GET /api/v1/base-images',
    'GET /api/v1/images',
    'GET /api/v1/images/templates',
    'POST /api/v1/conversions',
    'GET /api/v1/operations',
  ], { workspace: 'images' }),
  route('/images/import', 'image-import', views.importSource, [
    'GET /api/v1/sources/github',
    'POST /api/v1/sources/yaml',
    'POST /api/v1/images',
    'POST /api/v1/conversions',
  ], { workspace: 'images' }),
  route('/images/catalog', 'image-catalog', views.imageCatalog, [
    'GET /api/v1/images',
    'GET /api/v1/images/storage-stats',
    'POST /api/v1/images',
    'DELETE /api/v1/images/:digest',
    'POST /api/v1/images/:id/clone',
    'POST /api/v1/images/prune',
  ], { workspace: 'images' }),
  route('/images/:id', 'image-detail', views.imageDetail, [
    'GET /api/v1/images/:id',
    'POST /api/v1/images/:id/clone',
    'DELETE /api/v1/images/:digest',
  ], { workspace: 'images' }),
  route('/operations', 'operations', views.operations, [
    'GET /api/v1/operations',
  ], { workspace: 'operations' }),
  route('/operations/:id', 'operation-detail', views.operationDetail, [
    'GET /api/v1/operations/:id',
  ], { workspace: 'operations' }),
  route('/deployment-yaml', 'deployment-yaml', views.deploymentYaml, [
    'POST /api/v1/sources/yaml',
  ], { workspace: 'images' }),

  route('/workloads', 'workloads', views.workloads, [
    'GET /api/v1/vms',
    'POST /api/v1/vms',
  ], { workspace: 'workloads' }),
  route('/workloads/:id', 'workload-detail', views.workloadDetail, [
    'GET /api/v1/vms/:id',
    'POST /api/v1/vms/:id/start',
    'POST /api/v1/vms/:id/stop',
    'POST /api/v1/vms/:id/pause',
    'POST /api/v1/vms/:id/resume',
    'POST /api/v1/vms/:id/terminal',
    'DELETE /api/v1/vms/:id',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/process', 'workload-process', views.workloadProcess, [
    'GET /api/v1/vms/:id/process',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/config', 'workload-config', views.workloadConfig, [
    'GET /api/v1/vms/:id/config',
    'PUT /api/v1/vms/:id/config',
    'PUT /api/v1/vms/:id/config/patch',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/cpu', 'workload-cpu', views.workloadCpu, [
    'GET /api/v1/vms/:id/cpu-config',
    'PUT /api/v1/vms/:id/cpu-config',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/boot-source', 'workload-boot-source', views.workloadBoot, [
    'GET /api/v1/vms/:id/boot-source',
    'PUT /api/v1/vms/:id/boot-source',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/smt', 'workload-smt', views.workloadSmt, [
    'GET /api/v1/vms/:id/smt',
    'PUT /api/v1/vms/:id/smt',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/constraints', 'workload-constraints', views.workloadConstraints, [
    'GET /api/v1/vms/:id/constraints',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/observability', 'workload-observability', views.workloadObservability, [
    'GET /api/v1/vms/:id/logs',
    'GET /api/v1/vms/:id/metrics',
  ], { workspace: 'observability' }),
  route('/workloads/:id/vsock', 'workload-vsock', views.workloadVsock, [
    'GET /api/v1/vms/:id/vsock',
    'PUT /api/v1/vms/:id/vsock',
    'POST /api/v1/vms/:id/terminal',
  ], { workspace: 'workloads' }),
  route('/workloads/:id/snapshots', 'workload-snapshots', views.workloadSnapshots, [
    'POST /api/v1/vms/:id/snapshots',
    'POST /api/v1/vms/:id/snapshot',
    'POST /api/v1/vms/:id/snapshots/restore',
    'POST /api/v1/vms/:id/restore',
    'DELETE /api/v1/vms/:id/snapshots',
  ], { workspace: 'snapshots' }),
  route('/snapshots', 'snapshots', views.workloadSnapshots, [
    'POST /api/v1/vms/:id/snapshots',
    'POST /api/v1/vms/:id/snapshot',
    'POST /api/v1/vms/:id/snapshots/restore',
    'POST /api/v1/vms/:id/restore',
    'DELETE /api/v1/vms/:id/snapshots',
  ], { workspace: 'snapshots' }),

  route('/kernels', 'kernels', views.imagesKernels, [
    'GET /api/v1/images/kernels',
    'POST /api/v1/images/kernels',
    'DELETE /api/v1/images/kernels/:id',
  ], { workspace: 'settings' }),
  route('/storage', 'storage', views.storage, [
    'GET /api/v1/volumes',
    'POST /api/v1/volumes',
    'DELETE /api/v1/volumes/:id',
  ], { workspace: 'storage' }),
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

export default router
