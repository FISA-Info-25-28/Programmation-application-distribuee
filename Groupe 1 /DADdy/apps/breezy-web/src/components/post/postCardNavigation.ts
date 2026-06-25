const INTERACTIVE_POST_SELECTOR = 'a, button, input, textarea, select, video, audio, [role="button"]'

export function shouldOpenPostFromTarget(target: EventTarget | null): boolean {
  return target instanceof Element && target.closest(INTERACTIVE_POST_SELECTOR) === null
}
