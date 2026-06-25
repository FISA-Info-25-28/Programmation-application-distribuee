import { isAxiosError } from 'axios'
import type { ApiError } from '../types/api'

export function apiErrorMessage(error: unknown, fallback: string): string {
  if (isAxiosError<ApiError>(error)) {
    return error.response?.data?.message ?? fallback
  }
  return fallback
}
