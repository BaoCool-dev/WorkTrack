import { makeTask } from '@/features/tasks/test-data'
import type { Task } from '@/features/tasks'

/**
 * WebSocket giả cho test: không mở kết nối thật, test tự "bắn" open/message/close.
 * Gắn bằng vi.stubGlobal('WebSocket', FakeSocket); setup.ts tự gỡ sau mỗi test.
 */
export class FakeSocket {
  static instances: FakeSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  closed = false
  url: string

  constructor(url: string) {
    this.url = url
    FakeSocket.instances.push(this)
  }

  close() {
    this.closed = true
  }

  open() {
    this.onopen?.()
  }

  emit(event: object) {
    this.onmessage?.({ data: JSON.stringify(event) })
  }

  drop() {
    this.onclose?.()
  }

  static last(): FakeSocket {
    const s = FakeSocket.instances.at(-1)
    if (!s) throw new Error('chưa có WebSocket nào được mở')
    return s
  }

  static reset() {
    FakeSocket.instances = []
  }
}

const HA = { id: 'u-member', name: 'Đỗ Thu Hà' }
const MANH = { id: 'u-3', name: 'Lê Mạnh' }
const BACKLOG = { id: 'c-1', name: 'Backlog', statusGroup: 'todo' as const }
const DOING = { id: 'c-2', name: 'In Progress', statusGroup: 'in_progress' as const }

/** Bảng mẫu (dùng cột của makeBoard): 2 thẻ Backlog, 1 thẻ đang làm. */
export function boardTasks(): Task[] {
  return [
    makeTask({
      id: 't-1',
      title: 'Thiết kế API đăng nhập',
      column: BACKLOG,
      position: 0,
      assignee: HA,
    }),
    makeTask({
      id: 't-2',
      title: 'Viết trang hồ sơ',
      column: BACKLOG,
      position: 1,
      assignee: MANH,
      labels: [{ id: 'l-2', name: 'Frontend' }],
    }),
    makeTask({
      id: 't-3',
      title: 'Sửa lỗi phân trang',
      column: DOING,
      position: 0,
      assignee: HA,
      storyPoint: 3,
    }),
  ]
}
