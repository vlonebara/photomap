import { onUnmounted, ref, shallowRef } from 'vue'
import { errorMessage, isApiError } from '../api/client'
import type { ManagedPhoto } from '../api/types'

export function usePhotoList(load: (signal: AbortSignal) => Promise<ManagedPhoto[]>, onSessionExpired: () => void) {
  const items = shallowRef<ManagedPhoto[]>([])
  const loading = ref(true)
  const error = ref('')
  let activeRequest: AbortController | null = null

  async function reload() {
    activeRequest?.abort()
    const request = new AbortController()
    activeRequest = request
    loading.value = true
    error.value = ''

    try {
      const photos = await load(request.signal)
      if (!request.signal.aborted) {
        items.value = photos
      }
    } catch (cause) {
      if (request.signal.aborted) {
        return
      }
      if (isApiError(cause, 401)) {
        onSessionExpired()
      } else {
        error.value = errorMessage(cause)
      }
    } finally {
      if (activeRequest === request) {
        loading.value = false
        activeRequest = null
      }
    }
  }

  onUnmounted(() => activeRequest?.abort())
  return { items, loading, error, reload }
}
