import { useMemo, useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, ErrorState, FormAlert, Skeleton, useToast } from '@/shared/ui'
import { useBoard, useReplaceTransitions } from '../api/queries'
import {
  TRANSITION_ROLES,
  type BoardColumn,
  type Transition,
  type TransitionRole,
} from '../model/types'

const t = messages.projects

const key = (from: string, to: string, role: TransitionRole) => `${role}|${from}|${to}`

function toSet(list: readonly Transition[]): Set<string> {
  return new Set(list.map((x) => key(x.fromColumnId, x.toColumnId, x.role)))
}

/** Member không được chuyển vào/ra nhóm done (R: chỉ Manager chuyển sang hoàn thành). */
function lockedForMember(from: BoardColumn, to: BoardColumn): boolean {
  return from.statusGroup === 'done' || to.statusGroup === 'done'
}

/** Ma trận luật chuyển theo vai trò. Chỉ lưu cấu hình; áp dụng khi kéo thẻ thuộc feature/kanban. */
export function TransitionsPanel({
  projectId,
  canManage,
}: {
  projectId: string
  canManage: boolean
}) {
  const board = useBoard(projectId)
  if (board.isError && !board.data)
    return (
      <ErrorState
        title={t.columns.loadError}
        errorRef={board.error instanceof ApiError ? board.error.code : undefined}
        onRetry={() => void board.refetch()}
      />
    )
  if (!board.data)
    return (
      <div className="flex flex-col gap-2" aria-busy="true">
        <Skeleton className="h-6 w-96" />
        <Skeleton className="h-48 w-full" />
      </div>
    )
  // key theo dữ liệu server: lưu xong (setQueryData) thì nháp được khởi tạo lại.
  const version =
    board.data.columns.map((c) => c.id).join() +
    '#' +
    [...toSet(board.data.transitions)].sort().join()
  return (
    <Editor
      key={version}
      projectId={projectId}
      columns={board.data.columns}
      transitions={board.data.transitions}
      canManage={canManage}
    />
  )
}

function Editor({
  projectId,
  columns,
  transitions,
  canManage,
}: {
  projectId: string
  columns: readonly BoardColumn[]
  transitions: readonly Transition[]
  canManage: boolean
}) {
  const initial = useMemo(() => toSet(transitions), [transitions])
  const [draft, setDraft] = useState<Set<string>>(initial)
  const [error, setError] = useState<string | null>(null)
  const save = useReplaceTransitions(projectId)
  const toast = useToast()

  const dirty = draft.size !== initial.size || [...draft].some((k) => !initial.has(k))

  const toggle = (k: string) => {
    setDraft((cur) => {
      const next = new Set(cur)
      if (next.has(k)) next.delete(k)
      else next.add(k)
      return next
    })
  }

  const submit = async () => {
    setError(null)
    const body: Transition[] = []
    for (const role of TRANSITION_ROLES)
      for (const from of columns)
        for (const to of columns)
          if (from.id !== to.id && draft.has(key(from.id, to.id, role)))
            body.push({ fromColumnId: from.id, toColumnId: to.id, role })
    try {
      await save.mutateAsync(body)
      toast.show(t.transitions.saved)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-ink-muted">{t.transitions.intro}</p>
      <FormAlert message={error} />
      {TRANSITION_ROLES.map((role) => {
        const roleLabel = t.transitionRoles[role]
        const count = [...draft].filter((k) => k.startsWith(`${role}|`)).length
        return (
          <section key={role} className="flex flex-col gap-2">
            <h2 className="flex items-baseline gap-2 text-base font-semibold text-ink">
              {roleLabel}
              <span className="num text-sm font-normal text-ink-muted">
                {t.transitions.count(formatNumber(count))}
              </span>
            </h2>
            <div className="overflow-x-auto rounded-md border border-line bg-surface">
              <table
                aria-label={t.transitions.matrixLabel(roleLabel)}
                className="w-full border-collapse text-base"
              >
                <thead>
                  <tr>
                    <th
                      scope="col"
                      className="px-3 py-2 text-left text-sm font-semibold text-ink-faint"
                    >
                      {t.transitions.from}
                    </th>
                    {columns.map((c) => (
                      <th
                        key={c.id}
                        scope="col"
                        className="px-3 py-2 text-center text-sm font-semibold text-ink-muted"
                      >
                        {c.name}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {columns.map((from) => (
                    <tr key={from.id} className="border-t border-line">
                      <th scope="row" className="px-3 py-2 text-left font-semibold text-ink">
                        {from.name}
                      </th>
                      {columns.map((to) => {
                        if (from.id === to.id)
                          return (
                            <td
                              key={to.id}
                              className="bg-subtle px-3 py-2 text-center text-ink-faint"
                            >
                              —
                            </td>
                          )
                        const k = key(from.id, to.id, role)
                        const locked = role === 'member' && lockedForMember(from, to)
                        return (
                          <td key={to.id} className="px-3 py-2 text-center">
                            <input
                              type="checkbox"
                              aria-label={t.transitions.cell(from.name, to.name)}
                              title={locked ? t.transitions.memberDoneLocked : undefined}
                              checked={!locked && draft.has(k)}
                              disabled={!canManage || locked}
                              onChange={() => {
                                toggle(k)
                              }}
                              className="size-4 accent-accent"
                            />
                          </td>
                        )
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {role === 'member' && (
              <p className="text-sm text-ink-muted">{t.transitions.memberDoneLocked}</p>
            )}
          </section>
        )
      })}
      {canManage && (
        <div className="flex items-center justify-end gap-2">
          {dirty && <span className="text-sm text-warn">{t.transitions.unsaved}</span>}
          <Button
            variant="secondary"
            disabled={!dirty || save.isPending}
            onClick={() => {
              setDraft(new Set(initial))
              setError(null)
            }}
          >
            {t.transitions.reset}
          </Button>
          <Button disabled={!dirty} loading={save.isPending} onClick={() => void submit()}>
            {t.transitions.save}
          </Button>
        </div>
      )}
    </div>
  )
}
