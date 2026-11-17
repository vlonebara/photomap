<script setup lang="ts">
import { Check, CircleAlert, CircleX, MapPin, Plus, Search, ShieldCheck, UserRound } from '@lucide/vue'
import { onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import { errorMessage, isApiError } from './api/client'
import { photoMapApi } from './api/photomap'
import type { Coordinates, PublicPhoto, Viewer } from './api/types'
import AccountPanels from './components/AccountPanels.vue'
import MapCanvas from './components/MapCanvas.vue'
import SeriesPanel from './components/SeriesPanel.vue'
import { usePhotoMap } from './composables/usePhotoMap'

type Panel = 'auth' | 'upload' | 'account' | 'moderation'

const gallery = usePhotoMap()
const { series, selectedSeries, selectedPhoto, unavailable, loading, error, mapUpdate } = gallery
const viewer = shallowRef<Viewer | null>(null)
const sessionLoading = ref(true)
const reactionBusy = ref(false)
const panel = ref<Panel | null>(null)
const uploadKey = ref(0)
const query = ref('')
const searchError = ref('')
const picking = ref(false)
const point = ref<Coordinates | null>(null)
const focus = ref<{ coordinates: Coordinates; token: number } | null>(null)
const notice = ref('')
const noticeIsError = ref(false)

let focusToken = 0
let focusPhotoId: number | null = null
let noticeTimer: ReturnType<typeof setTimeout> | undefined
let sessionRequest: AbortController | null = null
let resumeUpload = false

function focusMap(coordinates: Coordinates) {
  focus.value = { coordinates, token: ++focusToken }
}

watch(selectedPhoto, (photo) => {
  if (photo && photo.id === focusPhotoId) {
    focusPhotoId = null
    focusMap(photo.coordinates)
  }
})

function notify(message: string, failed = false) {
  clearTimeout(noticeTimer)
  notice.value = message
  noticeIsError.value = failed
  noticeTimer = setTimeout(() => { notice.value = '' }, 3800)
}

function openPhoto(id: number, moveMap = false, fromSeries = false) {
  focusPhotoId = moveMap ? id : null
  void gallery.select(id, fromSeries)
}

function selectSeries(id: number, moveMap = false) {
  openPhoto(id, moveMap, true)
}

function parsePhotoId(value: string) {
  if (!/^\d+$/.test(value)) {
    return null
  }
  const id = Number(value)
  return Number.isSafeInteger(id) && id > 0 ? id : null
}

function searchPhoto() {
  const value = query.value.trim().replace(/^#\s*/, '')
  const id = parsePhotoId(value)
  if (id === null) {
    searchError.value = 'Введите номер снимка'
    return
  }

  searchError.value = ''
  openPhoto(id, true)
}

function resetExplore() {
  query.value = ''
  searchError.value = ''
  const [first] = series.value
  if (first) {
    selectSeries(first.id, true)
  } else {
    focusPhotoId = null
    gallery.clearSelection()
  }
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
  if (point.value) {
    panel.value = 'upload'
  }
}

function closePanel() {
  panel.value = null
  const startAfterSignin = resumeUpload && viewer.value
  resumeUpload = false
  point.value = null
  if (startAfterSignin) {
    startUpload()
  }
}

function setViewer(user: Viewer | null) {
  sessionRequest?.abort()
  viewer.value = user
  sessionLoading.value = false
}

watch(viewer, (user, previous) => {
  if (user?.id === previous?.id && user?.role === previous?.role) {
    return
  }

  gallery.refreshViewer()
  const privatePanel = panel.value === 'upload' || panel.value === 'account' || panel.value === 'moderation'
  if (previous && user?.id !== previous.id && (privatePanel || picking.value)) {
    picking.value = false
    uploadKey.value++
    closePanel()
  } else if (panel.value === 'moderation' && user?.role !== 'moderator') {
    closePanel()
  }
})

async function restoreSession() {
  sessionRequest?.abort()
  const request = new AbortController()
  sessionRequest = request

  try {
    const user = await photoMapApi.getViewer(request.signal)
    if (!request.signal.aborted) {
      viewer.value = user
    }
  } catch (error) {
    if (!request.signal.aborted && isApiError(error, 401)) {
      viewer.value = null
    }
  } finally {
    if (sessionRequest === request) {
      sessionLoading.value = false
      sessionRequest = null
    }
  }
}

function expireSession() {
  setViewer(null)
  picking.value = false
  point.value = null
  uploadKey.value++
  panel.value = 'auth'
  notify('Сессия закончилась. Войдите снова.', true)
}

async function onPublicPhoto(action: (photo: PublicPhoto) => Promise<void>) {
  if (reactionBusy.value) return
  if (!viewer.value) {
    panel.value = 'auth'
    return
  }

  const photo = selectedPhoto.value
  if (!photo) return

  reactionBusy.value = true
  try {
    await action(photo)
  } catch (error) {
    if (isApiError(error, 401)) {
      expireSession()
    } else {
      notify(errorMessage(error), true)
    }
  } finally {
    await gallery.refresh()
    reactionBusy.value = false
  }
}

function markStar() {
  return onPublicPhoto(async (photo) => {
    const result = await photoMapApi.setStar(photo.id, !photo.starredByMe)
    gallery.updateReaction(photo.id, result)
  })
}

function reportPhoto() {
  return onPublicPhoto(async (photo) => {
    if (photo.reportedByMe) return

    const result = await photoMapApi.report(photo.id)
    gallery.updateReaction(photo.id, result)
    notify('Жалоба отправлена')
  })
}

function readPhotoLink() {
  const match = /^#photo-(\d+)$/.exec(window.location.hash)
  const id = match ? parsePhotoId(match[1]) : null
  if (id !== null) {
    openPhoto(id, true)
  }
}

onMounted(() => {
  readPhotoLink()
  gallery.start()
  void restoreSession()
  window.addEventListener('focus', restoreSession)
  window.addEventListener('hashchange', readPhotoLink)
})

onUnmounted(() => {
  clearTimeout(noticeTimer)
  sessionRequest?.abort()
  window.removeEventListener('focus', restoreSession)
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
          :aria-label="viewer ? 'Личный кабинет' : 'Войти'" :disabled="sessionLoading" @click="panel = viewer ? 'account' : 'auth'">
          <UserRound :size="17" aria-hidden="true" />
          <span>{{ viewer ? 'Кабинет' : 'Войти' }}</span>
        </button>
        <button type="button" class="button button-primary add-button" aria-label="Добавить фото" :disabled="sessionLoading" @click="startUpload">
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
        :loading="loading"
        :error="error"
        :reaction-busy="reactionBusy || sessionLoading"
        @select-series="selectSeries($event, true)"
        @select-photo="openPhoto($event)"
        @star="markStar"
        @report="reportPhoto"
        @retry="gallery.refresh()"
      />
      <section class="map-pane" aria-label="Историческая карта">
        <MapCanvas
          :series="series"
          :active-id="selectedPhoto?.seriesId ?? null"
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
      :viewer="viewer"
      :map-update="mapUpdate"
      :upload-key="uploadKey"
      :coordinates="point"
      @close="closePanel"
      @choose-point="choosePoint"
      @notify="notify($event)"
      @open-photo="openPhoto($event, true)"
      @session="setViewer"
      @session-expired="expireSession"
      @changed="gallery.refresh()"
    />
    <div v-if="notice" class="toast" role="status">
      <CircleAlert v-if="noticeIsError" :size="18" aria-hidden="true" />
      <Check v-else :size="18" aria-hidden="true" />
      {{ notice }}
    </div>
  </div>
</template>
