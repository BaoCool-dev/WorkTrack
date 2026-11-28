import { ApiError } from '@/shared/api/http'
import { formatTimestampDate } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Badge, ErrorState, Skeleton } from '@/shared/ui'
import { useTaskReviews } from '../api/queries'

const t = messages.tasks.review

/** Lịch sử nghiệm thu của công việc cha (CF-09): ai duyệt / từ chối, khi nào, lý do. */
export function ReviewSection({ taskId }: { taskId: string }) {
  const query = useTaskReviews(taskId)

  let content
  if (query.isPending) content = <Skeleton className="h-10 w-full" />
  else if (query.isError)
    content = (
      <ErrorState
        title={t.historyError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  else if (query.data.length === 0)
    content = <p className="text-sm text-ink-faint">{t.historyEmpty}</p>
  else
    content = (
      <ul className="flex flex-col divide-y divide-line rounded-md border border-line bg-surface">
        {query.data.map((r) => (
          <li key={r.id} className="flex flex-col gap-1 px-3 py-2">
            <div className="flex items-center gap-2">
              <Badge tone={r.decision === 'approved' ? 'ok' : 'bad'}>
                {r.decision === 'approved' ? t.decisionApproved : t.decisionRejected}
              </Badge>
              <span className="num text-sm text-ink-faint">
                {t.by(r.reviewerName, formatTimestampDate(r.createdAt))}
              </span>
            </div>
            {r.reason && <p className="text-base whitespace-pre-wrap text-ink">{r.reason}</p>}
          </li>
        ))}
      </ul>
    )

  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-sm font-semibold text-ink-faint">{t.history}</h3>
      {content}
    </section>
  )
}
