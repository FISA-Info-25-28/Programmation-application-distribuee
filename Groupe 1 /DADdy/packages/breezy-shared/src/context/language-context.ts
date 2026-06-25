import { createContext } from 'react'
import { type Locale, translations } from '../i18n/translations'

interface LanguageContextValue {
  locale: Locale
  setLocale: (l: Locale) => void
  t: (typeof translations)[Locale]
}

const LanguageContext = createContext<LanguageContextValue | null>(null)

export { LanguageContext }
