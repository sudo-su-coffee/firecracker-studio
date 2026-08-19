<script setup>
import { useStudio } from '../composables/useStudio'

const { data } = useStudio()
</script>

<template>
  <section class="view">
    <div class="page-intro">
      <small class="overline">LOCAL ARTIFACTS</small>
      <h2>Images</h2>
      <p>Artifacts created by your conversions.</p>
    </div>

    <article class="card">
      <div v-if="data.images.length" class="compact-list">
        <div v-for="image in data.images" :key="image.digest" class="list-row static">
          <span class="list-icon">◈</span>
          <span><strong>{{ image.reference }}</strong><small class="mono">{{ image.digest }}</small></span>
          <span class="badge good">ready</span>
        </div>
      </div>
      <div v-else class="empty">No local images yet. Convert an image to create one.</div>
    </article>

    <article class="card">
      <div class="card-heading"><h3>Runtime assets</h3><span class="badge">{{ data.baseImages.length }}</span></div>
      <p class="hint">Firecracker runtime installation is separate from Studio. These entries describe host assets available to the server.</p>
      <div v-if="data.baseImages.length" class="compact-list">
        <div v-for="image in data.baseImages" :key="image.id" class="list-row static">
          <span class="list-icon">◇</span>
          <span><strong>{{ image.name || image.id }}</strong><small>{{ image.distribution || 'Firecracker' }} · {{ image.architecture || 'native' }}</small></span>
        </div>
      </div>
    </article>
  </section>
</template>
