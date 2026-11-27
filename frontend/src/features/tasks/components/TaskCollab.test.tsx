import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { makeBoard } from '@/features/projects/test-data'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { Comment, Task } from '../model/types'
import { makeComment, makeSubtask, makeTask, page } from '../test-data'

// CF-07 — việc con và bình luận trong hộp chi tiết công việc (mở từ "Công việc của tôi").
const t = messages.tasks
const ts = t.subtasks
const tc = t.comments

let current: Session | undefined
afterEach(() => {
  current?.dispose()
  current = undefined
})

const ME = { id: 'u-member', name: 'Đỗ Thu Hà' }
const parent = makeTask({
  assignee: ME,
  column: { id: 'c-2', name: 'In Progress', statusGroup: 'in_progress' },
})

interface Mocks {
  task?: Task
  subtasks?: Task[]
  comments?: Comment[]
}

function mockAll({ task = parent, subtasks = [], comments = [] }: Mocks = {}) {
  server.use(
    mock.get(`${API}/api/v1/me/tasks`, () => HttpResponse.json(page([task]))),
    mock.get(`${API}/api/v1/me/wip`, () => HttpResponse.json({ count: 1 })),
    mock.get(`${API}/api/v1/tasks/${task.id}`, () => HttpResponse.json(task)),
    mock.get(`${API}/api/v1/projects/p-1/board`, () => HttpResponse.json(makeBoard())),
    mock.get(`${API}/api/v1/tasks/${task.id}/subtasks`, () =>
      HttpResponse.json({ items: subtasks }),
    ),
    mock.get(`${API}/api/v1/tasks/${task.id}/comments`, () => HttpResponse.json(page(comments))),
  )
}

async function openDetail(role: Role = 'member', title = parent.title) {
  const r = renderApp('/my-tasks', role)
  current = r.session
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: t.list.open(title) }))
  const dialog = await screen.findByRole('dialog')
  return { user, dialog }
}

