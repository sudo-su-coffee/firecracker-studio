<script setup>
import { onMounted, onBeforeUnmount, reactive, ref, computed } from 'vue'
import { Servers, AddServer, CheckServer, RemoveServer, SwitchServer, SetConnection, Health, MetricsStream, BaseImages, MarketplaceImages, PullMarketplaceImage, Images, Operations, VMs, Convert, CreateVM, VMAction, Snapshot, RuntimeStatus, Logs } from './api'

const tabs = [
  { id: 'overview', label: 'Dashboard', icon: '⌂' },
  { id: 'images', label: 'Images', icon: '◈' },
  { id: 'vms', label: 'Workloads', icon: '▣' },
  { id: 'convert', label: 'Convert', icon: '⇄' },
  { id: 'servers', label: 'Servers', icon: '◇' },
]
const state = reactive({ tab: 'overview', connectionURL: '', source: 'alpine:latest', sourceType: 'oci', baseProfile: 'alpine', imageName: '', artifactDigest: '', vcpus: 1, memoryMiB: 512, hostPort: 15432, guestPort: 5432, protocol: 'tcp', selectedVM: null, serverID: '', showAddWorker: false, newServer: { name: 'Remote worker', url: '', kind: 'remote', username: '', token: '' } })
const data = reactive({ health: null, baseImages: [], marketplaceImages: [], images: [], operations: [], vms: [], servers: [], logs: [], message: '', runtime: {}, metrics: { workers: 0, microvms: 0, runningVms: 0, images: 0, operations: 0, timestamp: null, host: {} } })
const busy = ref(false)
const selectedVM = computed(() => data.vms.find(vm => vm.id === state.selectedVM) || data.vms[0])
const latestOperation = computed(() => data.operations.slice().sort((a, b) => new Date(b.updatedAt || b.createdAt) - new Date(a.updatedAt || a.createdAt))[0])
const activeServer = computed(() => data.servers.find(server => server.id === state.serverID) || data.servers[0])
const browserInfo = { protocol: window.location.protocol.replace(':', ''), host: window.location.host }
function formatBytes(value = 0) { if (!value) return '0 B'; const units = ['B', 'KB', 'MB', 'GB', 'TB']; const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1); return `${(value / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}` }
let metricsSource
let refreshTimer
function connectMetrics() {
  metricsSource?.close()
  metricsSource = MetricsStream(metrics => { data.metrics = metrics }, () => {})
}

