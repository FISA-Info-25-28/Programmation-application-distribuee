import { useState, useRef } from 'react'
import { createPortal } from 'react-dom'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { getUser } from '@/api/users'
import { useFollowMutation } from '@/hooks/useUser'
import { useAuth } from '@/hooks/useAuth'
import { UserAvatar } from '@/components/UserAvatar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

const CARD_WIDTH = 288

interface Coords { x: number; y: number; above: boolean }

export function UserHoverCard({ userId, children }: {
  userId: string
  children: React.ReactNode
}) {
  const [open, setOpen] = useState(false)
  const [coords, setCoords] = useState<Coords>({ x: 0, y: 0, above: false })
  const [enabled, setEnabled] = useState(false)
  const [localFollowState, setLocalFollowState] = useState<boolean | undefined>(undefined)
  const triggerRef = useRef<HTMLSpanElement>(null)
  const showTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const hideTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const { data: user, isLoading } = useQuery({
    queryKey: ['user', userId],
    queryFn: () => getUser(userId),
    enabled: enabled && !!userId,
    staleTime: 60_000,
  })

  const { user: me } = useAuth()
  const isMe = me?.id === userId
  const follow = useFollowMutation(userId)

  const followed = localFollowState ?? user?.isFollowedByMe ?? false

  const openCard = () => {
    clearTimeout(hideTimer.current)
    setEnabled(true)
    showTimer.current = setTimeout(() => {
      const el = triggerRef.current
      if (!el) return
      const rect = el.getBoundingClientRect()
      const above = rect.top > window.innerHeight * 0.55
      setCoords({
        x: Math.max(8, Math.min(rect.left, window.innerWidth - CARD_WIDTH - 8)),
        y: above ? rect.top - 8 : rect.bottom + 8,
        above,
      })
      setOpen(true)
    }, 280)
  }

  const scheduleClose = () => {
    clearTimeout(showTimer.current)
    hideTimer.current = setTimeout(() => setOpen(false), 180)
  }

  return (
    <>
      <span
        ref={triggerRef}
        onMouseEnter={openCard}
        onMouseLeave={scheduleClose}
        className="inline"
      >
        {children}
      </span>

      {open && createPortal(
        <div
          className="fixed z-50 w-72 rounded-2xl border border-border bg-popover shadow-xl p-4"
          style={{
            left: coords.x,
            ...(coords.above
              ? { bottom: window.innerHeight - coords.y }
              : { top: coords.y }),
          }}
          onClick={(e) => e.stopPropagation()}
          onMouseEnter={() => clearTimeout(hideTimer.current)}
          onMouseLeave={scheduleClose}
        >
          {isLoading ? (
            <div className="space-y-3">
              <div className="flex items-start justify-between">
                <Skeleton className="w-12 h-12 rounded-full" />
                <Skeleton className="h-8 w-20 rounded-full" />
              </div>
              <div className="space-y-1.5">
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-3 w-44" />
                <Skeleton className="h-3 w-36" />
              </div>
              <div className="flex gap-4">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-20" />
              </div>
            </div>
          ) : user ? (
            <div className="flex flex-col gap-3">
              <div className="flex items-start justify-between gap-2">
                <Link to={`/profile/${user.id}`} onClick={() => setOpen(false)}>
                  <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={48} />
                </Link>
                {!isMe && (
                  <Button
                    size="sm"
                    variant={followed ? 'outline' : 'default'}
                    className="rounded-full h-8 px-4 text-xs shrink-0"
                    disabled={follow.isPending}
                    onClick={() => {
                      const next = !followed
                      setLocalFollowState(next)
                      follow.mutate(next)
                    }}
                  >
                    {followed ? 'Abonné' : 'Suivre'}
                  </Button>
                )}
              </div>

              <div className="min-w-0">
                <Link
                  to={`/profile/${user.id}`}
                  className="font-semibold text-sm text-foreground hover:underline"
                  onClick={() => setOpen(false)}
                >
                  @{user.username}
                </Link>
                {user.bio && (
                  <p className="text-xs text-muted-foreground mt-1 leading-relaxed line-clamp-2">
                    {user.bio}
                  </p>
                )}
              </div>

              <div className="flex gap-4 text-xs">
                <span>
                  <span className="font-semibold text-foreground">{user.followingCount}</span>
                  <span className="text-muted-foreground ml-1">abonnements</span>
                </span>
                <span>
                  <span className="font-semibold text-foreground">{user.followersCount}</span>
                  <span className="text-muted-foreground ml-1">abonnés</span>
                </span>
              </div>
            </div>
          ) : null}
        </div>,
        document.body,
      )}
    </>
  )
}
