import type { Page } from '@/features/projects'
import { http } from '@/shared/api/http'
import type {
  Comment,
  Label,
  MyTaskFilters,
  Settings,
  SubtaskInput,
  Task,
  TaskFilters,
  TaskInput,
} from '../model/types'

const id = encodeURIComponent

export interface TaskListParams extends TaskFilters {
  page: number
  pageSize: number
}

export interface MyTaskListParams extends MyTaskFilters {
  page: number
  pageSize: number
}

function pageQuery(p: { page: number; pageSize: number; search: string; statusGroup: string }) {
  const q = new URLSearchParams({ page: String(p.page), pageSize: String(p.pageSize) })
  const search = p.search.trim()
  if (search) q.set('search', search)
  if (p.statusGroup) q.set('statusGroup', p.statusGroup)
  return q
}

function listQuery(p: TaskListParams): string {
  const q = pageQuery(p)
  if (p.assignee === 'none') q.set('unassigned', 'true')
  else if (p.assignee) q.set('assigneeId', p.assignee)
  if (p.labelId) q.set('labelId', p.labelId)
  return q.toString()
}

export const tasksApi = {
  list: (projectId: string, p: TaskListParams) =>
    http<Page<Task>>(`/api/v1/projects/${id(projectId)}/tasks?${listQuery(p)}`),
  create: (projectId: string, body: TaskInput) =>
    http<Task>(`/api/v1/projects/${id(projectId)}/tasks`, { method: 'POST', json: body }),
  get: (taskId: string) => http<Task>(`/api/v1/tasks/${id(taskId)}`),
  update: (taskId: string, body: TaskInput) =>
    http<Task>(`/api/v1/tasks/${id(taskId)}`, { method: 'PUT', json: body }),
  claim: (taskId: string) => http<Task>(`/api/v1/tasks/${id(taskId)}/claim`, { method: 'POST' }),
  mine: (p: MyTaskListParams) => http<Page<Task>>(`/api/v1/me/tasks?${pageQuery(p).toString()}`),
  myWip: () => http<{ count: number }>('/api/v1/me/wip'),
  labels: () => http<{ items: Label[] }>('/api/v1/labels'),
  createLabel: (name: string) => http<Label>('/api/v1/labels', { method: 'POST', json: { name } }),
  renameLabel: (labelId: string, name: string) =>
    http<Label>(`/api/v1/labels/${id(labelId)}`, { method: 'PUT', json: { name } }),
  deleteLabel: (labelId: string) =>
    http<undefined>(`/api/v1/labels/${id(labelId)}`, { method: 'DELETE' }),
  // CF-07 — công việc con và bình luận.
  subtasks: (taskId: string) => http<{ items: Task[] }>(`/api/v1/tasks/${id(taskId)}/subtasks`),
  createSubtask: (taskId: string, body: SubtaskInput) =>
    http<Task>(`/api/v1/tasks/${id(taskId)}/subtasks`, { method: 'POST', json: body }),
  moveSubtask: (subtaskId: string, columnId: string) =>
    http<Task>(`/api/v1/subtasks/${id(subtaskId)}/move`, { method: 'POST', json: { columnId } }),
  deleteSubtask: (subtaskId: string) =>
    http<undefined>(`/api/v1/subtasks/${id(subtaskId)}`, { method: 'DELETE' }),
  comments: (taskId: string, page: number, pageSize: number) =>
    http<Page<Comment>>(
      `/api/v1/tasks/${id(taskId)}/comments?${new URLSearchParams({ page: String(page), pageSize: String(pageSize) }).toString()}`,
    ),
  createComment: (taskId: string, body: string) =>
    http<Comment>(`/api/v1/tasks/${id(taskId)}/comments`, { method: 'POST', json: { body } }),
  settings: () => http<Settings>('/api/v1/settings'),
  updateSettings: (body: { storyPointHours: number }) =>
    http<Settings>('/api/v1/settings', { method: 'PUT', json: body }),
}
