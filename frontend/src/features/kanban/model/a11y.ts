import type { Announcements, ScreenReaderInstructions, UniqueIdentifier } from '@dnd-kit/core'

import type { BoardColumn } from '@/features/projects'
import type { Task } from '@/features/tasks'
import { messages } from '@/shared/lib/messages'

const t = messages.kanban.a11y

export const screenReaderInstructions: ScreenReaderInstructions = { draggable: t.instructions }

/**
 * Câu tiếng Việt cho vùng aria-live của dnd-kit (quy tắc 13: mọi chữ hiển thị/đọc lên là tiếng Việt).
 * Id kéo/thả có tiền tố `task:`/`col:` (xem dnd.ts) nên phải đổi id → tên thẻ, tên cột trước khi đọc.
 */
export function boardAnnouncements(tasks: Task[], columns: BoardColumn[]): Announcements {
  const taskById = new Map(tasks.map((x) => [x.id, x]))
  const colName = new Map(columns.map((c) => [c.id, c.name]))

  const cardTitle = (id: UniqueIdentifier) =>
    taskById.get(String(id).replace(/^task:/, ''))?.title ?? ''

  // Thả lên một thẻ khác nghĩa là thả vào cột của thẻ đó.
  const columnOf = (id: UniqueIdentifier | undefined): string | undefined => {
    if (id === undefined) return undefined
    const raw = String(id)
    if (raw.startsWith('col:')) return colName.get(raw.slice(4))
    const target = taskById.get(raw.replace(/^task:/, ''))
    return target ? colName.get(target.column.id) : undefined
  }

  return {
    onDragStart: ({ active }) => t.picked(cardTitle(active.id)),
    onDragOver: ({ active, over }) => {
      const col = columnOf(over?.id)
      return col ? t.over(cardTitle(active.id), col) : t.outside(cardTitle(active.id))
    },
    onDragEnd: ({ active, over }) => {
      const col = columnOf(over?.id)
      return col ? t.dropped(cardTitle(active.id), col) : t.droppedOutside(cardTitle(active.id))
    },
    onDragCancel: ({ active }) => t.cancelled(cardTitle(active.id)),
  }
}
