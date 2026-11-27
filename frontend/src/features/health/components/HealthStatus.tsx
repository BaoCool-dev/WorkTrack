import { useHealth } from '../api/health'
import { messages } from '@/shared/lib/messages'

const t = messages.health

export function HealthStatus() {
  const { isPending, isError, error, refetch, isFetching } = useHealth()

  if (isPending) {
    return (
      <p role="status" className="text-ink-muted">
        {t.checking}
      </p>
    )
  }

  if (isError) {
    return (
      <div
        role="alert"
        className="flex items-center gap-3 rounded-md border border-bad-border bg-bad-bg px-3 py-2 text-bad"
      >
        <span className="font-semibold">{t.error}</span>
        <span className="text-sm">{error.message}</span>
        <button
          type="button"
          onClick={() => void refetch()}
          disabled={isFetching}
          className="ml-auto h-10 rounded-md border border-line-strong bg-surface px-3 font-semibold text-ink disabled:text-disabled"
        >
          {t.retry}
        </button>
      </div>
    )
  }

  return (
    <p
      role="status"
      className="inline-flex rounded-sm border border-ok-border bg-ok-bg px-2 py-1 text-sm font-semibold text-ok"
    >
      {t.ok}
    </p>
  )
}
