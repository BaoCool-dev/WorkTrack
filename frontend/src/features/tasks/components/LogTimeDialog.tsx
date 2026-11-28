import { Lock } from 'lucide-react'
import { useId, useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { cn } from '@/shared/lib/cn'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, Input, Select, useToast } from '@/shared/ui'
import { useCreateTimeLog, useSubtasks } from '../api/queries'
import type { Task } from '../model/types'
import { todayValue, validateTimeLog, type TimeLogFormErrors } from '../model/timeLogForm'

const t = messages.tasks.timeLog

export interface LogTimeDialogProps {
  /** Công việc đang mở; nếu là việc cha thì được chọn ghi vào một việc con. */
  task: Task
  /** Đóng hộp Log giờ, trở về chi tiết công việc. */
  onClose: () => void
}

/**
 * Hộp Log giờ (CF-08). Kiểm tra ở client chỉ để báo lỗi sớm — server kiểm lại toàn bộ
 * (phút 1..1440, ngày làm, quyền, việc đã hoàn thành quá 24 giờ).
 */
export function LogTimeDialog({ task, onClose }: LogTimeDialogProps) {
  const isParent = task.parentId === null
  const subtasks = useSubtasks(task.id, isParent)
  const create = useCreateTimeLog(task)
  const toast = useToast()
  const noteId = useId()
  const today = todayValue()

  const [targetId, setTargetId] = useState(task.id)
  const [workDate, setWorkDate] = useState(today)
  const [hours, setHours] = useState('')
  const [note, setNote] = useState('')
  const [errors, setErrors] = useState<TimeLogFormErrors>({})
  const [serverError, setServerError] = useState<string | null>(null)
  // Server trả TIME_LOG_CLOSED: việc đã Done quá 24 giờ → thay form bằng khối khóa (§7).
  const [closed, setClosed] = useState(false)

  const targets = [
    { value: task.id, label: task.title },
    ...(subtasks.data ?? []).map((s) => ({ value: s.id, label: t.targetSubtask(s.title) })),
  ]

  const submit = async () => {
    const result = validateTimeLog(hours, note, workDate, today)
    setErrors(result.errors)
    if (result.minutes === undefined || Object.keys(result.errors).length > 0) return
    setServerError(null)
    try {
      await create.mutateAsync({
        taskId: targetId,
        minutes: result.minutes,
        note: note.trim(),
        workDate,
      })
      toast.show(t.saved)
      onClose()
    } catch (err) {
      if (err instanceof ApiError && err.code === 'TIME_LOG_CLOSED') setClosed(true)
      else setServerError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  const formId = useId()
  const body = closed ? (
    <div className="flex gap-3 rounded-md border border-line bg-subtle p-3" role="status">
      <Lock aria-hidden className="mt-0.5 size-4 shrink-0 text-ink-muted" />
      <div className="flex flex-col gap-1">
        <p className="text-base font-semibold text-ink">{t.closedTitle}</p>
        <p className="text-sm text-ink-muted">{t.closedHint}</p>
      </div>
    </div>
  ) : (
    <form
      id={formId}
      noValidate
      onSubmit={(e) => {
        e.preventDefault()
        void submit()
      }}
      className="flex flex-col gap-3"
    >
      <FormAlert message={serverError} />
      {isParent && targets.length > 1 ? (
        <Select
          label={t.target}
          options={targets}
          value={targetId}
          onChange={(e) => {
            setTargetId(e.target.value)
          }}
        />
      ) : (
        <Select label={t.target} options={targets} value={targetId} disabled />
      )}
      <div className="grid grid-cols-2 gap-3">
        <Input
          label={t.workDate}
          type="date"
          max={today}
          value={workDate}
          error={errors.workDate}
          onChange={(e) => {
            setWorkDate(e.target.value)
          }}
        />
        <Input
          label={t.hours}
          inputMode="decimal"
          autoComplete="off"
          className="num"
          value={hours}
          error={errors.hours}
          hint={t.hoursHint}
          onChange={(e) => {
            setHours(e.target.value)
          }}
        />
      </div>
      <div className="flex flex-col gap-1">
        <label htmlFor={noteId} className="text-sm font-semibold text-ink-muted">
          {t.note}
        </label>
        <textarea
          id={noteId}
          rows={3}
          value={note}
          placeholder={t.notePlaceholder}
          aria-invalid={errors.note ? true : false}
          aria-describedby={`${noteId}-note`}
          onChange={(e) => {
            setNote(e.target.value)
          }}
          className={cn(
            'rounded-md border bg-surface px-3 py-2 text-base text-ink placeholder:text-ink-faint',
            'focus:ring-1 focus:outline-none',
            errors.note
              ? 'border-bad focus:border-bad focus:ring-bad'
              : 'border-line-strong focus:border-accent focus:ring-accent',
          )}
        />
        <div id={`${noteId}-note`} className="flex items-center gap-2 text-sm">
          {errors.note && <span className="text-bad">{errors.note}</span>}
          <span className="num ml-auto text-ink-faint">
            {t.counter(formatNumber(Array.from(note).length))}
          </span>
        </div>
      </div>
    </form>
  )

  return (
    <Dialog
      open
      onClose={onClose}
      size="lg"
      title={t.dialogTitle}
      description={t.dialogDescription}
      body={body}
    >
      <Button variant="secondary" onClick={onClose}>
        {/* Khi việc đã khóa thì không còn gì để hủy → nút là "Đóng". */}
        {closed ? messages.common.close : t.cancelDialog}
      </Button>
      {!closed && (
        <Button type="submit" form={formId} loading={create.isPending}>
          {t.submit}
        </Button>
      )}
    </Dialog>
  )
}
