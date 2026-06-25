import { useState } from 'react'
import { Shield, ShieldCheck, ShieldOff, Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { setupMFA, confirmMFA, disableMFA } from '@/api/auth'
import { useLanguage } from '@/hooks/useLanguage'
import type { MFASetupResponse } from '@/types/api'

type Step = 'idle' | 'setup' | 'confirm' | 'disable'

interface MFASectionProps {
  mfaEnabled: boolean
  onToggle: () => void
}

export function MFASection({ mfaEnabled, onToggle }: MFASectionProps) {
  const { t } = useLanguage()
  const [step, setStep] = useState<Step>('idle')
  const [setupData, setSetupData] = useState<MFASetupResponse | null>(null)
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showBackupCodes, setShowBackupCodes] = useState(false)

  const ts = t.settings.mfa

  const handleStartSetup = async () => {
    setError('')
    setLoading(true)
    try {
      const data = await setupMFA()
      setSetupData(data)
      setStep('setup')
      setShowBackupCodes(true)
    } catch {
      setError(ts.setupError)
    } finally {
      setLoading(false)
    }
  }

  const handleConfirm = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await confirmMFA(code.trim())
      setCode('')
      setStep('idle')
      setSetupData(null)
      onToggle()
    } catch {
      setError(ts.invalidCode)
    } finally {
      setLoading(false)
    }
  }

  const handleDisable = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await disableMFA(code.trim())
      setCode('')
      setStep('idle')
      onToggle()
    } catch {
      setError(ts.invalidCode)
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    setStep('idle')
    setCode('')
    setError('')
    setSetupData(null)
    setShowBackupCodes(false)
  }

  const copySecret = async () => {
    if (!setupData) return
    await navigator.clipboard.writeText(setupData.secret)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (step === 'setup' && setupData) {
    return (
      <section className="rounded-2xl border border-border bg-card p-5 space-y-4">
        <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
          <Shield size={15} />
          {ts.title}
        </h2>

        <div className="space-y-3">
          <p className="text-xs text-muted-foreground">{ts.setupInstructions}</p>

          <div className="rounded-xl bg-muted/40 border border-border p-4 space-y-3">
            <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">{ts.secretLabel}</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-xs font-mono text-foreground break-all select-all">
                {setupData.secret}
              </code>
              <button
                type="button"
                onClick={copySecret}
                className="shrink-0 p-1.5 rounded-lg hover:bg-accent transition-colors"
                aria-label={ts.copySecret}
              >
                {copied ? <Check size={14} className="text-green-500" /> : <Copy size={14} />}
              </button>
            </div>
            <a
              href={setupData.uri}
              className="inline-flex items-center gap-1.5 text-xs text-primary hover:text-primary/80 transition-colors"
            >
              {ts.openInApp}
            </a>
          </div>

          {showBackupCodes && (
            <div className="rounded-xl bg-amber-500/10 border border-amber-500/30 p-4 space-y-2">
              <p className="text-xs font-semibold text-amber-600 dark:text-amber-400">{ts.backupCodesWarning}</p>
              <div className="grid grid-cols-2 gap-1.5">
                {setupData.backupCodes.map((code) => (
                  <code key={code} className="text-xs font-mono text-foreground bg-background/60 rounded px-2 py-1 text-center select-all">
                    {code}
                  </code>
                ))}
              </div>
            </div>
          )}
        </div>

        <form onSubmit={handleConfirm} className="space-y-3">
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {ts.confirmLabel}
            </label>
            <Input
              type="text"
              inputMode="numeric"
              placeholder="000000"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              maxLength={6}
              className="h-10 text-center tracking-widest text-base rounded-xl"
              autoFocus
            />
          </div>

          {error && (
            <p className="text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2">
              {error}
            </p>
          )}

          <div className="flex gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={handleCancel}
              className="flex-1 h-10 rounded-xl"
            >
              {ts.cancel}
            </Button>
            <Button
              type="submit"
              disabled={loading || code.trim().length < 6}
              className="flex-1 h-10 rounded-xl"
            >
              {loading ? ts.confirming : ts.confirmBtn}
            </Button>
          </div>
        </form>
      </section>
    )
  }

  if (step === 'disable') {
    return (
      <section className="rounded-2xl border border-border bg-card p-5 space-y-4">
        <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
          <ShieldOff size={15} />
          {ts.disableTitle}
        </h2>
        <p className="text-xs text-muted-foreground">{ts.disableInstructions}</p>

        <form onSubmit={handleDisable} className="space-y-3">
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {ts.confirmLabel}
            </label>
            <Input
              type="text"
              inputMode="numeric"
              placeholder="000000"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              maxLength={10}
              className="h-10 text-center tracking-widest text-base rounded-xl"
              autoFocus
            />
          </div>

          {error && (
            <p className="text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2">
              {error}
            </p>
          )}

          <div className="flex gap-2">
            <Button type="button" variant="ghost" onClick={handleCancel} className="flex-1 h-10 rounded-xl">
              {ts.cancel}
            </Button>
            <Button
              type="submit"
              disabled={loading || code.trim().length < 6}
              variant="destructive"
              className="flex-1 h-10 rounded-xl"
            >
              {loading ? ts.disabling : ts.disableBtn}
            </Button>
          </div>
        </form>
      </section>
    )
  }

  return (
    <section className="rounded-2xl border border-border bg-card p-5">
      <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
        {mfaEnabled ? <ShieldCheck size={15} className="text-green-500" /> : <Shield size={15} />}
        {ts.title}
      </h2>
      <div className="mt-4 flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">
            {mfaEnabled ? ts.enabledLabel : ts.disabledLabel}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {mfaEnabled ? ts.enabledHint : ts.disabledHint}
          </p>
        </div>
        <Button
          variant={mfaEnabled ? 'ghost' : 'default'}
          size="sm"
          onClick={() => mfaEnabled ? setStep('disable') : handleStartSetup()}
          disabled={loading}
          className="shrink-0 rounded-xl"
        >
          {loading ? ts.loading : mfaEnabled ? ts.disableBtn : ts.enableBtn}
        </Button>
      </div>
    </section>
  )
}
