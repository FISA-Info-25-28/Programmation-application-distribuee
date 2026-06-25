import { useState, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Link, useNavigate } from 'react-router-dom'
import { Globe, Loader2, Repeat2, X, ChevronLeft, ChevronRight, AlertTriangle } from 'lucide-react'
import { PostActions } from './PostActions'
import { LinkPreview } from './LinkPreview'
import { PollDisplay } from './PollDisplay'
import { shouldOpenPostFromTarget } from './postCardNavigation'
import { QuotePreview } from './QuotePreview'
import { PostComposer } from './PostComposer'
import { UserAvatar } from '@/components/UserAvatar'
import { UserHoverCard } from '@/components/UserHoverCard'
import { useAuth } from '@/hooks/useAuth'
import { useUser } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import { usePostTranslation } from '@/hooks/usePostTranslation'
import { timeAgo } from '@/lib/time'
import type { Post, PostMedia } from '@/types/api'

const RTL_LOCALES = new Set(['ar', 'he'])

function PostContent({
  content,
  mentions,
}: {
  content: string
  mentions: Array<{ id: string; username: string }>
}) {
  const mentionMap = Object.fromEntries((mentions ?? []).map((m) => [m.username.toLowerCase(), m.id]))
  const tokenRe = /#([\p{L}\p{N}_]{1,100})|@([a-zA-Z0-9_]{1,50})/gu
  const parts: React.ReactNode[] = []
  let last = 0
  let match: RegExpExecArray | null

  while ((match = tokenRe.exec(content)) !== null) {
    if (match.index > last) parts.push(content.slice(last, match.index))

    if (match[1] !== undefined) {
      const tag = match[1].toLowerCase()
      parts.push(
        <Link
          key={`h-${match.index}`}
          to={`/hashtag/${tag}`}
          onClick={(e) => e.stopPropagation()}
          className="text-primary/80 hover:text-primary transition-colors"
        >
          {match[0]}
        </Link>
      )
    } else {
      const username = match[2]
      const userId = mentionMap[username.toLowerCase()]
      parts.push(
        userId ? (
          <UserHoverCard key={`m-${match.index}`} userId={userId}>
            <Link
              to={`/profile/${userId}`}
              onClick={(e) => e.stopPropagation()}
              className="text-primary/80 hover:text-primary transition-colors"
            >
              {match[0]}
            </Link>
          </UserHoverCard>
        ) : (
          <span key={`m-${match.index}`} className="text-primary/70">
            {match[0]}
          </span>
        )
      )
    }
    last = match.index + match[0].length
  }
  if (last < content.length) parts.push(content.slice(last))

  return <p className="text-sm text-foreground/90 leading-relaxed whitespace-pre-wrap">{parts}</p>
}

// ── Media grid + lightbox ─────────────────────────────────────────────────────


function MediaFallback({ label }: { label: string }) {
  return (
    <div className="w-full h-full flex flex-col items-center justify-center gap-2 bg-accent text-muted-foreground">
      <AlertTriangle size={20} />
      <span className="text-xs text-center px-3">{label}</span>
    </div>
  )
}

function ImageLightbox({
  images,
  initialIndex,
  onClose,
}: {
  images: PostMedia[]
  initialIndex: number
  onClose: () => void
}) {
  const [index, setIndex] = useState(initialIndex)
  const [failedImageUrl, setFailedImageUrl] = useState<string | null>(null)
  const currentImage = images[index]
  const imgError = failedImageUrl === currentImage.url

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowRight') setIndex((i) => Math.min(i + 1, images.length - 1))
      if (e.key === 'ArrowLeft') setIndex((i) => Math.max(i - 1, 0))
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onClose, images.length])

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90"
      onClick={onClose}
    >
      <button
        className="absolute top-4 right-4 p-2 text-white/70 hover:text-white transition-colors"
        onClick={onClose}
        aria-label="Fermer"
      >
        <X size={28} />
      </button>

      {index > 0 && (
        <button
          className="absolute left-4 p-2 text-white/70 hover:text-white transition-colors"
          onClick={(e) => { e.stopPropagation(); setIndex((i) => i - 1) }}
          aria-label="Précédent"
        >
          <ChevronLeft size={36} />
        </button>
      )}

      {imgError ? (
        <div className="flex flex-col items-center justify-center gap-3 text-white/60" onClick={(e) => e.stopPropagation()}>
          <AlertTriangle size={32} />
          <span className="text-sm">Format non supporté par ce navigateur</span>
        </div>
      ) : (
        <img
          src={currentImage.url}
          alt=""
          className="max-w-full max-h-full object-contain select-none"
          onClick={(e) => e.stopPropagation()}
          draggable={false}
          onError={() => setFailedImageUrl(currentImage.url)}
        />
      )}

      {index < images.length - 1 && (
        <button
          className="absolute right-4 p-2 text-white/70 hover:text-white transition-colors"
          onClick={(e) => { e.stopPropagation(); setIndex((i) => i + 1) }}
          aria-label="Suivant"
        >
          <ChevronRight size={36} />
        </button>
      )}

      {images.length > 1 && (
        <div className="absolute bottom-4 flex gap-1.5">
          {images.map((_, i) => (
            <div
              key={i}
              className={`w-1.5 h-1.5 rounded-full transition-colors ${i === index ? 'bg-white' : 'bg-white/40'}`}
            />
          ))}
        </div>
      )}
    </div>,
    document.body,
  )
}

