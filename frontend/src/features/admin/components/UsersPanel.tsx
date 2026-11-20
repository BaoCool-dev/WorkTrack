import { Plus, Search, Users } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useCan } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Avatar,
  Badge,
  Button,
  EmptyState,
  ErrorState,
  Pagination,
  Select,
  FormAlert,
  useToast,
  type BadgeTone,
} from '@/shared/ui'
import { useCatalog, useDepartments, useResendActivation, useUsers } from '../api/queries'
import { departmentOptions, flattenDepartments } from '../model/departmentTree'
import {
  emptyFilters,
  PAGE_SIZE,
  ROLES,
  STATUSES,
  type AdminUser,
  type UserFilters,
  type UserStatus,
} from '../model/types'
import { cellClass, OrgTable, TableSkeleton } from './OrgTable'
import { SkillsDialog } from './SkillsDialog'
import { UserFormDialog } from './UserFormDialog'

const t = messages.admin.users
const SEARCH_DEBOUNCE_MS = 300
const SHOWN_SKILLS = 3

const statusTone: Record<UserStatus, BadgeTone> = {
  pending: 'warn',
  active: 'ok',
  disabled: 'draft',
}

type Dialog =
  { kind: 'create' } | { kind: 'edit'; user: AdminUser } | { kind: 'skills'; user: AdminUser }

function hasFilters(f: UserFilters): boolean {
  return Object.values(f).some((v) => v !== '')
}

