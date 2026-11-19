import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, vi } from 'vitest'
import { server } from './server'

// Request nào không có handler → lỗi ngay, tránh test âm thầm gọi mạng thật.
beforeAll(() => {
  server.listen({ onUnhandledFrame: 'error' })
})

afterEach(() => {
  cleanup()
  server.resetHandlers()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

afterAll(() => {
  server.close()
})
