import type { Role } from '@/features/auth'
import type { BoardColumn, Transition, TransitionRole } from '@/features/projects'
import type { Task } from '@/features/tasks'

/**
 * Luật thả thẻ phía giao diện — CHÉP LẠI đúng thứ tự kiểm của backend (kanban.Service.Move)
 * để tô vùng thả hợp lệ/bị chặn ngay khi đang kéo. Server vẫn là nguồn sự thật: nếu hai bên lệch
 * nhau, server từ chối và giao diện hiện thông báo của server.
 */

/** null = không được kéo thả (CEO, Admin). */
export type BoardRole = TransitionRole | null

export interface Viewer {
  id: string
  role: Role
}

/**
 * Vì sao Manager khác lại dùng luật member: quyền kéo mọi thẻ chỉ thuộc Manager PHỤ TRÁCH dự án.
 * Manager khác tham gia dự án chỉ như thành viên (khớp domain.BoardRole).
 */
export function boardRoleOf(viewer: Viewer, projectManagerId: string): BoardRole {
  if (viewer.role === 'manager' && viewer.id === projectManagerId) return 'manager'
  if (viewer.role === 'manager' || viewer.role === 'member') return 'member'
  return null
}

/** Member chỉ kéo thẻ đang giao cho chính mình. */
export function canDragTask(role: BoardRole, viewerId: string, task: Task): boolean {
  if (role === 'manager') return true
  if (role === 'member') return task.assignee?.id === viewerId
  return false
}

export type DropBlockCode = 'not_assignee' | 'done_group' | 'no_rule' | 'wip'

export type DropCheck = { ok: true } | { ok: false; code: DropBlockCode; reason: string }

export interface DropInput {
  task: Task
  from: BoardColumn
  to: BoardColumn
  transitions: Transition[]
  boardRole: BoardRole
  viewerId: string
  /** Số việc cha đang làm của người nhận thẻ trong dự án (trước khi thả). */
  wipCount: number
  wipLimit: number
}

const block = (code: DropBlockCode, reason: string): DropCheck => ({ ok: false, code, reason })

/** WIP chỉ tăng khi thẻ đi TỪ ngoài VÀO nhóm đang làm; kéo trong nhóm không đổi số việc đang làm. */
export function entersWip(from: BoardColumn, to: BoardColumn): boolean {
  return to.statusGroup === 'in_progress' && from.statusGroup !== 'in_progress'
}

export function checkDrop(i: DropInput): DropCheck {
  if (!canDragTask(i.boardRole, i.viewerId, i.task)) {
    return block('not_assignee', 'Bạn chỉ kéo được công việc đang giao cho mình')
  }
  // Đổi thứ tự trong cùng cột không đổi trạng thái → không cần luật chuyển cột.
  if (i.from.id === i.to.id) return { ok: true }

  // Nghiệm thu là việc của Quản lý: Member không đưa việc vào Done, cũng không kéo việc đã Done ra.
  const touchesDone = i.from.statusGroup === 'done' || i.to.statusGroup === 'done'
  if (i.boardRole !== 'manager' && touchesDone) {
    return block('done_group', 'Chỉ Quản lý dự án chuyển việc sang nhóm hoàn thành')
  }

  const allowed = i.transitions.some(
    (t) => t.fromColumnId === i.from.id && t.toColumnId === i.to.id && t.role === i.boardRole,
  )
  if (!allowed) {
    return block('no_rule', `Quy trình không cho chuyển từ “${i.from.name}” sang “${i.to.name}”`)
  }

  if (i.task.assignee && entersWip(i.from, i.to) && i.wipCount >= i.wipLimit) {
    return block('wip', `${i.task.assignee.name} đã đủ ${String(i.wipLimit)} việc đang làm`)
  }
  return { ok: true }
}
