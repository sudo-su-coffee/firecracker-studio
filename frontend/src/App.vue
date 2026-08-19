<script setup>
import { useStudio } from './composables/useStudio'
import OverviewView from './views/OverviewView.vue'
import ConvertView from './views/ConvertView.vue'
import WorkloadsView from './views/WorkloadsView.vue'
import ImagesView from './views/ImagesView.vue'

const tabs = [
  { id: 'overview', label: 'Dashboard', icon: '⌂', view: OverviewView },
  { id: 'convert', label: 'Convert', icon: '⇄', view: ConvertView },
  { id: 'vms', label: 'Workloads', icon: '▣', view: WorkloadsView },
  { id: 'images', label: 'Images', icon: '◈', view: ImagesView },
]

const { state, data, runtimeReady, saveToken, selectTab } = useStudio()

const activeTab = () => tabs.find(tab => tab.id === state.tab) || tabs[0]
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark">F</span>
        <div><strong>Firecracker Studio</strong><small>microVM control</small></div>
      </div>
      <div class="connection-card">
        <span class="status-dot" :class="{ online: data.health }"></span>
        <div><strong>{{ data.health ? 'Connected' : 'Offline' }}</strong><small>{{ runtimeReady ? 'Runtime ready' : 'Check readiness' }}</small></div>
      </div>
      <nav>
        <button v-for="tab in tabs" :key="tab.id" :class="{ active: state.tab === tab.id }" @click="selectTab(tab.id)">
          <span>{{ tab.icon }}</span>{{ tab.label }}
          <b v-if="tab.id === 'vms' && data.vms.length">{{ data.vms.length }}</b>
        </button>
      </nav>
      <div class="sidebar-foot">v1.5.0 · local control server</div>
    </aside>

    <main class="main">
      <header class="topbar">
        <div>
          <small class="overline">{{ activeTab().label }}</small>
          <h1>{{ state.tab === 'overview' ? 'Workspace' : activeTab().label }}</h1>
        </div>
        <div class="token-box">
          <input v-model="state.token" type="password" placeholder="API token (optional)" @keyup.enter="saveToken" />
          <button class="button subtle" @click="saveToken">Save</button>
        </div>
      </header>

      <div v-if="data.message" class="notice" role="status">{{ data.message }}</div>

      <component :is="activeTab().view" />
    </main>
  </div>
</template>
