import { useState, useEffect, useRef } from 'react'
import { Capacitor } from '@capacitor/core'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { AuthShell } from '@/components/auth/AuthShell'
import { ResendVerification } from '@/components/auth/ResendVerification'
import { useAuth } from '@/hooks/useAuth'
import { useLanguage } from '@/hooks/useLanguage'
import { login as loginApi, verifyMFA } from '@/api/auth'
import { classifyLoginError } from '@/lib/loginError'

const LOCKED_UNTIL_KEY = 'breezy_login_locked_until'
const WARN_THRESHOLD = 3
const BLOCK_THRESHOLD = 5
const BLOCK_STEP_MS = 30_000

// Same-origin path: nginx (prod) and Vite (dev) proxy `/api` to the api-gateway,
// which forwards `/auth/oauth/*` to the auth-service. Keeping it relative avoids
// hardcoding a backend host (the old default leaked `http://localhost:3104`).
function oauthRedirectURL(provider: string) {
  return `/api/auth/oauth/${provider}`
}

function getLockedUntil(): number {
  return parseInt(localStorage.getItem(LOCKED_UNTIL_KEY) ?? '0', 10) || 0
}
function setLockedUntil(ts: number) {
  localStorage.setItem(LOCKED_UNTIL_KEY, String(ts))
}
function clearLockedUntil() {
  localStorage.removeItem(LOCKED_UNTIL_KEY)
}

function formatCountdown(seconds: number): string {
  if (seconds >= 60) {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return s > 0 ? `${m}m ${s}s` : `${m}m`
  }
  return `${seconds}s`
}

