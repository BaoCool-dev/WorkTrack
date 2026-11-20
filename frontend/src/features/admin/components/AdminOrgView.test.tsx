import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Role, Session } from '@/features/auth'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'
import type { AdminUser, CatalogItem, Department } from '../model/types'

let current: Session | undefined
function setup(path: string, role: Role = 'admin') {
  const r = renderApp(path, role)
  current = r.session
  return r
}
afterEach(() => {
  current?.dispose()
  current = undefined
})

const DEPARTMENTS: Department[] = [
  { id: 'd-1', name: 'Khối Kỹ thuật', parentId: null, memberCount: 3 },
  { id: 'd-2', name: 'Phòng Backend', parentId: 'd-1', memberCount: 2 },
  { id: 'd-3', name: 'Phòng Kinh doanh', parentId: null, memberCount: 0 },
]
const JOB_TITLES: CatalogItem[] = [
  { id: 'j-1', name: 'Lập trình viên', memberCount: 2 },
  { id: 'j-2', name: 'Kiểm thử viên', memberCount: 0 },
]
const SKILLS: CatalogItem[] = [
  { id: 's-1', name: 'Go', memberCount: 1 },
  { id: 's-2', name: 'React', memberCount: 0 },
]

function makeAdminUser(over: Partial<AdminUser> = {}): AdminUser {
  return {
    id: 'u-1',
    email: 'an.nguyen@worktrack.local',
    fullName: 'Nguyễn Văn An',
    role: 'member',
    status: 'active',
    lastLoginAt: null,
    createdAt: '2026-11-01T00:00:00Z',
    permissions: [],
    department: { id: 'd-2', name: 'Phòng Backend' },
    jobTitle: { id: 'j-1', name: 'Lập trình viên' },
    skills: [{ skillId: 's-1', name: 'Go', level: 'senior' }],
    ...over,
  }
}

const USERS: AdminUser[] = [
  makeAdminUser(),
  makeAdminUser({
    id: 'u-2',
    email: 'binh@worktrack.local',
    fullName: 'Trần Bình',
    role: 'manager',
    status: 'pending',
    department: null,
    jobTitle: null,
    skills: [],
  }),
  makeAdminUser({
    id: 'u-3',
    email: 'hung@worktrack.local',
    fullName: 'Phạm Quốc Hùng',
    role: 'ceo',
    department: null,
    jobTitle: null,
    skills: [],
  }),
]

interface Captured {
  userQueries: URLSearchParams[]
  bodies: { method: string; path: string; body: unknown }[]
}

