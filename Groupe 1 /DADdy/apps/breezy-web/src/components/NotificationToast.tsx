import { useEffect, useState } from 'react'
import { Heart, MessageSquare, UserPlus, UserCheck, Newspaper, MessageCircle, AtSign, X, Repeat2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import type { NotificationType } from '@/types/api'

export interface ToastData {
  type: NotificationType
  actorUsername: string
  actorId: string
  entityId: string
}

const CONFIG: Record<
  NotificationType,
  { icon: React.ElementType; label: (username: string) => string; color: string; href: (entityId: string) => string }
> = {
  like: {
    icon: Heart,
    label: (u) => `@${u} a aimé votre publication`,
    color: 'text-rose-500 bg-rose-500/10',
    href: (id) => `/post/${id}`,
  },
  comment: {
    icon: MessageSquare,
    label: (u) => `@${u} a commenté votre publication`,
    color: 'text-blue-500 bg-blue-500/10',
    href: (id) => `/post/${id}`,
  },
  follow: {
    icon: UserPlus,
    label: (u) => `@${u} vous suit maintenant`,
    color: 'text-emerald-500 bg-emerald-500/10',
    href: (id) => `/profile/${id}`,
  },
  new_post: {
    icon: Newspaper,
    label: (u) => `@${u} a publié un nouveau breezy`,
    color: 'text-violet-500 bg-violet-500/10',
    href: (id) => `/post/${id}`,
  },
  new_message: {
    icon: MessageCircle,
    label: (u) => `Nouveau message de @${u}`,
    color: 'text-primary bg-primary/10',
    href: () => '/messages',
  },
  mention: {
    icon: AtSign,
    label: (u) => `@${u} vous a mentionné`,
    color: 'text-amber-500 bg-amber-500/10',
    href: (id) => `/post/${id}`,
  },
  rebreeze: {
    icon: Repeat2,
    label: (u) => `@${u} a rebreezé votre publication`,
    color: 'text-emerald-500 bg-emerald-500/10',
    href: (id) => `/post/${id}`,
  },
  follow_request: {
    icon: UserPlus,
    label: (u) => `@${u} souhaite s'abonner`,
    color: 'text-amber-500 bg-amber-500/10',
    href: () => '/follow-requests',
  },
  follow_request_accepted: {
    icon: UserCheck,
    label: (u) => `@${u} a accepté votre demande d'abonnement`,
    color: 'text-emerald-500 bg-emerald-500/10',
    href: (id) => `/profile/${id}`,
  },
}

interface Props {
  toast: ToastData | null
  onDismiss: () => void
}

export function NotificationToast({ toast, onDismiss }: Props) {
  if (!toast) return null

  return (
    <NotificationToastContent
      key={`${toast.type}-${toast.entityId}-${toast.actorUsername}`}
      toast={toast}
      onDismiss={onDismiss}
    />
  )
}

function NotificationToastContent({ toast, onDismiss }: { toast: ToastData; onDismiss: () => void }) {
  const navigate = useNavigate()
  const [visible, setVisible] = useState(true)

  useEffect(() => {
    const hide = setTimeout(() => setVisible(false), 4000)
    const remove = setTimeout(onDismiss, 4350)
    return () => { clearTimeout(hide); clearTimeout(remove) }
  }, [onDismiss])

  const cfg = CONFIG[toast.type]
  if (!cfg) return null

  const Icon = cfg.icon

  return (
    <div
      className={cn(
        'fixed bottom-6 right-6 z-50 flex max-w-72 items-center gap-3 rounded-xl bg-popover px-4 py-3 shadow-lg ring-1 ring-foreground/10 cursor-pointer transition-all duration-300',
        visible ? 'translate-y-0 opacity-100' : 'translate-y-4 opacity-0',
      )}
      onClick={() => {
        let href: string
        if (toast.type === 'follow') {
          href = `/profile/${toast.actorId}`
        } else if (toast.type === 'new_message') {
          href = '/messages'
        } else {
          href = cfg.href(toast.entityId)
        }
        navigate(href)
        onDismiss()
      }}
    >
      <span className={cn('flex h-8 w-8 shrink-0 items-center justify-center rounded-full', cfg.color)}>
        <Icon size={16} />
      </span>
      <p className="flex-1 text-xs text-foreground leading-snug">
        {cfg.label(toast.actorUsername)}
      </p>
      <button
        className="shrink-0 text-muted-foreground hover:text-foreground"
        onClick={(e) => { e.stopPropagation(); setVisible(false); setTimeout(onDismiss, 350) }}
      >
        <X size={14} />
      </button>
    </div>
  )
}
