import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import { auth as authApi, type User } from './api'

interface AuthContextType {
  user: User | null
  token: string | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  isAuthenticated: boolean
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const stored = localStorage.getItem('user')
    return stored ? JSON.parse(stored) : null
  })
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'))

  const login = useCallback(async (email: string, password: string) => {
    const res = await authApi.login(email, password)
    localStorage.setItem('token', res.token)
    localStorage.setItem('user', JSON.stringify(res.user))
    setToken(res.token)
    setUser(res.user)
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setToken(null)
    setUser(null)
  }, [])

  // Refresh user data on mount if we have a token
  useEffect(() => {
    if (token) {
      authApi.me().then(freshUser => {
        localStorage.setItem('user', JSON.stringify(freshUser))
        setUser(freshUser)
      }).catch(() => {
        // Token is invalid — clear auth
        logout()
      })
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Refresh token every 20 minutes to keep session alive
  useEffect(() => {
    if (!token) return
    const interval = setInterval(async () => {
      try {
        const res = await authApi.refresh()
        localStorage.setItem('token', res.token)
        setToken(res.token)
      } catch {
        logout()
      }
    }, 20 * 60 * 1000)
    return () => clearInterval(interval)
  }, [token, logout])

  return (
    <AuthContext.Provider value={{ user, token, login, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
