import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { Label } from '../model/types'
import { LABELS } from '../test-data'

const t = messages.tasks.labels
const PATH = '/admin/users?tab=labels'

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

function mockLabels(items: Label[] = LABELS) {
  server.use(mock.get(`${API}/api/v1/labels`, () => HttpResponse.json({ items })))
}

describe('Tab Nhãn trong Quản trị (CF-06)', () => {
  it('có tab "Nhãn" và hiện bảng nhãn kèm số công việc', async () => {
    mockLabels()
    setup()
    expect(
      await screen.findByRole('tab', { name: messages.admin.tabs.labels, selected: true }),
    ).toBeInTheDocument()
    const table = await screen.findByRole('table', { name: t.tableLabel })
    expect(within(table).getByText('Backend')).toBeInTheDocument()
    expect(within(table).getByText('Frontend')).toBeInTheDocument()
    expect(within(table).getByText('2')).toBeInTheDocument()
  })

  it('trạng thái rỗng khi chưa có nhãn', async () => {
    mockLabels([])
    setup()
    expect(await screen.findByText(t.emptyTitle)).toBeInTheDocument()
  })

  it('lỗi tải hiện ErrorState có nút thử lại', async () => {
    server.use(
      mock.get(`${API}/api/v1/labels`, () =>
        HttpResponse.json({ error: { code: 'INTERNAL', message: 'x' } }, { status: 500 }),
      ),
    )
    setup()
    expect(await screen.findByText(t.loadError)).toBeInTheDocument()
  })

  it('thêm nhãn: kiểm tra rỗng ở client rồi gửi POST, hiện toast', async () => {
    mockLabels()
    let body: unknown
    server.use(
      mock.post(`${API}/api/v1/labels`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ id: 'l-3', name: 'Bug', taskCount: 0 }, { status: 201 })
      }),
    )
    setup()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: t.add }))
    const dialog = await screen.findByRole('dialog', { name: t.createTitle })
    await user.click(within(dialog).getByRole('button', { name: t.save }))
    expect(await within(dialog).findByText(t.nameRequired)).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(t.name), '  Bug ')
    await user.click(within(dialog).getByRole('button', { name: t.save }))
    expect(await screen.findByText(t.created)).toBeInTheDocument()
    expect(body).toEqual({ name: 'Bug' })
  })

  it('trùng tên (409 LABEL_NAME_TAKEN) gắn lỗi vào ô tên', async () => {
    mockLabels()
    server.use(
      mock.put(`${API}/api/v1/labels/l-1`, () =>
        HttpResponse.json(
          { error: { code: 'LABEL_NAME_TAKEN', message: 'Tên nhãn đã tồn tại' } },
          { status: 409 },
        ),
      ),
    )
    setup()
    const user = userEvent.setup()
    const table = await screen.findByRole('table', { name: t.tableLabel })
    const row = within(table).getByText('Backend').closest('tr')
    if (!row) throw new Error('row')
    await user.click(within(row).getByRole('button', { name: t.edit }))
    const dialog = await screen.findByRole('dialog', { name: t.editTitle })
    const input = within(dialog).getByLabelText(t.name)
    expect(input).toHaveValue('Backend')
    await user.clear(input)
    await user.type(input, 'Frontend')
    await user.click(within(dialog).getByRole('button', { name: t.save }))
    expect(await within(dialog).findByText('Tên nhãn đã tồn tại')).toBeInTheDocument()
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })

  it('xóa nhãn đang dùng: báo số công việc, khóa nút Xóa kèm lý do', async () => {
    mockLabels()
    setup()
    const user = userEvent.setup()
    const table = await screen.findByRole('table', { name: t.tableLabel })
    const row = within(table).getByText('Backend').closest('tr')
    if (!row) throw new Error('row')
    await user.click(within(row).getByRole('button', { name: t.remove }))
    const dialog = await screen.findByRole('dialog', { name: t.removeTitle('Backend') })
    expect(within(dialog).getByText(t.removeInUse('2'))).toBeInTheDocument()
    expect(within(dialog).getByText(t.removeBlocked)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: t.remove })).toBeDisabled()
  })

  it('server trả LABEL_IN_USE (danh mục cũ) → lỗi hiện trong hộp thoại', async () => {
    mockLabels()
    server.use(
      mock.delete(`${API}/api/v1/labels/l-2`, () =>
        HttpResponse.json(
          { error: { code: 'LABEL_IN_USE', message: 'Nhãn đang được dùng' } },
          { status: 409 },
        ),
      ),
    )
    setup()
    const user = userEvent.setup()
    const table = await screen.findByRole('table', { name: t.tableLabel })
    const row = within(table).getByText('Frontend').closest('tr')
    if (!row) throw new Error('row')
    await user.click(within(row).getByRole('button', { name: t.remove }))
    const dialog = await screen.findByRole('dialog', { name: t.removeTitle('Frontend') })
    await user.click(within(dialog).getByRole('button', { name: t.remove }))
    expect(await within(dialog).findByText('Nhãn đang được dùng')).toBeInTheDocument()
  })

  it('xóa nhãn chưa dùng gửi DELETE và hiện toast', async () => {
    mockLabels()
    let deleted = false
    server.use(
      mock.delete(`${API}/api/v1/labels/l-2`, () => {
        deleted = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    setup()
    const user = userEvent.setup()
    const table = await screen.findByRole('table', { name: t.tableLabel })
    const row = within(table).getByText('Frontend').closest('tr')
    if (!row) throw new Error('row')
    await user.click(within(row).getByRole('button', { name: t.remove }))
    const dialog = await screen.findByRole('dialog', { name: t.removeTitle('Frontend') })
    await user.click(within(dialog).getByRole('button', { name: t.remove }))
    expect(await screen.findByText(t.removed)).toBeInTheDocument()
    await waitFor(() => {
      expect(deleted).toBe(true)
    })
  })
})
