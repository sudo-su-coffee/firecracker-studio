import * as wails from '../wailsjs/go/main/App'

const demoBases = [
  { id: 'alpine-3.24.1-x86_64', distribution: 'alpine', version: '3.24.1', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'openrc', status: 'catalog', default: true },
  { id: 'alpine-3.24.1-aarch64', distribution: 'alpine', version: '3.24.1', architecture: 'aarch64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'openrc', status: 'catalog' },
  { id: 'debian-12-x86_64', distribution: 'debian', version: '12', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
  { id: 'debian-12-aarch64', distribution: 'debian', version: '12', architecture: 'aarch64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
  { id: 'ubuntu-22.04-x86_64', distribution: 'ubuntu', version: '22.04', architecture: 'x86_64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
  { id: 'ubuntu-22.04-aarch64', distribution: 'ubuntu', version: '22.04', architecture: 'aarch64', kernelChannel: '6.1', rootfsFormat: 'ext4', initSystem: 'systemd', status: 'catalog' },
]

const native = () => typeof window !== 'undefined' && window.go?.main?.App ? window.go.main.App : null
const call = (name, fallback) => (...args) => native()?.[name] ? native()[name](...args) : Promise.resolve(fallback())
export const Accounts = call('Accounts', () => [{ id: 'preview-account', name: 'Personal', username: 'local' }])
export const Servers = call('Servers', () => [{ id: 'preview-server', name: 'Local worker', url: 'http://127.0.0.1:7822', kind: 'local', health: 'preview', active: true }])
export const AddAccount = call('AddAccount', (name, username) => ({ id: `account-${Date.now()}`, name, username }))
export const AddServer = call('AddServer', (name, url, kind, username) => ({ id: `server-${Date.now()}`, name, url, kind, username, health: 'preview', active: false }))
export const CheckServer = call('CheckServer', (id) => ({ id, health: 'preview' }))
export const SwitchServer = call('SwitchServer', (id) => ({ id, health: 'preview', active: true }))
export const SetConnection = call('SetConnection', () => undefined)
export const Health = call('Health', () => ({ status: 'preview', service: 'firecracker-studio' }))
export const BaseImages = call('BaseImages', () => ({ images: demoBases }))
export const Images = call('Images', () => ({ images: [] }))
export const Operations = call('Operations', () => ({ operations: [] }))
export const VMs = call('VMs', () => ({ vms: [] }))
export const Convert = call('Convert', () => ({ id: 'preview-operation' }))
export const CreateVM = call('CreateVM', () => ({ id: 'preview-vm', state: 'created' }))
export const VMAction = call('VMAction', () => ({ state: 'preview' }))
export const Snapshot = call('Snapshot', () => ({ status: 'preview' }))
