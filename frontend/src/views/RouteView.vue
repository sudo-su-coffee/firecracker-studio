<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  AuthStatus, BaseImages, CloneImage, Convert, CreateVM, CreateVolume, DeleteImage, DeleteKernel, DeleteVolume, DeleteVM,
  GuestCommand, Health, ImageDetail, ImageStorageStats, ImageTemplates, Images, Kernels, Login, Logout, Logs,
  Metrics, OperationDetail, Operations, PruneImages, RegisterImage, RegisterKernel, ResolveGitHub, RuntimeStatus, SnapshotCreate, SnapshotCreateAlias,
  SnapshotDelete, SnapshotRestore, SnapshotRestoreAlias, SystemInfo, SystemStats, ValidateYAML, VMAction, VMConfig, VMConstraints, VMCPUConfig,
  VMDetail, VMBootSource, VMMetrics, VMLogs, VMProcess, VMSMT, VMSock, VMs, Volumes,
} from '../api'

const route = useRoute()
const router = useRouter()
const busy = ref(false)
const error = ref('')
const message = ref('')
const result = ref(null)
const form = reactive({ reference: 'alpine:latest', sourceType: 'oci', baseProfile: 'alpine', architecture: 'native', command: 'uname -a', username: 'admin', password: '', yaml: 'image: nginx:latest\nports:\n  - 8080:80', json: '{}', artifactDigest: '', imageReference: '', vcpus: 1, memoryMiB: 512, volumeName: 'data', volumeSize: 1073741824, kernelPath: '', cid: 3, socketPath: '', action: 'start' })
const page = computed(() => route.name || 'overview')
const id = computed(() => route.params.id)
const title = computed(() => ({ overview: 'Overview', security: 'Security & Sessions', metrics: 'Metrics', activity: 'Activity Logs', readiness: 'Host Readiness', images: 'Images & Builds', 'image-import': 'Import Image', 'image-catalog': 'Image Catalog', 'image-detail': 'Image Detail', operations: 'Operations', 'operation-detail': 'Operation Detail', 'deployment-yaml': 'Deployment YAML', workloads: 'Workloads', 'workload-detail': 'Workload Detail', 'workload-process': 'Process Inspection', 'workload-config': 'Machine Configuration', 'workload-cpu': 'CPU Configuration', 'workload-boot-source': 'Boot Source', 'workload-smt': 'SMT Configuration', 'workload-constraints': 'Constraints', 'workload-observability': 'Workload Observability', 'workload-vsock': 'Vsock & Terminal', 'workload-snapshots': 'Workload Snapshots', snapshots: 'Snapshots', kernels: 'Kernel Catalog', storage: 'Storage' })[page.value] || 'Firecracker Studio')
const apiRoutes = computed(() => route.meta.api || [])
const rows = computed(() => Array.isArray(result.value) ? result.value : result.value?.items || result.value?.vms || result.value?.images || result.value?.operations || result.value?.volumes || result.value?.kernels || result.value?.events || [])

