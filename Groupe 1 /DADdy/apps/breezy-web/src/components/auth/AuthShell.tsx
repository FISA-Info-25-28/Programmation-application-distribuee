import type { ReactNode } from 'react'
import { BreezyLogo } from '@/components/BreezyLogo'
import { ThemeToggle } from '@/components/ThemeToggle'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'

/**
 * Coquille commune des écrans d'authentification : logo à gauche, panneau de
 * formulaire à droite avec le sélecteur de thème. Le contenu de la carte est
 * passé en children.
 */
export function AuthShell({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle?: ReactNode
  children: ReactNode
}) {
  return (
    <div
      className="flex bg-transparent transition-colors duration-200"
      style={{ zIndex: 1, minHeight: '100dvh' }}
    >
      {/* ── Left — logo (desktop only) ── */}
      <div className="hidden lg:flex flex-1 flex-col">
        <div className="px-10 pt-8">
          <BreezyLogo variant="text" size={28} />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <BreezyLogo variant="icon" size={275} className="drop-shadow-2xl" />
        </div>
      </div>

      {/* ── Right — form panel, natural document flow so keyboard scroll works ── */}
      <div
        className="w-full lg:w-[520px] flex flex-col overflow-x-hidden transition-colors duration-200"
        style={{
          paddingTop: 'env(safe-area-inset-top)',
          paddingBottom: 'env(safe-area-inset-bottom)',
        }}
      >
        {/* Controls row */}
        <div className="flex items-center justify-between px-5 pt-4 pb-2">
          <div className="lg:hidden">
            <BreezyLogo variant="text" size={20} />
          </div>
          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
            <LanguageSwitcher />
          </div>
        </div>

        {/* Form */}
        <div className="flex-1 flex flex-col justify-center px-5 py-4">
          <div className="w-full max-w-sm mx-auto lg:mx-0">
            <div className="mb-5">
              <h2 className="text-2xl font-bold text-foreground">{title}</h2>
              {subtitle && <p className="text-muted-foreground text-sm mt-1">{subtitle}</p>}
            </div>

            <div className="rounded-2xl border border-border bg-card px-5 py-5 shadow-lg shadow-black/8 ring-1 ring-primary/12">
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
