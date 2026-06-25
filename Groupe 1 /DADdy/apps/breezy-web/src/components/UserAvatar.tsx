import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar'
import { cn } from '@/lib/utils'

interface UserAvatarProps {
  username: string
  avatarUrl?: string | null
  /** px size — remplace les variants sm/default/lg */
  size?: number
  className?: string
}

/**
 * Affiche la photo de profil si `avatarUrl` est fournie,
 * sinon replie sur l'initiale du nom d'utilisateur.
 */
export function UserAvatar({ username, avatarUrl, size = 36, className }: UserAvatarProps) {
  const initial = username?.[0]?.toUpperCase() ?? '?'

  return (
    <Avatar
      className={cn('bg-primary/10 border border-primary/20', className)}
      style={{ width: size, height: size }}
    >
      {avatarUrl && (
        <AvatarImage
          src={avatarUrl}
          alt={`@${username}`}
        />
      )}
      <AvatarFallback
        className="text-primary font-medium bg-primary/10"
        style={{ fontSize: Math.max(10, Math.round(size * 0.38)) }}
      >
        {initial}
      </AvatarFallback>
    </Avatar>
  )
}
