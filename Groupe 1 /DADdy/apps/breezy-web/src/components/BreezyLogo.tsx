import { useTheme } from '@/hooks/useTheme'

interface BreezyLogoProps {
  /** 'icon' = B only | 'text' = wordmark only | 'full' = both */
  variant?: 'icon' | 'text' | 'full'
  size?: number
  className?: string
}

export function BreezyLogo({ variant = 'full', size = 32, className = '' }: BreezyLogoProps) {
  const { theme } = useTheme()
  const isDark = theme === 'dark'

  if (variant === 'icon') {
    return (
      <img
        src="/logos/logo-icon-purple.png"
        alt="Breezy"
        height={size}
        width={size}
        className={`object-contain ${className}`}
        style={{ height: size, width: size }}
      />
    )
  }

  if (variant === 'text') {
    return (
      <img
        src={isDark ? '/logos/logo-text-white.png' : '/logos/logo-text-dark.png'}
        alt="Breezy"
        height={size}
        className={`object-contain h-auto ${className}`}
        style={{ height: size }}
      />
    )
  }

  // full = icon + text side by side
  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <img
        src="/logos/logo-icon-purple.png"
        alt=""
        style={{ height: size, width: size }}
        className="object-contain"
      />
      <img
        src={isDark ? '/logos/logo-text-white.png' : '/logos/logo-text-dark.png'}
        alt="Breezy"
        style={{ height: size * 0.65 }}
        className="object-contain h-auto"
      />
    </div>
  )
}
