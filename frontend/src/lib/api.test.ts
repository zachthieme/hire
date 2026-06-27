import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { auth } from './api'

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch)
  localStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('auth.login', () => {
  it('sends credentials and returns token + user', async () => {
    const mockResponse = {
      token: 'test-token',
      user: { id: 1, email: 'test@test.com', name: 'Test', role: 'admin', created_at: '2024-01-01' },
    }
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockResponse),
    })

    const result = await auth.login('test@test.com', 'password123')
    expect(result).toEqual(mockResponse)
    expect(mockFetch).toHaveBeenCalledWith('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'test@test.com', password: 'password123' }),
    })
  })
})

describe('request with auth', () => {
  it('includes Authorization header when token exists', async () => {
    localStorage.setItem('token', 'my-token')
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: 1, email: 'test@test.com', name: 'Test', role: 'admin', created_at: '2024-01-01' }),
    })

    await auth.me()
    expect(mockFetch).toHaveBeenCalledWith('/api/me', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer my-token',
      },
      body: undefined,
    })
  })

  it('throws on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 400,
      statusText: 'Bad Request',
      json: () => Promise.resolve({ error: 'invalid body' }),
    })

    await expect(auth.login('', '')).rejects.toThrow('invalid body')
  })
})
