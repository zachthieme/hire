import { createContext, useContext, type ReactNode } from 'react'

const AuthContext = createContext<any>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  return <AuthContext.Provider value={{}}>{children}</AuthContext.Provider>
}

export function useAuth() {
  return useContext(AuthContext)
}
