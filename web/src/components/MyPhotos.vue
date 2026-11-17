<script setup lang="ts">
import { Camera, LogOut, UserRound } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { errorMessage, isApiError } from '../api/client'
import { photoMapApi } from '../api/photomap'
import type { Viewer } from '../api/types'
import { usePhotoList } from '../composables/usePhotoList'
import PhotoPreview from './PhotoPreview.vue'
import { photoStatuses } from './photoForm'

const props = defineProps<{ viewer: Viewer; mapUpdate: number }>()
const emit = defineEmits<{
  signedOut: []
  sessionExpired: []
  changed: []
  busy: [value: boolean]
  notify: [message: string]
  openPhoto: [id: number]
}>()

const { items, loading, error, reload } = usePhotoList(photoMapApi.listOwnPhotos, () => emit('sessionExpired'))
const published = computed(() => items.value.filter(photo => photo.status === 'published'))
const stars = computed(() => published.value.reduce((total, photo) => total + photo.stars, 0))
const deleteId = ref<number | null>(null)
const actionError = ref('')
const busy = ref(false)

watch(() => props.mapUpdate, reload, { immediate: true })

async function perform(action: () => Promise<void>) {
  if (busy.value) return false

  actionError.value = ''
  busy.value = true
  emit('busy', true)
  try {
    await action()
    return true
  } catch (error) {
    if (isApiError(error, 401)) {
      emit('sessionExpired')
    } else {
      actionError.value = errorMessage(error)
      if (isApiError(error, 404)) {
        void reload()
      }
    }
    return false
  } finally {
    busy.value = false
    emit('busy', false)
  }
}

async function leave() {
  const signedOut = await perform(async () => {
    try {
      await photoMapApi.logout()
    } catch (error) {
      if (!isApiError(error, 401)) {
        throw error
      }
    }
  })
  if (signedOut) {
    emit('signedOut')
  }
}

async function remove(id: number) {
  const removed = await perform(() => photoMapApi.removeOwnPhoto(id))
  if (removed) {
    deleteId.value = null
    items.value = items.value.filter(photo => photo.id !== id)
    void reload()
    emit('changed')
    emit('notify', 'Публикация удалена.')
  }
}
</script>

<template>
  <div class="panel-profile">
    <div class="panel-avatar" aria-hidden="true"><UserRound :size="22" /></div>
    <div class="panel-photo-info">
      <strong>{{ viewer.name }}</strong>
      <p v-if="items.length || (!loading && !error)" class="panel-hint">
        Фото на карте: {{ published.length }} · Звёзды: {{ stars }}
      </p>
    </div>
    <button type="button" class="button" :disabled="busy" @click="leave">
      <LogOut :size="16" aria-hidden="true" /> Выйти
    </button>
  </div>

  <section aria-labelledby="my-photos-title" :aria-busy="loading">
    <h3 id="my-photos-title" class="panel-subtitle">Мои публикации</h3>
    <p v-if="actionError" class="panel-error" role="alert">{{ actionError }}</p>
    <div v-if="error" class="panel-list-error">
      <p class="panel-error" role="alert">{{ error }}</p>
      <button type="button" class="button" :disabled="loading || busy" @click="reload">Повторить</button>
    </div>
    <p v-if="!items.length && loading" class="panel-empty" role="status">Загружаем публикации…</p>
    <p v-else-if="!items.length && !error" class="panel-empty">
      <Camera :size="22" aria-hidden="true" /> У вас пока нет публикаций.
    </p>
    <ul v-else-if="items.length" class="panel-list">
      <li v-for="photo in items" :key="photo.id" class="panel-list-item">
        <PhotoPreview :photo="photo" />
        <div class="panel-photo-info">
          <strong>{{ photo.title }}</strong>
          <p class="panel-hint">№{{ photo.id }}</p>
          <p class="panel-status">{{ photoStatuses[photo.status] }}</p>
        </div>
        <div v-if="deleteId === photo.id" class="panel-row-actions">
          <span>Удалить фото?</span>
          <button type="button" class="button" :disabled="busy" @click="deleteId = null">Отмена</button>
          <button type="button" class="button button-primary" :disabled="busy" @click="remove(photo.id)">Удалить</button>
        </div>
        <div v-else class="panel-row-actions">
          <button v-if="photo.status === 'published'" type="button" class="button" :disabled="busy" @click="emit('openPhoto', photo.id)">Открыть</button>
          <button type="button" class="button" :disabled="busy" @click="deleteId = photo.id">Удалить</button>
        </div>
      </li>
    </ul>
  </section>
</template>
