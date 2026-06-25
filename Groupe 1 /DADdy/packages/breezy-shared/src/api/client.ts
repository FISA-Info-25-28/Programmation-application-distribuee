import axios from 'axios'
import { Capacitor } from '@capacitor/core'
import type { AuthResponse } from '../types/api'
import { storage } from './storage'

const TOKEN_KEY = 'breezy_token'
const REFRESH_TOKEN_KEY = 'breezy_refresh_token'

export const getToken = () => storage.getSync(TOKEN_KEY)
export const setToken = (token: string) => { void storage.set(TOKEN_KEY, token) }
export const clearToken = () => {
  void storage.remove(TOKEN_KEY)
  void storage.remove(REFRESH_TOKEN_KEY)
}

export const getRefreshToken = () => storage.getSync(REFRESH_TOKEN_KEY)
export const setRefreshToken = (token: string) => { void storage.set(REFRESH_TOKEN_KEY, token) }

/**
 * URL de base de l'API.
 * - Web : '/api' (proxy Vite, rewrites /api/* → http://localhost:3001/*)
 * - Native (Capacitor) : VITE_API_URL (URL absolue du serveur)
 */
export const getApiBaseURL = (): string => {
  if (Capacitor.isNativePlatform()) {
    return (import.meta as unknown as { env: Record<string, string> }).env.VITE_API_URL ?? 'https://api.breezy.example.com'
  }
  return '/api'
}

export const client = axios.create({
  baseURL: getApiBaseURL(),
  headers: {
    'Content-Type': 'application/json',
    // Requis par le middleware CSRF du api-gateway (withCSRF).
    'X-Requested-With': 'XMLHttpRequest',
  },
})

client.interceptors.request.use((config) => {
  const token = getToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// Queue de callbacks à appeler une fois le refresh terminé
let isRefreshing = false
let refreshQueue: Array<(token: string) => void> = []

function drainQueue(token: string) {
  refreshQueue.forEach((cb) => cb(token))
  refreshQueue = []
}

client.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config

    // Ignorer les erreurs qui ne sont pas des 401, et éviter les boucles
    // (la requête de refresh elle-même ou une requête déjà retentée)
    if (error.response?.status !== 401 || original._retry || original.url === '/auth/refresh') {
      return Promise.reject(error)
    }

    const rt = getRefreshToken()
    if (!rt) {
      clearToken()
      window.location.href = '/login'
      return Promise.reject(error)
    }

    // Si un refresh est déjà en cours, on attend qu'il se termine
    if (isRefreshing) {
      return new Promise<string>((resolve) => {
        refreshQueue.push(resolve)
      }).then((newToken) => {
        original.headers.Authorization = `Bearer ${newToken}`
        return client(original)
      })
    }

    original._retry = true
    isRefreshing = true

    try {
      const { data } = await client.post<AuthResponse>('/auth/refresh', { refreshToken: rt })
      setToken(data.accessToken)
      setRefreshToken(data.refreshToken)
      drainQueue(data.accessToken)
      original.headers.Authorization = `Bearer ${data.accessToken}`
      return client(original)
    } catch {
      clearToken()
      drainQueue('')
      window.location.href = '/login'
      return Promise.reject(error)
    } finally {
      isRefreshing = false
    }
  },
)
