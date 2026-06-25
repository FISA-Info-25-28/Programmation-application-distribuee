import { Link } from 'react-router-dom'
import { Shield } from 'lucide-react'
import { UserAvatar } from '@/components/UserAvatar'
import { useAuth } from '@/hooks/useAuth'
import { useUser, useUpdateMe, useBlockedUsers, useBlockMutation } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import type { User } from '@/types/api'

function SettingSection({ title, hint, children }: { title: string; hint?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-2xl border border-border bg-card p-5">
      <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">{title}</h2>
      {hint && <p className="mt-1 mb-4 text-xs text-muted-foreground">{hint}</p>}
      {!hint && <div className="mt-4" />}
      {children}
    </section>
  )
}

function PrivacySection() {
  const { t } = useLanguage()
  const { user: me } = useAuth()
  const { data: profile } = useUser(me?.id ?? '')
  const updateMe = useUpdateMe()
  const isPrivate = profile?.isPrivate ?? false

  const toggle = () => {
    if (updateMe.isPending) return
    updateMe.mutate({ isPrivate: !isPrivate })
  }

  return (
    <SettingSection title={`🔒 ${t.settings.privacyTitle}`}>
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t.settings.privateAccount}</p>
          <p className="mt-1 text-xs text-muted-foreground">{t.settings.privateAccountHint}</p>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={isPrivate}
          aria-label={t.settings.privateAccount}
          onClick={toggle}
          disabled={updateMe.isPending || !profile}
          className={`relative h-6 w-11 shrink-0 rounded-full transition-colors disabled:opacity-50 ${
            isPrivate ? 'bg-primary' : 'bg-muted-foreground/30'
          }`}
        >
          <span
            className={`absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${
              isPrivate ? 'translate-x-5' : 'translate-x-0'
            }`}
          />
        </button>
      </div>
    </SettingSection>
  )
}

function BlockedUserRow({ user }: { user: User }) {
  const { t } = useLanguage()
  const mutation = useBlockMutation(user.id)

  return (
    <div className="flex items-center gap-3 py-3 border-b border-border last:border-0">
      <Link to={`/profile/${user.id}`} className="shrink-0">
        <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={36} />
      </Link>
      <div className="flex-1 min-w-0">
        <Link to={`/profile/${user.id}`} className="block hover:underline">
          <p className="text-sm font-medium text-foreground truncate">
            {user.displayName || `@${user.username}`}
          </p>
          {user.displayName && (
            <p className="text-xs text-muted-foreground truncate">@{user.username}</p>
          )}
        </Link>
      </div>
      <button
        onClick={() => mutation.mutate(false)}
        disabled={mutation.isPending}
        className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border text-muted-foreground hover:text-foreground hover:bg-accent transition-colors disabled:opacity-50"
      >
        <Shield size={12} />
        {t.settings.unblock}
      </button>
    </div>
  )
}

function BlockedUsersSection() {
  const { t } = useLanguage()
  const { data, isLoading } = useBlockedUsers()
  const users = data?.data ?? []

  return (
    <SettingSection title={t.settings.blockedTitle} hint={t.settings.blockedHint}>
      {isLoading ? (
        <div className="py-4 text-center text-xs text-muted-foreground">{t.settings.loading}</div>
      ) : users.length === 0 ? (
        <div className="py-6 flex flex-col items-center gap-2 text-muted-foreground">
          <Shield size={24} className="opacity-30" />
          <p className="text-xs">{t.settings.blockedEmpty}</p>
        </div>
      ) : (
        <div>
          {users.map((u) => (
            <BlockedUserRow key={u.id} user={u} />
          ))}
        </div>
      )}
    </SettingSection>
  )
}

export function PrivacySettingsPage() {
  return (
    <div className="px-4 py-6 max-w-lg space-y-4">
      <PrivacySection />
      <BlockedUsersSection />
    </div>
  )
}
