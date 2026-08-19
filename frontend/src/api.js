const webBase = () => (import.meta.env.VITE_FIRECRACKER_API_URL || '/api/v1').replace(/\/$/, '')

const demoBases = [
  { id: 'alpine-3.24.1-x86_64', distribution: 'alpine', version: '3.24.1', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'openrc', status: 'catalog', default: true },
  { id: 'alpine-3.24.1-aarch64', distribution: 'alpine', version: '3.24.1', architecture: 'aarch64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'openrc', status: 'catalog' },
  { id: 'debian-12-x86_64', distribution: 'debian', version: '12', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
  { id: 'ubuntu-22.04-x86_64', distribution: 'ubuntu', version: '22.04', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
]

const json = async (response) => {
  const body = await response.text()
  let data = {}
  try { data = body ? JSON.parse(body) : {} } catch { data = { raw: body } }
  if (!response.ok) throw new Error(data.message || data.error || `HTTP ${response.status}`)
  return data
}
const webRequest = async (path, options = {}) => {
  const response = await fetch(`${webBase()}${path}`, {
    ...options,
    headers: { Accept: 'application/json', ...(options.body ? { 'Content-Type': 'application/json' } : {}), ...(options.headers || {}) },
  })
  return json(response)
}
const call = (_name, webFallback) => (...args) => webFallback(...args)

const localServer = () => ({
  id: 'local-current-server',
  name: 'This Firecracker Studio server',
  url: window.location.origin,
  kind: 'local',
  health: 'healthy',
  active: true,
  managed: true,
  lastChecked: new Date().toISOString(),
})
const storedServers = () => {
  try {
    const value = JSON.parse(localStorage.getItem('firecracker-studio.servers') || '[]')
    return Array.isArray(value) ? value : []
  } catch { return [] }
}
const saveServers = (servers) => localStorage.setItem('firecracker-studio.servers', JSON.stringify(servers))
const webServers = () => {
  const current = localServer()
  const stored = storedServers()
  const existing = stored.find((entry) => entry.id === current.id)
  const local = { ...current, ...(existing || {}), url: current.url, health: 'healthy', managed: true, active: existing ? existing.active : true }
  const remote = stored.filter((entry) => entry.id !== current.id)
  if (!remote.some((entry) => entry.active) && !local.active) local.active = true
  const servers = [local, ...remote]
  saveServers(servers)
  return servers
}

export const Servers = call('Servers', async () => webServers())
export const AddServer = call('AddServer', async (server) => {
  const health = await fetch(`${server.url.replace(/\/$/, '')}/api/v1/health`, { headers: server.token ? { Authorization: `Bearer ${server.token}` } : {} })
  if (!health.ok) throw new Error(`health check returned HTTP ${health.status}`)
  const item = { ...server, id: server.id || crypto.randomUUID(), health: 'healthy', active: true, lastChecked: new Date().toISOString() }
  const servers = webServers().map((entry) => ({ ...entry, active: false }))
  saveServers([...servers, item])
  return item
})
export const CheckServer = call('CheckServer', async (id) => {
  const servers = webServers()
  const server = servers.find((entry) => entry.id === id)
  if (!server) throw new Error('server not found')
  const response = await fetch(`${server.url.replace(/\/$/, '')}/api/v1/health`, { headers: server.token ? { Authorization: `Bearer ${server.token}` } : {} })
  const updated = { ...server, health: response.ok ? 'healthy' : 'unhealthy', lastChecked: new Date().toISOString() }
  saveServers(servers.map((entry) => entry.id === id ? updated : entry))
  if (!response.ok) throw new Error(`health check returned HTTP ${response.status}`)
  return updated
})
export const RemoveServer = call('RemoveServer', async (id) => {
  if (id === 'local-current-server') throw new Error('the current Firecracker Studio server cannot be removed')
  return saveServers(webServers().filter((entry) => entry.id !== id))
})
export const SwitchServer = call('SwitchServer', async (id) => {
  const servers = webServers().map((entry) => ({ ...entry, active: entry.id === id }))
  saveServers(servers)
  return servers.find((entry) => entry.id === id)
})
export const SetConnection = call('SetConnection', async () => undefined)

export const Health = call('Health', () => webRequest('/health'))
export const Metrics = call('Metrics', () => webRequest('/metrics'))
export const Logs = call('Logs', () => webRequest('/logs'))
export const MetricsStream = (onMessage, onError) => {
  const source = new EventSource(`${webBase()}/metrics/stream`)
  source.addEventListener('metrics', event => {
    try { onMessage(JSON.parse(event.data)) } catch (error) { onError?.(error) }
  })
  source.onerror = event => onError?.(event)
  return source
}
export const BaseImages = call('BaseImages', () => webRequest('/base-images').catch(() => ({ images: demoBases })))
export const Images = call('Images', () => webRequest('/images'))
export const Operations = call('Operations', () => webRequest('/operations'))
export const VMs = call('VMs', () => webRequest('/vms'))
export const Convert = call('Convert', (source, sourceType, baseProfile) => webRequest('/conversions', { method: 'POST', body: JSON.stringify({ source, sourceType, baseProfile, architecture: 'native' }) }))
export const CreateVM = call('CreateVM', (artifactDigest, vcpus, memoryMiB, imageReference = '', portMappings = []) => webRequest('/vms', { method: 'POST', body: JSON.stringify({ artifactDigest, imageReference, vcpus, memoryMiB, portMappings }) }))
export const VMAction = call('VMAction', (id, action) => webRequest(`/vms/${encodeURIComponent(id)}/${action}`, { method: 'POST', body: '{}' }))
export const DeleteVM = call('DeleteVM', id => webRequest(`/vms/${encodeURIComponent(id)}`, { method: 'DELETE' }))
export const RuntimeStatus = call('RuntimeStatus', () => webRequest('/readiness').then(result => ({ platform: 'web', installed: result.runtime?.installed, firecracker: result.runtime?.firecracker, jailer: result.runtime?.jailer, kvm: result.runtime?.kvm, tap: result.runtime?.tap, kernel: result.runtime?.kernel, rootfs: result.runtime?.rootfs, ready: result.ready, message: result.message })))
