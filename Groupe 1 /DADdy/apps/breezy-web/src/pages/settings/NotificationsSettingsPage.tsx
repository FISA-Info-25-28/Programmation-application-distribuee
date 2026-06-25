import { useState, useEffect } from 'react'
import { Bell, Heart, MessageCircle, UserPlus, AtSign, Repeat2, Mail } from 'lucide-react'
import { useLanguage } from '@/hooks/useLanguage'

const STORAGE_KEY = 'breezy_notif_prefs'

interface NotifPrefs {
  likes: boolean
  comments: boolean
  follows: boolean
  mentions: boolean
  reposts: boolean
  messages: boolean
}

const DEFAULT_PREFS: NotifPrefs = {
  likes: true,
  comments: true,
  follows: true,
  mentions: true,
  reposts: true,
  messages: true,
}

function loadPrefs(): NotifPrefs {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    return stored ? { ...DEFAULT_PREFS, ...JSON.parse(stored) } : DEFAULT_PREFS
  } catch {
    return DEFAULT_PREFS
  }
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: () => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={onChange}
      className={`relative h-6 w-11 shrink-0 rounded-full transition-colors ${
        checked ? 'bg-primary' : 'bg-muted-foreground/30'
      }`}
    >
      <span
        className={`absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${
          checked ? 'translate-x-5' : 'translate-x-0'
        }`}
      />
    </button>
  )
}

interface NotifRowProps {
  icon: React.ReactNode
  label: string
  hint: string
  checked: boolean
  onChange: () => void
}

function NotifRow({ icon, label, hint, checked, onChange }: NotifRowProps) {
  return (
    <div className="flex items-center gap-3 py-3.5 border-b border-border last:border-0">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
        {icon}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-foreground">{label}</p>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </div>
      <Toggle checked={checked} onChange={onChange} />
    </div>
  )
}

export function NotificationsSettingsPage() {
  const { t } = useLanguage()
  const [prefs, setPrefs] = useState<NotifPrefs>(loadPrefs)
  const [justSaved, setJustSaved] = useState(false)

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs))
    setJustSaved(true)
    const timer = setTimeout(() => setJustSaved(false), 1500)
    return () => clearTimeout(timer)
  }, [prefs])

  const toggle = (key: keyof NotifPrefs) => {
    setPrefs((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const rows: { key: keyof NotifPrefs; icon: React.ReactNode; label: string; hint: string }[] = [
    { key: 'likes', icon: <Heart size={15} />, label: t.settings.notifLikes, hint: t.settings.notifLikesHint },
    { key: 'comments', icon: <MessageCircle size={15} />, label: t.settings.notifComments, hint: t.settings.notifCommentsHint },
    { key: 'follows', icon: <UserPlus size={15} />, label: t.settings.notifFollows, hint: t.settings.notifFollowsHint },
    { key: 'mentions', icon: <AtSign size={15} />, label: t.settings.notifMentions, hint: t.settings.notifMentionsHint },
    { key: 'reposts', icon: <Repeat2 size={15} />, label: t.settings.notifReposts, hint: t.settings.notifRepostsHint },
    { key: 'messages', icon: <Mail size={15} />, label: t.settings.notifMessages, hint: t.settings.notifMessagesHint },
  ]

  return (
    <div className="px-4 py-6 max-w-lg space-y-4">
      <section className="rounded-2xl border border-border bg-card p-5">
        <div className="flex items-center justify-between gap-2 mb-1">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Bell size={15} />
            {t.settings.notificationsPageTitle}
          </h2>
          {justSaved && (
            <span className="text-xs text-muted-foreground animate-in fade-in duration-200">
              {t.settings.notifSavedLocally}
            </span>
          )}
        </div>
        <p className="text-xs text-muted-foreground mb-4">{t.settings.notificationsHint}</p>
        <div>
          {rows.map(({ key, icon, label, hint }) => (
            <NotifRow
              key={key}
              icon={icon}
              label={label}
              hint={hint}
              checked={prefs[key]}
              onChange={() => toggle(key)}
            />
          ))}
        </div>
      </section>
    </div>
  )
}
