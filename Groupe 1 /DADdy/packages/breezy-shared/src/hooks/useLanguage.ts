import { use } from 'react'
import { LanguageContext } from '../context/language-context'

export function useLanguage() {
  const ctx = use(LanguageContext)
  if (!ctx) throw new Error('useLanguage must be used inside LanguageProvider')
  return ctx
}
