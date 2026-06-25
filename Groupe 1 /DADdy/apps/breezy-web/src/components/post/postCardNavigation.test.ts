// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'
import { shouldOpenPostFromTarget } from './postCardNavigation'

describe('shouldOpenPostFromTarget', () => {
  it('opens the post from non-interactive card content', () => {
    const content = document.createElement('p')

    expect(shouldOpenPostFromTarget(content)).toBe(true)
  })

  it.each(['a', 'button', 'input', 'textarea', 'select', 'video', 'audio'])(
    'ignores clicks inside %s elements',
    (tagName) => {
      const interactive = document.createElement(tagName)
      const child = document.createElement('span')
      interactive.append(child)

      expect(shouldOpenPostFromTarget(child)).toBe(false)
    },
  )

  it('ignores elements exposed as buttons', () => {
    const interactive = document.createElement('div')
    interactive.setAttribute('role', 'button')

    expect(shouldOpenPostFromTarget(interactive)).toBe(false)
  })

  it('ignores non-element event targets', () => {
    expect(shouldOpenPostFromTarget(document)).toBe(false)
  })
})
