import { MessageSquare } from 'lucide-react'
import { useId, useState } from 'react'
import { useAuth } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { cn } from '@/shared/lib/cn'
import { formatNumber, formatTimestampDate } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Avatar,
  Button,
  EmptyState,
  ErrorState,
  FormAlert,
  Pagination,
  Skeleton,
  useToast,
} from '@/shared/ui'
import { useComments, useCreateComment } from '../api/queries'
import { COMMENT_MAX, COMMENT_PAGE_SIZE } from '../model/types'

const t = messages.tasks.comments

/**
 * Bình luận gắn công việc (CF-07), cũ nhất trước. Manager và Member viết được;
 * CEO/Admin chỉ xem. Server kiểm lại quyền theo dự án và gửi thông báo task_commented.
 */
export function CommentsSection({ taskId }: { taskId: string }) {
  const { user } = useAuth()
  const [page, setPage] = useState(1)
  const query = useComments(taskId, page)
  const canWrite = user?.role === 'manager' || user?.role === 'member'

  let content
  if (query.isError && !query.data)
    content = (
      <ErrorState
        title={t.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  else if (!query.data) content = <Skeleton className="h-16 w-full" />
  else if (query.data.items.length === 0)
    content = <EmptyState icon={MessageSquare} title={t.emptyTitle} description={t.emptyHint} />
  else
    content = (
      <>
        <ul aria-label={t.listLabel} className="flex flex-col gap-3">
          {query.data.items.map((c) => (
            <li key={c.id} className="flex gap-2">
              <Avatar name={c.author.name} size={24} />
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <p className="text-sm">
                  <span className="font-semibold text-ink">{c.author.name}</span>{' '}
                  <span className="num text-ink-faint">{formatTimestampDate(c.createdAt)}</span>
                </p>
                <p className="text-base whitespace-pre-wrap text-ink">{c.body}</p>
              </div>
            </li>
          ))}
        </ul>
        {query.data.total > COMMENT_PAGE_SIZE && (
          <Pagination
            page={page}
            pageSize={COMMENT_PAGE_SIZE}
            total={query.data.total}
            onChange={setPage}
          />
        )}
      </>
    )

  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-sm font-semibold text-ink-faint">{t.title}</h3>
      {content}
      {canWrite ? (
        <CommentForm taskId={taskId} />
      ) : (
        <p className="text-sm text-ink-faint">{t.readOnly}</p>
      )}
    </section>
  )
}

function CommentForm({ taskId }: { taskId: string }) {
  const create = useCreateComment(taskId)
  const toast = useToast()
  const id = useId()
  const [body, setBody] = useState('')
  const [fieldError, setFieldError] = useState<string>()
  const [serverError, setServerError] = useState<string | null>(null)

  const submit = async () => {
    const value = body.trim()
    if (value === '') {
      setFieldError(t.required)
      return
    }
    // Đếm theo ký tự Unicode như backend (utf8.RuneCountInString), không theo UTF-16.
    if (Array.from(value).length > COMMENT_MAX) {
      setFieldError(t.tooLong)
      return
    }
    setFieldError(undefined)
    setServerError(null)
    try {
      await create.mutateAsync(value)
      setBody('')
      toast.show(t.sent)
    } catch (err) {
      setServerError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  return (
    <form
      noValidate
      onSubmit={(e) => {
        e.preventDefault()
        void submit()
      }}
      className="flex flex-col gap-1"
    >
      <FormAlert message={serverError} />
      <label htmlFor={id} className="text-sm font-semibold text-ink-muted">
        {t.label}
      </label>
      <textarea
        id={id}
        rows={3}
        value={body}
        placeholder={t.placeholder}
        aria-invalid={fieldError ? true : false}
        aria-describedby={`${id}-note`}
        onChange={(e) => {
          setBody(e.target.value)
        }}
        className={cn(
          'rounded-md border bg-surface px-3 py-2 text-base text-ink placeholder:text-ink-faint',
          'focus:ring-1 focus:outline-none',
          fieldError
            ? 'border-bad focus:border-bad focus:ring-bad'
            : 'border-line-strong focus:border-accent focus:ring-accent',
        )}
      />
      <div id={`${id}-note`} className="flex items-center gap-2 text-sm">
        {fieldError && <span className="text-bad">{fieldError}</span>}
        <span className="num ml-auto text-ink-faint">
          {t.counter(formatNumber(Array.from(body).length))}
        </span>
      </div>
      <div className="flex justify-end">
        <Button type="submit" loading={create.isPending}>
          {t.send}
        </Button>
      </div>
    </form>
  )
}
