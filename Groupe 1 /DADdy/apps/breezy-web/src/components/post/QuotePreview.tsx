import { useNavigate } from 'react-router-dom'
import { UserAvatar } from '@/components/UserAvatar'
import { useUser } from '@/hooks/useUser'
import { timeAgo } from '@/lib/time'
import { useLanguage } from '@/hooks/useLanguage'
import type { Post } from '@/types/api'

interface QuotePreviewProps {
  post: Post
}

export function QuotePreview({ post }: QuotePreviewProps) {
  const navigate = useNavigate()
  const { t, locale } = useLanguage()
  const { data: authorProfile } = useUser(post.author.id)
  const avatarUrl = authorProfile?.avatarUrl ?? post.author.avatarUrl ?? null
  const authorName = authorProfile?.displayName?.trim() || post.author.username

  const firstImage = post.media?.find((m) => m.media_type === 'image')

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={(e) => { e.stopPropagation(); navigate(`/post/${post.id}`) }}
      onKeyDown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); navigate(`/post/${post.id}`) } }}
      className="mt-2 rounded-xl border border-border bg-accent/20 hover:bg-accent/40 transition-colors cursor-pointer overflow-hidden"
    >
      <div className="px-3 py-2.5">
        <div className="flex items-center gap-1.5 mb-1">
          <UserAvatar username={post.author.username} avatarUrl={avatarUrl} size={16} />
          <span className="text-xs font-medium text-foreground truncate">{authorName}</span>
          <span className="text-xs text-muted-foreground truncate">@{post.author.username}</span>
          <span className="text-xs text-muted-foreground shrink-0">· {timeAgo(post.createdAt, t.common, locale)}</span>
        </div>
        {post.content && (
          <p className="text-sm text-foreground/80 line-clamp-3 whitespace-pre-wrap leading-snug">
            {post.content}
          </p>
        )}
        {!post.content && post.media && post.media.length > 0 && (
          <p className="text-xs text-muted-foreground italic">[média]</p>
        )}
        {post.poll && (
          <p className="text-xs text-muted-foreground italic mt-0.5">[sondage]</p>
        )}
      </div>
      {firstImage && (
        <img
          src={firstImage.url}
          alt=""
          className="w-full max-h-40 object-cover border-t border-border"
        />
      )}
    </div>
  )
}
