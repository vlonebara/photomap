<script setup lang="ts">
import { Check, CircleX, MapPin, Plus, Search, ShieldCheck, UserRound } from '@lucide/vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import AccountPanels from './components/AccountPanels.vue'
import MapCanvas from './components/MapCanvas.vue'
import SeriesPanel from './components/SeriesPanel.vue'
import { useDemo } from './demo/useDemo'
import type { Coordinates } from './demo/types'

type Panel = 'auth' | 'upload' | 'account' | 'moderation'

const { viewer, series, toggleStar, hasStar, report, hasReported } = useDemo()
const panel = ref<Panel | null>(null)
const uploadKey = ref(0)
const query = ref('')
const searchError = ref('')
const picking = ref(false)
const point = ref<Coordinates | null>(null)
const focus = ref<{ coordinates: Coordinates; token: number } | null>(null)
const notice = ref('')
let focusToken = 0
let noticeTimer: ReturnType<typeof setTimeout> | undefined
let resumeUpload = false

const initialSeries = computed(() => series.value.find((entry) => /нотр-дам|собор парижской/i.test(entry.title)) ?? series.value[0])
const photoId = ref<number | null>(initialSeries.value?.id ?? null)
const selectedSeries = computed(() => {
  const id = photoId.value
  return id === null ? initialSeries.value ?? null : series.value.find((entry) => entry.photos.some((photo) => photo.id === id)) ?? null
})
const selectedPhoto = computed(() => {
  const photos = selectedSeries.value?.photos ?? []
  return (photoId.value === null ? photos.find((photo) => photo.id === selectedSeries.value?.id) ?? photos[0] : photos.find((photo) => photo.id === photoId.value)) ?? null
})
const unavailable = computed(() => photoId.value !== null && !selectedPhoto.value)

function focusMap(coordinates: Coordinates) {
  focus.value = { coordinates, token: ++focusToken }
}

function notify(message: string) {
  clearTimeout(noticeTimer)
  notice.value = message
  noticeTimer = setTimeout(() => { notice.value = '' }, 3800)
}

function openPhoto(id: number, moveMap = false) {
  photoId.value = id
  if (moveMap && selectedPhoto.value) {
    focusMap(selectedPhoto.value.coordinates)
  }
}

function selectSeries(id: number, moveMap = false) {
  const entry = series.value.find((item) => item.id === id)
  if (entry?.photos[0]) openPhoto(entry.id)
  if (entry && moveMap) focusMap(entry.coordinates)
}

