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

export async function http<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response
  try {
    const headers = new Headers(init?.headers)
    if (!headers.has('Accept')) headers.set('Accept', 'application/json')
    res = await fetch(`${API_BASE_URL}${path}`, { ...init, headers, credentials: 'include' })
  } catch {
    throw new ApiError(0, 'NETWORK', messages.errors.network)
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as ErrorBody
    throw new ApiError(
      res.status,
      body.error?.code ?? 'UNKNOWN',
      body.error?.message ?? messages.errors.unknown,
    )
  }
  return (await res.json()) as T
}