describe('Công việc con trong chi tiết công việc (CF-07)', () => {
  it('hiện danh sách con và tiến độ số con đã xong', async () => {
    mockAll({
      subtasks: [
        makeSubtask({ assignee: ME }),
        makeSubtask({
          id: 's-2',
          title: 'Viết test đăng nhập',
          assignee: ME,
          column: { id: 'c-3', name: 'Done', statusGroup: 'done' },
        }),
      ],
    })
    const { dialog } = await openDetail()
    const list = await within(dialog).findByRole('list', { name: ts.listLabel })
    expect(within(list).getByText('Viết handler đăng nhập')).toBeInTheDocument()
    expect(within(list).getByText('Viết test đăng nhập')).toBeInTheDocument()
    expect(within(dialog).getByText(ts.progress('1', '2'))).toBeInTheDocument()
  })

  it('trạng thái rỗng khi chưa có việc con', async () => {
    mockAll()
    const { dialog } = await openDetail()
    expect(await within(dialog).findByText(ts.emptyTitle)).toBeInTheDocument()
  })

  it('người nhận việc cha thêm việc con — gửi POST, báo toast, không có Story Point', async () => {
    mockAll()
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/subtasks`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeSubtask({ title: 'Viết migration' }), { status: 201 })
      }),
    )
    const { user, dialog } = await openDetail()
    await user.click(await within(dialog).findByRole('button', { name: ts.add }))
    await user.click(within(dialog).getByRole('button', { name: ts.save }))
    expect(await within(dialog).findByText(ts.titleRequired)).toBeInTheDocument()

    await user.type(within(dialog).getByLabelText(ts.addTitle), '  Viết migration  ')
    await user.click(within(dialog).getByRole('button', { name: ts.save }))
    expect(await screen.findByText(ts.created)).toBeInTheDocument()
    expect(body).toEqual({ title: 'Viết migration', description: '' })
  })

  it('chuyển cột việc con của mình qua ô chọn', async () => {
    mockAll({ subtasks: [makeSubtask({ assignee: ME })] })
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/subtasks/s-1/move`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeSubtask({ assignee: ME }))
      }),
    )
    const { user, dialog } = await openDetail()
    const select = await within(dialog).findByLabelText(ts.column('Viết handler đăng nhập'))
    await user.selectOptions(select, 'c-3')
    expect(await screen.findByText(ts.moved)).toBeInTheDocument()
    expect(body).toEqual({ columnId: 'c-3' })
  })

  it('xóa việc con có bước xác nhận ngay trong danh sách', async () => {
    mockAll({ subtasks: [makeSubtask({ assignee: ME })] })
    let deleted = false
    server.use(
      mock.delete(`${API}/api/v1/subtasks/s-1`, () => {
        deleted = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const { user, dialog } = await openDetail()
    await user.click(
      await within(dialog).findByRole('button', { name: ts.remove('Viết handler đăng nhập') }),
    )
    expect(within(dialog).getByText(ts.removeTitle('Viết handler đăng nhập'))).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: ts.removeConfirm }))
    expect(await screen.findByText(ts.removed)).toBeInTheDocument()
    expect(deleted).toBe(true)
  })

  it('lỗi server khi thêm việc con hiện ngay trong khối', async () => {
    mockAll()
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/subtasks`, () =>
        HttpResponse.json(
          { error: { code: 'SUBTASK_FORBIDDEN', message: 'Bạn không được thêm việc con.' } },
          { status: 403 },
        ),
      ),
    )
    const { user, dialog } = await openDetail()
    await user.click(await within(dialog).findByRole('button', { name: ts.add }))
    await user.type(within(dialog).getByLabelText(ts.addTitle), 'Viết migration')
    await user.click(within(dialog).getByRole('button', { name: ts.save }))
    expect(await within(dialog).findByText('Bạn không được thêm việc con.')).toBeInTheDocument()
  })

  it('không phải người nhận: ẩn nút thêm, việc con chỉ hiện cột', async () => {
    const other = { id: 'u-other', name: 'Lê Minh Khoa' }
    const task = makeTask({ assignee: other })
    mockAll({ task, subtasks: [makeSubtask({ assignee: other })] })
    const { dialog } = await openDetail()
    const list = await within(dialog).findByRole('list', { name: ts.listLabel })
    expect(within(list).getByText('Backlog')).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: ts.add })).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText(ts.column('Viết handler đăng nhập'))).toBeNull()
    expect(within(dialog).getByText(ts.readOnly)).toBeInTheDocument()
  })

  it('chi tiết của một việc con không có khối việc con (chỉ 1 cấp)', async () => {
    const sub = makeSubtask({ assignee: ME })
    mockAll({ task: sub })
    const { dialog } = await openDetail('member', sub.title)
    expect(await within(dialog).findByRole('heading', { name: tc.title })).toBeInTheDocument()
    expect(within(dialog).queryByRole('heading', { name: ts.title })).not.toBeInTheDocument()
  })
})

describe('Bình luận trong chi tiết công việc (CF-07)', () => {
  it('hiện bình luận cũ nhất trước, kèm tác giả', async () => {
    mockAll({
      comments: [
        makeComment(),
        makeComment({ id: 'cm-2', author: ME, body: 'Đã thêm, anh xem giúp.' }),
      ],
    })
    const { dialog } = await openDetail()
    const list = await within(dialog).findByRole('list', { name: tc.listLabel })
    const items = within(list).getAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent('Nhớ thêm rate limit.')
    expect(items[0]).toHaveTextContent('Nguyễn Văn An')
    expect(items[1]).toHaveTextContent('Đã thêm, anh xem giúp.')
  })

  it('trạng thái rỗng khi chưa có bình luận', async () => {
    mockAll()
    const { dialog } = await openDetail()
    expect(await within(dialog).findByText(tc.emptyTitle)).toBeInTheDocument()
  })

  it('gửi bình luận: chặn rỗng, gửi POST, xóa ô nhập và báo toast', async () => {
    mockAll()
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/comments`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeComment({ author: ME, body: 'Xong phần API' }), {
          status: 201,
        })
      }),
    )
    const { user, dialog } = await openDetail()
    const box = await within(dialog).findByLabelText(tc.label)
    await user.click(within(dialog).getByRole('button', { name: tc.send }))
    expect(await within(dialog).findByText(tc.required)).toBeInTheDocument()

    await user.type(box, 'Xong phần API')
    expect(within(dialog).getByText(tc.counter('13'))).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: tc.send }))
    expect(await screen.findByText(tc.sent)).toBeInTheDocument()
    expect(body).toEqual({ body: 'Xong phần API' })
    await waitFor(() => {
      expect(box).toHaveValue('')
    })
  })

  it('lỗi tải bình luận hiện ErrorState', async () => {
    mockAll()
    server.use(
      mock.get(`${API}/api/v1/tasks/t-1/comments`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    const { dialog } = await openDetail()
    expect(await within(dialog).findByText(tc.loadError)).toBeInTheDocument()
  })
})
