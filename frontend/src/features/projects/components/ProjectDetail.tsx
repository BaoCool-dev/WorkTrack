import { ArrowLeft, FolderX, Pencil } from 'lucide-react'
import { useId, useState, type ReactNode } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useAuth } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { cn } from '@/shared/lib/cn'
import { formatDate, formatNumber, formatTimestampDate } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Badge, Button, EmptyState, ErrorState, Skeleton } from '@/shared/ui'
import { useProject, useUpdateProject } from '../api/queries'
import { projectStatusTone } from '../model/status'
import type { Project } from '../model/types'
import { ColumnsPanel } from './ColumnsPanel'
import { MembersPanel } from './MembersPanel'
import { ProjectFormDialog } from './ProjectFormDialog'
import { TransitionsPanel } from './TransitionsPanel'

const t = messages.projects

const TABS = [
  { value: 'info', label: t.detail.tabs.info },
  { value: 'board', label: messages.kanban.tab },
  { value: 'tasks', label: t.detail.tabs.tasks },
  { value: 'members', label: t.detail.tabs.members },
  { value: 'columns', label: t.detail.tabs.columns },
  { value: 'transitions', label: t.detail.tabs.transitions },
] as const
type Tab = (typeof TABS)[number]['value']

/** Tab do trang ghép vào (board, tasks) chỉ hợp lệ khi trang có truyền nội dung. */
function parseTab(v: string | null, slots: Partial<Record<Tab, unknown>>): Tab {
  const found = TABS.find((x) => x.value === v)?.value ?? 'info'
  const injected = found === 'tasks' || found === 'board'
  return injected && !slots[found] ? 'info' : found
}

/** Chỉ Manager phụ trách được sửa — chỉ để ẩn nút; server kiểm lại (R: server là nguồn sự thật). */
function useCanManage(p: Project | undefined): boolean {
  const { user } = useAuth()
  return !!p && !!user && user.role === 'manager' && user.id === p.manager.id
}

/** Chi tiết + cấu hình dự án (CF-03). Tab lưu trong ?tab để quay lại đúng chỗ. */
export interface ProjectDetailProps {
  projectId: string
  /** Tab Công việc do trang ghép vào (features/tasks) — tránh phụ thuộc vòng projects ↔ tasks. */
  renderTasks?: (project: Project, canManage: boolean) => ReactNode
  /** Tab Kanban do trang ghép vào (features/kanban) — cùng lý do như renderTasks. */
  renderBoard?: (project: Project, canManage: boolean) => ReactNode
}

