<script setup lang="ts">
import { ref } from 'vue'
import { isApiError } from '../api/client'
import { photoMapApi } from '../api/photomap'
import type { Viewer } from '../api/types'
import { useFormFeedback } from '../composables/useFormFeedback'
import FieldError from './FieldError.vue'

type AuthMode = 'signin' | 'register'

const mode = defineModel<AuthMode>('mode', { required: true })
const emit = defineEmits<{
  authenticated: [viewer: Viewer, registered: boolean]
  busy: [value: boolean]
}>()

const name = ref('')
const email = ref('')
const password = ref('')
const busy = ref(false)
const { message, errors, clearFeedback, showFeedback } = useFormFeedback()

function setMode(value: AuthMode) {
  if (mode.value === value) return

  mode.value = value
  password.value = ''
  clearFeedback()
}

async function authenticate() {
  if (busy.value) return

  clearFeedback()
  busy.value = true
  emit('busy', true)
  const registered = mode.value === 'register'
  const credentials = { email: email.value.trim(), password: password.value }
  let viewer: Viewer | null = null
  let checkingSession = false

  try {
    if (registered) {
      await photoMapApi.register({ ...credentials, name: name.value.trim() })
    } else {
      await photoMapApi.login(credentials)
    }

    checkingSession = true
    viewer = await photoMapApi.getViewer()
  } catch (error) {
    if (checkingSession && isApiError(error, 401)) {
      showFeedback(new Error('Сессия не сохранилась. Проверьте, что браузер разрешает cookies для PhotoMap.'))
    } else {
      showFeedback(error)
    }
  } finally {
    busy.value = false
    emit('busy', false)
  }

  if (viewer) {
    password.value = ''
    emit('authenticated', viewer, registered)
  }
}
</script>

<template>
  <div class="panel-filters" aria-label="Вход или регистрация">
    <button class="button" type="button" :disabled="busy" :aria-pressed="mode === 'signin'" @click="setMode('signin')">Войти</button>
    <button class="button" type="button" :disabled="busy" :aria-pressed="mode === 'register'" @click="setMode('register')">Регистрация</button>
  </div>

  <form class="panel-auth-form" @submit.prevent="authenticate">
    <fieldset class="panel-form" :disabled="busy">
      <label v-if="mode === 'register'">
        Имя
        <input v-model="name" class="field" name="name" maxlength="60" autocomplete="name" required :aria-invalid="Boolean(errors.name)">
        <FieldError :message="errors.name" />
      </label>
      <label>
        Email
        <input v-model="email" class="field" name="email" type="email" autocomplete="username" required :aria-invalid="Boolean(errors.email)">
        <FieldError :message="errors.email" />
      </label>
      <label>
        Пароль
        <input v-model="password" class="field" name="password" type="password" required
          :autocomplete="mode === 'register' ? 'new-password' : 'current-password'" :aria-invalid="Boolean(errors.password)">
        <span v-if="mode === 'register'" class="panel-hint">Минимум 8 символов.</span>
        <FieldError :message="errors.password" />
      </label>
      <FieldError :message="message" />
      <div class="panel-actions">
        <button class="button button-primary" type="submit">
          {{ mode === 'signin' ? 'Войти' : 'Создать профиль' }}
        </button>
      </div>
    </fieldset>
  </form>
</template>
