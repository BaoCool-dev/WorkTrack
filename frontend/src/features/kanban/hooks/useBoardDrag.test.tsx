import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { ReactNode } from 'react'

import type { BoardColumn, Transition } from '@/features/projects'
import { makeBoard } from '@/features/projects/test-data'
import type { Task } from '@/features/tasks'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { ToastProvider } from '@/shared/ui'
import { API, server } from '@/test/server'

import { useBoardTasks } from '../api/queries'
import type { BoardRole } from '../model/dropRules'
import { boardTasks } from '../test-data'
import { useBoardDrag } from './useBoardDrag'

const t = messages.kanban.move
const board = makeBoard()
const col = (id: string) => {
  const c = board.columns.find((x) => x.id === id)
  if (!c) throw new Error(id)
  return c
}

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={qc}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  )
}

/** Hook thật + danh sách thẻ đọc từ cache (để thấy được cập nhật lạc quan và rollback). */
function setup(
  opts: {
    role?: BoardRole
    viewerId?: string
    wipLimit?: number
    tasks?: Task[]
    backRule?: boolean
    columns?: BoardColumn[]
    transitions?: Transition[]
  } = {},
) {
  const onWipFull = vi.fn()
  server.use(
    mock.get(`${API}/api/v1/projects/p-1/board/tasks`, () =>
      HttpResponse.json({ items: opts.tasks ?? boardTasks() }),
    ),
  )
  const r = renderHook(
    () => {
      const tasks = useBoardTasks('p-1').data ?? []
      const drag = useBoardDrag({
        projectId: 'p-1',
        tasks,
        columns: opts.columns ?? board.columns,
        // backRule: thêm luật c-2 → c-1 để đường hoàn tác hợp lệ.
        transitions: opts.transitions
          ? opts.transitions
          : opts.backRule
            ? [
                ...board.transitions,
                { fromColumnId: 'c-2', toColumnId: 'c-1', role: 'manager' as const },
              ]
            : board.transitions,
        boardRole: opts.role ?? 'manager',
        viewerId: opts.viewerId ?? 'u-manager',
        wipLimit: opts.wipLimit ?? 5,
        onWipFull,
      })
      return { tasks, drag }
    },
    { wrapper },
  )
  return { ...r, onWipFull }
}

/** Mô phỏng một lần kéo: nhấc thẻ rồi thả vào cột (khoảng trống cuối cột) hoặc lên một thẻ. */
function dragTo(
  r: ReturnType<typeof setup>['result'],
  taskId: string,
  columnId: string,
  overTaskId?: string,
) {
  act(() => {
    r.current.drag.onDragStart({
      active: { data: { current: { taskId } } },
    } as unknown as DragStartEvent)
  })
  act(() => {
    r.current.drag.onDragEnd({
      over: { data: { current: { columnId, taskId: overTaskId } } },
    } as unknown as DragEndEvent)
  })
}

const columnOf = (tasks: Task[], id: string) => tasks.find((x) => x.id === id)?.column.id

