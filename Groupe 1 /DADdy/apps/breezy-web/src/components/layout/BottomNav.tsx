import { NavLink } from 'react-router-dom'
import { Home, Bell, Mail, User, Search } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import { useUnreadCount, useUnreadMessagesCount } from '@/hooks/useNotifications'
import { useUser } from '@/hooks/useUser'
import { UserAvatar } from '@/components/UserAvatar'

const navItemClass = (isActive: boolean) =>
  `flex-1 flex flex-col items-center justify-center gap-1 py-3 text-[10px] font-medium transition-all active:scale-90 active:opacity-60 ${
    isActive ? 'text-primary' : 'text-muted-foreground'
  }`

export function BottomNav() {
  const { user } = useAuth()
  const { t } = useLanguage()
  const { data: unreadCount = 0 } = useUnreadCount()
  const unreadMessages = useUnreadMessagesCount()
  const { data: profile } = useUser(user?.id ?? '')
  const avatarUrl = profile?.avatarUrl ?? user?.avatarUrl ?? null

  return (
    <nav
      className="lg:hidden fixed bottom-0 left-0 right-0 bg-background/90 backdrop-blur-md border-t border-border flex items-stretch z-10"
      style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}
    >
      <NavLink to="/" end className={({ isActive }) => navItemClass(isActive)}>
        <Home size={22} />
        {t.nav.home}
      </NavLink>

      <NavLink to="/search" className={({ isActive }) => navItemClass(isActive)}>
        <Search size={22} />
        Recherche
      </NavLink>

      <NavLink to="/notifications" className={({ isActive }) => navItemClass(isActive)}>
        <div className="relative">
          <Bell size={22} />
          {unreadCount > 0 && (
            <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-bold text-primary-foreground leading-none">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </div>
        {t.nav.notifs}
      </NavLink>

      <NavLink to="/messages" className={({ isActive }) => navItemClass(isActive)}>
        <div className="relative">
          <Mail size={22} />
          {unreadMessages > 0 && (
            <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-bold text-primary-foreground leading-none">
              {unreadMessages > 99 ? '99+' : unreadMessages}
            </span>
          )}
        </div>
        {t.nav.messages}
      </NavLink>

      <NavLink
        to={user ? `/profile/${user.id}` : '/login'}
        className={({ isActive }) => navItemClass(isActive)}
      >
        {avatarUrl
          ? <UserAvatar username={user?.username ?? ''} avatarUrl={avatarUrl} size={24} />
          : <User size={22} />
        }
        {t.nav.profile}
      </NavLink>
    </nav>
  )
}
