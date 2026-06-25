import { useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Flag, ShieldBan, Ban, Pause, RotateCcw, Trash2, Check, X, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { UserAvatar } from '@/components/UserAvatar'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import { isStaff } from '@/lib/roles'
import { adminT } from '@/i18n/admin'
import {
  listReports,
  updateReport,
  type ReportAction,
  sanctionUser,
  liftSanction,
} from '@/api/admin'
import { getUsers, searchUsers } from '@/api/users'
import type { ReportStatus, SanctionType, User } from '@/types/api'

type Tab = 'reports' | 'users'

const REPORT_FILTERS: Array<ReportStatus | 'all'> = ['pending', 'resolved', 'rejected', 'all']

function StatusBadge({ label, tone }: { label: string; tone: 'green' | 'red' | 'amber' | 'muted' }) {
  const tones: Record<string, string> = {
    green: 'bg-emerald-500/15 text-emerald-500',
    red: 'bg-rose-500/15 text-rose-500',
    amber: 'bg-amber-500/15 text-amber-500',
    muted: 'bg-accent text-muted-foreground',
  }
  return <span className={`px-2 py-0.5 rounded-full text-[11px] font-medium ${tones[tone]}`}>{label}</span>
}

// ── Reports tab ───────────────────────────────────────────────────────────────

function ReportsTab() {
  const { locale } = useLanguage()
  const a = adminT(locale)
  const qc = useQueryClient()
  const [filter, setFilter] = useState<ReportStatus | 'all'>('pending')

  const { data, isLoading } = useQuery({
    queryKey: ['admin-reports', filter],
    queryFn: () => listReports(filter === 'all' ? undefined : filter),
  })

  const mutation = useMutation({
    mutationFn: ({ id, action }: { id: number; action: ReportAction }) => updateReport(id, action),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-reports'] })
      qc.invalidateQueries({ queryKey: ['feed'] })
    },
  })

  const filterLabel: Record<ReportStatus | 'all', string> = {
    pending: a.filterPending,
    resolved: a.filterResolved,
    rejected: a.filterRejected,
    all: a.filterAll,
  }

  const reports = data?.data ?? []

  return (
    <div className="px-4 py-4">
      <div className="flex gap-2 mb-4 flex-wrap">
        {REPORT_FILTERS.map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${
              filter === f
                ? 'bg-primary/20 text-primary border border-primary/30'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent'
            }`}
          >
            {filterLabel[f]}
          </button>
        ))}
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground py-8 text-center">…</p>
      ) : reports.length === 0 ? (
        <p className="text-sm text-muted-foreground py-8 text-center">{a.noReports}</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {reports.map((r) => (
            <li key={r.id} className="rounded-2xl border border-border bg-card p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs text-muted-foreground">
                  {a.reportedPostBy} @{r.postAuthorUsername ?? '?'}
                </span>
                <StatusBadge
                  label={filterLabel[r.status]}
                  tone={r.status === 'pending' ? 'amber' : r.status === 'resolved' ? 'green' : 'muted'}
                />
              </div>

              {r.postContent != null ? (
                <Link to={`/post/${r.postId}`} className="block text-sm text-foreground/90 hover:underline mb-2 whitespace-pre-wrap">
                  {r.postContent || a.viewPost}
                </Link>
              ) : (
                <p className="text-sm italic text-muted-foreground mb-2">{a.deletedPost}</p>
              )}

              <p className="text-xs text-muted-foreground mb-3">
                <span className="font-medium text-foreground/70">{a.reason}:</span> {r.reason}
              </p>

              {r.status === 'pending' && (
                <div className="flex gap-2 flex-wrap">
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={mutation.isPending}
                    onClick={() => mutation.mutate({ id: r.id, action: 'resolve' })}
                    className="gap-1.5 text-emerald-500 hover:bg-emerald-500/10"
                  >
                    <Check size={14} /> {a.actionResolve}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={mutation.isPending}
                    onClick={() => mutation.mutate({ id: r.id, action: 'reject' })}
                    className="gap-1.5 text-muted-foreground hover:bg-accent"
                  >
                    <X size={14} /> {a.actionReject}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={mutation.isPending}
                    onClick={() => mutation.mutate({ id: r.id, action: 'delete' })}
                    className="gap-1.5 text-rose-500 hover:bg-rose-500/10 ml-auto"
                  >
                    <Trash2 size={14} /> {a.actionDelete}
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

// ── Sanction modal ────────────────────────────────────────────────────────────

function SanctionModal({
  user,
  type,
  onClose,
  onDone,
}: {
  user: User
  type: SanctionType
  onClose: () => void
  onDone: () => void
}) {
  const { locale } = useLanguage()
  const a = adminT(locale)
  const [reason, setReason] = useState('')
  const mutation = useMutation({
    mutationFn: () => sanctionUser(user.id, type, reason.trim() || undefined),
    onSuccess: () => {
      onDone()
      onClose()
    },
  })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-background/30 backdrop-blur-[2px]" onClick={onClose} />
      <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl border border-border bg-card shadow-xl p-5">
        <p className="text-sm font-semibold text-foreground mb-1">
          {type === 'ban' ? a.sanctionTitleBan : a.sanctionTitleSuspend}
        </p>
        <p className="text-xs text-muted-foreground mb-3">@{user.username}</p>
        <textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder={a.sanctionReasonPlaceholder}
          rows={3}
          className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-1 focus:ring-primary"
        />
        <div className="flex justify-end gap-2 mt-4">
          <Button variant="ghost" size="sm" onClick={onClose} disabled={mutation.isPending}>
            {a.cancel}
          </Button>
          <Button
            size="sm"
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending}
            className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
          >
            {mutation.isPending ? a.working : a.confirm}
          </Button>
        </div>
      </div>
    </div>
  )
}

// ── Users tab ─────────────────────────────────────────────────────────────────

function UsersTab() {
  const { locale } = useLanguage()
  const a = adminT(locale)
  const qc = useQueryClient()
  const { user: me } = useAuth()
  const [query, setQuery] = useState('')
  const [sanctionTarget, setSanctionTarget] = useState<{ user: User; type: SanctionType } | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin-users', query],
    queryFn: () => (query.trim() ? searchUsers(query.trim(), 1, 30) : getUsers(1, 30)),
  })

  const lift = useMutation({
    mutationFn: (id: string) => liftSanction(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-users'] }),
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['admin-users'] })

  const users = (data?.data ?? []).filter((u) => u.id !== me?.id)

  const statusBadge = (status: string) => {
    if (status === 'banned') return <StatusBadge label={a.statusBanned} tone="red" />
    if (status === 'suspended') return <StatusBadge label={a.statusSuspended} tone="amber" />
    return <StatusBadge label={a.statusActive} tone="green" />
  }

  return (
    <div className="px-4 py-4">
      <div className="relative mb-4">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={a.searchUsers}
          className="w-full rounded-xl border border-border bg-background pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground py-8 text-center">…</p>
      ) : users.length === 0 ? (
        <p className="text-sm text-muted-foreground py-8 text-center">{a.noUsers}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {users.map((u) => {
            const sanctioned = u.status === 'banned' || u.status === 'suspended'
            return (
              <li key={u.id} className="flex items-center gap-3 rounded-2xl border border-border bg-card p-3">
                <Link to={`/profile/${u.id}`}>
                  <UserAvatar username={u.username} avatarUrl={u.avatarUrl} size={36} />
                </Link>
                <div className="min-w-0 flex-1">
                  <Link to={`/profile/${u.id}`} className="text-sm font-medium text-foreground truncate hover:underline block">
                    {u.displayName?.trim() || u.username}
                  </Link>
                  <span className="text-xs text-muted-foreground truncate">@{u.username}</span>
                </div>
                {statusBadge(u.status)}
                {sanctioned ? (
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={lift.isPending}
                    onClick={() => lift.mutate(u.id)}
                    className="gap-1.5 text-emerald-500 hover:bg-emerald-500/10"
                  >
                    <RotateCcw size={14} /> {a.lift}
                  </Button>
                ) : (
                  <div className="flex gap-1">
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setSanctionTarget({ user: u, type: 'suspension' })}
                      className="gap-1.5 text-amber-500 hover:bg-amber-500/10"
                    >
                      <Pause size={14} /> {a.suspend}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setSanctionTarget({ user: u, type: 'ban' })}
                      className="gap-1.5 text-rose-500 hover:bg-rose-500/10"
                    >
                      <Ban size={14} /> {a.ban}
                    </Button>
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}

      {sanctionTarget && (
        <SanctionModal
          user={sanctionTarget.user}
          type={sanctionTarget.type}
          onClose={() => setSanctionTarget(null)}
          onDone={invalidate}
        />
      )}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export function AdminPage() {
  const { user } = useAuth()
  const { locale } = useLanguage()
  const a = adminT(locale)
  const [tab, setTab] = useState<Tab>('reports')

  // Garde de rôle : un non-staff est renvoyé vers l'accueil.
  if (!isStaff(user?.role)) return <Navigate to="/" replace />

  return (
    <div>
      <div className="px-4 py-4 border-b border-border sticky top-0 z-10 bg-background/80 backdrop-blur-sm">
        <h1 className="text-base font-semibold text-foreground flex items-center gap-2">
          <ShieldBan size={18} /> {a.title}
        </h1>
      </div>

      <div className="flex border-b border-border">
        <button
          onClick={() => setTab('reports')}
          className={`flex-1 flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors ${
            tab === 'reports' ? 'text-primary border-b-2 border-primary' : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <Flag size={15} /> {a.tabReports}
        </button>
        <button
          onClick={() => setTab('users')}
          className={`flex-1 flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors ${
            tab === 'users' ? 'text-primary border-b-2 border-primary' : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <ShieldBan size={15} /> {a.tabUsers}
        </button>
      </div>

      {tab === 'reports' ? <ReportsTab /> : <UsersTab />}
    </div>
  )
}
