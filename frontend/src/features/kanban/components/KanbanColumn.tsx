import { useDroppable } from '@dnd-kit/core'
import { Lock, Plus } from 'lucide-react'
import type { ReactNode } from 'react'

import type { BoardColumn } from '@/features/projects'
import { cn } from '@/shared/lib/cn'
import { messages } from '@/shared/lib/messages'

import { columnDndId, type DropState } from '../model/dnd'

const t = messages.kanban

interface Props {
  column: BoardColumn
  count: number
  dropState: DropState
  /** Chỉ truyền khi người xem là Manager phụ trách — Member không thấy nút "+". */
  onAdd?: () => void
  children: ReactNode
}

/**
 * Một cột Kanban. Khi đang kéo, cột tự nói rõ có nhận thẻ hay không và VÌ SAO không
 * (design-system §6: "không bao giờ chặn im lặng").
 */
export function KanbanColumn({ column, count, dropState, onAdd, children }: Props) {
  const { setNodeRef } = useDroppable({ id: columnDndId(column.id), data: { columnId: column.id } })
  const blocked = dropState !== null && !dropState.ok
  const valid = dropState?.ok === true

  return (
    <section
      ref={setNodeRef}
      aria-label={column.name}
      data-drop={valid ? 'valid' : blocked ? 'blocked' : undefined}
      className={cn(
        'flex max-h-full w-[min(260px,100%)] min-w-50 shrink-0 flex-col gap-1.5 rounded-md border bg-subtle p-2',
        valid && 'border-2 border-accent bg-accent-subtle p-[7px]',
        blocked && 'border-dashed border-line-strong opacity-60',
        !valid && !blocked && 'border-line',
      )}
    >
      <header className="flex items-center gap-1.5 px-0.5">
        <h2 className="truncate text-base font-semibold text-ink">{column.name}</h2>
        <span className="num text-sm text-ink-faint" aria-label={t.columnCount(count)}>
          {count}
        </span>
        {onAdd && (
          <button
            type="button"
            onClick={onAdd}
            aria-label={t.addTask(column.name)}
            title={t.addTask(column.name)}
            className="ml-auto inline-flex size-5 items-center justify-center rounded-sm text-ink-muted hover:bg-surface hover:text-ink focus-visible:outline-2 focus-visible:outline-accent"
          >
            <Plus size={14} strokeWidth={2} aria-hidden />
          </button>
        )}
      </header>

      {valid && <p className="px-0.5 text-sm font-semibold text-accent-hover">{t.dropHere}</p>}
      {blocked && (
        <p role="note" className="flex items-start gap-1 px-0.5 text-sm text-ink-muted">
          <Lock size={14} strokeWidth={2} className="mt-px shrink-0" aria-hidden />
          <span>{dropState.reason}</span>
        </p>
      )}

      <div className="flex min-h-13.5 flex-col gap-1.5 overflow-y-auto">{children}</div>
    </section>
  )
}
