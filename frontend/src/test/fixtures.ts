import type { CurrentUser, Role, SessionResponse } from '@/features/auth/model/types'

const PERMS: Record<Role, CurrentUser['permissions']> = {
  admin: ['users.manage', 'org.manage'],
  ceo: [
    'users.manage',
    'org.manage',
    'users.assign_ceo',
    'projects.view_all',
    'dashboard.company.view',
  ],
  manager: ['projects.create', 'dashboard.team.view'],
  member: [],
}

const NAMES: Record<Role, string> = {
  admin: 'Trần Thị Mai',
  ceo: 'Phạm Quốc Hùng',
  manager: 'Nguyễn Văn An',
  member: 'Đỗ Thu Hà',
}

export function makeUser(role: Role = 'member'): CurrentUser {
  return {
    id: `u-${role}`,
    email: `${role}@worktrack.local`,
    fullName: NAMES[role],
    role,
    status: 'active',
    lastLoginAt: null,
    createdAt: '2026-11-01T00:00:00Z',
    permissions: PERMS[role],
  }
}

export function makeSession(role: Role = 'member', token = 'tok-1'): SessionResponse {
  return {
    accessToken: token,
    expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
    user: makeUser(role),
  }
}
