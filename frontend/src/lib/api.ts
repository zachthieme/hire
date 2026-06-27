const API_BASE = '/api'

function getToken(): string | null {
  return localStorage.getItem('token')
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401 && token && path !== '/auth/login' && path !== '/auth/refresh') {
    // Try to refresh the token
    try {
      const refreshRes = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      })
      if (refreshRes.ok) {
        const data = await refreshRes.json()
        localStorage.setItem('token', data.token)
        // Retry original request with new token
        const retryHeaders = { ...headers, 'Authorization': `Bearer ${data.token}` }
        const retryRes = await fetch(`${API_BASE}${path}`, {
          method,
          headers: retryHeaders,
          body: body ? JSON.stringify(body) : undefined,
        })
        if (retryRes.status === 204) return undefined as T
        if (!retryRes.ok) {
          const err = await retryRes.json().catch(() => ({ error: retryRes.statusText }))
          throw new Error(err.error || retryRes.statusText)
        }
        return retryRes.json()
      }
    } catch {
      // Refresh failed — fall through to logout
    }
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    window.location.href = '/login'
    throw new Error('Session expired')
  }
  if (res.status === 204) return undefined as T
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

// Auth
export const auth = {
  login: (email: string, password: string) =>
    request<{ token: string; user: User }>('POST', '/auth/login', { email, password }),
  me: () => request<User>('GET', '/me'),
  refresh: () => request<{ token: string }>('POST', '/auth/refresh'),
}

// Users
export const users = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return request<User[]>('GET', `/users${qs ? '?' + qs : ''}`)
  },
  create: (data: CreateUserReq) => request<User>('POST', '/users', data),
  update: (id: number, data: CreateUserReq) => request<User>('PUT', `/users/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/users/${id}`),
}

// Candidates
export const candidates = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return request<Candidate[]>('GET', `/candidates${qs ? '?' + qs : ''}`)
  },
  get: (id: number) => request<Candidate>('GET', `/candidates/${id}`),
  create: (data: Partial<Candidate>) => request<Candidate>('POST', '/candidates', data),
  update: (id: number, data: Partial<Candidate>) => request<Candidate>('PUT', `/candidates/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/candidates/${id}`),
}

// Loops
export const loops = {
  list: (params?: { candidate_id?: number; status?: string; limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.candidate_id) q.set('candidate_id', String(params.candidate_id))
    if (params?.status) q.set('status', params.status)
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return request<InterviewLoop[]>('GET', `/loops${qs ? '?' + qs : ''}`)
  },
  get: (id: number) => request<LoopDetail>('GET', `/loops/${id}`),
  create: (data: { candidate_id: number }) => request<InterviewLoop>('POST', '/loops', data),
  update: (id: number, data: Partial<InterviewLoop>) => request<InterviewLoop>('PUT', `/loops/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/loops/${id}`),
}

// Interviews
export const interviews = {
  createInLoop: (loopId: number, data: Partial<Interview>) =>
    request<Interview>('POST', `/loops/${loopId}/interviews`, data),
  update: (id: number, data: Partial<Interview>) => request<Interview>('PUT', `/interviews/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/interviews/${id}`),
  listMine: () => request<Interview[]>('GET', '/me/interviews'),
}

// Feedback
export const feedback = {
  get: (interviewId: number) => request<Feedback>('GET', `/interviews/${interviewId}/feedback`),
  create: (interviewId: number, data: FeedbackCreate) => request<Feedback>('POST', `/interviews/${interviewId}/feedback`, data),
  update: (id: number, data: Partial<Feedback>) => request<Feedback>('PUT', `/feedback/${id}`, data),
}

// Competencies
export const competencies = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return request<Competency[]>('GET', `/competencies${qs ? '?' + qs : ''}`)
  },
  create: (data: Partial<Competency>) => request<Competency>('POST', '/competencies', data),
  update: (id: number, data: Partial<Competency>) => request<Competency>('PUT', `/competencies/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/competencies/${id}`),
}

// Notifications
export const notifications = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return request<Notification[]>('GET', `/notifications${qs ? '?' + qs : ''}`)
  },
  markRead: (id: number) => request<void>('PUT', `/notifications/${id}/read`),
}

// Types
export interface User {
  id: number
  email: string
  name: string
  role: 'admin' | 'scheduler' | 'interviewer'
  created_at: string
}

export interface CreateUserReq {
  email: string
  name: string
  password: string
  role: string
}

export interface Candidate {
  id: number
  name: string
  email: string
  resume_url: string
  status: 'active' | 'hired' | 'rejected' | 'withdrawn'
  created_at: string
}

export interface InterviewLoop {
  id: number
  candidate_id: number
  status: 'scheduling' | 'active' | 'complete'
  final_decision: string | null
  debrief_notes: string | null
  created_by: number
  created_at: string
}

export interface LoopDetail extends InterviewLoop {
  candidate: Candidate
  interviews: InterviewWithFeedback[]
}

export interface Interview {
  id: number
  loop_id: number
  interviewer_id: number
  focus_area: string
  scheduled_at: string
  video_link: string
  notes_for_interviewer: string
  status: 'pending' | 'complete'
  created_at: string
}

export interface InterviewWithFeedback extends Interview {
  interviewer_name: string
  feedback: Feedback | null
}

export interface Feedback {
  id: number
  interview_id: number
  recommendation: 'strong_hire' | 'hire' | 'no_hire' | 'strong_no_hire'
  recommendation_reason: string
  free_form_notes: string
  submitted_at: string
  competency_ratings: CompetencyRating[]
}

export interface FeedbackCreate {
  recommendation: string
  recommendation_reason: string
  free_form_notes: string
  competency_ratings: { competency_id: number; rating_value: string }[]
}

export interface Competency {
  id: number
  name: string
  rating_type: 'levels' | 'stars'
  ratings_json: string
  created_at: string
}

export interface CompetencyRating {
  id: number
  feedback_id: number
  competency_id: number
  rating_value: string
}

export interface Notification {
  id: number
  user_id: number
  message: string
  link: string
  read: boolean
  created_at: string
}
