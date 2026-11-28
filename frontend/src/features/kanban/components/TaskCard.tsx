import { useDraggable, useDroppable } from '@dnd-kit/core'

import type { Task } from '@/features/tasks'
import { cn } from '@/shared/lib/cn'
import { formatHours, formatHoursValue } from '@/shared/lib/format'
import { getHoursLevel } from '@/shared/lib/hoursLevel'
import { messages } from '@/shared/lib/messages'
import { Avatar, barClass, HoursLevelMark, LabelChip } from '@/shared/ui'

import { taskDndId } from '../model/dnd'

const t = messages.kanban

export interface CardDnd {
  task: Task
  draggable: boolean
  onOpen: (task: Task) => void
}

/**
 * Thẻ trên bảng. Vừa kéo được (nếu có quyền) vừa là đích thả: thả lên thẻ khác = chèn vào trước thẻ đó.
 * Khi đang kéo, chỗ cũ hiện khung nét đứt cùng chiều cao để cột không bị nhảy.
 */
export function DraggableTaskCard({ task, draggable, onOpen }: CardDnd) {
  const id = taskDndId(task.id)
  const drag = useDraggable({ id, data: { taskId: task.id }, disabled: !draggable })
  const drop = useDroppable({ id, data: { columnId: task.column.id, taskId: task.id } })

  const setRef = (el: HTMLElement | null) => {
    drag.setNodeRef(el)
    drop.setNodeRef(el)
  }

  if (drag.isDragging) {
    return (
      // Vẽ lại thẻ ở dạng ẩn bên trong khung nét đứt → chỗ trống cao đúng bằng thẻ thật.
      <div
        ref={setRef}
        className="rounded-md border border-dashed border-line-strong bg-page"
        aria-hidden
      >
        <div className="invisible">
          <TaskCardView task={task} />
        </div>
      </div>
    )
  }
  return (
    <div
      ref={setRef}
      {...drag.listeners}
      {...drag.attributes}
      aria-roledescription={draggable ? t.cardRole : undefined}
      aria-label={draggable ? t.dragHandle(task.title) : task.title}
      onClick={() => {
        onOpen(task)
      }}
      onKeyDown={(e) => {
        // Enter mở chi tiết; Space để dnd-kit bắt đầu kéo bằng bàn phím.
        if (e.key === 'Enter') onOpen(task)
        else drag.listeners?.onKeyDown?.(e)
      }}
      className={cn(
        'rounded-md focus-visible:outline-2 focus-visible:outline-accent',
        draggable ? 'cursor-grab' : 'cursor-pointer',
      )}
    >
      <TaskCardView task={task} />
    </div>
  )
}

/** Phần hiển thị thuần của thẻ — dùng lại cho bản nổi (DragOverlay) khi đang kéo. */
export function TaskCardView({ task, lifted = false }: { task: Task; lifted?: boolean }) {
  // Giờ thực tế = giờ log trên chính việc + giờ log trên các việc con (server tính sẵn, R07).
  const actualHours = task.actualMinutes / 60
  const hasEstimate = task.estimatedMinutes !== null
  const estHours = (task.estimatedMinutes ?? 0) / 60
  const level = getHoursLevel(actualHours, estHours)
  const fill = estHours > 0 ? Math.min(100, (actualHours / estHours) * 100) : 0

  return (
    <article
      className={cn(
        'flex flex-col gap-1.5 rounded-md border border-line bg-surface p-2.5',
        lifted && 'border-accent',
      )}
    >
      <div className="flex items-start gap-2">
        {/* Một dòng để mỗi thẻ cao như nhau và cột hiện đủ ≥ 6 thẻ ở 1366×768 (§6). */}
        <h3
          title={task.title}
          className="line-clamp-1 flex-1 text-base leading-tight font-semibold text-ink"
        >
          {task.title}
        </h3>
        {task.storyPoint !== null && (
          <span className="num shrink-0 rounded-sm bg-subtle px-1.5 py-px text-sm font-semibold text-ink-muted">
            {t.sp(task.storyPoint)}
          </span>
        )}
      </div>
      <div className="h-1 overflow-hidden rounded-full bg-line" aria-hidden>
        <div
          data-testid="hours-fill"
          className={cn('h-full', barClass[level])}
          style={{ width: `${String(fill)}%` }}
        />
      </div>
      <div className="flex min-w-0 items-center gap-1.5 text-sm text-ink-muted">
        {hasEstimate ? (
          <span className="num inline-flex shrink-0 items-center gap-1">
            <HoursLevelMark level={level} labelled />
            {t.hours(formatHoursValue(actualHours), formatHoursValue(estHours))}
          </span>
        ) : (
          // Việc con (và việc cha chưa ước lượng) không có định mức → chỉ hiện giờ đã log, không xếp mức.
          actualHours > 0 && <span className="num shrink-0">{formatHours(actualHours)}</span>
        )}
        {task.subtaskCount > 0 && (
          <span className="num inline-flex shrink-0 gap-1.5">
            <span aria-hidden>·</span>
            <span>{t.subtasks(task.subtaskDoneCount, task.subtaskCount)}</span>
          </span>
        )}
        <span className="flex min-w-0 flex-1 gap-1 overflow-hidden">
          {task.labels.slice(0, 2).map((l) => (
            <LabelChip key={l.id} name={l.name} />
          ))}
        </span>
        {task.assignee ? (
          <Avatar name={task.assignee.name} size={20} />
        ) : (
          <span className="shrink-0 text-ink-faint">{t.unassigned}</span>
        )}
      </div>
    </article>
  )
}
