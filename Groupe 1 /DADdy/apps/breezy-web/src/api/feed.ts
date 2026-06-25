import { client } from './client'
import type { Post, PaginatedResponse } from '@/types/api'
import { transformPost } from './transform'

const mapPage = (r: { data: { data: unknown[]; page: number; limit: number; total: number } }): PaginatedResponse<Post> => ({
  data: r.data.data.map(transformPost),
  page: r.data.page,
  limit: r.data.limit,
  total: r.data.total,
})

// getFeed alimente l'onglet « Pour toi » : le feed personnalisé scoré par le
// post-service. Le backend retombe automatiquement sur le feed global (récence)
// pour un visiteur anonyme ou un nouvel arrivant sans historique de préférences.
export const getFeed = (page = 1, limit = 20) =>
  client
    .get<{ data: unknown[]; page: number; limit: number; total: number }>('/posts/for-you', { params: { page, limit } })
    .then(mapPage)
