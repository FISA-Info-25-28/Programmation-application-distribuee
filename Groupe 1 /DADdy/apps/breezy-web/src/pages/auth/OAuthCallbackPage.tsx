import { useEffect, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { exchangeOAuthCode } from '@/api/auth'
import { useAuth } from '@/hooks/useAuth'
import { AuthShell } from '@/components/auth/AuthShell'

export function OAuthCallbackPage() {
  const [params] = useSearchParams()
  const { login } = useAuth()
  const navigate = useNavigate()
  const done = useRef(false)

  useEffect(() => {
    if (done.current) return
    done.current = true

    const code = params.get('code')
    const reason = params.get('reason')

    if (reason || !code) {
      navigate('/login?oauthError=' + (reason ?? 'unknown'), { replace: true })
      return
    }

    exchangeOAuthCode(code)
      .then(({ accessToken, refreshToken }) => {
        login(accessToken, refreshToken)
        navigate('/', { replace: true })
      })
      .catch(() => {
        navigate('/login?oauthError=exchange', { replace: true })
      })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <AuthShell title="Connexion en cours…" subtitle="Finalisation de votre connexion sociale.">
      <div className="flex items-center justify-center py-8">
        <div className="h-8 w-8 rounded-full border-2 border-primary border-t-transparent animate-spin" />
      </div>
    </AuthShell>
  )
}
