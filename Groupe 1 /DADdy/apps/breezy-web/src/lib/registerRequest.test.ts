import { describe, expect, it } from 'vitest'
import { createRegisterRequest } from './registerRequest'
import { TERMS_VERSION } from './terms'

describe('createRegisterRequest', () => {
  it('includes the displayed terms acceptance and version', () => {
    expect(createRegisterRequest('alice', 'alice@example.com', 'Secure1!', true)).toEqual({
      username: 'alice',
      email: 'alice@example.com',
      password: 'Secure1!',
      termsAccepted: true,
      termsVersion: TERMS_VERSION,
    })
  })
})
