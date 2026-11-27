import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Session } from '@/features/auth'
import { page } from '@/features/tasks/test-data'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { AppNotification } from '../model/types'

// CF-07 — chuông thông báo ở Topbar.
const n = messages.notifications
const BASE = `${API}/api/v1/me/notifications`

let current: Session | undefined
afterEach(() => {
  current?.dispose()
  current = undefined
})

function makeNotification(over: Partial<AppNotification> = {}): AppNotification {
  return {
    id: 'n-1',
    type: 'task_assigned',
    task: { id: 't-1', name: 'Thiết kế API đăng nhập' },
    project: { id: 'p-1', name: 'Cổng thông tin khách hàng' },
    actor: { id: 'u-manager', name: 'Trần Bình' },
    message: '',
    read: false,
    createdAt: '2026-11-27T02:00:00Z',
    ...over,
  }
}

function mockBell(count: number, items: AppNotification[]) {
  server.use(
    mock.get(`${API}/api/v1/me/tasks`, () => HttpResponse.json(page([]))),
    mock.get(`${API}/api/v1/me/wip`, () => HttpResponse.json({ count: 0 })),
    mock.get(`${BASE}/unread-count`, () => HttpResponse.json({ count })),
    mock.get(BASE, () => HttpResponse.json(page(items))),
  )
}

async function openBell(label: string) {
  const r = renderApp('/my-tasks', 'member')
  current = r.session
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: label }))
  const panel = await screen.findByRole('region', { name: n.title })
  return { user, panel }
}

describe('Chuông thông báo (CF-07)', () => {
  it('nhãn chuông nói số chưa đọc', async () => {
    mockBell(2, [])
    const r = renderApp('/my-tasks', 'member')
    current = r.session
    expect(await screen.findByRole('button', { name: n.bellUnread('2') })).toBeInTheDocument()
  })

  it('không có chưa đọc thì nhãn chỉ là "Thông báo"', async () => {
    mockBell(0, [])
    const r = renderApp('/my-tasks', 'member')
    current = r.session
    expect(await screen.findByRole('button', { name: n.bell })).toBeInTheDocument()
  })

  it('mở chuông hiện câu thông báo theo loại, kèm trích đoạn', async () => {
    mockBell(2, [
      makeNotification(),
      makeNotification({
        id: 'n-2',
        type: 'review_rejected',
        message: 'Thiếu test refresh token',
        read: true,
      }),
    ])
    const { panel } = await openBell(n.bellUnread('2'))
    const items = await within(panel).findAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent(n.text.task_assigned('Trần Bình', 'Thiết kế API đăng nhập'))
    expect(items[0]).toHaveTextContent(n.unread)
    expect(items[1]).toHaveTextContent('Thiếu test refresh token')
    expect(items[1]).not.toHaveTextContent(n.unread)
  })

  it('bấm một thông báo chưa đọc thì đánh dấu đã đọc', async () => {
    mockBell(1, [makeNotification()])
    let marked = ''
    server.use(
      mock.post(`${BASE}/:id/read`, ({ params }) => {
        marked = String(params.id)
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const { user, panel } = await openBell(n.bellUnread('1'))
    await user.click(
      await within(panel).findByRole('button', {
        name: n.markRead(n.text.task_assigned('Trần Bình', 'Thiết kế API đăng nhập')),
      }),
    )
    await vi.waitFor(() => {
      expect(marked).toBe('n-1')
    })
  })

  it('đánh dấu đọc hết báo toast', async () => {
    mockBell(1, [makeNotification()])
    let called = false
    server.use(
      mock.post(`${BASE}/read-all`, () => {
        called = true
        return HttpResponse.json({ count: 1 })
      }),
    )
    const { user, panel } = await openBell(n.bellUnread('1'))
    await user.click(await within(panel).findByRole('button', { name: n.markAll }))
    expect(await screen.findByText(n.markedAll)).toBeInTheDocument()
    expect(called).toBe(true)
  })

  it('trạng thái rỗng', async () => {
    mockBell(0, [])
    const { panel } = await openBell(n.bell)
    expect(await within(panel).findByText(n.emptyTitle)).toBeInTheDocument()
  })

  it('lỗi tải danh sách hiện ErrorState', async () => {
    mockBell(0, [])
    server.use(
      mock.get(BASE, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    const { panel } = await openBell(n.bell)
    expect(await within(panel).findByText(n.loadError)).toBeInTheDocument()
  })

  it('Escape đóng bảng thông báo', async () => {
    mockBell(0, [])
    const { user } = await openBell(n.bell)
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('region', { name: n.title })).not.toBeInTheDocument()
  })
})
