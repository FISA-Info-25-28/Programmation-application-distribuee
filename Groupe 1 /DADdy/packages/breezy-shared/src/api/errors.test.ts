import { describe, expect, it } from 'vitest'
import { apiErrorMessage } from './errors'

function axiosError(message?: string) {
  return {
    isAxiosError: true,
    response: message !== undefined ? { data: { message } } : undefined,
  }
}

describe('apiErrorMessage', () => {
  it('returns message from AxiosError response', () => {
    expect(apiErrorMessage(axiosError('oops'), 'fallback')).toBe('oops')
  })

  it('returns fallback when AxiosError has no response', () => {
    expect(apiErrorMessage(axiosError(), 'fallback')).toBe('fallback')
  })

  it('returns fallback for a standard Error', () => {
    expect(apiErrorMessage(new Error('boom'), 'fallback')).toBe('fallback')
  })
})
