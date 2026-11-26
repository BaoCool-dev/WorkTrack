import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  pointerWithin,
  rectIntersection,
  useSensor,
  useSensors,
  type CollisionDetection,
} from '@dnd-kit/core'
import { Eye } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router'

import { useAuth } from '@/features/auth'
import { useBoard, useProjectMembers, type Board, type Project } from '@/features/projects'
import { TaskDetailDialog, TaskFormDialog, useCreateTask, type Task } from '@/features/tasks'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState, ErrorState, Skeleton } from '@/shared/ui'

import { useBoardTasks } from '../api/queries'
import { useBoardDrag } from '../hooks/useBoardDrag'
import { useProjectSocket } from '../hooks/useProjectSocket'
import { boardAnnouncements, screenReaderInstructions } from '../model/a11y'
import { columnTasks, filterTasks, wipByAssignee, type BoardFilters } from '../model/boardState'
import { boardRoleOf, canDragTask, type BoardRole } from '../model/dropRules'
import { DisconnectBanner, SaveErrorBanner } from './Banners'
import { FilterBar, HoursLegend, LiveIndicator, WipChip } from './BoardToolbar'
import { KanbanColumn } from './KanbanColumn'
import { DraggableTaskCard, TaskCardView } from './TaskCard'
import { WipFullDialog } from './WipFullDialog'
import { WipLimitControl } from './WipLimitControl'

const t = messages.kanban

const NO_FILTERS: BoardFilters = { labelIds: [], assigneeIds: [] }

/**
 * Ưu tiên thẻ nằm dưới con trỏ (để chèn đúng chỗ), không có thì lấy cột; con trỏ ra ngoài mọi cột
 * (kéo bằng bàn phím) thì dùng phần giao nhau của khung thẻ.
 */
const collision: CollisionDetection = (args) => {
  const hits = pointerWithin(args)
  const cards = hits.filter((h) => String(h.id).startsWith('task:'))
  if (cards.length > 0) return cards
  return hits.length > 0 ? hits : rectIntersection(args)
}

export interface KanbanBoardProps {
  project: Project
  /** Người xem là Manager phụ trách dự án. */
  canManage: boolean
}

/** Bảng Kanban của dự án (CF-04, CF-05, UC05): tải dữ liệu rồi giao cho BoardCanvas. */
export function KanbanBoard({ project, canManage }: KanbanBoardProps) {
  const board = useBoard(project.id)
  const tasks = useBoardTasks(project.id)

  if (board.isPending || tasks.isPending) return <BoardSkeleton />
  if (board.isError || tasks.isError) {
    const err = board.error ?? tasks.error
    return (
      <ErrorState
        title={t.loadError}
        errorRef={err instanceof ApiError ? err.code : undefined}
        onRetry={() => {
          void board.refetch()
          void tasks.refetch()
        }}
      />
    )
  }
  return (
    <BoardCanvas project={project} canManage={canManage} board={board.data} tasks={tasks.data} />
  )
}

interface CanvasProps extends KanbanBoardProps {
  board: Board
  tasks: Task[]
}

