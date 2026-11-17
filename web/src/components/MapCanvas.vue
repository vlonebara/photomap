<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Crosshair, Layers, LocateFixed, Minus, Plus } from '@lucide/vue'
import { LngLatBounds, Map as MapLibreMap, Marker, ScaleControl } from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import type { Coordinates, SeriesSummary } from '../api/types'

const props = defineProps<{
  series: SeriesSummary[]
  activeId: number | null
  picking: boolean
  point: Coordinates | null
  focus: { coordinates: Coordinates; token: number } | null
}>()

const emit = defineEmits<{
  select: [id: number]
  pick: [coordinates: Coordinates]
}>()

const styleUrl = 'https://tiles.openfreemap.org/styles/positron'
const worldCenter: Coordinates = [0, 20]
const worldZoom = 1.4
const minZoom = 0
const maxZoom = 20
const container = ref<HTMLDivElement | null>(null)
const status = ref<'loading' | 'ready' | 'failed' | 'unavailable'>('loading')
const zoom = ref(worldZoom)
const initialized = ref(false)

let map: MapLibreMap | undefined
let resizeObserver: ResizeObserver | undefined
let loadTimer: number | undefined
let pointMarker: Marker | undefined
let mapFramed = false
const markers = new Map<number, Marker>()

function defaultZoom() {
  return (container.value?.clientWidth ?? 0) < 560 ? 13.65 : 14.2
}

function makeMarker(item: SeriesSummary, currentMap: MapLibreMap) {
  const button = document.createElement('button')
  button.type = 'button'
  button.className = 'series-marker'
  const badge = document.createElement('span')
  badge.className = 'series-marker__count'
  badge.setAttribute('aria-hidden', 'true')
  button.append(badge)
  button.addEventListener('click', (event) => {
    event.stopPropagation()
    if (props.picking) {
      const selected = props.series.find((series) => series.id === item.id)
      if (selected) emit('pick', selected.coordinates)
      return
    }
    emit('select', item.id)
  })
  return new Marker({ element: button }).setLngLat(item.coordinates).addTo(currentMap)
}

function renderMarkers() {
  if (!map) return
  const visibleIds = new Set(props.series.map(({ id }) => id))
  for (const [id, marker] of markers) {
    if (visibleIds.has(id)) continue
    marker.remove()
    markers.delete(id)
  }
  for (const item of props.series) {
    const marker = markers.get(item.id) ?? makeMarker(item, map)
    marker.setLngLat(item.coordinates)
    const button = marker.getElement()
    const selected = !props.picking && item.id === props.activeId
    const size = Math.min(44, 28 + Math.sqrt(Math.max(item.stars, 0)) * 1.9)
    button.style.setProperty('--marker-size', `${size}px`)
    button.style.zIndex = selected ? '2' : '1'
    button.classList.toggle('series-marker--active', selected)
    button.setAttribute('aria-pressed', String(selected))
    button.setAttribute(
      'aria-label',
      `${props.picking ? 'Выбрать точку' : 'Открыть серию'} «${item.title}», снимков: ${item.photoCount}`,
    )
    button.title = item.title
    const badge = button.firstElementChild
    if (badge) badge.textContent = item.photoCount > 99 ? '99+' : String(item.photoCount)
    markers.set(item.id, marker)
  }
}

function renderPoint() {
  if (!map || !props.point) {
    pointMarker?.remove()
    pointMarker = undefined
    return
  }
  if (!pointMarker) {
    const dot = document.createElement('div')
    dot.className = 'map-picked-point'
    dot.setAttribute('aria-hidden', 'true')
    pointMarker = new Marker({ element: dot }).setLngLat(props.point).addTo(map)
  } else {
    pointMarker.setLngLat(props.point)
  }
}

function focusPhoto() {
  if (!map || !props.focus) return
  map.easeTo({
    center: props.focus.coordinates,
    zoom: Math.max(map.getZoom(), 14.8),
    duration: 360,
  })
}

function pickCenter() {
  if (!map) return
  const { lng, lat } = map.getCenter().wrap()
  emit('pick', [lng, lat])
}

function frameSeries(duration: number) {
  if (!map) return

  const [first] = props.series
  if (!first) {
    map.easeTo({ center: worldCenter, zoom: worldZoom, duration })
    return
  }

  if (props.series.length === 1) {
    map.easeTo({ center: first.coordinates, zoom: defaultZoom(), duration })
    return
  }

  const bounds = new LngLatBounds(first.coordinates, first.coordinates)
  for (const item of props.series) {
    bounds.extend(item.coordinates)
  }
  map.fitBounds(bounds, { padding: 56, maxZoom: defaultZoom(), duration })
}

