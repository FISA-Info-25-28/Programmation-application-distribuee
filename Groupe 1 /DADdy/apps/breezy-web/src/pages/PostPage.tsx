import { useState, useRef, useEffect } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, Globe, Loader2, ChevronDown, ChevronUp, Repeat2, Heart, Trash2, MessageCircle, X, ImagePlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { PostCard, PostMediaGrid } from '@/components/post/PostCard'
import { ContextPost } from '@/components/post/ContextPost'
import { rebreezePost, unrebreezePost, likePost, unlikePost, deletePost } from '@/api/posts'
import { uploadMedia } from '@/api/media'
import { processMediaFile } from '@/lib/media'
import { useQueryClient } from '@tanstack/react-query'
import { MediaPreviewGrid, type MediaItem } from '@/components/post/MediaPreview'
import { UserAvatar } from '@/components/UserAvatar'
import { MentionDropdown } from '@/components/post/MentionDropdown'
import { UserHoverCard } from '@/components/UserHoverCard'
import { usePost, useComments, useAddComment, useCommentReplies, useAddReplyToComment } from '@/hooks/usePost'
import { useAuth } from '@/hooks/useAuth'
import { useUser } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import { usePostTranslation } from '@/hooks/usePostTranslation'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import { timeAgo } from '@/lib/time'
import type { Comment, MentionedUser } from '@/types/api'

function CommentContent({ content, mentions = [] }: { content: string; mentions?: MentionedUser[] }) {
  const mentionMap = Object.fromEntries(mentions.map((m) => [m.username.toLowerCase(), m.id]))
  const parts: React.ReactNode[] = []
  let last = 0
  let match: RegExpExecArray | null
  const tokenRe = /#([\p{L}\p{N}_]{1,100})|@([a-zA-Z0-9_]{1,50})/gu
  while ((match = tokenRe.exec(content)) !== null) {
    if (match.index > last) parts.push(content.slice(last, match.index))
    if (match[1] !== undefined) {
      parts.push(
        <Link key={`h-${match.index}`} to={`/hashtag/${match[1].toLowerCase()}`}
          onClick={(e) => e.stopPropagation()}
          className="text-primary/80 hover:text-primary transition-colors"
        >{match[0]}</Link>
      )
    } else {
      const username = match[2]
      const userId = mentionMap[username.toLowerCase()]
      parts.push(userId
        ? <UserHoverCard key={`m-${match.index}`} userId={userId}>
            <Link to={`/profile/${userId}`}
              onClick={(e) => e.stopPropagation()}
              className="text-primary/80 hover:text-primary transition-colors"
            >{match[0]}</Link>
          </UserHoverCard>
        : <span key={`m-${match.index}`} className="text-primary/70">{match[0]}</span>
      )
    }
    last = match.index + match[0].length
  }
  if (last < content.length) parts.push(content.slice(last))
  return <p className="text-sm text-foreground/90 leading-relaxed whitespace-pre-wrap">{parts}</p>
}

const RTL_LOCALES = new Set(['ar', 'he'])

function TranslationButtons({ id, content, mentions }: { id: string; content: string; mentions?: MentionedUser[] }) {
  const { t, locale } = useLanguage()
  const isRTL = RTL_LOCALES.has(locale)
  const { status, shouldOffer, displayText, sourceLangName, showOriginal, translate, toggleOriginal, reset } =
    usePostTranslation(id, content)

  return (
    <>
      {displayText ? (
        <p className="text-sm text-foreground/90 leading-relaxed" dir={isRTL ? 'rtl' : 'ltr'}>
          {displayText}
        </p>
      ) : (
        <CommentContent content={content} mentions={mentions} />
      )}
      <div className="mt-1 flex items-center gap-2 flex-wrap" onClick={(e) => e.stopPropagation()}>
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
            <button
              onClick={reset}
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              {t.post.retry}
            </button>
          </>
        )}
      </div>
    </>
  )
}

const MAX_MEDIA_REPLY = 4


