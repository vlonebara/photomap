<script setup lang="ts">
import { Camera, LogOut, UserRound, X } from '@lucide/vue'
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import type { Coordinates, PhotoDraft, Role } from '../demo/types'
import { useDemo } from '../demo/useDemo'
import ModerationPanel from './ModerationPanel.vue'
import PhotoUpload from './PhotoUpload.vue'
import { formError, photoStatuses } from './photoForm'

type Panel = 'auth' | 'upload' | 'account' | 'moderation'

const props = defineProps<{
  panel: Panel | null
  coordinates: Coordinates | null
  uploadKey: number
}>()
const emit = defineEmits<{
  close: []
  choosePoint: []
  notify: [message: string]
  openPhoto: [id: number]
}>()

const { viewer, photos, signin, signout, addPhoto, removePhoto } = useDemo()
const dialog = ref<HTMLDialogElement | null>(null)
const authMode = ref<'signin' | 'register'>('signin')
const name = ref('')
const role = ref<Role>('user')
const error = ref('')
const deleteId = ref<number | null>(null)
const uploadLoaded = ref(false)
const myPhotos = computed(() => photos.value
  .filter(photo => photo.authorId === viewer.value?.id && photo.status !== 'removed')
  .sort((a, b) => b.id - a.id))
const published = computed(() => myPhotos.value.filter(photo => photo.status === 'published'))
const stars = computed(() => published.value.reduce((total, photo) => total + photo.stars, 0))
const titles: Record<Panel, string> = {
  auth: 'Войти',
  upload: 'Добавить фото',
  account: 'Личный кабинет',
  moderation: 'Модерация',
}
const heading = computed(() => props.panel === 'auth' && authMode.value === 'register'
  ? 'Регистрация'
  : props.panel ? titles[props.panel] : '')

function setAuthMode(mode: 'signin' | 'register') {
  authMode.value = mode
  role.value = 'user'
  error.value = ''
}

function syncDialog() {
  if (props.panel && !dialog.value?.open) dialog.value?.showModal()
  if (!props.panel && dialog.value?.open) dialog.value.close()
}

onMounted(syncDialog)
watch(() => props.panel, async (panel) => {
  error.value = ''
  deleteId.value = null
  if (panel === 'auth') setAuthMode('signin')
  if (panel === 'upload') uploadLoaded.value = true
  await nextTick()
  syncDialog()
}, { immediate: true })

function close() {
  if (props.panel === 'upload') uploadLoaded.value = false
  emit('close')
}

function authenticate() {
  error.value = ''
  try {
    if (!name.value.trim()) throw new Error('Введите имя.')
    signin(name.value.trim(), role.value)
    name.value = ''
    emit('notify', authMode.value === 'register' ? 'Профиль создан.' : 'Вы вошли в аккаунт.')
    close()
  } catch (cause) {
    error.value = formError(cause)
  }
}

function leave() {
  signout()
  close()
  emit('notify', 'Вы вышли из аккаунта.')
}

function upload(draft: PhotoDraft) {
  error.value = ''
  try {
    addPhoto(draft)
    close()
    emit('notify', 'Снимок отправлен на модерацию.')
  } catch (cause) {
    error.value = formError(cause)
  }
}

function remove(id: number) {
  error.value = ''
  try {
    removePhoto(id)
    deleteId.value = null
    emit('notify', 'Публикация удалена.')
  } catch (cause) {
    error.value = formError(cause)
  }
}

function openPhoto(id: number) {
  close()
  emit('openPhoto', id)
}
</script>

