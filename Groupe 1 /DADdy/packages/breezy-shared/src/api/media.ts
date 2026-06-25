import { client } from './client'
import axios from 'axios'

export const MAX_MEDIA_UPLOAD_BYTES = 300 * 1024 * 1024
export const MAX_MEDIA_UPLOAD_LABEL = '300 Mio'
const MEDIA_UPLOAD_TIMEOUT_MS = 30 * 60 * 1000

export interface MediaUploadResult {
  id: number
  media_type: string
  url: string
}

export class MediaUploadError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'MediaUploadError'
  }
}

export function mediaUploadTooLargeMessage() {
  return `Ce fichier dépasse la limite de ${MAX_MEDIA_UPLOAD_LABEL}.`
}

export function getMediaUploadErrorMessage(error: unknown) {
  if (error instanceof MediaUploadError) return error.message
  if (axios.isAxiosError(error)) {
    if (error.response?.status === 413) return mediaUploadTooLargeMessage()
    if (error.code === 'ECONNABORTED') {
      return "L'envoi a pris trop de temps. Réessayez avec un fichier plus léger ou une meilleure connexion."
    }
    const message = error.response?.data?.message
    if (typeof message === 'string' && message.trim()) return message
  }
  return "Impossible d'envoyer ce fichier. Réessayez dans un instant."
}

export async function uploadMedia(file: File): Promise<MediaUploadResult> {
  if (file.size > MAX_MEDIA_UPLOAD_BYTES) {
    throw new MediaUploadError(mediaUploadTooLargeMessage())
  }

  const form = new FormData()
  form.append('file', file)

  const { data } = await client.post<{ data: MediaUploadResult }>('/media/upload', form, {
    timeout: MEDIA_UPLOAD_TIMEOUT_MS,
    // Surcharge le Content-Type JSON par défaut du client : sinon axios sérialise
    // le FormData en JSON et le serveur rejette avec « invalid multipart body ».
    // Sans boundary, l'adaptateur navigateur le régénère correctement.
    headers: { 'Content-Type': 'multipart/form-data' },
  })

  return data.data
}
