import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { buttonVariants } from '@/components/ui/button-variants'
import { cn } from '@/lib/utils'
import { AuthShell } from '@/components/auth/AuthShell'
import { resetPassword } from '@/api/auth'
import { passwordErrorMessage, passwordRequirementsHint } from '@/lib/passwordError'

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError('')
    if (password !== confirm) {
      setError('Les mots de passe ne correspondent pas.')
      return
    }
    setLoading(true)
    try {
      await resetPassword(token, password)
      setSuccess(true)
    } catch (err) {
      // Message flash français selon la cause (token invalide/expiré, mot de passe
      // trop courant, trop court, contenant l'identité, identique à l'actuel…).
      setError(passwordErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  // Pas de token dans l'URL : on n'a rien à réinitialiser.
  if (!token) {
    return (
      <AuthShell
        title="Lien invalide"
        subtitle="Ce lien de réinitialisation est incomplet. Demandez-en un nouveau."
      >
        <Link
          to="/forgot-password"
          className={cn(
            buttonVariants(),
            'w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all',
          )}
        >
          Demander un nouveau lien
        </Link>
      </AuthShell>
    )
  }

  if (success) {
    return (
      <AuthShell
        title="Mot de passe modifié"
        subtitle="Votre mot de passe a été mis à jour et toutes vos sessions ont été déconnectées."
      >
        <Link
          to="/login"
          className={cn(
            buttonVariants(),
            'w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all',
          )}
        >
          Se connecter
        </Link>
      </AuthShell>
    )
  }

  return (
    <AuthShell title="Nouveau mot de passe" subtitle="Choisissez un nouveau mot de passe pour votre compte.">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            Nouveau mot de passe
          </label>
          <Input
            type="password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={12}
            maxLength={72}
            required
            autoFocus
            className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl"
          />
          <p className="text-xs text-muted-foreground leading-snug">{passwordRequirementsHint}</p>
        </div>

        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            Confirmer le mot de passe
          </label>
          <Input
            type="password"
            placeholder="••••••••"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            minLength={12}
            maxLength={72}
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
          className="w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all"
        >
          {loading ? 'Mise à jour…' : 'Réinitialiser le mot de passe'}
        </Button>

        <p className="text-xs text-muted-foreground text-center">
          <Link to="/login" className="text-primary hover:text-primary/80 transition-colors font-medium">
            Retour à la connexion
          </Link>
        </p>
      </form>
    </AuthShell>
  )
}
