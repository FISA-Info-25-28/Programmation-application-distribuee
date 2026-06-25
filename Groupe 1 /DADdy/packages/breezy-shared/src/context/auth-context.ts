import { createContext } from 'react'
import type { User } from '../types/api'

interface AuthContextValue {
  user: User | null
  token: string | null
  isLoading: boolean
  isAuthenticated: boolean
  login: (accessToken: string, refreshToken: string) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export { AuthContext }
