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

// List endpoints: the Go API serializes an empty slice as `null`, so coalesce
// to [] here. Otherwise consumers that call .length/.map on the result crash.
async function requestList<T>(method: string, path: string): Promise<T[]> {
  return (await request<T[] | null>(method, path)) ?? []
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
    return requestList<User>('GET', `/users${qs ? '?' + qs : ''}`)
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
    return requestList<Candidate>('GET', `/candidates${qs ? '?' + qs : ''}`)
  },
  get: (id: number) => request<Candidate>('GET', `/candidates/${id}`),
  create: (data: Partial<Candidate>) => request<Candidate>('POST', '/candidates', data),
  update: (id: number, data: Partial<Candidate>) => request<Candidate>('PUT', `/candidates/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/candidates/${id}`),
}

// Jobs
export const jobs = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return requestList<Job>('GET', `/jobs${qs ? '?' + qs : ''}`)
  },
  get: (id: number) => request<JobDetail>('GET', `/jobs/${id}`),
  create: (data: Partial<Job>) => request<Job>('POST', '/jobs', data),
  update: (id: number, data: Partial<Job>) => request<Job>('PUT', `/jobs/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/jobs/${id}`),
}

// Applications
export const applications = {
  get: (id: number) => request<ApplicationDetail>('GET', `/applications/${id}`),
  create: (jobId: number, candidateId: number) =>
    request<Application>('POST', `/jobs/${jobId}/applications`, { candidate_id: candidateId }),
  update: (id: number, data: Partial<Application>) => request<Application>('PUT', `/applications/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/applications/${id}`),
}

// Stages
export const stages = {
  create: (applicationId: number, data: Partial<Stage>) =>
    request<Stage>('POST', `/applications/${applicationId}/stages`, data),
  update: (id: number, data: Partial<Stage>) => request<Stage>('PUT', `/stages/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/stages/${id}`),
  addInterviewer: (stageId: number, interviewerId: number) =>
    request<void>('POST', `/stages/${stageId}/interviewers`, { interviewer_id: interviewerId }),
  removeInterviewer: (stageId: number, interviewerId: number) =>
    request<void>('DELETE', `/stages/${stageId}/interviewers/${interviewerId}`),
  feedback: (stageId: number) => requestList<Feedback>('GET', `/stages/${stageId}/feedback`),
  submitFeedback: (stageId: number, data: FeedbackCreate) =>
    request<Feedback>('POST', `/stages/${stageId}/feedback`, data),
}

// Interviewer
export const myStages = {
  list: () => requestList<MyStage>('GET', '/me/stages'),
}

// Competencies
export const competencies = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return requestList<Competency>('GET', `/competencies${qs ? '?' + qs : ''}`)
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
    return requestList<Notification>('GET', `/notifications${qs ? '?' + qs : ''}`)
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
  created_at: string
}

export interface Job {
  id: number
  title: string
  description: string
  hiring_manager: string
  status: 'open' | 'closed' | 'filled'
  created_by: number
  created_at: string
}

export interface Application {
  id: number
  job_id: number
  candidate_id: number
  status: 'active' | 'rejected' | 'hired' | 'withdrawn'
  final_decision: 'strong_hire' | 'hire' | 'no_hire' | 'strong_no_hire' | null
  final_interview_notes: string | null
  created_by: number
  created_at: string
}

export interface ApplicationSummary extends Application {
  candidate_name: string
  candidate_email: string
}

export interface JobDetail extends Job {
  applications: ApplicationSummary[]
}

export interface Stage {
  id: number
  application_id: number
  type: 'phone_screen' | 'interview'
  focus_area: string
  scheduled_at: string
  video_link: string
  notes_for_interviewer: string
  status: 'pending' | 'complete' | 'canceled'
  created_at: string
}

export interface StageParticipant {
  interviewer_id: number
  interviewer_name: string
  feedback?: Feedback | null
}

export interface StageWithFeedback extends Stage {
  participants: StageParticipant[]
}

export interface ApplicationDetail extends Application {
  job: Job
  candidate: Candidate
  stages: StageWithFeedback[]
}

export interface MyStage extends Stage {
  candidate_name: string
  job_title: string
  has_my_feedback: boolean
}

export interface Feedback {
  id: number
  stage_id: number
  interviewer_id: number
  recommendation: 'strong_hire' | 'hire' | 'no_hire' | 'strong_no_hire'
  recommendation_reason: string
  free_form_notes: string
  competency_ratings?: CompetencyRating[]
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
