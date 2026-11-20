import { ListTodo, Plus, Search } from 'lucide-react'
import { useCallback, useState } from 'react'
import { PAGE_SIZE, STATUS_GROUPS, useProjectMembers, type Project } from '@/features/projects'
import { ApiError } from '@/shared/api/http'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState, ErrorState, Pagination, Select, TableSkeleton } from '@/shared/ui'
import { useCreateTask, useLabels, useProjectTasks } from '../api/queries'
import type { Task, TaskFilters } from '../model/types'
import { TaskDetailDialog } from './TaskDetailDialog'
import { TaskFormDialog } from './TaskFormDialog'
import { TaskSearchInput } from './TaskSearchInput'
import { TaskTable } from './TaskTable'

const t = messages.tasks.list
const groupLabels = messages.projects.groups
const emptyFilters: TaskFilters = { search: '', statusGroup: '', assignee: '', labelId: '' }

export interface ProjectTasksPanelProps {
  project: Project
  /** Manager phụ trách dự án: tạo/sửa/giao việc (server kiểm lại). */
  canManage: boolean
}

/** Tab "Công việc" của dự án (CF-06): bảng công việc cha có lọc + phân trang. Không phải Kanban. */
export function ProjectTasksPanel({ project, canManage }: ProjectTasksPanelProps) {
  const [filters, setFilters] = useState<TaskFilters>(emptyFilters)
  const [page, setPage] = useState(1)
  const [resetKey, setResetKey] = useState(0)
  const [creating, setCreating] = useState(false)
  const [open, setOpen] = useState<Task | null>(null)
  const tasks = useProjectTasks(project.id, { ...filters, page, pageSize: PAGE_SIZE })
  const members = useProjectMembers(project.id)
  const labels = useLabels()
  const create = useCreateTask(project.id)

  const patch = useCallback((next: Partial<TaskFilters>) => {
    setFilters((f) => {
      const merged = { ...f, ...next }
      return (Object.keys(next) as (keyof TaskFilters)[]).every((k) => f[k] === merged[k])
        ? f
        : merged
    })
    setPage(1)
  }, [])
  const onSearch = useCallback(
    (search: string) => {
      patch({ search })
    },
    [patch],
  )

  const filtered =
    filters.search !== '' ||
    filters.statusGroup !== '' ||
    filters.assignee !== '' ||
    filters.labelId !== ''
  const clearFilters = () => {
    setFilters(emptyFilters)
    setResetKey((k) => k + 1)
    setPage(1)
  }

  const addButton = canManage && (
    <Button
      icon={<Plus size={16} strokeWidth={2} aria-hidden />}
      onClick={() => {
        setCreating(true)
      }}
    >
      {t.add}
    </Button>
  )

  let content
  if (tasks.isError && !tasks.data)
    content = (
      <ErrorState
        title={t.loadError}
        errorRef={tasks.error instanceof ApiError ? tasks.error.code : undefined}
        onRetry={() => void tasks.refetch()}
      />
    )
  else if (!tasks.data) content = <TableSkeleton rows={6} />
  else if (tasks.data.total === 0)
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
        icon={ListTodo}
        title={t.emptyTitle}
        description={canManage ? t.emptyHintManager : t.emptyHint}
        actions={addButton}
      />
    )
  else
    content = (
      <>
        <TaskTable tasks={tasks.data.items} onOpen={setOpen} />
        <div className="flex items-center justify-between">
          <span className="num text-sm text-ink-muted">
            {t.total(formatNumber(tasks.data.total))}
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
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end gap-3">
        <TaskSearchInput onSearch={onSearch} resetKey={resetKey} />
        <div className="w-40">
          <Select
            label={t.filterGroup}
            value={filters.statusGroup}
            options={[
              { value: '', label: t.all },
              ...STATUS_GROUPS.map((g) => ({ value: g, label: groupLabels[g] })),
            ]}
            onChange={(e) => {
              patch({ statusGroup: STATUS_GROUPS.find((g) => g === e.target.value) ?? '' })
            }}
          />
        </div>
        <div className="w-48">
          <Select
            label={t.filterAssignee}
            value={filters.assignee}
            options={[
              { value: '', label: t.all },
              { value: 'none', label: t.unassigned },
              ...(members.data ?? [])
                .filter((m) => m.role !== 'ceo' && m.role !== 'admin')
                .map((m) => ({ value: m.userId, label: m.fullName })),
            ]}
            onChange={(e) => {
              patch({ assignee: e.target.value })
            }}
          />
        </div>
        <div className="w-40">
          <Select
            label={t.filterLabel}
            value={filters.labelId}
            options={[
              { value: '', label: t.all },
              ...(labels.data ?? []).map((l) => ({ value: l.id, label: l.name })),
            ]}
            onChange={(e) => {
              patch({ labelId: e.target.value })
            }}
          />
        </div>
        {filtered && (
          <Button variant="ghost" onClick={clearFilters}>
            {t.clearFilters}
          </Button>
        )}
        {addButton && <div className="ml-auto">{addButton}</div>}
      </div>
      {content}
      {creating && (
        <TaskFormDialog
          projectId={project.id}
          save={(body) => create.mutateAsync(body)}
          onClose={() => {
            setCreating(false)
          }}
        />
      )}
      {open && (
        <TaskDetailDialog
          initial={open}
          canEdit={canManage}
          onClose={() => {
            setOpen(null)
          }}
        />
      )}
    </div>
  )
}
