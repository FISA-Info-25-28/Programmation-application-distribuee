import { useState } from 'react'
import type { ReactNode } from 'react'
import { getMe, logout as apiLogout } from '../api/auth'
import { getToken, setToken, clearToken, setRefreshToken, getRefreshToken } from '../api/client'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AuthContext } from './auth-context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(getToken)
  const qc = useQueryClient()

  // Le user vit dans le cache React Query sous la clé ['me'].
  // Toute mutation qui appelle qc.setQueryData(['me'], user) met à jour
  // instantanément tous les composants qui lisent useAuth().user.
  const { data: user = null, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: getMe,
    enabled: !!token,
    retry: false,
    staleTime: Infinity,
  })

  const login = (accessToken: string, refreshToken: string) => {
    setRefreshToken(refreshToken)
    setToken(accessToken)
    setTokenState(accessToken)
    // Invalide ['me'] pour forcer un refetch avec le nouveau token
    qc.invalidateQueries({ queryKey: ['me'] })
  }

  const logout = () => {
    const rt = getRefreshToken()
    if (rt) apiLogout(rt)
    clearToken()
    setTokenState(null)
    qc.setQueryData(['me'], null)
    qc.removeQueries({ queryKey: ['me'] })
  }

  return (
    <AuthContext value={{ user, token, isLoading: !!token && isLoading, isAuthenticated: !!user, login, logout }}>
      {children}
    </AuthContext>
  )
}
