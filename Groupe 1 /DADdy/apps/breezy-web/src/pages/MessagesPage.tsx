import { useState, useRef, useEffect, useMemo } from 'react'
import { Search, Send, ArrowLeft, Check, CheckCheck, Circle, SquarePen, Users, Mic, Loader2, MicOff, ImagePlus, X } from 'lucide-react'
import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { useSearchParams, Link } from 'react-router-dom'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { UserAvatar } from '@/components/UserAvatar'
import { useAuth } from '@/hooks/useAuth'
import { useUser } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import { useHaptics } from '@/hooks/useHaptics'
import { timeAgo, timeLabel } from '@/lib/time'
import {
  listConversations,
  listMessages,
  sendMessage,
  markConversationRead,
  getOrCreateConversation,
} from '@/api/messages'
import { searchUsers, getFollowing } from '@/api/users'
import { uploadMedia, getMediaUploadErrorMessage } from '@/api/media'
import type { Conversation, DirectMessage, User } from '@/types/api'

function otherParticipant(conv: Conversation, meId: string): string {
  return conv.participants.find((id) => id !== meId) ?? conv.participants[0]
}

// ── UserRow ───────────────────────────────────────────────────────────────────

function UserRow({ user, onClick }: { user: User; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="w-full flex items-center gap-3 px-4 py-3 hover:bg-accent/50 transition-colors text-left"
    >
      <UserAvatar username={user.username} avatarUrl={user.avatarUrl} size={40} />
      <div className="min-w-0">
        <p className="text-sm font-medium text-foreground">@{user.username}</p>
        {user.bio && <p className="text-xs text-muted-foreground truncate">{user.bio}</p>}
      </div>
    </button>
  )
}

// ── UserSearchList ────────────────────────────────────────────────────────────

