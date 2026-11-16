<script setup lang="ts">
import { Camera, ChevronRight, Flag, Image, MapPin, Star } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import type { Photo, Series } from '../demo/types'
import { countLabel, photoDate } from '../format'

const props = defineProps<{
  series: Series[]
  activeSeries: Series | null
  photo: Photo | null
  unavailable: boolean
  starred: boolean
  reported: boolean
}>()

const emit = defineEmits<{
  selectSeries: [id: number]
  selectPhoto: [id: number]
  star: []
  report: []
}>()

const imageFailed = ref(false)
const nextSeries = computed(() => props.series.filter((entry) => entry.id !== props.activeSeries?.id))
const position = computed(() => (props.activeSeries?.photos.findIndex((item) => item.id === props.photo?.id) ?? -1) + 1)
watch(() => props.photo?.imageUrl, () => { imageFailed.value = false })
</script>

<template>
  <aside class="explorer" aria-label="Снимки на карте">
    <div v-if="unavailable" class="explorer-empty">
      <Image :size="27" aria-hidden="true" />
      <h1>Фото недоступно</h1>
      <p>Выберите другую серию на карте или в списке.</p>
    </div>

    <section v-else-if="activeSeries && photo" class="series-content">
      <div class="series-heading">
        <div v-if="activeSeries.place && activeSeries.place !== activeSeries.title" class="place-label">
          <MapPin :size="14" aria-hidden="true" />
          <span>{{ activeSeries.place }}</span>
        </div>
        <h1>{{ activeSeries.title }}</h1>
        <div class="series-stats">
          <span><Camera :size="15" aria-hidden="true" /> {{ countLabel(activeSeries.photos.length, ['снимок', 'снимка', 'снимков']) }}</span>
          <span><Star :size="15" aria-hidden="true" /> {{ countLabel(activeSeries.stars, ['звезда', 'звезды', 'звёзд']) }}</span>
        </div>
      </div>

      <div class="photo-frame">
        <img v-if="!imageFailed"
          :key="photo.imageUrl"
          :src="photo.imageUrl"
          :alt="photo.title"
          referrerpolicy="no-referrer"
          @error="imageFailed = true"
        />
        <div v-else class="image-empty">
          <Image :size="23" aria-hidden="true" />
          <span>Снимок не загрузился</span>
        </div>
        <div class="photo-index">{{ position }} / {{ activeSeries.photos.length }}</div>
      </div>

      <div v-if="activeSeries.photos.length > 1" class="timeline" aria-label="Снимки по годам">
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
            :aria-label="starred ? 'Снять звезду' : 'Поставить звезду'" @click="emit('star')">
            <Star :size="16" aria-hidden="true" :fill="starred ? 'currentColor' : 'none'" />
            {{ photo.stars }}
          </button>
          <button type="button" class="button report-button" :disabled="reported" @click="emit('report')">
            <Flag :size="15" aria-hidden="true" />
            {{ reported ? 'Жалоба отправлена' : 'Пожаловаться' }}
          </button>
        </div>
      </article>
    </section>

    <div v-else class="explorer-empty">
      <Image :size="27" aria-hidden="true" />
      <h1>Снимки на карте</h1>
      <p>Здесь появятся фотографии после публикации.</p>
    </div>

    <section v-if="nextSeries.length" class="other-series" aria-labelledby="other-series-title">
      <h2 id="other-series-title">{{ activeSeries ? 'Другие серии' : 'Серии на карте' }}</h2>
      <ul>
        <li v-for="entry in nextSeries" :key="entry.id">
          <button class="series-link" type="button" @click="emit('selectSeries', entry.id)">
            <img v-if="entry.photos[0]" :src="entry.photos[0].imageUrl" alt="" loading="lazy" referrerpolicy="no-referrer" />
            <span class="series-link-text">
              <strong>{{ entry.title }}</strong>
              <span>{{ countLabel(entry.photos.length, ['снимок', 'снимка', 'снимков']) }}</span>
            </span>
            <ChevronRight :size="16" aria-hidden="true" />
          </button>
        </li>
      </ul>
    </section>
  </aside>
</template>
