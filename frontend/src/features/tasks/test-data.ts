import type { Label, Task } from './model/types'

export function makeTask(over: Partial<Task> = {}): Task {
  return {
    id: 't-1',
    projectId: 'p-1',
    projectName: 'Cổng thông tin khách hàng',
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
