import { computed, onUnmounted, ref, shallowRef } from 'vue'
import { errorMessage, isApiError } from '../api/client'
import { mapEventsUrl, photoMapApi } from '../api/photomap'
import type { PublicPhoto, SeriesDetail, SeriesSummary } from '../api/types'

async function availableSeries(id: number, signal: AbortSignal) {
  try {
    return await photoMapApi.getSeries(id, signal)
  } catch (error) {
    if (isApiError(error, 404)) {
      return null
    }
    throw error
  }
}

function containsPhoto(series: SeriesDetail | null, photo: PublicPhoto) {
  return series?.id === photo.seriesId && series.photos.some(item => item.id === photo.id)
}

// resolve the current series again if its center moved between requests
async function readSelection(id: number, fromSeries: boolean, signal: AbortSignal) {
  let series = fromSeries ? await availableSeries(id, signal) : null
  let photo = await photoMapApi.getPhoto(id, signal)

  if (!containsPhoto(series, photo)) {
    series = await availableSeries(photo.seriesId, signal)
  }

  if (!containsPhoto(series, photo)) {
    photo = await photoMapApi.getPhoto(id, signal)
    series = await availableSeries(photo.seriesId, signal)
  }

  return { photo, series: containsPhoto(series, photo) ? series : null }
}

export function usePhotoMap() {
  const series = shallowRef<SeriesSummary[]>([])
  const selectedSeries = shallowRef<SeriesDetail | null>(null)
  const selectedPhoto = shallowRef<PublicPhoto | null>(null)
  const unavailable = ref(false)
  const loadingMap = ref(true)
  const loadingSelection = ref(false)
  const mapError = ref('')
  const selectionError = ref('')
  const loading = computed(() => loadingMap.value || loadingSelection.value)
  const error = computed(() => selectionError.value || mapError.value)
  const mapUpdate = ref(0)
  const lifetime = new AbortController()

  let selectedId: number | null = null
  let selectionRequest: AbortController | null = null
  let selectionQueued = false
  let mapRefresh: Promise<void> | null = null
  let mapRefreshQueued = false
  let events: EventSource | undefined
  let eventRetryTimer: number | undefined

  // let a click take priority over background refreshes
  async function loadSelection(id: number, fromSeries: boolean) {
    selectionRequest?.abort()
    const request = new AbortController()
    selectionRequest = request
    loadingSelection.value = true

    try {
      const next = await readSelection(id, fromSeries, request.signal)
      if (request.signal.aborted) {
        return null
      }

      selectedSeries.value = next.series
      selectedPhoto.value = next.photo
      selectionError.value = ''
      unavailable.value = false
      return next.photo
    } catch (error) {
      if (request.signal.aborted) {
        return null
      }

      if (isApiError(error, 404)) {
        selectedSeries.value = null
        selectedPhoto.value = null
        unavailable.value = true
        selectionError.value = ''
      } else {
        selectionError.value = errorMessage(error)
      }
      return null
    } finally {
      if (selectionRequest === request) {
        selectionRequest = null
        loadingSelection.value = false
        if (selectionQueued) {
          selectionQueued = false
          void refreshSelection()
        }
      }
    }
  }

  function select(id: number, fromSeries = false) {
    selectedId = id
    selectionQueued = false
    selectionError.value = ''
    unavailable.value = false
    if (selectedPhoto.value?.id !== id) {
      selectedPhoto.value = null
      selectedSeries.value = null
    }
    return loadSelection(id, fromSeries)
  }

  function clearSelection() {
    selectedId = null
    selectionQueued = false
    selectionRequest?.abort()
    selectedPhoto.value = null
    selectedSeries.value = null
    unavailable.value = false
    selectionError.value = ''
  }

  function refreshSelection() {
    if (selectedId === null) {
      return
    }
    if (selectionRequest) {
      selectionQueued = true
      return
    }
    return loadSelection(selectedId, false)
  }

  async function loadMap() {
    try {
      const items = await photoMapApi.listSeries(lifetime.signal)
      if (lifetime.signal.aborted) {
        return
      }
      series.value = items
      mapError.value = ''
      if (selectedId === null && items[0]) {
        await select(items[0].id, true)
      }
    } catch (error) {
      if (!lifetime.signal.aborted) {
        mapError.value = errorMessage(error)
      }
    } finally {
      loadingMap.value = false
    }
  }

  // combine bursts of events without missing a refresh requested during a fetch
  async function reloadMap() {
    while (mapRefreshQueued && !lifetime.signal.aborted) {
      mapRefreshQueued = false
      await Promise.all([loadMap(), refreshSelection()])
    }
  }

  function refresh() {
    mapRefreshQueued = true
    mapRefresh ??= reloadMap().finally(() => { mapRefresh = null })
    return mapRefresh
  }

  function updateReaction(id: number, changes: Partial<Pick<PublicPhoto, 'stars' | 'starredByMe' | 'reportedByMe'>>) {
    if (selectedPhoto.value?.id === id) {
      selectedPhoto.value = { ...selectedPhoto.value, ...changes }
    }
  }

  function refreshViewer() {
    if (selectedId === null) {
      return
    }
    updateReaction(selectedId, { starredByMe: undefined, reportedByMe: undefined })
    void loadSelection(selectedId, false)
  }

  function connectEvents() {
    const source = new EventSource(mapEventsUrl, { withCredentials: true })
    events = source
    source.addEventListener('map.updated', () => {
      mapUpdate.value++
      void refresh()
    })
    source.addEventListener('error', () => {
      // the browser may stop reconnecting after an HTTP error from the proxy
      if (events === source && source.readyState === EventSource.CLOSED && !lifetime.signal.aborted) {
        window.clearTimeout(eventRetryTimer)
        eventRetryTimer = window.setTimeout(connectEvents, 3000)
      }
    })
  }

  function start() {
    connectEvents()
    void refresh()
  }

  onUnmounted(() => {
    lifetime.abort()
    window.clearTimeout(eventRetryTimer)
    events?.close()
    selectionQueued = false
    selectionRequest?.abort()
  })

  return {
    series,
    selectedSeries,
    selectedPhoto,
    unavailable,
    loading,
    error,
    mapUpdate,
    select,
    clearSelection,
    refresh,
    updateReaction,
    refreshViewer,
    start,
  }
}
