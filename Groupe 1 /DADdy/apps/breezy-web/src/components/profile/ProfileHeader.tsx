import { useState, useRef } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Pencil, Check, X, Camera, Loader2, ImagePlus, Globe, Bell, BellOff, MessageCircle, Lock, MoreHorizontal, ShieldOff, Shield, Settings, Bookmark } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { UserAvatar } from '@/components/UserAvatar'
import { useAuth } from '@/hooks/useAuth'
import { useFollowMutation, useUpdateMe, useBlockMutation } from '@/hooks/useUser'
import { useSubscription } from '@/hooks/useNotifications'
import { useLanguage } from '@/hooks/useLanguage'
import { usePostTranslation } from '@/hooks/usePostTranslation'
import { uploadAvatar, uploadBanner } from '@/api/users'
import { ImageCropper } from './ImageCropper'
import type { User } from '@/types/api'

interface ProfileHeaderProps {
  user: User
}

export function ProfileHeader({ user }: ProfileHeaderProps) {
  const { t } = useLanguage()
  const { user: me } = useAuth()
  const isMe = me?.id === user.id
  const navigate = useNavigate()
  const qc = useQueryClient()

  const handleMessage = () => {
    qc.setQueryData(['user', user.id], user)
    navigate(`/messages?with=${user.id}`)
  }

  const {
    status: bioStatus,
    shouldOffer: bioShouldOffer,
    displayText: bioDisplayText,
    sourceLangName: bioSourceLangName,
    showOriginal: bioShowOriginal,
    translate: bioTranslate,
    toggleOriginal: bioToggleOriginal,
    reset: bioReset,
  } = usePostTranslation(`bio-${user.id}`, user.bio ?? '')

  const [followed, setFollowed] = useState(user.isFollowedByMe ?? false)
  const [requested, setRequested] = useState(user.followRequested ?? false)
  const [blocked, setBlocked] = useState(user.isBlockedByMe ?? false)
  const [menuOpen, setMenuOpen] = useState(false)
  const followMutation = useFollowMutation(user.id)
  const blockMutation = useBlockMutation(user.id)
  const { isSubscribed, toggle: toggleSubscription, isPending: subPending } = useSubscription(isMe ? '' : user.id)

  const [editing, setEditing] = useState(false)
  const [displayName, setDisplayName] = useState(user.displayName ?? '')
  const [pronouns, setPronouns] = useState(user.pronouns ?? '')
  const [bio, setBio] = useState(user.bio ?? '')
  const [avatarUrl, setAvatarUrl] = useState(user.avatarUrl ?? '')
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [bannerUrl, setBannerUrl] = useState(user.bannerUrl ?? '')
  const [bannerPreview, setBannerPreview] = useState<string | null>(null)
  const [bannerUploading, setBannerUploading] = useState(false)
  const [crop, setCrop] = useState<{ file: File; kind: 'avatar' | 'banner' } | null>(null)

  const updateMe = useUpdateMe()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const bannerInputRef = useRef<HTMLInputElement>(null)

  const handleFollow = () => {
    if (followed) {
      setFollowed(false)
      followMutation.mutate(false)
    } else if (requested) {
      // Annule la demande d'abonnement en attente.
      setRequested(false)
      followMutation.mutate(false)
    } else if (user.isPrivate) {
      // Compte privé : crée une demande au lieu d'un abonnement direct.
      setRequested(true)
      followMutation.mutate(true)
    } else {
      setFollowed(true)
      followMutation.mutate(true)
    }
  }

  const followLabel = followed
    ? t.common.followingBtn
    : requested
      ? t.profile.requested
      : t.common.follow

  const handleBlock = () => {
    const next = !blocked
    setBlocked(next)
    setMenuOpen(false)
    blockMutation.mutate(next)
    if (next && followed) {
      setFollowed(false)
      followMutation.mutate(false)
    }
  }

  const handleAvatarClick = () => {
    if (editing) fileInputRef.current?.click()
  }

  // La sélection d'un fichier n'uploade pas directement : elle ouvre le cropper.
  // L'upload est lancé sur le fichier recadré renvoyé par handleCropConfirm.
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (file) setCrop({ file, kind: 'avatar' })
  }

  const handleBannerClick = () => {
    if (editing) bannerInputRef.current?.click()
  }

  const handleBannerChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (file) setCrop({ file, kind: 'banner' })
  }

  const uploadAvatarFile = async (file: File) => {
    const preview = URL.createObjectURL(file)
    setAvatarPreview(preview)
    setUploading(true)
    try {
      const url = await uploadAvatar(file)
      setAvatarUrl(url)
      URL.revokeObjectURL(preview)
      setAvatarPreview(null)
    } catch {
      // garde la prévisualisation locale en fallback
    } finally {
      setUploading(false)
    }
  }

  const uploadBannerFile = async (file: File) => {
    const preview = URL.createObjectURL(file)
    setBannerPreview(preview)
    setBannerUploading(true)
    try {
      const url = await uploadBanner(file)
      setBannerUrl(url)
      URL.revokeObjectURL(preview)
      setBannerPreview(null)
    } catch {
      // garde la prévisualisation locale en fallback
    } finally {
      setBannerUploading(false)
    }
  }

  const handleCropConfirm = (file: File) => {
    const kind = crop?.kind
    setCrop(null)
    if (kind === 'avatar') void uploadAvatarFile(file)
    else if (kind === 'banner') void uploadBannerFile(file)
  }

  const handleSave = async () => {
    await updateMe.mutateAsync({
      // null (et non undefined) pour que la clé soit présente dans le JSON :
      // le back interprète une valeur vide comme une demande de remise à NULL.
      displayName: displayName.trim() || null,
      pronouns: pronouns.trim() || null,
      bio: bio.trim() || null,
    })
    setAvatarPreview(null)
    setBannerPreview(null)
    setEditing(false)
  }

  const handleCancelEdit = () => {
    if (avatarPreview) URL.revokeObjectURL(avatarPreview)
    if (bannerPreview) URL.revokeObjectURL(bannerPreview)
    setDisplayName(user.displayName ?? '')
    setPronouns(user.pronouns ?? '')
    setBio(user.bio ?? '')
    setAvatarUrl(user.avatarUrl ?? '')
    setAvatarPreview(null)
    setBannerUrl(user.bannerUrl ?? '')
    setBannerPreview(null)
    setEditing(false)
  }

  const displayAvatar = avatarPreview ?? avatarUrl
  const displayBanner = bannerPreview ?? bannerUrl

  return (
    <div className="border-b border-border">
      <div className="relative aspect-[7/2] overflow-hidden bg-gradient-to-br from-primary/20 via-primary/10 to-accent/30">
        {displayBanner && (
          <img src={displayBanner} alt="" className="absolute inset-0 h-full w-full object-cover" />
        )}
        {editing && isMe && (
          <>
            <button
              type="button"
              onClick={handleBannerClick}
              disabled={bannerUploading}
              className="absolute inset-0 w-full flex items-center justify-center gap-2 text-xs font-medium text-white bg-black/40 opacity-0 hover:opacity-100 disabled:opacity-100 transition-opacity disabled:cursor-not-allowed"
              title={t.profile.addBanner}
            >
              {bannerUploading ? (
                <Loader2 size={18} className="animate-spin" />
              ) : (
                <>
                  <ImagePlus size={16} />
                  {t.profile.addBanner}
                </>
              )}
            </button>
            <input
              ref={bannerInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleBannerChange}
            />
          </>
        )}
      </div>

      <div className="px-4 pb-6">
        <div className="flex items-end justify-between gap-4 -mt-8 mb-3">
          <div className="relative shrink-0">
            <button
              type="button"
              onClick={handleAvatarClick}
              disabled={!editing || uploading}
              className={`block rounded-full ring-4 ring-card transition-all ${
                editing ? 'cursor-pointer group' : 'cursor-default'
              }`}
              aria-label={editing ? t.profile.edit : undefined}
            >
              <UserAvatar
                username={user.username}
                avatarUrl={displayAvatar || null}
                size={72}
                className={editing && !uploading ? 'group-hover:brightness-75 transition-[filter]' : ''}
              />
              {editing && (
                <span className="absolute inset-0 flex items-center justify-center rounded-full bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                  {uploading
                    ? <Loader2 size={22} className="text-white animate-spin" />
                    : <Camera size={22} className="text-white" />
                  }
                </span>
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

          <div className="flex items-center gap-2 pb-1">
            {isMe ? (
              editing ? (
                <>
                  <Button variant="ghost" size="sm" onClick={handleCancelEdit} className="gap-1.5">
                    <X size={14} /> {t.common.cancel}
                  </Button>
                  <Button
                    size="sm"
                    onClick={handleSave}
                    disabled={updateMe.isPending || uploading || bannerUploading}
                    className="gap-1.5"
                  >
                    {updateMe.isPending ? <Loader2 size={14} className="animate-spin" /> : <Check size={14} />}
                    {t.profile.save}
                  </Button>
                </>
              ) : (
                <>
                  {/* Settings + Bookmarks — mobile only, inaccessibles sans sidebar */}
                  <Link
                    to="/bookmarks"
                    className="lg:hidden p-2 rounded-full border border-border text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                    aria-label="Favoris"
                  >
                    <Bookmark size={16} />
                  </Link>
                  <Link
                    to="/settings"
                    className="lg:hidden p-2 rounded-full border border-border text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                    aria-label="Paramètres"
                  >
                    <Settings size={16} />
                  </Link>
                  <Button variant="outline" size="sm" onClick={() => setEditing(true)} className="gap-1.5">
                    <Pencil size={14} /> {t.profile.edit}
                  </Button>
                </>
              )
            ) : (
              <>
                {!blocked && (
                  <button
                    onClick={handleMessage}
                    className="p-2 rounded-full border border-border text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                    title="Envoyer un message"
                  >
                    <MessageCircle size={16} />
                  </button>
                )}
                {!blocked && (
                  <button
                    onClick={toggleSubscription}
                    disabled={subPending}
                    title={isSubscribed ? t.profile.notifyOff : t.profile.notifyOn}
                    className={`p-2 rounded-full border transition-colors disabled:opacity-50 ${
                      isSubscribed
                        ? 'border-primary text-primary hover:bg-primary/10'
                        : 'border-border text-muted-foreground hover:text-foreground hover:border-foreground/40'
                    }`}
                  >
                    {isSubscribed ? <Bell size={15} fill="currentColor" /> : <BellOff size={15} />}
                  </button>
                )}
                {!blocked && (
                  <Button
                    variant={followed || requested ? 'outline' : 'default'}
                    size="sm"
                    onClick={handleFollow}
                    disabled={followMutation.isPending}
                  >
                    {followLabel}
                  </Button>
                )}
                <div className="relative">
                  <button
                    onClick={() => setMenuOpen((v) => !v)}
                    className="p-2 rounded-full border border-border text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                    title="Plus d'options"
                  >
                    <MoreHorizontal size={16} />
                  </button>
                  {menuOpen && (
                    <>
                      <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
                      <div className="absolute right-0 top-full mt-1 z-20 min-w-[160px] rounded-xl border border-border bg-popover shadow-lg py-1 overflow-hidden">
                        <button
                          onClick={handleBlock}
                          disabled={blockMutation.isPending}
                          className={`w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors disabled:opacity-50 ${
                            blocked
                              ? 'text-foreground hover:bg-accent'
                              : 'text-destructive hover:bg-destructive/10'
                          }`}
                        >
                          {blocked ? <Shield size={14} /> : <ShieldOff size={14} />}
                          {blocked ? 'Débloquer' : 'Bloquer'}
                        </button>
                      </div>
                    </>
                  )}
                </div>
              </>
            )}
          </div>
        </div>

        {editing ? (
          <div className="mt-2 space-y-2">
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={t.profile.displayNamePlaceholder}
              maxLength={50}
              aria-label={t.profile.displayName}
            />
            <Input
              value={pronouns}
              onChange={(e) => setPronouns(e.target.value)}
              placeholder={t.profile.pronounsPlaceholder}
              maxLength={50}
              aria-label={t.profile.pronouns}
            />
            <Textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder={t.profile.bioPlaceholder}
              rows={3}
              maxLength={160}
              className="resize-none text-sm"
            />
            <p className="text-xs text-muted-foreground">@{user.username}</p>
          </div>
        ) : (
          <>
            <div className="flex items-baseline gap-2 flex-wrap">
              <h2 className="font-semibold text-foreground flex items-center gap-1.5">
                {user.displayName || `@${user.username}`}
                {user.isPrivate && (
                  <Lock
                    size={14}
                    className="text-muted-foreground shrink-0"
                    aria-label={t.profile.privateAccount}
                  >
                    <title>{t.profile.privateAccount}</title>
                  </Lock>
                )}
              </h2>
              {user.pronouns && (
                <span className="text-xs text-muted-foreground">{user.pronouns}</span>
              )}
            </div>
            {user.displayName && (
              <p className="text-sm text-muted-foreground">@{user.username}</p>
            )}
            {user.bio ? (
            <div className="mt-1">
            <p className="text-sm text-muted-foreground leading-relaxed">
              {bioDisplayText ?? user.bio}
            </p>
            <div className="mt-1 flex items-center gap-2 flex-wrap">
              {bioShouldOffer && bioStatus === 'idle' && (
                <button
                  onClick={() => bioTranslate()}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-primary transition-colors"
                >
                  <Globe size={12} />
                  {t.post.translate}
                </button>
              )}
              {bioStatus === 'loading' && (
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Loader2 size={12} className="animate-spin" />
                  {t.post.translating}
                </span>
              )}
              {bioStatus === 'done' && (
                <>
                  <span className="flex items-center gap-1 text-xs text-muted-foreground/60">
                    <Globe size={11} />
                    {bioSourceLangName ? `${t.post.translatedFrom} ${bioSourceLangName}` : t.post.translated}
                  </span>
                  <span className="text-xs text-muted-foreground/40">·</span>
                  <button
                    onClick={bioToggleOriginal}
                    className="text-xs text-primary/70 hover:text-primary transition-colors"
                  >
                    {bioShowOriginal ? t.post.showTranslation : t.post.showOriginal}
                  </button>
                </>
              )}
              {bioStatus === 'error' && (
                <>
                  <span className="text-xs text-destructive/70">{t.post.translationError}</span>
                  <button
                    onClick={bioReset}
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors"
                  >
                    {t.post.retry}
                  </button>
                </>
              )}
            </div>
          </div>
            ) : null}
          </>
        )}

        <div className="flex gap-6 mt-4">
          <Link to={`/profile/${user.id}/following`} className="text-sm hover:underline">
            <span className="font-semibold text-foreground">{user.followingCount}</span>
            <span className="text-muted-foreground ml-1">{t.common.following}</span>
          </Link>
          <Link to={`/profile/${user.id}/followers`} className="text-sm hover:underline">
            <span className="font-semibold text-foreground">{user.followersCount}</span>
            <span className="text-muted-foreground ml-1">{t.common.followers}</span>
          </Link>
        </div>
      </div>

      {crop && (
        <ImageCropper
          file={crop.file}
          aspect={crop.kind === 'avatar' ? 1 : 7 / 2}
          round={crop.kind === 'avatar'}
          outputWidth={crop.kind === 'avatar' ? 512 : 1500}
          title={t.profile.cropTitle}
          zoomLabel={t.profile.cropZoom}
          applyLabel={t.profile.cropApply}
          cancelLabel={t.common.cancel}
          hint={t.profile.cropHint}
          onCancel={() => setCrop(null)}
          onConfirm={handleCropConfirm}
        />
      )}
    </div>
  )
}

export function ProfileHeaderSkeleton() {
  return (
    <div className="border-b border-border">
      <div className="aspect-[7/2] bg-gradient-to-br from-primary/20 via-primary/10 to-accent/30" />
      <div className="px-4 pb-6">
        <div className="flex items-end justify-between gap-4 -mt-8 mb-3">
          <Skeleton className="w-[72px] h-[72px] rounded-full ring-4 ring-card shrink-0" />
        </div>
        <div className="space-y-2">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-3 w-48" />
        </div>
        <div className="flex gap-6 mt-4">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-3 w-20" />
        </div>
      </div>
    </div>
  )
}
