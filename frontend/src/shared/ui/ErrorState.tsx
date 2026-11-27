import { TriangleAlert } from 'lucide-react'
import { messages } from '@/shared/lib/messages'
import { Button } from './Button'
import { EmptyState } from './EmptyState'

export interface ErrorStateProps {
  title: string
  description?: string
  /** Mã lỗi tham chiếu (code từ API) để người dùng báo lại. */
  errorRef?: string
  onRetry?: () => void
  onReport?: () => void
}

/** §7: như trạng thái rỗng nhưng viền bad-border, icon bad, mã lỗi + câu "dữ liệu có thể đã cũ". */
export function ErrorState({ title, description, errorRef, onRetry, onReport }: ErrorStateProps) {
  return (
    <EmptyState
      role="alert"
      tone="error"
      icon={TriangleAlert}
      title={title}
      description={
        <>
          {description && <p>{description}</p>}
          <p>{messages.common.staleData}</p>
          {errorRef && (
            <p className="num mt-1 text-ink-faint">
              {messages.common.errorRef}: {errorRef}
            </p>
          )}
        </>
      }
      actions={
        (onRetry ?? onReport) && (
          <>
            {onReport && (
              <Button variant="secondary" onClick={onReport}>
                {messages.common.reportError}
              </Button>
            )}
            {onRetry && <Button onClick={onRetry}>{messages.common.retry}</Button>}
          </>
        )
      }
    />
  )
}
