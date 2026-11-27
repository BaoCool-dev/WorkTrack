// Test bổ sung từ thiết kế test case (docs/testing/test-cases.md).
// Tên mỗi ca chứa mã TC để đối chiếu với bảng test case.
import { screen } from '@testing-library/react'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { messages } from '@/shared/lib/messages'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import { makeBoard, makeProject, MEMBERS } from '@/features/projects/test-data'

const t = messages.projects
const P = `${API}/api/v1/projects/p-1`
const FORBIDDEN = 'Bạn không có quyền xem trang này'

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

describe('Phân quyền màn hình theo vai trò (thiết kế test case)', () => {
  it('Admin gõ thẳng /my-tasks bị chặn [TC_TASK_041][CF-06]', async () => {
    setup('/my-tasks', 'admin')
    expect(await screen.findByRole('heading', { name: FORBIDDEN })).toBeInTheDocument()
  })

  it('CEO gõ thẳng /my-tasks bị chặn [TC_TASK_042][CF-06]', async () => {
    setup('/my-tasks', 'ceo')
    expect(await screen.findByRole('heading', { name: FORBIDDEN })).toBeInTheDocument()
  })

  it('CEO xem chi tiết dự án ở chế độ chỉ đọc [TC_PROJ_039][CF-03]', async () => {
    server.use(
      mock.get(P, () => HttpResponse.json(makeProject())),
      mock.get(`${P}/members`, () => HttpResponse.json({ items: MEMBERS })),
      mock.get(`${P}/board`, () => HttpResponse.json(makeBoard())),
    )
    setup('/projects/p-1', 'ceo')
    await screen.findByRole('heading', { name: 'Cổng thông tin khách hàng' })
    expect(screen.queryByRole('button', { name: t.detail.edit })).not.toBeInTheDocument()
    expect(screen.getByText(t.detail.readOnly)).toBeInTheDocument()
  })
})
