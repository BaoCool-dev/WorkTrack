import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import { makeBoard, makeProject, MEMBERS } from '../../projects/test-data'
import type { Task, TaskInput } from '../model/types'
import { LABELS, makeTask, page } from '../test-data'

const t = messages.tasks
const PATH = '/projects/p-1?tab=tasks'

let current: Session | undefined
function setup(role: Role) {
  const r = renderApp(PATH, role)
  current = r.session
  return r
}
afterEach(() => {
  current?.dispose()
  current = undefined
})

function mockProject(tasks: Task[], queries: URLSearchParams[] = []) {
  server.use(
    mock.get(`${API}/api/v1/projects/p-1`, () => HttpResponse.json(makeProject())),
    mock.get(`${API}/api/v1/projects/p-1/members`, () => HttpResponse.json({ items: MEMBERS })),
    mock.get(`${API}/api/v1/projects/p-1/board`, () => HttpResponse.json(makeBoard())),
    mock.get(`${API}/api/v1/projects/p-1/tasks`, ({ request }) => {
      queries.push(new URL(request.url).searchParams)
      return HttpResponse.json(page(tasks))
    }),
    mock.get(`${API}/api/v1/labels`, () => HttpResponse.json({ items: LABELS })),
    mock.get(`${API}/api/v1/settings`, () =>
      HttpResponse.json({ storyPointHours: 4, updatedAt: '2026-11-20T02:00:00Z' }),
    ),
  )
  return queries
}

