export type Role = 'admin' | 'manager' | 'member' | 'ceo'

export type Permission =
  | 'users.manage'
  | 'users.assign_ceo'
  | 'org.manage'
  | 'projects.create'
  | 'projects.view_all'
  | 'dashboard.team.view'
  | 'dashboard.company.view'

export interface CurrentUser {
  id: string
  email: string
  fullName: string
  role: Role
  status: string
  lastLoginAt: string | null
  createdAt: string
  permissions: Permission[]
}

/** Dạng trả về của POST /auth/login và POST /auth/refresh. */
export interface SessionResponse {
  accessToken: string
  expiresAt: string
  user: CurrentUser
}

export interface ActivationInfo {
  email: string
  fullName: string
}
