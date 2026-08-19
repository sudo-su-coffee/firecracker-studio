<script setup>
import { useStudio } from '../composables/useStudio'
import PortMappingEditor from '../components/PortMappingEditor.vue'

const { state, busy, latestOperation, convert, createVM } = useStudio()
</script>

<template>
  <section class="view narrow">
    <div class="page-intro">
      <small class="overline">IMAGE WORKFLOW</small>
      <h2>Convert an image</h2>
      <p>Turn an OCI image into a local Firecracker artifact.</p>
    </div>

    <article class="card form-card">
      <label>Image reference<input v-model="state.source" placeholder="alpine:latest" /></label>
      <div class="form-grid">
        <label>Source
          <select v-model="state.sourceType">
            <option value="oci">OCI / Docker image</option>
            <option value="dockerfile">Dockerfile</option>
            <option value="archive">OCI archive</option>
          </select>
        </label>
        <label>Base profile
          <select v-model="state.baseProfile">
            <option>alpine</option>
            <option>debian</option>
            <option>ubuntu</option>
          </select>
        </label>
      </div>
      <button class="button primary wide" :disabled="busy || !state.source" @click="convert">
        {{ busy ? 'Converting…' : 'Start conversion' }}
      </button>
    </article>

    <article class="card">
      <div class="card-heading">
        <div><small class="overline">BUILD HISTORY</small><h3>Operations</h3></div>
        <span v-if="latestOperation" class="badge" :class="latestOperation.state">{{ latestOperation.state }}</span>
      </div>
      <div v-if="latestOperation" class="operation-detail">
        <div class="detail-line"><span>Source</span><strong>{{ latestOperation.request.source }}</strong></div>
        <div class="detail-line"><span>Digest</span><strong class="mono">{{ latestOperation.artifact?.digest || 'pending' }}</strong></div>
        <pre>{{ (latestOperation.logs || []).join('\n') }}{{ latestOperation.error ? '\n' + latestOperation.error : '' }}</pre>
      </div>
      <div v-else class="empty">No conversion operations yet.</div>
    </article>

    <article v-if="latestOperation?.state === 'succeeded'" class="card form-card">
      <div class="card-heading"><div><small class="overline">DEPLOY</small><h3>Workload configuration</h3></div></div>
      <div class="form-grid">
        <label>vCPUs<input v-model="state.vcpus" type="number" min="1" max="32" /></label>
        <label>Memory (MiB)<input v-model="state.memoryMiB" type="number" min="128" step="128" max="262144" /></label>
      </div>
      <PortMappingEditor />
      <button class="button primary wide" :disabled="busy" @click="createVM">{{ busy ? 'Creating…' : 'Create workload' }}</button>
    </article>
  </section>
</template>