function ImageWithFallback({ src, onClick }: { src: string; onClick: () => void }) {
  const [error, setError] = useState(false)
  if (error) return <MediaFallback label="Format non supporté" />
  return (
    <img
      src={src}
      alt=""
      className="w-full h-full object-cover cursor-zoom-in"
      loading="lazy"
      onClick={onClick}
      onError={() => setError(true)}
    />
  )
}

function VideoWithFallback({ src }: { src: string }) {
  const [error, setError] = useState(false)
  if (error) return <MediaFallback label="Format vidéo non supporté" />
  return (
    <video
      src={src}
      className="w-full h-full object-contain"
      controls
      preload="metadata"
      onError={() => setError(true)}
    />
  )
}

function AudioWithFallback({ src }: { src: string }) {
  const [error, setError] = useState(false)
  if (error) return <MediaFallback label="Format audio non supporté" />
  return (
    <div className="w-full h-full flex items-center bg-accent px-3">
      <audio
        src={src}
        className="w-full"
        controls
        preload="metadata"
        onError={() => setError(true)}
      />
    </div>
  )
}

export function PostMediaGrid({ media }: { media: PostMedia[] }) {
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)
  const count = media.length
  if (count === 0) return null

  const images = media.filter((m) => m.media_type === 'image')
  const gridClass = count >= 2 ? 'grid-cols-2' : 'grid-cols-1'

  return (
    <>
      <div
        className={`grid gap-0.5 mt-2 rounded-xl overflow-hidden ${gridClass} max-h-[500px]`}
        onClick={(e) => e.stopPropagation()}
      >
        {media.map((item, i) => (
          <div
            key={item.id}
            className={`relative overflow-hidden ${count === 3 && i === 0 ? 'row-span-2' : ''} ${item.media_type === 'video' ? 'bg-black' : 'bg-accent'}`}
            style={{ aspectRatio: count === 1 ? '16/9' : '1/1' }}
          >
            {item.media_type === 'video' ? (
              <VideoWithFallback src={item.url} />
            ) : item.media_type === 'audio' ? (
              <AudioWithFallback src={item.url} />
            ) : (
              <ImageWithFallback
                src={item.url}
                onClick={() => setLightboxIndex(images.indexOf(item))}
              />
            )}
          </div>
        ))}
      </div>

      {lightboxIndex !== null && (
        <ImageLightbox
          images={images}
          initialIndex={lightboxIndex}
          onClose={() => setLightboxIndex(null)}
        />
      )}
    </>
  )
}

// ── PostCard ──────────────────────────────────────────────────────────────────

interface PostCardProps {
  post: Post & { rebreezerInfo?: { id: string; username: string } }
  inThread?: boolean
}

