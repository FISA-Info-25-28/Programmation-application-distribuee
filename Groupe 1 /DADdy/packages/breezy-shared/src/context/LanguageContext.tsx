import { useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { type Locale, translations } from '../i18n/translations'
import { LanguageContext } from './language-context'

// deepMerge superpose les traductions d'une locale sur une base (le français), de
// sorte qu'une clé absente d'une locale retombe sur la valeur française plutôt que
// de planter (undefined.x). Pattern i18n standard de repli.
function deepMerge<T>(base: T, override: unknown): T {
  if (override == null || typeof override !== 'object') return base
  const out: Record<string, unknown> = { ...(base as Record<string, unknown>) }
  for (const [k, o] of Object.entries(override as Record<string, unknown>)) {
    const b = (base as Record<string, unknown>)?.[k]
    out[k] =
      b && typeof b === 'object' && !Array.isArray(b) && o && typeof o === 'object'
        ? deepMerge(b, o)
        : o
  }
  return out as T
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const stored = localStorage.getItem('breezy_locale') as Locale | null
    return stored && stored in translations ? stored : 'fr'
  })

  const setLocale = (l: Locale) => {
    setLocaleState(l)
    localStorage.setItem('breezy_locale', l)
  }

  // Repli sur le français pour toute clé manquante dans la locale sélectionnée.
  const t = useMemo(() => deepMerge(translations.fr, translations[locale]), [locale])

  return (
    <LanguageContext value={{ locale, setLocale, t }}>
      {children}
    </LanguageContext>
  )
}
