import { apiBase, request } from './client'
import type {
  ManagedPhoto,
  ModerationAction,
  ModerationFilter,
  PhotoInput,
  PublicPhoto,
  SeriesDetail,
  SeriesSummary,
  Viewer,
} from './types'

interface Credentials {
  email: string
  password: string
}

interface StarResult {
  stars: number
  starredByMe: boolean
}

interface ReportResult {
  reportedByMe: boolean
}

export const mapEventsUrl = apiBase + '/events'

export const photoMapApi = {
  getViewer(signal?: AbortSignal) {
    return request<Viewer>('/auth/me', { signal })
  },
  login(body: Credentials) {
    return request<Viewer>('/auth/login', { method: 'POST', body })
  },
  register(body: Credentials & { name: string }) {
    return request<Viewer>('/auth/register', { method: 'POST', body })
  },
  logout() {
    return request<void>('/auth/logout', { method: 'POST' })
  },
  listSeries(signal?: AbortSignal) {
    return request<SeriesSummary[]>('/series', { signal })
  },
  getSeries(id: number, signal?: AbortSignal) {
    return request<SeriesDetail>(`/series/${id}`, { signal })
  },
  getPhoto(id: number, signal?: AbortSignal) {
    return request<PublicPhoto>(`/photos/${id}`, { signal })
  },
  setStar(id: number, starred: boolean) {
    return request<StarResult>(`/photos/${id}/star`, { method: starred ? 'PUT' : 'DELETE' })
  },
  report(id: number) {
    return request<ReportResult>(`/photos/${id}/reports`, { method: 'POST' })
  },
  upload(draft: PhotoInput, image: File) {
    const body = new FormData()
    body.set('image', image)

    for (const [key, value] of Object.entries(draft)) {
      if (value !== null) {
        body.set(key, String(value))
      }
    }

    return request<ManagedPhoto>('/photos', { method: 'POST', body })
  },
  listOwnPhotos(signal?: AbortSignal) {
    return request<ManagedPhoto[]>('/me/photos', { signal })
  },
  removeOwnPhoto(id: number) {
    return request<void>(`/me/photos/${id}`, { method: 'DELETE' })
  },
  listModerationPhotos(filter: ModerationFilter, signal?: AbortSignal) {
    const query = filter === 'queue' ? '' : `?status=${filter}`
    return request<ManagedPhoto[]>(`/moderation/photos${query}`, { signal })
  },
  getModerationPhoto(id: number, signal?: AbortSignal) {
    return request<ManagedPhoto>(`/moderation/photos/${id}`, { signal })
  },
  editPhoto(id: number, body: Partial<PhotoInput>) {
    return request<ManagedPhoto>(`/moderation/photos/${id}`, { method: 'PATCH', body })
  },
  moderatePhoto(id: number, action: ModerationAction) {
    return request<ManagedPhoto>(`/moderation/photos/${id}/decision`, { method: 'POST', body: { action } })
  },
}
