import { describe, expect, it } from 'vitest'
import { classifyPasswordError, passwordErrorMessage } from './passwordError'

function axiosError(status?: number, message?: string) {
  return {
    isAxiosError: true,
    response:
      status === undefined ? undefined : { status, data: { message } },
  }
}

describe('classifyPasswordError', () => {
  it('no response → network', () => {
    expect(classifyPasswordError(axiosError())).toBe('network')
  })

  it('non-axios error → network', () => {
    expect(classifyPasswordError(new Error('boom'))).toBe('network')
  })

  it('409 → conflict', () => {
    expect(classifyPasswordError(axiosError(409))).toBe('conflict')
  })

  it('500 → network', () => {
    expect(classifyPasswordError(axiosError(500))).toBe('network')
  })

  it.each([
    ['too common', 'tooCommon'],
    ['between 12 and 72', 'length'],
    ['at least 3 of', 'composition'],
    ['must not contain your username', 'identifier'],
    ['differ from the current', 'sameAsCurrent'],
    ['current password is incorrect', 'currentIncorrect'],
    ['reset token', 'invalidToken'],
    ['username must be between', 'usernameLength'],
    ['invalid email', 'invalidEmail'],
  ] as const)('message fragment "%s" → %s', (fragment, expected) => {
    expect(classifyPasswordError(axiosError(400, fragment))).toBe(expected)
  })

  it('unknown 4xx message → generic', () => {
    expect(classifyPasswordError(axiosError(400, 'something unknown'))).toBe('generic')
  })

  it('null data → generic (covers data?.message ?? empty string branch)', () => {
    expect(
      classifyPasswordError({ isAxiosError: true, response: { status: 400, data: null } }),
    ).toBe('generic')
  })
})

describe('passwordErrorMessage', () => {
  it('returns a non-empty string for each error kind', () => {
    const msg = passwordErrorMessage(axiosError(409))
    expect(typeof msg).toBe('string')
    expect(msg.length).toBeGreaterThan(0)
  })
})
