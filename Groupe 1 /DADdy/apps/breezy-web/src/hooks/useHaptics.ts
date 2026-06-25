import { Capacitor } from '@capacitor/core'
import { Haptics, ImpactStyle, NotificationType } from '@capacitor/haptics'

const isNative = Capacitor.isNativePlatform()

export function useHaptics() {
  const impact = (style: ImpactStyle = ImpactStyle.Light) => {
    if (!isNative) return
    Haptics.impact({ style }).catch(() => {})
  }

  const notification = (type: NotificationType = NotificationType.Success) => {
    if (!isNative) return
    Haptics.notification({ type }).catch(() => {})
  }

  return { impact, notification, ImpactStyle, NotificationType }
}
