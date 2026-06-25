import { describe, expect, it } from 'vitest'
import { cn } from './utils'

describe('cn', () => {
  it('concatenates class names', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('merges conflicting Tailwind classes (last wins)', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4')
  })

  it('ignores falsy values', () => {
    expect(cn(false && 'hidden', 'block')).toBe('block')
  })

  it('returns empty string with no arguments', () => {
    expect(cn()).toBe('')
  })
})