function UserSearchList({ meId, query, onSelect }: {
  meId: string
  query: string
  onSelect: (user: User) => void
}) {
  const searchQuery = query.startsWith('@') ? query.slice(1) : query
  const isSearching = searchQuery.trim().length > 0

  // Followings chargés une fois, réutilisés comme filtre local instantané
  const { data: followingData, isLoading: followingLoading } = useQuery({
    queryKey: ['following', meId],
    queryFn: () => getFollowing(meId),
    enabled: !!meId,
    staleTime: 60_000,
  })

  const allFollowing = useMemo(
    () => (followingData?.data ?? []).filter((u) => u.id !== meId),
    [followingData, meId],
  )

  // Filtre local instantané sur les followings (0 ms)
  const localMatches = useMemo(
    () => isSearching
      ? allFollowing.filter((u) => u.username.toLowerCase().includes(searchQuery.toLowerCase()))
      : allFollowing,
    [allFollowing, isSearching, searchQuery],
  )

  // Requête API — keepPreviousData évite le flash de skeleton entre frappes
  const { data: searchData, isFetching: remoteFetching } = useQuery({
    queryKey: ['msg-user-search', searchQuery],
    queryFn: () => searchUsers(searchQuery, 1, 10),
    enabled: isSearching,
    staleTime: 30_000,
    placeholderData: keepPreviousData,
  })

  // Résultats distants dédupliqués avec les locaux (followings d'abord)
  const remoteOnly = useMemo(() => {
    const remote = (searchData?.data ?? []).filter((u) => u.id !== meId)
    const localIds = new Set(localMatches.map((u) => u.id))
    return remote.filter((u) => !localIds.has(u.id))
  }, [searchData, localMatches, meId])

  const users = isSearching ? [...localMatches, ...remoteOnly] : allFollowing

  // Skeleton uniquement au tout premier chargement (followings jamais chargés)
  if (!isSearching && followingLoading) {
    return (
      <div className="px-4 py-2 space-y-1">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 py-2.5">
            <Skeleton className="w-10 h-10 rounded-full shrink-0" />
            <div className="flex-1 space-y-1.5">
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-2.5 w-36" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <>
      {!isSearching && (
        <div className="flex items-center gap-1.5 px-4 pt-3 pb-1">
          <Users size={11} className="text-muted-foreground" />
          <p className="text-[11px] text-muted-foreground font-medium uppercase tracking-wide">
            Personnes suivies
          </p>
        </div>
      )}
      {users.length === 0 && !remoteFetching ? (
        <p className="px-4 py-8 text-center text-xs text-muted-foreground">
          {isSearching ? 'Aucun résultat' : 'Vous ne suivez personne'}
        </p>
      ) : (
        users.map((user) => (
          <UserRow key={user.id} user={user} onClick={() => onSelect(user)} />
        ))
      )}
    </>
  )
}

// ── ConversationItem ──────────────────────────────────────────────────────────


function ConversationItem({ conv, meId, selected, onClick }: {
  conv: Conversation; meId: string; selected: boolean; onClick: () => void
}) {
  const { t, locale } = useLanguage()
  const otherId = otherParticipant(conv, meId)
  const { data: other } = useUser(otherId)
  const isLastFromMe = conv.last_message?.sender_id === meId

  return (
    <button
      onClick={onClick}
      className={`w-full flex items-center gap-3 px-4 py-3 text-left transition-colors ${
        selected ? 'bg-primary/10 border-r-2 border-primary' : 'hover:bg-accent/50'
      }`}
    >
      <div className="relative shrink-0">
        <UserAvatar username={other?.username ?? '?'} avatarUrl={other?.avatarUrl} size={40} />
        {conv.unread_count > 0 && (
          <span className="absolute -top-0.5 -right-0.5 w-4 h-4 rounded-full bg-primary text-primary-foreground text-[10px] font-bold flex items-center justify-center">
            {conv.unread_count > 9 ? '9+' : conv.unread_count}
          </span>
        )}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-baseline justify-between gap-1">
          <p className={`text-sm truncate ${conv.unread_count > 0 ? 'font-semibold' : 'font-medium'} text-foreground`}>
            @{other?.username ?? '…'}
          </p>
          {conv.last_message && (
            <span className="text-xs text-muted-foreground shrink-0">
              {timeAgo(conv.last_message.created_at, t.common, locale)}
            </span>
          )}
        </div>
        {conv.last_message && (
          <div className={`flex items-center gap-1 mt-0.5 ${conv.unread_count > 0 ? 'text-foreground' : 'text-muted-foreground'}`}>
            {isLastFromMe ? (
              <Check size={11} className="shrink-0 text-muted-foreground" />
            ) : conv.unread_count > 0 ? (
              <Circle size={7} className="shrink-0 fill-primary text-primary" />
            ) : (
              <CheckCheck size={11} className="shrink-0 text-muted-foreground" />
            )}
            <p className="text-xs truncate">
              {isLastFromMe ? t.post.you : ''}
              {conv.last_message.media_type === 'audio' && !conv.last_message.content
                ? '🎤 Voice message'
                : conv.last_message.content}
            </p>
          </div>
        )}
      </div>
    </button>
  )
}

function MessageBubble({ msg, isOwn, isRead, locale }: {
  msg: DirectMessage
  isOwn: boolean
  isRead?: boolean
  locale: string
}) {
  const hasMedia = !!msg.media_url
  const hasText = msg.content.trim().length > 0

  return (
    <div className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}>
      <div className={`max-w-[72%] rounded-2xl overflow-hidden ${
        isOwn
          ? 'bg-primary text-primary-foreground rounded-br-sm'
          : 'bg-accent text-foreground rounded-bl-sm'
      }`}>
        {hasMedia && (
          msg.media_type === 'video' ? (
            <video
              src={msg.media_url!}
              controls
              preload="metadata"
              className="w-full max-h-64 object-cover"
            />
          ) : msg.media_type === 'audio' ? (
            <div className="px-3.5 pt-2.5">
              <audio
                src={msg.media_url!}
                controls
                preload="metadata"
                className="w-full h-9"
                style={{ colorScheme: 'light dark' }}
              />
            </div>
          ) : (
            <img
              src={msg.media_url!}
              alt=""
              className="w-full max-h-64 object-cover cursor-zoom-in"
              loading="lazy"
            />
          )
        )}
        <div className={hasMedia || hasText ? 'px-3.5 py-2.5' : 'hidden'}>
          {hasText && <p className="text-sm leading-relaxed">{msg.content}</p>}
          <div className={`flex items-center gap-1 mt-1 ${isOwn ? 'justify-end' : ''}`}>
            <p className={`text-[10px] ${isOwn ? 'text-primary-foreground/60' : 'text-muted-foreground'}`}>
              {timeLabel(msg.created_at, locale as Parameters<typeof timeLabel>[1])}
            </p>
            {isOwn && (
              isRead
                ? <CheckCheck size={12} className="text-primary-foreground/60 shrink-0" />
                : <Check size={12} className="text-primary-foreground/50 shrink-0" />
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function ChatView({ conv, meId, onBack }: { conv: Conversation; meId: string; onBack: () => void }) {
  const { locale } = useLanguage()
  const { impact, ImpactStyle } = useHaptics()
  const qc = useQueryClient()
  const [draft, setDraft] = useState('')
  const [pendingMedia, setPendingMedia] = useState<{ url: string; type: string; uploading: boolean; previewUrl: string; isVideo: boolean } | null>(null)
  const [uploadError, setUploadError] = useState('')
  const [isRecording, setIsRecording] = useState(false)
  const [recordingSeconds, setRecordingSeconds] = useState(0)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const audioChunksRef = useRef<Blob[]>([])
  const recordingTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const otherId = otherParticipant(conv, meId)
  const { data: other } = useUser(otherId)

  const { data: rawMessages = [] } = useQuery({
    queryKey: ['conversations', conv.id, 'messages'],
    queryFn: () => listMessages(conv.id),
  })

  const messages = useMemo(() => [...rawMessages].reverse(), [rawMessages])

  const sendMut = useMutation({
    mutationFn: ({ content, mediaUrl, mediaType }: { content: string; mediaUrl?: string; mediaType?: string }) =>
      sendMessage(conv.id, content, mediaUrl, mediaType),
    onSuccess: (newMsg) => {
      qc.setQueryData<DirectMessage[]>(
        ['conversations', conv.id, 'messages'],
        (old = []) => [newMsg, ...old],
      )
      qc.invalidateQueries({ queryKey: ['conversations'] })
    },
  })

  useEffect(() => {
    if (conv.unread_count > 0) {
      void markConversationRead(conv.id).then(() =>
        qc.invalidateQueries({ queryKey: ['conversations'] })
      )
    }
  }, [conv.id, conv.unread_count, qc])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length])

  const handleFile = async (files: FileList | null) => {
    if (!files || files.length === 0) return
    const file = files[0]
    const previewUrl = URL.createObjectURL(file)
    const isVideo = file.type.startsWith('video/')
    const isAudio = file.type.startsWith('audio/')
    const mediaType = isVideo ? 'video' : isAudio ? 'audio' : 'image'
    setUploadError('')
    setPendingMedia({ url: '', type: mediaType, uploading: true, previewUrl, isVideo })
    try {
      const result = await uploadMedia(file)
      setPendingMedia({ url: result.url, type: result.media_type, uploading: false, previewUrl, isVideo })
    } catch (error) {
      URL.revokeObjectURL(previewUrl)
      setPendingMedia(null)
      setUploadError(getMediaUploadErrorMessage(error))
    }
  }

  const removePendingMedia = () => {
    if (pendingMedia) URL.revokeObjectURL(pendingMedia.previewUrl)
    setPendingMedia(null)
  }

  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mimeType = MediaRecorder.isTypeSupported('audio/webm') ? 'audio/webm' : 'audio/ogg'
      const recorder = new MediaRecorder(stream, { mimeType })
      mediaRecorderRef.current = recorder
      audioChunksRef.current = []

      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) audioChunksRef.current.push(e.data)
      }

      recorder.onstop = async () => {
        stream.getTracks().forEach((t) => t.stop())
        const blob = new Blob(audioChunksRef.current, { type: mimeType })
        const ext = mimeType === 'audio/webm' ? 'webm' : 'ogg'
        const file = new File([blob], `voice-message.${ext}`, { type: mimeType })
        const previewUrl = URL.createObjectURL(blob)
        setUploadError('')
        setPendingMedia({ url: '', type: 'audio', uploading: true, previewUrl, isVideo: false })
        try {
          const result = await uploadMedia(file)
          setPendingMedia({ url: result.url, type: 'audio', uploading: false, previewUrl, isVideo: false })
        } catch (error) {
          URL.revokeObjectURL(previewUrl)
          setPendingMedia(null)
          setUploadError(getMediaUploadErrorMessage(error))
        }
      }

      recorder.start()
      setIsRecording(true)
      setRecordingSeconds(0)
      recordingTimerRef.current = setInterval(() => setRecordingSeconds((s) => s + 1), 1000)
    } catch {
      // microphone access denied or unavailable
    }
  }

  const stopRecording = () => {
    if (recordingTimerRef.current) clearInterval(recordingTimerRef.current)
    recordingTimerRef.current = null
    mediaRecorderRef.current?.stop()
    setIsRecording(false)
    setRecordingSeconds(0)
  }

  const handleSend = () => {
    const content = draft.trim()
    if ((!content && !pendingMedia?.url) || sendMut.isPending || pendingMedia?.uploading) return
    impact(ImpactStyle.Light)
    sendMut.mutate({ content, mediaUrl: pendingMedia?.url, mediaType: pendingMedia?.type })
    setDraft('')
    setUploadError('')
    if (pendingMedia) URL.revokeObjectURL(pendingMedia.previewUrl)
    setPendingMedia(null)
  }

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() }
  }

  const canSend = (!!draft.trim() || !!pendingMedia?.url) && !sendMut.isPending && !pendingMedia?.uploading

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="flex items-center gap-3 px-4 py-3.5 border-b border-border shrink-0 bg-background/80 backdrop-blur-sm">
        <button onClick={onBack} className="lg:hidden p-1.5 -ml-1 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
          <ArrowLeft size={18} />
        </button>
        <Link to={`/profile/${otherId}`} className="shrink-0 hover:opacity-80 transition-opacity">
          <UserAvatar username={other?.username ?? '?'} avatarUrl={other?.avatarUrl} size={36} />
        </Link>
        <Link to={`/profile/${otherId}`} className="min-w-0 hover:underline">
          <p className="text-sm font-semibold text-foreground">@{other?.username ?? otherId}</p>
          {other?.bio && <p className="text-xs text-muted-foreground truncate max-w-[200px] no-underline">{other.bio}</p>}
        </Link>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-2 min-h-0">
        {messages.map((msg, idx) => {
          const isOwn = msg.sender_id === meId
          const isRead = isOwn && messages.slice(idx + 1).some((m) => m.sender_id !== meId)
          return <MessageBubble key={msg.id} msg={msg} isOwn={isOwn} isRead={isRead} locale={locale} />
        })}
        <div ref={bottomRef} />
      </div>

      <div className="px-4 py-3 border-t border-border shrink-0">
        {pendingMedia && (
          <div className="mb-2 relative inline-block">
            {pendingMedia.type === 'audio' ? (
              <div className="flex items-center gap-2 px-3 py-2 rounded-xl bg-accent min-w-48">
                <Mic size={14} className="text-primary shrink-0" />
                <audio src={pendingMedia.previewUrl} controls preload="metadata" className="h-8 flex-1" />
                {pendingMedia.uploading && <Loader2 size={14} className="animate-spin text-primary shrink-0" />}
              </div>
            ) : (
              <div className="w-20 h-20 rounded-xl overflow-hidden bg-accent">
                {pendingMedia.isVideo ? (
                  <video src={pendingMedia.previewUrl} className="w-full h-full object-cover" muted />
                ) : (
                  <img src={pendingMedia.previewUrl} alt="" className="w-full h-full object-cover" />
                )}
                {pendingMedia.uploading && (
                  <div className="absolute inset-0 bg-background/60 flex items-center justify-center rounded-xl">
                    <Loader2 size={18} className="animate-spin text-primary" />
                  </div>
                )}
              </div>
            )}
            <button
              onClick={removePendingMedia}
              className="absolute -top-1 -right-1 p-0.5 rounded-full bg-background border border-border text-foreground hover:bg-accent transition-colors"
            >
              <X size={11} />
            </button>
          </div>
        )}
        {uploadError && (
          <div className="mb-2 rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs font-medium text-destructive">
            {uploadError}
          </div>
        )}
        {isRecording ? (
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 flex-1 px-3 py-2 rounded-xl bg-destructive/10 border border-destructive/20">
              <span className="w-2 h-2 rounded-full bg-destructive animate-pulse shrink-0" />
              <span className="text-sm text-destructive font-medium">
                {String(Math.floor(recordingSeconds / 60)).padStart(2, '0')}:{String(recordingSeconds % 60).padStart(2, '0')}
              </span>
              <span className="text-xs text-muted-foreground">Recording…</span>
            </div>
            <Button
              size="icon"
              variant="destructive"
              onClick={stopRecording}
              className="shrink-0 rounded-xl h-9 w-9"
            >
              <MicOff size={15} />
            </Button>
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={!!pendingMedia}
              className="p-1.5 rounded-lg text-muted-foreground hover:text-primary hover:bg-accent transition-colors disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
              title="Add photo or video"
            >
              <ImagePlus size={17} />
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*,video/*,audio/*"
              className="hidden"
              onChange={(e) => handleFile(e.target.files)}
              onClick={(e) => { (e.target as HTMLInputElement).value = '' }}
            />
            <Input
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={handleKey}
              placeholder={`Message @${other?.username ?? '…'}`}
              className="flex-1 rounded-xl bg-accent border-transparent focus-visible:border-primary/40 placeholder:text-muted-foreground/60"
            />
            {!draft.trim() && !pendingMedia ? (
              <button
                type="button"
                onClick={startRecording}
                className="shrink-0 p-1.5 rounded-xl h-9 w-9 flex items-center justify-center text-muted-foreground hover:text-primary hover:bg-accent transition-colors"
                title="Voice message"
              >
                <Mic size={17} />
              </button>
            ) : (
              <Button
                size="icon"
                onClick={handleSend}
                disabled={!canSend}
                className="shrink-0 rounded-xl h-9 w-9"
              >
                <Send size={15} />
              </Button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export function MessagesPage() {
  const { user } = useAuth()
  const meId = user?.id ?? ''
  const qc = useQueryClient()
  const searchRef = useRef<HTMLInputElement>(null)
  const [searchParams, setSearchParams] = useSearchParams()

  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [composeMode, setComposeMode] = useState(false)
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 200)
    return () => clearTimeout(t)
  }, [query])

  const { data: conversations = [], isLoading } = useQuery({
    queryKey: ['conversations'],
    queryFn: listConversations,
  })

  const createConvMut = useMutation({
    mutationFn: (user: User) => getOrCreateConversation(user.id),
    onMutate: (user) => {
      qc.setQueryData(['user', user.id], user)
    },
    onSuccess: (conv) => {
      qc.invalidateQueries({ queryKey: ['conversations'] })
      setSelectedId(conv.id)
      setComposeMode(false)
      setQuery('')
    },
  })

  // Ouvre automatiquement la conversation si ?with=userId est présent dans l'URL
  useEffect(() => {
    const withId = searchParams.get('with')
    if (!withId) return
    setSearchParams({}, { replace: true })
    getOrCreateConversation(withId).then((conv) => {
      qc.invalidateQueries({ queryKey: ['conversations'] })
      setSelectedId(conv.id)
    })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const selectedConv = conversations.find((c) => c.id === selectedId) ?? null

  const filteredConversations = conversations.filter((c) => {
    if (!query.trim()) return true
    const otherId = otherParticipant(c, meId)
    const cached = qc.getQueryData<User>(['user', otherId])
    return cached?.username.toLowerCase().includes(query.toLowerCase()) ?? true
  })

  const handleComposeToggle = () => {
    setComposeMode((v) => !v)
    setQuery('')
    setTimeout(() => searchRef.current?.focus(), 50)
  }

  const handleEscape = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape' && composeMode) {
      setComposeMode(false)
      setQuery('')
    }
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Sidebar */}
      <div className={`flex flex-col border-r border-border shrink-0 w-full lg:w-[300px] ${selectedId ? 'hidden lg:flex' : 'flex'}`}>

        {/* En-tête */}
        <div className="px-4 py-4 border-b border-border shrink-0">
          <div className="flex items-center justify-between mb-3">
            <h1 className="text-base font-semibold text-foreground">
              {composeMode ? 'Nouveau message' : 'Messages'}
            </h1>
            <button
              onClick={handleComposeToggle}
              className={`p-1.5 rounded-lg transition-colors ${
                composeMode
                  ? 'text-primary bg-primary/10'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent'
              }`}
              title={composeMode ? 'Annuler' : 'Nouveau message'}
            >
              <SquarePen size={16} />
            </button>
          </div>
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none" />
            <Input
              ref={searchRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleEscape}
              placeholder={composeMode ? 'Rechercher un utilisateur…' : 'Rechercher'}
              className="pl-8 h-8 text-xs rounded-xl bg-accent border-transparent focus-visible:border-primary/40 placeholder:text-muted-foreground/60"
            />
          </div>
        </div>

        {/* Liste */}
        <div className="flex-1 overflow-y-auto min-h-0">
          {composeMode ? (
            <UserSearchList
              meId={meId}
              query={debouncedQuery}
              onSelect={(user) => createConvMut.mutate(user)}
            />
          ) : isLoading ? (
            Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 px-4 py-3">
                <Skeleton className="w-10 h-10 rounded-full shrink-0" />
                <div className="flex-1 space-y-1.5">
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-3 w-40" />
                </div>
              </div>
            ))
          ) : filteredConversations.length === 0 ? (
            <p className="px-4 py-8 text-center text-xs text-muted-foreground">Aucune conversation</p>
          ) : (
            filteredConversations.map((conv) => (
              <ConversationItem
                key={conv.id}
                conv={conv}
                meId={meId}
                selected={conv.id === selectedId}
                onClick={() => { setSelectedId(conv.id); setComposeMode(false); setQuery('') }}
              />
            ))
          )}
        </div>
      </div>

      <div className={`flex-1 min-w-0 ${selectedId ? 'flex flex-col' : 'hidden lg:flex'}`}>
        {selectedConv ? (
          <ChatView conv={selectedConv} meId={meId} onBack={() => setSelectedId(null)} />
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center gap-3 text-muted-foreground">
            <div className="w-14 h-14 rounded-2xl bg-accent flex items-center justify-center">
              <Send size={24} strokeWidth={1.5} />
            </div>
            <div className="text-center">
              <p className="text-sm font-medium text-foreground">Vos messages</p>
              <p className="text-xs mt-0.5">Sélectionnez une conversation ou commencez-en une nouvelle</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
