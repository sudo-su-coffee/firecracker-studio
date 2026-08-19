<script setup>
import { useStudio } from '../composables/useStudio'

const { state, data, busy, selectedVM, selectTab, vmAction, deleteVM } = useStudio()
</script>

<template>
  <section class="view">
    <div class="page-intro row">
      <div><small class="overline">LOCAL WORKLOADS</small><h2>Workloads</h2><p>Start, stop, inspect, or remove your microVMs.</p></div>
      <button class="button primary" @click="selectTab('convert')">Create workload</button>
    </div>

    <div class="workload-layout">
      <article class="card workload-list">
        <div v-if="data.vms.length" class="compact-list">
          <button
            v-for="vm in data.vms" :key="vm.id" class="list-row"
            :class="{ selected: selectedVM?.id === vm.id }"
            @click="state.selectedVM = vm.id"
          >
            <span class="list-icon">▣</span>
            <span><strong>{{ vm.id.slice(0, 12) }}</strong><small>{{ vm.imageReference || vm.artifactDigest }}</small></span>
            <span class="badge" :class="vm.state">{{ vm.state }}</span>
          </button>
        </div>
        <div v-else class="empty">No workloads yet. Convert an image to begin.</div>
      </article>

      <article v-if="selectedVM" class="detail-stack">
        <div class="card detail-header">
          <div>
            <small class="overline">MICROVM</small>
            <h3>{{ selectedVM.id.slice(0, 12) }}</h3>
            <p>{{ selectedVM.state }} · updated {{ new Date(selectedVM.updatedAt).toLocaleString() }}</p>
          </div>
          <div class="action-row">
            <button class="button primary" :disabled="busy || selectedVM.state === 'running'" @click="vmAction(selectedVM.id, 'start')">Start</button>
            <button class="button subtle" :disabled="busy || selectedVM.state !== 'running'" @click="vmAction(selectedVM.id, 'stop')">Stop</button>
            <button class="button danger" :disabled="busy" @click="deleteVM(selectedVM.id)">Delete</button>
          </div>
        </div>

        <div class="card">
          <div class="card-heading"><h3>Details</h3><span class="badge" :class="selectedVM.state">{{ selectedVM.state }}</span></div>
          <div class="detail-grid">
            <div><small>Image</small><strong>{{ selectedVM.imageReference || 'Unnamed image' }}</strong></div>
            <div><small>Artifact</small><strong class="mono">{{ selectedVM.artifactDigest }}</strong></div>
            <div><small>Published ports</small><strong>{{ selectedVM.portMappings?.length || 0 }}</strong></div>
          </div>
        </div>

        <div class="card">
          <div class="card-heading">
            <div><small class="overline">NETWORKING</small><h3>TAP device &amp; port forwards</h3></div>
            <span class="badge" :class="selectedVM.network ? 'good' : 'warn'">{{ selectedVM.network ? 'Attached' : 'No network' }}</span>
          </div>
          <div v-if="selectedVM.network" class="check-list">
            <div><span>TAP device</span><b class="mono">{{ selectedVM.network.tapDevice }}</b></div>
            <div><span>Guest address</span><b class="mono">{{ selectedVM.network.guestCidr }}</b></div>
            <div><span>Host address</span><b class="mono">{{ selectedVM.network.hostCidr }}</b></div>
            <div><span>Guest MAC</span><b class="mono">{{ selectedVM.network.guestMac }}</b></div>
          </div>
          <div v-else class="empty">This workload was created without networking enabled.</div>
          <div v-if="selectedVM.portMappings?.length" class="compact-list" style="margin-top:11px">
            <div v-for="(port, index) in selectedVM.portMappings" :key="index" class="list-row static">
              <span class="list-icon">↔</span>
              <span><strong>localhost:{{ port.hostPort }}</strong><small>forwards to guest port {{ port.guestPort }} · {{ port.protocol || 'tcp' }}</small></span>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="card-heading"><h3>Studio lifecycle events</h3></div>
          <pre class="logs">{{ selectedVM.logs?.join('\n') || 'No lifecycle events yet.' }}</pre>
        </div>
      </article>
      <div v-else class="card empty large">Select a workload to inspect it.</div>
    </div>
  </section>
</template>
