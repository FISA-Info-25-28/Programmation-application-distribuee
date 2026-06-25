import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import { changePassword } from '@/api/auth'
import { passwordErrorMessage, passwordRequirementsHint } from '@/lib/passwordError'

export function ChangePasswordForm() {
  const { logout } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError('')
    if (next !== confirm) {
      setError(t.settings.passwordMismatch)
      return
    }
    setLoading(true)
    try {
      await changePassword(current, next)
      logout()
      navigate('/login', { replace: true, state: { passwordChanged: true } })
    } catch (err) {
      setError(passwordErrorMessage(err))
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-1.5">
        <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          {t.settings.currentPassword}
        </label>
        <Input
          type="password"
          placeholder="••••••••"
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
          required
          autoComplete="current-password"
          className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl"
        />
      </div>

      <div className="space-y-1.5">
        <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          {t.settings.newPassword}
        </label>
        <Input
          type="password"
          placeholder="••••••••"
          value={next}
          onChange={(e) => setNext(e.target.value)}
          minLength={12}
          maxLength={72}
          required
          autoComplete="new-password"
          className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl"
        />
        <p className="text-xs text-muted-foreground leading-snug">{passwordRequirementsHint}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          {t.settings.confirmPassword}
        </label>
        <Input
          type="password"
          placeholder="••••••••"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          minLength={12}
          maxLength={72}
          required
          autoComplete="new-password"
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
        {loading ? t.settings.changing : t.settings.changePassword}
      </Button>
    </form>
  )
}
