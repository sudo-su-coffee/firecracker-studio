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
export const Servers = call('Servers', () => [])
export const AddAccount = call('AddAccount', (name, username) => ({ id: `account-${Date.now()}`, name, username }))
export const AddServer = call('AddServer', () => Promise.reject(new Error('Run Firecracker Studio as the Wails desktop app to store server profiles')))
export const CheckServer = call('CheckServer', () => Promise.reject(new Error('Run Firecracker Studio as the Wails desktop app to check workers')))
export const RemoveServer = call('RemoveServer', () => Promise.reject(new Error('Run Firecracker Studio as the Wails desktop app to remove worker profiles')))
export const SwitchServer = call('SwitchServer', () => Promise.reject(new Error('Run Firecracker Studio as the Wails desktop app to switch workers')))
export const SetConnection = call('SetConnection', () => undefined)
export const Health = call('Health', () => Promise.reject(new Error('Connect a real Porter worker to load data')))
export const BaseImages = call('BaseImages', () => ({ images: demoBases }))
export const Images = call('Images', () => ({ images: [] }))
export const Operations = call('Operations', () => ({ operations: [] }))
export const VMs = call('VMs', () => ({ vms: [] }))
export const Convert = call('Convert', () => Promise.reject(new Error('Connect a real Porter worker before converting images')))
export const CreateVM = call('CreateVM', () => Promise.reject(new Error('Connect a real Porter worker before creating microVMs')))
export const VMAction = call('VMAction', () => ({ state: 'preview' }))
export const Snapshot = call('Snapshot', () => ({ status: 'preview' }))
export const RuntimeStatus = call('RuntimeStatus', () => ({ platform: 'preview', installed: false, firecracker: 'desktop runtime required', jailer: 'desktop runtime required', kvm: 'check WSL2/Linux', tap: 'check WSL2/Linux', kernel: 'catalog', rootfs: 'catalog', message: 'Run the Wails desktop app to inspect the local runtime' }))
export const InstallRuntime = call('InstallRuntime', () => Promise.reject(new Error('Run the Wails desktop app to install the local Firecracker runtime')))
