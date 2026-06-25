import { useTheme } from '@/hooks/useTheme'

function rand(min: number, max: number) {
  return Math.random() * (max - min) + min
}

function randInt(min: number, max: number) {
  return Math.floor(rand(min, max))
}

export function AnimatedBackground() {
  const { theme } = useTheme()
  const isDark = theme === 'dark'

  const darkOrbs = DARK_ORBS
  const lightGradient = LIGHT_GRADIENT

  return (
    <div className="fixed inset-0 overflow-hidden pointer-events-none" style={{ zIndex: 0 }}>
      <div
        className="absolute inset-0 transition-opacity duration-300"
        style={{ background: lightGradient, opacity: isDark ? 0 : 1 }}
      />
      <div
        className="absolute inset-0 transition-opacity duration-300"
        style={{ opacity: isDark ? 1 : 0 }}
      >
        {darkOrbs.map((orb, i) => (
          <div
            key={i}
            className="absolute rounded-full"
            style={{
              width: orb.w,
              height: orb.w,
              top: orb.top,
              left: orb.left,
              background: `radial-gradient(circle, ${orb.color} 0%, transparent 70%)`,
              filter: `blur(${orb.blur}px)`,
              animation: orb.anim,
              animationDelay: orb.delay,
            }}
          />
        ))}
      </div>
    </div>
  )
}

type Orb = {
  w: number
  top: string
  left: string
  color: string
  blur: number
  anim: string
  delay: string
}

function buildDarkOrbs(): Orb[] {
  return Array.from({ length: 4 }, (_, i) => ({
    w: randInt(400, 720),
    top: `${rand(-15, 65)}%`,
    left: `${rand(-15, 70)}%`,
    color: `oklch(${rand(0.40, 0.55).toFixed(2)} ${rand(0.20, 0.26).toFixed(2)} ${randInt(282, 302)} / ${rand(0.30, 0.44).toFixed(2)})`,
    blur: randInt(55, 85),
    anim: `orb-${(i % 4) + 1} ${rand(18, 30).toFixed(0)}s ease-in-out infinite`,
    delay: `-${rand(0, 12).toFixed(1)}s`,
  }))
}

function buildLightGradient(): string {
  const base = `linear-gradient(90deg,
    oklch(0.64 0.10 285) 0%,
    oklch(0.74 0.08 288) 28%,
    oklch(0.84 0.06 290) 55%,
    oklch(0.92 0.035 292) 78%,
    oklch(0.98 0.02 292) 100%
  )`

  const accent1 = `radial-gradient(ellipse 70% 100% at 8% 40%, oklch(0.66 0.11 285 / 0.34) 0%, oklch(0.66 0.11 285 / 0) 70%)`
  const accent2 = `radial-gradient(ellipse 60% 80% at 28% 15%, oklch(0.72 0.10 290 / 0.30) 0%, oklch(0.72 0.10 290 / 0) 72%)`
  const accent3 = `radial-gradient(ellipse 65% 85% at 45% 70%, oklch(0.80 0.08 292 / 0.25) 0%, oklch(0.80 0.08 292 / 0) 75%)`
  const accent4 = `radial-gradient(ellipse 75% 95% at 70% 35%, oklch(0.88 0.06 292 / 0.22) 0%, oklch(0.88 0.06 292 / 0) 78%)`

  return [accent1, accent2, accent3, accent4, base].join(', ')
}

const DARK_ORBS = buildDarkOrbs()
const LIGHT_GRADIENT = buildLightGradient()
