import { useRef, useEffect, useState, useCallback } from 'react'
import { ArrowUp, Home, RefreshCw } from 'lucide-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { PostComposer } from '@/components/post/PostComposer'
import { PostCard } from '@/components/post/PostCard'
import { Skeleton } from '@/components/ui/skeleton'
import { useFeed, useFeedFollowing, useRebreezeMap } from '@/hooks/useFeed'
import { useBlockedIds, useBlockerIds } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import { useHaptics } from '@/hooks/useHaptics'
import { cn } from '@/lib/utils'

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

function ForYouFeed() {
  const { t } = useLanguage()
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useFeed()
  const rebreezeMap = useRebreezeMap()
  const blockedIds = useBlockedIds()
  const blockerIds = useBlockerIds()
  const loaderRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => { if (entries[0].isIntersecting && hasNextPage) fetchNextPage() },
      { threshold: 0.1 },
    )
    const el = loaderRef.current
    if (el) observer.observe(el)
    return () => { if (el) observer.unobserve(el) }
  }, [fetchNextPage, hasNextPage])

  const posts = (data?.pages.flatMap((p) => p.data) ?? []).filter((p) => {
    const isHidden = (id: string) => blockedIds.has(id) || blockerIds.has(id)
    return !isHidden(p.authorId) && !isHidden(p.rebreezerInfo?.id ?? '')
  })

  if (isLoading) {
    return <>{Array.from({ length: 5 }).map((_, i) => <PostSkeleton key={i} />)}</>
  }

  if (posts.length === 0) {
    return (
      <div className="py-16 text-center text-sm text-zinc-400">
        {t.common.endOfFeed}
      </div>
    )
  }

  return (
    <>
      {posts.map((post) => {
        const rebreezerInfo = rebreezeMap.get(post.id)
        return <PostCard key={post.id} post={rebreezerInfo ? { ...post, rebreezerInfo } : post} />
      })}
      <div ref={loaderRef} className="py-4 text-center">
        {isFetchingNextPage && <PostSkeleton />}
        {!hasNextPage && posts.length > 0 && (
          <p className="text-xs text-zinc-400">{t.common.endOfFeed}</p>
        )}
      </div>
    </>
  )
}

function FollowingFeed() {
  const { t } = useLanguage()
  const { posts, isLoading } = useFeedFollowing()

  if (isLoading) {
    return <>{Array.from({ length: 5 }).map((_, i) => <PostSkeleton key={i} />)}</>
  }

  if (posts.length === 0) {
    return (
      <div className="py-16 text-center text-sm text-zinc-400">
        {t.feed.emptyFollowing}
      </div>
    )
  }

  return <>{posts.map((post) => <PostCard key={post.id} post={post} />)}</>
}

type Tab = 'for-you' | 'following'

function NewPostsPill({ onLoad }: { onLoad: () => void }) {
  const { t } = useLanguage()
  const { data: count = 0 } = useQuery<number>({
    queryKey: ['feed-new-posts-count'],
    queryFn: () => 0,
    staleTime: Infinity,
    gcTime: Infinity,
    initialData: 0,
  })

  if (count === 0) return null

  return (
    <div className="flex justify-center pt-3 pb-1 pointer-events-none">
      <button
        onClick={onLoad}
        className="pointer-events-auto flex items-center gap-1.5 px-4 py-1.5 rounded-full bg-primary text-primary-foreground text-xs font-semibold shadow-lg hover:bg-primary/90 transition-colors"
        style={{ animation: 'pill-pulse 2s ease-in-out infinite' }}
      >
        <ArrowUp size={13} strokeWidth={2.5} />
        {count} {t.feed.newPosts}
      </button>
    </div>
  )
}

