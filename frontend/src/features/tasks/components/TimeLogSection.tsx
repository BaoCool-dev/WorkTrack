import { Clock } from 'lucide-react'
import { useState } from 'react'
import { useAuth } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import {
  formatDate,
  formatHours,
  formatHoursValue,
  formatMinutesAsHours,
} from '@/shared/lib/format'
import { getHoursLevel } from '@/shared/lib/hoursLevel'
import { messages } from '@/shared/lib/messages'
import {
  Avatar,
  Button,
  EmptyState,
  ErrorState,
  HoursLevelMark,
  Pagination,
  Skeleton,
  useToast,
} from '@/shared/ui'
import { useDeleteTimeLog, useTimeLogs } from '../api/queries'
import { TIME_LOG_PAGE_SIZE, type Task, type TimeLog } from '../model/types'

const t = messages.tasks.timeLog

export interface TimeLogSectionProps {
  task: Task
  /** Người dùng được log giờ vào công việc này (server kiểm lại). */
  canLog: boolean
  onLog: () => void
}

/**
 * Giờ làm thực tế (CF-08): tổng giờ so với ước tính kèm ký hiệu mức (B5) và lịch sử log.
 * Giờ thực tế = giờ của chính việc + giờ của các việc con — server tính sẵn (actualMinutes).
 */
export function TimeLogSection({ task, canLog, onLog }: TimeLogSectionProps) {
  const actual = task.actualMinutes / 60
  const hasEstimate = task.estimatedMinutes !== null
  const est = (task.estimatedMinutes ?? 0) / 60

  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <h3 className="text-sm font-semibold text-ink-faint">{t.title}</h3>
        <span className="num inline-flex items-center gap-1 text-base font-semibold text-ink">
          {hasEstimate ? (
            <>
              <HoursLevelMark level={getHoursLevel(actual, est)} labelled />
              {t.summary(formatHoursValue(actual), formatHoursValue(est))}
            </>
          ) : (
            formatHours(actual)
          )}
        </span>
        {!hasEstimate && task.parentId === null && (
          <span className="text-sm text-ink-faint">{t.noEstimate}</span>
        )}
        {canLog && (
          <Button
            variant="secondary"
            className="ml-auto"
            icon={<Clock aria-hidden className="size-4" />}
            onClick={onLog}
          >
            {t.log}
          </Button>
        )}
      </div>
      {task.subtaskCount > 0 && <p className="text-sm text-ink-faint">{t.includesSubtasks}</p>}
      <TimeLogHistory task={task} canLog={canLog} />
    </section>
  )
}

function TimeLogHistory({ task, canLog }: { task: Task; canLog: boolean }) {
  const [page, setPage] = useState(1)
  const query = useTimeLogs(task.id, page)

  if (query.isError && !query.data)
    return (
      <ErrorState
        title={t.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  if (!query.data) return <Skeleton className="h-16 w-full" />
  if (query.data.items.length === 0)
    return (
      <EmptyState
        icon={Clock}
        title={t.emptyTitle}
        description={canLog ? t.emptyHint : t.emptyHintReadOnly}
      />
    )

  return (
    <>
      <h4 className="sr-only">{t.historyTitle}</h4>
      <ul
        aria-label={t.listLabel}
        className="flex flex-col divide-y divide-line rounded-md border border-line"
      >
        {query.data.items.map((log) => (
          <TimeLogRow key={log.id} task={task} log={log} />
        ))}
      </ul>
      {query.data.total > TIME_LOG_PAGE_SIZE && (
        <Pagination
          page={page}
          pageSize={TIME_LOG_PAGE_SIZE}
          total={query.data.total}
          onChange={setPage}
        />
      )}
    </>
  )
}

function TimeLogRow({ task, log }: { task: Task; log: TimeLog }) {
  const { user } = useAuth()
  const remove = useDeleteTimeLog(task)
  const toast = useToast()
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Chỉ người ghi được xóa log của mình (server kiểm lại — CheckTimeLogDelete).
  const mine = user?.id === log.user.id
  const date = formatDate(log.workDate) ?? log.workDate

  const onDelete = async () => {
    setError(null)
    try {
      await remove.mutateAsync(log)
      toast.show(t.deleted)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  return (
    <li className="flex gap-2 p-2.5">
      <Avatar name={log.user.name} size={24} />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <p className="flex items-center gap-2 text-sm">
          <span className="font-semibold text-ink">{log.user.name}</span>
          <span className="num text-ink-faint">{date}</span>
          <span className="num ml-auto font-semibold text-ink">
            {formatMinutesAsHours(log.minutes)}
          </span>
        </p>
        {log.taskId !== task.id && (
          <p className="text-sm text-ink-muted">{t.onSubtask(log.taskTitle)}</p>
        )}
        <p className="text-base whitespace-pre-wrap text-ink">{log.note}</p>
        {error && <p className="text-sm text-bad">{error}</p>}
        {mine &&
          (confirming ? (
            <div className="flex items-center gap-2">
              <span className="text-sm text-ink-muted">{t.confirmDelete}</span>
              <Button variant="danger" loading={remove.isPending} onClick={() => void onDelete()}>
                {t.confirm}
              </Button>
              <Button
                variant="ghost"
                onClick={() => {
                  setConfirming(false)
                }}
              >
                {t.cancel}
              </Button>
            </div>
          ) : (
            <div>
              <Button
                variant="ghost"
                aria-label={t.deleteLabel(log.user.name, date)}
                onClick={() => {
                  setConfirming(true)
                }}
              >
                {t.delete}
              </Button>
            </div>
          ))}
      </div>
    </li>
  )
}
