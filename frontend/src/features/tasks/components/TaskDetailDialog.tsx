import { SearchX } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useCan } from '@/features/auth'
import { statusGroupTone } from '@/features/projects'
import { ApiError } from '@/shared/api/http'
import {
  formatDate,
  formatMinutesAsHours,
  formatNumber,
  formatTimestampDate,
} from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Badge,
  Button,
  Dialog,
  EmptyState,
  ErrorState,
  FormAlert,
  LabelChip,
  Skeleton,
  useToast,
} from '@/shared/ui'
import { useClaimTask, useTask, useUpdateTask } from '../api/queries'
import type { Task } from '../model/types'
import { CommentsSection } from './CommentsSection'
import { SubtasksSection } from './SubtasksSection'
import { TaskFormDialog } from './TaskFormDialog'

const t = messages.tasks.detail
const tl = messages.tasks

export interface TaskDetailDialogProps {
  /** Dòng trong bảng — hiện ngay tiêu đề trong lúc tải chi tiết. */
  initial: Task
  /** Manager phụ trách dự án được sửa (server kiểm lại). */
  canEdit: boolean
  onClose: () => void
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-sm font-semibold text-ink-faint">{label}</dt>
      <dd className="text-base text-ink">{children}</dd>
    </div>
  )
}

/** Chi tiết công việc (CF-06) kèm việc con và bình luận (CF-07). Log time thuộc nhánh sau. */
export function TaskDetailDialog({ initial, canEdit, onClose }: TaskDetailDialogProps) {
  const query = useTask(initial.id)
  const claim = useClaimTask()
  const update = useUpdateTask(initial.id)
  const canClaimPerm = useCan('tasks.claim')
  const toast = useToast()
  const [editing, setEditing] = useState(false)
  const [claimError, setClaimError] = useState<string | null>(null)

  const task = query.data
  if (editing && task)
    return (
      <TaskFormDialog
        projectId={task.projectId}
        task={task}
        save={(body) => update.mutateAsync(body)}
        onClose={() => {
          setEditing(false)
        }}
      />
    )

  const notFound = query.error instanceof ApiError && query.error.code === 'TASK_NOT_FOUND'
  const claimable =
    canClaimPerm &&
    task !== undefined &&
    task.assignee === null &&
    task.column.statusGroup === 'todo'

  const onClaim = async () => {
    setClaimError(null)
    try {
      await claim.mutateAsync(initial.id)
      toast.show(t.claimed)
    } catch (err) {
      setClaimError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  let body: ReactNode
  if (notFound)
    body = <EmptyState icon={SearchX} title={t.notFoundTitle} description={t.notFoundHint} />
  else if (query.isError && !task)
    body = (
      <ErrorState
        title={t.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  else if (!task)
    body = (
      <div className="flex flex-col gap-3" aria-busy="true">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    )
  else
    body = (
      <div className="flex flex-col gap-3">
        <FormAlert message={claimError} />
        <dl className="grid grid-cols-3 gap-3 rounded-md border border-line bg-surface p-3">
          <Field label={t.project}>{task.projectName}</Field>
          <Field label={t.column}>
            <Badge tone={statusGroupTone[task.column.statusGroup]}>{task.column.name}</Badge>
          </Field>
          <Field label={t.assignee}>
            {task.assignee ? (
              task.assignee.name
            ) : (
              <span className="text-ink-faint">{t.unassigned}</span>
            )}
          </Field>
          <Field label={t.storyPoint}>
            <span className="num">
              {task.storyPoint === null ? tl.list.none : tl.sp(formatNumber(task.storyPoint))}
            </span>
          </Field>
          <Field label={t.estimate}>
            <span className="num">
              {task.estimatedMinutes === null
                ? tl.list.none
                : formatMinutesAsHours(task.estimatedMinutes)}
            </span>
          </Field>
          <Field label={t.dueDate}>
            <span className="num">{formatDate(task.dueDate) ?? tl.list.none}</span>
          </Field>
          <Field label={t.createdBy}>{task.createdBy.name}</Field>
          <Field label={t.createdAt}>
            <span className="num">{formatTimestampDate(task.createdAt)}</span>
          </Field>
          <Field label={t.labels}>
            {task.labels.length === 0 ? (
              <span className="text-ink-faint">{tl.list.none}</span>
            ) : (
              <span className="flex flex-wrap gap-1">
                {task.labels.map((l) => (
                  <LabelChip key={l.id} name={l.name} />
                ))}
              </span>
            )}
          </Field>
        </dl>
        <section className="flex flex-col gap-1">
          <h3 className="text-sm font-semibold text-ink-faint">{t.description}</h3>
          {task.description ? (
            <p className="text-base whitespace-pre-wrap text-ink">{task.description}</p>
          ) : (
            <p className="text-base text-ink-faint">{t.noDescription}</p>
          )}
        </section>
        {/* Con chỉ 1 cấp: chi tiết của một việc con không có khối việc con. */}
        {task.parentId === null && <SubtasksSection task={task} canEdit={canEdit} />}
        <CommentsSection taskId={task.id} />
        <p className="rounded-md border border-line bg-subtle p-3 text-sm text-ink-muted">
          {t.later}
        </p>
      </div>
    )

  return (
    <Dialog open onClose={onClose} size="lg" title={task?.title ?? initial.title} body={body}>
      <Button variant="secondary" onClick={onClose}>
        {t.close}
      </Button>
      {canEdit && task && (
        <Button
          variant="secondary"
          onClick={() => {
            setEditing(true)
          }}
        >
          {t.edit}
        </Button>
      )}
      {claimable && (
        <Button loading={claim.isPending} onClick={() => void onClaim()}>
          {t.claim}
        </Button>
      )}
    </Dialog>
  )
}
