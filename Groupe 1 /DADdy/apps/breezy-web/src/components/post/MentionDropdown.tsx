import { UserAvatar } from '@/components/UserAvatar'
import { Skeleton } from '@/components/ui/skeleton'
import type { User } from '@/types/api'

interface MentionDropdownProps {
  active: boolean
  suggestions: User[]
  isLoading: boolean
  onSelect: (user: User) => void
  topOffset?: number
}

export function MentionDropdown({ active, suggestions, isLoading, onSelect, topOffset }: MentionDropdownProps) {
  if (!active) return null
  if (!isLoading && suggestions.length === 0) return null

  return (
    <div
      className="absolute left-0 right-0 z-50 rounded-xl border border-primary/40 bg-card shadow-lg overflow-hidden"
      style={topOffset !== undefined ? { top: topOffset } : { marginTop: 4 }}
    >
      {isLoading ? (
        <div className="px-3 py-2 space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex items-center gap-2">
              <Skeleton className="w-6 h-6 rounded-full shrink-0" />
              <Skeleton className="h-3 w-32" />
            </div>
          ))}
        </div>
      ) : (
        suggestions.map((user) => (
          <button
            key={user.id}
            type="button"
            onMouseDown={(e) => {
              // mouseDown avant blur du textarea → on empêche la perte de focus
              e.preventDefault()
              onSelect(user)
            }}
            className="w-full flex items-center gap-2.5 px-3 py-2 hover:bg-accent transition-colors text-left"
          >
            <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={24} />
            <div className="min-w-0">
              <span className="text-sm font-medium text-foreground">@{user.username}</span>
              {user.bio && (
                <p className="text-xs text-muted-foreground truncate">{user.bio}</p>
              )}
            </div>
          </button>
        ))
      )}
    </div>
  )
}
