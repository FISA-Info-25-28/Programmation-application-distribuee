import { useState, useRef } from 'react'
import { Camera, Loader2, User, Mail, Calendar, Shield, Globe, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { UserAvatar } from '@/components/UserAvatar'
import { ImageCropper } from '@/components/profile/ImageCropper'
import { useAuth } from '@/hooks/useAuth'
import { useUser, useUpdateMe } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import { uploadAvatar } from '@/api/users'
import { LOCALES } from '@/i18n/locales'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const ROLE_LABELS: Record<string, string> = {
  user: 'Utilisateur',
  moderator: 'Modérateur',
  administrator: 'Administrateur',
  admin: 'Administrateur',
}

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

export function AccountSettingsPage() {
  const { t, locale, setLocale } = useLanguage()
  const { user: me } = useAuth()
  const { data: profile } = useUser(me?.id ?? '')
  const updateMe = useUpdateMe()

  const [displayName, setDisplayName] = useState(me?.displayName ?? '')
  const [pronouns, setPronouns] = useState(me?.pronouns ?? '')
  const [bio, setBio] = useState(me?.bio ?? '')
  const [saved, setSaved] = useState(false)

  const [avatarPreview, setAvatarPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [cropFile, setCropFile] = useState<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [deleteConfirmInput, setDeleteConfirmInput] = useState('')

  const currentAvatar = avatarPreview ?? profile?.avatarUrl ?? me?.avatarUrl ?? null
  const currentLocale = LOCALES.find((l) => l.code === locale) ?? LOCALES[0]

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (file) setCropFile(file)
  }

  const handleCropConfirm = async (file: File) => {
    setCropFile(null)
    const preview = URL.createObjectURL(file)
    setAvatarPreview(preview)
    setUploading(true)
    try {
      await uploadAvatar(file)
      URL.revokeObjectURL(preview)
      setAvatarPreview(null)
    } catch {
      // keep local preview as fallback
    } finally {
      setUploading(false)
    }
  }

  const handleSave = async () => {
    await updateMe.mutateAsync({
      displayName: displayName.trim() || null,
      pronouns: pronouns.trim() || null,
      bio: bio.trim() || null,
    })
    setSaved(true)
    setTimeout(() => setSaved(false), 2500)
  }

  const memberSince = me?.createdAt
    ? new Date(me.createdAt).toLocaleDateString(undefined, { month: 'long', year: 'numeric', day: 'numeric' })
    : '—'

  return (
    <div className="px-4 py-6 max-w-lg space-y-4">

      {/* Profile edit */}
      <SettingSection title={t.settings.editProfileTitle}>
        <div className="flex items-center gap-4 mb-5">
          <div className="relative shrink-0">
            <UserAvatar
              username={me?.username ?? ''}
              avatarUrl={currentAvatar}
              size={72}
            />
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
              className="absolute inset-0 flex items-center justify-center rounded-full bg-black/50 opacity-0 hover:opacity-100 disabled:opacity-100 transition-opacity"
            >
              {uploading ? (
                <Loader2 size={16} className="text-white animate-spin" />
              ) : (
                <Camera size={16} className="text-white" />
              )}
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleFileChange}
            />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-semibold text-foreground">@{me?.username}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{me?.email}</p>
          </div>
        </div>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {t.profile.displayName}
            </label>
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={t.profile.displayNamePlaceholder}
              maxLength={50}
              className="h-10 bg-input border-border focus:border-primary/60 rounded-xl"
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {t.profile.pronouns}
            </label>
            <Input
              value={pronouns}
              onChange={(e) => setPronouns(e.target.value)}
              placeholder={t.profile.pronounsPlaceholder}
              maxLength={30}
              className="h-10 bg-input border-border focus:border-primary/60 rounded-xl"
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Bio</label>
            <Textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder={t.profile.bioPlaceholder}
              maxLength={300}
              rows={3}
              className="bg-input border-border focus:border-primary/60 rounded-xl resize-none"
            />
            <p className="text-xs text-muted-foreground text-right">{bio.length}/300</p>
          </div>

          <Button
            onClick={handleSave}
            disabled={updateMe.isPending}
            className="w-full h-10 rounded-xl font-semibold"
          >
            {updateMe.isPending
              ? t.settings.saving
              : saved
                ? t.settings.savedSuccess
                : t.settings.saveChanges}
          </Button>
        </div>
      </SettingSection>

      {/* Account info (read-only) */}
      <SettingSection title={t.settings.accountInfoTitle}>
        <div className="space-y-3">
          <InfoRow icon={<User size={14} />} label={t.settings.usernameLabel} value={`@${me?.username ?? '—'}`} />
          <InfoRow icon={<Mail size={14} />} label={t.settings.emailLabel} value={me?.email ?? '—'} />
          <InfoRow icon={<Calendar size={14} />} label={t.settings.memberSinceLabel} value={memberSince} />
          <InfoRow
            icon={<Shield size={14} />}
            label={t.settings.roleLabel}
            value={
              <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                me?.role === 'user'
                  ? 'bg-muted text-muted-foreground'
                  : 'bg-primary/15 text-primary'
              }`}>
                {ROLE_LABELS[me?.role ?? 'user'] ?? me?.role}
              </span>
            }
          />
        </div>
      </SettingSection>

      {/* Language */}
      <SettingSection title={t.settings.languageTitle} hint={t.settings.languageHint}>
        <DropdownMenu>
          <DropdownMenuTrigger className="flex items-center gap-3 w-full px-3 py-2.5 rounded-xl border border-border bg-input hover:bg-accent transition-colors text-sm font-medium text-foreground">
            <Globe size={16} className="text-muted-foreground shrink-0" />
            <currentLocale.Flag title={currentLocale.label} className="w-6 h-auto rounded-[2px] ring-1 ring-border/60 shrink-0" />
            <span>{currentLocale.label}</span>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="min-w-[200px] max-h-72 overflow-y-auto">
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
      </SettingSection>

      {/* Danger zone */}
      <SettingSection title={t.settings.dangerZoneTitle}>
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="text-sm font-medium text-foreground">{t.settings.deleteAccount}</p>
            <p className="mt-1 text-xs text-muted-foreground">{t.settings.deleteAccountHint}</p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setDeleteModalOpen(true)}
            className="shrink-0 text-destructive hover:text-destructive hover:bg-destructive/10 border border-destructive/30 rounded-xl"
          >
            <Trash2 size={14} className="mr-1.5" />
            {t.settings.deleteAccount}
          </Button>
        </div>
      </SettingSection>

      {/* Avatar cropper */}
      {cropFile && (
        <ImageCropper
          file={cropFile}
          aspect={1}
          round
          outputWidth={400}
          title={t.profile.cropTitle}
          zoomLabel={t.profile.cropZoom}
          applyLabel={t.profile.cropApply}
          cancelLabel={t.common.cancel}
          hint={t.profile.cropHint}
          onConfirm={handleCropConfirm}
          onCancel={() => setCropFile(null)}
        />
      )}

      {/* Delete account modal */}
      {deleteModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
          onClick={() => setDeleteModalOpen(false)}
        >
          <div
            className="w-full max-w-sm rounded-2xl border border-border bg-card p-6 shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-3 mb-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-destructive/15">
                <Trash2 size={18} className="text-destructive" />
              </div>
              <h3 className="text-base font-semibold text-foreground">{t.settings.deleteAccountConfirmTitle}</h3>
            </div>
            <p className="text-sm text-muted-foreground mb-4">{t.settings.deleteAccountHint}</p>
            <p className="text-xs font-medium text-foreground mb-2">{t.settings.deleteAccountConfirmText}</p>
            <Input
              value={deleteConfirmInput}
              onChange={(e) => setDeleteConfirmInput(e.target.value)}
              placeholder={`@${me?.username}`}
              className="h-10 bg-input border-border rounded-xl mb-4"
            />
            <div className="flex gap-2">
              <Button
                variant="ghost"
                className="flex-1 rounded-xl"
                onClick={() => { setDeleteModalOpen(false); setDeleteConfirmInput('') }}
              >
                {t.settings.deleteAccountCancelBtn}
              </Button>
              <Button
                disabled={deleteConfirmInput !== me?.username}
                className="flex-1 rounded-xl bg-destructive hover:bg-destructive/90 text-destructive-foreground"
              >
                {t.settings.deleteAccountConfirmBtn}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function InfoRow({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-3 py-2 border-b border-border last:border-0">
      <div className="flex items-center gap-2 text-xs text-muted-foreground shrink-0">
        {icon}
        {label}
      </div>
      <div className="text-sm text-foreground font-medium truncate text-right">{value}</div>
    </div>
  )
}
