<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  AuthStatus, BaseImages, CloneImage, Convert, CreateVM, CreateVolume, DeleteImage, DeleteKernel, DeleteVolume, DeleteVM,
  GuestCommand, Health, ImageDetail, ImageStorageStats, ImageTemplates, Images, Kernels, Login, Logout, Logs, Metrics,
  MetricsStream, OperationDetail, Operations, PruneImages, RegisterImage, RegisterKernel, ResolveGitHub, RuntimeStatus,
  SnapshotCreate, SnapshotCreateAlias, SnapshotDelete, SnapshotRestore, SnapshotRestoreAlias, SystemInfo, SystemStats,
  ValidateYAML, VMAction, VMConfig, VMConstraints, VMCPUConfig, VMDetail, VMBootSource, VMMetrics, VMLogs, VMProcess,
  VMSMT, VMSock, VMs, Volumes,
} from '../api'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const notice = ref('')
const result = ref(null)
const live = ref(false)
let refreshTimer
let stream

const form = reactive({
  reference: 'alpine:latest', sourceType: 'oci', baseProfile: 'alpine', architecture: 'native',
  artifactDigest: '', imageReference: 'nginx:latest', vcpus: 1, memoryMiB: 512,
  command: 'uname -a', username: 'admin', password: '', yaml: 'image: nginx:latest\nports:\n  - 8080:80',
  json: '{}', volumeName: 'data', volumeSize: 1073741824, kernelPath: '', cid: 3, socketPath: '', action: 'start',
})

const page = computed(() => route.name || 'overview')
const id = computed(() => String(route.params.id || ''))
const title = computed(() => ({
  overview: 'Overview', security: 'Security & Sessions', metrics: 'Metrics', activity: 'Activity Logs', readiness: 'Host Readiness',
  images: 'Images & Builds', 'image-import': 'Import Image', 'image-catalog': 'Image Catalog', 'image-detail': 'Image Detail',
  operations: 'Operations', 'operation-detail': 'Operation Detail', 'deployment-yaml': 'Deployment YAML', workloads: 'Workloads',
  'workload-detail': 'Workload Detail', 'workload-process': 'Process Inspection', 'workload-config': 'Machine Configuration',
  'workload-cpu': 'CPU Configuration', 'workload-boot-source': 'Boot Source', 'workload-smt': 'SMT Configuration',
  'workload-constraints': 'Constraints', 'workload-observability': 'Workload Observability', 'workload-vsock': 'Vsock & Terminal',
  'workload-snapshots': 'Workload Snapshots', snapshots: 'Snapshots', kernels: 'Kernel Catalog', storage: 'Storage',
})[page.value] || 'Firecracker Studio')
const isDetail = computed(() => Boolean(id.value))
const items = computed(() => {
  const value = result.value
  if (Array.isArray(value)) return value
  if (!value || typeof value !== 'object') return []
  return value.vms || value.images || value.operations || value.volumes || value.kernels || value.items || value.events || []
})
const hasData = computed(() => Boolean(result.value && (items.value.length || Object.keys(result.value).length)))

function clearStatus() { error.value = ''; notice.value = '' }
function fail(err) { error.value = String(err).replace(/^Error: /, ''); notice.value = '' }
function parseJSON() { try { return JSON.parse(form.json || '{}') } catch { throw new Error('Request JSON is invalid') } }
async function run(fn, success = '') {
  loading.value = true; clearStatus()
  try { result.value = await fn(); if (success) notice.value = success } catch (err) { fail(err) } finally { loading.value = false }
}
async function mutate(fn, success) {
  saving.value = true; clearStatus()
  try { result.value = await fn(); notice.value = success; await load() } catch (err) { fail(err) } finally { saving.value = false }
}
function resourceName(item) { return item?.name || item?.reference || item?.imageReference || item?.distribution || item?.kind || item?.id || 'Unnamed resource' }
function resourceId(item) { return item?.id || item?.digest || item?.operationId || '—' }
function resourceStatus(item) { return item?.status || item?.state || (item?.ready ? 'ready' : 'available') }
function open(path) { router.push(path) }

