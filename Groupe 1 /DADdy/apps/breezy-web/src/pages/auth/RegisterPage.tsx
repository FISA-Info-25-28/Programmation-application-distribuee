import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { AuthShell } from '@/components/auth/AuthShell'
import { ResendVerification } from '@/components/auth/ResendVerification'
import { useLanguage } from '@/hooks/useLanguage'
import { register as registerApi } from '@/api/auth'
import { passwordErrorMessage, passwordRequirementsHint } from '@/lib/passwordError'

export function RegisterPage() {
  const { t } = useLanguage()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [termsAccepted, setTermsAccepted] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [registered, setRegistered] = useState(false)

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await registerApi(username, email, password, termsAccepted)
      setRegistered(true)
    } catch (err) {
      // Message flash spécifique (mot de passe trop courant, trop court, contenant
      // l'identité, email déjà utilisé…) plutôt qu'un message générique.
      setError(passwordErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  if (registered) {
    return (
      <AuthShell
        title={t.auth.register.checkEmailTitle}
        subtitle={
          <>
            {t.auth.register.checkEmailSubtitleBefore}{' '}
            <span className="text-foreground font-medium">{email}</span>.{' '}
            {t.auth.register.checkEmailSubtitleAfter}
          </>
        }
      >
        <div className="space-y-4">
          <p className="text-xs text-muted-foreground">
            {t.auth.register.noEmail}
          </p>
          <ResendVerification defaultEmail={email} />
          <p className="text-xs text-muted-foreground text-center">
            <Link to="/login" className="text-primary hover:text-primary/80 transition-colors font-medium">
              {t.auth.register.backToLogin}
            </Link>
          </p>
        </div>
      </AuthShell>
    )
  }

  return (
    <AuthShell
      title={t.auth.register.title}
      subtitle={
        <>
          {t.auth.register.hasAccount}{' '}
          <Link to="/login" className="text-primary hover:text-primary/80 transition-colors font-medium">
            {t.auth.register.signIn}
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            {t.auth.register.username}
          </label>
          <Input
            type="text"
            placeholder="alice"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            minLength={3}
            maxLength={30}
            required
            className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl"
          />
        </div>
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            {t.auth.register.email}
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
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            {t.auth.register.password}
          </label>
          <Input
            type="password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={12}
            maxLength={72}
            required
            className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl"
          />
          <p className="text-xs text-muted-foreground leading-snug">{passwordRequirementsHint}</p>
        </div>

        <label className="flex items-start gap-3 text-xs text-muted-foreground leading-relaxed cursor-pointer">
          <input
            type="checkbox"
            checked={termsAccepted}
            onChange={(e) => setTermsAccepted(e.target.checked)}
            required
            className="mt-0.5 size-4 shrink-0 accent-primary"
          />
          <span>
            {t.auth.register.termsBefore}{' '}
            <Link
              to="/terms"
              target="_blank"
              rel="noreferrer"
              className="text-primary hover:text-primary/80 font-medium"
            >
              {t.auth.register.termsLink}
            </Link>
            {t.auth.register.termsAfter}
          </span>
        </label>

        {error && (
          <p className="text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2.5">
            {error}
          </p>
        )}

        <Button
          type="submit"
          disabled={loading}
          className="w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all"
        >
          {loading ? t.auth.register.submitting : t.auth.register.submit}
        </Button>
      </form>
    </AuthShell>
  )
}
