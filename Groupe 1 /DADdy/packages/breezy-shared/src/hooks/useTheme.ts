import { use } from 'react'
import { ThemeContext } from '../context/theme-context'

export function useTheme() {
  const ctx = use(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used inside ThemeProvider')
  return ctx
}
