import { http } from '@/shared/api/http'
import type {
  AdminUser,
  CatalogItem,
  CreateUserInput,
  CreateUserResult,
  Department,
  Page,
  SkillLevel,
  UpdateUserInput,
  UserFilters,
  UserSkill,
} from '../model/types'

const BASE = '/api/v1'

export type CatalogKind = 'job-titles' | 'skills'

export interface UserListParams extends UserFilters {
  page: number
  pageSize: number
}

function query(params: UserListParams): string {
  const q = new URLSearchParams({ page: String(params.page), pageSize: String(params.pageSize) })
  const search = params.search.trim()
  if (search) q.set('search', search)
  if (params.role) q.set('role', params.role)
  if (params.status) q.set('status', params.status)
  if (params.departmentId) q.set('departmentId', params.departmentId)
  if (params.jobTitleId) q.set('jobTitleId', params.jobTitleId)
  return q.toString()
}

const id = encodeURIComponent

export const adminApi = {
  listUsers: (p: UserListParams) => http<Page<AdminUser>>(`${BASE}/users?${query(p)}`),
  createUser: (body: CreateUserInput) =>
    http<CreateUserResult>(`${BASE}/users`, { method: 'POST', json: body }),
  updateUser: (userId: string, body: UpdateUserInput) =>
    http<AdminUser>(`${BASE}/users/${id(userId)}`, { method: 'PUT', json: body }),
  replaceSkills: (userId: string, skills: { skillId: string; level: SkillLevel }[]) =>
    http<{ skills: UserSkill[] }>(`${BASE}/users/${id(userId)}/skills`, {
      method: 'PUT',
      json: { skills },
    }),
  resendActivation: (userId: string) =>
    http<undefined>(`${BASE}/users/${id(userId)}/activation-email`, { method: 'POST' }),

  listDepartments: () => http<{ items: Department[] }>(`${BASE}/departments`),
  createDepartment: (body: { name: string; parentId: string | null }) =>
    http<Department>(`${BASE}/departments`, { method: 'POST', json: body }),
  updateDepartment: (depId: string, body: { name: string; parentId: string | null }) =>
    http<Department>(`${BASE}/departments/${id(depId)}`, { method: 'PUT', json: body }),
  deleteDepartment: (depId: string) =>
    http<undefined>(`${BASE}/departments/${id(depId)}`, { method: 'DELETE' }),

  listCatalog: (kind: CatalogKind) => http<{ items: CatalogItem[] }>(`${BASE}/${kind}`),
  createCatalog: (kind: CatalogKind, name: string) =>
    http<CatalogItem>(`${BASE}/${kind}`, { method: 'POST', json: { name } }),
  updateCatalog: (kind: CatalogKind, itemId: string, name: string) =>
    http<CatalogItem>(`${BASE}/${kind}/${id(itemId)}`, { method: 'PUT', json: { name } }),
  deleteCatalog: (kind: CatalogKind, itemId: string) =>
    http<undefined>(`${BASE}/${kind}/${id(itemId)}`, { method: 'DELETE' }),
}