describe('Tab Công việc của dự án (CF-06)', () => {
  it('hiện bảng công việc: cột, Story Point, giờ ước tính, hạn chót, nhãn', async () => {
    mockProject([makeTask()])
    setup('manager')
    const table = await screen.findByRole('table', { name: t.list.tableLabel })
    expect(
      within(table).getByRole('button', { name: t.list.open('Thiết kế API đăng nhập') }),
    ).toBeInTheDocument()
    expect(within(table).getByText('Backlog')).toBeInTheDocument()
    expect(within(table).getByText('5 SP')).toBeInTheDocument()
    expect(within(table).getByText('20h')).toBeInTheDocument()
    expect(within(table).getByText('15/12/2026')).toBeInTheDocument()
    expect(within(table).getByText('Backend')).toBeInTheDocument()
    expect(within(table).getByText(t.detail.unassigned)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.list.add })).toBeInTheDocument()
  })

  it.each<Role>(['member', 'ceo'])('%s không thấy nút Tạo công việc', async (role) => {
    mockProject([makeTask()])
    setup(role)
    await screen.findByRole('table', { name: t.list.tableLabel })
    expect(screen.queryByRole('button', { name: t.list.add })).not.toBeInTheDocument()
  })

  it('lọc theo nhóm trạng thái, người nhận "Chưa giao" và nhãn gửi đúng query', async () => {
    const queries = mockProject([makeTask()])
    setup('member')
    await screen.findByRole('table', { name: t.list.tableLabel })
    const user = userEvent.setup()
    await user.selectOptions(screen.getByLabelText(t.list.filterGroup), 'todo')
    await user.selectOptions(screen.getByLabelText(t.list.filterAssignee), 'none')
    await user.selectOptions(screen.getByLabelText(t.list.filterLabel), 'l-1')
    await waitFor(() => {
      const q = queries.at(-1)
      expect(q?.get('statusGroup')).toBe('todo')
      expect(q?.get('unassigned')).toBe('true')
      expect(q?.get('labelId')).toBe('l-1')
      expect(q?.get('page')).toBe('1')
    })
  })

  it('rỗng: Manager thấy hướng dẫn tạo việc', async () => {
    mockProject([])
    setup('manager')
    expect(await screen.findByText(t.list.emptyTitle)).toBeInTheDocument()
    expect(screen.getByText(t.list.emptyHintManager)).toBeInTheDocument()
  })

  it('rỗng: Member thấy "Backlog đang trống" và liên kết Xem công việc của tôi', async () => {
    mockProject([])
    setup('member')
    expect(await screen.findByText(t.list.emptyTitle)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: t.list.viewMine })).toHaveAttribute('href', '/my-tasks')
    expect(screen.queryByRole('button', { name: t.list.add })).not.toBeInTheDocument()
  })

  it('form tạo việc: danh mục nhãn lỗi tải thì báo rõ, vẫn gửi được form', async () => {
    mockProject([])
    server.use(
      mock.get(`${API}/api/v1/labels`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    setup('manager')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: t.list.add }))
    const dialog = await screen.findByRole('dialog', { name: t.form.createTitle })
    expect(await within(dialog).findByText(t.form.labelsError)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: t.form.create })).toBeEnabled()
  })

  it('lỗi tải hiện mã lỗi và nút Thử lại', async () => {
    mockProject([])
    server.use(
      mock.get(`${API}/api/v1/projects/p-1/tasks`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    setup('manager')
    expect(await screen.findByText(t.list.loadError)).toBeInTheDocument()
    expect(screen.getByText('INTERNAL', { exact: false })).toBeInTheDocument()
  })

  it('Manager tạo công việc: gửi đúng body và hiện toast', async () => {
    mockProject([])
    let body: TaskInput | undefined
    server.use(
      mock.post(`${API}/api/v1/projects/p-1/tasks`, async ({ request }) => {
        body = (await request.json()) as TaskInput
        return HttpResponse.json(makeTask({ id: 't-9', title: body.title }), { status: 201 })
      }),
    )
    setup('manager')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: t.list.add }))
    const dialog = await screen.findByRole('dialog', { name: t.form.createTitle })
    await user.type(within(dialog).getByLabelText(t.form.title), '  Viết API nhận việc  ')
    await user.selectOptions(within(dialog).getByLabelText(t.form.storyPoint), '8')
    expect(within(dialog).getByText(t.form.estimateHint('32h', '4h'))).toBeInTheDocument()
    await user.selectOptions(within(dialog).getByLabelText(t.form.assignee), 'u-2')
    await user.click(within(dialog).getByRole('checkbox', { name: 'Frontend' }))
    await user.click(within(dialog).getByRole('button', { name: t.form.create }))
    await waitFor(() => {
      expect(body).toEqual({
        title: 'Viết API nhận việc',
        description: '',
        storyPoint: 8,
        assigneeId: 'u-2',
        dueDate: null,
        labelIds: ['l-2'],
      })
    })
    expect(await screen.findByText(t.form.created)).toBeInTheDocument()
  })

  it('tên trống báo lỗi ngay, không gửi request', async () => {
    mockProject([])
    let called = false
    server.use(
      mock.post(`${API}/api/v1/projects/p-1/tasks`, () => {
        called = true
        return HttpResponse.json(makeTask())
      }),
    )
    setup('manager')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: t.list.add }))
    const dialog = await screen.findByRole('dialog', { name: t.form.createTitle })
    await user.click(within(dialog).getByRole('button', { name: t.form.create }))
    expect(await within(dialog).findByText(t.form.titleRequired)).toBeInTheDocument()
    expect(called).toBe(false)
  })

  it('lỗi người nhận từ server gắn vào ô Người nhận', async () => {
    mockProject([])
    server.use(
      mock.post(`${API}/api/v1/projects/p-1/tasks`, () =>
        HttpResponse.json(
          { error: { code: 'TASK_ASSIGNEE_INVALID', message: 'Người nhận không thuộc dự án.' } },
          { status: 422 },
        ),
      ),
    )
    setup('manager')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: t.list.add }))
    const dialog = await screen.findByRole('dialog', { name: t.form.createTitle })
    await user.type(within(dialog).getByLabelText(t.form.title), 'Việc A')
    await user.click(within(dialog).getByRole('button', { name: t.form.create }))
    expect(await within(dialog).findByText('Người nhận không thuộc dự án.')).toBeInTheDocument()
    expect(within(dialog).getByLabelText(t.form.assignee)).toHaveAttribute('aria-invalid', 'true')
  })
})

