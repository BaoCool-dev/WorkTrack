import type { Role } from '@/features/auth'

export type ProjectStatus = 'new' | 'in_progress' | 'completed' | 'paused' | 'cancelled'
export type StatusGroup = 'todo' | 'in_progress' | 'done'
export type TransitionRole = 'manager' | 'member'
export type MemberStatus = 'pending' | 'active' | 'disabled'

export const PROJECT_STATUSES: readonly ProjectStatus[] = [
  'new',
  'in_progress',
  'completed',
  'paused',
  'cancelled',
]
export const STATUS_GROUPS: readonly StatusGroup[] = ['todo', 'in_progress', 'done']
export const TRANSITION_ROLES: readonly TransitionRole[] = ['manager', 'member']

// Khớp domain/project.go.
export const PROJECT_NAME_MAX = 150
export const PROJECT_DESCRIPTION_MAX = 2000
export const COLUMN_NAME_MAX = 50
export const MIN_COLUMNS = 2
export const MAX_COLUMNS = 8
export const PAGE_SIZE = 20
export const WIP_LIMIT_MIN = 1
export const WIP_LIMIT_MAX = 20

export interface Page<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
}

export interface Project {
  id: string
  name: string
  description: string
  status: ProjectStatus
  /** "YYYY-MM-DD" hoặc null. */
  startDate: string | null
  endDate: string | null
  manager: { id: string; name: string }
  memberCount: number
  /** Số công việc cha "đang làm" tối đa mỗi người trong dự án (CF-05, mặc định 5). */
  wipLimit: number
  createdAt: string
  updatedAt: string
}

export interface ProjectInput {
  name: string
  description: string
  status?: ProjectStatus
  startDate: string | null
  endDate: string | null
}

export interface ProjectFilters {
  search: string
  status: ProjectStatus | ''
}

export interface ProjectMember {
  userId: string
  fullName: string
  email: string
  role: Role
  status: MemberStatus
  addedAt?: string
}

export interface BoardColumn {
  id: string
  name: string
  position: number
  statusGroup: StatusGroup
  taskCount: number
}

export interface Transition {
  fromColumnId: string
  toColumnId: string
  role: TransitionRole
}

export interface Board {
  columns: BoardColumn[]
  transitions: Transition[]
}
