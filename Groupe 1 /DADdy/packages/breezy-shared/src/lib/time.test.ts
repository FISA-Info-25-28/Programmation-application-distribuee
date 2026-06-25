import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { vi } from 'vitest'
import type { Locale } from '../i18n/translations'
import { timeAgo, timeLabel } from './time'

const t = { justNow: 'justNow', timeMin: 'min', timeHour: 'h', timeDay: 'j' }

describe('timeAgo', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('diff < 1 min → justNow', () => {
    vi.setSystemTime(new Date('2024-01-01T00:00:30Z'))
    expect(timeAgo('2024-01-01T00:00:00Z', t, 'fr')).toBe('justNow')
  })

  it.each([
    [5, '5min'],
    [59, '59min'],
  ])('diff %d min → %s', (minutes, expected) => {
    vi.setSystemTime(new Date(Date.UTC(2024, 0, 1, 0, minutes, 0)))
    expect(timeAgo('2024-01-01T00:00:00Z', t, 'fr')).toBe(expected)
  })

  it.each([
    [1, '1h'],
    [23, '23h'],
  ])('diff %d h → %s', (hours, expected) => {
    vi.setSystemTime(new Date(Date.UTC(2024, 0, 1, hours, 0, 0)))
    expect(timeAgo('2024-01-01T00:00:00Z', t, 'fr')).toBe(expected)
  })

  it.each([
    [1, '1j'],
    [6, '6j'],
  ])('diff %d day(s) → %s', (days, expected) => {
    vi.setSystemTime(new Date(Date.UTC(2024, 0, 1 + days, 0, 0, 0)))
    expect(timeAgo('2024-01-01T00:00:00Z', t, 'fr')).toBe(expected)
  })

  it.each<Locale>(['fr', 'en', 'ja', 'ar'])(
    'diff ≥ 7 days → toLocaleDateString for locale %s',
    (locale) => {
      vi.setSystemTime(new Date('2024-01-15T00:00:00Z'))
      const result = timeAgo('2024-01-01T00:00:00Z', t, locale)
      expect(typeof result).toBe('string')
      expect(result.length).toBeGreaterThan(0)
    },
  )
})

describe('timeLabel', () => {
  it('formats HH:MM for fr-FR', () => {
    const label = timeLabel('2024-01-01T14:30:00Z', 'fr')
    expect(label).toMatch(/\d{1,2}[: ]\d{2}/)
  })

  it('formats HH:MM for en-US', () => {
    const label = timeLabel('2024-01-01T14:30:00Z', 'en')
    expect(label).toMatch(/\d{1,2}:\d{2}/)
  })
})
