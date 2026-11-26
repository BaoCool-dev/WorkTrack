import type { Ref, Task } from '@/features/tasks'
import { messages } from '@/shared/lib/messages'

/**
 * Các phép biến đổi thuần trên danh sách thẻ của bảng. Dùng cho cập nhật lạc quan:
 * giao diện đổi ngay khi thả, rồi server xác nhận (hoặc trả về và ta khôi phục bản cũ).
 */

export function columnTasks(tasks: readonly Task[], columnId: string): Task[] {
  return tasks.filter((t) => t.column.id === columnId).sort((a, b) => a.position - b.position)
}

/**
 * Vị trí gửi lên server khi thả. Server bỏ thẻ ra khỏi cột trước rồi mới chèn (InsertAt),
 * nên ở đây cũng tính chỉ số trên cột ĐÃ BỎ thẻ đang kéo.
 * overTaskId = null nghĩa là thả vào khoảng trống của cột → cuối cột.
 */
export function dropPosition(
  tasks: readonly Task[],
  taskId: string,
  toColumnId: string,
  overTaskId: string | null,
): number {
  const column = columnTasks(tasks, toColumnId)
  const rest = column.filter((t) => t.id !== taskId)
  const i = overTaskId ? rest.findIndex((t) => t.id === overTaskId) : -1
  if (i === -1) return rest.length
  // Kéo XUỐNG trong cùng cột: thả lên thẻ bên dưới nghĩa là đứng SAU thẻ đó
  // (nếu không, kéo thẻ đầu lên thẻ thứ hai sẽ không đổi gì).
  const from = column.findIndex((t) => t.id === taskId)
  const over = column.findIndex((t) => t.id === overTaskId)
  return from !== -1 && from < over ? i + 1 : i
}

/** Chuyển thẻ sang cột `to` ở vị trí `position`, đánh lại thứ tự 0..n-1 cho các cột bị ảnh hưởng. */
export function moveTask(
  tasks: readonly Task[],
  taskId: string,
  to: Task['column'],
  position: number,
): Task[] {
  const moving = tasks.find((t) => t.id === taskId)
  if (!moving) return [...tasks]
  const fromId = moving.column.id

  const target = columnTasks(tasks, to.id).filter((t) => t.id !== taskId)
  const at = Math.max(0, Math.min(position, target.length))
  target.splice(at, 0, { ...moving, column: to })

  const renumbered = new Map<string, Task>()
  target.forEach((t, i) => renumbered.set(t.id, { ...t, position: i }))
  if (fromId !== to.id) {
    columnTasks(tasks, fromId)
      .filter((t) => t.id !== taskId)
      .forEach((t, i) => renumbered.set(t.id, { ...t, position: i }))
  }
  return tasks.map((t) => renumbered.get(t.id) ?? t)
}

/** WIP mỗi người = số việc cha (bảng chỉ có việc cha) ở cột thuộc nhóm đang làm. */
export function wipByAssignee(tasks: readonly Task[]): Map<string, number> {
  const counts = new Map<string, number>()
  for (const t of tasks) {
    if (t.column.statusGroup !== 'in_progress' || !t.assignee) continue
    counts.set(t.assignee.id, (counts.get(t.assignee.id) ?? 0) + 1)
  }
  return counts
}

export interface BoardFilters {
  labelIds: string[]
  /** 'none' = chưa giao. */
  assigneeIds: string[]
}

export const UNASSIGNED = 'none'

/** Trong một nhóm bộ lọc là HOẶC, giữa nhãn và người nhận là VÀ. Rỗng = không lọc. */
export function filterTasks(tasks: readonly Task[], f: BoardFilters): Task[] {
  return tasks.filter((t) => {
    const labelOk = f.labelIds.length === 0 || t.labels.some((l) => f.labelIds.includes(l.id))
    const who = t.assignee?.id ?? UNASSIGNED
    const assigneeOk = f.assigneeIds.length === 0 || f.assigneeIds.includes(who)
    return labelOk && assigneeOk
  })
}

/** Các lựa chọn lọc lấy từ chính các thẻ trên bảng — không hiện nhãn/người không có thẻ nào. */
export function filterOptions(tasks: readonly Task[], unassignedName = messages.kanban.unassigned) {
  const labels = new Map<string, Ref>()
  const people = new Map<string, Ref>()
  for (const task of tasks) {
    for (const l of task.labels) labels.set(l.id, l)
    const who = task.assignee ?? { id: UNASSIGNED, name: unassignedName }
    people.set(who.id, who)
  }
  const byName = (a: Ref, b: Ref) => a.name.localeCompare(b.name, 'vi')
  return { labels: [...labels.values()].sort(byName), people: [...people.values()].sort(byName) }
}
