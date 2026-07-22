import { defineConfig } from 'vitest/config'
import path from 'node:path'

export default defineConfig({
  resolve: { alias: { '@': path.resolve(__dirname, '.') } },
  test: {
    environment: 'jsdom',
    setupFiles: ['./test/setup.ts'],
    include: ['**/*.test.{ts,tsx}'],
    coverage: { reporter: ['text', 'json-summary'], include: ['app/auth/**', 'app/lib/**', 'app/auth-context.tsx'] },
  },
})
