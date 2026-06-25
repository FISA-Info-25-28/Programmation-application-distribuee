import { useState, useEffect } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { PenSquare } from 'lucide-react'
import { Capacitor } from '@capacitor/core'
import { App } from '@capacitor/app'
import { Sidebar } from './Sidebar'
import { RightPanel } from './RightPanel'
import { BottomNav } from './BottomNav'
import { PostComposerModal } from '@/components/post/PostComposerModal'
import { useNotificationStream } from '@/hooks/useNotifications'
import { NotificationToast } from '@/components/NotificationToast'
import { useLanguage } from '@/hooks/useLanguage'

export function AppLayout() {
  const { toast, dismissToast } = useNotificationStream()
  const { t } = useLanguage()
  const location = useLocation()
  const navigate = useNavigate()
  const isSearch = location.pathname === '/search'
  const [composerOpen, setComposerOpen] = useState(false)

  // Gestion du bouton retour Android
  useEffect(() => {
    if (!Capacitor.isNativePlatform()) return
    const handler = App.addListener('backButton', ({ canGoBack }) => {
      if (canGoBack) {
        navigate(-1)
      } else {
        App.exitApp()
      }
    })
    return () => { handler.then((h) => h.remove()) }
  }, [navigate])

  return (
    <div className="min-h-screen bg-background flex relative">
      {/* Desktop sidebar */}
      <div className="hidden lg:block shrink-0">
        <Sidebar />
      </div>

      {/* h-screen moins l'inset haut (Dynamic Island / notch) + marge pour
          décaler le scrollport sous l'inset : les en-têtes `sticky top-0` se
          collent alors sous la status bar au lieu de passer dessous. Sur
          desktop l'inset vaut 0 → comportement inchangé. */}
      <main
        className="flex-1 border-r border-border overflow-y-auto pb-20 lg:pb-0 min-w-0 [overscroll-behavior-y:contain]"
        style={{
          height: 'calc(100dvh - env(safe-area-inset-top))',
          marginTop: 'env(safe-area-inset-top)',
        }}
      >
        <Outlet />
      </main>

      {!isSearch && (
        <div className="hidden lg:flex shrink-0 flex-col px-5 py-6 border-l border-border sticky top-0 h-screen overflow-y-auto">
          <RightPanel />
        </div>
      )}

      {/* FAB compose — mobile uniquement, au-dessus de la BottomNav */}
      <button
        onClick={() => setComposerOpen(true)}
        aria-label={t.nav.publish}
        className="lg:hidden fixed z-20 flex items-center justify-center w-14 h-14 rounded-2xl bg-primary text-primary-foreground shadow-xl shadow-primary/30 active:scale-95 transition-transform"
        style={{
          right: '1rem',
          bottom: 'calc(4.5rem + env(safe-area-inset-bottom))',
        }}
      >
        <PenSquare size={22} />
      </button>

      <BottomNav />
      <PostComposerModal open={composerOpen} onClose={() => setComposerOpen(false)} />
      <NotificationToast toast={toast} onDismiss={dismissToast} />
    </div>
  )
}
