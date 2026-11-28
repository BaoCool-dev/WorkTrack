import type { Comment, Label, Task, TimeLog } from './model/types'

export function makeTask(over: Partial<Task> = {}): Task {
  return {
    id: 't-1',
    projectId: 'p-1',
    projectName: 'Cổng thông tin khách hàng',
    parentId: null,
    title: 'Thiết kế API đăng nhập',
    description: 'Endpoint đăng nhập + refresh token.',
    storyPoint: 5,
    estimatedMinutes: 1200,
    column: { id: 'c-1', name: 'Backlog', statusGroup: 'todo' },
    assignee: null,
    dueDate: '2026-12-15',
    labels: [{ id: 'l-1', name: 'Backend' }],
    createdBy: { id: 'u-manager', name: 'Trần Bình' },
    position: 0,
    version: 1,
    loggedMinutes: 0,
    actualMinutes: 0,
    subtaskCount: 0,
    subtaskDoneCount: 0,
    createdAt: '2026-11-20T02:00:00Z',
    updatedAt: '2026-11-20T02:00:00Z',
    ...over,
  }
}

export const LABELS: Label[] = [
  { id: 'l-1', name: 'Backend', taskCount: 2 },
  { id: 'l-2', name: 'Frontend', taskCount: 0 },
]

export function page<T>(items: T[], total = items.length) {
  return { items, page: 1, pageSize: 20, total }
}

export function makeSubtask(over: Partial<Task> = {}): Task {
  return makeTask({
    id: 's-1',
    parentId: 't-1',
    title: 'Viết handler đăng nhập',
    description: '',
    storyPoint: null,
    estimatedMinutes: null,
    labels: [],
    dueDate: null,
    ...over,
  })
}

export function makeComment(over: Partial<Comment> = {}): Comment {
  return {
    id: 'cm-1',
    taskId: 't-1',
    author: { id: 'u-manager', name: 'Nguyễn Văn An' },
    body: 'Nhớ thêm rate limit.',
    createdAt: '2026-11-27T03:00:00Z',
    ...over,
  }
}

export function makeTimeLog(over: Partial<TimeLog> = {}): TimeLog {
  return {
    id: 'tl-1',
    taskId: 't-1',
    taskTitle: 'Thiết kế API đăng nhập',
    parentId: null,
    user: { id: 'u-member', name: 'Đỗ Thu Hà' },
    minutes: 90,
    note: 'Viết handler và test.',
    workDate: '2026-11-26',
    createdAt: '2026-11-26T09:00:00Z',
    ...over,
  }
}
