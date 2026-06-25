import { describe, expect, it } from 'vitest'
import { isStaff } from './roles'

describe('isStaff', () => {
  it.each([
    ['moderator', true],
    ['administrator', true],
    ['admin', true],
    ['user', false],
    [null, false],
    [undefined, false],
  ])('isStaff(%s) → %s', (role, expected) => {
    expect(isStaff(role as string)).toBe(expected)
  })
})