function resetView() {
  frameSeries(360)
}

function frameInitialSeries() {
  if (!map || mapFramed || !props.series.length) return
  mapFramed = true

  if (!props.focus && !props.picking) {
    frameSeries(0)
  }
}

function zoomIn() {
  map?.zoomIn({ duration: 220 })
}

function zoomOut() {
  map?.zoomOut({ duration: 220 })
}

function disposeMap() {
  window.clearTimeout(loadTimer)
  resizeObserver?.disconnect()
  pointMarker?.remove()
  pointMarker = undefined
  for (const marker of markers.values()) marker.remove()
  markers.clear()
  map?.remove()
  map = undefined
  mapFramed = false
  initialized.value = false
}

function initMap() {
  if (!container.value) return
  disposeMap()
  status.value = 'loading'

  let currentMap: MapLibreMap
  try {
    currentMap = new MapLibreMap({
      container: container.value,
      style: styleUrl,
      center: worldCenter,
      zoom: worldZoom,
      minZoom,
      maxZoom,
      dragRotate: false,
      pitchWithRotate: false,
      attributionControl: {},
      locale: {
        'Map.Title': 'Интерактивная карта',
        'Marker.Title': 'Метка на карте',
        'AttributionControl.ToggleAttribution': 'Источники карты',
        'AttributionControl.MapFeedback': 'Сообщить о проблеме с картой',
        'ScaleControl.Meters': 'м',
        'ScaleControl.Kilometers': 'км',
      },
    })
  } catch {
    status.value = 'unavailable'
    return
  }

  map = currentMap
  initialized.value = true
  zoom.value = currentMap.getZoom()
  currentMap.touchZoomRotate.disableRotation()
  currentMap.addControl(new ScaleControl({ unit: 'metric', maxWidth: 85 }), 'bottom-left')

  currentMap.on('zoom', () => { zoom.value = currentMap.getZoom() })
  currentMap.on('click', ({ lngLat }) => {
    if (!props.picking) return
    const { lng, lat } = lngLat.wrap()
    emit('pick', [lng, lat])
  })
  currentMap.on('error', () => {
    if (status.value === 'loading' && !currentMap.getStyle()?.version) {
      window.clearTimeout(loadTimer)
      status.value = 'failed'
    }
  })
  currentMap.once('load', () => {
    window.clearTimeout(loadTimer)
    status.value = 'ready'
  })
  loadTimer = window.setTimeout(() => {
    if (status.value === 'loading') status.value = 'failed'
  }, 12_000)

  resizeObserver = new ResizeObserver(() => currentMap.resize())
  resizeObserver.observe(container.value)
  renderMarkers()
  renderPoint()
  frameInitialSeries()
  focusPhoto()
}

watch(() => props.series, frameInitialSeries)
watch(() => [props.series, props.activeId, props.picking], renderMarkers)
watch(() => props.point, renderPoint)
watch(() => props.focus?.token, focusPhoto)
onMounted(initMap)
onBeforeUnmount(disposeMap)
</script>

<template>
  <section class="map-canvas" :class="{ 'map-canvas--picking': picking }" aria-label="Карта исторических фотографий">
    <div ref="container" class="map-surface"></div>
    <div v-if="picking && status === 'ready'" class="map-crosshair" aria-hidden="true">
      <Crosshair :size="23" />
    </div>

    <div v-if="status !== 'ready'" class="map-message" role="status">
      <Layers :size="18" aria-hidden="true" />
      <span v-if="status === 'loading'">Загружаем карту…</span>
      <span v-else-if="status === 'unavailable'">Для карты нужен браузер с поддержкой WebGL.</span>
      <span v-else>Не удалось загрузить карту.</span>
      <button v-if="status === 'failed'" type="button" @click="initMap">Повторить</button>
    </div>

    <div v-if="initialized" class="map-controls" role="group" aria-label="Управление картой">
      <button
        v-if="picking && status === 'ready'"
        class="map-control map-control--single"
        type="button"
        title="Выбрать центр карты"
        aria-label="Выбрать центр карты"
        @click="pickCenter"
      >
        <Crosshair :size="19" aria-hidden="true" />
      </button>
      <div class="map-controls__zoom">
        <button
          class="map-control"
          type="button"
          title="Приблизить"
          aria-label="Приблизить"
          :disabled="zoom >= maxZoom"
          @click="zoomIn"
        >
          <Plus :size="20" aria-hidden="true" />
        </button>
        <button
          class="map-control"
          type="button"
          title="Отдалить"
          aria-label="Отдалить"
          :disabled="zoom <= minZoom"
          @click="zoomOut"
        >
          <Minus :size="20" aria-hidden="true" />
        </button>
      </div>
      <button
        class="map-control map-control--single"
        type="button"
        title="К исходному виду"
        aria-label="К исходному виду"
        @click="resetView"
      >
        <LocateFixed :size="18" aria-hidden="true" />
      </button>
    </div>
  </section>
