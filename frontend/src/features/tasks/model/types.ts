import type { StatusGroup } from '@/features/projects'

// Khớp domain/task.go.
export const STORY_POINTS = [1, 2, 3, 5, 8, 13, 21] as const
export const TASK_TITLE_MAX = 200
export const TASK_DESCRIPTION_MAX = 5000
export const TASK_LABELS_MAX = 10
export const LABEL_NAME_MAX = 100
export const SP_HOURS_MIN = 1
export const SP_HOURS_MAX = 40
// Khớp domain/collab.go (CF-07).
export const COMMENT_MAX = 2000
export const COMMENT_PAGE_SIZE = 20

export interface Ref {
  id: string
  name: string
}

export interface Task {
  id: string
  projectId: string
  projectName: string
  /** null = công việc cha (gốc); có giá trị = công việc con (CF-07, chỉ 1 cấp). */
  parentId: string | null
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
  /** Thứ tự trong cột (0 = đầu cột). */
  position: number
  /** Tăng sau mỗi lần ghi — gửi kèm khi kéo thả để server phát hiện xung đột (409). */
  version: number
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

/** Body POST công việc con — không có Story Point (CHECK ở CSDL). */
export interface SubtaskInput {
  title: string
  description: string
}

export interface Comment {
  id: string
  taskId: string
  author: Ref
  body: string
  createdAt: string
}
