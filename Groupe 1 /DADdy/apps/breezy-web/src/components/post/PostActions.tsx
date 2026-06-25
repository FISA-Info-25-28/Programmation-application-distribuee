import { useState, useEffect, useRef } from 'react'
import { Heart, MessageCircle, Repeat2, Trash2, X, Flag, Bookmark, Quote } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { Button } from '@/components/ui/button'
import { likePost, unlikePost, rebreezePost, unrebreezePost, bookmarkPost, unbookmarkPost } from '@/api/posts'
import { reportPost } from '@/api/admin'
import { useDeletePost } from '@/hooks/usePost'
import { useLanguage } from '@/hooks/useLanguage'
import { useHaptics } from '@/hooks/useHaptics'
import { adminT } from '@/i18n/admin'
import { bookmarksT } from '@/i18n/bookmarks'
import type { Post } from '@/types/api'

function RollingNumber({ value, dir = 'up' }: { value: number; dir?: 'up' | 'down' }) {
  return (
    <span className="relative overflow-hidden inline-flex items-center h-[1.1em] w-[2ch] justify-center">
      <span key={value} style={{ animation: `${dir === 'up' ? 'roll-up' : 'roll-down'} 0.18s ease forwards` }}>
        {value}
      </span>
    </span>
  )
}

const CONFETTI_PARTICLES = [0, 60, 120, 180, 240, 300].map((angle) => ({ angle, color: angle % 120 === 0 ? '#10b981' : '#34d399' }))

function RebreezeConfetti({ active }: { active: boolean }) {
  if (!active) return null
  return (
    <div className="absolute inset-0 pointer-events-none overflow-visible">
      {CONFETTI_PARTICLES.map((p, i) => (
        <div
          key={i}
          className="absolute top-1/2 left-1/2 w-1.5 h-1.5 rounded-full"
          style={{
            background: p.color,
            animation: 'confetti-fly 0.5s ease-out forwards',
            ['--cf-angle' as string]: `${p.angle}deg`,
          }}
        />
      ))}
    </div>
  )
}

