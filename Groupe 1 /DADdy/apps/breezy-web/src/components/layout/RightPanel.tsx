import { useState, useEffect, useRef } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Search, TrendingUp, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useQuery } from '@tanstack/react-query'
import { getSuggestions, searchUsers } from '@/api/users'
import { getTrending } from '@/api/posts'
import { useFollowMutation } from '@/hooks/useUser'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import { UserAvatar } from '@/components/UserAvatar'
import { MentionDropdown } from '@/components/post/MentionDropdown'
import type { User } from '@/types/api'

const MENTION_RE = /@([a-zA-Z0-9_]{0,50})$/

// ── Tendances ─────────────────────────────────────────────────────────────────


function TrendsWidget() {
  const { t } = useLanguage()
  const { data: trends = [], isLoading } = useQuery({
    queryKey: ['hashtags-trending'],
    queryFn: getTrending,
    staleTime: 60_000,
  })

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-center gap-2 mb-3">
        <TrendingUp size={14} className="text-primary" />
        <p className="text-sm font-semibold text-foreground">{t.rightPanel.trends}</p>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between py-1">
              <div className="space-y-1">
                <Skeleton className="h-2.5 w-16" />
                <Skeleton className="h-3.5 w-24" />
              </div>
              <Skeleton className="h-3 w-12" />
            </div>
          ))}
        </div>
      ) : trends.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t.rightPanel.noTrends}</p>
      ) : (
        <div className="flex flex-col gap-0.5">
          {trends.slice(0, 5).map(({ name, count }, i) => (
            <Link
              key={name}
              to={`/hashtag/${name}`}
              className="flex items-center justify-between py-2 px-1 rounded-lg hover:bg-accent/50 transition-colors group"
            >
              <div>
                <p className="text-xs text-muted-foreground">#{i + 1} · {t.rightPanel.trending}</p>
                <p className="text-sm font-medium text-foreground group-hover:text-primary transition-colors">
                  #{name}
                </p>
              </div>
              <p className="text-xs text-muted-foreground tabular-nums">
                {count} {count > 1 ? t.post.postPlural : t.post.postSingular}
              </p>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

function UnfollowModal({ username, open, onConfirm, onClose, isPending }: {
  username: string
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
          <p className="text-sm font-semibold text-foreground">{t.profile.unfollowTitle}</p>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <X size={16} />
          </button>
        </div>
        <div className="px-4 py-5">
          <p className="text-sm text-muted-foreground leading-relaxed">
            {t.profile.unfollowConfirm}{' '}
            <span className="font-medium text-foreground">@{username}</span> ?
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
              {isPending ? t.profile.unfollowing : t.profile.unfollow}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function SuggestedUser({ user }: { user: User }) {
  const { t } = useLanguage()
  const follow = useFollowMutation(user.id)
  const [optimisticFollowed, setOptimisticFollowed] = useState<boolean | null>(null)
  const [unfollowOpen, setUnfollowOpen] = useState(false)
  const followed = optimisticFollowed ?? (user.isFollowedByMe ?? false)

  const handleFollow = () => {
    setOptimisticFollowed(true)
    follow.mutate(true)
  }

  const handleUnfollow = () => {
    setOptimisticFollowed(false)
    setUnfollowOpen(false)
    follow.mutate(false)
  }

  return (
    <>
      <div className="flex items-center justify-between gap-2 py-2">
        <Link to={`/profile/${user.id}`} className="flex items-center gap-2.5 min-w-0">
          <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={32} />
          <div className="min-w-0">
            <p className="text-sm font-medium text-foreground truncate">@{user.username}</p>
            {user.bio && <p className="text-xs text-muted-foreground truncate">{user.bio}</p>}
          </div>
        </Link>
        <Button
          size="sm"
          variant={followed ? 'outline' : 'default'}
          disabled={follow.isPending}
          onClick={followed ? () => setUnfollowOpen(true) : handleFollow}
          className="shrink-0 h-7 px-3 text-xs rounded-full"
        >
          {followed ? t.common.followingBtn : t.common.follow}
        </Button>
      </div>

      <UnfollowModal
        username={user.username}
        open={unfollowOpen}
        onConfirm={handleUnfollow}
        onClose={() => setUnfollowOpen(false)}
        isPending={follow.isPending}
      />
    </>
  )
}

export function RightPanel() {
  const { t } = useLanguage()
  const { user: me } = useAuth()
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const [mentionActive, setMentionActive] = useState(false)
  const [rawMentionQuery, setRawMentionQuery] = useState('')
  const [debouncedMentionQuery, setDebouncedMentionQuery] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const t = setTimeout(() => setDebouncedMentionQuery(rawMentionQuery), 150)
    return () => clearTimeout(t)
  }, [rawMentionQuery])

  const { data: mentionData, isFetching: mentionLoading } = useQuery({
    queryKey: ['mention-search-panel', debouncedMentionQuery],
    queryFn: () => searchUsers(debouncedMentionQuery, 1, 5),
    enabled: mentionActive && debouncedMentionQuery.length > 0,
    staleTime: 0,
  })
  const mentionSuggestions = mentionData?.data ?? []

  const { data, isLoading } = useQuery({
    queryKey: ['users-suggestions'],
    queryFn: () => getSuggestions(),
    staleTime: 60_000,
  })

  const suggestions = (data?.data ?? []).filter((u) => u.id !== me?.id)

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    setSearchQuery(val)
    const match = val.match(MENTION_RE)
    if (match) {
      setRawMentionQuery(match[1])
      setMentionActive(true)
    } else {
      setRawMentionQuery('')
      setMentionActive(false)
    }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (mentionActive) return
    const q = searchQuery.trim()
    if (!q) return
    navigate(`/search?q=${encodeURIComponent(q)}`)
    inputRef.current?.blur()
  }

  const handleMentionSelect = (user: User) => {
    navigate(`/profile/${user.id}`)
    setSearchQuery('')
    setMentionActive(false)
    inputRef.current?.blur()
  }

  return (
    <aside className="w-72 flex flex-col gap-4">
      <form onSubmit={handleSearch} className="relative">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none z-10" />
        <Input
          ref={inputRef}
          placeholder={t.common.search}
          value={searchQuery}
          onChange={handleSearchChange}
          className="pl-9 rounded-full bg-accent border-transparent focus-visible:border-border focus-visible:bg-background placeholder:text-muted-foreground"
        />
        <MentionDropdown
          active={mentionActive}
          suggestions={mentionSuggestions}
          isLoading={mentionLoading}
          onSelect={handleMentionSelect}
        />
      </form>

      <TrendsWidget />

      <div className="rounded-2xl border border-border bg-card p-4">
        <p className="text-sm font-semibold text-foreground mb-3">{t.rightPanel.suggestions}</p>

        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-2.5 py-1">
                <Skeleton className="w-8 h-8 rounded-full shrink-0" />
                <div className="flex-1 space-y-1">
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-3 w-36" />
                </div>
              </div>
            ))}
          </div>
        ) : suggestions.length === 0 ? (
          <p className="text-xs text-muted-foreground">{t.rightPanel.noSuggestions}</p>
        ) : (
          <div className="divide-y divide-border">
            {suggestions.map((u) => <SuggestedUser key={u.id} user={u} />)}
          </div>
        )}
      </div>

      <p className="mt-auto pt-4 text-xs text-muted-foreground/60">
        {t.rightPanel.copyright}
      </p>
    </aside>
  )
}

export function RightPanelSkeleton() {
  return (
    <aside className="w-72 flex flex-col gap-4">
      <Skeleton className="h-9 w-full rounded-full" />
      <div className="rounded-2xl border border-border bg-card p-4 space-y-3">
        <Skeleton className="h-4 w-24" />
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-between justify-between py-1">
            <div className="space-y-1">
              <Skeleton className="h-2.5 w-16" />
              <Skeleton className="h-3.5 w-24" />
            </div>
            <Skeleton className="h-3 w-12" />
          </div>
        ))}
      </div>
      <div className="rounded-2xl border border-border bg-card p-4 space-y-3">
        <Skeleton className="h-4 w-32" />
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-center gap-2.5 py-1">
            <Skeleton className="w-8 h-8 rounded-full shrink-0" />
            <div className="flex-1 space-y-1">
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-3 w-36" />
            </div>
          </div>
        ))}
      </div>
    </aside>
  )
}