<template>
  <dialog
    ref="dialog"
    class="account-panel"
    :class="{ 'account-panel-wide': panel === 'moderation' || panel === 'account' }"
    aria-labelledby="account-panel-title"
    @cancel.prevent="close"
    @click.self="close"
  >
    <div class="panel-layout">
      <header class="panel-header">
        <h2 id="account-panel-title">{{ heading }}</h2>
        <button class="button button-icon panel-close" type="button" aria-label="Закрыть" @click="close">
          <X :size="20" aria-hidden="true" />
        </button>
      </header>

      <div class="panel-content">
        <template v-if="panel === 'auth'">
          <div class="panel-filters" aria-label="Вход или регистрация">
            <button class="button" type="button" :aria-pressed="authMode === 'signin'" @click="setAuthMode('signin')">Войти</button>
            <button class="button" type="button" :aria-pressed="authMode === 'register'" @click="setAuthMode('register')">Регистрация</button>
          </div>
          <p class="panel-hint panel-auth-hint">Для входа в демо достаточно имени.</p>
          <form class="panel-form" @submit.prevent="authenticate">
            <label>
              Имя
              <input v-model="name" class="field" name="displayName" maxlength="60" autocomplete="nickname" placeholder="Как к вам обращаться?" required>
            </label>
            <label v-if="authMode === 'signin'">
              Роль в демо
              <select v-model="role" class="field">
                <option value="user">Пользователь</option>
                <option value="moderator">Модератор</option>
              </select>
            </label>
            <p v-if="error" class="panel-error" role="alert">{{ error }}</p>
            <div class="panel-actions">
              <button class="button button-primary" type="submit">
                {{ authMode === 'signin' ? 'Войти' : 'Создать профиль' }}
              </button>
            </div>
          </form>
        </template>

        <template v-else-if="panel === 'account' && viewer">
          <div class="panel-profile">
            <div class="panel-avatar" aria-hidden="true"><UserRound :size="22" /></div>
            <div class="panel-photo-info">
              <strong>{{ viewer.name }}</strong>
              <p class="panel-hint">Фото на карте: {{ published.length }} · Звёзды: {{ stars }}</p>
            </div>
            <button type="button" class="button" @click="leave"><LogOut :size="16" aria-hidden="true" /> Выйти</button>
          </div>
          <section aria-labelledby="my-photos-title">
            <h3 id="my-photos-title" class="panel-subtitle">Мои публикации</h3>
            <p v-if="error" class="panel-error" role="alert">{{ error }}</p>
            <p v-if="!myPhotos.length" class="panel-empty"><Camera :size="22" aria-hidden="true" /> У вас пока нет публикаций.</p>
            <ul v-else class="panel-list">
              <li v-for="photo in myPhotos" :key="photo.id" class="panel-list-item">
                <img :src="photo.imageUrl" :alt="photo.title" loading="lazy">
                <div class="panel-photo-info">
                  <strong>{{ photo.title }}</strong>
                  <p class="panel-hint">№{{ photo.id }}</p>
                  <p class="panel-status">{{ photoStatuses[photo.status] }}</p>
                </div>
                <div v-if="deleteId === photo.id" class="panel-row-actions">
                  <span>Удалить фото?</span>
                  <button type="button" class="button" @click="deleteId = null">Отмена</button>
                  <button type="button" class="button button-primary" @click="remove(photo.id)">Удалить</button>
                </div>
                <div v-else class="panel-row-actions">
                  <button v-if="photo.status === 'published'" type="button" class="button" @click="openPhoto(photo.id)">Открыть</button>
                  <button type="button" class="button" @click="deleteId = photo.id">Удалить</button>
                </div>
              </li>
            </ul>
          </section>
        </template>

        <ModerationPanel v-else-if="panel === 'moderation' && viewer?.role === 'moderator'" @notify="emit('notify', $event)" />

        <PhotoUpload
          v-if="uploadLoaded"
          :key="uploadKey"
          v-show="panel === 'upload'"
          :coordinates="coordinates"
          :submit-error="error"
          @choose-point="emit('choosePoint')"
          @submit="upload"
        />
      </div>
    </div>
  </dialog>
</template>

<style>
.account-panel {
  position: fixed;
  inset: 0;
  width: min(100% - 28px, 480px);
  max-width: none;
  max-height: calc(100dvh - 28px);
  margin: auto;
  padding: 0;
  border: 1px solid var(--line);
  border-radius: 16px;
  color: var(--ink);
  background: var(--surface);
  overflow: auto;
  box-shadow: 0 24px 70px #1726223d;
}

