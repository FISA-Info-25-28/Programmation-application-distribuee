import { useState, useEffect, useRef } from 'react'
import { useSearchParams, useNavigate, Link } from 'react-router-dom'
import { ArrowLeft, Search, Hash, X } from 'lucide-react'
import { PostCard } from '@/components/post/PostCard'
import { UserAvatar } from '@/components/UserAvatar'
import { Skeleton } from '@/components/ui/skeleton'
import { useSearchPosts } from '@/hooks/usePost'
import { useSearchUsers, useBlockedIds, useBlockerIds } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import type { User } from '@/types/api'

type Tab = 'posts' | 'users' | 'tags'

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

function UserSkeleton() {
  return (
    <div className="px-4 py-3 border-b border-border flex items-center gap-3">
      <Skeleton className="w-10 h-10 rounded-full shrink-0" />
      <div className="flex-1 space-y-1.5">
        <Skeleton className="h-3 w-28" />
        <Skeleton className="h-3 w-44" />
      </div>
    </div>
  )
}

function UserRow({ user }: { user: User }) {
  const { t } = useLanguage()

  return (
    <Link
      to={`/profile/${user.id}`}
      className="flex items-center gap-3 px-4 py-3 border-b border-border hover:bg-accent/30 transition-colors"
    >
      <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={40} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-foreground truncate">
          {user.displayName?.trim() || user.username}
        </p>
        <p className="text-xs text-muted-foreground truncate">@{user.username}</p>
        {user.bio && (
          <p className="text-xs text-muted-foreground truncate mt-0.5">{user.bio}</p>
        )}
      </div>
      <div className="text-xs text-muted-foreground shrink-0">
        {user.followersCount} {t.common.followers}
      </div>
    </Link>
  )
}

function EmptyState({ q, tab }: { q: string; tab: Tab }) {
  const { t } = useLanguage()
  const labels: Record<Tab, string> = {
    posts: t.search.emptyPost,
    users: t.search.emptyUser,
    tags: t.search.emptyHashtag,
  }
  return (
    <div className="px-4 py-16 flex flex-col items-center gap-2 text-muted-foreground">
      <Search size={32} strokeWidth={1.5} />
      <p className="text-sm font-medium">{labels[tab]} &ldquo;{q}&rdquo;</p>
      <p className="text-xs">{t.search.trySomething}</p>
    </div>
  )
}

export function SearchPage() {
  const { t } = useLanguage()
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const initialQ = params.get('q') ?? ''

  const [inputValue, setInputValue] = useState(initialQ)
  const [debouncedQ, setDebouncedQ] = useState(initialQ)
  const [tab, setTab] = useState<Tab>('posts')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => { inputRef.current?.focus() }, [])

  useEffect(() => {
    const q = params.get('q') ?? ''
    const timer = setTimeout(() => {
      setInputValue(q)
      setDebouncedQ(q)
    }, 0)
    return () => clearTimeout(timer)
  }, [params])

  useEffect(() => {
    const timer = setTimeout(() => {
      const q = inputValue.trim()
      setDebouncedQ(q)
      if (q) setParams({ q }, { replace: true })
      else setParams({}, { replace: true })
    }, 300)
    return () => clearTimeout(timer)
  }, [inputValue, setParams])

  const { data: postsData, isLoading: postsLoading } = useSearchPosts(debouncedQ)
  const { data: usersData, isLoading: usersLoading } = useSearchUsers(debouncedQ)
  const blockedIds = useBlockedIds()
  const blockerIds = useBlockerIds()

  const isHidden = (id: string) => blockedIds.has(id) || blockerIds.has(id)
  const posts = (postsData?.data ?? []).filter((p) => !isHidden(p.authorId))
  const users = (usersData?.data ?? []).filter((u) => !isHidden(u.id))
  const tagPosts = debouncedQ
    ? posts.filter((p) => p.tags.some((tg) => tg.toLowerCase().includes(debouncedQ.toLowerCase())))
    : []

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: 'posts', label: t.search.tabPosts, count: postsData?.total },
    { key: 'users', label: t.search.tabUsers, count: usersData?.total },
    { key: 'tags', label: t.search.tabHashtags, count: tagPosts.length },
  ]

  return (
    <div>
      <div className="sticky top-0 z-10 bg-background/80 backdrop-blur-sm border-b border-border">
        <div className="px-4 py-3 flex items-center gap-3">
          <button
            onClick={() => navigate(-1)}
            className="p-1.5 -ml-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors shrink-0"
          >
            <ArrowLeft size={18} />
          </button>

          <div className="flex-1 relative">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none" />
            <input
              ref={inputRef}
              type="text"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              placeholder={t.search.placeholder}
              className="w-full pl-9 pr-8 py-2 rounded-full bg-accent border border-transparent text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-border focus:bg-background transition-colors"
            />
            {inputValue && (
              <button
                onClick={() => setInputValue('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
              >
                <X size={14} />
              </button>
            )}
          </div>
        </div>

        {debouncedQ && (
          <div className="flex">
            {tabs.map(({ key, label, count }) => (
              <button
                key={key}
                onClick={() => setTab(key)}
                className={`flex-1 py-3 text-sm font-medium transition-colors relative ${
                  tab === key ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {label}
                {count !== undefined && count > 0 && (
                  <span className="ml-1.5 text-xs text-muted-foreground">({count})</span>
                )}
                {tab === key && (
                  <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary rounded-full" />
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      {!debouncedQ ? (
        <div className="px-4 py-16 flex flex-col items-center gap-2 text-muted-foreground">
          <Search size={32} strokeWidth={1.5} />
          <p className="text-sm">{t.search.startTyping}</p>
        </div>
      ) : tab === 'posts' ? (
        postsLoading
          ? Array.from({ length: 5 }).map((_, i) => <PostSkeleton key={i} />)
          : posts.length === 0
          ? <EmptyState q={debouncedQ} tab="posts" />
          : posts.map((post) => <PostCard key={post.id} post={post} />)
      ) : tab === 'users' ? (
        usersLoading
          ? Array.from({ length: 4 }).map((_, i) => <UserSkeleton key={i} />)
          : users.length === 0
          ? <EmptyState q={debouncedQ} tab="users" />
          : users.map((user) => <UserRow key={user.id} user={user} />)
      ) : (
        tagPosts.length === 0
          ? (
            <div className="px-4 py-16 flex flex-col items-center gap-2 text-muted-foreground">
              <Hash size={32} strokeWidth={1.5} />
              <p className="text-sm font-medium">{t.search.emptyHashtag} &ldquo;{debouncedQ}&rdquo;</p>
              <p className="text-xs">{t.search.tryHashtag}</p>
            </div>
          )
          : (
            <div>
              <div className="px-4 py-3 border-b border-border bg-accent/20">
                <p className="text-xs text-muted-foreground">
                  {t.search.postsWithHashtag} &ldquo;{debouncedQ}&rdquo;
                </p>
              </div>
              {tagPosts.map((post) => <PostCard key={post.id} post={post} />)}
            </div>
          )
      )}
    </div>
  )
}