async function refresh() {
  data.runtime = await RuntimeStatus()
  if (!state.connectionURL) return
  try {
    data.health = await Health()
    data.servers = await Servers() || []
    data.baseImages = (await BaseImages()).images || []
    data.marketplaceImages = (await MarketplaceImages()).images || []
    data.images = (await Images()).images || []
    data.operations = (await Operations()).operations || []
    data.logs = (await Logs()).events || []
    data.vms = (await VMs()).vms || []
    state.serverID ||= data.servers.find(server => server.active)?.id || data.servers[0]?.id || ''
    data.message = 'Connected to this server'
    if (!state.selectedVM && data.vms[0]) state.selectedVM = data.vms[0].id
  } catch (error) {
    data.health = null
    data.message = String(error).replace(/^Error: /, '')
  }
}
async function switchServer() {
  if (!state.serverID) return
  busy.value = true
  try {
    const server = await SwitchServer(state.serverID)
    state.connectionURL = server.url
    await SetConnection(server.url, server.token || '')
    await refresh()
  } catch (error) { data.message = String(error) } finally { busy.value = false }
}
async function addServer() {
  busy.value = true
  try {
    const server = await AddServer({ ...state.newServer })
    data.servers = await Servers() || []
    state.serverID = server.id
    state.connectionURL = server.url
    await SetConnection(server.url, server.token || '')
    await SwitchServer(server.id)
    state.showAddWorker = false
    data.message = `Connected to ${server.name}`
    state.newServer = { name: 'Remote worker', url: '', kind: 'remote', username: '', token: '' }
    await refresh()
  } catch (error) { data.message = `Connection failed: ${String(error).replace(/^Error: /, '')}` } finally { busy.value = false }
}
async function checkServer(id) {
  busy.value = true
  try {
    const server = await CheckServer(id)
    data.servers = await Servers() || []
    data.message = `${server.name} is ${server.health}`
    if (server.health === 'healthy') { state.serverID = server.id; await switchServer() }
  } catch (error) { data.message = `Health check failed: ${String(error).replace(/^Error: /, '')}` } finally { busy.value = false }
}
async function removeServer(id) {
  busy.value = true
  try { await RemoveServer(id); data.servers = await Servers() || []; data.message = 'Worker removed' } catch (error) { data.message = String(error) } finally { busy.value = false }
}
async function pullImage(id) {
  busy.value = true
  try { await PullMarketplaceImage(id); data.message = 'Image downloaded into the local runtime store'; await refresh() } catch (error) { data.message = `Image pull failed: ${String(error).replace(/^Error: /, '')}` } finally { busy.value = false }
}
async function convert() {
  busy.value = true
  try { const operation = await Convert(state.source, state.sourceType, state.baseProfile); data.message = `Conversion queued: ${operation.id || 'created'}`; state.tab = 'convert'; await refresh() } catch (error) { data.message = String(error) } finally { busy.value = false }
}
async function createVM() {
  const operation = latestOperation.value
  if (!operation?.artifact?.digest) { data.message = 'Complete an image conversion before creating a workload'; return }
  busy.value = true
  try { const vm = await CreateVM(operation.artifact.digest, Number(state.vcpus), Number(state.memoryMiB), operation.request.source, [{ hostPort: Number(state.hostPort), guestPort: Number(state.guestPort), protocol: state.protocol }]); state.selectedVM = vm.id; state.tab = 'vms'; data.message = `Created workload from ${operation.request.source}`; await refresh() } catch (error) { data.message = String(error) } finally { busy.value = false }
}
async function vmAction(id, action) { try { await VMAction(id, action); await refresh() } catch (error) { data.message = String(error) } }
async function snapshot(id, action) { try { await Snapshot(id, action, `${id}.snapshot`, `${id}.mem`); data.message = `${action} requested`; await refresh() } catch (error) { data.message = String(error) } }
function openVM(vm) { state.selectedVM = vm.id; state.tab = 'vms' }

onMounted(async () => {
  connectMetrics()
  data.servers = await Servers() || []
  const active = data.servers.find(server => server.active) || data.servers[0]
  if (active) { state.serverID = active.id; state.connectionURL = active.url }
  await refresh()
  refreshTimer = window.setInterval(() => { if (state.tab === 'convert' || state.tab === 'vms') refresh() }, 3000)
})
onBeforeUnmount(() => { metricsSource?.close(); if (refreshTimer) window.clearInterval(refreshTimer) })
</script>

