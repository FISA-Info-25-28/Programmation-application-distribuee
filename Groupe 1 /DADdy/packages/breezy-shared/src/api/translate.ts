import type { Locale } from '../i18n/translations'

export const LOCALE_TO_LANG: Record<Locale, string> = {
  fr: 'fr', en: 'en', es: 'es', pt: 'pt', it: 'it',
  de: 'de', ru: 'ru', ar: 'ar', he: 'iw', ja: 'ja', ko: 'ko',
}

export interface TranslationResult {
  text: string
  /** Source language detected by Google (ISO code, e.g. "en"), or null if unknown. */
  detectedLang: string | null
}

// Google Translate unofficial endpoint — auto-detects source language,
// no API key required, returns segments as nested arrays.
export async function translateText(text: string, targetLocale: Locale): Promise<TranslationResult> {
  const tl = LOCALE_TO_LANG[targetLocale]
  const url =
    `https://translate.googleapis.com/translate_a/single` +
    `?client=gtx&sl=auto&tl=${tl}&dt=t&q=${encodeURIComponent(text)}`

  const res = await fetch(url)
  if (!res.ok) throw new Error(`Translation error: ${res.status}`)

  const data = await res.json()
  // data[0] is an array of [translatedSegment, originalSegment, …] tuples
  const translated = (data[0] as [string, ...unknown[]][])
    .map((seg) => seg[0])
    .filter(Boolean)
    .join('')

  if (!translated) throw new Error('Empty translation response')
  // data[2] holds the detected source language code.
  const detectedLang = typeof data[2] === 'string' ? data[2] : null
  return { text: translated, detectedLang }
}

// Translate several independent segments in parallel (e.g. post content + poll options).
// Empty/blank segments are passed through untouched so callers keep index alignment.
// The detected source language is taken from the first non-empty segment.
export async function translateTexts(
  texts: string[],
  targetLocale: Locale,
): Promise<{ texts: string[]; detectedLang: string | null }> {
  const results = await Promise.all(
    texts.map((text) =>
      text.trim()
        ? translateText(text, targetLocale)
        : Promise.resolve<TranslationResult>({ text, detectedLang: null }),
    ),
  )
  const detectedLang = results.find((r) => r.detectedLang)?.detectedLang ?? null
  return { texts: results.map((r) => r.text), detectedLang }
}