export function LoginPage() {
  const { login } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()
  const location = useLocation()
  const passwordChanged = (location.state as { passwordChanged?: boolean } | null)?.passwordChanged
  const oauthError = new URLSearchParams(location.search).get('oauthError')
  const isNativePlatform = Capacitor.isNativePlatform()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [notVerified, setNotVerified] = useState(false)
  const [loading, setLoading] = useState(false)

  const failCountRef = useRef(0)
  const rateLimitHitsRef = useRef(0)
  const [failCount, setFailCount] = useState(0)

  const [countdown, setCountdown] = useState(() => {
    const remaining = Math.max(0, getLockedUntil() - Date.now())
    return remaining > 0 ? Math.ceil(remaining / 1000) : 0
  })

  useEffect(() => {
    if (countdown <= 0) return
    const id = setInterval(() => {
      setCountdown((c) => {
        if (c <= 1) {
          clearInterval(id)
          clearLockedUntil()
          return 0
        }
        return c - 1
      })
    }, 1000)
    return () => clearInterval(id)
  }, [countdown])

  const isLocked = countdown > 0
  const showWarning = !isLocked && failCount >= WARN_THRESHOLD && failCount < BLOCK_THRESHOLD
  const attemptsBeforeLock = Math.max(0, BLOCK_THRESHOLD - failCount)

  function registerFailure() {
    failCountRef.current += 1
    setFailCount(failCountRef.current)
  }

  // MFA step
  const [mfaPendingToken, setMfaPendingToken] = useState<string | null>(null)
  const [mfaCode, setMfaCode] = useState('')
  const [mfaLoading, setMfaLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (isLocked) return
    setError('')
    setNotVerified(false)
    setLoading(true)
    try {
      const result = await loginApi(email, password)
      if ('mfaRequired' in result) {
        setMfaPendingToken(result.pendingToken)
      } else {
        failCountRef.current = 0
        rateLimitHitsRef.current = 0
        setFailCount(0)
        clearLockedUntil()
        login(result.accessToken, result.refreshToken)
        navigate('/')
      }
    } catch (err) {
      const kind = classifyLoginError(err)
      switch (kind) {
        case 'unverified':
          setNotVerified(true)
          break
        case 'suspended':
          setError(t.auth.login.suspendedMessage)
          break
        case 'rateLimited': {
          rateLimitHitsRef.current += 1
          registerFailure()
          const blockMs = Math.min(rateLimitHitsRef.current * BLOCK_STEP_MS, 30 * 60 * 1000)
          const until = Date.now() + blockMs
          setLockedUntil(until)
          setCountdown(Math.ceil(blockMs / 1000))
          break
        }
        case 'credentials':
          registerFailure()
          if (failCountRef.current >= BLOCK_THRESHOLD) {
            const blockMs = BLOCK_STEP_MS
            setLockedUntil(Date.now() + blockMs)
            setCountdown(Math.ceil(blockMs / 1000))
          } else {
            setError(t.auth.login.error)
          }
          break
        case 'network':
          setError(t.auth.login.errorNetwork)
          break
      }
    } finally {
      setLoading(false)
    }
  }

  const handleMFAVerify = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!mfaPendingToken) return
    setError('')
    setMfaLoading(true)
    try {
      const { accessToken, refreshToken } = await verifyMFA(mfaPendingToken, mfaCode.trim())
      login(accessToken, refreshToken)
      navigate('/')
    } catch {
      setError(t.auth.login.mfaError)
    } finally {
      setMfaLoading(false)
    }
  }

  if (mfaPendingToken) {
    return (
      <AuthShell
        title={t.auth.login.mfaTitle}
        subtitle={t.auth.login.mfaSubtitle}
      >
        <form onSubmit={handleMFAVerify} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {t.auth.login.mfaCodeLabel}
            </label>
            <Input
              type="text"
              inputMode="numeric"
              pattern="[0-9A-Fa-f]*"
              placeholder="000000"
              value={mfaCode}
              onChange={(e) => setMfaCode(e.target.value)}
              required
              maxLength={10}
              className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl text-center tracking-widest text-lg"
            />
            <p className="text-xs text-muted-foreground">{t.auth.login.mfaCodeHint}</p>
          </div>

          {error && (
            <p className="text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2.5">
              {error}
            </p>
          )}

          <Button
            type="submit"
            disabled={mfaLoading || mfaCode.trim().length < 6}
            className="w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all"
          >
            {mfaLoading ? t.auth.login.mfaSubmitting : t.auth.login.mfaSubmit}
          </Button>

          <button
            type="button"
            onClick={() => { setMfaPendingToken(null); setMfaCode(''); setError('') }}
            className="w-full text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {t.auth.login.mfaBack}
          </button>
        </form>
      </AuthShell>
    )
  }

  return (
    <AuthShell
      title={t.auth.login.title}
      subtitle={
        <>
          {t.auth.login.noAccount}{' '}
          <Link to="/register" className="text-primary hover:text-primary/80 transition-colors font-medium">
            {t.auth.login.signUp}
          </Link>
        </>
      }
    >
      {passwordChanged && (
        <p className="mb-4 text-xs text-foreground bg-primary/10 border border-primary/20 rounded-xl px-3 py-2.5">
          Votre mot de passe a été modifié et toutes vos sessions ont été
          déconnectées. Reconnectez-vous avec votre nouveau mot de passe.
        </p>
      )}

      {oauthError && (
        <p className="mb-4 text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2.5">
          Connexion sociale échouée. Réessayez ou utilisez une autre méthode.
        </p>
      )}

      {/* Social login — masqué sur natif : le flux OAuth redirige vers
          l'origine web et ne peut pas revenir vers capacitor://localhost sans
          deep link (schéma custom + redirect backend). À réactiver une fois le
          retour par deep link implémenté côté API. */}
      {!isNativePlatform && (
        <>
          <div className="flex gap-3 mb-4">
            <a
              href={oauthRedirectURL('google')}
              className="flex-1 flex items-center justify-center gap-2 h-11 rounded-xl border border-border bg-card hover:bg-muted/40 transition-colors text-sm font-medium text-foreground"
            >
              <GoogleIcon />
              Google
            </a>
            <a
              href={oauthRedirectURL('github')}
              className="flex-1 flex items-center justify-center gap-2 h-11 rounded-xl border border-border bg-card hover:bg-muted/40 transition-colors text-sm font-medium text-foreground"
            >
              <GitHubIcon />
              GitHub
            </a>
          </div>

          <div className="flex items-center gap-3 mb-4">
            <div className="flex-1 h-px bg-border" />
            <span className="text-xs text-muted-foreground">ou</span>
            <div className="flex-1 h-px bg-border" />
          </div>
        </>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            {t.auth.login.email}
          </label>
          <Input
            type="email"
            placeholder={t.auth.login.emailPlaceholder}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            disabled={isLocked}
            className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl disabled:opacity-50"
          />
        </div>

        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {t.auth.login.password}
            </label>
            <Link
              to="/forgot-password"
              className="text-xs text-primary hover:text-primary/80 transition-colors font-medium"
            >
              Mot de passe oublié ?
            </Link>
          </div>
          <Input
            type="password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            disabled={isLocked}
            className="h-11 bg-input border-border focus:border-primary/60 text-foreground placeholder:text-muted-foreground/60 rounded-xl disabled:opacity-50"
          />
        </div>

        {showWarning && (
          <div className="text-xs bg-yellow-500/10 border border-yellow-500/30 text-yellow-700 dark:text-yellow-400 rounded-xl px-3 py-2.5 flex items-start gap-2">
            <span className="shrink-0 mt-0.5">⚠️</span>
            <span>
              Encore{' '}
              <span className="font-semibold">
                {attemptsBeforeLock} tentative{attemptsBeforeLock > 1 ? 's' : ''}
              </span>{' '}
              avant blocage temporaire.
            </span>
          </div>
        )}

        {isLocked && (
          <div className="text-xs bg-destructive/10 border border-destructive/20 text-destructive rounded-xl px-3 py-3 text-center space-y-1">
            <p className="font-semibold">Trop de tentatives</p>
            <p>
              Réessayez dans{' '}
              <span className="font-mono font-bold tabular-nums">{formatCountdown(countdown)}</span>
            </p>
          </div>
        )}

        {error && !isLocked && (
          <p className="text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2.5">
            {error}
          </p>
        )}

        <Button
          type="submit"
          disabled={loading || isLocked}
          className="w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all disabled:opacity-60"
        >
          {loading
            ? t.auth.login.submitting
            : isLocked
              ? `Bloqué — ${formatCountdown(countdown)}`
              : t.auth.login.submit}
        </Button>
      </form>

      {notVerified && (
        <div className="mt-4 space-y-3 border-t border-border pt-4">
          <p className="text-xs text-muted-foreground bg-muted/40 border border-border rounded-xl px-3 py-2.5">
            {t.auth.login.unverifiedMessage}
          </p>
          <ResendVerification defaultEmail={email} />
        </div>
      )}
    </AuthShell>
  )
}

function GoogleIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
    </svg>
  )
}

function GitHubIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
      <path d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.115 2.504.337 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.202 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"/>
    </svg>
  )
}
