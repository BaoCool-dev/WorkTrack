import type { StatusGroup } from '@/features/projects'

// Khớp domain/task.go.
export const STORY_POINTS = [1, 2, 3, 5, 8, 13, 21] as const
export const TASK_TITLE_MAX = 200
export const TASK_DESCRIPTION_MAX = 5000
export const TASK_LABELS_MAX = 10
export const LABEL_NAME_MAX = 100
export const SP_HOURS_MIN = 1
export const SP_HOURS_MAX = 40

export interface Ref {
  id: string
  name: string
}

export interface Task {
  id: string
  projectId: string
  projectName: string
  title: string
  description: string
  storyPoint: number | null
  /** Giờ ước tính (phút) chụp lại lúc đặt Story Point theo hệ số B1. */
  estimatedMinutes: number | null
  column: { id: string; name: string; statusGroup: StatusGroup }
  assignee: Ref | null
  dueDate: string | null
  labels: Ref[]
  createdBy: Ref
  createdAt: string
  updatedAt: string
}

/** Body POST/PUT công việc cha (PUT thay toàn bộ). */
export interface TaskInput {
  title: string
  description: string
  storyPoint: number | null
  assigneeId: string | null
  dueDate: string | null
  labelIds: string[]
}

export interface TaskFilters {
  search: string
  statusGroup: StatusGroup | ''
  /** '' = tất cả, 'none' = chưa giao, còn lại là id người nhận. */
  assignee: string
  labelId: string
}

export interface MyTaskFilters {
  search: string
  statusGroup: StatusGroup | ''
}

export interface Label {
  id: string
  name: string
  taskCount: number
}

export interface Settings {
  storyPointHours: number
  updatedAt: string
}