describe('Chi tiết công việc (CF-06)', () => {
  function mockDetail(task: Task) {
    server.use(mock.get(`${API}/api/v1/tasks/${task.id}`, () => HttpResponse.json(task)))
  }

  it('mở chi tiết từ bảng: hiện thông tin và mục giờ làm thực tế', async () => {
    const task = makeTask({ description: 'Đăng nhập bằng email + mật khẩu.' })
    mockProject([task])
    mockDetail(task)
    setup('ceo')
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: t.list.open('Thiết kế API đăng nhập') }),
    )
    const dialog = await screen.findByRole('dialog', { name: 'Thiết kế API đăng nhập' })
    expect(await within(dialog).findByText('Đăng nhập bằng email + mật khẩu.')).toBeInTheDocument()
    expect(within(dialog).getByText('Trần Bình')).toBeInTheDocument()
    expect(within(dialog).getByText(t.timeLog.title)).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: t.detail.edit })).not.toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: t.detail.claim })).not.toBeInTheDocument()
  })

  it('Member nhận việc chưa giao ở nhóm Chưa làm', async () => {
    const task = makeTask()
    mockProject([task])
    mockDetail(task)
    let claimed = false
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/claim`, () => {
        claimed = true
        return HttpResponse.json(makeTask({ assignee: { id: 'u-member', name: 'Đỗ Thu Hà' } }))
      }),
    )
    setup('member')
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: t.list.open('Thiết kế API đăng nhập') }),
    )
    const dialog = await screen.findByRole('dialog', { name: 'Thiết kế API đăng nhập' })
    await user.click(await within(dialog).findByRole('button', { name: t.detail.claim }))
    await waitFor(() => {
      expect(claimed).toBe(true)
    })
    expect(await screen.findByText(t.detail.claimed)).toBeInTheDocument()
    expect(await within(dialog).findByText('Đỗ Thu Hà')).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: t.detail.claim })).not.toBeInTheDocument()
  })

  it('không hiện Nhận việc khi việc đã có người nhận hoặc không ở nhóm Chưa làm', async () => {
    const task = makeTask({
      column: { id: 'c-2', name: 'In Progress', statusGroup: 'in_progress' },
    })
    mockProject([task])
    mockDetail(task)
    setup('member')
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: t.list.open('Thiết kế API đăng nhập') }),
    )
    const dialog = await screen.findByRole('dialog', { name: 'Thiết kế API đăng nhập' })
    await within(dialog).findByText(t.timeLog.title)
    expect(within(dialog).queryByRole('button', { name: t.detail.claim })).not.toBeInTheDocument()
  })

  it('nhận việc bị người khác nhận trước: hiện lỗi trong hộp thoại', async () => {
    const task = makeTask()
    mockProject([task])
    mockDetail(task)
    server.use(
      mock.post(`${API}/api/v1/tasks/t-1/claim`, () =>
        HttpResponse.json(
          { error: { code: 'TASK_ALREADY_ASSIGNED', message: 'Công việc đã có người nhận.' } },
          { status: 409 },
        ),
      ),
    )
    setup('member')
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: t.list.open('Thiết kế API đăng nhập') }),
    )
    const dialog = await screen.findByRole('dialog', { name: 'Thiết kế API đăng nhập' })
    await user.click(await within(dialog).findByRole('button', { name: t.detail.claim }))
    expect(await within(dialog).findByText('Công việc đã có người nhận.')).toBeInTheDocument()
  })

  it('Manager phụ trách sửa công việc: điền sẵn giá trị và gửi PUT', async () => {
    const task = makeTask()
    mockProject([task])
    mockDetail(task)
    let body: TaskInput | undefined
    server.use(
      mock.put(`${API}/api/v1/tasks/t-1`, async ({ request }) => {
        body = (await request.json()) as TaskInput
        return HttpResponse.json(makeTask({ title: body.title }))
      }),
    )
    setup('manager')
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: t.list.open('Thiết kế API đăng nhập') }),
    )
    const detail = await screen.findByRole('dialog', { name: 'Thiết kế API đăng nhập' })
    await user.click(await within(detail).findByRole('button', { name: t.detail.edit }))
    const form = await screen.findByRole('dialog', { name: t.form.editTitle })
    const title = within(form).getByLabelText(t.form.title)
    expect(title).toHaveValue('Thiết kế API đăng nhập')
    expect(within(form).getByLabelText(t.form.storyPoint)).toHaveValue('5')
    expect(within(form).getByRole('checkbox', { name: 'Backend' })).toBeChecked()
    await user.clear(title)
    await user.type(title, 'Thiết kế API đăng nhập v2')
    await user.click(within(form).getByRole('button', { name: t.form.save }))
    await waitFor(() => {
      expect(body).toEqual({
        title: 'Thiết kế API đăng nhập v2',
        description: 'Endpoint đăng nhập + refresh token.',
        storyPoint: 5,
        assigneeId: null,
        dueDate: '2026-12-15',
        labelIds: ['l-1'],
      })
    })
    expect(await screen.findByText(t.form.updated)).toBeInTheDocument()
  })
})
