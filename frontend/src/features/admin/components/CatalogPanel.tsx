import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useCan } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState, ErrorState } from '@/shared/ui'
import type { CatalogKind } from '../api/adminApi'
import { useCatalog, useDeleteCatalog, useSaveCatalog } from '../api/queries'
import type { CatalogItem } from '../model/types'
import { ConfirmDeleteDialog } from './ConfirmDeleteDialog'
import { NameDialog } from './NameDialog'
import { cellClass, DataTable, TableSkeleton } from '@/shared/ui'

const t = messages.admin

const TEXT = { 'job-titles': t.jobTitles, skills: t.skills } as const

type Editing = { mode: 'create' } | { mode: 'edit'; item: CatalogItem }

/** Danh mục phẳng: chức danh hoặc kỹ năng (CF-02). */
export function CatalogPanel({ kind }: { kind: CatalogKind }) {
  const text = TEXT[kind]
  const canManage = useCan('org.manage')
  const query = useCatalog(kind)
  const save = useSaveCatalog(kind)
  const remove = useDeleteCatalog(kind)
  const [editing, setEditing] = useState<Editing | null>(null)
  const [deleting, setDeleting] = useState<CatalogItem | null>(null)
  const [openKey, setOpenKey] = useState(0)

  const open = (e: Editing) => {
    setOpenKey((k) => k + 1)
    setEditing(e)
  }

  let content
  if (query.isPending) content = <TableSkeleton />
  else if (query.isError)
    content = (
      <ErrorState
        title={t.org.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  else if (query.data.length === 0)
    content = <EmptyState title={text.emptyTitle} description={text.emptyHint} />
  else
    content = (
      <DataTable
        label={text.tableLabel}
        head={[
          { label: t.org.name },
          { label: t.org.members, numeric: true, className: 'w-32' },
          ...(canManage ? [{ label: t.org.actions, className: 'w-48' }] : []),
        ]}
      >
        {query.data.map((item) => (
          <tr key={item.id}>
            <td className={cellClass}>{item.name}</td>
            <td className={`${cellClass} num text-right`}>{formatNumber(item.memberCount)}</td>
            {canManage && (
              <td className={cellClass}>
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    onClick={() => {
                      open({ mode: 'edit', item })
                    }}
                  >
                    {t.org.rename}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setDeleting(item)
                    }}
                  >
                    {t.org.remove}
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
              open({ mode: 'create' })
            }}
          >
            {text.add}
          </Button>
        </div>
      )}
      {content}
      {editing && (
        <NameDialog
          key={openKey}
          title={editing.mode === 'edit' ? text.editTitle : text.createTitle}
          isEdit={editing.mode === 'edit'}
          defaultValues={{ name: editing.mode === 'edit' ? editing.item.name : '' }}
          save={(v) =>
            save.mutateAsync({
              id: editing.mode === 'edit' ? editing.item.id : undefined,
              name: v.name,
            })
          }
          onClose={() => {
            setEditing(null)
          }}
        />
      )}
      {deleting && (
        <ConfirmDeleteDialog
          name={deleting.name}
          onConfirm={() => remove.mutateAsync(deleting.id)}
          onClose={() => {
            setDeleting(null)
          }}
        />
      )}
    </div>
  )
}
