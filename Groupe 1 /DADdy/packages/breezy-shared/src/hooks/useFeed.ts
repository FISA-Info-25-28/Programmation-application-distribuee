import { useInfiniteQuery, useQuery, useQueries } from '@tanstack/react-query'
import { getFeed } from '../api/feed'
import { getFollowing, getUserPosts, getUserRebreezed } from '../api/users'
import { useAuth } from './useAuth'
import { useBlockedIds, useBlockerIds } from './useUser'
import type { Post, RebreezerInfo } from '../types/api'

export const nextPage = (last: { page: number; limit: number; data: unknown[]; total: number }) => {
  const loaded = (last.page - 1) * last.limit + last.data.length
  return loaded < last.total ? last.page + 1 : undefined
}

export function useFeed() {
  return useInfiniteQuery({
    queryKey: ['feed', 'for-you'],
    queryFn: ({ pageParam = 1 }) => getFeed(pageParam as number),
    initialPageParam: 1,
    getNextPageParam: nextPage,
  })
}

// Returns a Map<postId, RebreezerInfo> for posts rebreezed by followed users.
export function useRebreezeMap(): Map<string, RebreezerInfo> {
  const { user } = useAuth()

  const { data: followingPage } = useQuery({
    queryKey: ['following', user?.id],
    queryFn: () => getFollowing(user!.id),
    enabled: !!user?.id,
  })

  const followedUsers = followingPage?.data ?? []

  const rebreezeResults = useQueries({
    queries: followedUsers.map((u) => ({
      queryKey: ['user-rebreezed', u.id],
      queryFn: () => getUserRebreezed(u.id),
    })),
  })

  const map = new Map<string, RebreezerInfo>()
  rebreezeResults.forEach((r, i) => {
    const u = followedUsers[i]
    if (!u) return
    const rebreezerInfo: RebreezerInfo = {
      id: u.id,
      username: u.username,
      displayName: (u as { displayName?: string | null }).displayName ?? null,
    }
    r.data?.data.forEach((p) => map.set(p.id, rebreezerInfo))
  })

  return map
}

export function useFeedFollowing(): { posts: Post[]; isLoading: boolean } {
  const { user } = useAuth()
  const blockedIds = useBlockedIds()
  const blockerIds = useBlockerIds()

  const { data: followingPage, isLoading: followingLoading } = useQuery({
    queryKey: ['following', user?.id],
    queryFn: () => getFollowing(user!.id),
    enabled: !!user?.id,
  })

  const followedUsers = followingPage?.data ?? []

  const postsResults = useQueries({
    queries: followedUsers.map((u) => ({
      queryKey: ['user-posts', u.id],
      queryFn: () => getUserPosts(u.id),
    })),
  })

  const rebreezeResults = useQueries({
    queries: followedUsers.map((u) => ({
      queryKey: ['user-rebreezed', u.id],
      queryFn: () => getUserRebreezed(u.id),
    })),
  })

  const isLoading =
    followingLoading ||
    postsResults.some((r) => r.isLoading) ||
    rebreezeResults.some((r) => r.isLoading)

  const merged = new Map<string, Post>()

  const isHidden = (id: string) => blockedIds.has(id) || blockerIds.has(id)

  postsResults.forEach((r) => {
    r.data?.data.forEach((p) => {
      if (!isHidden(p.authorId)) merged.set(p.id, p)
    })
  })

  rebreezeResults.forEach((r, i) => {
    const u = followedUsers[i]
    if (!u || isHidden(u.id)) return
    const rebreezerInfo: RebreezerInfo = {
      id: u.id,
      username: u.username,
      displayName: (u as { displayName?: string | null }).displayName ?? null,
    }
    r.data?.data.forEach((p) => {
      if (!isHidden(p.authorId)) merged.set(p.id, { ...p, rebreezerInfo })
    })
  })

  const posts = [...merged.values()].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
  )

  return { posts, isLoading }
}
