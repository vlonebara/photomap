<script setup lang="ts">
import { ref, shallowRef, watch } from 'vue'
import { isApiError } from '../api/client'
import { photoMapApi } from '../api/photomap'
import type { Coordinates, ManagedPhoto, ModerationAction, ModerationFilter, PhotoStatus } from '../api/types'
import { useFormFeedback } from '../composables/useFormFeedback'
import { usePhotoList } from '../composables/usePhotoList'
import FieldError from './FieldError.vue'
import PhotoFields from './PhotoFields.vue'
import PhotoPreview from './PhotoPreview.vue'
import { photoChanges, photoFields, photoStatuses, toPhotoInput } from './photoForm'

const props = defineProps<{ mapUpdate: number }>()
const emit = defineEmits<{
  notify: [message: string]
  sessionExpired: []
  changed: []
  busy: [value: boolean]
}>()

const filter = ref<ModerationFilter>('queue')
const { items, loading, error, reload } = usePhotoList(
  (signal) => photoMapApi.listModerationPhotos(filter.value, signal),
  () => emit('sessionExpired'),
)
const editedPhoto = shallowRef<ManagedPhoto | null>(null)
const fields = ref(photoFields())
const location = ref<{ longitude: number | ''; latitude: number | '' }>({ longitude: '', latitude: '' })
const busy = ref(false)
const { message, errors, clearFeedback, showFeedback } = useFormFeedback()

const categories = [
  { id: 'queue', label: 'На проверке' },
  { id: 'published', label: 'На карте' },
  { id: 'rejected', label: 'Не опубликовано' },
] as const

watch([filter, () => props.mapUpdate], reload, { immediate: true })

function changeFilter(value: ModerationFilter) {
  if (filter.value === value) return

  items.value = []
  filter.value = value
  clearFeedback()
}

async function perform(action: () => Promise<void>) {
  if (busy.value) return

  clearFeedback()
  busy.value = true
  emit('busy', true)
  try {
    await action()
  } catch (error) {
    if (isApiError(error, 401)) {
      emit('sessionExpired')
    } else {
      showFeedback(error)
      if (isApiError(error, 404) || isApiError(error, 409)) {
        void reload()
      }
    }
  } finally {
    busy.value = false
    emit('busy', false)
  }
}

function edit(id: number) {
  return perform(async () => {
    const photo = await photoMapApi.getModerationPhoto(id)
    editedPhoto.value = photo
    fields.value = photoFields(photo)
    const [longitude, latitude] = photo.coordinates
    location.value = { longitude, latitude }
  })
}

function closeEdit() {
  editedPhoto.value = null
  clearFeedback()
}

function save() {
  const photo = editedPhoto.value
  if (!photo) return

  return perform(async () => {
    const { longitude, latitude } = location.value
    if (longitude === '' || latitude === '') {
      throw new Error('Укажите широту и долготу.')
    }

    const coordinates: Coordinates = [longitude, latitude]
    const input = toPhotoInput(fields.value, coordinates)
    const changes = photoChanges(input, photo)
    if (Object.keys(changes).length === 0) {
      closeEdit()
      return
    }

    await photoMapApi.editPhoto(photo.id, changes)
    closeEdit()
    void reload()
    emit('changed')
    emit('notify', 'Изменения сохранены.')
  })
}

function positiveAction(status: PhotoStatus): ModerationAction {
  return status === 'pending' ? 'approve' : 'restore'
}

function negativeAction(status: PhotoStatus): ModerationAction {
  return status === 'published' ? 'unpublish' : 'reject'
}

function decide(id: number, action: ModerationAction) {
  return perform(async () => {
    const photo = await photoMapApi.moderatePhoto(id, action)
    items.value = items.value.filter(item => item.id !== id)
    void reload()
    emit('changed')
    if (photo.status === 'published') {
      emit('notify', 'Фото доступно на карте.')
    } else {
      emit('notify', action === 'unpublish' ? 'Фото снято с карты.' : 'Фото отклонено.')
    }
  })
}
</script>

<template>
  <form v-if="editedPhoto" @submit.prevent="save">
    <fieldset class="panel-form" :disabled="busy">
      <strong>Снимок №{{ editedPhoto.id }}</strong>
      <PhotoFields v-model="fields" :errors="errors" />
      <div class="panel-field-grid">
        <label>
          Широта
          <input v-model.number="location.latitude" name="latitude" class="field" type="number" min="-90" max="90" step="any"
            required :aria-invalid="Boolean(errors.latitude)">
          <FieldError :message="errors.latitude" />
        </label>
        <label>
          Долгота
          <input v-model.number="location.longitude" name="longitude" class="field" type="number" min="-180" max="180" step="any"
            required :aria-invalid="Boolean(errors.longitude)">
          <FieldError :message="errors.longitude" />
        </label>
      </div>
      <FieldError :message="message" />
      <div class="panel-actions">
        <button type="button" class="button" @click="closeEdit">Отмена</button>
        <button type="submit" class="button button-primary">Сохранить</button>
      </div>
    </fieldset>
  </form>
  <template v-else>
    <div class="panel-filters" aria-label="Фильтр публикаций">
      <button v-for="category in categories" :key="category.id" type="button" class="button"
        :aria-pressed="category.id === filter" :disabled="busy" @click="changeFilter(category.id)">
        {{ category.label }}
      </button>
    </div>
    <FieldError :message="message" />
    <div v-if="error" class="panel-list-error">
      <p class="panel-error" role="alert">{{ error }}</p>
      <button type="button" class="button" :disabled="busy || loading" @click="reload">Повторить</button>
    </div>
    <p v-if="!items.length && loading" class="panel-empty" role="status">Загружаем снимки…</p>
    <p v-else-if="!items.length && !error" class="panel-empty">В этом разделе пока нет снимков.</p>
    <ul v-else-if="items.length" class="panel-list" :aria-busy="loading">
      <li v-for="photo in items" :key="photo.id" class="panel-list-item">
        <PhotoPreview :photo="photo" />
        <div class="panel-photo-info">
          <strong>{{ photo.title }}</strong>
          <p class="panel-hint">№{{ photo.id }} · {{ photo.authorName }}</p>
          <p class="panel-status">{{ photoStatuses[photo.status] }}</p>
        </div>
        <div class="panel-row-actions">
          <button type="button" class="button" :disabled="busy" @click="edit(photo.id)">Изменить</button>
          <button v-if="photo.status !== 'published'" type="button" class="button button-primary" :disabled="busy"
            @click="decide(photo.id, positiveAction(photo.status))">
            {{ photo.status === 'pending' ? 'Одобрить' : 'Восстановить' }}
          </button>
          <button v-if="photo.status !== 'rejected'" type="button" class="button" :disabled="busy"
            @click="decide(photo.id, negativeAction(photo.status))">
            {{ photo.status === 'published' ? 'Снять с карты' : 'Отклонить' }}
          </button>
        </div>
      </li>
    </ul>
  </template>
</template>
