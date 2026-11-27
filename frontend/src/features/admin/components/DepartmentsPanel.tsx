import { Plus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useCan } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState, ErrorState, type SelectOption } from '@/shared/ui'
import { useDeleteDepartment, useDepartments, useSaveDepartment } from '../api/queries'
import { departmentOptions, descendantsOf, flattenDepartments } from '../model/departmentTree'
import type { Department } from '../model/types'
import { ConfirmDeleteDialog } from './ConfirmDeleteDialog'
import { NameDialog } from './NameDialog'
import { cellClass, DataTable, TableSkeleton } from '@/shared/ui'

const t = messages.admin

// Lớp tĩnh để Tailwind quét được; sâu hơn 4 cấp giữ nguyên thụt lề cấp 4.
const INDENT = ['pl-0', 'pl-5', 'pl-10', 'pl-15', 'pl-20'] as const

type Editing = { mode: 'create'; parentId: string | null } | { mode: 'edit'; item: Department }

/** Cây phòng ban (CF-02): thêm cấp gốc / phòng ban con, đổi tên/chuyển cha, xóa. */
export function DepartmentsPanel() {
  const canManage = useCan('org.manage')
  const query = useDepartments()
  const save = useSaveDepartment()
  const remove = useDeleteDepartment()
  const [editing, setEditing] = useState<Editing | null>(null)
  const [deleting, setDeleting] = useState<Department | null>(null)
  const [openKey, setOpenKey] = useState(0)
  const rows = useMemo(() => flattenDepartments(query.data ?? []), [query.data])

  const open = (e: Editing) => {
    setOpenKey((k) => k + 1)
    setEditing(e)
  }

  const parentOptions = (exclude: Set<string>): SelectOption[] => [
    { value: '', label: t.departments.root },
    ...departmentOptions(rows.filter((d) => !exclude.has(d.id))),
  ]

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
  else if (rows.length === 0)
    content = <EmptyState title={t.departments.emptyTitle} description={t.departments.emptyHint} />
  else
    content = (
      <DataTable
        label={t.departments.tableLabel}
        head={[
          { label: t.org.name },
          { label: t.org.members, numeric: true, className: 'w-32' },
          ...(canManage ? [{ label: t.org.actions, className: 'w-80' }] : []),
        ]}
      >
        {rows.map((d) => (
          <tr key={d.id}>
            <td className={cellClass}>
              <span className={INDENT[Math.min(d.depth, INDENT.length - 1)]}>{d.name}</span>
            </td>
            <td className={`${cellClass} num text-right`}>{formatNumber(d.memberCount)}</td>
            {canManage && (
              <td className={cellClass}>
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    onClick={() => {
                      open({ mode: 'create', parentId: d.id })
                    }}
                  >
                    {t.departments.addChild}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => {
                      open({ mode: 'edit', item: d })
                    }}
                  >
                    {t.org.rename}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setDeleting(d)
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
              open({ mode: 'create', parentId: null })
            }}
          >
            {t.departments.add}
          </Button>
        </div>
      )}
      {content}
      {editing && (
        <NameDialog
          key={openKey}
          title={editing.mode === 'edit' ? t.departments.editTitle : t.departments.createTitle}
          isEdit={editing.mode === 'edit'}
          defaultValues={
            editing.mode === 'edit'
              ? { name: editing.item.name, parentId: editing.item.parentId ?? '' }
              : { name: '', parentId: editing.parentId ?? '' }
          }
          parentOptions={parentOptions(
            editing.mode === 'edit'
              ? descendantsOf(query.data ?? [], editing.item.id)
              : new Set<string>(),
          )}
          save={(v) =>
            save.mutateAsync({
              id: editing.mode === 'edit' ? editing.item.id : undefined,
              name: v.name,
              parentId: v.parentId ?? null,
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
