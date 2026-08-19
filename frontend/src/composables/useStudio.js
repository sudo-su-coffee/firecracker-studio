import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  BaseImages, Convert, CreateVM, DeleteVM, Health, Images,
  MetricsStream, Operations, RuntimeStatus, SetAuthToken, VMAction, VMs,
} from '../api'

// Single shared reactive store for the whole app. Vue's module-level
// `reactive()` call means every component that imports and calls
// `useStudio()` gets the *same* instance, so state stays in sync across
// views without prop drilling or a separate store library.
const state = reactive({
  tab: 'overview',
  token: localStorage.getItem('firecracker-studio.api-token') || '',
  source: 'alpine:latest',
  sourceType: 'oci',
  baseProfile: 'alpine',
  vcpus: 1,
  memoryMiB: 512,
  enableNetworking: true,
  portMappings: [{ hostPort: 15432, guestPort: 5432, protocol: 'tcp' }],
  selectedVM: '',
})

const data = reactive({
  health: null,
  runtime: {},
  metrics: { microvms: 0, runningVms: 0, images: 0, operations: 0, host: {} },
  vms: [],
  images: [],
  baseImages: [],
  operations: [],
  message: '',
})

const busy = ref(false)
let metricsSource
let refreshTimer
let mountedCount = 0

export function useStudio() {
  const selectedVM = computed(() => data.vms.find(vm => vm.id === state.selectedVM) || data.vms[0] || null)
  const latestOperation = computed(() =>
    data.operations.slice().sort((a, b) => new Date(b.updatedAt || b.createdAt) - new Date(a.updatedAt || a.createdAt))[0] || null
  )
  const runtimeReady = computed(() => data.runtime.ready === true)

  function formatBytes(value = 0) {
    if (!value) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
    return `${(value / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}`
  }
  function errorText(error) { return String(error).replace(/^Error: /, '') }
  function selectTab(tab) { state.tab = tab }

  function startMetrics() {
    metricsSource?.close()
    metricsSource = MetricsStream(metrics => { data.metrics = metrics }, () => {})
  }
  function saveToken() {
    SetAuthToken(state.token)
    data.message = state.token ? 'API token saved in this browser' : 'API token cleared'
    startMetrics()
    refresh()
  }

  async function refresh() {
    try {
      data.runtime = await RuntimeStatus()
      data.health = await Health()
      const [vms, images, baseImages, operations] = await Promise.all([VMs(), Images(), BaseImages(), Operations()])
      data.vms = vms.vms || []
      data.images = images.images || []
      data.baseImages = baseImages.images || []
      data.operations = operations.operations || []
      if (!state.selectedVM && data.vms[0]) state.selectedVM = data.vms[0].id
    } catch (error) {
      data.health = null
      data.message = errorText(error)
    }
  }

  async function convert() {
    busy.value = true
    try {
      const operation = await Convert(state.source, state.sourceType, state.baseProfile)
      data.message = `Conversion queued: ${operation.id}`
      state.tab = 'convert'
      await refresh()
    } catch (error) {
      data.message = errorText(error)
    } finally {
      busy.value = false
    }
  }

  function addPortMapping() {
    state.portMappings.push({ hostPort: '', guestPort: '', protocol: 'tcp' })
  }
  function removePortMapping(index) {
    state.portMappings.splice(index, 1)
  }

  async function createVM() {
    if (!latestOperation.value?.artifact?.digest) {
      data.message = 'Wait for a successful conversion before creating a workload'
      return
    }
    busy.value = true
    try {
      const mappings = state.portMappings
        .filter(m => m.hostPort && m.guestPort)
        .map(m => ({ hostPort: Number(m.hostPort), guestPort: Number(m.guestPort), protocol: m.protocol || 'tcp' }))
      const vm = await CreateVM(
        latestOperation.value.artifact.digest,
        Number(state.vcpus),
        Number(state.memoryMiB),
        latestOperation.value.request.source,
        mappings,
        state.enableNetworking,
      )
      state.selectedVM = vm.id
      state.tab = 'vms'
      data.message = 'Workload created. Press Start when ready.'
      await refresh()
    } catch (error) {
      data.message = errorText(error)
    } finally {
      busy.value = false
    }
  }

  async function vmAction(id, action) {
    busy.value = true
    try {
      await VMAction(id, action)
      data.message = `Workload ${action} requested`
      await refresh()
    } catch (error) {
      data.message = errorText(error)
    } finally {
      busy.value = false
    }
  }

  async function deleteVM(id) {
    if (!window.confirm('Delete this microVM and its managed files?')) return
    busy.value = true
    try {
      await DeleteVM(id)
      state.selectedVM = ''
      data.message = 'Workload deleted'
      await refresh()
    } catch (error) {
      data.message = errorText(error)
    } finally {
      busy.value = false
    }
  }

  onMounted(async () => {
    mountedCount += 1
    if (mountedCount > 1) return // only the root App instance drives polling
    SetAuthToken(state.token)
    startMetrics()
    await refresh()
    refreshTimer = window.setInterval(() => {
      if (state.tab === 'vms' || state.tab === 'convert') refresh()
    }, 4000)
  })
  onBeforeUnmount(() => {
    mountedCount -= 1
    if (mountedCount > 0) return
    metricsSource?.close()
    if (refreshTimer) window.clearInterval(refreshTimer)
  })

  return {
    state, data, busy,
    selectedVM, latestOperation, runtimeReady,
    formatBytes, errorText, selectTab,
    saveToken, refresh, convert, createVM, vmAction, deleteVM,
    addPortMapping, removePortMapping,
  }
}
