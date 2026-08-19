<script setup>
import { useStudio } from '../composables/useStudio'

const { state, data, runtimeReady, selectTab } = useStudio()
</script>

<template>
  <section class="view">
    <div class="hero">
      <div>
        <small class="overline">LOCAL FIRECRACKER CONTROL</small>
        <h2>Run microVMs without the socket work.</h2>
        <p>Studio controls the Firecracker runtime already installed on this host.</p>
      </div>
      <button class="button primary" @click="selectTab('convert')">Convert an image</button>
    </div>

    <div class="stat-grid">
      <div class="stat"><small>MicroVMs</small><strong>{{ data.metrics.microvms }}</strong><span>{{ data.metrics.runningVms }} running</span></div>
      <div class="stat"><small>Images</small><strong>{{ data.metrics.images }}</strong><span>local artifacts</span></div>
      <div class="stat"><small>Operations</small><strong>{{ data.metrics.operations }}</strong><span>conversion jobs</span></div>
      <div class="stat"><small>Runtime</small><strong>{{ runtimeReady ? 'Ready' : 'Check' }}</strong><span>{{ data.runtime.message || 'status unavailable' }}</span></div>
    </div>

    <div class="two-column">
      <article class="card">
        <div class="card-heading">
          <div><small class="overline">FIRST RUN</small><h3>Host readiness</h3></div>
          <span class="badge" :class="runtimeReady ? 'good' : 'warn'">{{ runtimeReady ? 'Ready' : 'Needs attention' }}</span>
        </div>
        <div class="check-list">
          <div><span>Firecracker</span><b>{{ data.runtime.installed ? 'Installed' : 'Missing' }}</b></div>
          <div><span>KVM</span><b>{{ data.runtime.kvm || 'Unknown' }}</b></div>
          <div><span>TAP networking</span><b>{{ data.runtime.tap || 'Unknown' }}</b></div>
          <div><span>Kernel</span><b>{{ data.runtime.kernel || 'Unknown' }}</b></div>
          <div><span>Rootfs</span><b>{{ data.runtime.rootfs || 'Unknown' }}</b></div>
        </div>
        <p class="hint">Install the runtime separately. Studio never hides a missing prerequisite.</p>
      </article>

      <article class="card">
        <div class="card-heading">
          <div><small class="overline">RECENT</small><h3>Workloads</h3></div>
          <button class="button subtle" @click="selectTab('vms')">View all</button>
        </div>
        <div v-if="data.vms.length" class="compact-list">
          <button
            v-for="vm in data.vms.slice(0, 4)" :key="vm.id" class="list-row"
            @click="state.selectedVM = vm.id; selectTab('vms')"
          >
            <span class="list-icon">▣</span>
            <span><strong>{{ vm.id.slice(0, 12) }}</strong><small>{{ vm.imageReference || vm.artifactDigest }}</small></span>
            <span class="badge" :class="vm.state">{{ vm.state }}</span>
          </button>
        </div>
        <div v-else class="empty">No workloads yet.</div>
      </article>
    </div>
  </section>
</template>