</template>

<style scoped>
.map-canvas {
  position: relative;
  isolation: isolate;
  width: 100%;
  height: 100%;
  min-height: 240px;
  background: var(--background);
  color: var(--ink);
}

.map-surface {
  position: absolute;
  inset: 0;
}

.map-canvas--picking :deep(.maplibregl-canvas-container.maplibregl-interactive) {
  cursor: crosshair;
}

.map-canvas :deep(.series-marker) {
  display: grid;
  width: 44px;
  height: 44px;
  padding: 0;
  place-items: center;
  border: 0;
  border-radius: 50%;
  background: transparent;
  cursor: pointer;
}

.map-canvas :deep(.series-marker__count) {
  display: grid;
  width: var(--marker-size);
  height: var(--marker-size);
  place-items: center;
  border: 2px solid #fff;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 1px 5px #18282640;
  color: #fff;
  font: 650 12px/1 system-ui, sans-serif;
  font-variant-numeric: tabular-nums;
}

.map-canvas :deep(.series-marker:hover .series-marker__count),
.map-canvas :deep(.series-marker--active .series-marker__count) {
  box-shadow: 0 0 0 5px color-mix(in srgb, var(--accent) 22%, transparent), 0 2px 5px #18282640;
}

.map-canvas :deep(.series-marker--active .series-marker__count) {
  background: var(--accent-hover);
}

.map-canvas :deep(.series-marker:focus-visible),
.map-control:focus-visible,
.map-message button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.map-canvas :deep(.map-picked-point) {
  z-index: 3;
  width: 18px;
  height: 18px;
  border: 3px solid #fff;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 5px color-mix(in srgb, var(--accent) 22%, transparent);
  pointer-events: none;
}

.map-crosshair {
  position: absolute;
  z-index: 3;
  top: 50%;
  left: 50%;
  color: var(--accent);
  filter: drop-shadow(0 0 2px #fff);
  pointer-events: none;
  transform: translate(-50%, -50%);
}

.map-controls {
  position: absolute;
  z-index: 4;
  right: 16px;
  bottom: 46px;
  display: grid;
  gap: 8px;
}

.map-controls__zoom,
.map-control--single {
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 9px;
  background: #fff;
  box-shadow: 0 2px 8px #18282610;
}

.map-controls__zoom:focus-within {
  overflow: visible;
}

.map-control {
  display: grid;
  width: 42px;
  height: 42px;
  padding: 0;
  place-items: center;
  background-color: #fff;
  color: var(--ink);
  cursor: pointer;
}

.map-controls__zoom .map-control:first-child {
  border-radius: 8px 8px 0 0;
}

.map-controls__zoom .map-control:last-child {
  border-top: 1px solid var(--line);
  border-radius: 0 0 8px 8px;
}

.map-control:hover:not(:disabled) {
  background: var(--accent-soft);
  color: var(--accent);
}

.map-control:disabled {
  color: var(--muted);
  cursor: default;
}

.map-message {
  position: absolute;
  z-index: 5;
  top: 50%;
  left: 50%;
  display: flex;
  width: max-content;
  max-width: calc(100% - 42px);
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 2px 8px #18282610;
  font-size: 13px;
  transform: translate(-50%, -50%);
}

.map-message svg {
  flex-shrink: 0;
  color: var(--accent);
}

.map-message button {
  color: var(--accent);
  font-weight: 650;
  text-decoration: underline;
  text-underline-offset: 3px;
  cursor: pointer;
}

.map-surface :deep(.maplibregl-ctrl-scale) {
  background: #ffffffe8;
  color: var(--ink);
  border-color: var(--ink);
}

@media (max-width: 560px) {
  .map-surface :deep(.maplibregl-ctrl-bottom-left) { bottom: 25px; }

  .map-controls {
    right: 12px;
    bottom: 38px;
  }

  .map-control {
    width: 40px;
    height: 40px;
  }
}
</style>