function setResult(value, text = '') { result.value = value; message.value = text; error.value = '' }
function fail(err) { error.value = String(err).replace(/^Error: /, ''); message.value = '' }
function parseJSON() { try { return JSON.parse(form.json || '{}') } catch { throw new Error('Request JSON is invalid') } }
async function run(label, fn) { busy.value = true; error.value = ''; message.value = ''; try { setResult(await fn(), label) } catch (err) { fail(err) } finally { busy.value = false } }
async function load() {
  const p = page.value
  if (p === 'overview') return run('Overview refreshed', async () => ({ health: await Health(), metrics: await Metrics(), system: await SystemInfo(), stats: await SystemStats(), runtime: await RuntimeStatus() }))
  if (p === 'security') return run('Session status loaded', AuthStatus)
  if (p === 'metrics') return run('Metrics loaded', async () => id.value ? VMMetrics(id.value) : Metrics())
  if (p === 'activity') return run('Logs loaded', async () => id.value ? VMLogs(id.value) : Logs())
  if (p === 'readiness') return run('Readiness checks completed', RuntimeStatus)
  if (p === 'images') return run('Image workspace loaded', async () => ({ bases: await BaseImages(), images: await Images(), operations: await Operations() }))
  if (p === 'image-import') return setResult(await BaseImages())
  if (p === 'image-catalog') return run('Image catalog loaded', async () => ({ images: await Images(), storage: await ImageStorageStats(), templates: await ImageTemplates() }))
  if (p === 'image-detail') return run('Image detail loaded', () => ImageDetail(id.value))
  if (p === 'operations') return run('Operations loaded', Operations)
  if (p === 'operation-detail') return run('Operation detail loaded', () => OperationDetail(id.value))
  if (p === 'workloads') return run('Workloads loaded', () => VMs())
  if (p === 'workload-detail') return run('Workload detail loaded', () => VMDetail(id.value))
  if (p === 'workload-process') return run('Process state loaded', () => VMProcess(id.value))
  if (p === 'workload-config') return run('Machine configuration loaded', () => VMConfig(id.value))
  if (p === 'workload-cpu') return run('CPU configuration loaded', () => VMCPUConfig(id.value))
  if (p === 'workload-boot-source') return run('Boot source loaded', () => VMBootSource(id.value))
  if (p === 'workload-smt') return run('SMT configuration loaded', () => VMSMT(id.value))
  if (p === 'workload-constraints') return run('Constraints loaded', () => VMConstraints(id.value))
  if (p === 'workload-observability') return run('Workload logs and metrics loaded', async () => ({ logs: await VMLogs(id.value), metrics: await VMMetrics(id.value) }))
  if (p === 'workload-vsock') return run('Vsock configuration loaded', () => VMSock(id.value))
  if (p === 'workloads' || p === 'snapshots' || p === 'workload-snapshots') return run('Snapshot workloads loaded', () => VMs())
  if (p === 'kernels') return run('Kernel catalog loaded', () => Kernels())
  if (p === 'storage') return run('Volumes loaded', () => Volumes())
}
async function submit() {
  const p = page.value
  try {
    if (p === 'security') return run('Signed in', () => Login(form.username, form.password))
    if (p === 'image-import') return run('Image import queued', async () => { if (form.sourceType === 'github' || form.sourceType === 'github-yaml') await ResolveGitHub(form.reference); return Convert(form.reference, form.sourceType, form.baseProfile, form.architecture) })
    if (p === 'deployment-yaml') return run('YAML validated', () => ValidateYAML(form.yaml))
    if (p === 'workloads') return run('Workload created', () => CreateVM(form.artifactDigest, Number(form.vcpus), Number(form.memoryMiB), form.imageReference || form.reference, [], 'ephemeral', ''))
    if (p === 'workload-detail') return run(`Workload ${form.action} requested`, () => VMAction(id.value, form.action))
    if (p === 'workload-config') return run('Machine configuration updated', () => VMConfig(id.value, 'PUT', parseJSON()))
    if (p === 'workload-cpu') return run('CPU configuration updated', () => VMCPUConfig(id.value, 'PUT', parseJSON()))
    if (p === 'workload-boot-source') return run('Boot source updated', () => VMBootSource(id.value, 'PUT', parseJSON()))
    if (p === 'workload-smt') return run('SMT configuration updated', () => VMSMT(id.value, 'PUT', parseJSON()))
    if (p === 'workload-vsock') return run('Vsock configuration updated', () => VMSock(id.value, 'PUT', { cid: Number(form.cid), socketPath: form.socketPath }))
    if (p === 'workload-snapshots' || p === 'snapshots') return run('Snapshot created', () => SnapshotCreate(id.value, parseJSON()))
    if (p === 'kernels') return run('Kernel registered', () => RegisterKernel({ path: form.kernelPath, architecture: form.architecture, version: 'custom' }))
    if (p === 'storage') return run('Volume created', () => CreateVolume({ name: form.volumeName, sizeBytes: Number(form.volumeSize), filesystem: 'ext4' }))
    if (p === 'image-catalog') return run('Image registered', () => RegisterImage(parseJSON()))
    if (p === 'image-catalog-prune') return run('Unused images pruned', () => PruneImages())
    if (p === 'image-detail') return run('Image cloned', () => CloneImage(id.value))
  } catch (err) { fail(err) }
}
async function command() { return run('Guest command completed', () => GuestCommand(id.value, form.command)) }
async function removeItem(item) {
  const itemId = item?.id || item?.digest
  if (!itemId) return
  if (page.value === 'kernels') return run('Kernel deleted', () => DeleteKernel(itemId))
  if (page.value === 'storage') return run('Volume deleted', () => DeleteVolume(itemId))
  if (page.value === 'image-catalog') return run('Image deleted', () => DeleteImage(item.digest || itemId))
  if (page.value === 'workload-detail') return run('Workload deleted', () => DeleteVM(id.value))
  if (page.value === 'workload-snapshots' || page.value === 'snapshots') return run('Snapshots deleted', () => SnapshotDelete(id.value))
}
function open(path) { router.push(path) }
watch(() => route.fullPath, load)
onMounted(load)
</script>

