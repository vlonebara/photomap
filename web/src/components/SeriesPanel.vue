<script setup lang="ts">
import { Camera, ChevronRight, Flag, Image, MapPin, Maximize2, Star } from '@lucide/vue'
import { computed, nextTick, ref, watch } from 'vue'
import type { PublicPhoto, SeriesDetail, SeriesSummary } from '../api/types'
import { countLabel, photoDate } from '../format'
import PhotoViewer from './PhotoViewer.vue'

const props = defineProps<{
  series: SeriesSummary[]
  activeSeries: SeriesDetail | null
  photo: PublicPhoto | null
  unavailable: boolean
  loading: boolean
  error: string
  reactionBusy: boolean
}>()

const emit = defineEmits<{
  selectSeries: [id: number]
  selectPhoto: [id: number]
  star: []
  report: []
  retry: []
}>()

const imageFailed = ref(false)
const viewerOpen = ref(false)
const photoButton = ref<HTMLButtonElement | null>(null)
const nextSeries = computed(() => props.series.filter((entry) => entry.id !== props.photo?.seriesId))
const position = computed(() => (props.activeSeries?.photos.findIndex((item) => item.id === props.photo?.id) ?? 0) + 1)
const starred = computed(() => props.photo?.starredByMe ?? false)
const reported = computed(() => props.photo?.reportedByMe ?? false)

watch([() => props.photo?.id, () => props.photo?.imageUrl], () => {
  imageFailed.value = false
  viewerOpen.value = false
})

function closeViewer() {
  viewerOpen.value = false
  nextTick(() => photoButton.value?.focus({ preventScroll: true }))
}
</script>

<template>
  <aside class="explorer" aria-label="Снимки на карте" :aria-busy="loading">
    <div v-if="error" class="explorer-error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="button" :disabled="loading" @click="emit('retry')">Повторить</button>
    </div>

    <div v-if="unavailable" class="explorer-empty">
      <Image :size="27" aria-hidden="true" />
      <h1>Фото недоступно</h1>
      <p>Выберите другую серию на карте или в списке.</p>
    </div>

    <section v-else-if="photo" class="series-content">
      <div class="series-heading">
        <div v-if="activeSeries && activeSeries.place !== activeSeries.title" class="place-label">
          <MapPin :size="14" aria-hidden="true" />
          <span>{{ activeSeries.place }}</span>
        </div>
        <h1>{{ activeSeries?.title ?? photo.place }}</h1>
        <div class="series-stats">
          <span><Camera :size="15" aria-hidden="true" /> {{ countLabel(activeSeries?.photoCount ?? 1, ['снимок', 'снимка', 'снимков']) }}</span>
          <span><Star :size="15" aria-hidden="true" /> {{ countLabel(activeSeries?.stars ?? photo.stars, ['звезда', 'звезды', 'звёзд']) }}</span>
        </div>
      </div>

      <div class="photo-frame">
        <button
          v-if="!imageFailed"
          ref="photoButton"
          type="button"
          class="photo-open"
          :aria-label="`Открыть фото «${photo.title}» в полном размере`"
          aria-haspopup="dialog"
          @click="viewerOpen = true"
        >
          <img
            :key="photo.imageUrl"
            :src="photo.imageUrl"
            :alt="photo.title"
            referrerpolicy="no-referrer"
            @error="imageFailed = true"
          />
          <span class="photo-expand" aria-hidden="true"><Maximize2 :size="16" /></span>
        </button>
        <div v-else class="image-empty">
          <Image :size="23" aria-hidden="true" />
          <span>Снимок не загрузился</span>
        </div>
        <div class="photo-index">{{ position }} / {{ activeSeries?.photoCount ?? 1 }}</div>
      </div>
      <PhotoViewer v-if="viewerOpen" :image-url="photo.imageUrl" :title="photo.title" @close="closeViewer" />

      <div v-if="activeSeries && activeSeries.photos.length > 1" class="timeline" aria-label="Снимки по годам">
        <button
          v-for="item in activeSeries.photos"
          :key="item.id"
          class="timeline-year"
          type="button"
          :aria-pressed="item.id === photo.id"
          @click="emit('selectPhoto', item.id)"
        >
          {{ photoDate(item.year, item.endYear) }}
        </button>
      </div>

      <article class="photo-details">
        <div class="photo-kicker">
          <time :datetime="String(photo.year)">{{ photoDate(photo.year, photo.endYear) }}</time>
          <span>Фото №{{ photo.id }}</span>
        </div>
        <h2>{{ photo.title }}</h2>
        <p v-if="photo.description">{{ photo.description }}</p>
        <div class="photo-credit">
          <span>Добавил: {{ photo.authorName }}</span>
          <a v-if="photo.sourceUrl" :href="photo.sourceUrl" target="_blank" rel="noopener noreferrer">{{ photo.sourceName || 'Источник' }}</a>
          <span v-else-if="photo.sourceName">{{ photo.sourceName }}</span>
        </div>
        <div class="photo-actions">
          <button type="button" class="button star-button" :class="{ 'starred': starred }" :aria-pressed="starred"
            :aria-label="starred ? 'Снять звезду' : 'Поставить звезду'" :disabled="reactionBusy" @click="emit('star')">
            <Star :size="16" aria-hidden="true" :fill="starred ? 'currentColor' : 'none'" />
            {{ photo.stars }}
          </button>
          <button type="button" class="button report-button" :disabled="reported || reactionBusy" @click="emit('report')">
            <Flag :size="15" aria-hidden="true" />
            {{ reported ? 'Жалоба отправлена' : 'Пожаловаться' }}
          </button>
        </div>
      </article>
    </section>

    <div v-else-if="loading" class="explorer-empty" role="status">
      <p>Загружаем снимки…</p>
    </div>

    <div v-else-if="!error" class="explorer-empty">
      <Image :size="27" aria-hidden="true" />
      <h1>Снимки на карте</h1>
      <p>Здесь появятся фотографии после публикации.</p>
    </div>

    <section v-if="nextSeries.length" class="other-series" aria-labelledby="other-series-title">
      <h2 id="other-series-title">{{ activeSeries ? 'Другие серии' : 'Серии на карте' }}</h2>
      <ul>
        <li v-for="entry in nextSeries" :key="entry.id">
          <button class="series-link" type="button" @click="emit('selectSeries', entry.id)">
            <img :src="entry.thumbnailUrl" alt="" loading="lazy" referrerpolicy="no-referrer" />
            <span class="series-link-text">
              <strong>{{ entry.title }}</strong>
              <span>{{ countLabel(entry.photoCount, ['снимок', 'снимка', 'снимков']) }}</span>
            </span>
            <ChevronRight :size="16" aria-hidden="true" />
          </button>
        </li>
      </ul>
    </section>
  </aside>
</template>
