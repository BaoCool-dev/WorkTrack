import { useState } from 'react'

import { useSetWipLimit, WIP_LIMIT_MAX, WIP_LIMIT_MIN, type Project } from '@/features/projects'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { useToast } from '@/shared/ui'

const t = messages.kanban.wip

/** Ô đổi giới hạn WIP — chỉ hiện cho Manager phụ trách (server vẫn kiểm lại quyền). */
export function WipLimitControl({ project }: { project: Project }) {
  const [value, setValue] = useState(String(project.wipLimit))
  const [error, setError] = useState<string | null>(null)
  const save = useSetWipLimit(project.id)
  const toast = useToast()

  const submit = () => {
    const n = Number(value)
    if (!Number.isInteger(n) || n < WIP_LIMIT_MIN || n > WIP_LIMIT_MAX) {
      setError(t.limitInvalid(WIP_LIMIT_MIN, WIP_LIMIT_MAX))
      return
    }
    setError(null)
    if (n === project.wipLimit) return
    save.mutate(n, {
      onSuccess: () => {
        toast.show(t.saved)
      },
      onError: (e) => {
        setError(e instanceof ApiError ? e.message : messages.errors.network)
      },
    })
  }

  return (
    <form
      className="flex items-center gap-1.5"
      onSubmit={(e) => {
        e.preventDefault()
        submit()
      }}
    >
      <label htmlFor="wip-limit" className="text-sm font-semibold text-ink-muted">
        {t.limitLabel}
      </label>
      <input
        id="wip-limit"
        type="number"
        inputMode="numeric"
        min={WIP_LIMIT_MIN}
        max={WIP_LIMIT_MAX}
        value={value}
        onChange={(e) => {
          setValue(e.target.value)
        }}
        aria-invalid={error ? true : false}
        aria-describedby="wip-limit-note"
        title={t.limitHint(WIP_LIMIT_MIN, WIP_LIMIT_MAX)}
        className="num h-8 w-14 rounded-md border border-line-strong bg-surface px-2 text-base text-ink focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none aria-invalid:border-bad"
      />
      <button
        type="submit"
        disabled={save.isPending}
        className="h-8 rounded-md border border-line-strong bg-surface px-2.5 text-sm font-semibold text-ink hover:bg-subtle disabled:text-disabled"
      >
        {t.save}
      </button>
      <span id="wip-limit-note" role={error ? 'alert' : undefined} className="text-sm text-bad">
        {error}
      </span>
    </form>
  )
}
