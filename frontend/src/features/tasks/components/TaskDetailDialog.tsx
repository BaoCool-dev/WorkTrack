import { SearchX } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useAuth, useCan } from '@/features/auth'
import { statusGroupTone, useBoard } from '@/features/projects'
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
import { useApproveTask, useClaimTask, useTask, useUpdateTask } from '../api/queries'
import { hasPendingSubtasks, reviewColumnId } from '../model/review'
import type { Task } from '../model/types'
import { CommentsSection } from './CommentsSection'
import { LogTimeDialog } from './LogTimeDialog'
import { RejectReviewDialog } from './RejectReviewDialog'
import { ReviewSection } from './ReviewSection'
import { SubtasksSection } from './SubtasksSection'
import { TaskFormDialog } from './TaskFormDialog'
import { TimeLogSection } from './TimeLogSection'

const t = messages.tasks.detail
const tl = messages.tasks
const tr = messages.tasks.review

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

/**
 * Chi tiết công việc (CF-06) kèm việc con, bình luận (CF-07), giờ làm thực tế (CF-08)
 * và nghiệm thu (CF-09).
 */
export function TaskDetailDialog({ initial, canEdit, onClose }: TaskDetailDialogProps) {
  const query = useTask(initial.id)
  const claim = useClaimTask()
  const update = useUpdateTask(initial.id)
  const canClaimPerm = useCan('tasks.claim')
  const toast = useToast()
  const { user } = useAuth()
  const [editing, setEditing] = useState(false)
  const [logging, setLogging] = useState(false)
  const [claimError, setClaimError] = useState<string | null>(null)
  const [rejecting, setRejecting] = useState(false)
  const [reviewError, setReviewError] = useState<string | null>(null)
  const approve = useApproveTask()

  const task = query.data
  // Nút nghiệm thu chỉ cho Manager phụ trách (canEdit) và chỉ với việc cha đang ở cột chờ nghiệm thu.
  // Bảng chỉ tải khi có quyền; server vẫn kiểm lại (CanReview + ReviewTargets).
  const board = useBoard(initial.projectId, canEdit)
  const reviewColumn = board.data ? reviewColumnId(board.data.columns) : null
  const inReview =
    canEdit && task !== undefined && task.parentId === null && task.column.id === reviewColumn
  // Việc con được log bởi người phụ trách việc cha → cần tải việc cha (đã có trong cache khi mở từ cha).
  const parent = useTask(task?.parentId ?? '', Boolean(task?.parentId))
  const owner = task?.parentId ? parent.data?.assignee : task?.assignee
  // Chỉ để ẩn/hiện nút — server kiểm lại quyền log (CheckTimeLog).
  const canLog =
    (user?.role === 'manager' || user?.role === 'member') &&
    (canEdit || (owner !== undefined && owner !== null && owner.id === user.id))

  if (logging && task)
    return (
      <LogTimeDialog
        task={task}
        onClose={() => {
          setLogging(false)
        }}
      />
    )
  if (rejecting && task)
    return (
      <RejectReviewDialog
        task={task}
        onClose={() => {
          setRejecting(false)
        }}
      />
    )
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

  const onApprove = async (current: Task) => {
    setReviewError(null)
    try {
      await approve.mutateAsync(current)
      toast.show(tr.approved)
    } catch (err) {
      setReviewError(err instanceof ApiError ? err.message : messages.errors.unknown)
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
        <FormAlert message={reviewError} />
        {/* Nút nghiệm thu phụ thuộc cột của bảng: báo rõ khi bảng chưa tải xong hoặc lỗi, không ẩn im lặng. */}
        {canEdit && task.parentId === null && board.isPending && (
          <p role="status" className="text-sm text-ink-muted">
            {tr.boardLoading}
          </p>
        )}
        {canEdit && task.parentId === null && board.isError && (
          <p
            role="status"
            className="rounded-md border border-warn-border bg-warn-bg px-3 py-2 text-sm text-warn"
          >
            {tr.boardError}
          </p>
        )}
        {inReview && hasPendingSubtasks(task) && (
          <p
            role="status"
            className="rounded-md border border-warn-border bg-warn-bg px-3 py-2 text-sm text-warn"
          >
            {tr.subtasksPending(
              formatNumber(task.subtaskDoneCount),
              formatNumber(task.subtaskCount),
            )}
          </p>
        )}
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
        <TimeLogSection
          task={task}
          canLog={canLog}
          onLog={() => {
            setLogging(true)
          }}
        />
        {task.parentId === null && <ReviewSection taskId={task.id} />}
        <CommentsSection taskId={task.id} />
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
      {inReview && (
        <>
          <Button
            variant="danger"
            onClick={() => {
              setRejecting(true)
            }}
          >
            {tr.reject}
          </Button>
          <Button variant="ok" loading={approve.isPending} onClick={() => void onApprove(task)}>
            {tr.approve}
          </Button>
        </>
      )}
      {claimable && (
        <Button loading={claim.isPending} onClick={() => void onClaim()}>
          {t.claim}
        </Button>
      )}
    </Dialog>
  )
}