/** Backend giả cho màn quản trị: ghi lại query và body để kiểm tra. */
function mockAdminApi(opts: { users?: AdminUser[]; total?: number } = {}): Captured {
  const captured: Captured = { userQueries: [], bodies: [] }
  const users = opts.users ?? USERS
  const record = async (request: Request, path: string) => {
    const text = await request.text()
    captured.bodies.push({
      method: request.method,
      path,
      body: text ? (JSON.parse(text) as unknown) : null,
    })
  }
  server.use(
    mock.get(`${API}/api/v1/users`, ({ request }) => {
      const q = new URL(request.url).searchParams
      captured.userQueries.push(q)
      const search = (q.get('search') ?? '').toLowerCase()
      const role = q.get('role')
      const items = users.filter(
        (u) => (!search || u.fullName.toLowerCase().includes(search)) && (!role || u.role === role),
      )
      return HttpResponse.json({
        items,
        page: Number(q.get('page')),
        pageSize: Number(q.get('pageSize')),
        total: opts.total ?? items.length,
      })
    }),
    mock.get(`${API}/api/v1/departments`, () => HttpResponse.json({ items: DEPARTMENTS })),
    mock.get(`${API}/api/v1/job-titles`, () => HttpResponse.json({ items: JOB_TITLES })),
    mock.get(`${API}/api/v1/skills`, () => HttpResponse.json({ items: SKILLS })),
    mock.post(`${API}/api/v1/users`, async ({ request }) => {
      await record(request, '/users')
      return HttpResponse.json(
        { user: makeAdminUser({ id: 'u-new' }), emailSent: true },
        { status: 201 },
      )
    }),
    mock.put(`${API}/api/v1/users/:id`, async ({ request, params }) => {
      await record(request, `/users/${String(params.id)}`)
      return HttpResponse.json(makeAdminUser())
    }),
    mock.put(`${API}/api/v1/users/:id/skills`, async ({ request, params }) => {
      await record(request, `/users/${String(params.id)}/skills`)
      return HttpResponse.json({ skills: [] })
    }),
    mock.post(`${API}/api/v1/users/:id/activation-email`, async ({ request, params }) => {
      await record(request, `/users/${String(params.id)}/activation-email`)
      return new HttpResponse(null, { status: 204 })
    }),
    mock.post(`${API}/api/v1/departments`, async ({ request }) => {
      await record(request, '/departments')
      return HttpResponse.json(
        { id: 'd-new', name: 'x', parentId: null, memberCount: 0 },
        { status: 201 },
      )
    }),
    mock.delete(`${API}/api/v1/departments/:id`, () =>
      HttpResponse.json(
        { error: { code: 'DEPARTMENT_HAS_CHILDREN', message: 'Phòng ban còn phòng ban con' } },
        { status: 409 },
      ),
    ),
    mock.post(`${API}/api/v1/job-titles`, async ({ request }) => {
      await record(request, '/job-titles')
      return HttpResponse.json({ id: 'j-new', name: 'x', memberCount: 0 }, { status: 201 })
    }),
    mock.delete(`${API}/api/v1/skills/:id`, async ({ request, params }) => {
      await record(request, `/skills/${String(params.id)}`)
      return new HttpResponse(null, { status: 204 })
    }),
  )
  return captured
}

const table = () => screen.findByRole('table', { name: 'Danh sách nhân sự' })
const rowOf = async (name: string) => {
  const t = await table()
  const cell = await within(t).findByText(name)
  const row = cell.closest('tr')
  if (!row) throw new Error(`không thấy dòng ${name}`)
  return within(row)
}

