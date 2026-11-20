import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { Board } from '../model/types'
import { makeBoard, makeProject, MEMBERS } from '../test-data'

const t = messages.projects
const P = `${API}/api/v1/projects/p-1`

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

function mockDetail(board: Board = makeBoard()) {
  server.use(
    mock.get(P, () => HttpResponse.json(makeProject())),
    mock.get(`${P}/members`, () => HttpResponse.json({ items: MEMBERS })),
    mock.get(`${P}/board`, () => HttpResponse.json(board)),
  )
}

describe('Chi tiết dự án (CF-03)', () => {
  it('hiện thông tin và nút Sửa cho Manager phụ trách', async () => {
    mockDetail()
    setup('/projects/p-1', 'manager')
    expect(
      await screen.findByRole('heading', { name: 'Cổng thông tin khách hàng' }),
    ).toBeInTheDocument()
    expect(screen.getByText(t.status.in_progress)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.detail.edit })).toBeInTheDocument()
    expect(screen.queryByText(t.detail.readOnly)).not.toBeInTheDocument()
  })

  it('Member chỉ xem: không có nút sửa, có ghi chú chỉ xem', async () => {
    mockDetail()
    setup('/projects/p-1', 'member')
    await screen.findByRole('heading', { name: 'Cổng thông tin khách hàng' })
    expect(screen.queryByRole('button', { name: t.detail.edit })).not.toBeInTheDocument()
    expect(screen.getByText(t.detail.readOnly)).toBeInTheDocument()
  })

  it('404 hiện trạng thái không tìm thấy', async () => {
    server.use(
      mock.get(P, () =>
        HttpResponse.json(
          { error: { code: 'PROJECT_NOT_FOUND', message: 'Không tìm thấy' } },
          { status: 404 },
        ),
      ),
    )
    setup('/projects/p-1', 'member')
    expect(await screen.findByText(t.detail.notFoundTitle)).toBeInTheDocument()
  })

  it('sửa thông tin gửi PUT với trạng thái mới', async () => {
    mockDetail()
    let body: unknown
    server.use(
      mock.put(P, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(makeProject({ status: 'paused' }))
      }),
    )
    setup('/projects/p-1', 'manager')
    await userEvent.click(await screen.findByRole('button', { name: t.detail.edit }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.selectOptions(within(dialog).getByLabelText(t.form.status), 'paused')
    await userEvent.click(within(dialog).getByRole('button', { name: t.form.save }))
    await waitFor(() => {
      expect(body).toMatchObject({ name: 'Cổng thông tin khách hàng', status: 'paused' })
    })
  })

  describe('Thành viên', () => {
    it('Manager thêm thành viên từ danh sách ứng viên', async () => {
      mockDetail()
      let added: unknown
      server.use(
        mock.get(`${P}/member-candidates`, () =>
          HttpResponse.json({
            items: [
              {
                userId: 'u-9',
                fullName: 'Lê Văn Cường',
                email: 'cuong@worktrack.local',
                role: 'member',
                status: 'pending',
              },
            ],
          }),
        ),
        mock.post(`${P}/members`, async ({ request }) => {
          added = await request.json()
          return new HttpResponse(null, { status: 204 })
        }),
      )
      setup('/projects/p-1?tab=members', 'manager')
      await userEvent.click(await screen.findByRole('button', { name: t.members.add }))
      const dialog = await screen.findByRole('dialog')
      await userEvent.click(
        await within(dialog).findByRole('button', { name: `${t.members.choose} Lê Văn Cường` }),
      )
      await waitFor(() => {
        expect(added).toEqual({ userId: 'u-9' })
      })
    })

    it('Manager bỏ thành viên qua hộp xác nhận; không có nút bỏ cho Manager phụ trách', async () => {
      mockDetail()
      let removedPath = ''
      server.use(
        mock.delete(`${P}/members/:userId`, ({ params }) => {
          removedPath = String(params.userId)
          return new HttpResponse(null, { status: 204 })
        }),
      )
      setup('/projects/p-1?tab=members', 'manager')
      const table = await screen.findByRole('table', { name: t.members.tableLabel })
      expect(within(table).getAllByRole('button', { name: t.members.remove })).toHaveLength(1)
      await userEvent.click(within(table).getByRole('button', { name: t.members.remove }))
      const dialog = await screen.findByRole('dialog')
      expect(within(dialog).getByText(t.members.removeTitle('Đỗ Thu Hà'))).toBeInTheDocument()
      await userEvent.click(within(dialog).getByRole('button', { name: t.members.remove }))
      await waitFor(() => {
        expect(removedPath).toBe('u-2')
      })
    })

    it('Member không thấy nút thêm/bỏ', async () => {
      mockDetail()
      setup('/projects/p-1?tab=members', 'member')
      await screen.findByRole('table', { name: t.members.tableLabel })
      expect(screen.queryByRole('button', { name: t.members.add })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: t.members.remove })).not.toBeInTheDocument()
    })
  })

  describe('Cột Kanban', () => {
    it('đổi thứ tự gửi columnIds mới', async () => {
      mockDetail()
      let body: unknown
      server.use(
        mock.put(`${P}/column-order`, async ({ request }) => {
          body = await request.json()
          return HttpResponse.json(makeBoard())
        }),
      )
      setup('/projects/p-1?tab=columns', 'manager')
      await userEvent.click(
        await screen.findByRole('button', { name: t.columns.moveDown('Backlog') }),
      )
      await waitFor(() => {
        expect(body).toEqual({ columnIds: ['c-2', 'c-1', 'c-3'] })
      })
    })

    it('xóa cột đang có việc: cảnh báo và hiện lỗi server COLUMN_HAS_TASKS', async () => {
      mockDetail()
      server.use(
        mock.delete(`${P}/columns/c-2`, () =>
          HttpResponse.json(
            { error: { code: 'COLUMN_HAS_TASKS', message: 'Cột còn công việc' } },
            { status: 409 },
          ),
        ),
      )
      setup('/projects/p-1?tab=columns', 'manager')
      const table = await screen.findByRole('table', { name: t.columns.tableLabel })
      const row = within(table).getByText('In Progress').closest('tr')
      if (!row) throw new Error('không thấy hàng In Progress')
      await userEvent.click(within(row).getByRole('button', { name: t.columns.remove }))
      const dialog = await screen.findByRole('dialog')
      expect(within(dialog).getByText(t.columns.removeHasTasks('3'))).toBeInTheDocument()
      await userEvent.click(within(dialog).getByRole('button', { name: t.columns.remove }))
      expect(await within(dialog).findByText('Cột còn công việc')).toBeInTheDocument()
    })

    it('thêm cột gửi tên và nhóm trạng thái', async () => {
      mockDetail()
      let body: unknown
      server.use(
        mock.post(`${P}/columns`, async ({ request }) => {
          body = await request.json()
          return HttpResponse.json(makeBoard(), { status: 201 })
        }),
      )
      setup('/projects/p-1?tab=columns', 'manager')
      await userEvent.click(await screen.findByRole('button', { name: t.columns.add }))
      const dialog = await screen.findByRole('dialog')
      await userEvent.type(within(dialog).getByLabelText(t.columns.name), 'Review')
      await userEvent.selectOptions(within(dialog).getByLabelText(t.columns.group), 'in_progress')
      await userEvent.click(within(dialog).getByRole('button', { name: t.columns.save }))
      await waitFor(() => {
        expect(body).toEqual({ name: 'Review', statusGroup: 'in_progress' })
      })
    })
  })

  describe('Luật chuyển', () => {
    it('ô Member vào/ra nhóm Hoàn thành bị khóa', async () => {
      mockDetail()
      setup('/projects/p-1?tab=transitions', 'manager')
      const matrix = await screen.findByRole('table', {
        name: t.transitions.matrixLabel(t.transitionRoles.member),
      })
      expect(
        within(matrix).getByRole('checkbox', { name: t.transitions.cell('In Progress', 'Done') }),
      ).toBeDisabled()
      expect(
        within(matrix).getByRole('checkbox', {
          name: t.transitions.cell('Backlog', 'In Progress'),
        }),
      ).toBeChecked()
    })

    it('lưu luật chuyển gửi danh sách mới', async () => {
      mockDetail()
      let body: { transitions: unknown[] } | undefined
      server.use(
        mock.put(`${P}/transitions`, async ({ request }) => {
          body = (await request.json()) as { transitions: unknown[] }
          return HttpResponse.json(makeBoard())
        }),
      )
      setup('/projects/p-1?tab=transitions', 'manager')
      const matrix = await screen.findByRole('table', {
        name: t.transitions.matrixLabel(t.transitionRoles.manager),
      })
      await userEvent.click(
        within(matrix).getByRole('checkbox', {
          name: t.transitions.cell('In Progress', 'Backlog'),
        }),
      )
      expect(screen.getByText(t.transitions.unsaved)).toBeInTheDocument()
      await userEvent.click(screen.getByRole('button', { name: t.transitions.save }))
      await waitFor(() => {
        expect(body?.transitions).toContainEqual({
          fromColumnId: 'c-2',
          toColumnId: 'c-1',
          role: 'manager',
        })
      })
      expect(body?.transitions).toHaveLength(4)
    })

    it('Member xem luật chuyển ở chế độ chỉ đọc', async () => {
      mockDetail()
      setup('/projects/p-1?tab=transitions', 'member')
      const matrix = await screen.findByRole('table', {
        name: t.transitions.matrixLabel(t.transitionRoles.manager),
      })
      for (const box of within(matrix).getAllByRole('checkbox')) expect(box).toBeDisabled()
      expect(screen.queryByRole('button', { name: t.transitions.save })).not.toBeInTheDocument()
    })
  })
})
