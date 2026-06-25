import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { AuthShell } from '@/components/auth/AuthShell'
import { requestPasswordReset } from '@/api/auth'

/**
 * Demande de réinitialisation : l'utilisateur saisit son email et reçoit un lien.
 * Le back répond toujours de la même façon (anti-énumération) : on affiche donc
 * une confirmation neutre dès que la requête aboutit, sans révéler si le compte
 * existe.
 */
export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await requestPasswordReset(email)
      setSent(true)
    } catch {
      setError('Envoi impossible pour le moment. Réessayez dans un instant.')
    } finally {
      setLoading(false)
    }
  }

  if (sent) {
    return (
      <AuthShell
        title="Vérifiez votre boîte mail"
        subtitle={
          <>
            Si un compte existe pour <span className="text-foreground font-medium">{email}</span>,
            un lien de réinitialisation vient d'être envoyé.
          </>
        }
      >
        <div className="space-y-4">
          <p className="text-xs text-muted-foreground">
            Le lien expire prochainement. Vous n'avez rien reçu ? Vérifiez vos spams.
          </p>
          <p className="text-xs text-muted-foreground text-center">
            <Link to="/login" className="text-primary hover:text-primary/80 transition-colors font-medium">
              Retour à la connexion
            </Link>
          </p>
        </div>
      </AuthShell>
    )
  }

  return (
    <AuthShell
      title="Mot de passe oublié"
      subtitle="Entrez votre adresse email pour recevoir un lien de réinitialisation."
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            Email
          </label>
          <Input
            type="email"
            placeholder="vous@exemple.com"
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
          className="w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all"
        >
          {loading ? 'Envoi…' : 'Envoyer le lien'}
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
