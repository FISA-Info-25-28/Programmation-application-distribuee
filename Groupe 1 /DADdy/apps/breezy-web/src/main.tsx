import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { initTokens } from '@breezy/shared'
import './index.css'
import App from './App.tsx'

// Pré-charge les tokens JWT depuis le storage natif avant le premier render.
// No-op sur web (localStorage est synchrone et déjà disponible).
initTokens().then(async () => {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
  try {
    const { SplashScreen } = await import('@capacitor/splash-screen')
    await SplashScreen.hide({ fadeOutDuration: 200 })
  } catch {
    // not in Capacitor context
  }
})
