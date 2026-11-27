import { ListChecks, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useAuth } from '@/features/auth'
import { statusGroupTone, useBoard } from '@/features/projects'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Avatar,
  Badge,
  Button,
  EmptyState,
  ErrorState,
  FormAlert,
  Input,
  Select,
  Skeleton,
  useToast,
} from '@/shared/ui'
import { useCreateSubtask, useDeleteSubtask, useMoveSubtask, useSubtasks } from '../api/queries'
import type { Task } from '../model/types'

const t = messages.tasks.subtasks
const TITLE_MAX = 200

function errorText(err: unknown): string {
  return err instanceof ApiError ? err.message : messages.errors.unknown
}

/**
 * Công việc con của một việc cha (CF-07). Con chỉ 1 cấp, không Story Point.
 * Quyền ở đây chỉ để ẩn nút — server (CheckSubtaskActor) mới là nơi chặn thật:
 * - thêm con: Manager phụ trách dự án hoặc người nhận việc cha;
 * - chuyển/xóa con: Manager hoặc người nhận chính việc con đó.
 */
export function SubtasksSection({ task, canEdit }: { task: Task; canEdit: boolean }) {
  const { user } = useAuth()
  const query = useSubtasks(task.id)
  const board = useBoard(task.projectId)
  const canAdd = canEdit || (user !== null && task.assignee?.id === user.id)
  const [adding, setAdding] = useState(false)

  const subs = query.data ?? []
  const done = subs.filter((s) => s.column.statusGroup === 'done').length
  const columns = (board.data?.columns ?? []).map((c) => ({ value: c.id, label: c.name }))

  let content
  if (query.isError)
    content = (
      <ErrorState
        title={t.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  else if (query.isPending) content = <Skeleton className="h-16 w-full" />
  else if (subs.length === 0 && !adding)
    content = <EmptyState icon={ListChecks} title={t.emptyTitle} description={t.emptyHint} />
  else
    content = (
      <ul
        aria-label={t.listLabel}
        className="flex flex-col divide-y divide-line rounded-md border border-line"
      >
        {subs.map((s) => (
          <SubtaskRow
            key={s.id}
            parentId={task.id}
            sub={s}
            columns={columns}
            canChange={canEdit || (user !== null && s.assignee?.id === user.id)}
          />
        ))}
      </ul>
    )

  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <h3 className="text-sm font-semibold text-ink-faint">{t.title}</h3>
        {subs.length > 0 && (
          <span className="num text-sm text-ink-muted">
            {t.progress(formatNumber(done), formatNumber(subs.length))}
          </span>
        )}
        {canAdd && !adding && (
          <Button
            variant="secondary"
            className="ml-auto"
            onClick={() => {
              setAdding(true)
            }}
          >
            {t.add}
          </Button>
        )}
      </div>
      {content}
      {adding && (
        <AddSubtaskForm
          parentId={task.id}
          onDone={() => {
            setAdding(false)
          }}
        />
      )}
      {!canAdd && <p className="text-sm text-ink-faint">{t.readOnly}</p>}
    </section>
  )
}

function AddSubtaskForm({ parentId, onDone }: { parentId: string; onDone: () => void }) {
  const create = useCreateSubtask(parentId)
  const toast = useToast()
  const [title, setTitle] = useState('')
  const [fieldError, setFieldError] = useState<string>()
  const [serverError, setServerError] = useState<string | null>(null)

  const submit = async () => {
    const value = title.trim()
    if (value === '') {
      setFieldError(t.titleRequired)
      return
    }
    if (Array.from(value).length > TITLE_MAX) {
      setFieldError(t.titleTooLong)
      return
    }
    setFieldError(undefined)
    setServerError(null)
    try {
      await create.mutateAsync({ title: value, description: '' })
      toast.show(t.created)
      onDone()
    } catch (err) {
      setServerError(errorText(err))
    }
  }

  return (
    <form
      noValidate
      onSubmit={(e) => {
        e.preventDefault()
        void submit()
      }}
      className="flex flex-col gap-2 rounded-md border border-line bg-subtle p-3"
    >
      <FormAlert message={serverError} />
      <Input
        label={t.addTitle}
        placeholder={t.addPlaceholder}
        value={title}
        error={fieldError}
        autoFocus
        onChange={(e) => {
          setTitle(e.target.value)
        }}
      />
      <div className="flex justify-end gap-2">
        <Button type="button" variant="secondary" onClick={onDone}>
          {t.cancel}
        </Button>
        <Button type="submit" loading={create.isPending}>
          {t.save}
        </Button>
      </div>
    </form>
  )
}

interface RowProps {
  parentId: string
  sub: Task
  columns: { value: string; label: string }[]
  canChange: boolean
}

function SubtaskRow({ parentId, sub, columns, canChange }: RowProps) {
  const move = useMoveSubtask(parentId)
  const remove = useDeleteSubtask(parentId)
  const toast = useToast()
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Con đi thẳng tới cột bất kỳ, không qua luật workflow_transition (R của CF-07).
  const onMove = async (columnId: string) => {
    setError(null)
    try {
      await move.mutateAsync({ id: sub.id, columnId })
      toast.show(t.moved)
    } catch (err) {
      setError(errorText(err))
    }
  }

  const onRemove = async () => {
    setError(null)
    try {
      await remove.mutateAsync(sub.id)
      toast.show(t.removed)
    } catch (err) {
      setError(errorText(err))
      setConfirming(false)
    }
  }

  return (
    <li className="flex flex-col gap-2 px-3 py-2">
      <div className="flex items-center gap-3">
        <span className="min-w-0 flex-1 truncate text-base text-ink">{sub.title}</span>
        {sub.assignee && <Avatar name={sub.assignee.name} size={24} />}
        {canChange && columns.length > 0 ? (
          <div className="w-44">
            <Select
              label={t.column(sub.title)}
              hideLabel
              options={columns}
              value={sub.column.id}
              disabled={move.isPending}
              onChange={(e) => void onMove(e.target.value)}
            />
          </div>
        ) : (
          <Badge tone={statusGroupTone[sub.column.statusGroup]}>{sub.column.name}</Badge>
        )}
        {canChange && (
          <Button
            variant="ghost"
            aria-label={t.remove(sub.title)}
            onClick={() => {
              setConfirming(true)
            }}
          >
            <Trash2 size={16} aria-hidden />
          </Button>
        )}
      </div>
      <FormAlert message={error} />
      {confirming && (
        <div
          role="group"
          className="flex items-center gap-2 rounded-md border border-bad-border bg-bad-bg p-2"
        >
          <div className="flex-1">
            <p className="text-sm font-semibold text-ink">{t.removeTitle(sub.title)}</p>
            <p className="text-sm text-ink-muted">{t.removeHint}</p>
          </div>
          <Button
            variant="secondary"
            onClick={() => {
              setConfirming(false)
            }}
          >
            {t.cancel}
          </Button>
          <Button variant="danger" loading={remove.isPending} onClick={() => void onRemove()}>
            {t.removeConfirm}
          </Button>
        </div>
      )}
    </li>
  )
}
