<script setup lang="ts">
import { useId } from 'vue'
import type { PhotoFieldsValue } from './photoForm'

const fields = defineModel<PhotoFieldsValue>({ required: true })
const rangeHintId = useId()
const thisYear = new Date().getFullYear()
</script>

<template>
  <label>
    Название
    <input v-model="fields.title" class="field" name="title" required maxlength="120" placeholder="Например, площадь перед собором">
  </label>
  <label>
    Место
    <input v-model="fields.place" class="field" name="place" required maxlength="120" placeholder="Например, собор Парижской Богоматери">
  </label>
  <label>
    Описание
    <textarea v-model="fields.description" class="field" name="description" required rows="3" maxlength="1000" placeholder="Что изображено на снимке?"></textarea>
  </label>
  <div class="panel-field-grid">
    <label>
      Год съёмки
      <input v-model.number="fields.year" class="field" name="year" type="number" min="1" :max="thisYear" required placeholder="1900">
    </label>
    <label>
      До какого года
      <input v-model.number="fields.endYear" class="field" name="endYear" type="number" min="1" :max="thisYear" :aria-describedby="rangeHintId" placeholder="Если это диапазон">
    </label>
  </div>
  <p :id="rangeHintId" class="panel-hint">Можно указать один год или диапазон до 50 лет.</p>
  <details class="panel-source" :open="Boolean(fields.sourceName || fields.sourceUrl)">
    <summary>Источник снимка <span>(необязательно)</span></summary>
    <label>
      Название источника
      <input v-model="fields.sourceName" class="field" name="sourceName" maxlength="160">
    </label>
    <label>
      Ссылка на источник
      <input v-model="fields.sourceUrl" class="field" name="sourceUrl" type="url" maxlength="2048" placeholder="https://">
    </label>
  </details>
</template>