export function ProjectDetail({ projectId, renderTasks, renderBoard }: ProjectDetailProps) {
  const project = useProject(projectId)
  const update = useUpdateProject(projectId)
  const canManage = useCanManage(project.data)
  const [params, setParams] = useSearchParams()
  const slots = { tasks: renderTasks, board: renderBoard }
  const tab = parseTab(params.get('tab'), slots)
  const baseId = useId()
  const [editing, setEditing] = useState(false)

  const back = (
    <Link
      to="/projects"
      className="inline-flex items-center gap-1 text-sm text-ink-muted hover:text-accent"
    >
      <ArrowLeft size={16} strokeWidth={2} aria-hidden />
      {t.detail.back}
    </Link>
  )

  if (project.isError && !project.data) {
    const notFound =
      project.error instanceof ApiError &&
      (project.error.status === 404 || project.error.code === 'PROJECT_ID_INVALID')
    return (
      <div className="flex flex-col gap-3">
        {back}
        {notFound ? (
          <EmptyState
            icon={FolderX}
            title={t.detail.notFoundTitle}
            description={t.detail.notFoundHint}
          />
        ) : (
          <ErrorState
            title={t.detail.loadError}
            errorRef={project.error instanceof ApiError ? project.error.code : undefined}
            onRetry={() => void project.refetch()}
          />
        )}
      </div>
    )
  }
  if (!project.data)
    return (
      <div className="flex flex-col gap-3" aria-busy="true">
        {back}
        <Skeleton className="h-7 w-80" />
        <Skeleton className="h-40 w-full" />
      </div>
    )

  const p = project.data
  const select = (next: Tab) => {
    setParams(
      (cur) => {
        const q = new URLSearchParams(cur)
        if (next === 'info') q.delete('tab')
        else q.set('tab', next)
        return q
      },
      { replace: true },
    )
  }

  return (
    <div className="flex flex-col gap-4">
      {back}
      <header className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold text-ink">{p.name}</h1>
        <Badge tone={projectStatusTone[p.status]}>{t.status[p.status]}</Badge>
        {canManage && (
          <div className="ml-auto">
            <Button
              variant="secondary"
              icon={<Pencil size={16} strokeWidth={2} aria-hidden />}
              onClick={() => {
                setEditing(true)
              }}
            >
              {t.detail.edit}
            </Button>
          </div>
        )}
      </header>
      {!canManage && <p className="text-sm text-ink-muted">{t.detail.readOnly}</p>}
      <div
        role="tablist"
        aria-label={t.detail.tabsLabel}
        className="flex gap-1 border-b border-line"
      >
        {TABS.filter((x) => (x.value !== 'tasks' && x.value !== 'board') || slots[x.value]).map(
          (x) => {
            const selected = x.value === tab
            return (
              <button
                key={x.value}
                type="button"
                role="tab"
                id={`${baseId}-${x.value}-tab`}
                aria-selected={selected}
                aria-controls={`${baseId}-panel`}
                onClick={() => {
                  select(x.value)
                }}
                className={cn(
                  '-mb-px border-b-2 px-3 py-2 text-base focus-visible:outline-2 focus-visible:outline-accent',
                  selected
                    ? 'border-accent font-semibold text-accent'
                    : 'border-transparent text-ink-muted hover:text-ink',
                )}
              >
                {x.label}
              </button>
            )
          },
        )}
      </div>
      <div role="tabpanel" id={`${baseId}-panel`} aria-labelledby={`${baseId}-${tab}-tab`}>
        {tab === 'info' && <InfoPanel project={p} />}
        {tab === 'board' && renderBoard?.(p, canManage)}
        {tab === 'tasks' && renderTasks?.(p, canManage)}
        {tab === 'members' && <MembersPanel project={p} canManage={canManage} />}
        {tab === 'columns' && <ColumnsPanel projectId={p.id} canManage={canManage} />}
        {tab === 'transitions' && <TransitionsPanel projectId={p.id} canManage={canManage} />}
      </div>
      {editing && (
        <ProjectFormDialog
          project={p}
          save={(body) => update.mutateAsync(body)}
          onClose={() => {
            setEditing(false)
          }}
        />
      )}
    </div>
  )
}

function InfoPanel({ project: p }: { project: Project }) {
  const rows: [string, string][] = [
    [t.detail.manager, p.manager.name],
    [t.detail.memberCount, formatNumber(p.memberCount)],
    [t.detail.startDate, formatDate(p.startDate) ?? t.list.none],
    [t.detail.endDate, formatDate(p.endDate) ?? t.list.none],
    [t.detail.createdAt, formatTimestampDate(p.createdAt)],
  ]
  return (
    <div className="grid max-w-3xl grid-cols-[1fr_2fr] gap-6">
      <dl className="flex flex-col gap-2 rounded-md border border-line bg-surface p-3">
        {rows.map(([k, v]) => (
          <div key={k} className="flex flex-col">
            <dt className="text-sm font-semibold text-ink-faint">{k}</dt>
            <dd className="num text-base text-ink">{v}</dd>
          </div>
        ))}
      </dl>
      <section className="flex flex-col gap-1">
        <h2 className="text-sm font-semibold text-ink-faint">{t.form.description}</h2>
        <p className="whitespace-pre-line text-base text-ink">
          {p.description || <span className="text-ink-muted">{t.detail.noDescription}</span>}
        </p>
      </section>
    </div>
  )
}
