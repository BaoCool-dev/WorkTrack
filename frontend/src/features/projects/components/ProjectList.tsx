import { FolderKanban, Plus, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useCan } from '@/features/auth'
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
  Pagination,
  Select,
  TableSkeleton,
} from '@/shared/ui'
import { useCreateProject, useProjects } from '../api/queries'
import { formatDate } from '@/shared/lib/format'
import { projectStatusTone } from '../model/status'
import { PAGE_SIZE, PROJECT_STATUSES, type ProjectFilters } from '../model/types'
import { ProjectFormDialog } from './ProjectFormDialog'

const t = messages.projects
const SEARCH_DEBOUNCE_MS = 300
const emptyFilters: ProjectFilters = { search: '', status: '' }

/** Danh sách dự án theo vai trò (CF-03): Manager/Member thấy dự án mình tham gia, CEO thấy tất cả. */
export function ProjectList() {
  const canCreate = useCan('projects.create')
  const viewAll = useCan('projects.view_all')
  const navigate = useNavigate()
  const [filters, setFilters] = useState<ProjectFilters>(emptyFilters)
  const [searchInput, setSearchInput] = useState('')
  const [page, setPage] = useState(1)
  const [creating, setCreating] = useState(false)
  const projects = useProjects({ ...filters, page, pageSize: PAGE_SIZE })
  const create = useCreateProject()

  useEffect(() => {
    const id = setTimeout(() => {
      setFilters((f) => (f.search === searchInput ? f : { ...f, search: searchInput }))
      setPage(1)
    }, SEARCH_DEBOUNCE_MS)
    return () => {
      clearTimeout(id)
    }
  }, [searchInput])

  const filtered = filters.search !== '' || filters.status !== ''
  const clearFilters = () => {
    setSearchInput('')
    setFilters(emptyFilters)
    setPage(1)
  }
  const addButton = canCreate && (
    <Button
      icon={<Plus size={16} strokeWidth={2} aria-hidden />}
      onClick={() => {
        setCreating(true)
      }}
    >
      {t.list.add}
    </Button>
  )

  let content
  if (projects.isError && !projects.data)
    content = (
      <ErrorState
        title={t.list.loadError}
        errorRef={projects.error instanceof ApiError ? projects.error.code : undefined}
        onRetry={() => void projects.refetch()}
      />
    )
  else if (!projects.data) content = <TableSkeleton rows={6} />
  else if (projects.data.total === 0)
    content = filtered ? (
      <EmptyState
        icon={Search}
        title={t.list.noMatchTitle}
        description={t.list.noMatchHint}
        actions={
          <Button variant="secondary" onClick={clearFilters}>
            {t.list.clearFilters}
          </Button>
        }
      />
    ) : (
      <EmptyState
        icon={FolderKanban}
        title={t.list.emptyTitle}
        description={canCreate ? t.list.emptyHintManager : t.list.emptyHint}
        actions={addButton}
      />
    )
  else
    content = (
      <>
        <DataTable
          label={t.list.tableLabel}
          head={[
            { label: t.list.colName },
            { label: t.list.colManager },
            { label: t.list.colStatus },
            { label: t.list.colDates },
            { label: t.list.colMembers, numeric: true },
          ]}
        >
          {projects.data.items.map((p) => {
            const from = formatDate(p.startDate)
            const to = formatDate(p.endDate)
            return (
              <tr key={p.id} className="hover:bg-subtle">
                <td className={cellClass}>
                  <Link
                    to={`/projects/${encodeURIComponent(p.id)}`}
                    className="font-semibold text-ink hover:text-accent hover:underline"
                  >
                    {p.name}
                  </Link>
                </td>
                <td className={cellClass}>{p.manager.name}</td>
                <td className={cellClass}>
                  <Badge tone={projectStatusTone[p.status]}>{t.status[p.status]}</Badge>
                </td>
                <td className={`${cellClass} num text-sm text-ink-muted`}>
                  {from || to ? t.list.dateRange(from ?? '…', to ?? '…') : t.list.none}
                </td>
                <td className={`${cellClass} num text-right`}>{formatNumber(p.memberCount)}</td>
              </tr>
            )
          })}
        </DataTable>
        <div className="flex items-center justify-between">
          <span className="num text-sm text-ink-muted">
            {t.list.total(formatNumber(projects.data.total))}
          </span>
          <Pagination
            page={page}
            pageSize={PAGE_SIZE}
            total={projects.data.total}
            onChange={setPage}
          />
        </div>
      </>
    )

  return (
    <div className="flex flex-col gap-3">
      <header className="flex items-baseline gap-3">
        <h1 className="text-lg font-semibold text-ink">{t.title}</h1>
        <p className="text-sm text-ink-muted">{viewAll ? t.metaAll : t.meta}</p>
      </header>
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
            aria-label={t.list.searchLabel}
            placeholder={t.list.search}
            value={searchInput}
            onChange={(e) => {
              setSearchInput(e.target.value)
            }}
            className="h-10 w-full rounded-md border border-line-strong bg-surface pr-3 pl-9 text-base text-ink placeholder:text-ink-faint focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none"
          />
        </div>
        <div className="w-44">
          <Select
            label={t.list.filterStatus}
            value={filters.status}
            options={[
              { value: '', label: t.list.all },
              ...PROJECT_STATUSES.map((s) => ({ value: s, label: t.status[s] })),
            ]}
            onChange={(e) => {
              const v = PROJECT_STATUSES.find((s) => s === e.target.value)
              setFilters((f) => ({ ...f, status: v ?? '' }))
              setPage(1)
            }}
          />
        </div>
        {filtered && (
          <Button variant="ghost" onClick={clearFilters}>
            {t.list.clearFilters}
          </Button>
        )}
        {addButton && <div className="ml-auto">{addButton}</div>}
      </div>
      {content}
      {creating && (
        <ProjectFormDialog
          save={async (body) => {
            const p = await create.mutateAsync(body)
            void navigate(`/projects/${encodeURIComponent(p.id)}`)
          }}
          onClose={() => {
            setCreating(false)
          }}
        />
      )}
    </div>
  )
}
