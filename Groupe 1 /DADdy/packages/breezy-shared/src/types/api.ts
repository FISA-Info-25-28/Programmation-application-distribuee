// Valeurs renvoyées par le back (auth/user-service). 'admin' reste toléré par
// compat, mais le rôle administrateur est 'administrator'.
export type UserRole = 'user' | 'moderator' | 'administrator' | 'admin'
export type UserStatus = 'active' | 'suspended' | 'banned'

export type ReportStatus = 'pending' | 'resolved' | 'rejected'

export interface Report {
  id: number
  postId: number
  reporterId: string
  reason: string
  status: ReportStatus
  moderatorId?: string | null
  createdAt: string
  postContent?: string | null
  postAuthorId?: string | null
  postAuthorUsername?: string | null
}

export type SanctionType = 'ban' | 'suspension'

export interface Sanction {
  id: number
  userId: string
  moderatorId?: string | null
  type: SanctionType
  reason?: string | null
  startedAt: string
  endedAt?: string | null
  createdAt: string
}

export interface User {
  id: string
  username: string
  email: string
  displayName?: string | null
  pronouns?: string | null
  bio?: string | null
  avatarUrl?: string | null
  bannerUrl?: string | null
  role: UserRole
  status: UserStatus
  language: string
  theme: string
  isPrivate: boolean
  followersCount: number
  followingCount: number
  mfaEnabled?: boolean
  isFollowedByMe?: boolean
  // followRequested : true si j'ai une demande d'abonnement en attente (compte privé).
  followRequested?: boolean
  // canViewPosts : false sur un compte privé dont je ne suis pas abonné → masque posts/re-posts.
  canViewPosts?: boolean
  isBlockedByMe?: boolean
  hasBlockedMe?: boolean
  createdAt: string
  updatedAt: string
}



export interface UserSummary {
  id: string
  username: string
  avatarUrl?: string | null
  bio?: string | null
}

export interface PostMedia {
  id: number
  media_type: 'image' | 'video' | 'audio'
  url: string
}

export interface RebreezerInfo {
  id: string
  username: string
  displayName?: string | null
}

export interface PollOption {
  id: number
  optionIndex: number
  text: string
  votesCount: number
}

export interface Poll {
  id: number
  endsAt: string
  totalVotes: number
  myVote: number | null
  options: PollOption[]
}

export interface Post {
  id: string
  authorId: string
  author: UserSummary
  content: string
  tags: string[]
  mentions: MentionedUser[]
  media: PostMedia[]
  parentId?: string
  likesCount: number
  commentsCount: number
  rebreezeCount: number
  likedByMe: boolean
  rebreezedByMe: boolean
  bookmarkedByMe: boolean
  createdAt: string
  updatedAt: string
  rebreezerInfo?: RebreezerInfo
  poll?: Poll
  quotedPostId?: string
  quotedPost?: Post
}

export interface Comment {
  id: string
  postId: string
  parentCommentId?: string
  authorId: string
  author: UserSummary
  content: string
  mentions?: MentionedUser[]
  media: PostMedia[]
  likesCount: number
  rebreezeCount: number
  commentsCount: number
  likedByMe: boolean
  rebreezedByMe: boolean
  createdAt: string
}

export type NotificationType = 'like' | 'comment' | 'follow' | 'new_post' | 'new_message' | 'mention' | 'rebreeze' | 'follow_request' | 'follow_request_accepted'

export interface Notification {
  id: number
  type: NotificationType
  actor_id: string
  actor_username: string
  entity_id: string
  entity_type: string
  read: boolean
  created_at: string
}

export interface TrendingHashtag {
  name: string
  count: number
}

export interface MentionedUser {
  id: string
  username: string
}

export interface PaginatedResponse<T> {
  data: T[]
  page: number
  limit: number
  total: number
}

export interface AuthResponse {
  accessToken: string
  refreshToken: string
}

export interface MFARequiredResponse {
  mfaRequired: true
  pendingToken: string
}

export type LoginResponse = AuthResponse | MFARequiredResponse

export interface MFASetupResponse {
  secret: string
  uri: string
  backupCodes: string[]
}

export interface MessageResponse {
  message: string
}

export interface ApiError {
  code: string
  message: string
}

// ── Messagerie (shapes du message-service) ───────────────────────────────────

export interface DirectMessage {
  id: number
  conversation_id: number
  sender_id: string
  content: string
  media_url?: string | null
  media_type?: string | null
  created_at: string
}

export interface Conversation {
  id: number
  created_at: string
  updated_at: string
  /** Tableau d'IDs uniquement — résoudre via GET /users/:id */
  participants: string[]
  last_message?: DirectMessage | null
  unread_count: number
}

// Données portées par les toasts de notification in-app (web + mobile).
export interface ToastData {
  type: NotificationType
  actorUsername: string
  actorId: string
  entityId: string
}