function searchPhoto() {
  const value = query.value.trim().replace(/^#\s*/, '')
  const id = Number(value)
  if (!/^\d+$/.test(value) || !Number.isSafeInteger(id) || id < 1) {
    searchError.value = 'Введите номер снимка'
    return
  }

  searchError.value = ''
  openPhoto(id, true)
}

function resetExplore() {
  photoId.value = initialSeries.value?.id ?? null
  query.value = ''
  searchError.value = ''
  if (initialSeries.value) focusMap(initialSeries.value.coordinates)
}

function choosePoint() {
  panel.value = null
  picking.value = true
}

function startUpload() {
  if (!viewer.value) {
    resumeUpload = true
    panel.value = 'auth'
    return
  }
  uploadKey.value++
  point.value = null
  choosePoint()
}

function pickPoint(coordinates: Coordinates) {
  if (!picking.value) return
  point.value = coordinates
  picking.value = false
  panel.value = 'upload'
}

function cancelPicking() {
  picking.value = false
  if (point.value) panel.value = 'upload'
}

function closePanel() {
  panel.value = null
  const startAfterSignin = resumeUpload && viewer.value
  resumeUpload = false
  point.value = null
  if (startAfterSignin) startUpload()
}

function onPublicPhoto(action: (id: number) => void) {
  if (!viewer.value) {
    panel.value = 'auth'
    return
  }
  if (!selectedPhoto.value) return
  try {
    action(selectedPhoto.value.id)
  } catch (error) {
    notify(error instanceof Error ? error.message : 'Повторите действие')
  }
}

function markStar() {
  onPublicPhoto(toggleStar)
}

function reportPhoto() {
  onPublicPhoto((id) => {
    if (hasReported(id)) return
    report(id)
    notify('Жалоба отправлена')
  })
}

function readPhotoLink() {
  const match = /^#photo-(\d+)$/.exec(location.hash)
  if (match) openPhoto(Number(match[1]), true)
}

onMounted(() => {
  readPhotoLink()
  window.addEventListener('hashchange', readPhotoLink)
})
onUnmounted(() => {
  clearTimeout(noticeTimer)
  window.removeEventListener('hashchange', readPhotoLink)
})
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <button class="brand" type="button" aria-label="PhotoMap, вернуться к обзору" @click="resetExplore">
        PhotoMap
      </button>

      <div class="search-box">
        <form class="search-form" role="search" @submit.prevent="searchPhoto">
          <Search :size="18" aria-hidden="true" />
          <input
            v-model="query"
            type="search"
            inputmode="numeric"
            placeholder="Поиск по ID фото"
            aria-label="Поиск по ID фото"
            :aria-invalid="Boolean(searchError)"
            :aria-describedby="searchError ? 'search-error' : undefined"
            @input="searchError = ''"
          />
          <button type="submit" class="search-submit">Найти</button>
        </form>
        <p v-if="searchError" id="search-error" class="search-error" role="alert">{{ searchError }}</p>
      </div>

      <div class="header-actions">
        <button v-if="viewer?.role === 'moderator'" type="button" class="button button-icon"
          aria-label="Модерация" title="Модерация" @click="panel = 'moderation'">
          <ShieldCheck :size="19" aria-hidden="true" />
        </button>
        <button type="button" class="button login-button"
          :aria-label="viewer ? 'Личный кабинет' : 'Войти'" @click="panel = viewer ? 'account' : 'auth'">
          <UserRound :size="17" aria-hidden="true" />
          <span>{{ viewer ? 'Кабинет' : 'Войти' }}</span>
        </button>
        <button type="button" class="button button-primary add-button" aria-label="Добавить фото" @click="startUpload">
          <Plus :size="19" aria-hidden="true" />
          <span>Добавить фото</span>
        </button>
      </div>
    </header>

    <main class="workspace">
      <SeriesPanel
        :series="series"
        :active-series="selectedSeries"
        :photo="selectedPhoto"
        :unavailable="unavailable"
        :starred="Boolean(selectedPhoto && hasStar(selectedPhoto.id))"
        :reported="Boolean(selectedPhoto && hasReported(selectedPhoto.id))"
        @select-series="selectSeries($event, true)"
        @select-photo="openPhoto($event)"
        @star="markStar"
        @report="reportPhoto"
      />
      <section class="map-pane" aria-label="Историческая карта">
        <MapCanvas
          :series="series"
          :active-id="selectedSeries?.id ?? null"
          :picking="picking"
          :point="point"
          :focus="focus"
          @select="selectSeries($event)"
          @pick="pickPoint"
        />
        <div v-if="picking" class="pick-notice" role="status">
          <MapPin :size="18" aria-hidden="true" />
          <span>Укажите на карте объект в кадре</span>
          <button class="button button-icon" type="button" aria-label="Отменить выбор точки" @click="cancelPicking">
            <CircleX :size="18" aria-hidden="true" />
          </button>
        </div>
      </section>
    </main>

    <AccountPanels
      :panel="panel"
      :upload-key="uploadKey"
      :coordinates="point"
      @close="closePanel"
      @choose-point="choosePoint"
      @notify="notify"
      @open-photo="openPhoto($event, true)"
    />
    <div v-if="notice" class="toast" role="status"><Check :size="18" aria-hidden="true" /> {{ notice }}</div>
  </div>
</template>