<template>
  <div class="desktop-shell">
    <aside class="sidebar">
      <div class="brand"><span class="brand-mark">F</span><div><strong>Firecracker Studio</strong><small>microVM control</small></div></div>
      <div class="worker-card"><div class="worker-title"><span class="status-dot" :class="{ online: data.health }"></span><span>{{ data.health ? 'Connected' : 'Offline' }}</span></div><strong>{{ data.health?.status || 'Checking server' }}</strong><small>{{ state.connectionURL || 'No server selected' }}</small></div>
      <nav><button v-for="tab in tabs" :key="tab.id" :class="{ active: state.tab === tab.id }" @click="state.tab = tab.id"><span class="nav-icon">{{ tab.icon }}</span>{{ tab.label }}<span v-if="tab.id === 'vms' && data.vms.length" class="nav-count">{{ data.vms.length }}</span></button></nav>
      <div class="sidebar-bottom"><small>v1.2.0 · BlackLoverTech</small></div>
    </aside>

    <main class="main-area">
      <header class="topbar"><div><p class="eyebrow">{{ tabs.find(tab => tab.id === state.tab)?.label.toUpperCase() }}</p><h1>{{ state.tab === 'overview' ? 'Workspace' : tabs.find(tab => tab.id === state.tab)?.label }}</h1></div></header>

      <div class="content">
        <template v-if="state.tab === 'overview'">
          <section class="hero compact"><div><p class="eyebrow">FIRECRACKER CONTROL CENTER</p><h2>Run isolated workloads.</h2><p>{{ data.message || 'Your current server is ready for microVM management.' }}</p></div><button class="primary small" @click="state.tab = 'convert'">Convert image</button></section>
          <section class="metric-grid"><div class="metric"><span>MicroVMs</span><strong>{{ data.metrics.microvms }}</strong><small>{{ data.metrics.runningVms }} running · live</small></div><div class="metric"><span>Images</span><strong>{{ data.metrics.images }}</strong><small>local store · live</small></div><div class="metric"><span>Operations</span><strong>{{ data.metrics.operations }}</strong><small>conversion jobs · live</small></div><div class="metric"><span>Runtime</span><strong>{{ data.health ? 'Ready' : 'Offline' }}</strong><small>server connection</small></div></section>
          <section class="grid two"><article class="panel"><div class="panel-head"><div><p class="eyebrow">QUICK CONVERT</p><h2>OCI image to microVM</h2></div></div><label>Image reference<input v-model="state.source" placeholder="alpine:latest" /></label><div class="form-row"><label>Source<select v-model="state.sourceType"><option value="oci">OCI / Docker</option><option value="dockerfile">Dockerfile</option><option value="archive">OCI archive</option></select></label><label>Base<select v-model="state.baseProfile"><option>alpine</option><option>debian</option><option>ubuntu</option></select></label></div><button class="primary" @click="convert" :disabled="busy || !state.source">Convert</button></article><article class="panel"><div class="panel-head"><div><p class="eyebrow">RECENT MICROVMS</p><h2>Workloads</h2></div><button class="ghost small" @click="state.tab = 'vms'">View all</button></div><div v-if="data.vms.length" class="table"><div v-for="vm in data.vms.slice(0, 4)" :key="vm.id" class="table-row"><span class="mono">{{ vm.id.slice(0, 12) }}</span><span class="state" :class="vm.state">{{ vm.state }}</span><button class="ghost small" @click="openVM(vm)">Open</button></div></div><div v-else class="empty">No microVMs yet.</div></article></section>
        </template>

        <template v-else-if="state.tab === 'images'"><section class="toolbar"><div><p class="eyebrow">IMAGE STORE</p><h2>Images</h2><p class="section-note">Use familiar OCI names. Studio keeps the source name and adds the Firecracker artifacts behind it.</p></div><button class="primary small" @click="state.tab = 'convert'">Pull image</button></section><section class="grid two"><article class="panel"><div class="panel-head"><div><p class="eyebrow">LOCAL STORE</p><h2>Downloaded images</h2></div><span class="chip">{{ data.images.length }} local</span></div><div v-for="image in data.images" :key="image.digest" class="operation"><div class="operation-main"><strong>{{ image.reference }}</strong><small class="mono">{{ image.sourceType }} · {{ image.digest?.slice(0, 16) }}</small></div><span class="state succeeded">ready</span></div><div v-if="!data.images.length" class="empty">No local images yet. Pull an OCI image to create a reusable Firecracker image.</div></article><article class="panel"><div class="panel-head"><div><p class="eyebrow">STARTER IMAGES</p><h2>Built-in defaults</h2></div><span class="chip">verified sources</span></div><p class="section-note">These are optional bare kernel/rootfs starting points. Normal workloads should use the original OCI image name below.</p><div v-for="image in data.marketplaceImages" :key="image.id" class="operation"><div class="operation-main"><strong>{{ image.name }}</strong><small>{{ image.distribution }} {{ image.version }} · {{ image.architecture }}</small></div><span class="state" :class="image.status">{{ image.kernelUrl && image.rootfsUrl ? 'ready' : 'info' }}</span><button class="primary small" :disabled="busy || !image.kernelUrl || !image.rootfsUrl" @click="pullImage(image.id)">{{ image.kernelUrl && image.rootfsUrl ? 'Download' : 'Info' }}</button></div><div v-if="!data.marketplaceImages.length" class="empty">No starter images available from this server.</div></article></section></template>

        <template v-else-if="state.tab === 'convert'"><section class="toolbar"><div><p class="eyebrow">IMAGE WORKFLOW</p><h2>Pull and convert</h2><p class="section-note">Enter the same image name you use with Docker. Studio pulls it, converts it, and keeps that name for the resulting microVM image.</p></div></section><section class="panel form-panel"><label>OCI / Docker image name<input v-model="state.source" placeholder="alpine:3.20 or postgres:16-alpine" /></label><div class="form-row"><label>Source type<select v-model="state.sourceType"><option value="oci">OCI / Docker image</option><option value="dockerfile">Dockerfile</option><option value="archive">OCI archive</option></select></label><label>Guest base<select v-model="state.baseProfile"><option>alpine</option><option>debian</option><option>ubuntu</option></select></label></div><p class="hint">The source name is preserved. The downloaded kernel and generated rootfs are stored locally and reused by workloads.</p><div class="form-row"><label>Host port<input v-model.number="state.hostPort" type="number" min="1" max="65535" /></label><label>Guest port<input v-model.number="state.guestPort" type="number" min="1" max="65535" /></label><label>Protocol<select v-model="state.protocol"><option>tcp</option><option>udp</option></select></label></div><p class="hint">These ports are applied when you click Create workload after conversion. PostgreSQL normally uses guest port 5432.</p><button class="primary" @click="convert" :disabled="busy || !state.source">{{ busy ? 'Pulling and converting…' : 'Pull and convert' }}</button></section><section class="panel"><div class="panel-head"><div><p class="eyebrow">BUILD OUTPUT</p><h2>Operations</h2></div><span v-if="latestOperation" class="state" :class="latestOperation.state">{{ latestOperation.state }}</span></div><div v-if="latestOperation" class="build-output"><div class="detail-grid"><div><span>Source</span><strong>{{ latestOperation.request.source }}</strong></div><div><span>Operation</span><strong class="mono">{{ latestOperation.id.slice(0, 12) }}</strong></div><div><span>Digest</span><strong class="mono">{{ latestOperation.artifact?.digest || 'pending' }}</strong></div><div><span>Rootfs</span><strong class="mono">{{ latestOperation.artifact?.rootfs || 'pending' }}</strong></div></div><pre>{{ (latestOperation.logs || []).join('\\n') }}{{ latestOperation.error ? '\\n' + latestOperation.error : '' }}</pre><button v-if="latestOperation.state === 'succeeded'" class="primary small" @click="createVM">Create workload from {{ latestOperation.request.source }}</button></div><div class="operation-list"><div v-for="operation in data.operations.slice().sort((a, b) => new Date(b.updatedAt || b.createdAt) - new Date(a.updatedAt || a.createdAt))" :key="operation.id" class="operation"><div class="operation-main"><strong>{{ operation.request.source }}</strong><small class="mono">{{ operation.id.slice(0, 12) }} · {{ operation.kind }}</small></div><span class="state" :class="operation.state">{{ operation.state }}</span></div></div><div v-if="!data.operations.length" class="empty">No build operations yet. Try Alpine or PostgreSQL from the test workflow.</div></section></template>

        <template v-else-if="state.tab === 'vms'"><section class="toolbar"><div><p class="eyebrow">WORKLOADS</p><h2>Workloads</h2><p class="section-note">Run and inspect isolated Firecracker microVMs from one place.</p></div><button class="primary small" @click="state.tab = 'convert'">Create workload</button></section><div class="workload-layout"><section class="panel workload-list"><div class="panel-head"><div><p class="eyebrow">LOCAL WORKLOADS</p><h2>{{ data.vms.length }} microVM{{ data.vms.length === 1 ? '' : 's' }}</h2></div><button class="ghost small" @click="refresh">Refresh</button></div><div v-if="data.vms.length" v-for="vm in data.vms" :key="vm.id" class="workload-row" :class="{ selected: selectedVM?.id === vm.id }" @click="openVM(vm)"><span class="vm-avatar">▣</span><div class="workload-main"><strong>{{ vm.id.slice(0, 12) }}</strong><small>{{ vm.imageReference || vm.artifactDigest || 'No image reference' }}</small></div><span class="state" :class="vm.state">{{ vm.state }}</span></div><div v-else class="empty">No workloads yet. Convert an image to create your first microVM.</div></section><section class="workload-detail"><div v-if="selectedVM"><article class="panel workload-header"><div><p class="eyebrow">MICROVM WORKLOAD</p><h2>{{ selectedVM.id.slice(0, 12) }}</h2><p class="section-note">{{ selectedVM.state }} · created {{ new Date(selectedVM.createdAt).toLocaleString() }}</p></div><div class="action-row"><button class="primary small" @click="vmAction(selectedVM.id, 'start')" :disabled="selectedVM.state === 'running'">Start</button><button class="ghost small" @click="vmAction(selectedVM.id, 'stop')" :disabled="selectedVM.state !== 'running'">Stop</button><button class="ghost small" @click="snapshot(selectedVM.id, 'create')">Snapshot</button><button class="ghost small" @click="snapshot(selectedVM.id, 'restore')">Restore</button></div></article><article class="panel"><div class="panel-head"><div><p class="eyebrow">IMAGE</p><h2>Runtime artifact</h2></div><span class="state" :class="selectedVM.state">{{ selectedVM.state }}</span></div><div class="detail-grid"><div><span>Image</span><strong>{{ selectedVM.imageReference || 'Unnamed image' }}</strong></div><div><span>Artifact digest</span><strong class="mono">{{ selectedVM.artifactDigest || '—' }}</strong></div><div><span>CPU / memory</span><strong>Configured by worker</strong></div><div><span>Created</span><strong>{{ new Date(selectedVM.createdAt).toLocaleString() }}</strong></div><div><span>Updated</span><strong>{{ new Date(selectedVM.updatedAt).toLocaleString() }}</strong></div></div></article><article class="panel"><div class="panel-head"><div><p class="eyebrow">PORTS</p><h2>Published ports</h2></div></div><div v-if="selectedVM.portMappings?.length" class="port-list"><div v-for="port in selectedVM.portMappings" :key="`${port.hostPort}-${port.guestPort}-${port.protocol}`" class="port-row"><strong>{{ port.hostPort }}</strong><span>→</span><strong>{{ port.guestPort }}</strong><span class="chip">{{ port.protocol }}</span></div></div><div v-else class="empty small">No ports published for this workload.</div></article><article class="panel"><div class="panel-head"><div><p class="eyebrow">LOGS</p><h2>Workload logs</h2></div><span class="chip">lifecycle</span></div><pre class="workload-logs">{{ selectedVM.logs?.join('\\n') || 'No lifecycle logs yet.' }}</pre><p class="hint">Guest application stdout requires a guest agent or a configured serial-log transport. This panel currently shows Firecracker Studio lifecycle events.</p></article></div><div v-else class="panel empty large">Select a workload to inspect its image, ports, actions, and logs.</div></section></div></template>

        <template v-else-if="state.tab === 'servers'"><section class="toolbar"><div><p class="eyebrow">INFRASTRUCTURE</p><h2>Servers</h2></div><div class="action-row"><button class="ghost small" @click="refresh">Refresh</button><button class="primary small" @click="state.showAddWorker = true">Add server</button></div></section><section class="grid two"><article class="panel"><div class="panel-head"><div><p class="eyebrow">ACTIVE SERVER</p><h2>{{ activeServer?.name || 'No server' }}</h2></div><span class="state" :class="activeServer?.health">{{ activeServer?.managed ? 'managed' : (activeServer?.health || 'offline') }}</span></div><div class="detail-grid"><div><span>Endpoint</span><strong class="mono">{{ activeServer?.url || '—' }}</strong></div><div><span>Type</span><strong>{{ activeServer?.kind === 'local' ? 'Local host' : 'Remote worker' }}</strong></div><div><span>API status</span><strong>{{ data.health?.status || 'Unavailable' }}</strong></div><div><span>Last check</span><strong>{{ activeServer?.lastChecked || 'Not checked' }}</strong></div></div><div class="action-row"><button class="primary small" @click="checkServer(activeServer.id)">Check health</button><button class="ghost small" @click="switchServer">Use server</button></div></article><article class="panel"><div class="panel-head"><div><p class="eyebrow">RUNTIME</p><h2>Host readiness</h2></div><span class="state" :class="data.health ? 'healthy' : 'unhealthy'">{{ data.health ? 'online' : 'offline' }}</span></div><div class="check-row"><span>Firecracker</span><span class="state catalog">{{ data.runtime.firecracker || 'managed runtime' }}</span></div><div class="check-row"><span>Jailer</span><span class="state catalog">{{ data.runtime.jailer || 'managed runtime' }}</span></div><div class="check-row"><span>KVM</span><span class="state catalog">{{ data.runtime.kvm || 'runtime check' }}</span></div><div class="check-row"><span>TAP networking</span><span class="state catalog">{{ data.runtime.tap || 'runtime check' }}</span></div></article></section><section class="grid two"><article class="panel"><div class="panel-head"><div><p class="eyebrow">SERVER LOG</p><h2>Recent events</h2></div><button class="ghost small" @click="refresh">Refresh</button></div><div class="terminal-panel"><div class="terminal-head"><span class="status-dot" :class="{ online: data.health }"></span>firecracker-studio <span class="terminal-muted">recent server events</span></div><pre>{{ data.logs.slice(-12).join('\n') || data.message || 'Waiting for server events…' }}</pre></div></article><article class="panel"><div class="panel-head"><div><p class="eyebrow">HOST RESOURCES</p><h2>CPU, memory, disk, network</h2></div><span class="chip">live</span></div><div class="resource-grid"><div><span>CPU load</span><strong>{{ data.metrics.host?.cpuPercent?.toFixed?.(1) || '0.0' }}%</strong></div><div><span>RAM</span><strong>{{ formatBytes(data.metrics.host?.memoryUsedBytes) }} / {{ formatBytes(data.metrics.host?.memoryTotalBytes) }}</strong><small>{{ data.metrics.host?.memoryPercent?.toFixed?.(1) || '0.0' }}%</small></div><div><span>Disk</span><strong>{{ formatBytes(data.metrics.host?.diskUsedBytes) }} / {{ formatBytes(data.metrics.host?.diskTotalBytes) }}</strong><small>{{ data.metrics.host?.diskPercent?.toFixed?.(1) || '0.0' }}%</small></div><div><span>Bandwidth RX</span><strong>{{ formatBytes(data.metrics.host?.networkRxBytes) }}</strong><small>since boot</small></div><div><span>Bandwidth TX</span><strong>{{ formatBytes(data.metrics.host?.networkTxBytes) }}</strong><small>since boot</small></div><div><span>GPU</span><strong>{{ data.metrics.host?.gpuAvailable ? `${data.metrics.host.gpuUsagePercent || 0}%` : 'Not detected' }}</strong><small>{{ data.metrics.host?.gpuMessage || 'No GPU telemetry' }}</small></div></div></article></section><section class="panel"><div class="panel-head"><div><p class="eyebrow">SERVER LIST</p><h2>All servers</h2></div><span class="chip">{{ data.servers.length }} configured</span></div><div v-for="server in data.servers" :key="server.id" class="server-row"><div><strong>{{ server.name }}</strong><small>{{ server.kind === 'local' ? 'Local host' : 'Remote worker' }} · {{ server.url }}</small></div><span class="state" :class="server.health">{{ server.managed ? 'managed' : server.health }}</span><button class="ghost small" @click="checkServer(server.id)">Check</button><button class="primary small" :disabled="server.health !== 'healthy'" @click="state.serverID = server.id; switchServer()">Use</button><button v-if="!server.managed" class="ghost small danger" @click="removeServer(server.id)">Remove</button></div><div v-if="!data.servers.length" class="empty">No servers configured.</div></section></template>
      </div>
    </main>

    <div v-if="state.showAddWorker" class="modal-backdrop" @click.self="state.showAddWorker = false"><form class="modal panel" @submit.prevent="addServer"><div class="panel-head"><div><p class="eyebrow">REMOTE WORKER</p><h2>Add worker</h2></div><button type="button" class="ghost" @click="state.showAddWorker = false">×</button></div><p class="section-note">Enter the remote Firecracker Studio URL and optional bearer token. The worker is checked before it is added.</p><label>Name<input v-model="state.newServer.name" required /></label><label>Worker URL<input v-model="state.newServer.url" required placeholder="https://worker.example.com" /></label><label>Bearer token<input v-model="state.newServer.token" type="password" /></label><div class="action-row"><button type="button" class="ghost" @click="state.showAddWorker = false">Cancel</button><button class="primary small" :disabled="busy">{{ busy ? 'Checking…' : 'Add worker' }}</button></div></form></div>
  </div>
</template>
