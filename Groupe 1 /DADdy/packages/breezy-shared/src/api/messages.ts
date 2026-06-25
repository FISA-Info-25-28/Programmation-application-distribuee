import { client } from './client'
import type { Conversation, DirectMessage } from '../types/api'

type ConvData = { data: Conversation }
type MsgData = { data: DirectMessage }
type ConvListData = { data: Conversation[] }
type MsgListData = { data: DirectMessage[] }

/** Trouve ou crée une conversation 1-à-1 avec participant_id. */
export const getOrCreateConversation = (participantId: string) =>
  client
    .post<ConvData>('/conversations', { participant_id: participantId })
    .then((r) => r.data.data)

/** Liste toutes les conversations de l'utilisateur courant. */
export const listConversations = () =>
  client.get<ConvListData>('/conversations').then((r) => r.data.data)

/** Récupère une conversation par ID. */
export const getConversation = (id: number) =>
  client.get<ConvData>(`/conversations/${id}`).then((r) => r.data.data)

/**
 * Liste les messages d'une conversation (newest first côté API).
 * Le caller est responsable d'inverser pour l'affichage chronologique.
 */
export const listMessages = (convId: number, page = 1, limit = 50) =>
  client
    .get<MsgListData>(`/conversations/${convId}/messages`, { params: { page, limit } })
    .then((r) => r.data.data)

/** Envoie un message dans une conversation. */
export const sendMessage = (convId: number, content: string, mediaUrl?: string, mediaType?: string) =>
  client
    .post<MsgData>(`/conversations/${convId}/messages`, {
      content,
      ...(mediaUrl ? { media_url: mediaUrl, media_type: mediaType } : {}),
    })
    .then((r) => r.data.data)

/** Marque tous les messages de la conversation comme lus. */
export const markConversationRead = (convId: number) =>
  client.put(`/conversations/${convId}/read`)
