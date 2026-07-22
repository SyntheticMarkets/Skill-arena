import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiFetch, postJSON } from './api'

describe('API client', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('uses cookie credentials and persistent device identity', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 200 }))
    await postJSON('/api/v1/auth/login', { email: 'player@example.com' })
    const [, init] = fetchMock.mock.calls[0]
    const headers = new Headers(init?.headers)
    expect(init?.credentials).toBe('include')
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('X-Device-Fingerprint')).toBe('00000000-0000-4000-8000-000000000001')
    expect(localStorage.getItem('skill-arena-device')).toBeTruthy()
  })

  it('recovers once from an expired access session', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ code: 'AUTH_UNAUTHORIZED', message: 'expired' }), { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ authenticated: true }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ id: 'player-1' }), { status: 200 }))
    const body = await apiFetch<{ id: string }>('/api/v1/profile')
    expect(body.id).toBe('player-1')
    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(String(fetchMock.mock.calls[1][0])).toContain('/api/v1/auth/refresh-token')
  })
})
