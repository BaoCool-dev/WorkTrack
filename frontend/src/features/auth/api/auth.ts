import { http } from '@/shared/api/http'
import type { ActivationInfo, CurrentUser, SessionResponse } from '../model/types'

const BASE = '/api/v1/auth'

// Các endpoint /auth công khai dùng auth:false — không gắn Bearer, không tự làm mới khi 401.
export const authApi = {
  login: (email: string, password: string) =>
    http<SessionResponse>(`${BASE}/login`, {
      method: 'POST',
      json: { email, password },
      auth: false,
    }),
  refresh: () => http<SessionResponse>(`${BASE}/refresh`, { method: 'POST', auth: false }),
  logout: () => http<undefined>(`${BASE}/logout`, { method: 'POST', auth: false }),
  me: () => http<CurrentUser>(`${BASE}/me`),
  verifyActivation: (token: string) =>
    http<ActivationInfo>(`${BASE}/activation/verify`, {
      method: 'POST',
      json: { token },
      auth: false,
    }),
  activate: (token: string, password: string) =>
    http<undefined>(`${BASE}/activation`, {
      method: 'POST',
      json: { token, password },
      auth: false,
    }),
}
