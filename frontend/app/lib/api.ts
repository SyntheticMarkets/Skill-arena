export const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'
let refreshInFlight: Promise<boolean> | null = null

export type ApiErrorBody = {
  code: string
  message: string
}

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.name = 'ApiError'
    this.status = status
    this.code = body.code
  }
}

function deviceHeaders() {
  if (typeof window === 'undefined') return {}
  let fingerprint = window.localStorage.getItem('skill-arena-device')
  if (!fingerprint) {
    fingerprint = window.crypto.randomUUID()
    window.localStorage.setItem('skill-arena-device', fingerprint)
  }
  return {
    'X-Device-Fingerprint': fingerprint,
    'X-Device-Name': navigator.platform || 'Browser',
    'X-Device-OS': navigator.platform || 'Unknown',
    'X-Device-Browser': navigator.userAgent,
  }
}

async function parseError(response: Response) {
  const body = await response.json().catch(() => null) as Partial<ApiErrorBody> | null
  return new ApiError(response.status, {
    code: body?.code ?? 'REQUEST_FAILED',
    message: body?.message ?? 'The request could not be completed.',
  })
}

function recoverSession() {
  if (!refreshInFlight) {
    const refreshHeaders = new Headers()
    for (const [name, value] of Object.entries(deviceHeaders())) refreshHeaders.set(name, value)
    refreshInFlight = fetch(`${apiBase}/api/v1/auth/refresh-token`, {
      method: 'POST',
      credentials: 'include',
      headers: refreshHeaders,
      signal: AbortSignal.timeout(10_000),
    }).then((response) => response.ok).finally(() => { refreshInFlight = null })
  }
  return refreshInFlight
}

export async function apiFetch<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  for (const [name, value] of Object.entries(deviceHeaders())) headers.set(name, value)
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    credentials: 'include',
    headers,
    signal: init.signal ?? AbortSignal.timeout(10_000),
  })

  if (response.status === 401 && retry && path !== '/api/v1/auth/refresh-token') {
    if (await recoverSession()) return apiFetch<T>(path, init, false)
  }

  if (!response.ok) throw await parseError(response)
  if (response.status === 204) return undefined as T
  const payload = await response.text()
  if (!payload) return undefined as T
  return JSON.parse(payload) as T
}

export function postJSON<T>(path: string, body?: unknown) {
  return apiFetch<T>(path, {
    method: 'POST',
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}
