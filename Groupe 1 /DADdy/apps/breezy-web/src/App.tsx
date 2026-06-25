import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '@/context/AuthContext'
import { ThemeProvider } from '@/context/ThemeContext'
import { LanguageProvider } from '@/context/LanguageContext'
import { useAuth } from '@/hooks/useAuth'
import { AppLayout } from '@/components/layout/AppLayout'
import { AnimatedBackground } from '@/components/AnimatedBackground'
import { LoginPage } from '@/pages/auth/LoginPage'
import { RegisterPage } from '@/pages/auth/RegisterPage'
import { VerifyEmailPage } from '@/pages/auth/VerifyEmailPage'
import { ForgotPasswordPage } from '@/pages/auth/ForgotPasswordPage'
import { ResetPasswordPage } from '@/pages/auth/ResetPasswordPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { AccountSettingsPage } from '@/pages/settings/AccountSettingsPage'
import { AppearanceSettingsPage } from '@/pages/settings/AppearanceSettingsPage'
import { PrivacySettingsPage } from '@/pages/settings/PrivacySettingsPage'
import { SecuritySettingsPage } from '@/pages/settings/SecuritySettingsPage'
import { NotificationsSettingsPage } from '@/pages/settings/NotificationsSettingsPage'
import { FeedPage } from '@/pages/FeedPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { PostPage } from '@/pages/PostPage'
import { FollowersPage } from '@/pages/FollowersPage'
import { FollowingPage } from '@/pages/FollowingPage'
import { NotificationsPage } from '@/pages/NotificationsPage'
import { FollowRequestsPage } from '@/pages/FollowRequestsPage'
import { MessagesPage } from '@/pages/MessagesPage'
import { SearchPage } from '@/pages/SearchPage'
import { HashtagPage } from '@/pages/HashtagPage'
import { AdminPage } from '@/pages/AdminPage'
import { BookmarksPage } from '@/pages/BookmarksPage'
import { TermsPage } from '@/pages/TermsPage'
import { OAuthCallbackPage } from '@/pages/auth/OAuthCallbackPage'

const qc = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 30_000 } },
})

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  if (isLoading) return null
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  if (isLoading) return null
  return isAuthenticated ? <Navigate to="/" replace /> : <>{children}</>
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<PublicRoute><LoginPage /></PublicRoute>} />
      <Route path="/register" element={<PublicRoute><RegisterPage /></PublicRoute>} />
      <Route path="/forgot-password" element={<PublicRoute><ForgotPasswordPage /></PublicRoute>} />
      {/* Accessible sans condition d'auth : on arrive ici depuis le lien du mail. */}
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />
      <Route path="/auth/oauth/callback" element={<OAuthCallbackPage />} />
      <Route path="/terms" element={<TermsPage />} />

      <Route element={<PrivateRoute><AppLayout /></PrivateRoute>}>
        <Route index element={<FeedPage />} />
        <Route path="notifications" element={<NotificationsPage />} />
        <Route path="follow-requests" element={<FollowRequestsPage />} />
        <Route path="messages" element={<MessagesPage />} />
        <Route path="profile/:id" element={<ProfilePage />} />
        <Route path="profile/:id/followers" element={<FollowersPage />} />
        <Route path="profile/:id/following" element={<FollowingPage />} />
        <Route path="post/:id" element={<PostPage />} />
        <Route path="search" element={<SearchPage />} />
        <Route path="bookmarks" element={<BookmarksPage />} />
        <Route path="hashtag/:name" element={<HashtagPage />} />
        <Route path="settings" element={<SettingsPage />}>
          <Route path="account" element={<AccountSettingsPage />} />
          <Route path="appearance" element={<AppearanceSettingsPage />} />
          <Route path="privacy" element={<PrivacySettingsPage />} />
          <Route path="security" element={<SecuritySettingsPage />} />
          <Route path="notifications" element={<NotificationsSettingsPage />} />
        </Route>
        {/* Modération : la page elle-même redirige les non-staff vers l'accueil. */}
        <Route path="admin" element={<AdminPage />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <ThemeProvider>
      <LanguageProvider>
        <QueryClientProvider client={qc}>
          <BrowserRouter>
            <AuthProvider>
              <AnimatedBackground />
              <div className="relative z-10">
                <AppRoutes />
              </div>
            </AuthProvider>
          </BrowserRouter>
        </QueryClientProvider>
      </LanguageProvider>
    </ThemeProvider>
  )
}
