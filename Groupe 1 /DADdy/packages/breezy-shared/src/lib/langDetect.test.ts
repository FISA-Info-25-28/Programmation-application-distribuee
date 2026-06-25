import { describe, expect, it } from 'vitest'
import { detectLang } from './langDetect'

describe('detectLang', () => {
  it('returns null for empty text', () => {
    expect(detectLang('')).toBeNull()
  })

  it('returns null for text shorter than 8 chars', () => {
    expect(detectLang('bonjou')).toBeNull()
  })

  it('detects Arabic', () => {
    expect(detectLang('مرحبا بالعالم هذا نص عربي طويل')).toBe('ar')
  })

  it('detects Hebrew', () => {
    expect(detectLang('שלום עולם זה טקסט עברי ארוך')).toBe('iw')
  })

  it('detects Russian', () => {
    expect(detectLang('Привет мир это русский текст длинный')).toBe('ru')
  })

  it('detects Japanese', () => {
    expect(detectLang('こんにちは世界これは日本語のテキストです')).toBe('ja')
  })

  it('detects Korean', () => {
    expect(detectLang('안녕하세요 세계 이것은 한국어 텍스트입니다')).toBe('ko')
  })

  it('detects unambiguous French', () => {
    expect(
      detectLang('le chat est dans la maison avec les amis et les enfants'),
    ).toBe('fr')
  })

  it('detects unambiguous English', () => {
    expect(
      detectLang('the cat is in the house with the friends and their children'),
    ).toBe('en')
  })

  it('returns null for Latin text with fewer than 3 words', () => {
    // 11 chars (passes length check) but only 2 words → null
    expect(detectLang('hello world')).toBeNull()
  })

  it('returns null when text has no Latin words (only digits/symbols)', () => {
    // Passes length check but match() returns null → words = [] < 3 words
    expect(detectLang('123456789')).toBeNull()
  })

  it('returns null when Latin words exist but no stop word matches any language', () => {
    // 3 real words, none are stop words → all scores = 0 → null
    expect(detectLang('dog cat rabbit fish')).toBeNull()
  })

  it('returns null for ambiguous text with equal scores', () => {
    // le/et/mais → fr; the/and/but → en; equal scores → null
    expect(detectLang('le the et and mais but hello')).toBeNull()
  })
})
