import { ListTodo, Search } from 'lucide-react'
import { useCallback, useState } from 'react'
import { Link } from 'react-router'
import { PAGE_SIZE, STATUS_GROUPS } from '@/features/projects'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Badge,
  Button,
  EmptyState,
  ErrorState,
  Pagination,
  Select,
  TableSkeleton,
} from '@/shared/ui'
import { useMyTasks, useMyWip } from '../api/queries'
import type { MyTaskFilters, Task } from '../model/types'
import { TaskDetailDialog } from './TaskDetailDialog'
import { TaskSearchInput } from './TaskSearchInput'
import { TaskTable } from './TaskTable'

const t = messages.tasks
const groupLabels = messages.projects.groups
const emptyFilters: MyTaskFilters = { search: '', statusGroup: '' }

/** Trang "Công việc của tôi" (CF-06): công việc cha được giao cho tôi ở mọi dự án, dạng bảng. */
export function MyTasksView() {
  const [filters, setFilters] = useState<MyTaskFilters>(emptyFilters)
  const [page, setPage] = useState(1)
  const [resetKey, setResetKey] = useState(0)
  const [open, setOpen] = useState<Task | null>(null)
  const tasks = useMyTasks({ ...filters, page, pageSize: PAGE_SIZE })
  const wip = useMyWip()

  const onSearch = useCallback((search: string) => {
    setFilters((f) => (f.search === search ? f : { ...f, search }))
    setPage(1)
  }, [])
  const filtered = filters.search !== '' || filters.statusGroup !== ''
  const clearFilters = () => {
    setFilters(emptyFilters)
    setResetKey((k) => k + 1)
    setPage(1)
  }

  let content
  if (tasks.isError && !tasks.data)
    content = (
      <ErrorState
        title={t.mine.loadError}
        errorRef={tasks.error instanceof ApiError ? tasks.error.code : undefined}
        onRetry={() => void tasks.refetch()}
      />
    )
  else if (!tasks.data) content = <TableSkeleton rows={6} />
  else if (tasks.data.total === 0)
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
        icon={ListTodo}
        title={t.mine.emptyTitle}
        description={t.mine.emptyHint}
        actions={
          <Link
            to="/projects"
            className="inline-flex h-10 items-center rounded-md border border-line-strong bg-surface px-4 text-base font-semibold text-ink hover:bg-subtle focus-visible:outline-2 focus-visible:outline-accent"
          >
            {t.mine.goProjects}
          </Link>
        }
      />
    )
  else
    content = (
      <>
        <TaskTable tasks={tasks.data.items} showProject onOpen={setOpen} />
        <div className="flex items-center justify-between">
          <span className="num text-sm text-ink-muted">
            {t.list.total(formatNumber(tasks.data.total))}
          </span>
          <Pagination
            page={page}
            pageSize={PAGE_SIZE}
            total={tasks.data.total}
            onChange={setPage}
          />
        </div>
      </>
    )

  return (
    <div className="flex flex-col gap-4">
      <header className="flex items-baseline gap-3">
        <h1 className="text-lg font-semibold text-ink">{t.mine.title}</h1>
        <p className="text-sm text-ink-muted">{t.mine.meta}</p>
        {wip.data !== undefined && (
          <span className="ml-auto" title={t.wipHint}>
            <Badge tone="active">{t.wip(formatNumber(wip.data))}</Badge>
          </span>
        )}
      </header>
      <div className="flex flex-wrap items-end gap-3">
        <TaskSearchInput onSearch={onSearch} resetKey={resetKey} />
        <div className="w-40">
          <Select
            label={t.list.filterGroup}
            value={filters.statusGroup}
            options={[
              { value: '', label: t.list.all },
              ...STATUS_GROUPS.map((g) => ({ value: g, label: groupLabels[g] })),
            ]}
            onChange={(e) => {
              const statusGroup = STATUS_GROUPS.find((g) => g === e.target.value) ?? ''
              setFilters((f) => ({ ...f, statusGroup }))
              setPage(1)
            }}
          />
        </div>
        {filtered && (
          <Button variant="ghost" onClick={clearFilters}>
            {t.list.clearFilters}
          </Button>
        )}
      </div>
      {content}
      {open && (
        <TaskDetailDialog
          initial={open}
          canEdit={false}
          onClose={() => {
            setOpen(null)
          }}
        />
      )}
    </div>
  )
}
