import { useRef, useEffect } from 'react'
import { useInfiniteQuery } from '@tanstack/react-query'
import { PostCard } from '@/components/post/PostCard'
import { Skeleton } from '@/components/ui/skeleton'
import { getUserBookmarks } from '@/api/users'
import { nextPage } from '@/hooks/useFeed'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import { bookmarksT } from '@/i18n/bookmarks'

function PostSkeleton() {
  return (
    <div className="px-4 py-4 border-b border-zinc-100 flex gap-3">
      <Skeleton className="w-9 h-9 rounded-full shrink-0" />
      <div className="flex-1 space-y-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-3/4" />
      </div>
    </div>
  )
}

export function BookmarksPage() {
  const { user } = useAuth()
  const { locale } = useLanguage()
  const b = bookmarksT(locale)
  const loaderRef = useRef<HTMLDivElement>(null)

  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useInfiniteQuery({
    queryKey: ['bookmarks', user?.id],
    queryFn: ({ pageParam = 1 }) => getUserBookmarks(user!.id, pageParam as number),
    initialPageParam: 1,
    getNextPageParam: nextPage,
    enabled: !!user?.id,
  })

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => { if (entries[0].isIntersecting && hasNextPage) fetchNextPage() },
      { threshold: 0.1 },
    )
    const el = loaderRef.current
    if (el) observer.observe(el)
    return () => { if (el) observer.unobserve(el) }
  }, [fetchNextPage, hasNextPage])

  const posts = data?.pages.flatMap((p) => p.data) ?? []

  return (
    <div>
      <div className="sticky top-0 z-10 bg-background/80 backdrop-blur-sm border-b border-border px-4 py-3">
        <h1 className="text-base font-semibold text-foreground">{b.title}</h1>
        <p className="text-xs text-muted-foreground">{b.subtitle}</p>
      </div>

      {isLoading ? (
        <>{Array.from({ length: 5 }).map((_, i) => <PostSkeleton key={i} />)}</>
      ) : posts.length === 0 ? (
        <div className="py-16 text-center text-sm text-zinc-400">{b.empty}</div>
      ) : (
        <>
          {posts.map((post) => <PostCard key={post.id} post={post} />)}
          <div ref={loaderRef} className="py-4 text-center">
            {isFetchingNextPage && <PostSkeleton />}
          </div>
        </>
      )}
    </div>
  )
}
