import { client } from './client'
import type { LoginResponse, AuthResponse, MessageResponse, MFASetupResponse, User } from '../types/api'
import { createRegisterRequest } from '../lib/registerRequest'

export const login = (email: string, password: string) =>
  client.post<LoginResponse>('/auth/login', { email, password }).then((r) => r.data)

// register ne connecte plus : il crée un compte non vérifié et déclenche l'envoi
// du mail de confirmation. La connexion reste refusée tant que l'email n'est pas
// vérifié (cf. verifyEmail).
export const register = (username: string, email: string, password: string, termsAccepted: boolean) =>
  client.post<MessageResponse>(
    '/auth/register',
    createRegisterRequest(username, email, password, termsAccepted),
  ).then((r) => r.data)

export const verifyEmail = (token: string) =>
  client.post<MessageResponse>('/auth/verify-email', { token }).then((r) => r.data)

export const resendVerification = (email: string) =>
  client.post<MessageResponse>('/auth/resend-verification', { email }).then((r) => r.data)

// requestPasswordReset déclenche l'envoi d'un lien de réinitialisation. Le back
// répond toujours de la même façon (anti-énumération), qu'un compte existe ou non.
export const requestPasswordReset = (email: string) =>
  client.post<MessageResponse>('/auth/request-password-reset', { email }).then((r) => r.data)

// resetPassword consomme le token reçu par mail et définit le nouveau mot de passe.
export const resetPassword = (token: string, password: string) =>
  client.post<MessageResponse>('/auth/reset-password', { token, password }).then((r) => r.data)

// changePassword change le mot de passe d'un utilisateur connecté (paramètres).
export const changePassword = (currentPassword: string, newPassword: string) =>
  client.post<MessageResponse>('/auth/change-password', { currentPassword, newPassword }).then((r) => r.data)

export const getMe = () =>
  client.get<User>('/auth/me').then((r) => r.data)

export const refreshToken = (token: string) =>
  client.post<AuthResponse>('/auth/refresh', { refreshToken: token }).then((r) => r.data)

export const logout = (token: string) =>
  client.post('/auth/logout', { refreshToken: token }).catch(() => {})

export const setupMFA = () =>
  client.post<MFASetupResponse>('/auth/mfa/setup').then((r) => r.data)

export const confirmMFA = (code: string) =>
  client.post<MessageResponse>('/auth/mfa/confirm', { code }).then((r) => r.data)

export const disableMFA = (code: string) =>
  client.delete<MessageResponse>('/auth/mfa', { data: { code } }).then((r) => r.data)

export const verifyMFA = (pendingToken: string, code: string) =>
  client.post<AuthResponse>('/auth/mfa/verify', { pendingToken, code }).then((r) => r.data)

// OAuth2 social login: exchange the short-lived code from the callback URL for tokens
export const exchangeOAuthCode = (code: string) =>
  client.post<AuthResponse>('/auth/oauth/exchange', { code }).then((r) => r.data)
