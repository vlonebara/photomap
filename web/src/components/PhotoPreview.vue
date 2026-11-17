<script setup lang="ts">
import { ref } from 'vue'
import type { Photo } from '../api/types'
import PhotoViewer from './PhotoViewer.vue'

defineProps<{ photo: Photo }>()

const viewerOpen = ref(false)
</script>

<template>
  <button
    type="button"
    class="panel-preview"
    :aria-label="`Открыть фото «${photo.title}» в полном размере`"
    aria-haspopup="dialog"
    @click="viewerOpen = true"
  >
    <img :src="photo.thumbnailUrl" :alt="photo.title" loading="lazy">
  </button>
  <PhotoViewer v-if="viewerOpen" :image-url="photo.imageUrl" :title="photo.title" @close="viewerOpen = false" />
</template>

<style scoped>
.panel-preview {
  overflow: hidden;
  width: 100%;
  height: 54px;
  border-radius: 5px;
  background: var(--background);
}

.panel-preview img { width: 100%; height: 100%; object-fit: cover; }

@media (max-width: 480px) {
  .panel-preview { height: 45px; }
}
</style>