describe('useBoardDrag — kéo thả có luật, lạc quan, hoàn tác (CF-04, CF-05)', () => {
  it('khi đang kéo: cột hợp lệ ok, cột bị chặn có lý do cụ thể', async () => {
    const { result } = setup({ role: 'member', viewerId: 'u-member' })
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    act(() => {
      result.current.drag.onDragStart({
        active: { data: { current: { taskId: 't-1' } } },
      } as unknown as DragStartEvent)
    })
    expect(result.current.drag.dropStateFor('c-2')).toEqual({ ok: true })
    expect(result.current.drag.dropStateFor('c-3')).toEqual({
      ok: false,
      reason: 'Chỉ Quản lý dự án chuyển việc sang nhóm hoàn thành',
    })
  })

  it('thả hợp lệ: thẻ đổi cột ngay (lạc quan), gửi đúng body, toast có "Hoàn tác"', async () => {
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/tasks/t-2/move`, async ({ request }) => {
        body = await request.json()
        const moved = boardTasks()[1]
        return HttpResponse.json({ ...moved, column: col('c-2'), position: 1, version: 2 })
      }),
    )
    const { result } = setup()
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-2', 'c-2')
    await waitFor(() => {
      expect(columnOf(result.current.tasks, 't-2')).toBe('c-2')
    })
    expect(await screen.findByText(t.moved('In Progress'))).toBeInTheDocument()
    expect(body).toEqual({ columnId: 'c-2', position: 1, version: 1 })
    // Fixture không có luật c-2 → c-1: không hứa hoàn tác khi đường quay lại bị cấm.
    expect(screen.queryByRole('button', { name: t.undo })).not.toBeInTheDocument()
  })

  it('Hoàn tác gửi lại về cột cũ với version mới', async () => {
    const calls: unknown[] = []
    server.use(
      mock.post(`${API}/api/v1/tasks/t-2/move`, async ({ request }) => {
        const b = (await request.json()) as { columnId: string; position: number; version: number }
        calls.push(b)
        const moved = boardTasks()[1]
        return HttpResponse.json({
          ...moved,
          column: col(b.columnId),
          position: b.position,
          version: b.version + 1,
        })
      }),
    )
    const { result } = setup({ backRule: true })
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-2', 'c-2')
    await userEvent.setup().click(await screen.findByRole('button', { name: t.undo }))
    await waitFor(() => {
      expect(calls).toHaveLength(2)
    })
    expect(calls[1]).toEqual({ columnId: 'c-1', position: 1, version: 2 })
  })

  it('bị chặn ở client: không gọi API, toast nêu lý do', async () => {
    const { result } = setup({ role: 'member', viewerId: 'u-member' })
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-1', 'c-3')
    expect(
      await screen.findByText('Chỉ Quản lý dự án chuyển việc sang nhóm hoàn thành'),
    ).toBeInTheDocument()
    expect(columnOf(result.current.tasks, 't-1')).toBe('c-1')
  })

  it('WIP đã đủ ở client: mở hộp thoại cho người nhận, không gọi API', async () => {
    const { result, onWipFull } = setup({ wipLimit: 1 })
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-1', 'c-2') // Hà đã có t-3 đang làm, giới hạn 1
    expect(onWipFull).toHaveBeenCalledWith({ id: 'u-member', name: 'Đỗ Thu Hà' })
    expect(columnOf(result.current.tasks, 't-1')).toBe('c-1')
  })

  it('server trả 409 TASK_WIP_LIMIT (người khác vừa nhận thêm việc): mở hộp thoại, thẻ về chỗ cũ', async () => {
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/move`, () =>
        HttpResponse.json(
          { error: { code: 'TASK_WIP_LIMIT', message: 'Đã đủ WIP' } },
          { status: 409 },
        ),
      ),
    )
    const { result, onWipFull } = setup()
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-1', 'c-2')
    await waitFor(() => {
      expect(onWipFull).toHaveBeenCalled()
    })
    await waitFor(() => {
      expect(columnOf(result.current.tasks, 't-1')).toBe('c-1')
    })
  })

  it('409 xung đột version: dải lỗi giữ câu của server và nói thẻ đã về cột cũ', async () => {
    server.use(
      mock.post(`${API}/api/v1/tasks/t-2/move`, () =>
        HttpResponse.json(
          {
            error: { code: 'TASK_VERSION_CONFLICT', message: 'Thẻ vừa được người khác thay đổi.' },
          },
          { status: 409 },
        ),
      ),
    )
    const { result } = setup()
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-2', 'c-2')
    await waitFor(() => {
      expect(result.current.drag.saveError).toBe(t.reverted('Thẻ vừa được người khác thay đổi.'))
    })
    expect(columnOf(result.current.tasks, 't-2')).toBe('c-1')
  })

  it('lỗi mạng/5xx: dải lỗi câu chuẩn, thẻ trở về cột cũ', async () => {
    server.use(
      mock.post(`${API}/api/v1/tasks/t-2/move`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    const { result } = setup()
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-2', 'c-2')
    await waitFor(() => {
      expect(result.current.drag.saveError).toBe(t.saveFailed)
    })
    expect(columnOf(result.current.tasks, 't-2')).toBe('c-1')
    act(() => {
      result.current.drag.dismissError()
    })
    expect(result.current.drag.saveError).toBeNull()
  })

  it('thả vào đúng chỗ cũ: không gửi gì', async () => {
    const { result } = setup()
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-2', 'c-1') // đã ở cuối cột → vị trí không đổi; không có handler POST → MSW sẽ báo nếu gọi
    expect(screen.queryByText(t.reordered)).not.toBeInTheDocument()
  })
})

