import { ArrowDown, ArrowUp, Columns3, Plus } from 'lucide-react'
import { useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Badge,
  Button,
  cellClass,
  DataTable,
  EmptyState,
  ErrorState,
  FormAlert,
  TableSkeleton,
  useToast,
} from '@/shared/ui'
import {
  useAddColumn,
  useBoard,
  useDeleteColumn,
  useReorderColumns,
  useUpdateColumn,
} from '../api/queries'
import { statusGroupTone } from '../model/status'
import { MAX_COLUMNS, MIN_COLUMNS, type BoardColumn } from '../model/types'
import { ColumnFormDialog } from './ColumnFormDialog'
import { ConfirmDialog } from './MembersPanel'

const t = messages.projects

type Editing = { mode: 'create' } | { mode: 'edit'; column: BoardColumn }

/** Cấu hình 2–8 cột Kanban: thêm, đổi tên/nhóm, xóa, sắp xếp bằng nút lên/xuống. */
export function ColumnsPanel({ projectId, canManage }: { projectId: string; canManage: boolean }) {
  const board = useBoard(projectId)
  const add = useAddColumn(projectId)
  const update = useUpdateColumn(projectId)
  const del = useDeleteColumn(projectId)
  const reorder = useReorderColumns(projectId)
  const toast = useToast()
  const [editing, setEditing] = useState<Editing | null>(null)
  const [removing, setRemoving] = useState<BoardColumn | null>(null)
  const [reorderError, setReorderError] = useState<string | null>(null)

  if (board.isError && !board.data)
    return (
      <ErrorState
        title={t.columns.loadError}
        errorRef={board.error instanceof ApiError ? board.error.code : undefined}
        onRetry={() => void board.refetch()}
      />
    )
  if (!board.data) return <TableSkeleton rows={5} />

  const columns = board.data.columns
  const atMax = columns.length >= MAX_COLUMNS
  const atMin = columns.length <= MIN_COLUMNS

  const move = async (index: number, delta: -1 | 1) => {
    const a = columns[index]
    const b = columns[index + delta]
    if (!a || !b) return
    const ids = columns.map((c) => (c.id === a.id ? b.id : c.id === b.id ? a.id : c.id))
    setReorderError(null)
    try {
      await reorder.mutateAsync(ids)
      toast.show(t.columns.reordered)
    } catch (err) {
      setReorderError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <p className="text-sm text-ink-muted">
          {t.columns.limits(formatNumber(MIN_COLUMNS), formatNumber(MAX_COLUMNS))}
        </p>
        {canManage && (
          <div className="ml-auto flex items-center gap-2">
            {atMax && <span className="text-sm text-ink-muted">{t.columns.maxReached}</span>}
            <Button
              icon={<Plus size={16} strokeWidth={2} aria-hidden />}
              disabled={atMax}
              onClick={() => {
                setEditing({ mode: 'create' })
              }}
            >
              {t.columns.add}
            </Button>
          </div>
        )}
      </div>
      <FormAlert message={reorderError} />
      {columns.length === 0 ? (
        <EmptyState icon={Columns3} title={t.columns.loadError} />
      ) : (
        <DataTable
          label={t.columns.tableLabel}
          head={[
            { label: t.columns.colOrder },
            { label: t.columns.colName },
            { label: t.columns.colGroup },
            { label: t.columns.colTasks },
            ...(canManage ? [{ label: t.columns.colActions }] : []),
          ]}
        >
          {columns.map((c, i) => (
            <tr key={c.id}>
              <td className={cellClass}>
                <span className="num">{formatNumber(i + 1)}</span>
              </td>
              <td className={cellClass}>
                <span className="font-semibold text-ink">{c.name}</span>
              </td>
              <td className={cellClass}>
                <Badge tone={statusGroupTone[c.statusGroup]}>{t.groups[c.statusGroup]}</Badge>
              </td>
              <td className={cellClass}>
                <span className="num">{formatNumber(c.taskCount)}</span>
              </td>
              {canManage && (
                <td className={cellClass}>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      aria-label={t.columns.moveUp(c.name)}
                      title={t.columns.moveUp(c.name)}
                      disabled={i === 0 || reorder.isPending}
                      icon={<ArrowUp size={16} strokeWidth={2} aria-hidden />}
                      onClick={() => void move(i, -1)}
                    />
                    <Button
                      variant="ghost"
                      aria-label={t.columns.moveDown(c.name)}
                      title={t.columns.moveDown(c.name)}
                      disabled={i === columns.length - 1 || reorder.isPending}
                      icon={<ArrowDown size={16} strokeWidth={2} aria-hidden />}
                      onClick={() => void move(i, 1)}
                    />
                    <Button
                      variant="ghost"
                      onClick={() => {
                        setEditing({ mode: 'edit', column: c })
                      }}
                    >
                      {t.columns.edit}
                    </Button>
                    <Button
                      variant="ghost"
                      disabled={atMin}
                      title={atMin ? t.columns.minReached : undefined}
                      onClick={() => {
                        setRemoving(c)
                      }}
                    >
                      {t.columns.remove}
                    </Button>
                  </div>
                </td>
              )}
            </tr>
          ))}
        </DataTable>
      )}
      {canManage && atMin && <p className="text-sm text-ink-muted">{t.columns.minReached}</p>}
      {editing && (
        <ColumnFormDialog
          column={editing.mode === 'edit' ? editing.column : undefined}
          save={(v) =>
            editing.mode === 'edit'
              ? update.mutateAsync({ columnId: editing.column.id, ...v })
              : add.mutateAsync(v)
          }
          onClose={() => {
            setEditing(null)
          }}
        />
      )}
      {removing && (
        <ConfirmDialog
          title={t.columns.removeTitle(removing.name)}
          description={
            removing.taskCount > 0
              ? t.columns.removeHasTasks(formatNumber(removing.taskCount))
              : t.columns.removeHint
          }
          confirmLabel={t.columns.remove}
          doneMessage={t.columns.removed}
          onConfirm={() => del.mutateAsync(removing.id)}
          onClose={() => {
            setRemoving(null)
          }}
        />
      )}
    </div>
  )
}
