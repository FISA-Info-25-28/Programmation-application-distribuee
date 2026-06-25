import { describe, expect, it } from 'vitest'
import { classifyPasswordError } from './passwordError'

function axiosError(status?: number, message?: string) {
  return {
    isAxiosError: true,
    response: status === undefined ? undefined : { status, data: { message } },
  }
}

describe('classifyPasswordError', () => {
  it('classifies a common password', () => {
    expect(classifyPasswordError(axiosError(400, 'validation: password is too common'))).toBe('tooCommon')
  })

  it('classifies a too short / too long password', () => {
    expect(classifyPasswordError(axiosError(400, 'password must be between 12 and 72 characters'))).toBe('length')
  })

  it('classifies insufficient character categories', () => {
    expect(
      classifyPasswordError(axiosError(400, 'password must contain at least 3 of these: lowercase letter, ...')),
    ).toBe('composition')
  })

  it('classifies a password containing the identity', () => {
    expect(classifyPasswordError(axiosError(400, 'password must not contain your username or email'))).toBe(
      'identifier',
    )
  })

  it('classifies a password identical to the current one', () => {
    expect(classifyPasswordError(axiosError(400, 'new password must differ from the current one'))).toBe(
      'sameAsCurrent',
    )
  })

  it('classifies an incorrect current password', () => {
    expect(classifyPasswordError(axiosError(400, 'current password is incorrect'))).toBe('currentIncorrect')
  })

  it('classifies an invalid or expired reset token', () => {
    expect(classifyPasswordError(axiosError(400, 'invalid or expired reset token'))).toBe('invalidToken')
  })

  it('stays robust to a wrapping prefix on the server message', () => {
    expect(
      classifyPasswordError(axiosError(400, 'reset password transaction: validation: password is too common')),
    ).toBe('tooCommon')
  })

  it('classifies a conflict (existing account) from the status code', () => {
    expect(classifyPasswordError(axiosError(409, 'username or email already in use'))).toBe('conflict')
  })

  it('falls back to generic for an unrecognised 400', () => {
    expect(classifyPasswordError(axiosError(400, 'something else'))).toBe('generic')
  })

  it.each([axiosError(500), axiosError(), new Error('offline')])(
    'classifies server and network failures as unavailable',
    (error) => {
      expect(classifyPasswordError(error)).toBe('network')
    },
  )
})