async function load() {
  const p = page.value
  if (p === 'overview') return run(async () => ({ health: await Health(), runtime: await RuntimeStatus(), system: await SystemInfo(), stats: await SystemStats(), metrics: await Metrics() }))
  if (p === 'security') return run(AuthStatus)
  if (p === 'metrics') return run(() => id.value ? VMMetrics(id.value) : Metrics())
  if (p === 'activity') return run(() => id.value ? VMLogs(id.value) : Logs())
  if (p === 'readiness') return run(RuntimeStatus)
  if (p === 'images') return run(async () => ({ bases: await BaseImages(), images: await Images(), operations: await Operations() }))
  if (p === 'image-import') return run(BaseImages)
  if (p === 'image-catalog') return run(async () => ({ images: await Images(), storage: await ImageStorageStats(), templates: await ImageTemplates() }))
  if (p === 'image-detail') return run(() => ImageDetail(id.value))
  if (p === 'operations') return run(Operations)
  if (p === 'operation-detail') return run(() => OperationDetail(id.value))
  if (p === 'workloads') return run(VMs)
  if (p === 'workload-detail') return run(() => VMDetail(id.value))
  if (p === 'workload-process') return run(() => VMProcess(id.value))
  if (p === 'workload-config') return run(() => VMConfig(id.value))
  if (p === 'workload-cpu') return run(() => VMCPUConfig(id.value))
  if (p === 'workload-boot-source') return run(() => VMBootSource(id.value))
  if (p === 'workload-smt') return run(() => VMSMT(id.value))
  if (p === 'workload-constraints') return run(() => VMConstraints(id.value))
  if (p === 'workload-observability') return run(async () => ({ logs: await VMLogs(id.value), metrics: await VMMetrics(id.value) }))
  if (p === 'workload-vsock') return run(() => VMSock(id.value))
  if (p === 'workload-snapshots') return run(() => VMDetail(id.value))
  if (p === 'snapshots') return run(VMs)
  if (p === 'kernels') return run(Kernels)
  if (p === 'storage') return run(Volumes)
}
async function submit() {
  const p = page.value
  if (p === 'security') return mutate(() => Login(form.username, form.password), 'Signed in successfully')
  if (p === 'image-import') return mutate(async () => { if (form.sourceType === 'github' || form.sourceType === 'github-yaml') await ResolveGitHub(form.reference); return Convert(form.reference, form.sourceType, form.baseProfile, form.architecture) }, 'Image conversion queued')
  if (p === 'deployment-yaml') return mutate(() => ValidateYAML(form.yaml), 'Deployment YAML validated')
  if (p === 'workloads') return mutate(() => CreateVM(form.artifactDigest, Number(form.vcpus), Number(form.memoryMiB), form.imageReference || form.reference, [], 'ephemeral', ''), 'MicroVM creation requested')
  if (p === 'workload-detail') return mutate(() => VMAction(id.value, form.action), `Workload ${form.action} requested`)
  if (p === 'workload-config') return mutate(() => VMConfig(id.value, 'PUT', parseJSON()), 'Machine configuration saved')
  if (p === 'workload-cpu') return mutate(() => VMCPUConfig(id.value, 'PUT', parseJSON()), 'CPU configuration saved')
  if (p === 'workload-boot-source') return mutate(() => VMBootSource(id.value, 'PUT', parseJSON()), 'Boot source saved')
  if (p === 'workload-smt') return mutate(() => VMSMT(id.value, 'PUT', parseJSON()), 'SMT configuration saved')
  if (p === 'workload-vsock') return mutate(() => VMSock(id.value, 'PUT', { cid: Number(form.cid), socketPath: form.socketPath }), 'Vsock configuration saved')
  if (p === 'workload-snapshots') return mutate(() => SnapshotCreate(id.value, parseJSON()), 'Snapshot creation requested')
  if (p === 'kernels') return mutate(() => RegisterKernel({ path: form.kernelPath, architecture: form.architecture, version: 'custom' }), 'Kernel registered')
  if (p === 'storage') return mutate(() => CreateVolume({ name: form.volumeName, sizeBytes: Number(form.volumeSize), filesystem: 'ext4' }), 'Volume created')
  if (p === 'image-catalog') return mutate(() => RegisterImage(parseJSON()), 'Image registered')
}
async function guestCommand() { return mutate(() => GuestCommand(id.value, form.command), 'Guest command completed') }
async function lifecycle(action) { form.action = action; return mutate(() => VMAction(id.value, action), `Workload ${action} requested`) }
async function remove(item) {
  const target = item?.id || item?.digest || id.value
  if (page.value === 'kernels') return mutate(() => DeleteKernel(target), 'Kernel deleted')
  if (page.value === 'storage') return mutate(() => DeleteVolume(target), 'Volume deleted')
  if (page.value === 'image-catalog') return mutate(() => DeleteImage(item?.digest || target), 'Image deleted')
  if (page.value === 'workload-detail') return mutate(() => DeleteVM(target), 'Workload deleted')
  if (page.value === 'workload-snapshots') return mutate(() => SnapshotDelete(target), 'Snapshots deleted')
}
function startLive() {
  if (stream || page.value !== 'metrics') return
  stream = MetricsStream(value => { result.value = value; live.value = true }, () => { live.value = false })
}
function stopLive() { stream?.close?.(); stream = null; live.value = false }
watch(() => route.fullPath, async () => { stopLive(); await load(); if (page.value === 'metrics') startLive() })
onMounted(async () => { await load(); if (page.value === 'metrics') startLive(); refreshTimer = window.setInterval(() => { if (!document.hidden && !saving.value) load() }, 15000) })
onBeforeUnmount(() => { stopLive(); window.clearInterval(refreshTimer) })
</script>

