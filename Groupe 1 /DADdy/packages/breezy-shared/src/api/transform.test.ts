import { describe, expect, it } from 'vitest'
import { toPoll, transformPost, transformReply } from './transform'

const MINIMAL_POST = {
  id: 42,
  author_id: 'usr_1',
  author_username: 'alice',
  author_avatar_url: null,
  content: 'Hello world',
  likes_count: 0,
  comments_count: 0,
  liked_by_me: false,
  created_at: '2024-01-01T00:00:00Z',
}

describe('toPoll', () => {
  it('options absent → empty array', () => {
    const poll = toPoll({ id: 1, ends_at: '2024-12-31Z' })
    expect(poll.options).toEqual([])
  })

  it('option with absent votes_count defaults to 0', () => {
    const poll = toPoll({
      id: 1,
      ends_at: '2024-12-31Z',
      options: [{ id: 1, option_index: 0, text: 'A' }],
    })
    expect(poll.options[0].votesCount).toBe(0)
  })

  it('myVote null → null', () => {
    const poll = toPoll({ id: 1, ends_at: '2024-12-31Z', my_vote: null })
    expect(poll.myVote).toBeNull()
  })

  it('totalVotes absent → 0', () => {
    const poll = toPoll({ id: 1, ends_at: '2024-12-31Z' })
    expect(poll.totalVotes).toBe(0)
  })

  it('maps poll options correctly', () => {
    const poll = toPoll({
      id: 5,
      ends_at: '2024-12-31Z',
      total_votes: 100,
      my_vote: 2,
      options: [
        { id: 1, option_index: 0, text: 'Yes', votes_count: 60 },
        { id: 2, option_index: 1, text: 'No', votes_count: 40 },
      ],
    })
    expect(poll.totalVotes).toBe(100)
    expect(poll.myVote).toBe(2)
    expect(poll.options).toHaveLength(2)
    expect(poll.options[0]).toEqual({ id: 1, optionIndex: 0, text: 'Yes', votesCount: 60 })
  })
})