/** Danh sách nhân sự: tìm kiếm, lọc, phân trang (B9), thêm/sửa, hồ sơ kỹ năng (CF-02). */
export function UsersPanel() {
  const canManage = useCan('users.manage')
  const canAssignCeo = useCan('users.assign_ceo')
  const toast = useToast()
  const [filters, setFilters] = useState<UserFilters>(emptyFilters)
  const [searchInput, setSearchInput] = useState('')
  const [page, setPage] = useState(1)
  const [dialog, setDialog] = useState<Dialog | null>(null)
  const [dialogKey, setDialogKey] = useState(0)

  const users = useUsers({ ...filters, page, pageSize: PAGE_SIZE })
  const departments = useDepartments()
  const jobTitles = useCatalog('job-titles')
  const resend = useResendActivation()
  const [resendError, setResendError] = useState<string | null>(null)

  useEffect(() => {
    const id = setTimeout(() => {
      setFilters((f) => (f.search === searchInput ? f : { ...f, search: searchInput }))
      setPage(1)
    }, SEARCH_DEBOUNCE_MS)
    return () => {
      clearTimeout(id)
    }
  }, [searchInput])

  const setFilter = <K extends keyof UserFilters>(key: K, value: UserFilters[K]) => {
    setFilters((f) => ({ ...f, [key]: value }))
    setPage(1)
  }
  const clearFilters = () => {
    setSearchInput('')
    setFilters(emptyFilters)
    setPage(1)
  }
  const open = (d: Dialog) => {
    setDialogKey((k) => k + 1)
    setDialog(d)
  }

  const all = { value: '', label: t.all }
  const deptOptions = useMemo(
    () => [all, ...departmentOptions(flattenDepartments(departments.data ?? []))],
    // eslint-disable-next-line react-hooks/exhaustive-deps -- `all` là hằng theo messages
    [departments.data],
  )
  const filtered = hasFilters(filters)
  const canEdit = (u: AdminUser) => canManage && (u.role !== 'ceo' || canAssignCeo)

  const onResend = async (u: AdminUser) => {
    setResendError(null)
    try {
      await resend.mutateAsync(u.id)
      toast.show(t.resent)
    } catch (err) {
      // Lỗi cần đọc kỹ: hiện dải lỗi, không dùng toast (design-system §5).
      setResendError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  let content
  if (users.isError && !users.data)
    content = (
      <ErrorState
        title={t.loadError}
        errorRef={users.error instanceof ApiError ? users.error.code : undefined}
        onRetry={() => void users.refetch()}
      />
    )
  else if (!users.data) content = <TableSkeleton rows={8} />
  else if (users.data.total === 0)
    content = filtered ? (
      <EmptyState
        icon={Search}
        title={t.noMatchTitle}
        description={t.noMatchHint}
        actions={
          <Button variant="secondary" onClick={clearFilters}>
            {t.clearFilters}
          </Button>
        }
      />
    ) : (
      <EmptyState
        icon={Users}
        title={t.emptyTitle}
        description={t.emptyHint}
        actions={
          canManage && (
            <Button
              icon={<Plus size={16} strokeWidth={2} aria-hidden />}
              onClick={() => {
                open({ kind: 'create' })
              }}
            >
              {t.add}
            </Button>
          )
        }
      />
    )
  else
    content = (
      <>
        <OrgTable
          label={t.tableLabel}
          head={[
            { label: t.colName },
            { label: t.colRole },
            { label: t.colDepartment },
            { label: t.colJobTitle },
            { label: t.colSkills },
            { label: t.colStatus },
            ...(canManage ? [{ label: t.colActions }] : []),
          ]}
        >
          {users.data.items.map((u) => (
            <tr key={u.id}>
              <td className={cellClass}>
                <div className="flex items-center gap-2">
                  <Avatar name={u.fullName} />
                  <div className="flex min-w-0 flex-col">
                    <span className="truncate font-semibold text-ink">{u.fullName}</span>
                    <span className="truncate text-sm text-ink-muted">{u.email}</span>
                  </div>
                </div>
              </td>
              <td className={cellClass}>{messages.roles[u.role]}</td>
              <td className={cellClass}>{u.department?.name ?? t.none}</td>
              <td className={cellClass}>{u.jobTitle?.name ?? t.none}</td>
              <td className={`${cellClass} text-sm text-ink-muted`}>
                {u.skills.length === 0
                  ? t.none
                  : u.skills
                      .slice(0, SHOWN_SKILLS)
                      .map((s) => `${s.name} · ${messages.admin.levels[s.level]}`)
                      .join(', ')}
                {u.skills.length > SHOWN_SKILLS &&
                  ` ${t.moreSkills(formatNumber(u.skills.length - SHOWN_SKILLS))}`}
              </td>
              <td className={cellClass}>
                <Badge tone={statusTone[u.status]}>{messages.admin.status[u.status]}</Badge>
              </td>
              {canManage && (
                <td className={cellClass}>
                  <div className="flex gap-1">
                    {canEdit(u) && (
                      <>
                        <Button
                          variant="ghost"
                          onClick={() => {
                            open({ kind: 'edit', user: u })
                          }}
                        >
                          {t.edit}
                        </Button>
                        <Button
                          variant="ghost"
                          onClick={() => {
                            open({ kind: 'skills', user: u })
                          }}
                        >
                          {t.editSkills}
                        </Button>
                      </>
                    )}
                    {u.status === 'pending' && canEdit(u) && (
                      <Button variant="ghost" onClick={() => void onResend(u)}>
                        {t.resend}
                      </Button>
                    )}
                  </div>
                </td>
              )}
            </tr>
          ))}
        </OrgTable>
        <div className="flex items-center justify-between">
          <span className="num text-sm text-ink-muted">
            {t.total(formatNumber(users.data.total))}
          </span>
          <Pagination
            page={page}
            pageSize={PAGE_SIZE}
            total={users.data.total}
            onChange={setPage}
          />
        </div>
      </>
    )

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end gap-3">
        <div className="relative w-72">
          <Search
            size={16}
            strokeWidth={2}
            aria-hidden
            className="pointer-events-none absolute top-3 left-3 text-ink-faint"
          />
          <input
            type="search"
            aria-label={t.searchLabel}
            placeholder={t.search}
            value={searchInput}
            onChange={(e) => {
              setSearchInput(e.target.value)
            }}
            className="h-10 w-full rounded-md border border-line-strong bg-surface pr-3 pl-9 text-base text-ink placeholder:text-ink-faint focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none"
          />
        </div>
        <div className="w-40">
          <Select
            label={t.filterRole}
            value={filters.role}
            options={[all, ...ROLES.map((r) => ({ value: r, label: messages.roles[r] }))]}
            onChange={(e) => {
              const v = ROLES.find((r) => r === e.target.value)
              setFilter('role', v ?? '')
            }}
          />
        </div>
        <div className="w-40">
          <Select
            label={t.filterStatus}
            value={filters.status}
            options={[all, ...STATUSES.map((s) => ({ value: s, label: messages.admin.status[s] }))]}
            onChange={(e) => {
              const v = STATUSES.find((s) => s === e.target.value)
              setFilter('status', v ?? '')
            }}
          />
        </div>
        <div className="w-48">
          <Select
            label={t.filterDepartment}
            value={filters.departmentId}
            options={deptOptions}
            onChange={(e) => {
              setFilter('departmentId', e.target.value)
            }}
          />
        </div>
        <div className="w-44">
          <Select
            label={t.filterJobTitle}
            value={filters.jobTitleId}
            options={[all, ...(jobTitles.data ?? []).map((j) => ({ value: j.id, label: j.name }))]}
            onChange={(e) => {
              setFilter('jobTitleId', e.target.value)
            }}
          />
        </div>
        {filtered && (
          <Button variant="ghost" onClick={clearFilters}>
            {t.clearFilters}
          </Button>
        )}
        {canManage && (
          <div className="ml-auto">
            <Button
              icon={<Plus size={16} strokeWidth={2} aria-hidden />}
              onClick={() => {
                open({ kind: 'create' })
              }}
            >
              {t.add}
            </Button>
          </div>
        )}
      </div>
      <FormAlert message={resendError} />
      {content}
      {dialog?.kind === 'create' && (
        <UserFormDialog
          key={dialogKey}
          onClose={() => {
            setDialog(null)
          }}
        />
      )}
      {dialog?.kind === 'edit' && (
        <UserFormDialog
          key={dialogKey}
          user={dialog.user}
          onClose={() => {
            setDialog(null)
          }}
        />
      )}
      {dialog?.kind === 'skills' && (
        <SkillsDialog
          key={dialogKey}
          user={dialog.user}
          onClose={() => {
            setDialog(null)
          }}
        />
      )}
    </div>
  )
}
