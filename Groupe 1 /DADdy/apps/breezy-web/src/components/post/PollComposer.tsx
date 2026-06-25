import { X, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { useLanguage } from '@/hooks/useLanguage'
import type { PollDraft } from '@/api/posts'

const DURATIONS = [1, 3, 7] as const

interface PollComposerProps {
  poll: PollDraft
  onChange: (poll: PollDraft) => void
  onRemove: () => void
}

export function PollComposer({ poll, onChange, onRemove }: PollComposerProps) {
  const { t } = useLanguage()

  const setOption = (index: number, value: string) => {
    const next = [...poll.options]
    next[index] = value
    onChange({ ...poll, options: next })
  }

  const addOption = () => {
    if (poll.options.length >= 4) return
    onChange({ ...poll, options: [...poll.options, ''] })
  }

  const removeOption = (index: number) => {
    if (poll.options.length <= 2) return
    const next = poll.options.filter((_, i) => i !== index)
    onChange({ ...poll, options: next })
  }

  const durationLabel = (d: number) => {
    if (d === 1) return t.poll.day1
    if (d === 3) return t.poll.day3
    return t.poll.day7
  }

  return (
    <div className="mt-3 rounded-xl border border-border bg-card/50 p-3 space-y-2">
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          {t.poll.add}
        </span>
        <button
          type="button"
          onClick={onRemove}
          className="p-1 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
          aria-label={t.poll.remove}
        >
          <X size={14} />
        </button>
      </div>

      <div className="space-y-1.5">
        {poll.options.map((opt, i) => (
          <div key={i} className="flex items-center gap-1.5">
            <Input
              value={opt}
              onChange={(e) => setOption(i, e.target.value.slice(0, 25))}
              placeholder={t.poll.optionPlaceholder.replace('{{n}}', String(i + 1))}
              className="h-8 text-sm"
              maxLength={25}
            />
            {poll.options.length > 2 && (
              <button
                type="button"
                onClick={() => removeOption(i)}
                className="p-1 shrink-0 text-muted-foreground hover:text-destructive transition-colors"
                aria-label="Supprimer"
              >
                <X size={13} />
              </button>
            )}
          </div>
        ))}
      </div>

      {poll.options.length < 4 && (
        <button
          type="button"
          onClick={addOption}
          className="flex items-center gap-1.5 text-xs text-primary/70 hover:text-primary transition-colors"
        >
          <Plus size={13} />
          {t.poll.addOption}
        </button>
      )}

      <div className="flex items-center gap-2 pt-1 border-t border-border">
        <span className="text-xs text-muted-foreground shrink-0">{t.poll.duration} :</span>
        <div className="flex gap-1.5">
          {DURATIONS.map((d) => (
            <button
              key={d}
              type="button"
              onClick={() => onChange({ ...poll, durationDays: d })}
              className={`px-2.5 py-0.5 rounded-full text-xs font-medium border transition-colors ${
                poll.durationDays === d
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'border-border text-muted-foreground hover:border-primary/50 hover:text-foreground'
              }`}
            >
              {durationLabel(d)}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
