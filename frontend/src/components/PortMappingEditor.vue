<script setup>
import { useStudio } from '../composables/useStudio'

const { state, addPortMapping, removePortMapping } = useStudio()
</script>

<template>
  <div class="port-editor">
    <label class="checkbox-row">
      <input v-model="state.enableNetworking" type="checkbox" />
      <span>Attach a network interface (TAP device on the host)</span>
    </label>

    <template v-if="state.enableNetworking">
      <div v-for="(mapping, index) in state.portMappings" :key="index" class="port-row">
        <label>Host port<input v-model="mapping.hostPort" type="number" min="1" max="65535" placeholder="15432" /></label>
        <span class="port-arrow">→</span>
        <label>Guest port<input v-model="mapping.guestPort" type="number" min="1" max="65535" placeholder="5432" /></label>
        <label>Protocol
          <select v-model="mapping.protocol">
            <option value="tcp">TCP</option>
            <option value="udp">UDP</option>
          </select>
        </label>
        <button type="button" class="button subtle icon-button" :disabled="state.portMappings.length === 1" @click="removePortMapping(index)" title="Remove port mapping">✕</button>
      </div>
      <button type="button" class="button subtle" @click="addPortMapping">+ Add port mapping</button>
    </template>
  </div>
</template>
