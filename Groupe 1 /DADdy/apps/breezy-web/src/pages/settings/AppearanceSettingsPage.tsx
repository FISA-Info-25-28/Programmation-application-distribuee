import { Moon, Sun, Check } from 'lucide-react'
import { useLanguage } from '@/hooks/useLanguage'
import { useTheme } from '@/hooks/useTheme'
import type { Theme } from '@/context/theme-context'

function SettingSection({ title, hint, children }: { title: string; hint?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-2xl border border-border bg-card p-5">
      <h2 className="text-sm font-semibold text-foreground">{title}</h2>
      {hint && <p className="mt-1 mb-4 text-xs text-muted-foreground">{hint}</p>}
      {!hint && <div className="mt-4" />}
      {children}
    </section>
  )
}

interface ThemeCardProps {
  value: Theme
  current: Theme
  icon: React.ReactNode
  label: string
  onSelect: (t: Theme) => void
}

function ThemeCard({ value, current, icon, label, onSelect }: ThemeCardProps) {
  const active = value === current
  return (
    <button
      type="button"
      onClick={() => onSelect(value)}
      className={`flex-1 flex flex-col items-center gap-3 py-5 rounded-xl border-2 transition-all ${
        active
          ? 'border-primary bg-primary/10'
          : 'border-border hover:border-border/80 hover:bg-accent'
      }`}
    >
      <div className={`flex items-center justify-center w-10 h-10 rounded-full ${
        active ? 'bg-primary/20 text-primary' : 'bg-muted text-muted-foreground'
      }`}>
        {icon}
      </div>
      <span className={`text-sm font-medium ${active ? 'text-primary' : 'text-foreground'}`}>
        {label}
      </span>
      {active && (
        <div className="flex items-center justify-center w-5 h-5 rounded-full bg-primary">
          <Check size={11} className="text-primary-foreground" />
        </div>
      )}
    </button>
  )
}

export function AppearanceSettingsPage() {
  const { t } = useLanguage()
  const { theme, toggleTheme } = useTheme()

  const selectTheme = (selected: Theme) => {
    if (selected !== theme) toggleTheme()
  }

  return (
    <div className="px-4 py-6 max-w-lg space-y-4">
      <SettingSection title={t.settings.themeTitle} hint={t.settings.themeHint}>
        <div className="flex gap-3">
          <ThemeCard
            value="dark"
            current={theme}
            icon={<Moon size={20} />}
            label={t.settings.themeDark}
            onSelect={selectTheme}
          />
          <ThemeCard
            value="light"
            current={theme}
            icon={<Sun size={20} />}
            label={t.settings.themeLight}
            onSelect={selectTheme}
          />
        </div>
      </SettingSection>
    </div>
  )
}
