import { useState, useRef, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { ImagePlus, BarChart2 } from 'lucide-react'
import { MediaPreviewGrid, type MediaItem } from './MediaPreview'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { UserAvatar } from '@/components/UserAvatar'
import { createPost, type PollDraft } from '@/api/posts'
import { uploadMedia, getMediaUploadErrorMessage } from '@/api/media'
import { apiErrorMessage } from '@/api/errors'
import { useAuth } from '@/hooks/useAuth'
import { useUser } from '@/hooks/useUser'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import { MentionDropdown } from './MentionDropdown'
import { PollComposer } from './PollComposer'
import { QuotePreview } from './QuotePreview'
import { useLanguage } from '@/hooks/useLanguage'
import { processMediaFile } from '@/lib/media'
import type { Post } from '@/types/api'

const MAX = 280
const MAX_MEDIA = 4
const DEFAULT_POLL: PollDraft = { options: ['', ''], durationDays: 1 }

function CharCounter({ remaining }: { remaining: number }) {
  const used = MAX - remaining
  if (used === 0) return null
  const r = 9
  const circ = 2 * Math.PI * r
  const pct = Math.min(used / MAX, 1)
  const dash = circ * (1 - pct)
  const over = remaining < 0
  const warn = remaining < 20
  const stroke = over ? 'var(--color-destructive)' : warn ? 'oklch(0.75 0.18 60)' : 'var(--color-primary)'
  return (
    <div className="relative flex items-center justify-center w-6 h-6 shrink-0">
      <svg width="24" height="24" viewBox="0 0 24 24" style={{ transform: 'rotate(-90deg)' }}>
        <circle cx="12" cy="12" r={r} fill="none" stroke="currentColor" className="text-muted-foreground/20" strokeWidth="2" />
        <circle cx="12" cy="12" r={r} fill="none" stroke={stroke} strokeWidth="2"
          strokeDasharray={circ} strokeDashoffset={dash}
          strokeLinecap="round" style={{ transition: 'stroke-dashoffset 0.15s ease, stroke 0.15s ease' }} />
      </svg>
      {warn && (
        <span className={`absolute text-[9px] font-semibold tabular-nums leading-none ${over ? 'text-destructive' : 'text-amber-500'}`}>
          {remaining}
        </span>
      )}
    </div>
  )
}


interface PostComposerProps {
  quotedPost?: Post
  onSuccess?: () => void
}

export function PostComposer({ quotedPost, onSuccess }: PostComposerProps = {}) {
  const { t } = useLanguage()
  const { user } = useAuth()
  const { data: profile } = useUser(user?.id ?? '')
  const avatarUrl = profile?.avatarUrl ?? user?.avatarUrl ?? null
  const [content, setContent] = useState('')
  const [media, setMedia] = useState<MediaItem[]>([])
  const [poll, setPoll] = useState<PollDraft | null>(null)
  const [loading, setLoading] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const [focused, setFocused] = useState(false)
  const [placeholderIdx, setPlaceholderIdx] = useState(0)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const PLACEHOLDERS = [t.post.placeholder, 'Quoi de neuf ?', 'Partagez quelque chose…', 'Exprimez-vous…']
  useEffect(() => {
    if (!focused || content.length > 0) return
    const id = setInterval(() => setPlaceholderIdx((i) => (i + 1) % PLACEHOLDERS.length), 2500)
    return () => clearInterval(id)
  }, [focused, content.length, PLACEHOLDERS.length])
  const fileInputRef = useRef<HTMLInputElement>(null)
  const qc = useQueryClient()

  const mention = useMentionAutocomplete(content, setContent)

  const remaining = MAX - content.length
  const uploadedIds = media.filter((m) => m.id != null).map((m) => m.id!)
  const pollValid = poll === null || (
    poll.options.length >= 2 &&
    poll.options.every((o) => o.trim().length > 0 && o.length <= 25)
  )
  const canPost =
    (content.trim().length > 0 || uploadedIds.length > 0 || poll !== null || !!quotedPost) &&
    content.length <= MAX &&
    !media.some((m) => m.uploading) &&
    pollValid

  const togglePoll = () => {
    if (poll) {
      setPoll(null)
    } else {
      setMedia([])
      setPoll({ ...DEFAULT_POLL, options: ['', ''] })
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const val = e.target.value
    const cursor = e.target.selectionStart ?? val.length
    setContent(val)
    mention.onContentChange(val, cursor)
    e.target.style.height = 'auto'
    e.target.style.height = `${e.target.scrollHeight}px`
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
      await createPost(content.trim(), user?.username ?? '', mention.mentions, uploadedIds, poll ?? undefined, quotedPost?.id)
      setContent('')
      mention.resetMentions()
      media.forEach((m) => URL.revokeObjectURL(m.previewUrl))
      setMedia([])
      setPoll(null)
      qc.invalidateQueries({ queryKey: ['feed'] })
      qc.invalidateQueries({ queryKey: ['search-posts'] })
      if (user) qc.invalidateQueries({ queryKey: ['user-posts', user.id] })
      onSuccess?.()
    } catch (err) {
      setSubmitError(apiErrorMessage(err, 'Publication impossible. Réessayez dans un instant.'))
    } finally {
      setLoading(false)
    }
  }

  if (!user) return null

  return (
    <form onSubmit={handleSubmit} className="px-4 py-4 border-b border-border">
      <div className="flex gap-3">
        <div className={`shrink-0 mt-0.5 rounded-full transition-all duration-200 ${focused || content.length > 0 ? 'ring-2 ring-primary/60 ring-offset-2 ring-offset-background' : ''}`}>
          <UserAvatar username={user.username} avatarUrl={avatarUrl} size={36} />
        </div>

        <div className="flex-1 min-w-0 relative">
          <Textarea
            ref={textareaRef}
            placeholder={PLACEHOLDERS[placeholderIdx]}
            value={content}
            onChange={handleChange}
            onFocus={() => setFocused(true)}
            onBlur={() => { setFocused(false); setPlaceholderIdx(0) }}
            rows={1}
            className="resize-none border-none shadow-none p-0 text-sm focus-visible:ring-0 placeholder:text-muted-foreground/60 bg-transparent overflow-hidden transition-all"
            style={{ minHeight: '3rem' }}
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

          {poll && (
            <PollComposer
              poll={poll}
              onChange={setPoll}
              onRemove={() => setPoll(null)}
            />
          )}

          {quotedPost && <QuotePreview post={quotedPost} />}

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
                disabled={media.length >= MAX_MEDIA || poll !== null}
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
              <button
                type="button"
                onClick={togglePoll}
                disabled={media.length > 0}
                className={`p-1.5 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
                  poll
                    ? 'text-primary bg-primary/10'
                    : 'text-muted-foreground hover:text-primary hover:bg-accent'
                }`}
                title={t.poll.add}
              >
                <BarChart2 size={17} />
              </button>
              <CharCounter remaining={remaining} />
            </div>
            <Button type="submit" size="sm" disabled={!canPost || loading}>
              {loading ? t.post.publishing : t.post.publish}
            </Button>
          </div>
        </div>
      </div>
    </form>
  )
}
