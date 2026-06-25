import { useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button-variants'
import { cn } from '@/lib/utils'
import { AuthShell } from '@/components/auth/AuthShell'
import { ResendVerification } from '@/components/auth/ResendVerification'
import { useLanguage } from '@/hooks/useLanguage'
import { verifyEmail } from '@/api/auth'

type Status = 'verifying' | 'success' | 'error' | 'missing'

export function VerifyEmailPage() {
  const { t } = useLanguage()
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [status, setStatus] = useState<Status>(token ? 'verifying' : 'missing')
  const verified = useRef(false)

  useEffect(() => {
    if (!token || verified.current) return
    verified.current = true
    verifyEmail(token)
      .then(() => setStatus('success'))
      .catch(() => setStatus('error'))
  }, [token])

  if (status === 'verifying') {
    return (
      <AuthShell title={t.verify.verifyingTitle} subtitle={t.verify.verifyingSubtitle}>
        <p className="text-sm text-muted-foreground">{t.verify.moment}</p>
      </AuthShell>
    )
  }

  if (status === 'success') {
    return (
      <AuthShell title={t.verify.successTitle} subtitle={t.verify.successSubtitle}>
        <Link
          to="/login"
          className={cn(
            buttonVariants(),
            'w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-xl shadow-lg shadow-primary/25 transition-all',
          )}
        >
          {t.verify.signIn}
        </Link>
      </AuthShell>
    )
  }

  return (
    <AuthShell title={t.verify.errorTitle} subtitle={t.verify.errorSubtitle}>
      <div className="space-y-4">
        <ResendVerification />
        <p className="text-xs text-muted-foreground text-center">
          <Link to="/login" className="text-primary hover:text-primary/80 transition-colors font-medium">
            {t.verify.backToLogin}
          </Link>
        </p>
      </div>
    </AuthShell>
  )
}