export function FeedPage() {
  const { t } = useLanguage()
  const { impact, ImpactStyle } = useHaptics()
  const [tab, setTab] = useState<Tab>('for-you')
  const [pullY, setPullY] = useState(0)
  const [refreshing, setRefreshing] = useState(false)
  const pullStartY = useRef<number | null>(null)
  const qc = useQueryClient()
  const feedTopRef = useRef<HTMLDivElement>(null)

  const loadNewPosts = useCallback(() => {
    qc.invalidateQueries({ queryKey: ['feed', 'for-you'] })
    qc.invalidateQueries({ queryKey: ['following'] })
    qc.invalidateQueries({ queryKey: ['user-posts'] })
    qc.invalidateQueries({ queryKey: ['user-rebreezed'] })
    qc.setQueryData(['feed-new-posts-count'], 0)
    feedTopRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [qc])

  const PULL_THRESHOLD = 72

  const handleTouchStart = (e: React.TouchEvent) => {
    const el = e.currentTarget.closest('.overflow-y-auto') as HTMLElement | null
    if (el && el.scrollTop === 0) pullStartY.current = e.touches[0].clientY
  }

  const handleTouchMove = (e: React.TouchEvent) => {
    if (pullStartY.current === null || refreshing) return
    const delta = e.touches[0].clientY - pullStartY.current
    if (delta > 0) setPullY(Math.min(delta * 0.4, PULL_THRESHOLD + 16))
  }

  const handleTouchEnd = () => {
    if (pullY >= PULL_THRESHOLD && !refreshing) {
      impact(ImpactStyle.Medium)
      setRefreshing(true)
      const start = Date.now()
      loadNewPosts()
      const MIN_DELAY = 1200
      const elapsed = Date.now() - start
      setTimeout(() => setRefreshing(false), Math.max(MIN_DELAY - elapsed, MIN_DELAY))
    }
    setPullY(0)
    pullStartY.current = null
  }

  return (
    <div onTouchStart={handleTouchStart} onTouchMove={handleTouchMove} onTouchEnd={handleTouchEnd}>
      {/* Indicateur pull-to-refresh */}
      {pullY > 0 && (
        <div
          className="flex justify-center items-center overflow-hidden transition-none"
          style={{ height: pullY }}
        >
          <RefreshCw
            size={20}
            className={cn('text-primary transition-transform', refreshing && 'animate-spin')}
            style={{ transform: `rotate(${pullY * 3}deg)` }}
          />
        </div>
      )}
      <div ref={feedTopRef} className="sticky top-0 z-10 bg-background/80 backdrop-blur-sm border-b border-border">
        <div className="px-4 py-3">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/15 text-primary ring-1 ring-primary/25">
              <Home size={18} strokeWidth={2.4} aria-hidden="true" />
            </div>
            <h1 className="text-base font-semibold text-foreground">{t.feed.title}</h1>
          </div>
        </div>
        <div className="flex relative">
          <button
            onClick={() => setTab('for-you')}
            className={cn(
              'flex-1 py-3 text-sm font-medium transition-colors',
              tab === 'for-you' ? 'text-foreground' : 'text-zinc-400 hover:text-foreground',
            )}
          >
            {t.feed.forYou}
          </button>
          <button
            onClick={() => setTab('following')}
            className={cn(
              'flex-1 py-3 text-sm font-medium transition-colors',
              tab === 'following' ? 'text-foreground' : 'text-zinc-400 hover:text-foreground',
            )}
          >
            {t.feed.following}
          </button>
          <div
            className="absolute bottom-0 w-1/2 transition-transform duration-200 ease-out pointer-events-none"
            style={{ transform: tab === 'following' ? 'translateX(100%)' : 'translateX(0)' }}
          >
            <div className="mx-auto w-12 h-0.5 rounded-full bg-foreground" />
          </div>
        </div>
      </div>

      <NewPostsPill onLoad={loadNewPosts} />

      {tab === 'for-you' && <PostComposer />}

      {tab === 'for-you' ? <ForYouFeed /> : <FollowingFeed />}
    </div>
  )
}
