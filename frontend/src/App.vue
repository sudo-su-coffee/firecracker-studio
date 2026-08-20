<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AuthStatus, Login, Logout, SetAuthToken } from './api'

const router = useRouter()
const route = useRoute()
const busy = ref(false)
const message = ref('')
const state = reactive({ configured: false, authenticated: false, username: '', usernameInput: 'admin', password: '', token: '' })
const nav = [
  { path: '/', label: 'Overview', icon: '▦' },
  { path: '/workloads', label: 'Workloads', icon: '▣' },
  { path: '/images', label: 'Images & Builds', icon: '◇' },
  { path: '/operations', label: 'Operations', icon: '◌' },
  { path: '/snapshots', label: 'Snapshots', icon: '◫' },
  { path: '/storage', label: 'Storage', icon: '▤' },
  { path: '/kernels', label: 'Kernels', icon: '⌁' },
  { path: '/metrics', label: 'Metrics', icon: '▥' },
  { path: '/host-readiness', label: 'Host Readiness', icon: '◈' },
  { path: '/activity', label: 'Activity Logs', icon: '☷' },
  { path: '/security', label: 'Security', icon: '⚿' },
]
const isActive = path => path === '/' ? route.path === '/' : route.path.startsWith(path)
async function login() { busy.value = true; try { const result = await Login(state.usernameInput, state.password); state.authenticated = result.authenticated === true; state.username = result.username || state.usernameInput; state.password = ''; message.value = 'Signed in' } catch (error) { message.value = String(error).replace(/^Error: /, '') } finally { busy.value = false } }
async function logout() { await Logout().catch(() => {}); state.authenticated = false; state.username = ''; message.value = 'Signed out' }
onMounted(async () => { SetAuthToken(state.token); try { const auth = await AuthStatus(); state.configured = auth.configured === true; state.authenticated = auth.authenticated === true; state.username = auth.username || '' } catch (error) { message.value = String(error).replace(/^Error: /, '') } })
</script>

<template>
  <section v-if="state.configured && !state.authenticated" class="login-shell">
    <div class="login-card panel"><div class="brand-block login-brand"><div class="brand-mark">⌁</div><div><h1>Firecracker<br />Studio</h1><p>Remote microVM console</p></div></div><span class="section-kicker">SECURE SIGN IN</span><h2>Open your Studio workspace</h2><form @submit.prevent="login"><label>ADMIN USERNAME<input v-model="state.usernameInput" autocomplete="username" required /></label><label>ADMIN PASSWORD<input v-model="state.password" type="password" autocomplete="current-password" required /></label><button class="primary-button" :disabled="busy">{{ busy ? 'Signing in…' : 'Sign in to Studio' }}</button></form><p v-if="message" class="login-error">{{ message }}</p></div>
  </section>
  <div v-else class="studio-shell">
    <aside class="side-nav"><div class="brand-block"><div class="brand-mark">⌁</div><div><h1>Firecracker<br />Studio</h1><p><span class="dot success"></span> Remote control plane</p></div></div><button class="new-vm-button" @click="router.push('/images/import')"><span>＋</span> New MicroVM</button><nav class="nav-list" aria-label="Main navigation"><button v-for="item in nav" :key="item.path" class="nav-item" :class="{ active: isActive(item.path) }" @click="router.push(item.path)"><span class="nav-icon">{{ item.icon }}</span><span>{{ item.label }}</span></button></nav><div class="sidebar-footer">v2.0.0 · 58 API routes</div></aside>
    <main class="studio-main"><header class="top-nav"><div class="top-title"><strong>Firecracker Studio</strong><div class="search-box"><span>⌕</span><input placeholder="Search routes and resources..." @keyup.enter="router.push('/workloads')" /></div></div><div class="top-actions"><span v-if="state.username" class="api-pill">{{ state.username }}</span><button class="avatar-button" title="Security" @click="router.push('/security')">⚿</button><button v-if="state.authenticated" class="icon-button" title="Sign out" @click="logout">⇥</button></div></header><div v-if="message" class="toast" role="status"><span>{{ message }}</span><button @click="message = ''">×</button></div><router-view /></main>
  </div>
</template>