describe('useBoardDrag — đưa vào cột chờ nghiệm thu (CF-09)', () => {
  // Cột chờ nghiệm thu = cột in_progress cuối cùng (c-r), đứng trước Done.
  const columns: BoardColumn[] = [
    ...board.columns.filter((c) => c.id !== 'c-3'),
    { id: 'c-r', name: 'Chờ nghiệm thu', position: 2, statusGroup: 'in_progress', taskCount: 0 },
    { ...col('c-3'), position: 3 },
  ]
  const transitions: Transition[] = [{ fromColumnId: 'c-2', toColumnId: 'c-r', role: 'manager' }]
  const review = { id: 'c-r', name: 'Chờ nghiệm thu', statusGroup: 'in_progress' as const }

  function withSubtasks(done: number, total: number): Task[] {
    return boardTasks().map((x) =>
      x.id === 't-3' ? { ...x, subtaskCount: total, subtaskDoneCount: done } : x,
    )
  }

  function mockMove(tasks: Task[]) {
    server.use(
      mock.post(`${API}/api/v1/tasks/t-3/move`, () => {
        const t3 = tasks.find((x) => x.id === 't-3')
        return HttpResponse.json({ ...t3, column: review, position: 0, version: 2 })
      }),
    )
  }

  it('còn việc con chưa xong: vẫn chuyển, toast cảnh báo thay vì chặn', async () => {
    const tasks = withSubtasks(1, 3)
    mockMove(tasks)
    const { result } = setup({ tasks, columns, transitions })
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-3', 'c-r')
    expect(
      await screen.findByText(t.reviewSubtasksWarning(formatNumber(1), formatNumber(3))),
    ).toBeInTheDocument()
    expect(columnOf(result.current.tasks, 't-3')).toBe('c-r')
  })

  it('việc con đã xong hết: toast thường', async () => {
    const tasks = withSubtasks(2, 2)
    mockMove(tasks)
    const { result } = setup({ tasks, columns, transitions })
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-3', 'c-r')
    expect(await screen.findByText(t.moved('Chờ nghiệm thu'))).toBeInTheDocument()
  })

  it('hoàn tác đưa việc cha trở lại cột chờ nghiệm thu: vẫn hiện cảnh báo việc con', async () => {
    const tasks = withSubtasks(1, 3).map((x) =>
      x.id === 't-3' ? { ...x, column: review, position: 0 } : x,
    )
    const both: Transition[] = [
      { fromColumnId: 'c-r', toColumnId: 'c-2', role: 'manager' },
      { fromColumnId: 'c-2', toColumnId: 'c-r', role: 'manager' },
    ]
    server.use(
      mock.post(`${API}/api/v1/tasks/t-3/move`, async ({ request }) => {
        const body = (await request.json()) as { columnId: string; version: number }
        const to = columns.find((c) => c.id === body.columnId)
        const t3 = tasks.find((x) => x.id === 't-3')
        return HttpResponse.json({
          ...t3,
          column: to && { id: to.id, name: to.name, statusGroup: to.statusGroup },
          position: 0,
          version: body.version + 1,
        })
      }),
    )
    const { result } = setup({ tasks, columns, transitions: both })
    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(3)
    })
    dragTo(result, 't-3', 'c-2')
    await userEvent.setup().click(await screen.findByRole('button', { name: t.undo }))
    expect(
      await screen.findByText(t.reviewSubtasksWarning(formatNumber(1), formatNumber(3))),
    ).toBeInTheDocument()
  })
})
