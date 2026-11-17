<script setup lang="ts">
import { ImagePlus, MapPin } from '@lucide/vue'
import { onUnmounted, ref, shallowRef } from 'vue'
import { isApiError } from '../api/client'
import { photoMapApi } from '../api/photomap'
import type { Coordinates } from '../api/types'
import { useFormFeedback } from '../composables/useFormFeedback'
import FieldError from './FieldError.vue'
import PhotoFields from './PhotoFields.vue'
import { photoFields, toPhotoInput } from './photoForm'

const props = defineProps<{ coordinates: Coordinates | null }>()
const emit = defineEmits<{
  choosePoint: []
  uploaded: []
  sessionExpired: []
  busy: [value: boolean]
}>()

const maxImageBytes = 10 * 1024 * 1024
const maxImagePixels = 40_000_000
const fields = ref(photoFields())
const selectedFile = shallowRef<File | null>(null)
const previewUrl = ref('')
const fileError = ref('')
const reading = ref(false)
const busy = ref(false)
const { message, errors, clearFeedback, showFeedback } = useFormFeedback()
let fileSelection = 0

function releasePreview() {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }
}

async function selectFile(event: Event) {
  const file = (event.currentTarget as HTMLInputElement).files?.[0]
  const selection = ++fileSelection
  releasePreview()
  selectedFile.value = null
  fileError.value = ''
  reading.value = false
  clearFeedback()

  if (!file) return
  if (file.type && !['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
    fileError.value = 'Выберите файл JPEG, PNG или WebP.'
    return
  }
  if (file.size > maxImageBytes) {
    fileError.value = 'Выберите файл размером до 10 МиБ.'
    return
  }

  reading.value = true
  const url = URL.createObjectURL(file)
  try {
    const image = new Image()
    image.src = url
    await image.decode()
    if (selection !== fileSelection) return

    if (image.naturalWidth * image.naturalHeight > maxImagePixels) {
      fileError.value = 'Изображение должно содержать не больше 40 миллионов пикселей.'
      return
    }

    selectedFile.value = file
    previewUrl.value = url
  } catch {
    if (selection === fileSelection) {
      fileError.value = 'Не удалось открыть изображение. Выберите другой файл.'
    }
  } finally {
    if (url !== previewUrl.value) {
      URL.revokeObjectURL(url)
    }
    if (selection === fileSelection) {
      reading.value = false
    }
  }
}

async function submit() {
  if (busy.value || reading.value) return
  clearFeedback()

  const file = selectedFile.value
  if (!file) {
    if (!fileError.value) {
      fileError.value = 'Прикрепите фотографию.'
    }
    return
  }

  busy.value = true
  emit('busy', true)
  let uploaded = false
  try {
    const draft = toPhotoInput(fields.value, props.coordinates)
    await photoMapApi.upload(draft, file)
    uploaded = true
  } catch (error) {
    if (isApiError(error, 401)) {
      emit('sessionExpired')
    } else {
      showFeedback(error)
    }
  } finally {
    busy.value = false
    emit('busy', false)
  }

  if (uploaded) {
    emit('uploaded')
  }
}

onUnmounted(() => {
  fileSelection++
  releasePreview()
})
</script>

<template>
  <form @submit.prevent="submit">
    <fieldset class="panel-form" :disabled="busy">
      <div class="panel-location">
        <MapPin :size="19" aria-hidden="true" />
        <div>
          <strong>Объект на карте</strong>
          <p class="panel-hint">{{ coordinates ? `${coordinates[1].toFixed(5)}, ${coordinates[0].toFixed(5)}` : 'Точка ещё не выбрана' }}</p>
          <FieldError :message="errors.longitude || errors.latitude" />
        </div>
        <button type="button" class="button" @click="emit('choosePoint')">
          {{ coordinates ? 'Изменить' : 'Выбрать' }}
        </button>
      </div>

      <div class="panel-upload">
        <img v-if="previewUrl" :src="previewUrl" alt="Предпросмотр выбранного снимка">
        <ImagePlus v-else :size="24" aria-hidden="true" />
        <label>
          {{ selectedFile?.name || 'Изображение' }}
          <input class="field" type="file" accept="image/jpeg,image/png,image/webp" aria-describedby="photo-file-hint"
            :aria-invalid="Boolean(fileError || errors.image)" @change="selectFile">
        </label>
      </div>
      <p id="photo-file-hint" class="panel-hint">JPEG, PNG или WebP до 10 МиБ.</p>
      <FieldError :message="fileError || errors.image" />

      <PhotoFields v-model="fields" :errors="errors" />

      <FieldError :message="message" />
      <div class="panel-actions">
        <button class="button button-primary" :disabled="reading" type="submit">{{ busy ? 'Отправляем…' : 'Отправить на модерацию' }}</button>
      </div>
    </fieldset>
  </form>
</template>