describe('transformPost', () => {
  it('transforms minimal post', () => {
    const post = transformPost(MINIMAL_POST)
    expect(post.id).toBe('42')
    expect(post.authorId).toBe('usr_1')
    expect(post.author.username).toBe('alice')
    expect(post.author.avatarUrl).toBeNull()
    expect(post.tags).toEqual([])
    expect(post.mentions).toEqual([])
    expect(post.media).toEqual([])
    expect(post.rebreezeCount).toBe(0)
    expect(post.rebreezedByMe).toBe(false)
    expect(post.bookmarkedByMe).toBe(false)
    expect(post.poll).toBeUndefined()
    expect(post.quotedPost).toBeUndefined()
    expect(post.quotedPostId).toBeUndefined()
  })

  it('converts id from number to string', () => {
    const post = transformPost({ ...MINIMAL_POST, id: 99 })
    expect(post.id).toBe('99')
    expect(typeof post.id).toBe('string')
  })

  it('absent hashtags/mentions/media → empty arrays', () => {
    const post = transformPost({ ...MINIMAL_POST })
    expect(post.tags).toEqual([])
    expect(post.mentions).toEqual([])
    expect(post.media).toEqual([])
  })

  it('provided hashtags and mentions arrays are forwarded', () => {
    const post = transformPost({
      ...MINIMAL_POST,
      hashtags: ['#breezy', '#test'],
      mentions: ['@alice'],
    })
    expect(post.tags).toEqual(['#breezy', '#test'])
    expect(post.mentions).toEqual(['@alice'])
  })

  it('parentId is set when parent_id is present', () => {
    const post = transformPost({ ...MINIMAL_POST, parent_id: 7 })
    expect(post.parentId).toBe('7')
  })

  it('rebreezeCount uses provided value, not default 0', () => {
    const post = transformPost({ ...MINIMAL_POST, rebreeze_count: 3, rebreezed_by_me: true, bookmarked_by_me: true })
    expect(post.rebreezeCount).toBe(3)
    expect(post.rebreezedByMe).toBe(true)
    expect(post.bookmarkedByMe).toBe(true)
  })

  it('transforms post with image and video media', () => {
    const post = transformPost({
      ...MINIMAL_POST,
      media: [
        { id: 10, media_type: 'image', url: 'https://example.com/img.jpg' },
        { id: 11, media_type: 'video', url: 'https://example.com/clip.mp4' },
      ],
    })
    expect(post.media).toHaveLength(2)
    expect(post.media[0]).toEqual({ id: 10, media_type: 'image', url: 'https://example.com/img.jpg' })
    expect(post.media[1]).toEqual({ id: 11, media_type: 'video', url: 'https://example.com/clip.mp4' })
  })

  it('transforms post with a complete poll', () => {
    const post = transformPost({
      ...MINIMAL_POST,
      poll: {
        id: 5,
        ends_at: '2024-12-31T00:00:00Z',
        total_votes: 100,
        my_vote: 1,
        options: [{ id: 1, option_index: 0, text: 'A', votes_count: 60 }],
      },
    })
    expect(post.poll).toBeDefined()
    expect(post.poll!.totalVotes).toBe(100)
    expect(post.poll!.myVote).toBe(1)
    expect(post.poll!.options[0].text).toBe('A')
  })

  it('transforms post with nested quotedPost (recursion)', () => {
    const post = transformPost({
      ...MINIMAL_POST,
      quote_id: 10,
      quoted_post: {
        id: 10,
        author_id: 'usr_2',
        author_username: 'bob',
        content: 'Quoted',
        likes_count: 5,
        comments_count: 1,
        liked_by_me: true,
        created_at: '2024-01-01T00:00:00Z',
      },
    })
    expect(post.quotedPostId).toBe('10')
    expect(post.quotedPost).toBeDefined()
    expect(post.quotedPost!.id).toBe('10')
    expect(post.quotedPost!.author.username).toBe('bob')
  })
})

describe('transformReply', () => {
  const MINIMAL_REPLY = {
    id: 7,
    author_id: 'usr_2',
    author_username: 'bob',
    content: 'Reply text',
    created_at: '2024-01-01T00:00:00Z',
  }

  it('transforms minimal reply', () => {
    const reply = transformReply(MINIMAL_REPLY, 'post_1')
    expect(reply.id).toBe('7')
    expect(reply.postId).toBe('post_1')
    expect(reply.authorId).toBe('usr_2')
    expect(reply.author.username).toBe('bob')
    expect(reply.mentions).toEqual([])
    expect(reply.media).toEqual([])
    expect(reply.likesCount).toBe(0)
    expect(reply.rebreezeCount).toBe(0)
    expect(reply.commentsCount).toBe(0)
    expect(reply.likedByMe).toBe(false)
    expect(reply.rebreezedByMe).toBe(false)
  })

  it('reply with mentions and media arrays', () => {
    const reply = transformReply(
      {
        ...MINIMAL_REPLY,
        mentions: ['@alice'],
        media: [{ id: 3, media_type: 'image', url: 'https://example.com/img.jpg' }],
      },
      'post_1',
    )
    expect(reply.mentions).toEqual(['@alice'])
    expect(reply.media).toHaveLength(1)
    expect(reply.media[0].id).toBe(3)
  })

  it('null counts fall back to 0, null arrays fall back to []', () => {
    const reply = transformReply(
      {
        ...MINIMAL_REPLY,
        likes_count: null,
        rebreeze_count: null,
        comments_count: null,
        mentions: null,
        media: null,
      },
      'post_1',
    )
    expect(reply.likesCount).toBe(0)
    expect(reply.rebreezeCount).toBe(0)
    expect(reply.commentsCount).toBe(0)
    expect(reply.mentions).toEqual([])
    expect(reply.media).toEqual([])
  })
})
