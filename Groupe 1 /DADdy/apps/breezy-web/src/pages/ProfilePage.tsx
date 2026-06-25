import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Repeat2, Lock, Heart, ShieldOff } from 'lucide-react'
import { ProfileHeader, ProfileHeaderSkeleton } from '@/components/profile/ProfileHeader'
import { PostCard } from '@/components/post/PostCard'
import { ContextPost } from '@/components/post/ContextPost'
import { Skeleton } from '@/components/ui/skeleton'
import { useUser, useUserPosts, useUserRebreezed, useUserLiked } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'

function PostSkeleton() {
  return (
    <div className="px-4 py-4 border-b border-border flex gap-3">
      <Skeleton className="w-9 h-9 rounded-full shrink-0" />
      <div className="flex-1 space-y-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-3/4" />
      </div>
    </div>
  )
}

export function ProfilePage() {
  const { t } = useLanguage()
  const { id } = useParams<{ id: string }>()
  const [tab, setTab] = useState<'posts' | 'rebreezed' | 'likes'>('posts')

  const { data: user, isLoading: userLoading } = useUser(id ?? '')
  const isBlocked = !!(user?.isBlockedByMe || user?.hasBlockedMe)
  // Compte privé non suivi → on n'interroge même pas les posts (l'API renverrait 403).
  const canViewPosts = user ? user.canViewPosts !== false : true
  const gatedId = isBlocked || !canViewPosts ? '' : (id ?? '')
  const { data: posts, isLoading: postsLoading } = useUserPosts(gatedId)
  const { data: rebreezed, isLoading: rebreezedLoading } = useUserRebreezed(gatedId)
  const { data: liked, isLoading: likedLoading } = useUserLiked(gatedId)

  const activeItems =
    tab === 'posts' ? posts?.data ?? [] : tab === 'rebreezed' ? rebreezed?.data ?? [] : liked?.data ?? []
  const isLoading = tab === 'posts' ? postsLoading : tab === 'rebreezed' ? rebreezedLoading : likedLoading
  const isPrivate = !!user && !canViewPosts && !isBlocked

  return (
    <div>
      <div className="px-4 py-4 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm">
        <h1 className="text-base font-semibold text-foreground">{t.profile.title}</h1>
      </div>

      {userLoading ? (
        <ProfileHeaderSkeleton />
      ) : user ? (
        <ProfileHeader key={user.id} user={user} />
      ) : (
        <div className="px-4 py-8 text-center text-sm text-muted-foreground">{t.profile.notFound}</div>
      )}

      {/* Contenu bloqué */}
      {isBlocked && (
        <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
          <ShieldOff size={32} className="text-destructive/60" />
          <p className="text-sm font-medium text-foreground">
            {user?.isBlockedByMe ? 'Utilisateur bloqué' : 'Contenu non disponible'}
          </p>
          <p className="text-xs text-center max-w-xs">
            {user?.isBlockedByMe
              ? 'Vous avez bloqué cet utilisateur. Son contenu est masqué.'
              : 'Vous ne pouvez pas voir le contenu de ce profil.'}
          </p>
        </div>
      )}

      {/* Compte privé non suivi */}
      {isPrivate && (
        <div className="flex flex-col items-center justify-center gap-3 px-4 py-16 text-center text-muted-foreground">
          <Lock size={36} strokeWidth={1.25} />
          <p className="text-sm font-medium text-foreground">{t.profile.privateTitle}</p>
          <p className="text-xs max-w-xs">{t.profile.privateHint}</p>
        </div>
      )}

      {!isBlocked && !isPrivate && (
        <>
          {/* Tabs */}
          <div className="flex border-b border-border">
            <button
              onClick={() => setTab('posts')}
              className={`flex-1 py-3 text-sm font-medium transition-colors ${
                tab === 'posts'
                  ? 'text-foreground border-b-2 border-primary'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {t.search.tabPosts}
            </button>
            <button
              onClick={() => setTab('rebreezed')}
              className={`flex-1 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-1.5 ${
                tab === 'rebreezed'
                  ? 'text-emerald-500 border-b-2 border-emerald-500'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Repeat2 size={14} />
              {t.post.rebreeze}
            </button>
            <button
              onClick={() => setTab('likes')}
              className={`flex-1 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-1.5 ${
                tab === 'likes'
                  ? 'text-rose-500 border-b-2 border-rose-500'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Heart size={14} />
              {t.post.likes}
            </button>
          </div>

          <div>
            {isLoading
              ? Array.from({ length: 3 }).map((_, i) => <PostSkeleton key={i} />)
              : activeItems.length === 0
                ? (
                  <div className="px-4 py-12 text-center text-sm text-muted-foreground">
                    {tab === 'rebreezed' ? t.post.noRebreeze : tab === 'likes' ? t.post.noLikes : t.post.noPosts}
                  </div>
                )
                : activeItems.map((post) => (
                  tab === 'rebreezed' ? (
                    <div key={post.id} className="relative">
                      <div className="flex items-center gap-1.5 px-4 pt-3 text-xs text-emerald-500">
                        <Repeat2 size={12} />
                        <span>{t.post.rebreeze}</span>
                      </div>
                      <PostCard post={post} />
                    </div>
                  ) : tab === 'posts' && post.parentId ? (
                    // Réponse : on affiche le post original au-dessus (avec trait
                    // de fil) puis le commentaire en dessous, façon Twitter/X.
                    <div key={post.id}>
                      <ContextPost id={post.parentId} />
                      <PostCard post={post} inThread />
                    </div>
                  ) : (
                    <PostCard key={post.id} post={post} />
                  )
                ))}
          </div>
        </>
      )}
    </div>
  )
}
