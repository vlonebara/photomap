<script setup lang="ts">
import { X } from '@lucide/vue'
import { onMounted, ref, useId } from 'vue'

defineProps<{
  imageUrl: string
  title: string
}>()

const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement | null>(null)
const titleId = useId()
const imageFailed = ref(false)

function close() {
  dialog.value?.close()
}

onMounted(() => dialog.value?.showModal())
</script>

<template>
  <dialog
    ref="dialog"
    class="photo-viewer"
    :aria-labelledby="titleId"
    @click.self="close"
    @close="emit('close')"
  >
    <header class="viewer-header">
      <h2 :id="titleId">{{ title }}</h2>
      <button class="button button-icon viewer-close" type="button" aria-label="Закрыть фото" autofocus @click="close">
        <X :size="20" aria-hidden="true" />
      </button>
    </header>

    <div class="viewer-image">
      <img v-if="!imageFailed" :src="imageUrl" :alt="title" referrerpolicy="no-referrer" @error="imageFailed = true" />
      <p v-else role="status">Снимок не загрузился</p>
    </div>
  </dialog>
</template>

<style scoped>
.photo-viewer {
  position: fixed;
  inset: 0;
  width: min(calc(100% - 24px), 1440px);
  max-width: none;
  height: calc(100dvh - 24px);
  max-height: none;
  margin: auto;
  padding: 0;
  overflow: hidden;
  border-radius: 12px;
  background: var(--ink);
  color: var(--surface);
}

.photo-viewer[open] {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
}

.photo-viewer::backdrop { background: rgb(0 0 0 / 70%); }

.viewer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px 10px 16px;
  font-size: .9rem;
  font-weight: 650;
}

.viewer-header h2 { overflow-wrap: anywhere; }

.viewer-close {
  border-color: transparent;
  background: rgb(255 255 255 / 15%);
}

.viewer-close:not(:disabled):hover { background: rgb(255 255 255 / 25%); }
.viewer-close:focus-visible { outline-color: var(--surface); }

.viewer-image {
  display: grid;
  grid-template: minmax(0, 1fr) / minmax(0, 1fr);
  place-items: center;
  min-width: 0;
  min-height: 0;
  padding: 0 12px 12px;
}

.viewer-image img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: scale-down;
}

:global(body:has(.photo-viewer[open])) { overflow: hidden; }
</style>
