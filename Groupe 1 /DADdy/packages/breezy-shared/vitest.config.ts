import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      include: ['src/lib/**', 'src/api/transform.ts', 'src/api/errors.ts'],
      thresholds: { lines: 90, functions: 90, branches: 90 },
    },
  },
})
