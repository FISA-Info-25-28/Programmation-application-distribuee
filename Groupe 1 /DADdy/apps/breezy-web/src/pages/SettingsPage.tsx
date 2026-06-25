import { Outlet, NavLink, Navigate, useMatch } from 'react-router-dom'
import { User, Palette, Lock, Shield, Bell } from 'lucide-react'
import { useLanguage } from '@/hooks/useLanguage'

const NAV_ITEMS = [
  { path: 'account', Icon: User, labelKey: 'navAccount' as const },
  { path: 'appearance', Icon: Palette, labelKey: 'navAppearance' as const },
  { path: 'privacy', Icon: Lock, labelKey: 'navPrivacy' as const },
  { path: 'security', Icon: Shield, labelKey: 'navSecurity' as const },
  { path: 'notifications', Icon: Bell, labelKey: 'navNotifications' as const },
]

export function SettingsPage() {
  const { t } = useLanguage()
  const isIndex = useMatch('/settings') !== null

  if (isIndex) {
    return <Navigate to="/settings/account" replace />
  }

  return (
    <div className="flex min-h-screen">
      {/* Desktop left sidebar nav */}
      <aside className="hidden md:flex flex-col w-56 shrink-0 border-r border-border sticky top-0 h-screen overflow-y-auto py-6 px-3">
        <h1 className="px-3 mb-5 text-base font-semibold text-foreground">{t.settings.title}</h1>
        <nav className="flex flex-col gap-1">
          {NAV_ITEMS.map(({ path, Icon, labelKey }) => (
            <NavLink
              key={path}
              to={`/settings/${path}`}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-primary/20 text-primary border border-primary/30'
                    : 'text-muted-foreground hover:text-foreground hover:bg-accent'
                }`
              }
            >
              <Icon size={16} />
              {t.settings[labelKey]}
            </NavLink>
          ))}
        </nav>
      </aside>

      {/* Content area */}
      <div className="flex-1 min-w-0">
        {/* Mobile header with tab bar */}
        <div className="md:hidden sticky top-0 z-10 bg-background/80 backdrop-blur-sm border-b border-border">
          <div className="px-4 py-3">
            <h1 className="text-base font-semibold text-foreground">{t.settings.title}</h1>
          </div>
          <div className="flex overflow-x-auto scrollbar-none">
            {NAV_ITEMS.map(({ path, Icon, labelKey }) => (
              <NavLink
                key={path}
                to={`/settings/${path}`}
                className={({ isActive }) =>
                  `flex flex-col items-center gap-1 px-4 py-2.5 text-xs font-medium shrink-0 border-b-2 transition-colors ${
                    isActive
                      ? 'border-primary text-primary'
                      : 'border-transparent text-muted-foreground'
                  }`
                }
              >
                <Icon size={16} />
                {t.settings[labelKey]}
              </NavLink>
            ))}
          </div>
        </div>

        {/* Desktop page header */}
        <div className="hidden md:block px-4 py-4 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm">
          <h1 className="text-base font-semibold text-foreground">{t.settings.title}</h1>
        </div>

        <Outlet />
      </div>
    </div>
  )
}
