import { Capacitor } from '@capacitor/core'
import { Preferences } from '@capacitor/preferences'

// Cache in-process pour accès synchrone depuis les interceptors axios.
// Sur web, localStorage est synchrone donc ce cache n'est pas utilisé.
const memCache: Record<string, string | null> = {}

/**
 * À appeler dans main.tsx AVANT le premier render React.
 * Pré-charge les tokens depuis Preferences dans le cache mémoire
 * pour que getSync() retourne la bonne valeur dès le premier render.
 * No-op sur web.
 */
export async function initTokens(): Promise<void> {
  if (!Capacitor.isNativePlatform()) return
  const keys = ['breezy_token', 'breezy_refresh_token']
  await Promise.all(
    keys.map(async (key) => {
      const { value } = await Preferences.get({ key })
      memCache[key] = value
    }),
  )
}

export const storage = {
  /** Lecture synchrone — lit depuis memCache sur native, localStorage sur web. */
  getSync(key: string): string | null {
    if (Capacitor.isNativePlatform()) return memCache[key] ?? null
    return localStorage.getItem(key)
  },

  /**
   * Écriture — met à jour memCache immédiatement (synchrone) puis persiste
   * de façon asynchrone dans Preferences (native) ou localStorage (web).
   * La lecture via getSync() retourne la nouvelle valeur dès l'appel, même
   * avant que la Promise se resolve.
   */
  async set(key: string, value: string): Promise<void> {
    memCache[key] = value
    if (Capacitor.isNativePlatform()) {
      await Preferences.set({ key, value })
    } else {
      localStorage.setItem(key, value)
    }
  },

  async remove(key: string): Promise<void> {
    memCache[key] = null
    if (Capacitor.isNativePlatform()) {
      await Preferences.remove({ key })
    } else {
      localStorage.removeItem(key)
    }
  },
}
