import { http } from '@/shared/api/http'
import type {
  Board,
  Page,
  Project,
  ProjectFilters,
  ProjectInput,
  ProjectMember,
  StatusGroup,
  Transition,
} from '../model/types'

const BASE = '/api/v1/projects'
const id = encodeURIComponent

export interface ProjectListParams extends ProjectFilters {
  page: number
  pageSize: number
}

function query(p: ProjectListParams): string {
  const q = new URLSearchParams({ page: String(p.page), pageSize: String(p.pageSize) })
  const search = p.search.trim()
  if (search) q.set('search', search)
  if (p.status) q.set('status', p.status)
  return q.toString()
}

export interface ColumnInput {
  name: string
  statusGroup: StatusGroup
}

export const projectsApi = {
  list: (p: ProjectListParams) => http<Page<Project>>(`${BASE}?${query(p)}`),
  get: (projectId: string) => http<Project>(`${BASE}/${id(projectId)}`),
  create: (body: ProjectInput) => http<Project>(BASE, { method: 'POST', json: body }),
  update: (projectId: string, body: ProjectInput) =>
    http<Project>(`${BASE}/${id(projectId)}`, { method: 'PUT', json: body }),

  members: (projectId: string) =>
    http<{ items: ProjectMember[] }>(`${BASE}/${id(projectId)}/members`),
  candidates: (projectId: string, search: string) => {
    const q = search.trim() ? `?${new URLSearchParams({ search: search.trim() }).toString()}` : ''
    return http<{ items: ProjectMember[] }>(`${BASE}/${id(projectId)}/member-candidates${q}`)
  },
  addMember: (projectId: string, userId: string) =>
    http<undefined>(`${BASE}/${id(projectId)}/members`, { method: 'POST', json: { userId } }),
  removeMember: (projectId: string, userId: string) =>
    http<undefined>(`${BASE}/${id(projectId)}/members/${id(userId)}`, { method: 'DELETE' }),

  board: (projectId: string) => http<Board>(`${BASE}/${id(projectId)}/board`),
  addColumn: (projectId: string, body: ColumnInput) =>
    http<Board>(`${BASE}/${id(projectId)}/columns`, { method: 'POST', json: body }),
  updateColumn: (projectId: string, columnId: string, body: ColumnInput) =>
    http<Board>(`${BASE}/${id(projectId)}/columns/${id(columnId)}`, { method: 'PUT', json: body }),
  deleteColumn: (projectId: string, columnId: string) =>
    http<Board>(`${BASE}/${id(projectId)}/columns/${id(columnId)}`, { method: 'DELETE' }),
  reorderColumns: (projectId: string, columnIds: string[]) =>
    http<Board>(`${BASE}/${id(projectId)}/column-order`, {
      method: 'PUT',
      json: { columnIds },
    }),
  replaceTransitions: (projectId: string, transitions: Transition[]) =>
    http<Board>(`${BASE}/${id(projectId)}/transitions`, {
      method: 'PUT',
      json: { transitions },
    }),
}
