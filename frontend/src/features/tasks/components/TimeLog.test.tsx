import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { makeBoard } from '@/features/projects/test-data'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { Task, TimeLog } from '../model/types'
import { makeSubtask, makeTask, makeTimeLog, page } from '../test-data'

// CF-08 — mục giờ làm thực tế, hộp Log giờ và lịch sử trong chi tiết công việc.
const t = messages.tasks
const tl = t.timeLog

let current: Session | undefined
afterEach(() => {
  current?.dispose()
  current = undefined
})

const ME = { id: 'u-member', name: 'Đỗ Thu Hà' }
const parent = makeTask({
  assignee: ME,
  estimatedMinutes: 960,
  actualMinutes: 870,
  column: { id: 'c-2', name: 'In Progress', statusGroup: 'in_progress' },
})

interface Mocks {
  task?: Task
  subtasks?: Task[]
  logs?: TimeLog[]
}

function mockAll({ task = parent, subtasks = [], logs = [] }: Mocks = {}) {
  server.use(
    mock.get(`${API}/api/v1/me/tasks`, () => HttpResponse.json(page([task]))),
    mock.get(`${API}/api/v1/me/wip`, () => HttpResponse.json({ count: 1 })),
    mock.get(`${API}/api/v1/tasks/${task.id}`, () => HttpResponse.json(task)),
    mock.get(`${API}/api/v1/projects/p-1/board`, () => HttpResponse.json(makeBoard())),
    mock.get(`${API}/api/v1/tasks/${task.id}/subtasks`, () =>
      HttpResponse.json({ items: subtasks }),
    ),
    mock.get(`${API}/api/v1/tasks/${task.id}/comments`, () => HttpResponse.json(page([]))),
    mock.get(`${API}/api/v1/tasks/${task.id}/time-logs`, () => HttpResponse.json(page(logs))),
  )
}

async function openDetail(role: Role = 'member') {
  const r = renderApp('/my-tasks', role)
  current = r.session
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: t.list.open(parent.title) }))
  const dialog = await screen.findByRole('dialog')
  return { user, dialog }
}

async function openLogDialog(role: Role = 'member') {
  const { user, dialog } = await openDetail(role)
  await user.click(await within(dialog).findByRole('button', { name: tl.log }))
  const form = await screen.findByRole('dialog', { name: tl.dialogTitle })
  return { user, form }
}

describe('Giờ làm thực tế trong chi tiết công việc (CF-08)', () => {
  it('hiện giờ thực tế/ước tính kèm ký hiệu mức', async () => {
    mockAll()
    const { dialog } = await openDetail()
    expect(await within(dialog).findByText('14,5/16h')).toBeInTheDocument()
    expect(within(dialog).getByRole('img', { name: messages.kanban.hoursLevel.ok })).toBeVisible()
  })

  it('người nhận việc thấy nút Log giờ; CEO thì không', async () => {
    mockAll()
    const { dialog } = await openDetail('member')
    expect(await within(dialog).findByRole('button', { name: tl.log })).toBeInTheDocument()
  })

  it('CEO chỉ xem, không có nút Log giờ', async () => {
    mockAll({ task: makeTask({ ...parent, assignee: { id: 'u-x', name: 'Lê Minh' } }) })
    const { dialog } = await openDetail('ceo')
    await within(dialog).findByText('14,5/16h')
    expect(within(dialog).queryByRole('button', { name: tl.log })).not.toBeInTheDocument()
  })
})

