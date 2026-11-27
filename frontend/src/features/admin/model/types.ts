import type { Permission, Role } from '@/features/auth'

export type UserStatus = 'pending' | 'active' | 'disabled'
export type SkillLevel = 'beginner' | 'intermediate' | 'senior'

export const ROLES: readonly Role[] = ['admin', 'manager', 'member', 'ceo']
export const STATUSES: readonly UserStatus[] = ['pending', 'active', 'disabled']
export const SKILL_LEVELS: readonly SkillLevel[] = ['beginner', 'intermediate', 'senior']
export const MAX_SKILLS = 50
export const PAGE_SIZE = 20

export interface Ref {
  id: string
  name: string
}

export interface UserSkill {
  skillId: string
  name: string
  level: SkillLevel
}

export interface AdminUser {
  id: string
  email: string
  fullName: string
  role: Role
  status: UserStatus
  lastLoginAt: string | null
  createdAt: string
  permissions: Permission[]
  department: Ref | null
  jobTitle: Ref | null
  skills: UserSkill[]
}

/** B9: phân trang trả {items, page, pageSize, total}. */
export interface Page<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
}

export interface UserFilters {
  search: string
  role: Role | ''
  status: UserStatus | ''
  departmentId: string
  jobTitleId: string
}

export const emptyFilters: UserFilters = {
  search: '',
  role: '',
  status: '',
  departmentId: '',
  jobTitleId: '',
}

export interface Department {
  id: string
  name: string
  parentId: string | null
  memberCount: number
}

export interface CatalogItem {
  id: string
  name: string
  memberCount: number
}

export interface CreateUserInput {
  email: string
  fullName: string
  role: Role
  departmentId: string | null
  jobTitleId: string | null
}

export type UpdateUserInput = Omit<CreateUserInput, 'email'>

export interface CreateUserResult {
  user: AdminUser
  emailSent: boolean
}
