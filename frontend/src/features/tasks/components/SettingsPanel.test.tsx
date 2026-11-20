import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'

const t = messages.tasks.settings
const PATH = '/admin/users?tab=settings'

let current: Session | undefined
function setup(role: Role = 'admin') {
  const r = renderApp(PATH, role)
  current = r.session
  return r
}
afterEach(() => {
  current?.dispose()
  current = undefined
})

function mockSettings(hours = 4) {
  server.use(
    mock.get(`${API}/api/v1/settings`, () =>
      HttpResponse.json({ storyPointHours: hours, updatedAt: '2026-11-20T02:00:00Z' }),
    ),
  )
}

describe('Tab Cấu hình B1 (CF-06)', () => {
  it('hiện tab "Cấu hình" với giá trị hiện tại', async () => {
    mockSettings(4)
    setup()
    expect(
      await screen.findByRole('tab', { name: messages.admin.tabs.settings, selected: true }),
    ).toBeInTheDocument()
    expect(await screen.findByLabelText(t.hours)).toHaveValue(4)
    expect(screen.getByText(t.updatedAt('20/11/2026'))).toBeInTheDocument()
  })

  it('chặn giá trị ngoài 1–40 ở client, không gửi request', async () => {
    mockSettings(4)
    let called = false
    server.use(
      mock.put(`${API}/api/v1/settings`, () => {
        called = true
        return HttpResponse.json({})
      }),
    )
    setup()
    const user = userEvent.setup()
    const input = await screen.findByLabelText(t.hours)
    await user.clear(input)
    await user.type(input, '41')
    await user.click(screen.getByRole('button', { name: t.save }))
    expect(await screen.findByText(t.invalid)).toBeInTheDocument()
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(called).toBe(false)
  })

  it('lưu hệ số mới gửi PUT dạng số và hiện toast', async () => {
    mockSettings(4)
    let body: unknown
    server.use(
      mock.put(`${API}/api/v1/settings`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ storyPointHours: 6, updatedAt: '2026-11-21T02:00:00Z' })
      }),
    )
    setup()
    const user = userEvent.setup()
    const input = await screen.findByLabelText(t.hours)
    await user.clear(input)
    await user.type(input, '6')
    await user.click(screen.getByRole('button', { name: t.save }))
    expect(await screen.findByText(t.saved)).toBeInTheDocument()
    expect(body).toEqual({ storyPointHours: 6 })
  })

  it('lỗi server SETTING_INVALID gắn vào ô nhập', async () => {
    mockSettings(4)
    server.use(
      mock.put(`${API}/api/v1/settings`, () =>
        HttpResponse.json(
          { error: { code: 'SETTING_INVALID', message: 'Hệ số không hợp lệ' } },
          { status: 422 },
        ),
      ),
    )
    setup()
    const user = userEvent.setup()
    const input = await screen.findByLabelText(t.hours)
    await user.clear(input)
    await user.type(input, '8')
    await user.click(screen.getByRole('button', { name: t.save }))
    const form = screen.getByRole('form', { name: t.title })
    expect(await within(form).findByText('Hệ số không hợp lệ')).toBeInTheDocument()
  })

  it('lỗi tải hiện ErrorState', async () => {
    server.use(
      mock.get(`${API}/api/v1/settings`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    setup()
    expect(await screen.findByText(t.loadError)).toBeInTheDocument()
  })
})
