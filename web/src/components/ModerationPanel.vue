<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Coordinates, Photo } from '../demo/types'
import { useDemo } from '../demo/useDemo'
import PhotoFields from './PhotoFields.vue'
import { formError, photoFields, photoStatuses, toPhotoDraft } from './photoForm'

const emit = defineEmits<{ notify: [message: string] }>()
const { photos, moderate } = useDemo()
type Filter = 'queue' | 'published' | 'rejected'
const filter = ref<Filter>('queue')
const editedId = ref<number | null>(null)
const fields = ref(photoFields())
const location = ref<{ longitude: number | ''; latitude: number | '' }>({ longitude: '', latitude: '' })
const error = ref('')
const editedPhoto = computed(() => photos.value.find(photo => photo.id === editedId.value))

const categories = [
  { id: 'queue', label: 'На проверке' },
  { id: 'published', label: 'На карте' },
  { id: 'rejected', label: 'Не опубликовано' },
] as const
const items = computed(() => photos.value.filter(photo => filter.value === 'queue'
  ? photo.status === 'pending' || photo.status === 'review'
  : photo.status === filter.value))

function changeFilter(value: Filter) {
  filter.value = value
  error.value = ''
}

function edit(photo: Photo) {
  editedId.value = photo.id
  fields.value = photoFields(photo)
  location.value = { longitude: photo.coordinates[0], latitude: photo.coordinates[1] }
  error.value = ''
}

function closeEdit() {
  editedId.value = null
  error.value = ''
}

function save() {
  const photo = editedPhoto.value
  if (!photo || photo.status === 'removed') return
  try {
    const { longitude, latitude } = location.value
    if (longitude === '' || latitude === '') throw new Error('Укажите широту и долготу.')
    const coordinates: Coordinates = [longitude, latitude]
    const edits = toPhotoDraft(fields.value, coordinates, photo.imageUrl)
    moderate(photo.id, photo.status, edits)
    closeEdit()
    emit('notify', 'Изменения сохранены.')
  } catch (cause) {
    error.value = formError(cause)
  }
}

function decide(photo: Photo, status: 'published' | 'rejected') {
  error.value = ''
  try {
    moderate(photo.id, status)
    emit('notify', status === 'published' ? 'Фото доступно на карте.' : 'Фото скрыто с карты.')
  } catch (cause) {
    error.value = formError(cause)
  }
}
</script>

<template>
  <form v-if="editedPhoto && editedPhoto.status !== 'removed'" class="panel-form" @submit.prevent="save">
    <strong>Снимок №{{ editedPhoto.id }}</strong>
    <PhotoFields v-model="fields" />
    <div class="panel-field-grid">
      <label>
        Широта
        <input v-model.number="location.latitude" class="field" type="number" min="-90" max="90" step="any" required>
      </label>
      <label>
        Долгота
        <input v-model.number="location.longitude" class="field" type="number" min="-180" max="180" step="any" required>
      </label>
    </div>
    <p v-if="error" class="panel-error" role="alert">{{ error }}</p>
    <div class="panel-actions">
      <button type="button" class="button" @click="closeEdit">Отмена</button>
      <button type="submit" class="button button-primary">Сохранить</button>
    </div>
  </form>
  <template v-else>
    <div class="panel-filters" aria-label="Фильтр публикаций">
      <button v-for="category in categories" :key="category.id" type="button" class="button" :aria-pressed="category.id === filter" @click="changeFilter(category.id)">
        {{ category.label }}
      </button>
    </div>
    <p v-if="error" class="panel-error" role="alert">{{ error }}</p>
    <p v-if="!items.length" class="panel-empty">В этом разделе пока нет снимков.</p>
    <ul v-else class="panel-list">
      <li v-for="photo in items" :key="photo.id" class="panel-list-item">
        <img :src="photo.imageUrl" :alt="photo.title" loading="lazy">
        <div class="panel-photo-info">
          <strong>{{ photo.title }}</strong>
          <p class="panel-hint">№{{ photo.id }} · {{ photo.authorName }}</p>
          <p class="panel-status">{{ photoStatuses[photo.status] }}</p>
        </div>
        <div class="panel-row-actions">
          <button type="button" class="button" @click="edit(photo)">Изменить</button>
          <button v-if="photo.status !== 'published'" type="button" class="button button-primary" @click="decide(photo, 'published')">
            {{ photo.status === 'pending' ? 'Одобрить' : 'Восстановить' }}
          </button>
          <button v-if="photo.status !== 'rejected'" type="button" class="button" @click="decide(photo, 'rejected')">
            {{ photo.status === 'pending' ? 'Отклонить' : 'Снять с карты' }}
          </button>
        </div>
      </li>
    </ul>
  </template>
</template>
