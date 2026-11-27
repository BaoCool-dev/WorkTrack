import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { Project } from '../model/types'
import { makeProject } from '../test-data'

const t = messages.projects

let current: Session | undefined
function setup(path: string, role: Role) {
  const r = renderApp(path, role)
  current = r.session
  return r
}
afterEach(() => {
  current?.dispose()
  current = undefined
})

function mockList(items: Project[], queries: URLSearchParams[] = []) {
  server.use(
    mock.get(`${API}/api/v1/projects`, ({ request }) => {
      queries.push(new URL(request.url).searchParams)
      return HttpResponse.json({ items, page: 1, pageSize: 20, total: items.length })
    }),
  )
  return queries
}

describe('Danh sách dự án (CF-03)', () => {
  it('Manager thấy bảng dự án và nút Tạo dự án', async () => {
    mockList([makeProject()])
    setup('/projects', 'manager')
    const table = await screen.findByRole('table', { name: t.list.tableLabel })
    expect(within(table).getByRole('link', { name: 'Cổng thông tin khách hàng' })).toHaveAttribute(
      'href',
      '/projects/p-1',
    )
    expect(within(table).getByText(t.status.in_progress)).toBeInTheDocument()
    expect(within(table).getByText('01/12/2026', { exact: false })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.list.add })).toBeInTheDocument()
  })

  it.each<Role>(['member', 'ceo'])('%s không thấy nút Tạo dự án', async (role) => {
    mockList([makeProject()])
    setup('/projects', role)
    await screen.findByRole('table', { name: t.list.tableLabel })
    expect(screen.queryByRole('button', { name: t.list.add })).not.toBeInTheDocument()
  })

  it('trạng thái rỗng theo vai trò', async () => {
    mockList([])
    setup('/projects', 'member')
    expect(await screen.findByText(t.list.emptyTitle)).toBeInTheDocument()
    expect(screen.getByText(t.list.emptyHint)).toBeInTheDocument()
  })

  it('lỗi tải có nút Thử lại', async () => {
    server.use(
      mock.get(`${API}/api/v1/projects`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'Lỗi' } }, { status: 500 }),
      ),
    )
    setup('/projects', 'manager')
    expect(await screen.findByText(t.list.loadError)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /thử lại/i })).toBeInTheDocument()
  })

  it('lọc theo trạng thái gửi status lên server', async () => {
    const queries = mockList([makeProject()])
    setup('/projects', 'manager')
    await screen.findByRole('table', { name: t.list.tableLabel })
    await userEvent.selectOptions(screen.getByLabelText(t.list.filterStatus), 'paused')
    await waitFor(() => {
      expect(queries.at(-1)?.get('status')).toBe('paused')
    })
  })

  it('Manager tạo dự án: gửi đúng body rồi mở trang chi tiết', async () => {
    mockList([])
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/projects`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeProject({ id: 'p-new', name: 'Ví điện tử', status: 'new' }), {
          status: 201,
        })
      }),
      mock.get(`${API}/api/v1/projects/p-new`, () =>
        HttpResponse.json(makeProject({ id: 'p-new', name: 'Ví điện tử', status: 'new' })),
      ),
    )
    const { router } = setup('/projects', 'manager')
    await userEvent.click(await screen.findByRole('button', { name: t.list.add }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText(t.form.name), '  Ví điện tử ')
    await userEvent.type(within(dialog).getByLabelText(t.form.startDate), '2026-12-01')
    await userEvent.click(within(dialog).getByRole('button', { name: t.form.create }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/projects/p-new')
    })
    expect(body).toEqual({
      name: 'Ví điện tử',
      description: '',
      startDate: '2026-12-01',
      endDate: null,
    })
  })

  it('ngày kết thúc trước ngày bắt đầu bị chặn ở form', async () => {
    mockList([])
    setup('/projects', 'manager')
    await userEvent.click(await screen.findByRole('button', { name: t.list.add }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText(t.form.name), 'Dự án A')
    await userEvent.type(within(dialog).getByLabelText(t.form.startDate), '2026-12-10')
    await userEvent.type(within(dialog).getByLabelText(t.form.endDate), '2026-12-01')
    await userEvent.click(within(dialog).getByRole('button', { name: t.form.create }))
    expect(await within(dialog).findByText(t.form.datesInvalid)).toBeInTheDocument()
  })

  it('trùng tên từ server hiện dưới ô tên', async () => {
    mockList([])
    server.use(
      mock.post(`${API}/api/v1/projects`, () =>
        HttpResponse.json(
          { error: { code: 'PROJECT_NAME_TAKEN', message: 'Tên dự án đã tồn tại' } },
          { status: 409 },
        ),
      ),
    )
    setup('/projects', 'manager')
    await userEvent.click(await screen.findByRole('button', { name: t.list.add }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText(t.form.name), 'Trùng')
    await userEvent.click(within(dialog).getByRole('button', { name: t.form.create }))
    expect(await within(dialog).findByText('Tên dự án đã tồn tại')).toBeInTheDocument()
  })

  it('Admin không vào được màn dự án', async () => {
    mockList([])
    setup('/projects', 'admin')
    await waitFor(() => {
      expect(screen.queryByRole('table', { name: t.list.tableLabel })).not.toBeInTheDocument()
    })
    expect(screen.queryByRole('heading', { name: t.title })).not.toBeInTheDocument()
  })
})
