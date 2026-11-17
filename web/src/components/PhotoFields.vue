<script setup lang="ts">
import { useId } from 'vue'
import type { FieldErrors } from '../api/client'
import FieldError from './FieldError.vue'
import type { PhotoFieldsValue } from './photoForm'

defineProps<{ errors: FieldErrors }>()

const fields = defineModel<PhotoFieldsValue>({ required: true })
const rangeHintId = useId()
</script>

<template>
  <label>
    Название
    <input v-model="fields.title" class="field" name="title" required maxlength="120"
      :aria-invalid="Boolean(errors.title)" placeholder="Например, площадь перед собором">
    <FieldError :message="errors.title" />
  </label>
  <label>
    Место
    <input v-model="fields.place" class="field" name="place" required maxlength="120"
      :aria-invalid="Boolean(errors.place)" placeholder="Например, собор Парижской Богоматери">
    <FieldError :message="errors.place" />
  </label>
  <label>
    Описание
    <textarea v-model="fields.description" class="field" name="description" required rows="3" maxlength="1000"
      :aria-invalid="Boolean(errors.description)" placeholder="Что изображено на снимке?"></textarea>
    <FieldError :message="errors.description" />
  </label>
  <div class="panel-field-grid">
    <label>
      Год съёмки
      <input v-model.number="fields.year" class="field" name="year" type="number" min="1" required
        :aria-invalid="Boolean(errors.year)" placeholder="1900">
      <FieldError :message="errors.year" />
    </label>
    <label>
      До какого года
      <input v-model.number="fields.endYear" class="field" name="endYear" type="number" min="1"
        :aria-invalid="Boolean(errors.endYear)" :aria-describedby="rangeHintId" placeholder="Если это диапазон">
      <FieldError :message="errors.endYear" />
    </label>
  </div>
  <p :id="rangeHintId" class="panel-hint">Можно указать один год или диапазон до 50 лет.</p>
  <details class="panel-source" :open="Boolean(fields.sourceName || fields.sourceUrl || errors.sourceName || errors.sourceUrl)">
    <summary>Источник снимка <span>(необязательно)</span></summary>
    <label>
      Название источника
      <input v-model="fields.sourceName" class="field" name="sourceName" maxlength="160" :aria-invalid="Boolean(errors.sourceName)">
      <FieldError :message="errors.sourceName" />
    </label>
    <label>
      Ссылка на источник
      <input v-model="fields.sourceUrl" class="field" name="sourceUrl" type="url" maxlength="2048"
        :aria-invalid="Boolean(errors.sourceUrl)" placeholder="https://">
      <FieldError :message="errors.sourceUrl" />
    </label>
  </details>
</template>
