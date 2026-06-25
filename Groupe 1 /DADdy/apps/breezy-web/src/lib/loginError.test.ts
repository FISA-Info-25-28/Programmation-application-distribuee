import { describe, expect, it } from 'vitest'
import { classifyLoginError } from './loginError'

function axiosError(status?: number, message?: string) {
  return {
    isAxiosError: true,
    response: status === undefined ? undefined : { status, data: { message } },
  }
}

describe('classifyLoginError', () => {
  it('classifies invalid credentials', () => {
    expect(classifyLoginError(axiosError(401, 'invalid credentials'))).toBe('credentials')
  })

  it('classifies unverified accounts separately', () => {
    expect(classifyLoginError(axiosError(403, 'email not verified'))).toBe('unverified')
  })

  it('classifies account suspended responses as suspended', () => {
    expect(classifyLoginError(axiosError(403, 'account suspended'))).toBe('suspended')
  })

  it('classifies unknown forbidden responses as network errors', () => {
    expect(classifyLoginError(axiosError(403, 'forbidden origin'))).toBe('network')
    expect(classifyLoginError(axiosError(403, 'missing X-Requested-With header'))).toBe('network')
  })

  it('classifies rate limiting separately', () => {
    expect(classifyLoginError(axiosError(429, 'too many attempts'))).toBe('rateLimited')
  })

  it.each([axiosError(500), axiosError(), new Error('offline')])(
    'classifies server and network failures as unavailable',
    (error) => {
      expect(classifyLoginError(error)).toBe('network')
    },
  )
})
