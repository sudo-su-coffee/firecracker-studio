const webBase = () => (import.meta.env.VITE_FIRECRACKER_API_URL || '/api/v1').replace(/\/$/, '')
let apiToken = ''

export const SetAuthToken = (token = '') => { apiToken = token.trim() }

const demoBases = [
  { id: 'alpine-3.24.1-x86_64', distribution: 'alpine', version: '3.24.1', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'openrc', status: 'catalog', default: true },
  { id: 'alpine-3.24.1-aarch64', distribution: 'alpine', version: '3.24.1', architecture: 'aarch64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'openrc', status: 'catalog' },
  { id: 'debian-12-x86_64', distribution: 'debian', version: '12', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
  { id: 'ubuntu-22.04-x86_64', distribution: 'ubuntu', version: '22.04', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
]

const json = async response => {
  const body = await response.text()
  let data = {}
  try { data = body ? JSON.parse(body) : {} } catch { data = { raw: body } }
  if (!response.ok) throw new Error(data.message || data.error || `HTTP ${response.status}`)
  return data
}

const webRequest = async (path, options = {}) => {
  const response = await fetch(`${webBase()}${path}`, {
    credentials: 'include',
    ...options,
    headers: { Accept: 'application/json', ...(options.body ? { 'Content-Type': 'application/json' } : {}), ...(apiToken ? { Authorization: `Bearer ${apiToken}` } : {}), ...(options.headers || {}) },
  })
  return json(response)
}
const call = (_name, webFallback) => (...args) => webFallback(...args)

export const AuthStatus = () => webRequest('/auth/status')
export const Login = (username, password) => webRequest('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) })
export const Logout = () => webRequest('/auth/logout', { method: 'POST', body: '{}' })
export const Health = call('Health', () => webRequest('/health'))
export const Metrics = call('Metrics', () => webRequest('/metrics'))
export const MetricsStream = (onMessage, onError) => {
  const streamToken = apiToken ? `?access_token=${encodeURIComponent(apiToken)}` : ''
  const source = new EventSource(`${webBase()}/metrics/stream${streamToken}`, { withCredentials: true })
  source.addEventListener('metrics', event => { try { onMessage(JSON.parse(event.data)) } catch (error) { onError?.(error) } })
  source.onerror = event => onError?.(event)
  return source
}
export const BaseImages = call('BaseImages', () => webRequest('/base-images').catch(() => ({ images: demoBases })))
export const Images = call('Images', () => webRequest('/images'))
export const ImageDetail = call('ImageDetail', id => webRequest(`/images/${encodeURIComponent(id)}`))
export const ImageStorageStats = call('ImageStorageStats', () => webRequest('/images/storage-stats'))
export const ImageTemplates = call('ImageTemplates', () => webRequest('/images/templates'))
export const RegisterImage = call('RegisterImage', image => webRequest('/images', { method: 'POST', body: JSON.stringify(image) }))
export const DeleteImage = call('DeleteImage', digest => webRequest(`/images/${encodeURIComponent(digest)}`, { method: 'DELETE' }))
export const CloneImage = call('CloneImage', id => webRequest(`/images/${encodeURIComponent(id)}/clone`, { method: 'POST' }))
export const PruneImages = call('PruneImages', () => webRequest('/images/prune', { method: 'POST' }))
export const ResolveGitHub = call('ResolveGitHub', reference => webRequest(`/sources/github?reference=${encodeURIComponent(reference)}`))
export const ValidateYAML = call('ValidateYAML', yaml => webRequest('/sources/yaml', { method: 'POST', body: JSON.stringify(typeof yaml === 'string' ? { yaml } : yaml) }))
export const Operations = call('Operations', () => webRequest('/operations'))
export const OperationDetail = call('OperationDetail', id => webRequest(`/operations/${encodeURIComponent(id)}`))
export const Convert = call('Convert', (source, sourceType, baseProfile, architecture = 'native') => webRequest('/conversions', { method: 'POST', body: JSON.stringify({ source, sourceType, baseProfile, architecture }) }))

export const VMs = call('VMs', () => webRequest('/vms'))
export const VMDetail = call('VMDetail', id => webRequest(`/vms/${encodeURIComponent(id)}`))
export const VMProcess = call('VMProcess', id => webRequest(`/vms/${encodeURIComponent(id)}/process`))
export const CreateVM = call('CreateVM', (artifactDigest, vcpus, memoryMiB, imageReference = '', portMappings = [], storageMode = 'ephemeral', persistentDisk = '') => webRequest('/vms', { method: 'POST', body: JSON.stringify({ artifactDigest, imageReference, vcpus, memoryMiB, portMappings, storageMode, persistentDisk }) }))
export const VMAction = call('VMAction', (id, action) => webRequest(`/vms/${encodeURIComponent(id)}/${action}`, { method: 'POST' }))
export const GuestCommand = call('GuestCommand', (id, command) => webRequest(`/vms/${encodeURIComponent(id)}/terminal`, { method: 'POST', body: JSON.stringify({ command }) }))
export const DeleteVM = call('DeleteVM', id => webRequest(`/vms/${encodeURIComponent(id)}`, { method: 'DELETE' }))
export const VMConfig = call('VMConfig', (id, method = 'GET', body) => webRequest(`/vms/${encodeURIComponent(id)}/config`, { method, body: body ? JSON.stringify(body) : undefined }))
export const VMCPUConfig = call('VMCPUConfig', (id, method = 'GET', body) => webRequest(`/vms/${encodeURIComponent(id)}/cpu-config`, { method, body: body ? JSON.stringify(body) : undefined }))
export const VMBootSource = call('VMBootSource', (id, method = 'GET', body) => webRequest(`/vms/${encodeURIComponent(id)}/boot-source`, { method, body: body ? JSON.stringify(body) : undefined }))
export const VMSMT = call('VMSMT', (id, method = 'GET', body) => webRequest(`/vms/${encodeURIComponent(id)}/smt`, { method, body: body ? JSON.stringify(body) : undefined }))
export const VMConstraints = call('VMConstraints', id => webRequest(`/vms/${encodeURIComponent(id)}/constraints`))
export const VMLogs = call('VMLogs', id => webRequest(`/vms/${encodeURIComponent(id)}/logs`))
export const VMMetrics = call('VMMetrics', id => webRequest(`/vms/${encodeURIComponent(id)}/metrics`))
export const VMSock = call('VMSock', (id, method = 'GET', body) => webRequest(`/vms/${encodeURIComponent(id)}/vsock`, { method, body: body ? JSON.stringify(body) : undefined }))
export const SnapshotCreate = call('SnapshotCreate', (id, body = {}) => webRequest(`/vms/${encodeURIComponent(id)}/snapshots`, { method: 'POST', body: JSON.stringify(body) }))
export const SnapshotCreateAlias = call('SnapshotCreateAlias', (id, body = {}) => webRequest(`/vms/${encodeURIComponent(id)}/snapshot`, { method: 'POST', body: JSON.stringify(body) }))
export const SnapshotRestore = call('SnapshotRestore', (id, body = {}) => webRequest(`/vms/${encodeURIComponent(id)}/snapshots/restore`, { method: 'POST', body: JSON.stringify(body) }))
export const SnapshotRestoreAlias = call('SnapshotRestoreAlias', (id, body = {}) => webRequest(`/vms/${encodeURIComponent(id)}/restore`, { method: 'POST', body: JSON.stringify(body) }))
export const SnapshotDelete = call('SnapshotDelete', id => webRequest(`/vms/${encodeURIComponent(id)}/snapshots`, { method: 'DELETE' }))

export const RuntimeStatus = call('RuntimeStatus', () => webRequest('/readiness').then(result => ({ platform: 'web', installed: result.runtime?.installed, firecracker: result.runtime?.firecracker, jailer: result.runtime?.jailer, kvm: result.runtime?.kvm, tap: result.runtime?.tap, kernel: result.runtime?.kernel, rootfs: result.runtime?.rootfs, ready: result.ready, message: result.message })))
export const Kernels = call('Kernels', () => webRequest('/images/kernels'))
export const RegisterKernel = call('RegisterKernel', kernel => webRequest('/images/kernels', { method: 'POST', body: JSON.stringify(kernel) }))
export const DeleteKernel = call('DeleteKernel', id => webRequest(`/images/kernels/${encodeURIComponent(id)}`, { method: 'DELETE' }))
export const Volumes = call('Volumes', () => webRequest('/volumes'))
export const CreateVolume = call('CreateVolume', volume => webRequest('/volumes', { method: 'POST', body: JSON.stringify(volume) }))
export const DeleteVolume = call('DeleteVolume', id => webRequest(`/volumes/${encodeURIComponent(id)}`, { method: 'DELETE' }))
export const SystemInfo = call('SystemInfo', () => webRequest('/system/info'))
export const SystemStats = call('SystemStats', () => webRequest('/system/stats'))
export const Logs = call('Logs', () => webRequest('/logs'))
