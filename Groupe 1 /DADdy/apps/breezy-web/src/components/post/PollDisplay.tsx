import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { votePoll } from '@/api/posts'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import type { Poll } from '@/types/api'

interface PollDisplayProps {
  poll: Poll
  postId: string
  authorId: string
  /** Translated option labels, index-aligned with poll.options. Null when not translating. */
  translatedOptions?: string[] | null
}

function timeLeft(endsAt: string, t: { poll: Record<string, string> }): string {
  const diff = new Date(endsAt).getTime() - Date.now()
  if (diff <= 0) return t.poll.finalResults
  const hours = Math.floor(diff / 3_600_000)
  const days = Math.floor(hours / 24)
  if (days >= 1) return `${days} ${days === 1 ? t.poll.dayLeft : t.poll.daysLeft}`
  return `${hours} ${t.poll.hoursLeft}`
}

export function PollDisplay({ poll: initialPoll, postId, authorId, translatedOptions }: PollDisplayProps) {
  const { t } = useLanguage()
  const { user } = useAuth()
  const qc = useQueryClient()
  const [poll, setPoll] = useState(initialPoll)
  const [voting, setVoting] = useState(false)
  const [error, setError] = useState('')

  const [now] = useState(Date.now)
  const isExpired = new Date(poll.endsAt).getTime() <= now
  const isOwn = user?.id === authorId
  const hasVoted = poll.myVote !== null
  const showResults = hasVoted || isExpired || isOwn

  const handleVote = async (optionIndex: number) => {
    if (showResults || voting) return
    if (isOwn) { setError(t.poll.cantVoteOwn); return }
    setVoting(true)
    setError('')
    try {
      const updated = await votePoll(postId, optionIndex)
      setPoll(updated)
      qc.invalidateQueries({ queryKey: ['feed'] })
      qc.invalidateQueries({ queryKey: ['post', postId] })
    } catch {
      setError(t.poll.voteError)
    } finally {
      setVoting(false)
    }
  }

  const maxVotes = Math.max(...poll.options.map((o) => o.votesCount), 1)

  return (
    <div className="mt-2 space-y-1.5" onClick={(e) => e.stopPropagation()}>
      {poll.options.map((option, i) => {
        const label = translatedOptions?.[i] || option.text
        const pct = poll.totalVotes > 0 ? Math.round((option.votesCount / poll.totalVotes) * 100) : 0
        const isMyChoice = poll.myVote === option.optionIndex
        const isWinner = showResults && option.votesCount === maxVotes && poll.totalVotes > 0

        if (showResults) {
          return (
            <div key={option.id} className="relative overflow-hidden rounded-lg border border-border">
              <div
                className={`absolute inset-y-0 left-0 transition-all duration-500 ${
                  isMyChoice ? 'bg-primary/20' : 'bg-accent/60'
                }`}
                style={{ width: `${pct}%` }}
              />
              <div className="relative flex items-center justify-between px-3 py-2">
                <span className={`text-sm ${isWinner ? 'font-semibold text-foreground' : 'text-foreground/80'}`}>
                  {isMyChoice && (
                    <span className="inline-block w-2 h-2 rounded-full bg-primary mr-1.5 align-middle" />
                  )}
                  {label}
                </span>
                <span className={`text-xs shrink-0 ml-2 ${isWinner ? 'font-bold text-foreground' : 'text-muted-foreground'}`}>
                  {pct}%
                </span>
              </div>
            </div>
          )
        }

        return (
          <button
            key={option.id}
            type="button"
            disabled={voting}
            onClick={() => handleVote(option.optionIndex)}
            className="w-full text-left px-3 py-2 rounded-lg border border-primary/40 text-sm text-foreground hover:bg-primary/10 hover:border-primary transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {voting && <Loader2 size={12} className="inline animate-spin mr-1.5" />}
            {label}
          </button>
        )
      })}

      <div className="flex items-center gap-2 text-xs text-muted-foreground pt-0.5">
        <span>
          {poll.totalVotes} {poll.totalVotes === 1 ? t.poll.vote : t.poll.votes}
        </span>
        <span>·</span>
        <span>{timeLeft(poll.endsAt, t)}</span>
      </div>

      {error && (
        <p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-1.5">{error}</p>
      )}
    </div>
  )
}
