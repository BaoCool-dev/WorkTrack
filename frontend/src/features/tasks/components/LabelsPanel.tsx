import { Plus, Tags, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useCan } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Button,
  cellClass,
  DataTable,
  Dialog,
  EmptyState,
  ErrorState,
  FormAlert,
  TableSkeleton,
  useToast,
} from '@/shared/ui'
import { useDeleteLabel, useLabels, useSaveLabel } from '../api/queries'
import type { Label } from '../model/types'
import { LabelNameDialog } from './LabelNameDialog'

const t = messages.tasks.labels

type Editing = { mode: 'create' } | { mode: 'edit'; label: Label }

/** Tab "Nhãn" ở trang Quản trị (CF-06). Sửa cần quyền org.manage — server kiểm lại. */
export function LabelsPanel() {
  const canManage = useCan('org.manage')
  const query = useLabels()
  const save = useSaveLabel()
  const [editing, setEditing] = useState<Editing | null>(null)
  const [deleting, setDeleting] = useState<Label | null>(null)

  let content
  if (query.isPending) content = <TableSkeleton />
  else if (query.isError)
    content = (
      <ErrorState
        title={t.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  else if (query.data.length === 0)
    content = <EmptyState icon={Tags} title={t.emptyTitle} description={t.emptyHint} />
  else
    content = (
      <DataTable
        label={t.tableLabel}
        head={[
          { label: t.colName },
          { label: t.colTasks, numeric: true, className: 'w-32' },
          ...(canManage ? [{ label: t.colActions, className: 'w-48' }] : []),
        ]}
      >
        {query.data.map((label) => (
          <tr key={label.id}>
            <td className={cellClass}>{label.name}</td>
            <td className={`${cellClass} num text-right`}>{formatNumber(label.taskCount)}</td>
            {canManage && (
              <td className={cellClass}>
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setEditing({ mode: 'edit', label })
                    }}
                  >
                    {t.edit}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setDeleting(label)
                    }}
                  >
                    {t.remove}
                  </Button>
                </div>
              </td>
            )}
          </tr>
        ))}
      </DataTable>
    )

  return (
    <div className="flex flex-col gap-3">
      {canManage && (
        <div className="flex justify-end">
          <Button
            icon={<Plus size={16} strokeWidth={2} aria-hidden />}
            onClick={() => {
              setEditing({ mode: 'create' })
            }}
          >
            {t.add}
          </Button>
        </div>
      )}
      {content}
      {editing && (
        <LabelNameDialog
          key={editing.mode === 'edit' ? editing.label.id : 'new'}
          isEdit={editing.mode === 'edit'}
          defaultName={editing.mode === 'edit' ? editing.label.name : ''}
          save={(name) =>
            save.mutateAsync({
              id: editing.mode === 'edit' ? editing.label.id : undefined,
              name,
            })
          }
          onClose={() => {
            setEditing(null)
          }}
        />
      )}
      {deleting && (
        <DeleteLabelDialog
          label={deleting}
          onClose={() => {
            setDeleting(null)
          }}
        />
      )}
    </div>
  )
}

/** Xác nhận xóa; nhãn đang dùng thì khóa nút kèm lý do, lỗi LABEL_IN_USE từ server (dữ liệu cũ) hiện ngay trong hộp thoại. */
function DeleteLabelDialog({ label, onClose }: { label: Label; onClose: () => void }) {
  const remove = useDeleteLabel()
  const toast = useToast()
  const [error, setError] = useState<string | null>(null)
  const inUse = label.taskCount > 0
  const confirm = async () => {
    setError(null)
    try {
      await remove.mutateAsync(label.id)
      toast.show(t.removed)
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }
  return (
    <Dialog
      open
      onClose={onClose}
      title={t.removeTitle(label.name)}
      description={inUse ? t.removeInUse(formatNumber(label.taskCount)) : t.removeHint}
      icon={<Trash2 size={16} strokeWidth={2} aria-hidden className="text-bad" />}
      body={
        error ? (
          <FormAlert message={error} />
        ) : inUse ? (
          <p className="text-sm text-ink-muted">{t.removeBlocked}</p>
        ) : undefined
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button
        variant="danger"
        disabled={inUse}
        loading={remove.isPending}
        onClick={() => void confirm()}
      >
        {t.remove}
      </Button>
    </Dialog>
  )
}
