import { useId, useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { cn } from '@/shared/lib/cn'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, useToast } from '@/shared/ui'
import { useRejectTask } from '../api/queries'
import { REVIEW_REASON_MAX, type Task } from '../model/types'

const t = messages.tasks.review

export interface RejectReviewDialogProps {
  task: Task
  onClose: () => void
}

/** Kiểm tra sớm ở client; server kiểm lại (NormalizeReviewReason: cắt khoảng trắng, 1–500 ký tự). */
function validateReason(reason: string): string | undefined {
  const trimmed = reason.trim()
  if (trimmed === '') return t.reasonRequired
  if (Array.from(trimmed).length > REVIEW_REASON_MAX) return t.reasonTooLong
  return undefined
}

/**
 * Hộp Từ chối nghiệm thu (CF-09). Lý do bắt buộc. Không có checkbox "trả thẻ về Đang làm" như artboard 13
 * vì server luôn trả việc về cột đang làm đầu tiên (decisions.md mục C).
 */
export function RejectReviewDialog({ task, onClose }: RejectReviewDialogProps) {
  const reject = useRejectTask()
  const toast = useToast()
  const formId = useId()
  const reasonId = useId()
  const [reason, setReason] = useState('')
  const [error, setError] = useState<string | undefined>()
  const [serverError, setServerError] = useState<string | null>(null)

  const submit = async () => {
    const invalid = validateReason(reason)
    setError(invalid)
    if (invalid) return
    setServerError(null)
    try {
      await reject.mutateAsync({ task, reason: reason.trim() })
      toast.show(t.rejected)
      onClose()
    } catch (err) {
      setServerError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  const body = (
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
      <div className="flex flex-col gap-1">
        <label htmlFor={reasonId} className="text-sm font-semibold text-ink-muted">
          {t.reason}
        </label>
        <textarea
          id={reasonId}
          rows={4}
          value={reason}
          placeholder={t.reasonPlaceholder}
          aria-required
          aria-invalid={error ? true : false}
          aria-describedby={`${reasonId}-hint`}
          onChange={(e) => {
            setReason(e.target.value)
          }}
          className={cn(
            'rounded-md border bg-surface px-3 py-2 text-base text-ink placeholder:text-ink-faint',
            'focus:ring-1 focus:outline-none',
            error
              ? 'border-bad focus:border-bad focus:ring-bad'
              : 'border-line-strong focus:border-accent focus:ring-accent',
          )}
        />
        <div id={`${reasonId}-hint`} className="flex items-center gap-2 text-sm">
          {error && <span className="text-bad">{error}</span>}
          <span className="num ml-auto text-ink-faint">
            {formatNumber(Array.from(reason).length)}/{formatNumber(REVIEW_REASON_MAX)}
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
        {t.cancel}
      </Button>
      <Button type="submit" form={formId} variant="bad" loading={reject.isPending}>
        {t.confirmReject}
      </Button>
    </Dialog>
  )
}
