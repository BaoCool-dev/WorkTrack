import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { Task } from '../model/types'
import { makeTask, page } from '../test-data'

const t = messages.tasks
const PATH = '/my-tasks'

let current: Session | undefined
function setup(role: Role = 'member') {
  const r = renderApp(PATH, role)
  current = r.session
  return r
}
afterEach(() => {
  current?.dispose()
  current = undefined
})

const mine = makeTask({
  assignee: { id: 'u-member', name: 'Đỗ Thu Hà' },
  column: { id: 'c-2', name: 'Đang làm', statusGroup: 'in_progress' },
})

function mockMine(tasks: Task[], total = tasks.length, queries: URLSearchParams[] = []) {
  server.use(
    mock.get(`${API}/api/v1/me/tasks`, ({ request }) => {
      queries.push(new URL(request.url).searchParams)
      return HttpResponse.json(page(tasks, total))
    }),
    mock.get(`${API}/api/v1/me/wip`, () => HttpResponse.json({ count: 1 })),
    mock.get(`${API}/api/v1/tasks/t-1`, () => HttpResponse.json(mine)),
  )
  return queries
}

describe('Trang Công việc của tôi (CF-06)', () => {
  it('hiện tiêu đề, chip WIP và bảng có cột Dự án', async () => {
    mockMine([mine])
    setup()
    expect(await screen.findByRole('heading', { name: t.mine.title })).toBeInTheDocument()
    expect(await screen.findByText(t.wip('1'))).toBeInTheDocument()
    const table = await screen.findByRole('table', { name: t.list.tableLabel })
    expect(within(table).getByText('Cổng thông tin khách hàng')).toBeInTheDocument()
    expect(within(table).getByText('Đang làm')).toBeInTheDocument()
  })

  it('trạng thái rỗng có lối tới Dự án', async () => {
    mockMine([])
    setup()
    expect(await screen.findByText(t.mine.emptyTitle)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: t.mine.goProjects })).toHaveAttribute(
      'href',
      '/projects',
    )
  })

  it('lỗi tải hiện ErrorState', async () => {
    server.use(
      mock.get(`${API}/api/v1/me/tasks`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
      mock.get(`${API}/api/v1/me/wip`, () => HttpResponse.json({ count: 0 })),
    )
    setup()
    expect(await screen.findByText(t.mine.loadError)).toBeInTheDocument()
  })

  it('lọc nhóm trạng thái và phân trang gửi đúng query', async () => {
    const queries = mockMine([mine], 45)
    setup()
    await screen.findByRole('table', { name: t.list.tableLabel })
    const user = userEvent.setup()
    await user.selectOptions(screen.getByLabelText(t.list.filterGroup), 'in_progress')
    await waitFor(() => {
      expect(queries.at(-1)?.get('statusGroup')).toBe('in_progress')
    })
    await user.click(screen.getByRole('button', { name: messages.pagination.next }))
    await waitFor(() => {
      expect(queries.at(-1)?.get('page')).toBe('2')
    })
  })

  it('bấm tên công việc mở chi tiết', async () => {
    mockMine([mine])
    setup()
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: t.list.open('Thiết kế API đăng nhập') }),
    )
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })

  it.each<Role>(['manager', 'member'])('%s vào được trang', async (role) => {
    mockMine([mine])
    setup(role)
    expect(await screen.findByRole('heading', { name: t.mine.title })).toBeInTheDocument()
  })
})
