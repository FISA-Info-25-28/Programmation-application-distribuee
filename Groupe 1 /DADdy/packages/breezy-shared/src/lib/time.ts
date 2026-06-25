import type { Locale } from '../i18n/translations'

const LOCALE_TO_BCP47: Record<Locale, string> = {
  fr: 'fr-FR', en: 'en-US', es: 'es-ES', pt: 'pt-BR',
  it: 'it-IT', de: 'de-DE', ru: 'ru-RU', ar: 'ar-SA',
  he: 'he-IL', ja: 'ja-JP', ko: 'ko-KR',
}

interface TimeStrings {
  justNow: string
  timeMin: string
  timeHour: string
  timeDay: string
}

export function timeAgo(date: string, t: TimeStrings, locale: Locale): string {
  const diff = Date.now() - new Date(date).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return t.justNow
  if (m < 60) return `${m}${t.timeMin}`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}${t.timeHour}`
  const d = Math.floor(h / 24)
  if (d < 7) return `${d}${t.timeDay}`
  return new Date(date).toLocaleDateString(LOCALE_TO_BCP47[locale], { day: 'numeric', month: 'short' })
}

export function timeLabel(date: string, locale: Locale): string {
  return new Date(date).toLocaleTimeString(LOCALE_TO_BCP47[locale], { hour: '2-digit', minute: '2-digit' })
}