.account-panel-wide { width: min(100% - 28px, 690px); }
.account-panel::backdrop { background: #18242170; }

.account-panel .panel-header {
  position: sticky;
  z-index: 1;
  top: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 15px 22px;
  border-bottom: 1px solid var(--line);
  background: var(--surface);
}

.account-panel h2 {
  margin: 0;
  font-size: 1.35rem;
  line-height: 1.25;
  font-weight: 650;
}

.account-panel .panel-content { padding: 20px 22px 22px; }
.account-panel .panel-form { display: grid; gap: 13px; }

.account-panel label {
  display: grid;
  min-width: 0;
  gap: 5px;
  font-size: .86rem;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.account-panel .field {
  display: block;
  width: 100%;
  min-width: 0;
  padding: 9px 11px;
  border: 1px solid var(--line);
  border-radius: 8px;
  color: var(--ink);
  background: var(--surface);
  font-family: inherit;
  font-size: .9rem;
  font-weight: 400;
}

.account-panel textarea.field { resize: vertical; }
.account-panel .button { justify-content: center; min-height: 38px; }
.account-panel .panel-close { flex: none; }

.account-panel .panel-field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.account-panel .panel-source {
  display: grid;
  padding: 9px 0;
  border-top: 1px solid var(--line);
  border-bottom: 1px solid var(--line);
}

.account-panel .panel-source summary { cursor: pointer; font-size: .9rem; font-weight: 600; }
.account-panel .panel-source summary span { color: var(--muted); font-weight: 400; }
.account-panel .panel-source label { margin-top: 11px; }

.account-panel .panel-hint {
  margin: 0;
  color: var(--muted);
  font-size: .8rem;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.account-panel .panel-auth-hint { margin: 11px 0 16px; }
.account-panel .panel-error { margin: 0; color: #a42427; font-size: .85rem; }
.account-panel .panel-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; }

.account-panel .panel-upload,
.account-panel .panel-location,
.account-panel .panel-profile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--background);
}

.account-panel .panel-location strong { font-size: .85rem; }
.account-panel .panel-location > svg,
.account-panel .panel-upload > svg { flex: none; color: var(--accent); }
.account-panel .panel-location > .button { margin-left: auto; }
.account-panel .panel-upload > label { flex: 1; }
.account-panel .panel-upload > img { width: 58px; height: 44px; object-fit: cover; border-radius: 5px; }
.account-panel .panel-upload input::file-selector-button { margin-right: 8px; color: var(--accent); cursor: pointer; }

.account-panel .panel-profile > .button { margin-left: auto; flex: none; }
.account-panel .panel-avatar {
  display: grid;
  flex: none;
  width: 44px;
  height: 44px;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 50%;
  color: var(--accent);
}

.account-panel .panel-subtitle { margin: 21px 0 11px; font-weight: 650; }
.account-panel .panel-filters { display: flex; flex-wrap: wrap; gap: 7px; }
.account-panel .panel-filters [aria-pressed='true'] { background: var(--ink); border-color: var(--ink); color: var(--surface); }

.account-panel .panel-empty {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 0;
  margin: 0;
  color: var(--muted);
  font-size: .92rem;
}

.account-panel .panel-list { margin: 11px 0 0; padding: 0; list-style: none; }
.account-panel .panel-list-item {
  display: grid;
  grid-template-columns: 70px minmax(0, 1fr);
  align-items: start;
  gap: 7px 12px;
  padding: 13px 0;
  border-top: 1px solid var(--line);
}
.account-panel .panel-list-item > img { width: 70px; height: 54px; object-fit: cover; border-radius: 5px; }
.account-panel .panel-photo-info { min-width: 0; overflow-wrap: anywhere; }

.account-panel .panel-status {
  color: var(--accent);
  margin: 3px 0 0;
  font-size: .78rem;
  line-height: 1.2;
  font-weight: 600;
}

.account-panel .panel-row-actions {
  grid-column: 2;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: .84rem;
}

.account-panel .panel-row-actions .button { min-height: 34px; }
@media (max-width: 480px) {
  .account-panel .panel-header,
  .account-panel .panel-content { padding-left: 14px; padding-right: 14px; }
  .account-panel .panel-field-grid { grid-template-columns: 1fr; }
  .account-panel .panel-list-item { grid-template-columns: 50px minmax(0, 1fr); gap: 7px 9px; }
  .account-panel .panel-list-item > img { width: 50px; height: 45px; }
}
</style>