describe('Danh sách nhân sự [CF-02][B9]', () => {
  it('hiện họ tên, email, vai trò, phòng ban, chức danh, kỹ năng, trạng thái và tổng số', async () => {
    mockAdminApi()
    setup('/admin/users')
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Nhân sự & phòng ban' }),
    ).toBeInTheDocument()
    const an = await rowOf('Nguyễn Văn An')
    expect(an.getByText('an.nguyen@worktrack.local')).toBeInTheDocument()
    expect(an.getByText('Thành viên')).toBeInTheDocument()
    expect(an.getByText('Phòng Backend')).toBeInTheDocument()
    expect(an.getByText('Lập trình viên')).toBeInTheDocument()
    expect(an.getByText(/Go · Senior/)).toBeInTheDocument()
    expect(an.getByText('Đang hoạt động')).toBeInTheDocument()
    const binh = await rowOf('Trần Bình')
    expect(binh.getByText('Chờ kích hoạt')).toBeInTheDocument()
    expect(screen.getByText('3 nhân sự')).toBeInTheDocument()
  })

  it('tìm kiếm gửi search, lọc vai trò gửi role và về trang 1', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users')
    await table()
    await user.type(screen.getByRole('searchbox', { name: 'Tìm kiếm nhân sự' }), 'bình')
    await waitFor(() => {
      expect(cap.userQueries.at(-1)?.get('search')).toBe('bình')
    })
    await user.selectOptions(screen.getByLabelText('Vai trò'), 'manager')
    await waitFor(() => {
      expect(cap.userQueries.at(-1)?.get('role')).toBe('manager')
    })
    expect(cap.userQueries.at(-1)?.get('page')).toBe('1')
    expect(cap.userQueries.at(-1)?.get('pageSize')).toBe('20')
  })

  it('phân trang: bấm Trang sau gửi page=2', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi({ total: 45 })
    setup('/admin/users')
    await table()
    expect(screen.getByText('Trang 1/3')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Trang sau' }))
    await waitFor(() => {
      expect(cap.userQueries.at(-1)?.get('page')).toBe('2')
    })
  })

  it('chưa có ai → trạng thái rỗng; có bộ lọc mà không khớp → gợi ý xóa bộ lọc', async () => {
    const user = userEvent.setup()
    mockAdminApi({ users: [] })
    setup('/admin/users')
    expect(await screen.findByText('Chưa có nhân sự nào')).toBeInTheDocument()
    await user.type(screen.getByRole('searchbox', { name: 'Tìm kiếm nhân sự' }), 'zzz')
    expect(await screen.findByText('Không có nhân sự phù hợp')).toBeInTheDocument()
    await user.click(screen.getAllByRole('button', { name: 'Xóa bộ lọc' })[0] as HTMLElement)
    expect(await screen.findByText('Chưa có nhân sự nào')).toBeInTheDocument()
  })

  it('lỗi tải danh sách → trạng thái lỗi kèm mã lỗi và nút thử lại', async () => {
    mockAdminApi()
    server.use(
      mock.get(`${API}/api/v1/users`, () =>
        HttpResponse.json(
          { error: { code: 'INTERNAL', message: 'Lỗi hệ thống' } },
          { status: 500 },
        ),
      ),
    )
    setup('/admin/users')
    const alert = await screen.findByRole('alert')
    expect(within(alert).getByText('Không tải được danh sách nhân sự')).toBeInTheDocument()
    expect(within(alert).getByText(/INTERNAL/)).toBeInTheDocument()
    expect(within(alert).getByRole('button', { name: 'Thử lại' })).toBeInTheDocument()
  })
})

