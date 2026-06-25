import type { Post, Comment, PostMedia, Poll } from '../types/api'

const toMedia = (raw: Record<string, unknown>): PostMedia => ({ id: Number(raw.id), media_type: raw.media_type as 'image' | 'video' | 'audio', url: raw.url as string })

export function toPoll(raw: Record<string, unknown>): Poll {
  return {
    id: Number(raw.id),
    endsAt: raw.ends_at as string,
    totalVotes: (raw.total_votes as number) ?? 0,
    myVote: (raw.my_vote as number) ?? null,
    options: Array.isArray(raw.options)
      ? (raw.options as Array<Record<string, unknown>>).map((o) => ({
          id: Number(o.id),
          optionIndex: o.option_index as number,
          text: o.text as string,
          votesCount: (o.votes_count as number) ?? 0,
        }))
      : [],
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function transformPost(raw: any): Post {
  return {
    id: String(raw.id),
    authorId: raw.author_id,
    author: {
      id: raw.author_id,
      username: raw.author_username,
      avatarUrl: (raw.author_avatar_url as string | null | undefined) ?? null,
    },
    content: raw.content,
    tags: Array.isArray(raw.hashtags) ? raw.hashtags : [],
    mentions: Array.isArray(raw.mentions) ? raw.mentions : [],
    media: Array.isArray(raw.media) ? (raw.media as Array<Record<string, unknown>>).map(toMedia) : [],
    parentId: raw.parent_id != null ? String(raw.parent_id) : undefined,
    likesCount: raw.likes_count,
    commentsCount: raw.comments_count,
    rebreezeCount: raw.rebreeze_count ?? 0,
    likedByMe: raw.liked_by_me,
    rebreezedByMe: raw.rebreezed_by_me ?? false,
    bookmarkedByMe: raw.bookmarked_by_me ?? false,
    createdAt: raw.created_at,
    updatedAt: raw.created_at,
    poll: raw.poll ? toPoll(raw.poll) : undefined,
    quotedPostId: raw.quote_id != null ? String(raw.quote_id) : undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    quotedPost: raw.quoted_post ? transformPost(raw.quoted_post as any) : undefined,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function transformReply(raw: any, postId: string): Comment {
  return {
    id: String(raw.id),
    postId,
    parentCommentId: undefined,
    authorId: raw.author_id,
    author: { id: raw.author_id, username: raw.author_username },
    content: raw.content,
    mentions: Array.isArray(raw.mentions) ? raw.mentions : [],
    likesCount: raw.likes_count ?? 0,
    rebreezeCount: raw.rebreeze_count ?? 0,
    commentsCount: raw.comments_count ?? 0,
    likedByMe: raw.liked_by_me ?? false,
    rebreezedByMe: raw.rebreezed_by_me ?? false,
    media: Array.isArray(raw.media) ? raw.media.map(toMedia) : [],
    createdAt: raw.created_at,
  }
}
