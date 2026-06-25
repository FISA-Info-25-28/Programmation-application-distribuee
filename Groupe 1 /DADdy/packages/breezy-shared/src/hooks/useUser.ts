import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getUser,
  followUser,
  unfollowUser,
  getUserPosts,
  getUserRebreezed,
  getUserLiked,
  updateMe,
  getFollowers,
  getFollowing,
  searchUsers,
  getFollowRequests,
  acceptFollowRequest,
  rejectFollowRequest,
  blockUser,
  unblockUser,
  getBlockedUsers,
  getBlockedByUsers,
} from '../api/users'
import { useAuth } from './useAuth'
import type { User, PaginatedResponse } from '../types/api'

export function useUser(id: string) {
  return useQuery({
    queryKey: ['user', id],
    queryFn: () => getUser(id),
    enabled: !!id,
  })
}

export function useUserPosts(id: string) {
  return useQuery({
    queryKey: ['user-posts', id],
    queryFn: () => getUserPosts(id),
    enabled: !!id,
  })
}

export function useUserRebreezed(id: string) {
  return useQuery({
    queryKey: ['user-rebreezed', id],
    queryFn: () => getUserRebreezed(id),
    enabled: !!id,
  })
}

export function useUserLiked(id: string) {
  return useQuery({
    queryKey: ['user-liked', id],
    queryFn: () => getUserLiked(id),
    enabled: !!id,
  })
}

export function useFollowers(id: string) {
  return useQuery({
    queryKey: ['followers', id],
    queryFn: () => getFollowers(id),
    enabled: !!id,
  })
}

export function useFollowing(id: string) {
  return useQuery({
    queryKey: ['following', id],
    queryFn: () => getFollowing(id),
    enabled: !!id,
  })
}

export function useFollowMutation(userId: string) {
  const qc = useQueryClient()
  const { user: me } = useAuth()
  return useMutation({
    mutationFn: (follow: boolean): Promise<'requested' | 'followed' | void> =>
      follow ? followUser(userId) : unfollowUser(userId).then(() => undefined),
    onSuccess: (result, follow) => {
      // Sur un compte privé, suivre crée une demande (pas un abonnement direct).
      const requested = follow && result === 'requested'
      // Met à jour le profil de la cible (followersCount) et le mien (followingCount)
      qc.invalidateQueries({ queryKey: ['user', userId] })
      if (me?.id) qc.invalidateQueries({ queryKey: ['user', me.id] })
      qc.invalidateQueries({ queryKey: ['followers', userId] })
      qc.invalidateQueries({ queryKey: ['following', userId] })
      qc.invalidateQueries({ queryKey: ['users-suggestions'] })
      // Patch optimiste dans le cache de la cible
      qc.setQueryData<User>(['user', userId], (old) => {
        if (!old) return old
        if (!follow) {
          // Désabonnement ou annulation de demande
          return {
            ...old,
            isFollowedByMe: false,
            followRequested: false,
            canViewPosts: old.isPrivate ? false : old.canViewPosts,
            followersCount: Math.max(0, old.followersCount - (old.isFollowedByMe ? 1 : 0)),
          }
        }
        if (requested) {
          return { ...old, followRequested: true }
        }
        return {
          ...old,
          isFollowedByMe: true,
          followRequested: false,
          canViewPosts: true,
          followersCount: old.followersCount + 1,
        }
      })
    },
  })
}

// ── Demandes d'abonnement (comptes privés) ──────────────────────────────────────

export function useFollowRequests() {
  return useQuery({
    queryKey: ['follow-requests'],
    queryFn: () => getFollowRequests(),
    staleTime: 15_000,
  })
}

export function useAcceptFollowRequest() {
  const qc = useQueryClient()
  const { user: me } = useAuth()
  return useMutation({
    mutationFn: (requesterId: string) => acceptFollowRequest(requesterId),
    onSuccess: (_, requesterId) => {
      // Retire la demande de la liste et rafraîchit compteurs/abonnés.
      qc.setQueryData<PaginatedResponse<User>>(['follow-requests'], (old) =>
        old ? { ...old, data: old.data.filter((u) => u.id !== requesterId), total: Math.max(0, old.total - 1) } : old,
      )
      if (me?.id) {
        qc.invalidateQueries({ queryKey: ['user', me.id] })
        qc.invalidateQueries({ queryKey: ['followers', me.id] })
      }
    },
  })
}

export function useRejectFollowRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (requesterId: string) => rejectFollowRequest(requesterId),
    onSuccess: (_, requesterId) => {
      qc.setQueryData<PaginatedResponse<User>>(['follow-requests'], (old) =>
        old ? { ...old, data: old.data.filter((u) => u.id !== requesterId), total: Math.max(0, old.total - 1) } : old,
      )
    },
  })
}

export function useSearchUsers(q: string) {
  return useQuery({
    queryKey: ['search-users', q],
    queryFn: () => searchUsers(q),
    enabled: q.trim().length > 0,
    staleTime: 0,
  })
}

export function useUpdateMe() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Pick<User, 'displayName' | 'pronouns' | 'bio' | 'isPrivate'>>) => updateMe(data),
    onSuccess: (updated) => {
      qc.setQueryData(['me'], (old: User | null) =>
        old ? { ...old, displayName: updated.displayName, pronouns: updated.pronouns, bio: updated.bio, isPrivate: updated.isPrivate } : old
      )
      qc.setQueryData(['user', updated.id], updated)
    },
  })
}

export function useBlockMutation(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (block: boolean) => block ? blockUser(userId) : unblockUser(userId),
    onMutate: (block) => {
      const previous = qc.getQueryData<User>(['user', userId])
      qc.setQueryData<User>(['user', userId], (old) =>
        old ? { ...old, isBlockedByMe: block } : old
      )
      return { previous }
    },
    onError: (_err, _block, ctx) => {
      if (ctx?.previous) qc.setQueryData(['user', userId], ctx.previous)
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ['user', userId] })
      qc.invalidateQueries({ queryKey: ['blocked-users'] })
      qc.invalidateQueries({ queryKey: ['blocked-by-users'] })
    },
  })
}

export function useBlockedUsers() {
  return useQuery({
    queryKey: ['blocked-users'],
    queryFn: () => getBlockedUsers(),
    staleTime: 60_000,
  })
}

export function useBlockedIds(): Set<string> {
  const { data } = useBlockedUsers()
  return new Set((data?.data ?? []).map((u) => u.id))
}

export function useBlockerUsers() {
  return useQuery({
    queryKey: ['blocked-by-users'],
    queryFn: () => getBlockedByUsers(),
    staleTime: 60_000,
  })
}

export function useBlockerIds(): Set<string> {
  const { data } = useBlockerUsers()
  return new Set((data?.data ?? []).map((u) => u.id))
}
