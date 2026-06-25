import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useLanguage } from '@/hooks/useLanguage'
import { resendVerification } from '@/api/auth'

/**
 * Formulaire de renvoi du lien de vérification. Le back répond toujours de la
 * même façon (anti-énumération) : on affiche donc une confirmation neutre dès
 * que la requête aboutit, sans révéler si le compte existe.
 */
export function ResendVerification({ defaultEmail = '' }: { defaultEmail?: string }) {
  const { t } = useLanguage()
  const [email, setEmail] = useState(defaultEmail)
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await resendVerification(email)
      setSent(true)
    } catch {
      setError(t.auth.resend.error)
    } finally {
      setLoading(false)
    }
  }

  if (sent) {
    return (
      <p className="text-xs text-muted-foreground bg-muted/40 border border-border rounded-xl px-3 py-2.5">
        {t.auth.resend.sent}
      </p>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div className="space-y-1.5">
        <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          {t.auth.resend.email}
        </label>
        <Input
          type="email"
          placeholder={t.auth.resend.placeholder}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl"
        />
      </div>

      {error && (
        <p className="text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2.5">
          {error}
        </p>
      )}

      <Button
        type="submit"
        disabled={loading}
        variant="outline"
        className="w-full h-11 rounded-xl font-semibold"
      >
        {loading ? t.auth.resend.submitting : t.auth.resend.submit}
      </Button>
    </form>
  )
}