function ReportModal({ postId, open, onClose }: { postId: string; open: boolean; onClose: () => void }) {
  const { locale } = useLanguage() // kept for reportModal — t already destructured above
  const a = adminT(locale)
  const [reason, setReason] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    if (open) document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const submit = async () => {
    if (!reason.trim() || pending) return
    setPending(true)
    setError(null)
    try {
      await reportPost(postId, reason.trim())
      setDone(true)
      setTimeout(onClose, 900)
    } catch (err) {
      setError(isAxiosError(err) && err.response?.status === 409 ? a.alreadyReported : a.reportError)
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" aria-modal="true" role="dialog">
      <div className="absolute inset-0 bg-background/20 backdrop-blur-[2px]" onClick={onClose} />
      <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl border border-border bg-card shadow-xl">
        <div className="flex items-center justify-between px-4 pt-4 pb-3 border-b border-border">
          <p className="text-sm font-semibold text-foreground">{a.reportTitle}</p>
          <button onClick={onClose} className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
            <X size={16} />
          </button>
        </div>
        <div className="px-4 py-4">
          {done ? (
            <p className="text-sm text-emerald-500 py-2">{a.reported} ✓</p>
          ) : (
            <>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder={a.reportReasonPlaceholder}
                rows={3}
                maxLength={255}
                className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-1 focus:ring-primary"
              />
              {error && <p className="text-xs text-destructive mt-2">{error}</p>}
              <div className="flex justify-end gap-2 mt-4">
                <Button variant="ghost" size="sm" onClick={onClose} disabled={pending}>{a.cancel}</Button>
                <Button size="sm" onClick={submit} disabled={pending || !reason.trim()}>
                  {pending ? a.reportSubmitting : a.reportSubmit}
                </Button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function DeleteConfirmModal({ open, onConfirm, onClose, isPending }: {
  open: boolean
  onConfirm: () => void
  onClose: () => void
  isPending: boolean
}) {
  const { t } = useLanguage()

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    if (open) document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" aria-modal="true" role="dialog">
      <div className="absolute inset-0 bg-background/20 backdrop-blur-[2px]" onClick={onClose} />

      <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl border border-border bg-card shadow-xl">
        <div className="flex items-center justify-between px-4 pt-4 pb-3 border-b border-border">
          <p className="text-sm font-semibold text-foreground">{t.post.deleteTitle}</p>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        <div className="px-4 py-5">
          <p className="text-sm text-muted-foreground leading-relaxed">
            {t.post.deleteConfirm}
          </p>

          <div className="flex justify-end gap-2 mt-5">
            <Button variant="ghost" size="sm" onClick={onClose} disabled={isPending}>
              {t.common.cancel}
            </Button>
            <Button
              size="sm"
              onClick={onConfirm}
              disabled={isPending}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {isPending ? t.post.deleting : t.post.delete}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

interface PostActionsProps {
  post: Post
  isOwn?: boolean
  onComment?: () => void
  onQuote?: () => void
}

export function PostActions({ post, isOwn = false, onComment, onQuote }: PostActionsProps) {
  const navigate = useNavigate()
  const { impact, ImpactStyle } = useHaptics()
  const [liked, setLiked] = useState(post.likedByMe)
  const [likeCount, setLikeCount] = useState(post.likesCount)
  const [likeDir, setLikeDir] = useState<'up' | 'down'>('up')
  const [likePending, setLikePending] = useState(false)
  const [likeAnim, setLikeAnim] = useState(false)
  const [rebreezed, setRebreezed] = useState(post.rebreezedByMe)
  const [rebreezeCount, setRebreezeCount] = useState(post.rebreezeCount)
  const [rebreezeDir, setRebreezeDir] = useState<'up' | 'down'>('up')
  const [rebreezePending, setRebreezePending] = useState(false)
  const [rebreezeAnim, setRebreezeAnim] = useState(false)
  const [bookmarked, setBookmarked] = useState(post.bookmarkedByMe)
  const [bookmarkPending, setBookmarkPending] = useState(false)
  const [rebreezeMenuOpen, setRebreezeMenuOpen] = useState(false)
  const rebreezeMenuRef = useRef<HTMLDivElement>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  const { t, locale } = useLanguage()

  const qc = useQueryClient()
  const deleteMutation = useDeletePost(post.id)

  // Quand la query refetch (ex. retour de navigation), synchronise le state
  // local avec les données serveur — sauf si une mutation est en cours.
  const likeInFlight = useRef(false)
  const rebreezeInFlight = useRef(false)
  const bookmarkInFlight = useRef(false)

  useEffect(() => {
    if (!likeInFlight.current) {
      setLiked(post.likedByMe)
      setLikeCount(post.likesCount)
    }
  }, [post.likedByMe, post.likesCount])

  useEffect(() => {
    if (!rebreezeInFlight.current) {
      setRebreezed(post.rebreezedByMe)
      setRebreezeCount(post.rebreezeCount)
    }
  }, [post.rebreezedByMe, post.rebreezeCount])

  useEffect(() => {
    if (!bookmarkInFlight.current) setBookmarked(post.bookmarkedByMe)
  }, [post.bookmarkedByMe])

  // Ferme le menu rebreeze si clic hors du composant
  useEffect(() => {
    if (!rebreezeMenuOpen) return
    const handler = (e: MouseEvent) => {
      if (rebreezeMenuRef.current && !rebreezeMenuRef.current.contains(e.target as Node)) {
        setRebreezeMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [rebreezeMenuOpen])

  const toggleLike = async () => {
    if (likePending) return
    setLikePending(true)
    likeInFlight.current = true
    const next = !liked
    setLiked(next)
    setLikeDir(next ? 'up' : 'down')
    setLikeCount((c) => c + (next ? 1 : -1))
    if (next) { setLikeAnim(true); setTimeout(() => setLikeAnim(false), 500) }
    impact(next ? ImpactStyle.Medium : ImpactStyle.Light)
    try {
      if (next) await likePost(post.id)
      else await unlikePost(post.id)
      qc.invalidateQueries({ queryKey: ['feed'] })
      qc.invalidateQueries({ queryKey: ['user-posts', post.authorId] })
    } catch {
      setLiked(!next)
      setLikeCount((c) => c + (next ? -1 : 1))
    } finally {
      setLikePending(false)
      likeInFlight.current = false
    }
  }

  const toggleRebreeze = async () => {
    if (rebreezePending) return
    setRebreezePending(true)
    rebreezeInFlight.current = true
    const next = !rebreezed
    setRebreezed(next)
    setRebreezeDir(next ? 'up' : 'down')
    setRebreezeCount((c) => c + (next ? 1 : -1))
    if (next) { setRebreezeAnim(true); setTimeout(() => setRebreezeAnim(false), 600) }
    impact(next ? ImpactStyle.Medium : ImpactStyle.Light)
    try {
      if (next) await rebreezePost(post.id)
      else await unrebreezePost(post.id)
      qc.invalidateQueries({ queryKey: ['feed'] })
      qc.invalidateQueries({ queryKey: ['user-posts', post.authorId] })
      qc.invalidateQueries({ queryKey: ['user-rebreezed'] })
    } catch {
      setRebreezed(!next)
      setRebreezeCount((c) => c + (next ? -1 : 1))
    } finally {
      setRebreezePending(false)
      rebreezeInFlight.current = false
    }
  }

  const toggleBookmark = async () => {
    if (bookmarkPending) return
    setBookmarkPending(true)
    bookmarkInFlight.current = true
    const next = !bookmarked
    setBookmarked(next)
    impact(ImpactStyle.Light)
    try {
      if (next) await bookmarkPost(post.id)
      else await unbookmarkPost(post.id)
      qc.invalidateQueries({ queryKey: ['bookmarks'] })
    } catch {
      setBookmarked(!next)
    } finally {
      setBookmarkPending(false)
      bookmarkInFlight.current = false
    }
  }

  const handleConfirmDelete = async () => {
    await deleteMutation.mutateAsync()
    setConfirmOpen(false)
    navigate('/', { replace: true })
  }

  return (
    <>
      <div className="flex items-center gap-4 mt-3">
        <button
          onClick={toggleLike}
          disabled={likePending}
          className={`flex items-center gap-1.5 text-sm transition-colors px-1 py-1 -mx-1 -my-1 rounded-lg hover:bg-rose-500/10 ${
            liked ? 'text-rose-500' : 'text-muted-foreground hover:text-rose-500'
          }`}
        >
          <Heart
            size={16}
            fill={liked ? 'currentColor' : 'none'}
            style={likeAnim ? { animation: 'heart-burst 0.5s ease' } : undefined}
          />
          <RollingNumber value={likeCount} dir={likeDir} />
        </button>

        <button
          onClick={onComment ?? (() => navigate(`/post/${post.id}`))}
          className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-sky-500 transition-colors px-1 py-1 -mx-1 -my-1 rounded-lg hover:bg-sky-500/10"
        >
          <MessageCircle size={16} />
          <RollingNumber value={post.commentsCount} />
        </button>

        <div ref={rebreezeMenuRef} className="relative">
          <button
            onClick={() => setRebreezeMenuOpen((o) => !o)}
            disabled={rebreezePending}
            className={`flex items-center gap-1.5 text-sm transition-colors px-1 py-1 -mx-1 -my-1 rounded-lg hover:bg-emerald-500/10 ${
              rebreezed ? 'text-emerald-500' : 'text-muted-foreground hover:text-emerald-500'
            }`}
          >
            <Repeat2 size={16} />
            {rebreezeCount > 0 && <RollingNumber value={rebreezeCount} dir={rebreezeDir} />}
            <RebreezeConfetti active={rebreezeAnim} />
          </button>
          {rebreezeMenuOpen && (
            <div className="absolute bottom-full left-0 mb-1 z-20 min-w-[140px] rounded-xl border border-border bg-card shadow-lg overflow-hidden">
              <button
                onClick={() => { setRebreezeMenuOpen(false); toggleRebreeze() }}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent transition-colors"
              >
                <Repeat2 size={14} />
                {rebreezed ? t.post.unrebreeze : t.post.rebreeze}
              </button>
              {onQuote && (
                <button
                  onClick={() => { setRebreezeMenuOpen(false); onQuote() }}
                  className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent transition-colors"
                >
                  <Quote size={14} />
                  {t.post.quote}
                </button>
              )}
            </div>
          )}
        </div>

        <button
          onClick={toggleBookmark}
          disabled={bookmarkPending}
          aria-label={bookmarked ? bookmarksT(locale).removeBookmark : bookmarksT(locale).bookmark}
          title={bookmarked ? bookmarksT(locale).removeBookmark : bookmarksT(locale).bookmark}
          className={`ml-auto flex items-center gap-1.5 text-sm transition-colors px-1 py-1 -mx-1 -my-1 rounded-lg hover:bg-amber-500/10 ${
            bookmarked ? 'text-amber-500' : 'text-muted-foreground hover:text-amber-500'
          }`}
        >
          <Bookmark size={16} fill={bookmarked ? 'currentColor' : 'none'} />
        </button>

        {isOwn ? (
          <button
            onClick={() => setConfirmOpen(true)}
            disabled={deleteMutation.isPending}
            className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-destructive transition-colors px-1 py-1 -mx-1 -my-1 rounded-lg hover:bg-destructive/10"
          >
            <Trash2 size={15} />
          </button>
        ) : (
          <button
            onClick={() => setReportOpen(true)}
            aria-label={adminT(locale).report}
            title={adminT(locale).report}
            className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-amber-500 transition-colors px-1 py-1 -mx-1 -my-1 rounded-lg hover:bg-amber-500/10"
          >
            <Flag size={15} />
          </button>
        )}
      </div>

      <ReportModal postId={post.id} open={reportOpen} onClose={() => setReportOpen(false)} />

      <DeleteConfirmModal
        open={confirmOpen}
        onConfirm={handleConfirmDelete}
        onClose={() => setConfirmOpen(false)}
        isPending={deleteMutation.isPending}
      />
    </>
  )
}
