import { Link, useNavigate } from 'react-router-dom'
import { Skeleton } from '@/components/ui/skeleton'
import { UserAvatar } from '@/components/UserAvatar'
import { usePost } from '@/hooks/usePost'
import { useUser } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import { timeAgo } from '@/lib/time'

/**
 * Remonte récursivement la chaîne des parents et affiche chaque ancêtre du post
 * avec un trait de fil qui le relie au post suivant (style Twitter/X). Utilisé
 * dans la page d'un post et dans l'onglet « Posts » du profil pour donner le
 * contexte d'une réponse (on voit le post original au-dessus du commentaire).
 */
export function ContextPost({ id }: { id: string }) {
  const { data: post, isLoading } = usePost(id)
  const navigate = useNavigate()
  const { t, locale } = useLanguage()
  const { data: authorProfile } = useUser(post?.author.id ?? '')
  const avatarUrl = authorProfile?.avatarUrl ?? post?.author.avatarUrl ?? null

  return (
    <>
      {/* L'ancêtre le plus lointain s'affiche en premier (récursion vers le haut) */}
      {post?.parentId && <ContextPost id={post.parentId} />}

      {isLoading && !post ? (
        <div className="flex gap-3 px-4 pt-4">
          <div className="flex flex-col items-center w-8 shrink-0">
            <Skeleton className="w-8 h-8 rounded-full" />
            <div className="w-px bg-border/40 flex-1 mt-2 min-h-8" />
          </div>
          <div className="flex-1 pb-4 space-y-2 pt-0.5">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-3 w-3/4" />
          </div>
        </div>
      ) : post ? (
        <div
          className="flex gap-3 px-4 pt-4 cursor-pointer hover:bg-muted/40 transition-colors"
          onClick={() => navigate(`/post/${post.id}`)}
        >
          <div className="flex flex-col items-center w-8 shrink-0">
            <Link to={`/profile/${post.author.id}`} onClick={(e) => e.stopPropagation()}>
              <UserAvatar username={post.author.username} avatarUrl={avatarUrl} size={32} />
            </Link>
            {/* Trait de fil reliant vers le post suivant */}
            <div className="w-px bg-border/60 flex-1 mt-2" />
          </div>
          <div className="flex-1 min-w-0 pb-3">
            <div className="flex items-baseline gap-2 mb-1">
              <Link
                to={`/profile/${post.author.id}`}
                className="font-medium text-sm text-foreground hover:underline"
                onClick={(e) => e.stopPropagation()}
              >
                @{post.author.username}
              </Link>
              <span className="text-xs text-muted-foreground">
                {timeAgo(post.createdAt, t.common, locale)}
              </span>
            </div>
            <p className="text-sm text-foreground/90 leading-relaxed">{post.content}</p>
          </div>
        </div>
      ) : null}
    </>
  )
}
