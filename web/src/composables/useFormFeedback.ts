import { ref, shallowRef } from 'vue'
import { errorMessage, fieldErrors } from '../api/client'
import type { FieldErrors } from '../api/client'

export function useFormFeedback() {
  const message = ref('')
  const errors = shallowRef<FieldErrors>({})

  function clearFeedback() {
    message.value = ''
    errors.value = {}
  }

  function showFeedback(error: unknown) {
    errors.value = fieldErrors(error)
    const text = errorMessage(error)
    message.value = Object.values(errors.value).includes(text) ? '' : text
  }

  return { message, errors, clearFeedback, showFeedback }
}
