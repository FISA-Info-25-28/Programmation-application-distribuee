import { useRef, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { useInfiniteQuery } from '@tanstack/react-query'
import { PostCard } from '@/components/post/PostCard'
import { Skeleton } from '@/components/ui/skeleton'
import { getHashtagFeed } from '@/api/posts'
import { useLanguage } from '@/hooks/useLanguage'

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

export function HashtagPage() {
  const { t } = useLanguage()
  const { name = '' } = useParams<{ name: string }>()
  const loaderRef = useRef<HTMLDivElement>(null)

  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useInfiniteQuery({
    queryKey: ['hashtag-feed', name],
    queryFn: ({ pageParam = 1 }) => getHashtagFeed(name, pageParam as number),
    initialPageParam: 1,
    getNextPageParam: (last) => {
      const loaded = (last.page - 1) * last.limit + last.data.length
      return loaded < last.total ? last.page + 1 : undefined
    },
    enabled: name.length > 0,
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
  const total = data?.pages[0]?.total ?? 0

  return (
    <div>
      <div className="px-4 py-4 border-b border-border sticky top-0 bg-background/80 backdrop-blur-sm flex items-center gap-3">
        <Link
          to="/"
          className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
        >
          <ArrowLeft size={18} />
        </Link>
        <div>
          <h1 className="text-base font-semibold text-foreground">#{name}</h1>
          {!isLoading && (
            <p className="text-xs text-muted-foreground">
              {total} {total > 1 ? t.post.postPlural : t.post.postSingular}
            </p>
          )}
        </div>
      </div>

      {isLoading
        ? Array.from({ length: 5 }).map((_, i) => <PostSkeleton key={i} />)
        : posts.map((post) => <PostCard key={post.id} post={post} />)}

      <div ref={loaderRef} className="py-4 text-center">
        {isFetchingNextPage && <PostSkeleton />}
        {!isLoading && !hasNextPage && posts.length === 0 && (
          <p className="text-xs text-muted-foreground pt-8">{t.hashtag.empty} #{name}</p>
        )}
        {!hasNextPage && posts.length > 0 && (
          <p className="text-xs text-zinc-400">{t.hashtag.end}</p>
        )}
      </div>
    </div>
  )
}
