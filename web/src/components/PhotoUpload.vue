<script setup lang="ts">
import { ImagePlus, MapPin } from '@lucide/vue'
import { ref } from 'vue'
import type { Coordinates, PhotoDraft } from '../demo/types'
import PhotoFields from './PhotoFields.vue'
import { formError, photoFields, toPhotoDraft } from './photoForm'

const props = defineProps<{ coordinates: Coordinates | null; submitError: string }>()
const emit = defineEmits<{
  choosePoint: []
  submit: [draft: PhotoDraft]
}>()

const fields = ref(photoFields())
const imageUrl = ref('')
const fileName = ref('')
const fileError = ref('')
const error = ref('')
const reading = ref(false)
let fileSelection = 0

async function selectFile(event: Event) {
  const file = (event.currentTarget as HTMLInputElement).files?.[0]
  const selection = ++fileSelection
  imageUrl.value = ''
  fileName.value = ''
  fileError.value = ''
  error.value = ''
  reading.value = false

  if (!file) return
  if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
    fileError.value = 'Выберите файл JPEG, PNG или WebP.'
    return
  }
  if (file.size > 1024 * 1024) {
    fileError.value = 'Для демо выберите файл размером до 1 МБ.'
    return
  }

  reading.value = true
  try {
    const reader = new FileReader()
    const result = await new Promise<string>((resolve, reject) => {
      reader.onload = () => typeof reader.result === 'string'
        ? resolve(reader.result)
        : reject(new Error('Файл не удалось прочитать.'))
      reader.onerror = () => reject(new Error('Файл не удалось прочитать.'))
      reader.readAsDataURL(file)
    })
    if (selection !== fileSelection) return
    const image = new Image()
    image.src = result
    try {
      await image.decode()
    } catch {
      throw new Error('Не удалось открыть изображение. Выберите другой файл.')
    }
    if (selection !== fileSelection) return
    imageUrl.value = result
    fileName.value = file.name
  } catch (cause) {
    if (selection === fileSelection) fileError.value = formError(cause)
  } finally {
    if (selection === fileSelection) reading.value = false
  }
}

function submit() {
  error.value = ''
  try {
    emit('submit', toPhotoDraft(fields.value, props.coordinates, imageUrl.value))
  } catch (cause) {
    error.value = formError(cause)
  }
}
</script>

<template>
  <form class="panel-form" @submit.prevent="submit">
    <div class="panel-location">
      <MapPin :size="19" aria-hidden="true" />
      <div>
        <strong>Объект на карте</strong>
        <p class="panel-hint">{{ coordinates ? `${coordinates[1].toFixed(5)}, ${coordinates[0].toFixed(5)}` : 'Точка ещё не выбрана' }}</p>
      </div>
      <button type="button" class="button" @click="emit('choosePoint')">
        {{ coordinates ? 'Изменить' : 'Выбрать' }}
      </button>
    </div>

    <div class="panel-upload">
      <img v-if="imageUrl" :src="imageUrl" alt="Предпросмотр выбранного снимка">
      <ImagePlus v-else :size="24" aria-hidden="true" />
      <label>
        {{ fileName || 'Изображение' }}
        <input class="field" type="file" accept="image/jpeg,image/png,image/webp" aria-describedby="photo-file-hint" @change="selectFile">
      </label>
    </div>
    <p id="photo-file-hint" class="panel-hint">JPEG, PNG или WebP до 1 МБ в демо.</p>
    <p v-if="fileError" class="panel-error" role="alert">{{ fileError }}</p>

    <PhotoFields v-model="fields" />

    <p v-if="error || submitError" class="panel-error" role="alert">{{ error || submitError }}</p>
    <div class="panel-actions">
      <button class="button button-primary" :disabled="reading" type="submit">Отправить на модерацию</button>
    </div>
  </form>
</template>
