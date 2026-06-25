import { client } from './client'
import type { Comment, Post, Poll, PaginatedResponse, TrendingHashtag } from '../types/api'
import { transformPost, transformReply, toPoll } from './transform'

export const getHashtagFeed = (name: string, page = 1, limit = 20) =>
  client
    .get<{ data: unknown[]; page: number; limit: number; total: number }>(
      `/hashtags/${encodeURIComponent(name)}/posts`,
      { params: { page, limit } },
    )
    .then((r) => ({
      data: r.data.data.map(transformPost),
      page: r.data.page,
      limit: r.data.limit,
      total: r.data.total,
    } satisfies PaginatedResponse<Post>))

export const getTrending = () =>
  client
    .get<{ data: TrendingHashtag[] }>('/hashtags/trending')
    .then((r) => r.data.data ?? [])

export type PollDraft = {
  options: string[]
  durationDays: number
}

export const createPost = (
  content: string,
  authorUsername = '',
  mentions: Array<{ id: string; username: string }> = [],
  mediaIds: number[] = [],
  poll?: PollDraft,
  quoteId?: string,
) =>
  client
    .post<{ data: unknown }>('/posts', {
      content,
      author_username: authorUsername,
      mentions,
      media_ids: mediaIds,
      poll: poll ? { options: poll.options, duration_days: poll.durationDays } : undefined,
      quote_id: quoteId ? Number(quoteId) : undefined,
    })
    .then((r) => transformPost(r.data.data))

export const votePoll = (postId: string, optionIndex: number): Promise<Poll> =>
  client
    .post<{ data: unknown }>(`/posts/${postId}/poll/vote`, { option_index: optionIndex })
    .then((r) => toPoll(r.data.data as Record<string, unknown>))

export const getPost = (id: string) =>
  client.get<{ data: unknown }>(`/posts/${id}`).then((r) => transformPost(r.data.data))

export const deletePost = (id: string) =>
  client.delete(`/posts/${id}`)

export const likePost = (id: string) =>
  client.post(`/posts/${id}/likes`)

export const unlikePost = (id: string) =>
  client.delete(`/posts/${id}/likes`)

export const rebreezePost = (id: string) =>
  client.post(`/posts/${id}/rebreezers`)

export const unrebreezePost = (id: string) =>
  client.delete(`/posts/${id}/rebreezers`)

export const bookmarkPost = (id: string) =>
  client.post(`/posts/${id}/bookmarks`)

export const unbookmarkPost = (id: string) =>
  client.delete(`/posts/${id}/bookmarks`)

export const getComments = (postId: string) =>
  client
    .get<{ data: unknown[] }>(`/posts/${postId}/replies`)
    .then((r) => ({
      data: r.data.data.map((raw) => transformReply(raw, postId)),
      page: 1,
      limit: r.data.data.length,
      total: r.data.data.length,
    } satisfies PaginatedResponse<Comment>))

export const addComment = (
  postId: string,
  content: string,
  authorUsername = '',
  mentions: Array<{ id: string; username: string }> = [],
  mediaIds: number[] = [],
) =>
  client
    .post<{ data: unknown }>(`/posts/${postId}/replies`, { content, author_username: authorUsername, mentions, media_ids: mediaIds })
    .then((r) => transformReply(r.data.data, postId))

export const searchPosts = (q: string, page = 1, limit = 20) =>
  client
    .get<{ data: unknown[]; page: number; limit: number; total: number }>('/posts/search', { params: { q, page, limit } })
    .then((r) => ({
      data: r.data.data.map(transformPost),
      page: r.data.page,
      limit: r.data.limit,
      total: r.data.total,
    } satisfies PaginatedResponse<Post>))
