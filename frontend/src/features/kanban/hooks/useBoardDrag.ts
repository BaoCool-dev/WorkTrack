import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core'
import { useState } from 'react'

import type { BoardColumn, Transition } from '@/features/projects'
import { hasPendingSubtasks, reviewColumnId, type Task } from '@/features/tasks'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { useToast } from '@/shared/ui'

import { useMoveTask } from '../api/queries'
import { columnTasks, dropPosition, wipByAssignee } from '../model/boardState'
import { checkDrop, type BoardRole, type DropCheck } from '../model/dropRules'

const t = messages.kanban.move

interface Options {
  projectId: string
  tasks: readonly Task[]
  columns: readonly BoardColumn[]
  transitions: Transition[]
  boardRole: BoardRole
  viewerId: string
  wipLimit: number
  /** Mở hộp thoại "Không thể nhận thêm việc" cho người này. */
  onWipFull: (person: { id: string; name: string }) => void
}

/** Đích thả đọc từ dữ liệu droppable: thẻ (chèn trước thẻ đó) hoặc khoảng trống của cột (cuối cột). */
function readTarget(e: DragEndEvent): { columnId: string; overTaskId: string | null } | null {
  const data = e.over?.data.current as { columnId?: string; taskId?: string } | undefined
  if (!data?.columnId) return null
  return { columnId: data.columnId, overTaskId: data.taskId ?? null }
}

/**
 * Toàn bộ logic kéo thả của bảng, tách khỏi phần vẽ để dễ đọc:
 * kiểm luật trước khi gửi (chặn có lý do) → gửi lạc quan → toast "Hoàn tác" hoặc dải lỗi.
 */
export function useBoardDrag(o: Options) {
  const move = useMoveTask(o.projectId)
  const toast = useToast()
  const [activeId, setActiveId] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)

  const wip = wipByAssignee(o.tasks)
  const columnById = new Map(o.columns.map((c) => [c.id, c]))
  const active = o.tasks.find((x) => x.id === activeId) ?? null

  const check = (task: Task, toColumnId: string): DropCheck => {
    const from = columnById.get(task.column.id)
    const to = columnById.get(toColumnId)
    if (!from || !to) return { ok: false, code: 'no_rule', reason: messages.kanban.loadError }
    return checkDrop({
      task,
      from,
      to,
      transitions: o.transitions,
      boardRole: o.boardRole,
      viewerId: o.viewerId,
      wipCount: task.assignee ? (wip.get(task.assignee.id) ?? 0) : 0,
      wipLimit: o.wipLimit,
    })
  }

  /** Lỗi server: WIP → hộp thoại; lỗi nghiệp vụ → câu của server; còn lại (mạng/5xx) → câu chuẩn. */
  const handleError = (task: Task, err: unknown) => {
    if (err instanceof ApiError && err.code === 'TASK_WIP_LIMIT' && task.assignee) {
      o.onWipFull(task.assignee)
      return
    }
    const known = err instanceof ApiError && err.status > 0 && err.status < 500
    setSaveError(known ? t.reverted(err.message) : t.saveFailed)
  }

  const send = (
    task: Task,
    to: BoardColumn,
    position: number,
    version: number,
    undoable: boolean,
  ) => {
    setSaveError(null)
    const fromColumnId = task.column.id
    const fromPosition = task.position
    move.mutate(
      {
        taskId: task.id,
        columnId: to.id,
        position,
        version,
        column: { id: to.id, name: to.name, statusGroup: to.statusGroup },
      },
      {
        onError: (err) => {
          handleError(task, err)
        },
        onSuccess: (saved) => {
          if (!undoable) return
          const msg = moveMessage(saved, fromColumnId, to, o.columns)
          // Chỉ cho hoàn tác khi đường quay lại cũng hợp lệ (vd. Member không kéo ngược ra khỏi Done).
          const back = columnById.get(fromColumnId)
          const canUndo = back !== undefined && check(saved, fromColumnId).ok
          toast.show(
            msg,
            canUndo
              ? {
                  label: t.undo,
                  onClick: () => {
                    send(saved, back, fromPosition, saved.version, false)
                  },
                }
              : undefined,
          )
        },
      },
    )
  }

  const onDragStart = (e: DragStartEvent) => {
    const id = (e.active.data.current as { taskId?: string } | undefined)?.taskId
    setActiveId(id ?? null)
  }

  const onDragEnd = (e: DragEndEvent) => {
    const task = active
    setActiveId(null)
    const target = readTarget(e)
    if (!task || !target) return
    const to = columnById.get(target.columnId)
    if (!to) return

    const verdict = check(task, to.id)
    if (!verdict.ok) {
      // Không bao giờ chặn im lặng: WIP có hộp thoại riêng, các lý do khác báo bằng toast.
      if (verdict.code === 'wip' && task.assignee) o.onWipFull(task.assignee)
      else toast.show(verdict.reason)
      return
    }

    const position = dropPosition(o.tasks, task.id, to.id, target.overTaskId)
    const unchanged = to.id === task.column.id && position === indexInColumn(o.tasks, task)
    if (unchanged) return
    send(task, to, position, task.version, true)
  }

  return {
    active,
    saveError,
    dismissError: () => {
      setSaveError(null)
    },
    /** Trạng thái vùng thả của từng cột khi đang kéo — dùng để tô màu và ghi lý do bị chặn. */
    dropStateFor: (columnId: string) => {
      if (!active) return null
      const v = check(active, columnId)
      return v.ok ? { ok: true as const } : { ok: false as const, reason: v.reason }
    },
    onDragStart,
    onDragEnd,
    onDragCancel: () => {
      setActiveId(null)
    },
  }
}

/**
 * Câu toast sau khi lưu. CF-09: đưa việc cha vào cột chờ nghiệm thu khi còn việc con chưa xong
 * thì vẫn cho chuyển (không chặn) nhưng nhắc rõ — Manager sẽ cân nhắc khi nghiệm thu.
 */
function moveMessage(
  saved: Task,
  fromColumnId: string,
  to: BoardColumn,
  columns: readonly BoardColumn[],
): string {
  if (fromColumnId === to.id) return t.reordered
  if (to.id === reviewColumnId(columns) && hasPendingSubtasks(saved)) {
    return t.reviewSubtasksWarning(
      formatNumber(saved.subtaskDoneCount),
      formatNumber(saved.subtaskCount),
    )
  }
  return t.moved(to.name)
}

function indexInColumn(tasks: readonly Task[], task: Task): number {
  return columnTasks(tasks, task.column.id).findIndex((x) => x.id === task.id)
}
