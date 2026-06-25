import { client } from './client'
import type { User, Post, PaginatedResponse } from '@/types/api'
import { transformPost } from './transform'

export const getUsers = (page = 1, limit = 10) =>
  client
    .get<PaginatedResponse<User>>('/users', { params: { page, limit } })
    .then((r) => r.data)

// Suggestions d'abonnement classées côté serveur par recouvrement de thèmes
// (signal interne, jamais exposé) ; repli sur des comptes récents sans signal.
export const getSuggestions = () =>
  client
    .get<PaginatedResponse<User>>('/users/suggestions')
    .then((r) => r.data)

export const searchUsers = (q: string, page = 1, limit = 10) =>
  client
    .get<PaginatedResponse<User>>('/users', { params: { q, page, limit } })
    .then((r) => r.data)

export const getUser = (id: string) =>
  client.get<User>(`/users/${id}`).then((r) => r.data)

export const updateMe = (data: Partial<Pick<User, 'displayName' | 'pronouns' | 'bio' | 'isPrivate'>>) =>
  client.patch<User>('/users/me', data).then((r) => r.data)

// followUser : 204 (abonnement direct) sur un compte public, 202 + {status:'requested'}
// sur un compte privé. On renvoie 'requested' | 'followed' pour piloter l'UI.
export const followUser = (id: string): Promise<'requested' | 'followed'> =>
  client.post(`/users/${id}/follow`).then((r) =>
    r.status === 202 || r.data?.status === 'requested' ? 'requested' : 'followed',
  )

export const unfollowUser = (id: string) =>
  client.delete(`/users/${id}/follow`)

// ── Demandes d'abonnement (comptes privés) ──────────────────────────────────────

export const getFollowRequests = (page = 1, limit = 20) =>
  client
    .get<PaginatedResponse<User>>('/users/me/follow-requests', { params: { page, limit } })
    .then((r) => r.data)

export const acceptFollowRequest = (requesterId: string) =>
  client.post(`/users/me/follow-requests/${requesterId}/accept`)

export const rejectFollowRequest = (requesterId: string) =>
  client.delete(`/users/me/follow-requests/${requesterId}`)

export const getFollowers = (id: string, page = 1, limit = 20) =>
  client
    .get<PaginatedResponse<User>>(`/users/${id}/followers`, { params: { page, limit } })
    .then((r) => r.data)

export const getFollowing = (id: string, page = 1, limit = 20) =>
  client
    .get<PaginatedResponse<User>>(`/users/${id}/following`, { params: { page, limit } })
    .then((r) => r.data)

export const getUserPosts = (id: string, page = 1, limit = 20) =>
  client
    .get<{ data: unknown[]; page: number; limit: number; total: number }>(`/users/${id}/posts`, { params: { page, limit } })
    .then((r) => ({
      data: r.data.data.map(transformPost),
      page: r.data.page,
      limit: r.data.limit,
      total: r.data.total,
    } satisfies PaginatedResponse<Post>))

export const getUserRebreezed = (id: string, page = 1, limit = 20) =>
  client
    .get<{ data: unknown[]; page: number; limit: number; total: number }>(`/users/${id}/rebreezed`, { params: { page, limit } })
    .then((r) => ({
      data: r.data.data.map(transformPost),
      page: r.data.page,
      limit: r.data.limit,
      total: r.data.total,
    } satisfies PaginatedResponse<Post>))

export const getUserLiked = (id: string, page = 1, limit = 20) =>
  client
    .get<{ data: unknown[]; page: number; limit: number; total: number }>(`/users/${id}/liked`, { params: { page, limit } })
    .then((r) => ({
      data: r.data.data.map(transformPost),
      page: r.data.page,
      limit: r.data.limit,
      total: r.data.total,
    } satisfies PaginatedResponse<Post>))

export const getUserBookmarks = (id: string, page = 1, limit = 20) =>
  client
    .get<{ data: unknown[]; page: number; limit: number; total: number }>(`/users/${id}/bookmarks`, { params: { page, limit } })
    .then((r) => ({
      data: r.data.data.map(transformPost),
      page: r.data.page,
      limit: r.data.limit,
      total: r.data.total,
    } satisfies PaginatedResponse<Post>))

export const blockUser = (id: string) =>
  client.post(`/users/${id}/block`)

export const unblockUser = (id: string) =>
  client.delete(`/users/${id}/block`)

export const getBlockedUsers = (page = 1, limit = 100) =>
  client
    .get<PaginatedResponse<User>>('/users/me/blocked', { params: { page, limit } })
    .then((r) => r.data)

export const getBlockedByUsers = (page = 1, limit = 100) =>
  client
    .get<PaginatedResponse<User>>('/users/me/blocked-by', { params: { page, limit } })
    .then((r) => r.data)

export const uploadAvatar = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return client
    .put<{ avatarUrl: string }>('/users/me/avatar', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data.avatarUrl)
}

export const uploadBanner = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return client
    .put<{ bannerUrl: string }>('/users/me/banner', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data.bannerUrl)
}
