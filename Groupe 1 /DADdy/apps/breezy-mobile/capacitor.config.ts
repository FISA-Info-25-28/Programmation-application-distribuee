import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.breezy.mobile.app',
  appName: 'Breezy',
  webDir: '../breezy-web/dist',
  plugins: {
    SplashScreen: {
      launchShowDuration: 2000,
      backgroundColor: '#000000',
      showSpinner: false,
    },
    StatusBar: {
      style: 'Dark',
      backgroundColor: '#000000',
    },
    PushNotifications: {
      presentationOptions: ['badge', 'sound', 'alert'],
    },
    Keyboard: {
      resize: 'none',
      style: 'dark',
      resizeOnFullScreen: true,
    },
  },
  server: {
    androidScheme: 'http',
  },
}

export default config
