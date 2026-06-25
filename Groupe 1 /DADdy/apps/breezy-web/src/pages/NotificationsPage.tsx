import { useState, useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Heart, MessageCircle, UserPlus, UserCheck, Rss, Mail, X, CheckCheck, Bell, AtSign, Repeat2 } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { UserAvatar } from '@/components/UserAvatar'
import {
  useNotifications,
  useUnreadCount,
  useMarkAllRead,
} from '@/hooks/useNotifications'
import { markNotificationRead, deleteNotification } from '@/api/notifications'
import { useUser, useFollowRequests } from '@/hooks/useUser'
import type { Notification, NotificationType } from '@/types/api'

// ── Helpers ───────────────────────────────────────────────────────────────────


function timeAgo(date: string) {
  const diff = Date.now() - new Date(date).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return "à l'instant"
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h`
  return `${Math.floor(h / 24)}j`
}

// ── Config par type ───────────────────────────────────────────────────────────

const TYPE_CONFIG: Record<NotificationType, {
  icon: React.ElementType
  color: string
  singular: string
  plural: string
}> = {
  like:        { icon: Heart,         color: 'text-rose-500',    singular: 'a aimé votre post',          plural: 'ont aimé votre post' },
  comment:     { icon: MessageCircle, color: 'text-sky-400',     singular: 'a commenté votre post',      plural: 'ont commenté votre post' },
  follow:      { icon: UserPlus,      color: 'text-emerald-500', singular: 'vous suit maintenant',       plural: 'vous suivent maintenant' },
  new_post:    { icon: Rss,           color: 'text-violet-400',  singular: 'a publié un nouveau post',   plural: 'ont publié de nouveaux posts' },
  new_message: { icon: Mail,          color: 'text-primary',     singular: 'vous a envoyé un message',   plural: 'vous ont envoyé des messages' },
  mention:     { icon: AtSign,        color: 'text-amber-400',   singular: 'vous a mentionné',           plural: 'vous ont mentionné' },
  rebreeze:    { icon: Repeat2,       color: 'text-emerald-400', singular: 'a rebreezé votre post',      plural: 'ont rebreezé votre post' },
  follow_request:          { icon: UserPlus,  color: 'text-amber-500',   singular: "souhaite s'abonner",                  plural: "souhaitent s'abonner" },
  follow_request_accepted: { icon: UserCheck, color: 'text-emerald-500', singular: "a accepté votre demande d'abonnement", plural: "ont accepté votre demande d'abonnement" },
}

// ── Grouping ──────────────────────────────────────────────────────────────────

interface NotificationGroup {
  key: string
  type: NotificationType
  entity_id: string
  actors: Array<{ id: string; username: string }>
  latestId: number
  allIds: number[]
  read: boolean
  created_at: string
}

function groupNotifications(notifications: Notification[]): NotificationGroup[] {
  const order: string[] = []
  const map = new Map<string, NotificationGroup>()

  for (const n of notifications) {
    // follow / follow_request / new_message → un groupe unique par type
    let key: string
    if (n.type === 'follow') key = 'follow'
    else if (n.type === 'follow_request') key = 'follow_request'
    else if (n.type === 'new_message') key = 'new_message'
    else key = `${n.type}:${n.entity_id || n.id}`

    if (!map.has(key)) {
      order.push(key)
      map.set(key, {
        key,
        type: n.type as NotificationType,
        entity_id: n.entity_id,
        actors: [{ id: n.actor_id, username: n.actor_username }],
        latestId: n.id,
        allIds: [n.id],
        read: n.read,
        created_at: n.created_at,
      })
    } else {
      const g = map.get(key)!
      if (!g.actors.find((a) => a.id === n.actor_id)) {
        g.actors.push({ id: n.actor_id, username: n.actor_username })
      }
      g.allIds.push(n.id)
      if (!n.read) g.read = false
      if (n.id > g.latestId) {
        g.latestId = n.id
        g.created_at = n.created_at
      }
    }
  }

  return order.map((k) => map.get(k)!)
}

function getGroupDestination(group: NotificationGroup): string | null {
  switch (group.type) {
    case 'like':
    case 'comment':
    case 'new_post':
    case 'mention':
    case 'rebreeze':
      return group.entity_id ? `/post/${group.entity_id}` : null
    case 'follow':
    case 'follow_request_accepted':
      return group.actors.length === 1 ? `/profile/${group.actors[0].id}` : null
    case 'follow_request':
      return '/follow-requests'
    case 'new_message':
      return '/messages'
    default:
      return null
  }
}

// ── Sous-composants ───────────────────────────────────────────────────────────

/** Affiche @username en fetchant le profil si le username est absent dans la notif. */
function ActorLink({ id, username }: { id: string; username: string }) {
  const { data: profile } = useUser(username ? '' : id)
  const name = username || profile?.username || id
  return (
    <Link
      to={`/profile/${id}`}
      className="font-semibold hover:underline"
      onClick={(e) => e.stopPropagation()}
    >
      @{name}
    </Link>
  )
}

function ActorsText({ actors }: { actors: Array<{ id: string; username: string }> }) {
  if (actors.length === 1) return <ActorLink {...actors[0]} />
  if (actors.length === 2) {
    return <><ActorLink {...actors[0]} />{' et '}<ActorLink {...actors[1]} /></>
  }
  const extra = actors.length - 2
  return (
    <>
      <ActorLink {...actors[0]} />
      {', '}
      <ActorLink {...actors[1]} />
      {` et ${extra} autre${extra > 1 ? 's' : ''}`}
    </>
  )
}

// ── Actor avatars ─────────────────────────────────────────────────────────────

function ActorAvatarBubble({ id, username }: { id: string; username: string }) {
  const { data: profile } = useUser(id)
  return <UserAvatar username={profile?.username ?? username} avatarUrl={profile?.avatarUrl ?? null} size={20} />
}

function ActorAvatars({ actors }: { actors: Array<{ id: string; username: string }> }) {
  const shown = actors.slice(0, 3)
  return (
    <div className="flex -space-x-1.5 mt-1">
      {shown.map((a, i) => (
        <div key={a.id} className="rounded-full border-2 border-background" style={{ zIndex: shown.length - i }}>
          <ActorAvatarBubble id={a.id} username={a.username} />
        </div>
      ))}
    </div>
  )
}

// ── NotificationGroupItem ─────────────────────────────────────────────────────

function NotificationGroupItem({ group }: { group: NotificationGroup }) {
  const config = TYPE_CONFIG[group.type] ?? { icon: Bell, color: 'text-muted-foreground', singular: group.type, plural: group.type }
  const { icon: Icon, color, singular, plural } = config
  const label = group.actors.length > 1 ? plural : singular
  const navigate = useNavigate()
  const qc = useQueryClient()
  const destination = getGroupDestination(group)

  const handleClick = () => {
    if (!group.read) {
      group.allIds.forEach((id) => markNotificationRead(id).catch(() => {}))
      qc.setQueryData<{ data: Notification[]; total: number }>(['notifications'], (old) =>
        old ? { ...old, data: old.data.map((n) => group.allIds.includes(n.id) ? { ...n, read: true } : n) } : old
      )
      qc.invalidateQueries({ queryKey: ['notifications-unread-count'] })
    }
    if (destination) navigate(destination)
  }

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    group.allIds.forEach((id) => deleteNotification(id).catch(() => {}))
    qc.setQueryData<{ data: Notification[]; total: number }>(['notifications'], (old) =>
      old ? { ...old, data: old.data.filter((n) => !group.allIds.includes(n.id)) } : old
    )
    qc.invalidateQueries({ queryKey: ['notifications-unread-count'] })
  }

  return (
    <article
      className={`group/item relative flex items-start gap-3 px-4 py-3.5 border-b border-border transition-all hover:bg-accent/50 hover:-translate-y-px ${
        destination ? 'cursor-pointer' : 'cursor-default'
      } ${!group.read ? 'bg-primary/5' : ''}`}
      onClick={handleClick}
    >
      {!group.read && (
        <span className="absolute left-2 top-1/2 -translate-y-1/2 h-1.5 w-1.5 rounded-full bg-primary" />
      )}

      <div className={`mt-0.5 shrink-0 ${color}`}>
        <Icon size={18} />
      </div>

      <div className="flex-1 min-w-0">
        <p className="text-sm text-foreground leading-snug">
          <ActorsText actors={group.actors} />
          {' '}
          <span className="text-foreground/80">{label}</span>
        </p>
        {group.actors.length > 1 && <ActorAvatars actors={group.actors} />}
        <p className="text-xs text-muted-foreground mt-0.5">{timeAgo(group.created_at)}</p>
      </div>

      <button
        type="button"
        aria-label="Supprimer"
        className="opacity-0 group-hover/item:opacity-100 shrink-0 mt-0.5 p-1 rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-all"
        onClick={handleDelete}
      >
        <X size={14} />
      </button>
    </article>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────


type Filter = 'all' | 'unread'

export function NotificationsPage() {
  const [filter, setFilter] = useState<Filter>('all')
  const { data, isLoading, refetch } = useNotifications()
  const { data: unreadCount = 0 } = useUnreadCount()
  const { data: followRequests } = useFollowRequests()
  const pendingRequests = followRequests?.total ?? 0
  const markAllRead = useMarkAllRead()

  const groups = useMemo(() => {
    const notifications = data?.data ?? []
    const all = groupNotifications(notifications)
    return filter === 'unread' ? all.filter((g) => !g.read) : all
  }, [data?.data, filter])

  return (
    <div>
      <div className="px-4 py-3 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm">
        <div className="flex items-center justify-between mb-3">
          <h1 className="text-base font-semibold text-foreground">Notifications</h1>
          {unreadCount > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="gap-1.5 text-xs text-muted-foreground hover:text-foreground h-8 px-2"
              onClick={() => markAllRead.mutate()}
              disabled={markAllRead.isPending}
            >
              <CheckCheck size={14} />
              Tout marquer comme lu
            </Button>
          )}
        </div>

        <div className="flex gap-1">
          {(['all', 'unread'] as Filter[]).map((f) => (
            <button
              key={f}
              type="button"
              onClick={() => setFilter(f)}
              className={`px-3 py-1 text-xs font-medium rounded-full transition-colors ${
                filter === f
                  ? 'bg-primary/20 text-primary border border-primary/30'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent'
              }`}
            >
              {f === 'all' ? 'Toutes' : `Non lues${unreadCount > 0 ? ` (${unreadCount})` : ''}`}
            </button>
          ))}
        </div>
      </div>

      {pendingRequests > 0 && (
        <Link
          to="/follow-requests"
          className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border hover:bg-accent/30 transition-colors"
        >
          <span className="flex items-center gap-2 text-sm text-foreground">
            <UserPlus size={16} className="text-amber-500" />
            {pendingRequests} demande{pendingRequests > 1 ? 's' : ''} d'abonnement
          </span>
          <span className="text-xs text-primary">Voir</span>
        </Link>
      )}

      {isLoading ? (
        <div className="flex flex-col">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="flex items-start gap-3 px-4 py-3.5 border-b border-border">
              <Skeleton className="h-5 w-5 rounded-full shrink-0 mt-0.5" />
              <div className="flex-1 space-y-1.5">
                <Skeleton className="h-3.5 w-3/5 rounded" />
                <Skeleton className="h-3 w-1/4 rounded" />
              </div>
            </div>
          ))}
        </div>
      ) : groups.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 gap-4 text-muted-foreground">
          <Bell size={40} strokeWidth={1.25} />
          <p className="text-sm">
            {filter === 'unread' ? 'Aucune notification non lue' : 'Aucune notification'}
          </p>
          <Button variant="ghost" size="sm" className="text-xs" onClick={() => refetch()}>
            Rafraîchir
          </Button>
        </div>
      ) : (
        <div>
          {groups.map((g) => (
            <NotificationGroupItem key={g.key} group={g} />
          ))}
        </div>
      )}
    </div>
  )
}
