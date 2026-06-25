import { useLanguage } from '@/hooks/useLanguage'
import { LOCALES } from '@/i18n/locales'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

interface LanguageSwitcherProps {
  className?: string
}

export function LanguageSwitcher({ className = '' }: LanguageSwitcherProps) {
  const { locale, setLocale } = useLanguage()
  const current = LOCALES.find((l) => l.code === locale) ?? LOCALES[0]

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="Change language"
        className={`w-8 h-8 rounded-lg flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-accent/60 transition-colors ${className}`}
      >
        <current.Flag title={current.label} className="w-5 h-auto rounded-[2px] ring-1 ring-border/60" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-[160px] max-h-80 overflow-y-auto">
        {LOCALES.map(({ code, Flag, label }) => (
          <DropdownMenuItem
            key={code}
            onClick={() => setLocale(code)}
            className={`gap-2 cursor-pointer ${locale === code ? 'text-primary font-medium' : ''}`}
          >
            <Flag title={label} className="w-5 h-auto rounded-[2px] ring-1 ring-border/60 shrink-0" />
            <span>{label}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
