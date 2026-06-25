import { describe, expect, it } from 'vitest'
import { classifyLoginError } from './loginError'

function axiosError(status?: number, message?: string) {
  return {
    isAxiosError: true,
    response:
      status === undefined ? undefined : { status, data: { message } },
  }
}

describe('classifyLoginError', () => {
  it('no response → network', () => {
    expect(classifyLoginError(axiosError())).toBe('network')
  })

  it('non-axios error → network', () => {
    expect(classifyLoginError(new Error('boom'))).toBe('network')
  })

  it('429 → rateLimited', () => {
    expect(classifyLoginError(axiosError(429))).toBe('rateLimited')
  })

  it('403 email not verified → unverified', () => {
    expect(classifyLoginError(axiosError(403, 'email not verified'))).toBe('unverified')
  })

  it('403 account suspended → suspended', () => {
    expect(classifyLoginError(axiosError(403, 'account suspended'))).toBe('suspended')
  })

  it('403 other message → network', () => {
    expect(classifyLoginError(axiosError(403, 'other reason'))).toBe('network')
  })

  it.each([
    [400, 'credentials'],
    [401, 'credentials'],
    [422, 'credentials'],
  ])('4xx status %d → credentials', (status, expected) => {
    expect(classifyLoginError(axiosError(status))).toBe(expected)
  })

  it('500 → network', () => {
    expect(classifyLoginError(axiosError(500))).toBe('network')
  })
})
