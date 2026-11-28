import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import type { Board } from '@/features/projects'
import { MEMBERS, makeProject } from '@/features/projects/test-data'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { Task, TaskReview } from '../model/types'
import { makeReview, makeTask, page } from '../test-data'

// CF-09 — nút Nghiệm thu / Từ chối và lịch sử nghiệm thu trong chi tiết công việc.
const t = messages.tasks
const tr = t.review

let current: Session | undefined
afterEach(() => {
  current?.dispose()
  current = undefined
})

/** Bảng có cột "Chờ nghiệm thu" = cột in_progress cuối cùng (c-3), giống backend ReviewTargets. */
function reviewBoard(): Board {
  return {
    columns: [
      { id: 'c-1', name: 'Backlog', position: 0, statusGroup: 'todo', taskCount: 0 },
      { id: 'c-2', name: 'Đang làm', position: 1, statusGroup: 'in_progress', taskCount: 0 },
      { id: 'c-3', name: 'Chờ nghiệm thu', position: 2, statusGroup: 'in_progress', taskCount: 1 },
      { id: 'c-4', name: 'Hoàn thành', position: 3, statusGroup: 'done', taskCount: 0 },
    ],
    transitions: [],
  }
}

const inReview = makeTask({
  assignee: { id: 'u-2', name: 'Đỗ Thu Hà' },
  column: { id: 'c-3', name: 'Chờ nghiệm thu', statusGroup: 'in_progress' },
})

function mockAll(task: Task = inReview, reviews: TaskReview[] = []) {
  server.use(
    mock.get(`${API}/api/v1/projects/p-1`, () => HttpResponse.json(makeProject())),
    mock.get(`${API}/api/v1/projects/p-1/board`, () => HttpResponse.json(reviewBoard())),
    mock.get(`${API}/api/v1/projects/p-1/members`, () => HttpResponse.json({ items: MEMBERS })),
    mock.get(`${API}/api/v1/projects/p-1/tasks`, () => HttpResponse.json(page([task]))),
    mock.get(`${API}/api/v1/labels`, () => HttpResponse.json({ items: [] })),
    mock.get(`${API}/api/v1/tasks/${task.id}`, () => HttpResponse.json(task)),
    mock.get(`${API}/api/v1/tasks/${task.id}/subtasks`, () => HttpResponse.json({ items: [] })),
    mock.get(`${API}/api/v1/tasks/${task.id}/comments`, () => HttpResponse.json(page([]))),
    mock.get(`${API}/api/v1/tasks/${task.id}/time-logs`, () => HttpResponse.json(page([]))),
    mock.get(`${API}/api/v1/tasks/${task.id}/reviews`, () => HttpResponse.json({ items: reviews })),
  )
}

async function openDetail(role: Role = 'manager', task: Task = inReview) {
  const r = renderApp('/projects/p-1?tab=tasks', role)
  current = r.session
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: t.list.open(task.title) }))
  const dialog = await screen.findByRole('dialog')
  return { user, dialog }
}

describe('Nút nghiệm thu theo vai trò (CF-09)', () => {
  it('Manager phụ trách thấy Nghiệm thu và Từ chối khi việc ở cột chờ nghiệm thu', async () => {
    mockAll()
    const { dialog } = await openDetail('manager')
    expect(await within(dialog).findByRole('button', { name: tr.approve })).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: tr.reject })).toBeInTheDocument()
  })

  it.each<Role>(['member', 'ceo'])('%s không thấy nút nghiệm thu', async (role) => {
    mockAll()
    const { dialog } = await openDetail(role)
    await within(dialog).findByText(tr.history)
    expect(within(dialog).queryByRole('button', { name: tr.approve })).not.toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: tr.reject })).not.toBeInTheDocument()
  })

  it('việc chưa ở cột chờ nghiệm thu: không có nút', async () => {
    const doing = makeTask({ column: { id: 'c-2', name: 'Đang làm', statusGroup: 'in_progress' } })
    mockAll(doing)
    const { dialog } = await openDetail('manager', doing)
    await within(dialog).findByText(tr.history)
    expect(within(dialog).queryByRole('button', { name: tr.approve })).not.toBeInTheDocument()
  })

  it('còn việc con chưa xong: chỉ cảnh báo, vẫn cho nghiệm thu', async () => {
    const pending = makeTask({ ...inReview, subtaskCount: 3, subtaskDoneCount: 1 })
    mockAll(pending)
    const { dialog } = await openDetail('manager', pending)
    expect(await within(dialog).findByText(tr.subtasksPending('1', '3'))).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: tr.approve })).toBeEnabled()
  })
})

