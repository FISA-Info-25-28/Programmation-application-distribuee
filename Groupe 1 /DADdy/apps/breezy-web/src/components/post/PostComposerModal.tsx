import { useState, useEffect, useRef, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { X, ImagePlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { UserAvatar } from '@/components/UserAvatar'
import { createPost } from '@/api/posts'
import { uploadMedia, getMediaUploadErrorMessage } from '@/api/media'
import { apiErrorMessage } from '@/api/errors'
import { useAuth } from '@/hooks/useAuth'
import { useUser } from '@/hooks/useUser'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import { MentionDropdown } from './MentionDropdown'
import { useLanguage } from '@/hooks/useLanguage'
import { processMediaFile } from '@/lib/media'
import { MediaPreviewGrid, type MediaItem } from './MediaPreview'

const MAX = 280
const MAX_MEDIA = 4

interface PostComposerModalProps {
  open: boolean
  onClose: () => void
}

export function PostComposerModal({ open, onClose }: PostComposerModalProps) {
  const { t } = useLanguage()
  const { user } = useAuth()
  const { data: profile } = useUser(user?.id ?? '')
  const avatarUrl = profile?.avatarUrl ?? user?.avatarUrl ?? null
  const [content, setContent] = useState('')
  const [media, setMedia] = useState<MediaItem[]>([])
  const [loading, setLoading] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const qc = useQueryClient()
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const mention = useMentionAutocomplete(content, setContent)

  const remaining = MAX - content.length
  const uploadedIds = media.filter((m) => m.id != null).map((m) => m.id!)
  const canPost =
    (content.trim().length > 0 || uploadedIds.length > 0) &&
    content.length <= MAX &&
    !media.some((m) => m.uploading)

  const resetComposer = useCallback(() => {
    media.forEach((m) => URL.revokeObjectURL(m.previewUrl))
    setContent('')
    mention.resetMentions()
    setMedia([])
    setSubmitError('')
  }, [media, mention])

  const handleClose = useCallback(() => {
    resetComposer()
    onClose()
  }, [onClose, resetComposer])

  useEffect(() => {
    if (open) {
      setTimeout(() => textareaRef.current?.focus(), 50)
    }
  }, [open])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') handleClose() }
    if (open) document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, handleClose])

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const val = e.target.value
    const cursor = e.target.selectionStart ?? val.length
    setContent(val)
    mention.onContentChange(val, cursor)
  }


  const handleFiles = async (files: FileList | null) => {
    if (!files) return
    const rawFiles = Array.from(files).slice(0, MAX_MEDIA - media.length)
    if (rawFiles.length === 0) return

    const processed = await Promise.all(rawFiles.map(processMediaFile))

    const items: MediaItem[] = processed.map((p, i) => ({
      key: `${rawFiles[i].name}-${rawFiles[i].size}-${Date.now()}`,
      previewUrl: p.previewUrl,
      isVideo: p.isVideo,
      isAudio: p.isAudio,
      uploading: true,
      error: false,
      unsupported: p.unsupported,
    }))
    setMedia((prev) => [...prev, ...items])

    await Promise.all(
      processed.map(async (p, i) => {
        const key = items[i].key
        try {
          const result = await uploadMedia(p.file)
          setMedia((prev) =>
            prev.map((m) => (m.key === key ? { ...m, id: result.id, uploading: false } : m))
          )
        } catch (err) {
          const message = getMediaUploadErrorMessage(err)
          setMedia((prev) =>
            prev.map((m) =>
              m.key === key ? { ...m, uploading: false, error: true, errorMessage: message } : m
            )
          )
        }
      })
    )
  }

  const removeMedia = (key: string) => {
    setMedia((prev) => {
      const item = prev.find((m) => m.key === key)
      if (item) URL.revokeObjectURL(item.previewUrl)
      return prev.filter((m) => m.key !== key)
    })
  }

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!canPost) return
    setLoading(true)
    setSubmitError('')
    try {
      await createPost(content.trim(), user?.username ?? '', mention.mentions, uploadedIds)
      qc.invalidateQueries({ queryKey: ['feed'] })
      qc.invalidateQueries({ queryKey: ['search-posts'] })
      if (user) qc.invalidateQueries({ queryKey: ['user-posts', user.id] })
      handleClose()
    } catch (err) {
      setSubmitError(apiErrorMessage(err, 'Publication impossible. Réessayez dans un instant.'))
    } finally {
      setLoading(false)
    }
  }

  if (!open || !user) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" aria-modal="true" role="dialog">
      <div className="absolute inset-0 bg-background/20 backdrop-blur-[2px]" onClick={handleClose} />

      <div className="relative z-10 w-full max-w-lg mx-4 rounded-2xl border border-border bg-card shadow-xl">
        <div className="flex items-center justify-between px-4 pt-4 pb-3 border-b border-border">
          <p className="text-sm font-semibold text-foreground">{t.post.newPost}</p>
          <button
            onClick={handleClose}
            className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="px-4 py-4">
          <div className="flex gap-3">
            <div className="shrink-0 mt-0.5">
              <UserAvatar username={user.username} avatarUrl={avatarUrl} size={36} />
            </div>

            <div className="flex-1 min-w-0 relative">
              <Textarea
                ref={textareaRef}
                placeholder={t.post.placeholder}
                value={content}
                onChange={handleChange}
                rows={4}
                className="resize-none border-none shadow-none p-0 text-sm focus-visible:ring-0 placeholder:text-muted-foreground/60 bg-transparent"
              />

              <MentionDropdown
                active={mention.mentionActive}
                suggestions={mention.suggestions}
                isLoading={mention.isLoading}
                topOffset={mention.dropdownTop}
                onSelect={(user) => {
                  mention.selectUser(user)
                  textareaRef.current?.focus()
                }}
              />

              {media.length > 0 && (
                <MediaPreviewGrid items={media} onRemove={removeMedia} />
              )}

              {submitError && (
                <p className="mt-2 text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2">
                  {submitError}
                </p>
              )}

              <div className="flex items-center justify-between mt-3 pt-3 border-t border-border">
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={media.length >= MAX_MEDIA}
                    className="p-1.5 rounded-lg text-muted-foreground hover:text-primary hover:bg-accent transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                    title="Add photo or video"
                  >
                    <ImagePlus size={17} />
                  </button>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept="image/*,video/*,audio/*"
                    multiple
                    className="hidden"
                    onChange={(e) => handleFiles(e.target.files)}
                    onClick={(e) => { (e.target as HTMLInputElement).value = '' }}
                  />
                  <span className={`text-xs tabular-nums ${remaining < 20 ? 'text-destructive' : 'text-muted-foreground'}`}>
                    {remaining}
                  </span>
                </div>
                <div className="flex gap-2">
                  <Button type="button" variant="ghost" size="sm" onClick={handleClose}>
                    {t.common.cancel}
                  </Button>
                  <Button type="submit" size="sm" disabled={!canPost || loading}>
                    {loading ? t.post.publishing : t.post.publish}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