function BoardCanvas({ project, canManage, board, tasks }: CanvasProps) {
  const { user } = useAuth()
  const viewerId = user?.id ?? ''
  const boardRole: BoardRole = user ? boardRoleOf(user, project.manager.id) : null
  const members = useProjectMembers(project.id)
  const socket = useProjectSocket(project.id)
  const create = useCreateTask(project.id)
  const navigate = useNavigate()

  const [filters, setFilters] = useState<BoardFilters>(NO_FILTERS)
  const [wipPerson, setWipPerson] = useState<{ id: string; name: string } | null>(null)
  const [opened, setOpened] = useState<Task | null>(null)
  const [creating, setCreating] = useState(false)

  const columns = [...board.columns].sort((a, b) => a.position - b.position)
  const drag = useBoardDrag({
    projectId: project.id,
    tasks,
    columns,
    transitions: board.transitions,
    boardRole,
    viewerId,
    wipLimit: project.wipLimit,
    onWipFull: setWipPerson,
  })
  const sensors = useSensors(
    // Phải di chuột ≥ 5px mới tính là kéo → nhấp chuột vẫn mở chi tiết thẻ.
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor),
  )

  const announcements = boardAnnouncements(tasks, columns)
  const visible = filterTasks(tasks, filters)
  const nameOf = new Map((members.data ?? []).map((m) => [m.userId, m.fullName]))
  const viewers = socket.presence.map((id) => nameOf.get(id) ?? t.unassigned)

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <FilterBar
          tasks={tasks}
          filters={filters}
          onChange={setFilters}
          mineId={boardRole === 'member' ? viewerId : undefined}
        />
        <div className="ml-auto flex flex-wrap items-center gap-3">
          <HoursLegend />
          {boardRole === 'member' && (
            <WipChip current={wipByAssignee(tasks).get(viewerId) ?? 0} limit={project.wipLimit} />
          )}
          {canManage && <WipLimitControl key={project.wipLimit} project={project} />}
          <LiveIndicator status={socket.status} viewers={viewers} />
        </div>
      </div>

      {socket.status === 'lost' && <DisconnectBanner />}
      {drag.saveError && <SaveErrorBanner message={drag.saveError} onDismiss={drag.dismissError} />}
      <RoleNote boardRole={boardRole} />

      {tasks.length === 0 ? (
        <EmptyState
          title={canManage ? t.emptyManager : t.emptyMember}
          description={canManage ? t.emptyManagerHint : t.emptyMemberHint}
          actions={
            canManage ? (
              <Button
                onClick={() => {
                  setCreating(true)
                }}
              >
                {t.createTask}
              </Button>
            ) : (
              <Button
                variant="secondary"
                onClick={() => {
                  void navigate('/my-tasks')
                }}
              >
                {t.viewMine}
              </Button>
            )
          }
        />
      ) : (
        <>
          {visible.length === 0 && <p className="text-sm text-ink-muted">{t.filters.noMatch}</p>}
          <DndContext
            sensors={sensors}
            collisionDetection={collision}
            onDragStart={drag.onDragStart}
            onDragEnd={drag.onDragEnd}
            onDragCancel={drag.onDragCancel}
            accessibility={{ announcements, screenReaderInstructions }}
          >
            <div
              role="region"
              aria-label={t.boardLabel}
              className="flex h-[calc(100vh-16rem)] min-h-90 items-start gap-2 overflow-x-auto pb-2"
            >
              {columns.map((col, i) => {
                const cards = columnTasks(visible, col.id)
                return (
                  <KanbanColumn
                    key={col.id}
                    column={col}
                    count={cards.length}
                    dropState={drag.dropStateFor(col.id)}
                    // Việc mới luôn vào cột đầu (Backlog) nên chỉ cột đầu có nút "+".
                    onAdd={
                      canManage && i === 0
                        ? () => {
                            setCreating(true)
                          }
                        : undefined
                    }
                  >
                    {cards.map((task) => (
                      <DraggableTaskCard
                        key={task.id}
                        task={task}
                        draggable={canDragTask(boardRole, viewerId, task)}
                        onOpen={setOpened}
                      />
                    ))}
                    {cards.length === 0 && (
                      <p className="px-0.5 py-2 text-sm text-ink-faint">{t.columnEmpty}</p>
                    )}
                  </KanbanColumn>
                )
              })}
            </div>
            <DragOverlay>{drag.active && <TaskCardView task={drag.active} lifted />}</DragOverlay>
          </DndContext>
        </>
      )}

      <WipFullDialog
        person={wipPerson}
        wipLimit={project.wipLimit}
        onClose={() => {
          setWipPerson(null)
        }}
        onViewDoing={(id) => {
          setFilters({ labelIds: [], assigneeIds: [id] })
          setWipPerson(null)
        }}
      />
      {creating && (
        <TaskFormDialog
          projectId={project.id}
          save={(body) => create.mutateAsync(body)}
          onClose={() => {
            setCreating(false)
          }}
        />
      )}
      {opened && (
        <TaskDetailDialog
          initial={opened}
          canEdit={canManage}
          onClose={() => {
            setOpened(null)
          }}
        />
      )}
    </div>
  )
}

/** Câu giải thích quyền ngay trên bảng: Member biết mình kéo được gì, CEO/Admin biết mình chỉ xem. */
function RoleNote({ boardRole }: { boardRole: BoardRole }) {
  if (boardRole === 'manager') return null
  return (
    <p className="flex items-center gap-1.5 text-sm text-ink-muted">
      <Eye size={14} strokeWidth={2} aria-hidden />
      {boardRole === 'member' ? t.memberHint : t.readOnly}
    </p>
  )
}

/** Skeleton gần đúng kích thước thật: 4 cột, thẻ cao ~72px (ba hàng của TaskCardView). */
function BoardSkeleton() {
  return (
    <div aria-busy="true" aria-label={t.boardLabel} className="flex gap-2">
      {[0, 1, 2, 3].map((c) => (
        <div
          key={c}
          className="flex w-65 shrink-0 flex-col gap-1.5 rounded-md border border-line bg-subtle p-2"
        >
          <Skeleton className="h-5 w-24 bg-line" />
          {[0, 1, 2].map((r) => (
            <Skeleton key={r} className="h-18 w-full bg-surface" />
          ))}
        </div>
      ))}
    </div>
  )
}