<template>
  <main class="page-content operational-view">
    <header class="page-heading"><div><span class="section-kicker">FIRECRACKER STUDIO</span><h2>{{ title }}</h2><p class="page-subtitle">{{ isDetail ? `Managing ${id}` : 'Live resource management and operational controls' }}</p></div><div class="heading-actions"><span v-if="live" class="live-pill"><span class="dot success" /> LIVE</span><button class="small-button" :disabled="loading" @click="load">{{ loading ? 'Loading…' : 'Refresh' }}</button></div></header>
    <div v-if="error" class="error-banner"><strong>Operation failed</strong><span>{{ error }}</span><button @click="error = ''">Dismiss</button></div>
    <div v-if="notice" class="success-banner"><strong>Success</strong><span>{{ notice }}</span><button @click="notice = ''">Dismiss</button></div>

    <section v-if="page === 'overview' && result" class="metric-grid operational-metrics"><article class="metric-card"><span>CONTROL PLANE</span><strong>{{ result.health?.status || result.health?.ready ? 'Healthy' : 'Unavailable' }}</strong><small>API health</small></article><article class="metric-card"><span>FIRECRACKER</span><strong>{{ result.runtime?.firecracker || 'Unknown' }}</strong><small>Runtime readiness</small></article><article class="metric-card"><span>WORKLOADS</span><strong>{{ result.stats?.running ?? result.stats?.vms ?? '—' }}</strong><small>Running microVMs</small></article><article class="metric-card"><span>LIVE METRICS</span><strong>{{ result.metrics ? 'Connected' : 'Waiting' }}</strong><small>Observability feed</small></article></section>
    <section v-if="page === 'overview'" class="resource-grid"><article class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">QUICK ACTIONS</span><h3>Operate your platform</h3></div></div><div class="action-grid"><button class="primary-button" @click="open('/images/import')">Import image</button><button class="small-button" @click="open('/workloads')">Create MicroVM</button><button class="small-button" @click="open('/storage')">Manage storage</button><button class="small-button" @click="open('/host-readiness')">Check readiness</button></div></article><article class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">SYSTEM</span><h3>Host and runtime</h3></div><span class="status-badge" :class="result?.runtime?.ready ? 'success' : 'warning'">{{ result?.runtime?.ready ? 'READY' : 'CHECK' }}</span></div><dl class="detail-list"><div><dt>Platform</dt><dd>{{ result?.system?.platform || result?.runtime?.platform || '—' }}</dd></div><div><dt>KVM</dt><dd>{{ result?.runtime?.kvm || '—' }}</dd></div><div><dt>TAP</dt><dd>{{ result?.runtime?.tap || '—' }}</dd></div><div><dt>Message</dt><dd>{{ result?.runtime?.message || 'No message' }}</dd></div></dl></article></section>

    <section v-if="['workloads','images','operations','kernels','storage','snapshots'].includes(page)" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">RESOURCE INVENTORY</span><h3>{{ title }}</h3></div><button v-if="page === 'images'" class="primary-button compact" @click="open('/images/import')">＋ Import image</button><button v-if="page === 'snapshots'" class="small-button" @click="open('/workloads')">Select workload</button><button v-if="page === 'images'" class="small-button" @click="open('/images/catalog')">Catalog</button></div><div v-if="loading" class="loading-state">Loading live resources…</div><div v-else-if="!items.length" class="empty-state"><strong>No {{ page }} found</strong><span>When the control API is connected, live resources will appear here.</span></div><div v-else class="operational-table"><div class="table-head"><span>NAME</span><span>STATUS</span><span>IDENTIFIER</span><span>ACTIONS</span></div><div v-for="item in items" :key="resourceId(item)" class="resource-row"><span><strong>{{ resourceName(item) }}</strong><small>{{ item.reference || item.architecture || item.kind || '' }}</small></span><span><span class="status-badge" :class="resourceStatus(item) === 'running' || resourceStatus(item) === 'ready' ? 'success' : 'neutral'">{{ resourceStatus(item) }}</span></span><code>{{ resourceId(item) }}</code><span class="row-actions"><button class="small-button" @click="open(page === 'workloads' ? `/workloads/${item.id}` : page === 'operations' ? `/operations/${item.id}` : page === 'images' ? `/images/${item.id}` : '#')">Inspect</button><button v-if="['workloads','kernels','storage'].includes(page)" class="small-button danger" @click="remove(item)">Delete</button></span></div></div></section>

    <section v-if="page === 'workloads'" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">CREATE WORKLOAD</span><h3>Launch a Firecracker microVM</h3></div></div><form class="form-grid" @submit.prevent="submit"><label>IMAGE REFERENCE<input v-model="form.imageReference" placeholder="nginx:latest" /></label><label>ARTIFACT DIGEST<input v-model="form.artifactDigest" placeholder="sha256:..." /></label><label>VCPUS<input v-model.number="form.vcpus" type="number" min="1" /></label><label>MEMORY MIB<input v-model.number="form.memoryMiB" type="number" min="128" /></label><button class="primary-button" :disabled="saving">{{ saving ? 'Creating…' : 'Create MicroVM' }}</button></form></section>

    <section v-if="page === 'images' && result" class="resource-grid"><article class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">BASE PROFILES</span><h3>Available guest bases</h3></div></div><div class="mini-list"><div v-for="base in result.bases?.images || result.bases || []" :key="base.id"><strong>{{ base.distribution }} {{ base.version }}</strong><span>{{ base.architecture }} · {{ base.status }}</span></div></div></article><article class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">CONVERSION QUEUE</span><h3>Recent operations</h3></div></div><div class="mini-list"><div v-for="op in result.operations?.operations || result.operations || []" :key="op.id"><strong>{{ op.kind || op.id }}</strong><span>{{ op.state || op.status }}</span></div></div></article></section>

    <section v-if="page === 'image-import'" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">IMAGE SOURCE</span><h3>Import a guest image</h3><p>Resolve a Docker Hub image, GitHub repository, or YAML manifest into a Firecracker artifact.</p></div></div><form class="form-grid" @submit.prevent="submit"><label>REFERENCE<input v-model="form.reference" required placeholder="alpine:latest or github.com/org/repo" /></label><label>SOURCE TYPE<select v-model="form.sourceType"><option value="oci">Docker Hub / OCI</option><option value="docker">Docker image</option><option value="github">GitHub repository</option><option value="github-yaml">GitHub YAML</option><option value="archive">Firecracker archive</option></select></label><label>BASE PROFILE<select v-model="form.baseProfile"><option>alpine</option><option>debian</option><option>ubuntu</option></select></label><label>ARCHITECTURE<select v-model="form.architecture"><option>native</option><option>x86_64</option><option>aarch64</option></select></label><button class="primary-button" :disabled="saving">{{ saving ? 'Queueing…' : 'Import and convert' }}</button></form></section>

    <section v-if="page === 'security'" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">AUTHENTICATION</span><h3>Session management</h3><p>Use the configured admin account to control the remote Studio session.</p></div></div><form class="form-grid narrow" @submit.prevent="submit"><label>USERNAME<input v-model="form.username" autocomplete="username" /></label><label>PASSWORD<input v-model="form.password" type="password" autocomplete="current-password" /></label><button class="primary-button" :disabled="saving">{{ saving ? 'Signing in…' : 'Sign in' }}</button></form><div class="technical-detail" v-if="result"><strong>Session status</strong><pre>{{ JSON.stringify(result, null, 2) }}</pre></div></section>

    <section v-if="['workload-detail','workload-process','workload-config','workload-cpu','workload-boot-source','workload-smt','workload-constraints','workload-observability','workload-vsock','workload-snapshots','image-detail','operation-detail'].includes(page)" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">RESOURCE DETAIL</span><h3>{{ title }}</h3><p>{{ id }}</p></div><button class="small-button" @click="load">Refresh detail</button></div><div v-if="page === 'workload-detail'" class="action-grid"><button v-for="action in ['start','stop','pause','resume']" :key="action" class="small-button" :disabled="saving" @click="lifecycle(action)">{{ action }}</button><button class="small-button" @click="open(`/workloads/${id}/process`)">Process</button><button class="small-button" @click="open(`/workloads/${id}/config`)">Config</button><button class="small-button" @click="open(`/workloads/${id}/observability`)">Observability</button><button class="small-button" @click="open(`/workloads/${id}/vsock`)">Vsock / terminal</button><button class="small-button" @click="open(`/workloads/${id}/snapshots`)">Snapshots</button><button class="small-button danger" @click="remove({ id })">Delete</button></div><div v-if="page === 'workload-vsock'" class="form-grid"><label>VSOCK CID<input v-model.number="form.cid" type="number" /></label><label>SOCKET PATH<input v-model="form.socketPath" /></label><label class="wide">GUEST COMMAND<input v-model="form.command" @keyup.enter="guestCommand" /></label><button class="primary-button" @click="submit">Save vsock</button><button class="small-button" @click="guestCommand">Run command</button></div><div v-if="['workload-config','workload-cpu','workload-boot-source','workload-smt'].includes(page)" class="form-grid"><label class="wide">CONFIGURATION JSON<textarea v-model="form.json" rows="12" spellcheck="false" /></label><button class="primary-button" :disabled="saving" @click="submit">Save configuration</button></div><div v-if="page === 'workload-snapshots'" class="action-grid"><button class="primary-button" @click="submit">Create snapshot</button><button class="small-button" @click="mutate(() => SnapshotCreateAlias(id.value, {}), 'Snapshot alias created')">Create alias</button><button class="small-button" @click="mutate(() => SnapshotRestore(id.value, {}), 'Snapshot restored')">Restore snapshot</button><button class="small-button" @click="mutate(() => SnapshotRestoreAlias(id.value, {}), 'Snapshot alias restored')">Restore alias</button><button class="small-button danger" @click="remove({ id })">Delete snapshots</button></div><div v-if="page === 'image-detail'" class="action-grid"><button class="primary-button" @click="mutate(() => CloneImage(id.value), 'Image clone requested')">Clone image</button><button class="small-button danger" @click="remove({ id })">Delete image</button></div><div class="technical-detail"><strong>Live response</strong><pre>{{ result ? JSON.stringify(result, null, 2) : 'No detail loaded.' }}</pre></div></section>

    <section v-if="page === 'deployment-yaml'" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">DECLARATIVE DEPLOYMENT</span><h3>Validate deployment YAML</h3></div></div><form @submit.prevent="submit"><textarea class="code-editor" v-model="form.yaml" rows="16" spellcheck="false" /><button class="primary-button" :disabled="saving">{{ saving ? 'Validating…' : 'Validate YAML' }}</button></form></section>
    <section v-if="page === 'metrics' || page === 'activity' || page === 'readiness'" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">LIVE OPERATIONS</span><h3>{{ page === 'metrics' && live ? 'Streaming metrics' : 'Current response' }}</h3></div><span v-if="page === 'metrics'" class="status-badge" :class="live ? 'success' : 'neutral'">{{ live ? 'LIVE' : 'POLLING' }}</span></div><pre class="technical-output">{{ result ? JSON.stringify(result, null, 2) : 'No response yet.' }}</pre></section>
    <section v-if="page === 'image-catalog'" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">IMAGE ADMINISTRATION</span><h3>Catalog maintenance</h3></div><button class="small-button danger" @click="mutate(PruneImages, 'Unused images pruned')">Prune unused images</button></div><form @submit.prevent="submit"><label class="wide">IMAGE JSON<textarea v-model="form.json" rows="8" spellcheck="false" /></label><button class="primary-button">Register image</button></form></section>
    <section v-if="page === 'kernels' || page === 'storage'" class="panel operational-card"><div class="panel-header"><div><span class="section-kicker">RESOURCE ADMINISTRATION</span><h3>{{ page === 'kernels' ? 'Register kernel' : 'Create volume' }}</h3></div></div><form class="form-grid" @submit.prevent="submit"><label v-if="page === 'kernels'">KERNEL PATH<input v-model="form.kernelPath" placeholder="/var/lib/firecracker/vmlinux" /></label><label v-if="page === 'storage'">VOLUME NAME<input v-model="form.volumeName" /></label><label v-if="page === 'storage'">SIZE BYTES<input v-model.number="form.volumeSize" type="number" /></label><button class="primary-button" :disabled="saving">{{ page === 'kernels' ? 'Register kernel' : 'Create volume' }}</button></form></section>
  </main>
</template>
