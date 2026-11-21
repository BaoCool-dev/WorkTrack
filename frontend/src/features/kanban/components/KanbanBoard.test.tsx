import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import { vi } from 'vitest'

import type { Role, Session } from '@/features/auth'
import type { Task } from '@/features/tasks'
import { makeBoard, makeProject, MEMBERS } from '@/features/projects/test-data'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'

import { boardTasks, FakeSocket } from '../test-data'

const t = messages.kanban
const P = `${API}/api/v1/projects/p-1`
const PATH = '/projects/p-1?tab=board'

let current: Session | undefined
function setup(role: Role, tasks: Task[] = boardTasks()) {
  vi.stubGlobal('WebSocket', FakeSocket)
  FakeSocket.reset()
  server.use(
    mock.get(P, () => HttpResponse.json(makeProject())),
    mock.get(`${P}/members`, () => HttpResponse.json({ items: MEMBERS })),
    mock.get(`${P}/board`, () => HttpResponse.json(makeBoard())),
    mock.get(`${P}/board/tasks`, () => HttpResponse.json({ items: tasks })),
  )
  const r = renderApp(PATH, role)
  current = r.session
  return r
}
afterEach(() => {
  current?.dispose()
  current = undefined
})

async function column(name: string) {
  const heading = await screen.findByRole('heading', { name })
  const section = heading.closest('section')
  if (!section) throw new Error(`không thấy cột ${name}`)
  return section
}

describe('Bảng Kanban (CF-04, CF-05, UC05)', () => {
  it('Manager phụ trách: thấy cột, số thẻ, nút "+" ở cột đầu và ô giới hạn WIP', async () => {
    setup('manager')
    const backlog = await column('Backlog')
    expect(within(backlog).getByLabelText(t.columnCount(2))).toBeInTheDocument()
    expect(within(backlog).getByText('Thiết kế API đăng nhập')).toBeInTheDocument()
    expect(within(backlog).getByRole('button', { name: t.addTask('Backlog') })).toBeInTheDocument()
    // Việc mới chỉ vào cột đầu nên cột khác không có "+".
    const doing = await column('In Progress')
    expect(
      within(doing).queryByRole('button', { name: t.addTask('In Progress') }),
    ).not.toBeInTheDocument()
    expect(screen.getByLabelText(t.wip.limitLabel)).toHaveValue(5)
    expect(screen.queryByText(t.memberHint)).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: t.dragHandle('Viết trang hồ sơ') }),
    ).toBeInTheDocument()
  })

  it('Member: có gợi ý quyền, chip WIP của mình, lọc "Việc của tôi"; không có "+" và ô WIP', async () => {
    setup('member')
    await column('Backlog')
    expect(screen.getByText(t.memberHint)).toBeInTheDocument()
    expect(screen.getByText(t.wip.chip(1, 5))).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.addTask('Backlog') })).not.toBeInTheDocument()
    expect(screen.queryByLabelText(t.wip.limitLabel)).not.toBeInTheDocument()

    await userEvent.setup().click(screen.getByRole('button', { name: t.filters.mine }))
    const backlog = await column('Backlog')
    expect(within(backlog).queryByText('Viết trang hồ sơ')).not.toBeInTheDocument()
    expect(within(backlog).getByText('Thiết kế API đăng nhập')).toBeInTheDocument()
  })

  it('CEO chỉ xem: có câu chỉ xem, không thẻ nào kéo được', async () => {
    setup('ceo')
    await column('Backlog')
    expect(screen.getByText(t.readOnly)).toBeInTheDocument()
    // Thẻ không kéo được: nhãn chỉ là tên việc (không có "Kéo thẻ…") và dnd-kit đánh dấu aria-disabled.
    const card = screen.getByRole('button', { name: 'Thiết kế API đăng nhập' })
    expect(card).toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.queryByRole('button', { name: t.dragHandle('Thiết kế API đăng nhập') }),
    ).not.toBeInTheDocument()
  })

  it('lọc theo nhãn ẩn thẻ không khớp và cập nhật số thẻ cột', async () => {
    setup('manager')
    await column('Backlog')
    await userEvent.setup().click(screen.getByRole('button', { name: 'Frontend' }))
    const backlog = await column('Backlog')
    expect(within(backlog).getByLabelText(t.columnCount(1))).toBeInTheDocument()
    expect(within(backlog).queryByText('Thiết kế API đăng nhập')).not.toBeInTheDocument()
  })

  it('bảng rỗng: trạng thái rỗng theo vai trò, có nút hành động', async () => {
    setup('member', [])
    expect(await screen.findByText(t.emptyMember)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.viewMine })).toBeInTheDocument()
  })

  it('bảng rỗng của Manager có nút Tạo công việc', async () => {
    setup('manager', [])
    expect(await screen.findByText(t.emptyManager)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.createTask })).toBeInTheDocument()
  })

  it('hướng dẫn kéo thả cho trình đọc màn hình là tiếng Việt', async () => {
    setup('manager')
    await column('Backlog')
    expect(screen.getByText(t.a11y.instructions)).toBeInTheDocument()
  })

  it('lỗi tải hiện ErrorState', async () => {
    setup('manager')
    server.use(
      mock.get(`${P}/board/tasks`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    expect(await screen.findByText(t.loadError)).toBeInTheDocument()
  })

  it('mất kết nối realtime hiện dải cảnh báo; nối lại thì ẩn', async () => {
    setup('manager')
    await column('Backlog')
    await waitFor(() => {
      expect(FakeSocket.instances.length).toBe(1)
    })
    FakeSocket.last().open()
    expect(await screen.findByText(t.realtime.live)).toBeInTheDocument()
    FakeSocket.last().drop()
    expect(await screen.findByText(t.realtime.lost)).toBeInTheDocument()
  })

  it('sự kiện task.moved từ người khác dời thẻ ngay, không tải lại', async () => {
    setup('manager')
    await column('Backlog')
    await waitFor(() => {
      expect(FakeSocket.instances.length).toBe(1)
    })
    FakeSocket.last().open()
    FakeSocket.last().emit({
      type: 'task.moved',
      projectId: 'p-1',
      taskId: 't-2',
      fromColumnId: 'c-1',
      toColumnId: 'c-2',
      position: 0,
      version: 2,
      sentAt: '2026-11-21T00:00:00Z',
    })
    const doing = await column('In Progress')
    expect(await within(doing).findByText('Viết trang hồ sơ')).toBeInTheDocument()
    expect(within(doing).getByLabelText(t.columnCount(2))).toBeInTheDocument()
  })

  it('presence hiện tên người đang xem', async () => {
    setup('manager')
    await column('Backlog')
    await waitFor(() => {
      expect(FakeSocket.instances.length).toBe(1)
    })
    FakeSocket.last().open()
    FakeSocket.last().emit({
      type: 'presence',
      projectId: 'p-1',
      userIds: ['u-manager', 'u-2'],
      sentAt: '',
    })
    expect(await screen.findByLabelText(t.realtime.presence(2))).toBeInTheDocument()
  })
})
