import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      include: ['src/lib/**', 'src/components/**/*.ts'],
      exclude: ['src/components/ui/**'],
      thresholds: { lines: 90, functions: 90, branches: 90 },
    },
  },
})
