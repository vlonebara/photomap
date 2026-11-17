export const apiBase = '/api'

export type FieldErrors = Record<string, string>

interface ErrorResponse {
  code?: string
  message?: string
  fields?: FieldErrors
}

export class ApiError extends Error {
  readonly code: string
  readonly fields: FieldErrors

  constructor(readonly status: number, body: ErrorResponse | null) {
    super(body?.message || `Не удалось выполнить запрос (HTTP ${status}).`)
    this.code = body?.code ?? ''
    this.fields = body?.fields ?? {}
  }
}

export function isApiError(error: unknown, status: number) {
  return error instanceof ApiError && error.status === status
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Не удалось выполнить действие. Попробуйте ещё раз.'
}

export function fieldErrors(error: unknown): FieldErrors {
  return error instanceof ApiError ? error.fields : {}
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: object
  signal?: AbortSignal
}

// keep the session cookie on the same origin for JSON and file requests
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers({ Accept: 'application/json' })
  const { body, method = 'GET', signal } = options
  let requestBody: BodyInit | undefined

  if (body instanceof FormData) {
    requestBody = body
  } else if (body !== undefined) {
    headers.set('Content-Type', 'application/json')
    requestBody = JSON.stringify(body)
  }

  let response: Response
  try {
    response = await fetch(apiBase + path, {
      method,
      headers,
      body: requestBody,
      signal,
      credentials: 'include',
    })
  } catch (error) {
    if (signal?.aborted) {
      throw error
    }
    throw new Error('Не удалось связаться с сервером PhotoMap.')
  }

  if (!response.ok) {
    const result: ErrorResponse | null = await response.json().catch(() => null)
    throw new ApiError(response.status, result)
  }

  if (response.status === 204) {
    return undefined as T
  }

  if (!response.headers.get('Content-Type')?.includes('application/json')) {
    throw new Error('Сервер PhotoMap вернул ответ в неожиданном формате.')
  }

  return response.json()
}
