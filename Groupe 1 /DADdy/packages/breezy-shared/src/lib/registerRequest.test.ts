import { describe, expect, it } from 'vitest'
import { createRegisterRequest } from './registerRequest'
import { TERMS_VERSION } from './terms'

describe('createRegisterRequest', () => {
  it('includes all provided fields and current terms version', () => {
    const req = createRegisterRequest('alice', 'alice@example.com', 'S3cur3!Pass', true)
    expect(req.username).toBe('alice')
    expect(req.email).toBe('alice@example.com')
    expect(req.password).toBe('S3cur3!Pass')
    expect(req.termsAccepted).toBe(true)
    expect(req.termsVersion).toBe(TERMS_VERSION)
  })

  it('preserves termsAccepted false', () => {
    const req = createRegisterRequest('bob', 'bob@example.com', 'P4ss!word', false)
    expect(req.termsAccepted).toBe(false)
  })
})