export function PostCard({ post, inThread = false }: PostCardProps) {
  const navigate = useNavigate()
  const { user: me } = useAuth()
  const { t, locale } = useLanguage()
  const isOwn = me?.id === post.authorId
  const [quoteOpen, setQuoteOpen] = useState(false)

  const { data: authorProfile } = useUser(post.author.id)
  const avatarUrl = authorProfile?.avatarUrl ?? post.author.avatarUrl ?? null
  // displayName par défaut = username (cf. user-service) ; fallback sur le username si absent.
  const authorName = authorProfile?.displayName?.trim() || post.author.username

  const { status, shouldOffer, displayText, displayExtras, sourceLangName, showOriginal, translate, toggleOriginal, reset } =
    usePostTranslation(post.id, post.content, post.poll?.options.map((o) => o.text) ?? [])

  const isRTL = RTL_LOCALES.has(locale)

  const [mountTime] = useState(Date.now)
  const postAgeMs = mountTime - new Date(post.createdAt).getTime()
  const isOld = postAgeMs > 6 * 60 * 60 * 1000
  const isActiveNow = postAgeMs < 30 * 60 * 1000

  const rebreezerName = post.rebreezerInfo
    ? (post.rebreezerInfo.displayName?.trim() || `@${post.rebreezerInfo.username}`)
    : null

  const handleCardClick = (event: React.MouseEvent<HTMLElement>) => {
    if (!shouldOpenPostFromTarget(event.target)) return
    navigate(`/post/${post.id}`)
  }

  return (
    <article
      onClick={handleCardClick}
      className={`px-4 border-b border-border transition-all cursor-pointer ${isOld ? 'opacity-[0.88]' : ''} ${inThread ? 'pt-1 pb-4' : 'py-4 hover:bg-accent/50 active:bg-accent/70 hover:-translate-y-px hover:shadow-[inset_3px_0_0_var(--color-primary)] hover:opacity-100'}`}
    >
      {rebreezerName && (
        <Link
          to={`/profile/${post.rebreezerInfo!.id}`}
          onClick={(e) => e.stopPropagation()}
          className="flex items-center gap-1.5 mb-1.5 text-xs text-emerald-500 hover:text-emerald-400 transition-colors"
        >
          <Repeat2 size={13} />
          <span>{rebreezerName} {t.post.rebreezeBy}</span>
        </Link>
      )}
      <div className="flex gap-3">
        {inThread ? (
          <div className="flex flex-col items-center w-8 shrink-0">
            <Link to={`/profile/${post.author.id}`} className="relative block">
              <UserAvatar username={post.author.username} avatarUrl={avatarUrl} size={32} />
              {isActiveNow && (
                <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-500 border-2 border-background" />
              )}
            </Link>
          </div>
        ) : (
          <Link to={`/profile/${post.author.id}`} className="relative block shrink-0">
            <UserAvatar username={post.author.username} avatarUrl={avatarUrl} size={36} />
            {isActiveNow && (
              <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-500 border-2 border-background" />
            )}
          </Link>
        )}

        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2 mb-1">
            <Link
              to={`/profile/${post.author.id}`}
              className="flex items-baseline gap-1.5 min-w-0 hover:underline"
            >
              <span className="font-semibold text-foreground text-sm truncate">{authorName}</span>
              <span className="text-xs text-muted-foreground truncate">@{post.author.username}</span>
            </Link>
            <span className="text-xs text-muted-foreground shrink-0">{timeAgo(post.createdAt, t.common, locale)}</span>
          </div>

          {/* Content — original or translated */}
          {displayText ? (
            <div dir={isRTL ? 'rtl' : 'ltr'}>
              <p className="text-sm text-foreground/90 leading-relaxed whitespace-pre-wrap cursor-pointer">
                {displayText}
              </p>
            </div>
          ) : (
            <PostContent content={post.content} mentions={post.mentions} />
          )}

          {(post.media ?? []).length > 0 && <PostMediaGrid media={post.media} />}
          {(post.media ?? []).length === 0 && !post.poll && !post.quotedPost && <LinkPreview content={post.content} />}

          {post.poll && (
            <PollDisplay
              poll={post.poll}
              postId={post.id}
              authorId={post.authorId}
              translatedOptions={displayExtras}
            />
          )}

          {post.quotedPost && <QuotePreview post={post.quotedPost} />}

          {/* Translation controls */}
          <div className="mt-1.5 flex items-center gap-2 flex-wrap">
            {shouldOffer && status === 'idle' && (
              <button
                onClick={() => translate()}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-primary transition-colors"
              >
                <Globe size={12} />
                {t.post.translate}
              </button>
            )}
            {status === 'loading' && (
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <Loader2 size={12} className="animate-spin" />
                {t.post.translating}
              </span>
            )}
            {status === 'done' && (
              <>
                <span className="flex items-center gap-1 text-xs text-muted-foreground/60">
                  <Globe size={11} />
                  {sourceLangName ? `${t.post.translatedFrom} ${sourceLangName}` : t.post.translated}
                </span>
                <span className="text-xs text-muted-foreground/40">·</span>
                <button
                  onClick={toggleOriginal}
                  className="text-xs text-primary/70 hover:text-primary transition-colors"
                >
                  {showOriginal ? t.post.showTranslation : t.post.showOriginal}
                </button>
              </>
            )}
            {status === 'error' && (
              <>
                <span className="text-xs text-destructive/70">{t.post.translationError}</span>
                <button onClick={reset} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
                  {t.post.retry}
                </button>
              </>
            )}
          </div>

          <PostActions post={post} isOwn={isOwn} onQuote={() => setQuoteOpen(true)} />
        </div>
      </div>

      {quoteOpen && createPortal(
        <div className="fixed inset-0 z-50 flex items-start justify-center pt-16 px-4" aria-modal="true" role="dialog">
          <div className="absolute inset-0 bg-background/40 backdrop-blur-[2px]" onClick={() => setQuoteOpen(false)} />
          <div className="relative z-10 w-full max-w-lg rounded-2xl border border-border bg-card shadow-xl overflow-hidden">
            <div className="flex items-center justify-between px-4 pt-3 pb-2 border-b border-border">
              <span className="text-sm font-semibold">{t.post.quote}</span>
              <button onClick={() => setQuoteOpen(false)} className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
                <X size={16} />
              </button>
            </div>
            <PostComposer quotedPost={post} onSuccess={() => setQuoteOpen(false)} />
          </div>
        </div>,
        document.body,
      )}
    </article>
  )
}
