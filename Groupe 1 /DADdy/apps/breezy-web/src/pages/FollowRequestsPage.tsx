import { Link } from 'react-router-dom'
import { ArrowLeft, Check, X, UserPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { UserAvatar } from '@/components/UserAvatar'
import { useFollowRequests, useAcceptFollowRequest, useRejectFollowRequest } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import type { User } from '@/types/api'

function RequestRow({ user }: { user: User }) {
  const { t } = useLanguage()
  const accept = useAcceptFollowRequest()
  const reject = useRejectFollowRequest()
  const pending = accept.isPending || reject.isPending

  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border hover:bg-accent/20 transition-colors">
      <Link to={`/profile/${user.id}`} className="flex items-center gap-3 min-w-0">
        <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={40} />
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground truncate">
            {user.displayName || `@${user.username}`}
          </p>
          {user.displayName && <p className="text-xs text-muted-foreground truncate">@{user.username}</p>}
        </div>
      </Link>
      <div className="flex items-center gap-2 shrink-0">
        <Button
          size="sm"
          disabled={pending}
          onClick={() => accept.mutate(user.id)}
          className="gap-1.5"
        >
          <Check size={14} />
          {t.followRequests.accept}
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={pending}
          onClick={() => reject.mutate(user.id)}
          className="gap-1.5"
        >
          <X size={14} />
          {t.followRequests.reject}
        </Button>
      </div>
    </div>
  )
}

function RequestRowSkeleton() {
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

export function FollowRequestsPage() {
  const { t } = useLanguage()
  const { data, isLoading } = useFollowRequests()
  const requests = data?.data ?? []

  return (
    <div>
      <div className="px-4 py-4 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm flex items-center gap-3">
        <Link to="/notifications" className="text-muted-foreground hover:text-foreground transition-colors">
          <ArrowLeft size={18} />
        </Link>
        <h1 className="text-base font-semibold text-foreground">{t.followRequests.title}</h1>
      </div>

      {isLoading ? (
        Array.from({ length: 4 }).map((_, i) => <RequestRowSkeleton key={i} />)
      ) : requests.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20 text-muted-foreground">
          <UserPlus size={36} strokeWidth={1.25} />
          <p className="text-sm">{t.followRequests.empty}</p>
        </div>
      ) : (
        requests.map((u) => <RequestRow key={u.id} user={u} />)
      )}
    </div>
  )
}