<template>
  <main class="page-content route-view">
    <header class="page-heading"><div><span class="section-kicker">FIRECRACKER STUDIO · {{ page.toUpperCase() }}</span><h2>{{ title }}</h2><p>API-backed control surface for {{ apiRoutes.length }} mapped backend {{ apiRoutes.length === 1 ? 'capability' : 'capabilities' }}.</p></div><button class="primary-button compact" :disabled="busy" @click="load">↻ Refresh</button></header>
    <div v-if="error" class="warning-banner panel"><strong>Request failed</strong><span>{{ error }}</span></div>
    <div v-if="message" class="info-banner panel"><strong>{{ message }}</strong></div>
    <section class="panel route-api-map"><div class="section-kicker">MAPPED API ROUTES</div><div class="api-route-list"><code v-for="item in apiRoutes" :key="item">{{ item }}</code></div></section>

    <section v-if="['image-import','deployment-yaml','security','workloads','workload-config','workload-cpu','workload-boot-source','workload-smt','workload-vsock','kernels','storage','image-catalog'].includes(page)" class="panel route-form">
      <form @submit.prevent="submit">
        <label v-if="page === 'security'">USERNAME<input v-model="form.username" autocomplete="username" /></label><label v-if="page === 'security'">PASSWORD<input v-model="form.password" type="password" autocomplete="current-password" /></label>
        <label v-if="page === 'image-import'">REFERENCE<input v-model="form.reference" /></label><label v-if="page === 'image-import'">SOURCE TYPE<select v-model="form.sourceType"><option value="oci">Docker Hub image</option><option value="docker">Docker image</option><option value="github">GitHub repository</option><option value="github-yaml">GitHub YAML</option><option value="archive">Firecracker image archive</option></select></label>
        <label v-if="page === 'image-import'">BASE PROFILE<select v-model="form.baseProfile"><option>alpine</option><option>debian</option><option>ubuntu</option></select></label><label v-if="page === 'image-import'">ARCHITECTURE<select v-model="form.architecture"><option>native</option><option>x86_64</option><option>aarch64</option></select></label>
        <label v-if="page === 'deployment-yaml'">DEPLOYMENT YAML<textarea v-model="form.yaml" rows="8" /></label>
        <label v-if="page === 'workloads'">ARTIFACT DIGEST<input v-model="form.artifactDigest" placeholder="sha256:..." /></label><label v-if="page === 'workloads'">IMAGE REFERENCE<input v-model="form.imageReference" placeholder="nginx:latest" /></label><label v-if="page === 'workloads'">VCPUS<input v-model.number="form.vcpus" type="number" min="1" /></label><label v-if="page === 'workloads'">MEMORY MIB<input v-model.number="form.memoryMiB" type="number" min="128" /></label>
        <label v-if="['workload-config','workload-cpu','workload-boot-source','workload-smt','image-catalog'].includes(page)">REQUEST JSON<textarea v-model="form.json" rows="8" /></label>
        <label v-if="page === 'workload-vsock'">VSOCK CID<input v-model.number="form.cid" type="number" /><span>Socket path</span><input v-model="form.socketPath" /></label>
        <label v-if="page === 'kernels'">KERNEL PATH<input v-model="form.kernelPath" placeholder="/var/lib/firecracker/vmlinux" /></label>
        <label v-if="page === 'storage'">VOLUME NAME<input v-model="form.volumeName" /><span>Size bytes</span><input v-model.number="form.volumeSize" type="number" /></label>
        <button class="primary-button compact" type="submit" :disabled="busy">{{ busy ? 'Working…' : 'Run API action' }}</button>
      </form>
    </section>

    <section v-if="page === 'workload-detail'" class="panel route-form"><button class="small-button" @click="run('Image clone requested', () => CloneImage(id.value))">Clone image</button><label>LIFECYCLE ACTION<select v-model="form.action"><option>start</option><option>stop</option><option>pause</option><option>resume</option></select></label><button class="primary-button compact" @click="submit">Run lifecycle action</button><button class="small-button" @click="open(`/workloads/${id}/process`)">Process</button><button class="small-button" @click="open(`/workloads/${id}/config`)">Config</button><button class="small-button" @click="open(`/workloads/${id}/observability`)">Observability</button><button class="small-button" @click="open(`/workloads/${id}/vsock`)">Vsock</button><button class="small-button" @click="open(`/workloads/${id}/snapshots`)">Snapshots</button><button class="small-button" @click="removeItem({ id })">Delete workload</button></section>
    <section v-if="page === 'workload-vsock'" class="panel route-form"><label>GUEST COMMAND<input v-model="form.command" @keyup.enter="command" /></label><button class="primary-button compact" @click="command">Run guest command</button></section>
    <section v-if="page === 'workload-snapshots' || page === 'snapshots'" class="panel route-form"><button class="primary-button compact" @click="submit">Create snapshot</button><button class="small-button" @click="run('Snapshot alias created', () => SnapshotCreateAlias(id, {}))">Create alias</button><button class="small-button" @click="run('Snapshot restored', () => SnapshotRestore(id, {}))">Restore snapshot</button><button class="small-button" @click="run('Snapshot alias restored', () => SnapshotRestoreAlias(id, {}))">Restore alias</button><button class="small-button" @click="removeItem({ id })">Delete snapshots</button></section>

    <section v-if="page === 'image-catalog'" class="panel route-form"><button class="small-button" @click="run('Unused images pruned', () => PruneImages())">Prune unused images</button></section>
    <section class="panel route-result"><div class="section-kicker">API RESPONSE</div><pre v-if="result">{{ JSON.stringify(result, null, 2) }}</pre><div v-else class="empty-state">No response loaded yet. Use Refresh or run an API action.</div></section>
    <section v-if="rows.length" class="panel resource-table"><div class="table-head"><span>RESOURCE</span><span>STATUS</span><span>IDENTIFIER</span><span>ACTION</span></div><div v-for="item in rows" :key="item.id || item.digest || JSON.stringify(item)" class="resource-row"><span>{{ item.name || item.reference || item.kind || item.distribution || 'Resource' }}</span><span>{{ item.status || item.state || 'available' }}</span><code>{{ item.id || item.digest || item.updatedAt || '—' }}</code><button v-if="['kernels','storage','image-catalog'].includes(page)" class="small-button" @click="removeItem(item)">Delete</button><button v-else class="small-button" @click="item.id && open(`/workloads/${item.id}`)">Inspect</button></div></section>
  </main>
</template>
