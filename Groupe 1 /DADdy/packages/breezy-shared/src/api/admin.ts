import { client } from './client'
import type {
  PaginatedResponse,
  Report,
  ReportStatus,
  Sanction,
  SanctionType,
} from '../types/api'

// ── Signalements (modération) ────────────────────────────────────────────────

// reportPost signale un post. Ouvert à tout utilisateur authentifié.
export const reportPost = (postId: string, reason: string) =>
  client.post(`/posts/${postId}/report`, { reason })

// listReports renvoie les signalements (staff). Filtrable par statut.
export const listReports = (status?: ReportStatus, page = 1, limit = 20) =>
  client
    .get<PaginatedResponse<Report>>('/reports', {
      params: { status, page, limit },
    })
    .then((r) => r.data)

export type ReportAction = 'resolve' | 'reject' | 'delete'

// updateReport traite un signalement (staff) : conserver (resolve), rejeter
// (reject) ou supprimer le post signalé (delete).
export const updateReport = (reportId: number, action: ReportAction) =>
  client.patch(`/reports/${reportId}`, { action })

// ── Sanctions (modération) ───────────────────────────────────────────────────

// sanctionUser bannit ou suspend un utilisateur (staff).
export const sanctionUser = (userId: string, type: SanctionType, reason?: string) =>
  client.post<Sanction>(`/users/${userId}/sanctions`, { type, reason }).then((r) => r.data)

// liftSanction lève la sanction active et réactive le compte (staff).
export const liftSanction = (userId: string) =>
  client.delete(`/users/${userId}/sanctions`)

// listSanctions renvoie l'historique des sanctions d'un utilisateur (staff).
export const listSanctions = (userId: string) =>
  client.get<{ data: Sanction[] }>(`/users/${userId}/sanctions`).then((r) => r.data.data ?? [])
