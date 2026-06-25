import { client } from './client'
import type { Notification } from '../types/api'

export const getNotifications = (limit = 50) =>
  client
    .get<{ data: Notification[]; total: number }>('/notifications', { params: { limit } })
    .then((r) => r.data)

export const getUnreadCount = () =>
  client
    .get<{ data: { count: number } }>('/notifications/unread-count')
    .then((r) => r.data.data.count)

export const markNotificationRead = (id: number) =>
  client.put(`/notifications/${id}/read`)

export const markAllNotificationsRead = () =>
  client.put('/notifications/read-all')

export const deleteNotification = (id: number) =>
  client.delete(`/notifications/${id}`)

export const subscribe = (targetUserId: string) =>
  client.post('/subscriptions', { target_user_id: targetUserId })

export const unsubscribe = (targetUserId: string) =>
  client.delete(`/subscriptions/${targetUserId}`)

export const listSubscriptions = () =>
  client.get<{ data: string[] }>('/subscriptions').then((r) => r.data.data)