describe('Duyệt / Từ chối (CF-09)', () => {
  it('Nghiệm thu gửi version và báo toast', async () => {
    mockAll()
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/review/approve`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(
          makeTask({ ...inReview, column: { id: 'c-4', name: 'Hoàn thành', statusGroup: 'done' } }),
        )
      }),
    )
    const { user, dialog } = await openDetail()
    await user.click(await within(dialog).findByRole('button', { name: tr.approve }))
    expect(await screen.findByText(tr.approved)).toBeInTheDocument()
    expect(body).toEqual({ version: inReview.version })
  })

  it('Nghiệm thu bị xung đột: hiện lỗi từ server trong hộp chi tiết', async () => {
    mockAll()
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/review/approve`, () =>
        HttpResponse.json(
          {
            error: { code: 'TASK_VERSION_CONFLICT', message: 'Công việc vừa được người khác sửa.' },
          },
          { status: 409 },
        ),
      ),
    )
    const { user, dialog } = await openDetail()
    await user.click(await within(dialog).findByRole('button', { name: tr.approve }))
    expect(await within(dialog).findByRole('alert')).toBeInTheDocument()
  })

  it('Từ chối không có lý do: báo lỗi, không gửi', async () => {
    mockAll()
    let posted = false
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/review/reject`, () => {
        posted = true
        return HttpResponse.json(inReview)
      }),
    )
    const { user, dialog } = await openDetail()
    await user.click(await within(dialog).findByRole('button', { name: tr.reject }))
    const form = await screen.findByRole('dialog', { name: tr.dialogTitle })
    await user.click(within(form).getByRole('button', { name: tr.confirmReject }))
    expect(await within(form).findByText(tr.reasonRequired)).toBeInTheDocument()
    expect(posted).toBe(false)
  })

  it('Từ chối kèm lý do gửi version + reason và báo toast', async () => {
    mockAll()
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/review/reject`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(
          makeTask({
            ...inReview,
            column: { id: 'c-2', name: 'Đang làm', statusGroup: 'in_progress' },
          }),
        )
      }),
    )
    const { user, dialog } = await openDetail()
    await user.click(await within(dialog).findByRole('button', { name: tr.reject }))
    const form = await screen.findByRole('dialog', { name: tr.dialogTitle })
    await user.type(within(form).getByLabelText(tr.reason), '  Thiếu test API.  ')
    await user.click(within(form).getByRole('button', { name: tr.confirmReject }))
    expect(await screen.findByText(tr.rejected)).toBeInTheDocument()
    await waitFor(() => {
      expect(body).toEqual({ version: inReview.version, reason: 'Thiếu test API.' })
    })
  })
})

describe('Lịch sử nghiệm thu (CF-09)', () => {
  it('hiện badge quyết định và lý do từ chối', async () => {
    mockAll(inReview, [
      makeReview(),
      makeReview({ id: 'rv-2', decision: 'approved', reason: null }),
    ])
    const { dialog } = await openDetail('member')
    expect(await within(dialog).findByText(tr.decisionRejected)).toBeInTheDocument()
    expect(within(dialog).getByText(tr.decisionApproved)).toBeInTheDocument()
    expect(within(dialog).getByText('Thiếu test.')).toBeInTheDocument()
  })

  it('chưa có lần nghiệm thu nào: hiện câu rỗng', async () => {
    mockAll()
    const { dialog } = await openDetail('member')
    expect(await within(dialog).findByText(tr.historyEmpty)).toBeInTheDocument()
  })
})
