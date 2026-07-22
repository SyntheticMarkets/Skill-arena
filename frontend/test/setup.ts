import '@testing-library/jest-dom/vitest'

Object.defineProperty(window, 'crypto', {
  value: { ...window.crypto, randomUUID: () => '00000000-0000-4000-8000-000000000001' },
})
