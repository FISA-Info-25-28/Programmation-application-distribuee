import { useQueryClient } from '@tanstack/react-query'
import { useLanguage } from '@/hooks/useLanguage'
import { useAuth } from '@/hooks/useAuth'
import { useUser } from '@/hooks/useUser'
import { ChangePasswordForm } from '@/components/settings/ChangePasswordForm'
import { MFASection } from '@/components/settings/MFASection'

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

export function SecuritySettingsPage() {
  const { t } = useLanguage()
  const { user: me } = useAuth()
  const { data: profile } = useUser(me?.id ?? '')
  const queryClient = useQueryClient()

  const handleMFAToggle = () => {
    queryClient.invalidateQueries({ queryKey: ['me'] })
  }

  return (
    <div className="px-4 py-6 max-w-lg space-y-4">
      <SettingSection title={t.settings.passwordTitle} hint={t.settings.passwordHint}>
        <ChangePasswordForm />
      </SettingSection>
      <MFASection mfaEnabled={profile?.mfaEnabled ?? false} onToggle={handleMFAToggle} />
    </div>
  )
}
