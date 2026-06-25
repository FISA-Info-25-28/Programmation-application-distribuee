import { Link, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { UserAvatar } from '@/components/UserAvatar'
import { useFollowers, useFollowMutation } from '@/hooks/useUser'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import type { User } from '@/types/api'

function UserRow({ user }: { user: User }) {
  const { t } = useLanguage()
  const { user: me } = useAuth()
  const isMe = me?.id === user.id
  const follow = useFollowMutation(user.id)

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
          variant={user.isFollowedByMe ? 'outline' : 'default'}
          size="sm"
          disabled={follow.isPending}
          onClick={() => follow.mutate(!user.isFollowedByMe)}
          className="shrink-0"
        >
          {user.isFollowedByMe ? t.common.followingBtn : t.common.follow}
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

export function FollowersPage() {
  const { t } = useLanguage()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading } = useFollowers(id ?? '')

  return (
    <div>
      <div className="px-4 py-4 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm flex items-center gap-3">
        <Link to={`/profile/${id}`} className="text-muted-foreground hover:text-foreground transition-colors">
          <ArrowLeft size={18} />
        </Link>
        <h1 className="text-base font-semibold text-foreground">{t.followers.title}</h1>
      </div>

      {isLoading
        ? Array.from({ length: 5 }).map((_, i) => <UserRowSkeleton key={i} />)
        : (data?.data ?? []).length === 0
        ? <p className="px-4 py-8 text-center text-sm text-muted-foreground">{t.followers.empty}</p>
        : (data?.data ?? []).map((u) => <UserRow key={u.id} user={u} />)
      }
    </div>
  )
}
