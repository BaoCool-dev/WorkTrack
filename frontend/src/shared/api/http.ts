import { messages } from '@/shared/lib/messages'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

/** Lỗi API theo định dạng {"error":{"code","message"}} của backend. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

interface ErrorBody {
  error?: { code?: string; message?: string }
}

/**
 * Cầu nối với phiên đăng nhập. http client không biết token nằm ở đâu —
 * features/auth đăng ký vào đây (token chỉ trong bộ nhớ, không localStorage).
 */
export interface HttpAuth {
  getToken: () => string | null
  /** Làm mới access token (single-flight). Trả token mới hoặc null nếu phiên đã hết. */
  refresh: () => Promise<string | null>
}

let auth: HttpAuth | null = null

export function configureHttpAuth(next: HttpAuth | null): void {
  auth = next
}

/**
 * URL WebSocket kèm access token hiện tại (trình duyệt không cho đặt header Authorization cho WebSocket).
 * Token hết hạn thì làm mới trước — kết nối realtime mở lâu và tự kết nối lại nhiều lần.
 */
export async function wsUrl(
  path: string,
  opts: { refresh?: boolean } = {},
): Promise<string | null> {
  if (!auth) return null
  const token = opts.refresh ? await auth.refresh() : auth.getToken()
  if (!token) return null
  const base = API_BASE_URL.replace(/^http/, 'ws')
  const sep = path.includes('?') ? '&' : '?'
  return `${base}${path}${sep}access_token=${encodeURIComponent(token)}`
}

export interface HttpInit extends Omit<RequestInit, 'body'> {
  /** Body JSON — tự đặt Content-Type. */
  json?: unknown
  body?: RequestInit['body']
  /** false: không gắn Bearer và không tự làm mới khi 401 (các endpoint /auth công khai). */
  auth?: boolean
}

async function send(path: string, init: HttpInit, token: string | null): Promise<Response> {
  const { json, ...rest } = init
  delete rest.auth
  const headers = new Headers(init.headers)
  if (!headers.has('Accept')) headers.set('Accept', 'application/json')
  if (json !== undefined) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)
  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      ...rest,
      body: json !== undefined ? JSON.stringify(json) : init.body,
      headers,
      credentials: 'include',
    })
  } catch {
    throw new ApiError(0, 'NETWORK', messages.errors.network)
  }
}

export async function http<T>(path: string, init: HttpInit = {}): Promise<T> {
  const useAuth = init.auth !== false && auth !== null
  let res = await send(path, init, useAuth ? (auth?.getToken() ?? null) : null)

  // Access token hết hạn → làm mới một lần rồi gửi lại; thất bại thì trả 401 cho nơi gọi.
  if (res.status === 401 && useAuth && auth) {
    const fresh = await auth.refresh()
    if (fresh) res = await send(path, init, fresh)
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as ErrorBody
    throw new ApiError(
      res.status,
      body.error?.code ?? 'UNKNOWN',
      body.error?.message ?? messages.errors.unknown,
    )
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}