function InlineReplyForm({
  avatarUrl,
  username,
  isPending,
  onSubmit,
  onCancel,
}: {
  avatarUrl: string | null
  username: string
  isPending: boolean
  onSubmit: (content: string, mentions: MentionedUser[], mediaIds: number[]) => Promise<void>
  onCancel: () => void
}) {
  const { t } = useLanguage()
  const [content, setContent] = useState('')
  const [media, setMedia] = useState<MediaItem[]>([])
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const mention = useMentionAutocomplete(content, setContent)

  const remaining = MAX_REPLY - content.length
  const uploadedIds = media.filter((m) => m.id != null).map((m) => m.id!)
  const canSubmit =
    (content.trim().length > 0 || uploadedIds.length > 0) &&
    content.length <= MAX_REPLY &&
    !media.some((m) => m.uploading) &&
    !isPending

  const handleFiles = async (files: FileList | null) => {
    if (!files) return
    const rawFiles = Array.from(files).slice(0, MAX_MEDIA_REPLY - media.length)
    if (!rawFiles.length) return
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
    await Promise.all(processed.map(async (p, i) => {
      const key = items[i].key
      try {
        const result = await uploadMedia(p.file)
        setMedia((prev) => prev.map((m) => m.key === key ? { ...m, id: result.id, uploading: false } : m))
      } catch {
        setMedia((prev) => prev.map((m) => m.key === key ? { ...m, uploading: false, error: true } : m))
      }
    }))
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
    if (!canSubmit) return
    await onSubmit(content.trim(), mention.mentions, uploadedIds)
    setContent('')
    mention.resetMentions()
    media.forEach((m) => URL.revokeObjectURL(m.previewUrl))
    setMedia([])
  }

  return (
    <form onSubmit={handleSubmit} className="mt-2 flex gap-2 items-start">
      <div className="shrink-0 mt-0.5">
        <UserAvatar username={username} avatarUrl={avatarUrl} size={20} />
      </div>
      <div className="flex-1 relative">
        <Textarea
          ref={textareaRef}
          placeholder={t.post.replyPlaceholder}
          value={content}
          onChange={(e) => {
            setContent(e.target.value)
            mention.onContentChange(e.target.value, e.target.selectionStart ?? e.target.value.length)
          }}
          rows={1}
          autoFocus
          className="resize-none border-none shadow-none p-0 text-sm focus-visible:ring-0 placeholder:text-muted-foreground/60 bg-transparent min-h-0"
        />
        <MentionDropdown
          active={mention.mentionActive}
          suggestions={mention.suggestions}
          isLoading={mention.isLoading}
          topOffset={mention.dropdownTop}
          onSelect={(user) => { mention.selectUser(user); textareaRef.current?.focus() }}
        />
        {media.length > 0 && <MediaPreviewGrid items={media} onRemove={removeMedia} />}
        <div className="flex items-center justify-between mt-1 pt-1 border-t border-border/50">
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={media.length >= MAX_MEDIA_REPLY}
              className="p-1 rounded-lg text-muted-foreground hover:text-primary hover:bg-accent transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <ImagePlus size={15} />
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*,video/*"
              multiple
              className="hidden"
              onChange={(e) => handleFiles(e.target.files)}
              onClick={(e) => { (e.target as HTMLInputElement).value = '' }}
            />
            <span className={`text-xs tabular-nums ${remaining < 20 ? 'text-destructive' : 'text-muted-foreground/50'}`}>
              {remaining}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" type="button" onClick={onCancel}>
              {t.common.cancel}
            </Button>
            <Button type="submit" size="sm" disabled={!canSubmit}>
              {t.post.reply}
            </Button>
          </div>
        </div>
      </div>
    </form>
  )
}

const MAX_REPLY = 280

function ConfirmDeleteModal({ open, onConfirm, onClose, isPending }: {
  open: boolean; onConfirm: () => void; onClose: () => void; isPending: boolean
}) {
  const { t } = useLanguage()
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    if (open) document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" aria-modal="true" role="dialog" onClick={(e) => e.stopPropagation()}>
      <div className="absolute inset-0 bg-background/20 backdrop-blur-[2px]" onClick={onClose} />
      <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl border border-border bg-card shadow-xl">
        <div className="flex items-center justify-between px-4 pt-4 pb-3 border-b border-border">
          <p className="text-sm font-semibold text-foreground">{t.post.deleteTitle}</p>
          <button onClick={onClose} className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
            <X size={16} />
          </button>
        </div>
        <div className="px-4 py-5">
          <p className="text-sm text-muted-foreground leading-relaxed">{t.post.deleteConfirm}</p>
          <div className="flex justify-end gap-2 mt-5">
            <Button variant="ghost" size="sm" onClick={onClose} disabled={isPending}>{t.common.cancel}</Button>
            <Button size="sm" onClick={onConfirm} disabled={isPending} className="bg-destructive hover:bg-destructive/90 text-destructive-foreground">
              {isPending ? t.post.deleting : t.post.delete}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function CommentRepliesBtn({ comment }: { comment: Comment }) {
  const navigate = useNavigate()
  return (
    <button
      onClick={(e) => { e.stopPropagation(); navigate(`/post/${comment.id}`) }}
      className="flex items-center gap-1 text-xs text-muted-foreground hover:text-sky-500 transition-colors"
    >
      <MessageCircle size={13} />
      {comment.commentsCount > 0 && <span>{comment.commentsCount}</span>}
    </button>
  )
}

function CommentLike({ comment }: { comment: Comment }) {
  const [liked, setLiked] = useState(comment.likedByMe)
  const [count, setCount] = useState(comment.likesCount)
  const [pending, setPending] = useState(false)

  const toggle = async () => {
    if (pending) return
    setPending(true)
    const next = !liked
    setLiked(next)
    setCount((c) => c + (next ? 1 : -1))
    try {
      if (next) await likePost(comment.id)
      else await unlikePost(comment.id)
    } catch {
      setLiked(!next)
      setCount((c) => c + (next ? -1 : 1))
    } finally {
      setPending(false)
    }
  }

  return (
    <button
      onClick={toggle}
      disabled={pending}
      className={`flex items-center gap-1 text-xs transition-colors ${
        liked ? 'text-rose-500' : 'text-muted-foreground hover:text-rose-500'
      }`}
    >
      <Heart size={13} fill={liked ? 'currentColor' : 'none'} />
      {count > 0 && <span>{count}</span>}
    </button>
  )
}

function CommentDelete({ comment, parentId }: { comment: Comment; parentId: string }) {
  const { user } = useAuth()
  const qc = useQueryClient()
  const [pending, setPending] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)

  if (user?.id !== comment.authorId) return null

  const handleConfirm = async () => {
    if (pending) return
    setPending(true)
    try {
      await deletePost(comment.id)
      qc.invalidateQueries({ queryKey: ['comments', parentId] })
      qc.invalidateQueries({ queryKey: ['post', comment.postId] })
      setConfirmOpen(false)
    } catch {
      setPending(false)
    }
  }

  return (
    <>
      <button
        onClick={(e) => { e.stopPropagation(); setConfirmOpen(true) }}
        className="ml-auto text-muted-foreground hover:text-destructive transition-colors"
      >
        <Trash2 size={12} />
      </button>
      <ConfirmDeleteModal
        open={confirmOpen}
        onConfirm={handleConfirm}
        onClose={() => setConfirmOpen(false)}
        isPending={pending}
      />
    </>
  )
}

function CommentRebreeze({ comment }: { comment: Comment }) {
  const [rebreezed, setRebreezed] = useState(comment.rebreezedByMe)
  const [count, setCount] = useState(comment.rebreezeCount)
  const [pending, setPending] = useState(false)

  const toggle = async () => {
    if (pending) return
    setPending(true)
    const next = !rebreezed
    setRebreezed(next)
    setCount((c) => c + (next ? 1 : -1))
    try {
      if (next) await rebreezePost(comment.id)
      else await unrebreezePost(comment.id)
    } catch {
      setRebreezed(!next)
      setCount((c) => c + (next ? -1 : 1))
    } finally {
      setPending(false)
    }
  }

  return (
    <button
      onClick={toggle}
      disabled={pending}
      className={`flex items-center gap-1 text-xs transition-colors ${
        rebreezed ? 'text-emerald-500' : 'text-muted-foreground hover:text-emerald-500'
      }`}
    >
      <Repeat2 size={13} />
      {count > 0 && <span>{count}</span>}
    </button>
  )
}

/** Depth-2 — replies to a reply. Visual nesting stops here. */
function SubReplyRow({
  reply,
  postId,
  parentId,
}: {
  reply: Comment
  postId: string
  parentId: string
}) {
  const [isReplying, setIsReplying] = useState(false)
  const { t, locale } = useLanguage()
  const { user } = useAuth()
  const navigate = useNavigate()
  const { data: authorProfile } = useUser(reply.author.id)
  const { data: myProfile } = useUser(user?.id ?? '')
  const avatarUrl = authorProfile?.avatarUrl ?? reply.author.avatarUrl ?? null
  const myAvatarUrl = myProfile?.avatarUrl ?? user?.avatarUrl ?? null
  const addReply = useAddReplyToComment(postId, reply.id)

  return (
    <div
      className="pt-2 pb-1 rounded-md cursor-pointer hover:bg-muted/40 transition-colors"
      onClick={() => navigate(`/post/${reply.id}`)}
    >
      <div className="flex gap-2">
        <Link
          to={`/profile/${reply.author.id}`}
          className="shrink-0 mt-0.5"
          onClick={(e) => e.stopPropagation()}
        >
          <UserAvatar username={reply.author.username} avatarUrl={avatarUrl} size={20} />
        </Link>
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2 mb-0.5">
            <Link
              to={`/profile/${reply.author.id}`}
              className="font-medium text-sm text-foreground hover:underline"
              onClick={(e) => e.stopPropagation()}
            >
              @{reply.author.username}
            </Link>
            <span className="text-xs text-muted-foreground">
              {timeAgo(reply.createdAt, t.common, locale)}
            </span>
          </div>
          <TranslationButtons id={reply.id} content={reply.content} mentions={reply.mentions} />
          {(reply.media ?? []).length > 0 && <div className="mt-2" onClick={(e) => e.stopPropagation()}><PostMediaGrid media={reply.media} /></div>}
          <div className="mt-1 flex items-center gap-3" onClick={(e) => e.stopPropagation()}>
            {user && (
              <button
                onClick={() => setIsReplying((v) => !v)}
                className="text-xs text-muted-foreground hover:text-primary transition-colors"
              >
                {t.post.reply}
              </button>
            )}
            <CommentLike comment={reply} />
            <CommentRepliesBtn comment={reply} />
            <CommentRebreeze comment={reply} />
            <CommentDelete comment={reply} parentId={parentId} />
          </div>
          {isReplying && user && (
            <div onClick={(e) => e.stopPropagation()}>
              <InlineReplyForm
                avatarUrl={myAvatarUrl}
                username={user.username}
                isPending={addReply.isPending}
                onSubmit={async (c, mentions, mediaIds) => { await addReply.mutateAsync({ content: c, mentions, mediaIds }); setIsReplying(false) }}
                onCancel={() => setIsReplying(false)}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

/** Depth-1 — direct reply to a comment. Loads and shows its own sub-replies. */
function ReplyRow({
  reply,
  postId,
  parentId,
}: {
  reply: Comment
  postId: string
  parentId: string
}) {
  const [isReplying, setIsReplying] = useState(false)
  const [isExpanded, setIsExpanded] = useState(false)

  const { t, locale } = useLanguage()
  const { user } = useAuth()
  const navigate = useNavigate()
  const { data: authorProfile } = useUser(reply.author.id)
  const { data: myProfile } = useUser(user?.id ?? '')
  const avatarUrl = authorProfile?.avatarUrl ?? reply.author.avatarUrl ?? null
  const myAvatarUrl = myProfile?.avatarUrl ?? user?.avatarUrl ?? null

  const { data: subRepliesData } = useCommentReplies(reply.id)
  const addReply = useAddReplyToComment(postId, reply.id)
  const subReplies = subRepliesData?.data ?? []

  return (
    <div className="pt-2 pb-1">
      <div
        className="flex gap-2 rounded-md cursor-pointer hover:bg-muted/40 transition-colors"
        onClick={() => navigate(`/post/${reply.id}`)}
      >
        <Link
          to={`/profile/${reply.author.id}`}
          className="shrink-0 mt-0.5"
          onClick={(e) => e.stopPropagation()}
        >
          <UserAvatar username={reply.author.username} avatarUrl={avatarUrl} size={26} />
        </Link>
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2 mb-0.5">
            <Link
              to={`/profile/${reply.author.id}`}
              className="font-medium text-sm text-foreground hover:underline"
              onClick={(e) => e.stopPropagation()}
            >
              @{reply.author.username}
            </Link>
            <span className="text-xs text-muted-foreground">
              {timeAgo(reply.createdAt, t.common, locale)}
            </span>
          </div>
          <TranslationButtons id={reply.id} content={reply.content} mentions={reply.mentions} />
          {(reply.media ?? []).length > 0 && <div className="mt-2" onClick={(e) => e.stopPropagation()}><PostMediaGrid media={reply.media} /></div>}
          <div className="mt-1 flex items-center gap-3 flex-wrap" onClick={(e) => e.stopPropagation()}>
            {user && (
              <button
                onClick={() => setIsReplying((v) => !v)}
                className="text-xs text-muted-foreground hover:text-primary transition-colors"
              >
                {t.post.reply}
              </button>
            )}
            <CommentLike comment={reply} />
            <CommentRepliesBtn comment={reply} />
            <CommentRebreeze comment={reply} />
            <CommentDelete comment={reply} parentId={parentId} />
            {subReplies.length > 0 && (
              <button
                onClick={() => setIsExpanded((v) => !v)}
                className="flex items-center gap-1 text-xs text-primary/70 hover:text-primary transition-colors font-medium"
              >
                {isExpanded ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
                {isExpanded
                  ? t.post.hideReplies
                  : `${subReplies.length} ${t.post.viewReplies}`}
              </button>
            )}
          </div>
          {isReplying && user && (
            <div onClick={(e) => e.stopPropagation()}>
              <InlineReplyForm
                avatarUrl={myAvatarUrl}
                username={user.username}
                isPending={addReply.isPending}
                onSubmit={async (c, mentions, mediaIds) => {
                  await addReply.mutateAsync({ content: c, mentions, mediaIds })
                  setIsReplying(false)
                  setIsExpanded(true)
                }}
                onCancel={() => setIsReplying(false)}
              />
            </div>
          )}
        </div>
      </div>

      {/* Sub-replies — indented with a visible left border */}
      {isExpanded && subReplies.length > 0 && (
        <div className="mt-2 ml-3 pl-3 border-l-2 border-border/50">
          {subReplies.map((sr) => (
            <SubReplyRow key={sr.id} reply={sr} postId={postId} parentId={reply.id} />
          ))}
        </div>
      )}
    </div>
  )
}

/** Depth-0 — top-level comment. Loads and shows its direct replies. */
function CommentItem({ comment, postId }: { comment: Comment; postId: string }) {
  const [isReplying, setIsReplying] = useState(false)
  const [isExpanded, setIsExpanded] = useState(false)

  const { t, locale } = useLanguage()
  const { user } = useAuth()
  const navigate = useNavigate()
  const { data: authorProfile } = useUser(comment.author.id)
  const { data: myProfile } = useUser(user?.id ?? '')
  const avatarUrl = authorProfile?.avatarUrl ?? comment.author.avatarUrl ?? null
  const myAvatarUrl = myProfile?.avatarUrl ?? user?.avatarUrl ?? null

  const { data: repliesData } = useCommentReplies(comment.id)
  const addReply = useAddReplyToComment(postId, comment.id)
  const replies = repliesData?.data ?? []

  return (
    <div className="border-b border-border">
      {/* Comment row — clicking content navigates to the comment's own thread */}
      <div
        className="flex gap-3 px-4 pt-4 pb-3 cursor-pointer hover:bg-muted/40 transition-colors"
        onClick={() => navigate(`/post/${comment.id}`)}
      >
        <Link
          to={`/profile/${comment.author.id}`}
          className="shrink-0 mt-0.5"
          onClick={(e) => e.stopPropagation()}
        >
          <UserAvatar username={comment.author.username} avatarUrl={avatarUrl} size={32} />
        </Link>
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2 mb-1">
            <Link
              to={`/profile/${comment.author.id}`}
              className="font-medium text-sm text-foreground hover:underline"
              onClick={(e) => e.stopPropagation()}
            >
              @{comment.author.username}
            </Link>
            <span className="text-xs text-muted-foreground">
              {timeAgo(comment.createdAt, t.common, locale)}
            </span>
          </div>
          <TranslationButtons id={comment.id} content={comment.content} mentions={comment.mentions} />
          {(comment.media ?? []).length > 0 && <div className="mt-2" onClick={(e) => e.stopPropagation()}><PostMediaGrid media={comment.media} /></div>}
          <div className="mt-1.5 flex items-center gap-3 flex-wrap" onClick={(e) => e.stopPropagation()}>
            {user && (
              <button
                onClick={() => setIsReplying((v) => !v)}
                className="text-xs text-muted-foreground hover:text-primary transition-colors"
              >
                {t.post.reply}
              </button>
            )}
            <CommentLike comment={comment} />
            <CommentRepliesBtn comment={comment} />
            <CommentRebreeze comment={comment} />
            <CommentDelete comment={comment} parentId={postId} />
            {replies.length > 0 && (
              <button
                onClick={() => setIsExpanded((v) => !v)}
                className="flex items-center gap-1 text-xs text-primary/80 hover:text-primary transition-colors font-medium"
              >
                {isExpanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                {isExpanded
                  ? t.post.hideReplies
                  : `${replies.length} ${t.post.viewReplies}`}
              </button>
            )}
          </div>
          {isReplying && user && (
            <div onClick={(e) => e.stopPropagation()}>
              <InlineReplyForm
                avatarUrl={myAvatarUrl}
                username={user.username}
                isPending={addReply.isPending}
                onSubmit={async (c, mentions, mediaIds) => {
                  await addReply.mutateAsync({ content: c, mentions, mediaIds })
                  setIsReplying(false)
                  setIsExpanded(true)
                }}
                onCancel={() => setIsReplying(false)}
              />
            </div>
          )}
        </div>
      </div>

      {/* Replies thread — indented under the comment with a prominent left border */}
      {isExpanded && replies.length > 0 && (
        <div className="mx-4 mb-3 pl-4 border-l-2 border-border/70 ml-[52px]">
          {replies.map((r) => (
            <ReplyRow key={r.id} reply={r} postId={postId} parentId={comment.id} />
          ))}
        </div>
      )}
    </div>
  )
}

export function PostPage() {
  const { t } = useLanguage()
  const { id } = useParams<{ id: string }>()
  const { user } = useAuth()
  const { data: myProfile } = useUser(user?.id ?? '')
  const myAvatarUrl = myProfile?.avatarUrl ?? user?.avatarUrl ?? null
  const { data: post, isLoading } = usePost(id ?? '')
  const { data: comments } = useComments(id ?? '')
  const addComment = useAddComment(id ?? '')
  const [content, setContent] = useState('')
  const [media, setMedia] = useState<MediaItem[]>([])
  const commentRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const submittingRef = useRef(false)
  const mention = useMentionAutocomplete(content, setContent)

  const remaining = MAX_REPLY - content.length
  const uploadedIds = media.filter((m) => m.id != null).map((m) => m.id!)
  const canComment =
    (content.trim().length > 0 || uploadedIds.length > 0) &&
    content.length <= MAX_REPLY &&
    !media.some((m) => m.uploading) &&
    !addComment.isPending

  const handleFiles = async (files: FileList | null) => {
    if (!files) return
    const rawFiles = Array.from(files).slice(0, MAX_MEDIA_REPLY - media.length)
    if (!rawFiles.length) return
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
    await Promise.all(processed.map(async (p, i) => {
      const key = items[i].key
      try {
        const result = await uploadMedia(p.file)
        setMedia((prev) => prev.map((m) => m.key === key ? { ...m, id: result.id, uploading: false } : m))
      } catch {
        setMedia((prev) => prev.map((m) => m.key === key ? { ...m, uploading: false, error: true } : m))
      }
    }))
  }

  const removeMedia = (key: string) => {
    setMedia((prev) => {
      const item = prev.find((m) => m.key === key)
      if (item) URL.revokeObjectURL(item.previewUrl)
      return prev.filter((m) => m.key !== key)
    })
  }

  const handleComment = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!canComment || submittingRef.current) return
    submittingRef.current = true
    await addComment.mutateAsync({ content: content.trim(), mentions: mention.mentions, mediaIds: uploadedIds }).finally(() => { submittingRef.current = false })
    setContent('')
    mention.resetMentions()
    media.forEach((m) => URL.revokeObjectURL(m.previewUrl))
    setMedia([])
  }

  return (
    <div>
      <div className="px-4 py-4 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm flex items-center gap-3">
        <Link to="/" className="text-muted-foreground hover:text-foreground transition-colors">
          <ArrowLeft size={18} />
        </Link>
        <h1 className="text-base font-semibold text-foreground">{t.post.postPageTitle}</h1>
      </div>

      {/* Ancestor chain — only shown once the main post is loaded */}
      {!isLoading && post?.parentId && <ContextPost id={post.parentId} />}

      {isLoading ? (
        <div className="px-4 py-4 flex gap-3">
          <Skeleton className="w-9 h-9 rounded-full shrink-0" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-3 w-full" />
          </div>
        </div>
      ) : post ? (
        <PostCard post={post} inThread={!!post.parentId} />
      ) : null}

      {user && (
        <form onSubmit={handleComment} className="px-4 py-4 border-b border-border flex gap-3">
          <div className="shrink-0 mt-0.5">
            <UserAvatar username={user.username} avatarUrl={myAvatarUrl} size={32} />
          </div>
          <div className="flex-1 relative">
            <Textarea
              ref={commentRef}
              placeholder={t.post.replyPlaceholder}
              value={content}
              onChange={(e) => {
                setContent(e.target.value)
                mention.onContentChange(e.target.value, e.target.selectionStart ?? e.target.value.length)
              }}
              rows={2}
              className="resize-none border-none shadow-none p-0 text-sm focus-visible:ring-0 placeholder:text-muted-foreground/60 bg-transparent min-h-0"
            />
            <MentionDropdown
              active={mention.mentionActive}
              suggestions={mention.suggestions}
              isLoading={mention.isLoading}
              topOffset={mention.dropdownTop}
              onSelect={(user) => { mention.selectUser(user); commentRef.current?.focus() }}
            />
            {media.length > 0 && <MediaPreviewGrid items={media} onRemove={removeMedia} />}
            <div className="flex items-center justify-between mt-2 pt-2 border-t border-border/50">
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={media.length >= MAX_MEDIA_REPLY}
                  className="p-1.5 rounded-lg text-muted-foreground hover:text-primary hover:bg-accent transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <ImagePlus size={16} />
                </button>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*,video/*"
                  multiple
                  className="hidden"
                  onChange={(e) => handleFiles(e.target.files)}
                  onClick={(e) => { (e.target as HTMLInputElement).value = '' }}
                />
                <span className={`text-xs tabular-nums ${remaining < 20 ? 'text-destructive' : 'text-muted-foreground/50'}`}>
                  {remaining}
                </span>
              </div>
              <Button type="submit" size="sm" disabled={!canComment}>
                {t.post.reply}
              </Button>
            </div>
          </div>
        </form>
      )}

      <div>
        {(comments?.data ?? []).map((c) => (
          <CommentItem key={c.id} comment={c} postId={id ?? ''} />
        ))}
      </div>
    </div>
  )
}