describe('Thêm / sửa nhân sự [CF-02]', () => {
  it('validate phía client: bỏ trống và email sai định dạng', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users')
    await table()
    await user.click(screen.getByRole('button', { name: 'Thêm nhân sự' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Thêm nhân sự' }))
    await user.click(dialog.getByRole('button', { name: 'Tạo tài khoản' }))
    expect(await dialog.findByText('Vui lòng nhập email.')).toBeInTheDocument()
    expect(dialog.getByText('Vui lòng nhập họ tên.')).toBeInTheDocument()
    await user.type(dialog.getByLabelText('Email'), 'khong-phai-email')
    await user.click(dialog.getByRole('button', { name: 'Tạo tài khoản' }))
    expect(await dialog.findByText('Email chưa đúng định dạng.')).toBeInTheDocument()
    expect(cap.bodies).toHaveLength(0)
  })

  it('email trùng (409) → báo "Email đã tồn tại" ngay dưới ô email', async () => {
    const user = userEvent.setup()
    mockAdminApi()
    server.use(
      mock.post(`${API}/api/v1/users`, () =>
        HttpResponse.json(
          { error: { code: 'EMAIL_TAKEN', message: 'Email đã được sử dụng' } },
          { status: 409 },
        ),
      ),
    )
    setup('/admin/users')
    await table()
    await user.click(screen.getByRole('button', { name: 'Thêm nhân sự' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Thêm nhân sự' }))
    await user.type(dialog.getByLabelText('Email'), 'an.nguyen@worktrack.local')
    await user.type(dialog.getByLabelText('Họ tên'), 'Nguyễn Văn An')
    await user.click(dialog.getByRole('button', { name: 'Tạo tài khoản' }))
    const email = dialog.getByLabelText('Email')
    await waitFor(() => {
      expect(email).toHaveAccessibleDescription('Email đã tồn tại')
    })
    expect(email).toHaveAttribute('aria-invalid', 'true')
  })

  it('tạo thành công → gửi đúng body, đóng form và hiện toast', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users')
    await table()
    await user.click(screen.getByRole('button', { name: 'Thêm nhân sự' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Thêm nhân sự' }))
    await user.type(dialog.getByLabelText('Email'), '  ha.do@worktrack.local ')
    await user.type(dialog.getByLabelText('Họ tên'), 'Đỗ Thu Hà')
    await user.selectOptions(dialog.getByLabelText('Vai trò'), 'manager')
    await user.selectOptions(dialog.getByLabelText('Phòng ban'), 'd-2')
    await user.click(dialog.getByRole('button', { name: 'Tạo tài khoản' }))
    expect(await screen.findByText('Đã tạo tài khoản và gửi email kích hoạt')).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(cap.bodies).toEqual([
      {
        method: 'POST',
        path: '/users',
        body: {
          email: 'ha.do@worktrack.local',
          fullName: 'Đỗ Thu Hà',
          role: 'manager',
          departmentId: 'd-2',
          jobTitleId: null,
        },
      },
    ])
  })

  it('Admin không thấy lựa chọn vai trò CEO; CEO thì thấy', async () => {
    const user = userEvent.setup()
    mockAdminApi()
    setup('/admin/users', 'admin')
    await table()
    await user.click(screen.getByRole('button', { name: 'Thêm nhân sự' }))
    let dialog = within(await screen.findByRole('dialog', { name: 'Thêm nhân sự' }))
    expect(dialog.queryByRole('option', { name: /CEO/ })).not.toBeInTheDocument()
    current?.dispose()
    current = undefined
    // dựng lại với vai trò CEO
    document.body.innerHTML = ''
    mockAdminApi()
    setup('/admin/users', 'ceo')
    await table()
    await user.click(screen.getByRole('button', { name: 'Thêm nhân sự' }))
    dialog = within(await screen.findByRole('dialog', { name: 'Thêm nhân sự' }))
    expect(dialog.getByRole('option', { name: /CEO/ })).toBeInTheDocument()
  })

  it('sửa: email chỉ đọc và không gửi email trong body PUT', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users')
    const an = await rowOf('Nguyễn Văn An')
    await user.click(an.getByRole('button', { name: 'Sửa' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Sửa thông tin nhân sự' }))
    expect(dialog.getByLabelText('Email')).toBeDisabled()
    expect(dialog.getByLabelText('Phòng ban')).toHaveValue('d-2')
    const name = dialog.getByLabelText('Họ tên')
    await user.clear(name)
    await user.type(name, 'Nguyễn Văn Ân')
    await user.click(dialog.getByRole('button', { name: 'Lưu thay đổi' }))
    expect(await screen.findByText('Đã lưu thông tin nhân sự')).toBeInTheDocument()
    expect(cap.bodies).toEqual([
      {
        method: 'PUT',
        path: '/users/u-1',
        body: { fullName: 'Nguyễn Văn Ân', role: 'member', departmentId: 'd-2', jobTitleId: 'j-1' },
      },
    ])
  })

  it('Admin không có nút Sửa ở dòng của CEO', async () => {
    mockAdminApi()
    setup('/admin/users', 'admin')
    const ceo = await rowOf('Phạm Quốc Hùng')
    expect(ceo.queryByRole('button', { name: 'Sửa' })).not.toBeInTheDocument()
  })

  it('gửi lại email kích hoạt cho tài khoản chờ kích hoạt', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users')
    const binh = await rowOf('Trần Bình')
    expect(
      (await rowOf('Nguyễn Văn An')).queryByRole('button', { name: 'Gửi lại email kích hoạt' }),
    ).toBeNull()
    await user.click(binh.getByRole('button', { name: 'Gửi lại email kích hoạt' }))
    expect(await screen.findByText('Đã gửi lại email kích hoạt')).toBeInTheDocument()
    expect(cap.bodies[0]?.path).toBe('/users/u-2/activation-email')
  })

  it('gửi lại kích hoạt thất bại → dải lỗi (role=alert), không dùng toast', async () => {
    const user = userEvent.setup()
    mockAdminApi()
    server.use(
      mock.post(`${API}/api/v1/users/:id/activation-email`, () =>
        HttpResponse.json(
          {
            error: {
              code: 'MAIL_SEND_FAILED',
              message: 'Không gửi được email, vui lòng thử lại sau.',
            },
          },
          { status: 502 },
        ),
      ),
    )
    setup('/admin/users')
    const binh = await rowOf('Trần Bình')
    await user.click(binh.getByRole('button', { name: 'Gửi lại email kích hoạt' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Không gửi được email')
  })
})

describe('Hồ sơ kỹ năng [CF-02]', () => {
  it('thêm kỹ năng, chọn mức độ rồi lưu → PUT danh sách đầy đủ', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users')
    const an = await rowOf('Nguyễn Văn An')
    await user.click(an.getByRole('button', { name: 'Kỹ năng' }))
    const dialog = within(
      await screen.findByRole('dialog', { name: 'Hồ sơ kỹ năng — Nguyễn Văn An' }),
    )
    // Kỹ năng đã có không xuất hiện lại trong ô chọn thêm.
    const picker = dialog.getByLabelText('Chọn kỹ năng')
    expect(within(picker).queryByRole('option', { name: 'Go' })).not.toBeInTheDocument()
    await user.selectOptions(picker, 's-2')
    await user.click(dialog.getByRole('button', { name: 'Thêm kỹ năng' }))
    await user.selectOptions(dialog.getByLabelText('Mức độ React'), 'intermediate')
    await user.click(dialog.getByRole('button', { name: 'Lưu hồ sơ kỹ năng' }))
    expect(await screen.findByText('Đã lưu hồ sơ kỹ năng')).toBeInTheDocument()
    expect(cap.bodies).toEqual([
      {
        method: 'PUT',
        path: '/users/u-1/skills',
        body: {
          skills: [
            { skillId: 's-1', level: 'senior' },
            { skillId: 's-2', level: 'intermediate' },
          ],
        },
      },
    ])
  })
})

describe('Phòng ban, chức danh, kỹ năng [CF-02]', () => {
  it('cây phòng ban: con nằm dưới cha; thêm phòng ban con gửi parentId', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users?tab=departments')
    const tree = await screen.findByRole('table', { name: 'Cây phòng ban' })
    const names = within(tree)
      .getAllByRole('row')
      .slice(1)
      .map((r) => within(r).getAllByRole('cell')[0]?.textContent)
    expect(names).toEqual(['Khối Kỹ thuật', 'Phòng Backend', 'Phòng Kinh doanh'])
    const row = within(within(tree).getByText('Khối Kỹ thuật').closest('tr') as HTMLElement)
    await user.click(row.getByRole('button', { name: 'Thêm phòng ban con' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Thêm phòng ban' }))
    expect(dialog.getByLabelText('Phòng ban cha')).toHaveValue('d-1')
    await user.type(dialog.getByLabelText('Tên'), 'Phòng Frontend')
    await user.click(dialog.getByRole('button', { name: 'Lưu' }))
    expect(await screen.findByText('Đã thêm')).toBeInTheDocument()
    expect(cap.bodies).toEqual([
      { method: 'POST', path: '/departments', body: { name: 'Phòng Frontend', parentId: 'd-1' } },
    ])
  })

  it('xóa phòng ban còn phòng ban con → hiện lý do từ server, không đóng hộp thoại', async () => {
    const user = userEvent.setup()
    mockAdminApi()
    setup('/admin/users?tab=departments')
    const tree = await screen.findByRole('table', { name: 'Cây phòng ban' })
    const row = within(within(tree).getByText('Khối Kỹ thuật').closest('tr') as HTMLElement)
    await user.click(row.getByRole('button', { name: 'Xóa' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Xóa “Khối Kỹ thuật”?' }))
    await user.click(dialog.getByRole('button', { name: 'Xóa' }))
    expect(await dialog.findByText('Phòng ban còn phòng ban con')).toBeInTheDocument()
  })

  it('sửa phòng ban: không chọn được chính nó hay phòng ban con làm cha', async () => {
    const user = userEvent.setup()
    mockAdminApi()
    setup('/admin/users?tab=departments')
    const tree = await screen.findByRole('table', { name: 'Cây phòng ban' })
    const row = within(within(tree).getByText('Khối Kỹ thuật').closest('tr') as HTMLElement)
    await user.click(row.getByRole('button', { name: 'Đổi tên' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Sửa phòng ban' }))
    const parent = within(dialog.getByLabelText('Phòng ban cha'))
    expect(parent.queryByRole('option', { name: /Khối Kỹ thuật/ })).not.toBeInTheDocument()
    expect(parent.queryByRole('option', { name: /Phòng Backend/ })).not.toBeInTheDocument()
    expect(parent.getByRole('option', { name: /Phòng Kinh doanh/ })).toBeInTheDocument()
  })

  it('thêm chức danh; tên trùng (409) báo ngay dưới ô tên', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users?tab=job-titles')
    await screen.findByRole('table', { name: 'Danh sách chức danh' })
    await user.click(screen.getByRole('button', { name: 'Thêm chức danh' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Thêm chức danh' }))
    await user.type(dialog.getByLabelText('Tên'), 'Business Analyst')
    await user.click(dialog.getByRole('button', { name: 'Lưu' }))
    expect(await screen.findByText('Đã thêm')).toBeInTheDocument()
    expect(cap.bodies).toEqual([
      { method: 'POST', path: '/job-titles', body: { name: 'Business Analyst' } },
    ])

    server.use(
      mock.post(`${API}/api/v1/job-titles`, () =>
        HttpResponse.json(
          { error: { code: 'JOB_TITLE_NAME_TAKEN', message: 'Chức danh đã tồn tại' } },
          { status: 409 },
        ),
      ),
    )
    await user.click(screen.getByRole('button', { name: 'Thêm chức danh' }))
    const again = within(await screen.findByRole('dialog', { name: 'Thêm chức danh' }))
    await user.type(again.getByLabelText('Tên'), 'Lập trình viên')
    await user.click(again.getByRole('button', { name: 'Lưu' }))
    await waitFor(() => {
      expect(again.getByLabelText('Tên')).toHaveAccessibleDescription('Chức danh đã tồn tại')
    })
  })

  it('xóa kỹ năng sau khi xác nhận', async () => {
    const user = userEvent.setup()
    const cap = mockAdminApi()
    setup('/admin/users?tab=skills')
    const list = await screen.findByRole('table', { name: 'Danh mục kỹ năng' })
    const row = within(within(list).getByText('React').closest('tr') as HTMLElement)
    await user.click(row.getByRole('button', { name: 'Xóa' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Xóa “React”?' }))
    await user.click(dialog.getByRole('button', { name: 'Xóa' }))
    expect(await screen.findByText('Đã xóa')).toBeInTheDocument()
    expect(cap.bodies).toEqual([{ method: 'DELETE', path: '/skills/s-2', body: null }])
  })

  it('chuyển tab cập nhật ?tab trên URL', async () => {
    const user = userEvent.setup()
    mockAdminApi()
    const { router } = setup('/admin/users')
    await table()
    await user.click(screen.getByRole('tab', { name: 'Kỹ năng' }))
    expect(router.state.location.search).toBe('?tab=skills')
    expect(await screen.findByRole('table', { name: 'Danh mục kỹ năng' })).toBeInTheDocument()
  })
})
