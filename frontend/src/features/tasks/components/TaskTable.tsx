import { statusGroupTone } from '@/features/projects'
import { formatDate, formatMinutesAsHours, formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Badge, cellClass, DataTable, LabelChip } from '@/shared/ui'
import type { Task } from '../model/types'

const t = messages.tasks

export interface TaskTableProps {
  tasks: readonly Task[]
  /** Hiện cột Dự án (trang "Công việc của tôi" gom nhiều dự án). */
  showProject?: boolean
  onOpen: (task: Task) => void
}

/** Bảng công việc cha (CF-06): dùng chung cho tab Công việc của dự án và "Công việc của tôi". */
export function TaskTable({ tasks, showProject = false, onOpen }: TaskTableProps) {
  return (
    <DataTable
      label={t.list.tableLabel}
      head={[
        { label: t.list.colTitle },
        ...(showProject ? [{ label: t.list.colProject }] : []),
        { label: t.list.colColumn },
        { label: t.list.colStoryPoint, numeric: true },
        { label: t.list.colEstimate, numeric: true },
        { label: t.list.colAssignee },
        { label: t.list.colDue },
        { label: t.list.colLabels },
      ]}
    >
      {tasks.map((task) => (
        <tr key={task.id} className="hover:bg-subtle">
          <td className={cellClass}>
            <button
              type="button"
              aria-label={t.list.open(task.title)}
              onClick={() => {
                onOpen(task)
              }}
              className="text-left font-semibold text-ink hover:text-accent hover:underline focus-visible:outline-2 focus-visible:outline-accent"
            >
              {task.title}
            </button>
          </td>
          {showProject && <td className={`${cellClass} text-ink-muted`}>{task.projectName}</td>}
          <td className={cellClass}>
            <Badge tone={statusGroupTone[task.column.statusGroup]}>{task.column.name}</Badge>
          </td>
          <td className={`${cellClass} num text-right`}>
            {task.storyPoint === null ? t.list.none : t.sp(formatNumber(task.storyPoint))}
          </td>
          <td className={`${cellClass} num text-right`}>
            {task.estimatedMinutes === null
              ? t.list.none
              : formatMinutesAsHours(task.estimatedMinutes)}
          </td>
          <td className={cellClass}>
            {task.assignee ? (
              task.assignee.name
            ) : (
              <span className="text-ink-faint">{t.detail.unassigned}</span>
            )}
          </td>
          <td className={`${cellClass} num text-ink-muted`}>
            {formatDate(task.dueDate) ?? t.list.none}
          </td>
          <td className={cellClass}>
            {task.labels.length === 0 ? (
              <span className="text-ink-faint">{t.list.none}</span>
            ) : (
              <span className="flex flex-wrap gap-1">
                {task.labels.map((l) => (
                  <LabelChip key={l.id} name={l.name} />
                ))}
              </span>
            )}
          </td>
        </tr>
      ))}
    </DataTable>
  )
}
