import { ApiError } from '@/shared/api/http'
import { authApi } from '../api/auth'
import type { CurrentUser, SessionResponse } from './types'

export type SessionState =
  | { status: 'loading'; user: null }
  | { status: 'anonymous'; user: null }
  | { status: 'authenticated'; user: CurrentUser }

/** Làm mới trước khi access token hết hạn bao lâu. */
const REFRESH_LEAD_MS = 60_000
const MIN_REFRESH_DELAY_MS = 5_000

/**
 * Phiên đăng nhập: access token chỉ nằm trong biến của closure (không localStorage),
 * refresh token nằm trong cookie HttpOnly do backend đặt.
 * Dùng với useSyncExternalStore qua subscribe/getSnapshot.
 */
export function createSession() {
  let state: SessionState = { status: 'loading', user: null }
  let token: string | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  // Refresh token xoay vòng: hai request làm mới song song sẽ bị backend coi là dùng lại
  // token cũ và thu hồi cả phiên → mọi lời gọi refresh dùng chung một promise.
  let inflight: Promise<string | null> | null = null
  let started: Promise<void> | null = null
  const listeners = new Set<() => void>()

  function set(next: SessionState) {
    state = next
    listeners.forEach((l) => {
      l()
    })
  }

  function clear() {
    clearTimeout(timer)
    timer = undefined
    token = null
    set({ status: 'anonymous', user: null })
  }

  function apply(res: SessionResponse) {
    token = res.accessToken
    clearTimeout(timer)
    const delay = Math.max(
      new Date(res.expiresAt).getTime() - Date.now() - REFRESH_LEAD_MS,
      MIN_REFRESH_DELAY_MS,
    )
    timer = setTimeout(() => {
      void refresh()
    }, delay)
    set({ status: 'authenticated', user: res.user })
  }

  function refresh(): Promise<string | null> {
    inflight ??= authApi
      .refresh()
      .then((res) => {
        apply(res)
        return res.accessToken
      })
      .catch((err: unknown) => {
        // Lỗi mạng khi đã đăng nhập: giữ phiên, lần gọi API sau sẽ thử lại.
        if (err instanceof ApiError && err.status === 0 && state.status === 'authenticated') {
          return null
        }
        clear()
        return null
      })
      .finally(() => {
        inflight = null
      })
    return inflight
  }

  // Arrow function: an toàn khi tách ra (useSyncExternalStore(session.subscribe, …)).
  return {
    getSnapshot: (): SessionState => state,
    getToken: (): string | null => token,
    subscribe: (listener: () => void): (() => void) => {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
    refresh,
    /** Khôi phục phiên từ cookie khi mở app. Gọi nhiều lần vẫn chỉ chạy một lần. */
    start: (): Promise<void> => {
      started ??= refresh().then(() => undefined)
      return started
    },
    login: async (email: string, password: string): Promise<CurrentUser> => {
      const res = await authApi.login(email, password)
      started ??= Promise.resolve()
      apply(res)
      return res.user
    },
    logout: async (): Promise<void> => {
      try {
        await authApi.logout()
      } catch {
        // Máy chủ lỗi vẫn xóa phiên phía trình duyệt; cookie sẽ hết hạn theo TTL.
      }
      clear()
    },
    dispose: () => {
      clearTimeout(timer)
      listeners.clear()
    },
  }
}

export type Session = ReturnType<typeof createSession>

/** Phiên dùng chung của ứng dụng. */
export const session = createSession()
