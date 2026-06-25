import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Search } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { UserAvatar } from '@/components/UserAvatar'
import { useFollowing, useFollowMutation } from '@/hooks/useUser'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import { adminT } from '@/i18n/admin'
import type { User } from '@/types/api'

function UserRow({ user, ownerId }: { user: User; ownerId: string }) {
  const { t } = useLanguage()
  const { user: me } = useAuth()
  const qc = useQueryClient()
  const isMe = me?.id === user.id
  const follow = useFollowMutation(user.id)
  // État local pour un retour immédiat : le bouton reflète l'action sans attendre
  // le refetch de la liste.
  const [following, setFollowing] = useState(user.isFollowedByMe)
  const [hover, setHover] = useState(false)

  const toggle = () => {
    const next = !following
    setFollowing(next)
    follow.mutate(next, {
      onError: () => setFollowing(!next),
      onSettled: () => qc.invalidateQueries({ queryKey: ['following', ownerId] }),
    })
  }

  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border hover:bg-accent/20 transition-colors">
      <Link to={`/profile/${user.id}`} className="flex items-center gap-3 min-w-0">
        <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={40} />
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground truncate">@{user.username}</p>
          {user.bio && <p className="text-xs text-muted-foreground truncate">{user.bio}</p>}
        </div>
      </Link>
      {!isMe && (
        <Button
          variant={following ? 'outline' : 'default'}
          size="sm"
          disabled={follow.isPending}
          onClick={toggle}
          onMouseEnter={() => setHover(true)}
          onMouseLeave={() => setHover(false)}
          className={`shrink-0 ${following && hover ? 'border-destructive/40 text-destructive hover:bg-destructive/10' : ''}`}
        >
          {following ? (hover ? t.common.unfollow : t.common.followingBtn) : t.common.follow}
        </Button>
      )}
    </div>
  )
}

function UserRowSkeleton() {
  return (
    <div className="flex items-center gap-3 px-4 py-3 border-b border-border">
      <Skeleton className="w-10 h-10 rounded-full shrink-0" />
      <div className="flex-1 space-y-1.5">
        <Skeleton className="h-3 w-28" />
        <Skeleton className="h-3 w-44" />
      </div>
    </div>
  )
}

export function FollowingPage() {
  const { t, locale } = useLanguage()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading } = useFollowing(id ?? '')
  const [query, setQuery] = useState('')

  const all = data?.data ?? []
  const q = query.trim().toLowerCase()
  const filtered = q
    ? all.filter((u) =>
        u.username.toLowerCase().includes(q) ||
        (u.displayName ?? '').toLowerCase().includes(q),
      )
    : all

  return (
    <div>
      <div className="px-4 py-4 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm flex items-center gap-3">
        <Link to={`/profile/${id}`} className="text-muted-foreground hover:text-foreground transition-colors">
          <ArrowLeft size={18} />
        </Link>
        <h1 className="text-base font-semibold text-foreground">{t.followingPage.title}</h1>
      </div>

      {!isLoading && all.length > 0 && (
        <div className="px-4 py-3 border-b border-border">
          <div className="relative">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={adminT(locale).searchUsers}
              className="h-9 pl-9 bg-input border-border rounded-xl"
            />
          </div>
        </div>
      )}

      {isLoading
        ? Array.from({ length: 5 }).map((_, i) => <UserRowSkeleton key={i} />)
        : all.length === 0
        ? <p className="px-4 py-8 text-center text-sm text-muted-foreground">{t.followingPage.empty}</p>
        : filtered.length === 0
        ? <p className="px-4 py-8 text-center text-sm text-muted-foreground">{adminT(locale).noUsers}</p>
        : filtered.map((u) => <UserRow key={u.id} user={u} ownerId={id ?? ''} />)
      }
    </div>
  )
}
