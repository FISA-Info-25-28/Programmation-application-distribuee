import { isAxiosError } from 'axios'

export type LoginErrorKind = 'credentials' | 'network' | 'rateLimited' | 'suspended' | 'unverified'

export function classifyLoginError(error: unknown): LoginErrorKind {
  if (!isAxiosError(error) || !error.response) return 'network'

  const { status, data } = error.response
  if (status === 429) return 'rateLimited'
  if (status === 403) {
    if (data?.message === 'email not verified') return 'unverified'
    if (data?.message === 'account suspended') return 'suspended'
    return 'network'
  }
  return status < 500 ? 'credentials' : 'network'
}
