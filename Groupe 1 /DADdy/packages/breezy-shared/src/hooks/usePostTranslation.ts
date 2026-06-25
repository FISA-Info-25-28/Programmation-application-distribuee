import { useState, useRef, useEffect } from 'react'
import { translateTexts, LOCALE_TO_LANG } from '../api/translate'
import { useLanguage } from './useLanguage'
import { detectLang } from '../lib/langDetect'
import type { Locale } from '../i18n/translations'

type Status = 'idle' | 'loading' | 'done' | 'error'

interface CachedTranslation {
  segments: string[]
  detectedLang: string | null
}

// cache: contentId → { locale → { segments: [content, ...extras], detectedLang } }
const cache = new Map<string, Map<Locale, CachedTranslation>>()

/**
 * Translates a post's content along with optional extra segments (e.g. poll options).
 * The first slot is always the post content; `extraTexts` follow in order.
 */
export function usePostTranslation(contentId: string, originalText: string, extraTexts: string[] = []) {
  const { locale } = useLanguage()
  const [status, setStatus] = useState<Status>('idle')
  const [translated, setTranslated] = useState<string[] | null>(null)
  const [sourceLang, setSourceLang] = useState<string | null>(null)
  const [showOriginal, setShowOriginal] = useState(false)
  const translatedLocale = useRef<Locale | null>(null)

  const segments = [originalText, ...extraTexts]

  // Offer translation only when the text appears to be in a different language.
  // detectLang returns null when uncertain → show button to be safe.
  // Combine all segments so a poll with empty content still gets a fair guess.
  const detectedLang = detectLang(segments.join(' '))
  const shouldOffer =
    segments.some((s) => s.trim().length > 0) &&
    (detectedLang === null || detectedLang !== LOCALE_TO_LANG[locale])

  // Reset whenever the user switches language while a translation is displayed
  useEffect(() => {
    if (translatedLocale.current !== null && translatedLocale.current !== locale) {
      setStatus('idle')
      setTranslated(null)
      setSourceLang(null)
      setShowOriginal(false)
      translatedLocale.current = null
    }
  }, [locale])

  async function translate() {
    // Serve from cache if available
    const postCache = cache.get(contentId)
    if (postCache?.has(locale)) {
      const cached = postCache.get(locale)!
      setTranslated(cached.segments)
      setSourceLang(cached.detectedLang)
      translatedLocale.current = locale
      setShowOriginal(false)
      setStatus('done')
      return
    }

    setStatus('loading')
    setShowOriginal(false)
    try {
      const result = await translateTexts(segments, locale)
      if (!cache.has(contentId)) cache.set(contentId, new Map())
      cache.get(contentId)!.set(locale, { segments: result.texts, detectedLang: result.detectedLang })

      setTranslated(result.texts)
      setSourceLang(result.detectedLang)
      translatedLocale.current = locale
      setStatus('done')
    } catch {
      setStatus('error')
    }
  }

  function toggleOriginal() {
    setShowOriginal((v) => !v)
  }

  function reset() {
    setStatus('idle')
    setTranslated(null)
    setSourceLang(null)
    setShowOriginal(false)
    translatedLocale.current = null
  }

  const active = status === 'done' && !showOriginal && translated !== null

  return {
    status,
    shouldOffer,
    showOriginal,
    translate,
    toggleOriginal,
    reset,
    displayText: active ? translated![0] : null,
    displayExtras: active ? translated!.slice(1) : null,
    sourceLang,
    sourceLangName: localizedLangName(sourceLang, locale),
  }
}

// Localized human-readable name of an ISO language code, in the current UI locale.
// Returns null when unknown or unsupported by the runtime.
function localizedLangName(code: string | null, locale: Locale): string | null {
  if (!code) return null
  // Google returns the legacy "iw" code for Hebrew; Intl expects "he".
  const normalized = code === 'iw' ? 'he' : code
  try {
    return new Intl.DisplayNames([locale], { type: 'language' }).of(normalized) ?? null
  } catch {
    return null
  }
}