describe('Hộp Log giờ (CF-08)', () => {
  it('số giờ bằng 0 hoặc âm: báo lỗi dưới ô, không gửi', async () => {
    mockAll()
    let posted = false
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/time-logs`, () => {
        posted = true
        return HttpResponse.json(makeTimeLog(), { status: 201 })
      }),
    )
    const { user, form } = await openLogDialog()
    await user.type(within(form).getByLabelText(tl.hours), '-2')
    await user.type(within(form).getByLabelText(tl.note), 'Sửa lỗi')
    await user.click(within(form).getByRole('button', { name: tl.submit }))
    expect(await within(form).findByText(tl.negative)).toBeInTheDocument()
    expect(within(form).getByLabelText(tl.hours)).toHaveAttribute('aria-invalid', 'true')
    expect(posted).toBe(false)
  })

  it('ghi 1,5 giờ = 90 phút, báo toast và trở về chi tiết', async () => {
    mockAll()
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/time-logs`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeTimeLog(), { status: 201 })
      }),
    )
    const { user, form } = await openLogDialog()
    await user.type(within(form).getByLabelText(tl.hours), '1,5')
    await user.type(within(form).getByLabelText(tl.note), 'Viết handler và test.')
    await user.click(within(form).getByRole('button', { name: tl.submit }))
    await waitFor(() => {
      expect(body).toMatchObject({ minutes: 90, note: 'Viết handler và test.' })
    })
    expect((body as { workDate: string }).workDate).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    expect(await screen.findByText(tl.saved)).toBeInTheDocument()
    expect(await screen.findByRole('dialog', { name: parent.title })).toBeInTheDocument()
  })

  it('chọn việc con làm đích: gửi vào việc con', async () => {
    mockAll({ subtasks: [makeSubtask({ assignee: ME })] })
    let hit = false
    server.use(
      mock.post(`${API}/api/v1/tasks/s-1/time-logs`, () => {
        hit = true
        return HttpResponse.json(makeTimeLog({ taskId: 's-1' }), { status: 201 })
      }),
    )
    const { user, form } = await openLogDialog()
    const target = within(form).getByLabelText(tl.target)
    await within(form).findByRole('option', { name: tl.targetSubtask('Viết handler đăng nhập') })
    await user.selectOptions(target, 's-1')
    await user.type(within(form).getByLabelText(tl.hours), '2')
    await user.type(within(form).getByLabelText(tl.note), 'Xong handler')
    await user.click(within(form).getByRole('button', { name: tl.submit }))
    await waitFor(() => {
      expect(hit).toBe(true)
    })
  })

  it('việc đã hoàn thành quá 24 giờ: thay form bằng khối khóa', async () => {
    mockAll()
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/time-logs`, () =>
        HttpResponse.json(
          { error: { code: 'TIME_LOG_CLOSED', message: 'Công việc đã đóng.' } },
          { status: 409 },
        ),
      ),
    )
    const { user, form } = await openLogDialog()
    await user.type(within(form).getByLabelText(tl.hours), '1')
    await user.type(within(form).getByLabelText(tl.note), 'Bổ sung')
    await user.click(within(form).getByRole('button', { name: tl.submit }))
    expect(await within(form).findByText(tl.closedTitle)).toBeInTheDocument()
    expect(within(form).queryByLabelText(tl.hours)).not.toBeInTheDocument()
  })
})

describe('Lịch sử Log giờ (CF-08)', () => {
  it('liệt kê log, log của việc con có ghi chú tên việc con', async () => {
    mockAll({
      logs: [
        makeTimeLog(),
        makeTimeLog({
          id: 'tl-2',
          taskId: 's-1',
          taskTitle: 'Viết handler đăng nhập',
          parentId: 't-1',
          minutes: 120,
          note: 'Test handler',
        }),
      ],
    })
    const { dialog } = await openDetail()
    const list = await within(dialog).findByRole('list', { name: tl.listLabel })
    expect(within(list).getByText('Viết handler và test.')).toBeInTheDocument()
    expect(within(list).getByText('1,5h')).toBeInTheDocument()
    expect(within(list).getByText(tl.onSubtask('Viết handler đăng nhập'))).toBeInTheDocument()
  })

  it('trạng thái rỗng khi chưa có log', async () => {
    mockAll()
    const { dialog } = await openDetail()
    expect(await within(dialog).findByText(tl.emptyTitle)).toBeInTheDocument()
  })

  it('chỉ xóa được log của chính mình, có xác nhận', async () => {
    mockAll({
      logs: [
        makeTimeLog(),
        makeTimeLog({ id: 'tl-2', user: { id: 'u-manager', name: 'Trần Bình' }, note: 'Review' }),
      ],
    })
    let deleted = ''
    server.use(
      mock.delete(`${API}/api/v1/time-logs/:id`, ({ params }) => {
        deleted = String(params.id)
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const { user, dialog } = await openDetail()
    const list = await within(dialog).findByRole('list', { name: tl.listLabel })
    const buttons = within(list).getAllByRole('button', { name: /^Xóa log giờ/ })
    expect(buttons).toHaveLength(1)
    await user.click(buttons[0])
    await user.click(within(list).getByRole('button', { name: tl.confirm }))
    await waitFor(() => {
      expect(deleted).toBe('tl-1')
    })
    expect(await screen.findByText(tl.deleted)).toBeInTheDocument()
  })
})
