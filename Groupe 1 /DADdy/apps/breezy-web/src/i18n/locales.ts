import type { Locale } from '@/i18n/translations'
import { FR, GB, ES, BR, IT, DE, RU, SA, IL, JP, KR } from 'country-flag-icons/react/3x2'

/**
 * SVG flag components are used instead of flag emoji because Windows fonts
 * (Segoe UI Emoji) don't render regional-indicator pairs as flags — they fall
 * back to the country letters (FR, GB, …). SVGs render identically everywhere.
 */
export interface LocaleOption {
  code: Locale
  /** SVG flag component (3x2 ratio) for the language. */
  Flag: typeof FR
  label: string
}

export const LOCALES: LocaleOption[] = [
  { code: 'fr', Flag: FR, label: 'Français' },
  { code: 'en', Flag: GB, label: 'English' },
  { code: 'es', Flag: ES, label: 'Español' },
  { code: 'pt', Flag: BR, label: 'Português' },
  { code: 'it', Flag: IT, label: 'Italiano' },
  { code: 'de', Flag: DE, label: 'Deutsch' },
  { code: 'ru', Flag: RU, label: 'Русский' },
  { code: 'ar', Flag: SA, label: 'العربية' },
  { code: 'he', Flag: IL, label: 'עברית' },
  { code: 'ja', Flag: JP, label: '日本語' },
  { code: 'ko', Flag: KR, label: '한국어' },
]
