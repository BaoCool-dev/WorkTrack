import type { Task } from '@/features/tasks'
import { http } from '@/shared/api/http'

import type { MoveVars } from '../model/types'

const id = encodeURIComponent

export const kanbanApi = {
  boardTasks: (projectId: string) =>
    http<{ items: Task[] }>(`/api/v1/projects/${id(projectId)}/board/tasks`).then((r) => r.items),
  move: ({ taskId, ...body }: MoveVars) =>
    http<Task>(`/api/v1/tasks/${id(taskId)}/move`, { method: 'POST', json: body }),
  socketPath: (projectId: string) => `/api/v1/projects/${id(projectId)}/ws`,
}
