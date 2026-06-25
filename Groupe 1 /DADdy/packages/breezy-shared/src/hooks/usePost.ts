import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getPost, getComments, addComment, deletePost, searchPosts } from '../api/posts'
import { useAuth } from './useAuth'
import type { MentionedUser } from '../types/api'

type CommentInput = { content: string; mentions?: MentionedUser[]; mediaIds?: number[] }

export function useCommentReplies(commentId: string) {
  return useQuery({
    queryKey: ['comments', commentId],
    queryFn: () => getComments(commentId),
    enabled: !!commentId,
  })
}

export function useAddReplyToComment(postId: string, commentId: string) {
  const qc = useQueryClient()
  const { user } = useAuth()
  return useMutation({
    mutationFn: ({ content, mentions = [], mediaIds = [] }: CommentInput) =>
      addComment(commentId, content, user?.username ?? '', mentions, mediaIds),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['comments', commentId] })
      qc.invalidateQueries({ queryKey: ['post', postId] })
    },
  })
}


export function usePost(id: string) {
  return useQuery({
    queryKey: ['post', id],
    queryFn: () => getPost(id),
    enabled: !!id,
  })
}

export function useComments(postId: string) {
  return useQuery({
    queryKey: ['comments', postId],
    queryFn: () => getComments(postId),
    enabled: !!postId,
  })
}

export function useAddComment(postId: string) {
  const qc = useQueryClient()
  const { user } = useAuth()
  return useMutation({
    mutationFn: ({ content, mentions = [], mediaIds = [] }: CommentInput) =>
      addComment(postId, content, user?.username ?? '', mentions, mediaIds),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['comments', postId] })
      qc.invalidateQueries({ queryKey: ['post', postId] })
    },
  })
}

export function useSearchPosts(q: string) {
  return useQuery({
    queryKey: ['search-posts', q],
    queryFn: () => searchPosts(q),
    enabled: q.trim().length > 0,
    staleTime: 0,
  })
}

export function useDeletePost(postId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => deletePost(postId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['feed'] })
      qc.removeQueries({ queryKey: ['post', postId] })
    },
  })
}
