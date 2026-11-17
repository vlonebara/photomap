<script setup lang="ts">
import { X } from '@lucide/vue'
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import type { Coordinates, Viewer } from '../api/types'
import AuthPanel from './AuthPanel.vue'
import ModerationPanel from './ModerationPanel.vue'
import MyPhotos from './MyPhotos.vue'
import PhotoUpload from './PhotoUpload.vue'

type Panel = 'auth' | 'upload' | 'account' | 'moderation'

const props = defineProps<{
  panel: Panel | null
  viewer: Viewer | null
  mapUpdate: number
  coordinates: Coordinates | null
  uploadKey: number
}>()
const emit = defineEmits<{
  close: []
  choosePoint: []
  notify: [message: string]
  openPhoto: [id: number]
  session: [viewer: Viewer | null]
  sessionExpired: []
  changed: []
}>()

const dialog = ref<HTMLDialogElement | null>(null)
const authMode = ref<'signin' | 'register'>('signin')
const uploadLoaded = ref(false)
const busy = ref(false)
const titles: Record<Panel, string> = {
  auth: 'Войти',
  upload: 'Добавить фото',
  account: 'Личный кабинет',
  moderation: 'Модерация',
}
const heading = computed(() => {
  if (props.panel === 'auth' && authMode.value === 'register') {
    return 'Регистрация'
  }
  return props.panel ? titles[props.panel] : ''
})

function syncDialog() {
  if (props.panel && !dialog.value?.open) {
    dialog.value?.showModal()
  }
  if (!props.panel && dialog.value?.open) {
    dialog.value.close()
  }
}

onMounted(syncDialog)
watch(() => props.uploadKey, () => { uploadLoaded.value = false })
watch(() => props.panel, async (panel) => {
  busy.value = false
  if (panel === 'auth') {
    authMode.value = 'signin'
  }
  if (panel === 'upload') {
    uploadLoaded.value = true
  } else if (panel) {
    uploadLoaded.value = false
  }
  await nextTick()
  syncDialog()
}, { immediate: true })

function close() {
  if (busy.value) return

  if (props.panel === 'upload') {
    uploadLoaded.value = false
  }
  emit('close')
}

function authenticated(viewer: Viewer, registered: boolean) {
  emit('session', viewer)
  emit('notify', registered ? 'Профиль создан.' : 'Вы вошли в аккаунт.')
  close()
}

function signedOut() {
  emit('session', null)
  emit('notify', 'Вы вышли из аккаунта.')
  close()
}

function uploaded() {
  emit('notify', 'Снимок отправлен на модерацию.')
  close()
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
        <button class="button button-icon panel-close" type="button" aria-label="Закрыть" :disabled="busy" @click="close">
          <X :size="20" aria-hidden="true" />
        </button>
      </header>

      <div class="panel-content">
        <AuthPanel v-if="panel === 'auth'" v-model:mode="authMode" @authenticated="authenticated" @busy="busy = $event" />

        <MyPhotos
          v-else-if="panel === 'account' && viewer"
          :key="viewer.id"
          :viewer="viewer"
          :map-update="mapUpdate"
          @signed-out="signedOut"
          @session-expired="emit('sessionExpired')"
          @busy="busy = $event"
          @changed="emit('changed')"
          @notify="emit('notify', $event)"
          @open-photo="openPhoto"
        />

        <ModerationPanel
          v-else-if="panel === 'moderation' && viewer?.role === 'moderator'"
          :key="viewer.id"
          :map-update="mapUpdate"
          @session-expired="emit('sessionExpired')"
          @busy="busy = $event"
          @changed="emit('changed')"
          @notify="emit('notify', $event)"
        />

        <PhotoUpload
          v-if="uploadLoaded"
          :key="uploadKey"
          v-show="panel === 'upload'"
          :coordinates="coordinates"
          @choose-point="emit('choosePoint')"
          @uploaded="uploaded"
          @session-expired="emit('sessionExpired')"
          @busy="busy = $event"
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

.account-panel .panel-auth-form { margin-top: 16px; }
.account-panel .panel-error { margin: 0; color: #a42427; font-size: .85rem; }

.account-panel .panel-list-error {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
}
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
}
</style>
