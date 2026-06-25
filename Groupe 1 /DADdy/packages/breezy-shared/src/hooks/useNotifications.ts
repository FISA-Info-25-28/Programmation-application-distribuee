import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useCallback, useState } from 'react'
import {
  getNotifications,
  getUnreadCount,
  markNotificationRead,
  markAllNotificationsRead,
  deleteNotification,
  subscribe,
  unsubscribe,
  listSubscriptions,
} from '../api/notifications'
import { listConversations } from '../api/messages'
import { getToken, getApiBaseURL } from '../api/client'
import type { ToastData, Notification, NotificationType } from '../types/api'

// Son de notification — no-op si l'API Audio n'est pas disponible (SSR, tests).
const popSound = typeof Audio !== 'undefined' ? new Audio('/sound/breezy_pop.mp3') : null

type NotificationsData = { data: Notification[]; total: number }

export function useNotifications() {
  return useQuery({
    queryKey: ['notifications'],
    queryFn: () => getNotifications(),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ['notifications-unread-count'],
    queryFn: getUnreadCount,
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

export function useUnreadMessagesCount() {
  const { data: conversations = [] } = useQuery({
    queryKey: ['conversations'],
    queryFn: listConversations,
    staleTime: 30_000,
    refetchInterval: 60_000,
  })
  return conversations.reduce((sum, c) => sum + c.unread_count, 0)
}

const KNOWN_TYPES = new Set<NotificationType>(['like', 'comment', 'follow', 'new_post', 'new_message', 'mention', 'rebreeze', 'follow_request', 'follow_request_accepted'])

export function useNotificationStream() {
  const qc = useQueryClient()
  const [toast, setToast] = useState<ToastData | null>(null)
  const dismissToast = useCallback(() => setToast(null), [])

  useEffect(() => {
    let active = true
    let abort: AbortController | null = null
    let retryTimer: ReturnType<typeof setTimeout> | null = null

    const connect = () => {
      if (!active) return
      const token = getToken()
      if (!token) {
        retryTimer = setTimeout(connect, 5_000)
        return
      }

      abort = new AbortController()

      // URL dynamique : /api/notifications/stream sur web, URL absolue sur native.
      const streamUrl = `${getApiBaseURL()}/notifications/stream`

      fetch(streamUrl, {
        headers: { Authorization: `Bearer ${token}` },
        signal: abort.signal,
      })
        .then(async (res) => {
          if (!res.ok || !res.body) throw new Error(`${res.status}`)

          const reader = res.body.getReader()
          const decoder = new TextDecoder()
          let buf = ''

          while (active) {
            const { done, value } = await reader.read()
            if (done) break
            buf += decoder.decode(value, { stream: true })

            const parts = buf.split('\n\n')
            buf = parts.pop() ?? ''
            for (const part of parts) {
              if (!part.includes('event: new_notification')) continue
              if (popSound) {
                popSound.currentTime = 0
                popSound.play().catch(() => {})
              }
              qc.invalidateQueries({ queryKey: ['notifications'] })
              qc.invalidateQueries({ queryKey: ['notifications-unread-count'] })

              const dataLine = part.split('\n').find((l) => l.startsWith('data:'))
              if (dataLine) {
                try {
                  const payload = JSON.parse(dataLine.slice(5).trim())
                  if (payload.actor_username && KNOWN_TYPES.has(payload.type)) {
                    setToast({
                      type: payload.type as NotificationType,
                      actorUsername: payload.actor_username,
                      actorId: payload.actor_id ?? '',
                      entityId: payload.entity_id ?? '',
                    })
                  }
                  if (payload.type === 'new_post') {
                    qc.setQueryData<number>(['feed-new-posts-count'], (old = 0) => old + 1)
                  }
                  if (payload.type === 'new_message') {
                    qc.invalidateQueries({ queryKey: ['conversations'] })
                  }
                  if (payload.type === 'comment' && payload.entity_id) {
                    qc.invalidateQueries({ queryKey: ['comments', payload.entity_id] })
                    qc.invalidateQueries({ queryKey: ['post', payload.entity_id] })
                  }
                } catch { /* non-JSON payload, ignore */ }
              }
            }
          }
        })
        .catch(() => {})
        .finally(() => {
          if (active) retryTimer = setTimeout(connect, 5_000)
        })
    }

    connect()

    return () => {
      active = false
      abort?.abort()
      if (retryTimer) clearTimeout(retryTimer)
    }
  }, [qc])

  return { toast, dismissToast }
}

export function useMarkNotificationRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => markNotificationRead(id),
    onSuccess: (_, id) => {
      qc.setQueryData<NotificationsData>(['notifications'], (old) =>
        old ? { ...old, data: old.data.map((n) => (n.id === id ? { ...n, read: true } : n)) } : old,
      )
      qc.invalidateQueries({ queryKey: ['notifications-unread-count'] })
    },
  })
}

export function useMarkAllRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: () => {
      qc.setQueryData<NotificationsData>(['notifications'], (old) =>
        old ? { ...old, data: old.data.map((n) => ({ ...n, read: true })) } : old,
      )
      qc.setQueryData(['notifications-unread-count'], 0)
    },
  })
}

export function useDeleteNotification() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteNotification(id),
    onSuccess: (_, id) => {
      qc.setQueryData<NotificationsData>(['notifications'], (old) =>
        old ? { ...old, data: old.data.filter((n) => n.id !== id) } : old,
      )
      qc.invalidateQueries({ queryKey: ['notifications-unread-count'] })
    },
  })
}

export function useSubscription(targetUserId: string) {
  const qc = useQueryClient()

  const { data: subscriptions = [], isLoading } = useQuery({
    queryKey: ['subscriptions'],
    queryFn: listSubscriptions,
    staleTime: 60_000,
    enabled: !!targetUserId,
  })

  const isSubscribed = subscriptions.includes(targetUserId)

  const toggle = useMutation({
    mutationFn: () => (isSubscribed ? unsubscribe(targetUserId) : subscribe(targetUserId)),
    onMutate: async () => {
      await qc.cancelQueries({ queryKey: ['subscriptions'] })
      const previous = qc.getQueryData<string[]>(['subscriptions']) ?? []
      const next = isSubscribed
        ? previous.filter((id) => id !== targetUserId)
        : [...previous, targetUserId]
      qc.setQueryData(['subscriptions'], next)
      return { previous }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previous) qc.setQueryData(['subscriptions'], ctx.previous)
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ['subscriptions'] })
    },
  })

  return { isSubscribed, isLoading, toggle: () => toggle.mutate(), isPending: toggle.isPending }
}
