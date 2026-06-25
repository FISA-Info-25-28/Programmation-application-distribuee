import { useState } from 'react'
import { NavLink, Link, useNavigate } from 'react-router-dom'
import { Home, Bell, Mail, User, Settings, LogOut, Feather, ShieldBan, Bookmark } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { BreezyLogo } from '@/components/BreezyLogo'
import { ThemeToggle } from '@/components/ThemeToggle'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { useAuth } from '@/hooks/useAuth'
import { useUser } from '@/hooks/useUser'
import { useLanguage } from '@/hooks/useLanguage'
import { useUnreadCount, useUnreadMessagesCount } from '@/hooks/useNotifications'
import { PostComposerModal } from '@/components/post/PostComposerModal'
import { UserAvatar } from '@/components/UserAvatar'
import { isStaff } from '@/lib/roles'
import { adminT } from '@/i18n/admin'
import { bookmarksT } from '@/i18n/bookmarks'

interface SidebarProps {
  onNavigate?: () => void
}

export function Sidebar({ onNavigate }: SidebarProps) {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { t, locale } = useLanguage()
  const staff = isStaff(user?.role)
  const [composerOpen, setComposerOpen] = useState(false)
  const { data: profile } = useUser(user?.id ?? '')
  const { data: unreadCount = 0 } = useUnreadCount()
  const unreadMessages = useUnreadMessagesCount()
  const avatarUrl = profile?.avatarUrl ?? user?.avatarUrl ?? null
  const followersCount = profile?.followersCount ?? user?.followersCount ?? 0
  const followingCount = profile?.followingCount ?? user?.followingCount ?? 0

  const handleLogout = () => {
    logout()
    navigate('/login')
    onNavigate?.()
  }

  return (
    <aside className="flex flex-col h-screen sticky top-0 w-64 px-4 py-6 bg-sidebar border-r border-border">
      <div className="mb-6 px-2">
        <BreezyLogo size={60} />
      </div>

      <nav className="flex flex-col gap-1 flex-1">
        <NavLink
          to="/"
          end
          onClick={onNavigate}
          className={({ isActive }) =>
            `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              isActive
                ? 'bg-primary/20 text-primary border border-primary/30'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent'
            }`
          }
        >
          <Home size={18} />
          {t.nav.home}
        </NavLink>

        <NavLink
          to="/notifications"
          onClick={onNavigate}
          className={({ isActive }) =>
            `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              isActive
                ? 'bg-primary/20 text-primary border border-primary/30'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent'
            }`
          }
        >
          <div className="relative">
            <Bell size={18} />
            {unreadCount > 0 && (
              <>
                <span
                  className="absolute -top-1.5 -right-1.5 h-4 min-w-4 rounded-full bg-primary/50"
                  style={{ animation: 'badge-ping 1.5s ease-out infinite' }}
                />
                <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-bold text-primary-foreground leading-none">
                  {unreadCount > 99 ? '99+' : unreadCount}
                </span>
              </>
            )}
          </div>
          {t.nav.notifications}
        </NavLink>

        <NavLink
          to="/messages"
          onClick={onNavigate}
          className={({ isActive }) =>
            `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              isActive
                ? 'bg-primary/20 text-primary border border-primary/30'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent'
            }`
          }
        >
          <div className="relative">
            <Mail size={18} />
            {unreadMessages > 0 && (
              <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-bold text-primary-foreground leading-none">
                {unreadMessages > 99 ? '99+' : unreadMessages}
              </span>
            )}
          </div>
          {t.nav.messages}
        </NavLink>

        {user && (
          <NavLink
            to={`/profile/${user.id}`}
            onClick={onNavigate}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-primary/20 text-primary border border-primary/30'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent'
              }`
            }
          >
            <User size={18} />
            {t.nav.profile}
          </NavLink>
        )}

        <NavLink
          to="/bookmarks"
          onClick={onNavigate}
          className={({ isActive }) =>
            `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              isActive
                ? 'bg-primary/20 text-primary border border-primary/30'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent'
            }`
          }
        >
          <Bookmark size={18} />
          {bookmarksT(locale).nav}
        </NavLink>

        <NavLink
          to="/settings"
          onClick={onNavigate}
          className={({ isActive }) =>
            `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              isActive
                ? 'bg-primary/20 text-primary border border-primary/30'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent'
            }`
          }
        >
          <Settings size={18} />
          {t.settings.title}
        </NavLink>

        {staff && (
          <NavLink
            to="/admin"
            onClick={onNavigate}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-primary/20 text-primary border border-primary/30'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent'
              }`
            }
          >
            <ShieldBan size={18} />
            {adminT(locale).menu}
          </NavLink>
        )}

        <div className="mt-4">
          <Button
            className="w-full gap-2 rounded-xl font-semibold shadow-md shadow-primary/20"
            onClick={() => setComposerOpen(true)}
          >
            <Feather size={16} />
            {t.nav.publish}
          </Button>
        </div>
      </nav>

      {user && (
        <div className="border-t border-border pt-4 mt-4">
          <div className="flex items-center gap-3 px-2 mb-2">
            <UserAvatar username={user.username} avatarUrl={avatarUrl} size={36} />
            <div className="min-w-0 flex-1">
              <Link
                to={`/profile/${user.id}`}
                className="text-sm font-medium text-foreground truncate hover:underline block"
              >
                @{user.username}
              </Link>
              <p className="text-xs text-muted-foreground truncate">{user.email}</p>
            </div>
            <ThemeToggle />
            <LanguageSwitcher />
          </div>
          <div className="flex gap-3 px-2 mb-3 text-xs text-muted-foreground">
            <Link to={`/profile/${user.id}/followers`} onClick={onNavigate} className="hover:text-foreground transition-colors">
              <span className="font-semibold text-foreground">{followersCount}</span> {t.nav.followers}
            </Link>
            <Link to={`/profile/${user.id}/following`} onClick={onNavigate} className="hover:text-foreground transition-colors">
              <span className="font-semibold text-foreground">{followingCount}</span> {t.nav.following}
            </Link>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start gap-3 text-muted-foreground hover:text-foreground hover:bg-accent px-3"
            onClick={handleLogout}
          >
            <LogOut size={16} />
            {t.nav.logout}
          </Button>
        </div>
      )}

      <PostComposerModal open={composerOpen} onClose={() => setComposerOpen(false)} />
    </aside>
  )
}
