import { computed, shallowRef } from 'vue'
import { initialPhotos } from './photos'
import { makeSeries } from './series'
import { validatePhoto } from './validate'
import type { Photo, PhotoDraft, PhotoStatus, Role, Viewer } from './types'

interface Report {
  viewerId: string
  active: boolean
}

interface ArchiveState {
  photos: Photo[]
  stars: Record<number, string[]>
  reports: Record<number, Report[]>
}

const archiveKey = 'photomap.demo.archive.v1'
const viewerKey = 'photomap.demo.viewer.v1'
const allowedTransitions: Record<PhotoStatus, readonly PhotoStatus[]> = {
  pending: ['pending', 'published', 'rejected'],
  published: ['published', 'rejected'],
  review: ['review', 'published', 'rejected'],
  rejected: ['rejected', 'published'],
  removed: [],
}

function readArchive(): ArchiveState | undefined {
  try {
    const stored = window.localStorage.getItem(archiveKey)
    if (!stored) return
    const saved = JSON.parse(stored) as Partial<ArchiveState>
    if (Array.isArray(saved.photos) && saved.stars && saved.reports) {
      return saved as ArchiveState
    }
  } catch {
    // The archive can still be viewed when browser storage is unavailable.
  }
}

function readViewer(): Viewer | null {
  try {
    const stored = window.sessionStorage.getItem(viewerKey)
    return stored ? JSON.parse(stored) as Viewer : null
  } catch {
    return null
  }
}

function initialArchive(): ArchiveState {
  return { photos: initialPhotos, stars: {}, reports: {} }
}

const state = shallowRef<ArchiveState>(readArchive() ?? initialArchive())
const viewer = shallowRef<Viewer | null>(readViewer())
const photos = computed(() => state.value.photos)
const series = computed(() => makeSeries(photos.value))

window.addEventListener('storage', event => {
  if (event.key === archiveKey || event.key === null) {
    state.value = readArchive() ?? initialArchive()
  }
})

function updateArchive<T>(apply: (next: ArchiveState) => T): T {
  const latest = readArchive() ?? state.value
  const next: ArchiveState = {
    photos: [...latest.photos],
    stars: { ...latest.stars },
    reports: { ...latest.reports },
  }
  const result = apply(next)

  try {
    window.localStorage.setItem(archiveKey, JSON.stringify(next))
  } catch (error) {
    if (error instanceof DOMException && error.name === 'QuotaExceededError') {
      throw new Error('В браузере не осталось места для демо. Попробуйте загрузить фото меньшего размера.')
    }
    throw new Error('Браузер не дал сохранить изменения. Проверьте настройки хранения данных сайта.')
  }

  state.value = next
  return result
}

function setViewer(user: Viewer | null) {
  viewer.value = user
  try {
    if (user) window.sessionStorage.setItem(viewerKey, JSON.stringify(user))
    else window.sessionStorage.removeItem(viewerKey)
  } catch {
    // Keep the mock session in memory until the page closes.
  }
}

function signin(name: string, role: Role = 'user'): Viewer {
  const trimmed = name.trim()
  if (!trimmed) throw new Error('Введите имя.')
  const normalized = trimmed.normalize('NFKC').toLocaleLowerCase('ru')
  const user = { id: `${role}:${normalized}`, name: trimmed, role }
  setViewer(user)
  return user
}

function signout() {
  setViewer(null)
}

function requireViewer() {
  if (!viewer.value) throw new Error('Войдите, чтобы продолжить.')
  return viewer.value
}

function findPhoto(next: ArchiveState, id: number, publishedOnly = false) {
  const photo = next.photos.find(photo => photo.id === id &&
    (!publishedOnly || photo.status === 'published'))
  if (!photo) throw new Error('Фото недоступно.')
  return photo
}

function replacePhoto(next: ArchiveState, updated: Photo) {
  next.photos = next.photos.map(photo => photo.id === updated.id ? updated : photo)
  return updated
}

function addPhoto(draft: PhotoDraft): Photo {
  const user = requireViewer()
  const fields = validatePhoto(draft)

  return updateArchive(next => {
    const id = next.photos.reduce((lastId, photo) => Math.max(lastId, photo.id), 0) + 1
    const photo: Photo = {
      ...fields,
      id,
      authorId: user.id,
      authorName: user.name,
      stars: 0,
      status: 'pending',
    }
    next.photos.push(photo)
    return photo
  })
}

function hasStar(id: number) {
  const user = viewer.value
  return user ? (state.value.stars[id] ?? []).includes(user.id) : false
}

function toggleStar(id: number): boolean {
  const user = requireViewer()
  return updateArchive(next => {
    const photo = findPhoto(next, id, true)
    const voters = next.stars[id] ?? []
    const alreadyStarred = voters.includes(user.id)
    next.stars[id] = alreadyStarred
      ? voters.filter(voter => voter !== user.id)
      : [...voters, user.id]

    replacePhoto(next, { ...photo, stars: photo.stars + (alreadyStarred ? -1 : 1) })
    return !alreadyStarred
  })
}

function hasReported(id: number) {
  const user = viewer.value
  return user ? (state.value.reports[id] ?? []).some(report => report.viewerId === user.id) : false
}

function report(id: number): PhotoStatus {
  const user = requireViewer()
  return updateArchive(next => {
    const photo = findPhoto(next, id, true)
    const reports = next.reports[id] ?? []
    if (reports.some(report => report.viewerId === user.id)) {
      throw new Error('Вы уже пожаловались на эту публикацию.')
    }

    const updated = [...reports, { viewerId: user.id, active: true }]
    next.reports[id] = updated
    if (updated.filter(report => report.active).length >= 3) {
      return replacePhoto(next, { ...photo, status: 'review' }).status
    }
    return photo.status
  })
}

function removePhoto(id: number): Photo {
  const user = requireViewer()
  return updateArchive(next => {
    const photo = findPhoto(next, id)
    if (photo.authorId !== user.id) throw new Error('Удалить публикацию может только её автор.')
    return replacePhoto(next, { ...photo, status: 'removed' })
  })
}

function moderate(id: number, status: PhotoStatus, edits?: Partial<PhotoDraft>): Photo {
  if (requireViewer().role !== 'moderator') throw new Error('Действие доступно модератору.')
  return updateArchive(next => {
    const photo = findPhoto(next, id)
    if (!allowedTransitions[photo.status].includes(status)) {
      throw new Error('Для этой публикации такое действие недоступно.')
    }
    const fields = edits ? validatePhoto({ ...photo, ...edits }) : {}
    if (status !== photo.status && next.reports[id]) {
      next.reports[id] = next.reports[id].map(report => ({ ...report, active: false }))
    }
    return replacePhoto(next, { ...photo, ...fields, status })
  })
}

export function useDemo() {
  return {
    viewer,
    photos,
    series,
    signin,
    signout,
    addPhoto,
    toggleStar,
    hasStar,
    report,
    hasReported,
    removePhoto,
    moderate,
  }
}
