import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http as mock, HttpResponse } from 'msw'
import type { Session } from '@/features/auth'
import { makeSession } from '@/test/fixtures'
import { renderApp } from '@/test/render'
import { API, server } from '@/test/server'

let current: Session | undefined
function setup(...args: Parameters<typeof renderApp>) {
  const r = renderApp(...args)
  current = r.session
  return r
}
afterEach(() => {
  current?.dispose()
  current = undefined
})

const sidebarLinks = async () => {
  const nav = await screen.findByRole('navigation', { name: 'Điều hướng chính' })
  return within(nav)
    .getAllByRole('link')
    .map((a) => a.textContent)
}

describe('Route guard [CF-01]', () => {
  it('chưa đăng nhập vào trang trong → chuyển /login kèm next', async () => {
    const { router } = setup('/reports?tab=sp', 'anonymous')
    expect(await screen.findByRole('heading', { name: 'Đăng nhập' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
    expect(router.state.location.search).toBe(`?next=${encodeURIComponent('/reports?tab=sp')}`)
  })

  it('đang kiểm tra phiên → hiện skeleton có nhãn, không nháy trang đăng nhập', () => {
    setup('/', 'manager')
    expect(
      screen.getByRole('status', { name: 'Đang kiểm tra phiên đăng nhập…' }),
    ).toBeInTheDocument()
  })

  it('Manager gõ thẳng /admin/users → trạng thái không có quyền nêu vai trò', async () => {
    const user = userEvent.setup()
    const { router } = setup('/admin/users', 'manager')
    expect(
      await screen.findByRole('heading', { name: 'Bạn không có quyền xem trang này' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Vai trò Quản lý dự án/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Về trang của tôi' }))
    expect(router.state.location.pathname).toBe('/')
  })

  it('Admin vào /admin/users → được vào', async () => {
    setup('/admin/users', 'admin')
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Nhân sự & phòng ban' }),
    ).toBeInTheDocument()
  })

  it('đường dẫn không tồn tại → trang không tìm thấy trong app shell', async () => {
    setup('/khong-co', 'member')
    expect(await screen.findByRole('heading', { name: 'Không tìm thấy trang' })).toBeInTheDocument()
  })
})

describe('Sidebar theo vai trò [CF-01]', () => {
  it.each([
    ['admin', ['Trang của tôi', 'Nhân sự & phòng ban']],
    ['ceo', ['Trang của tôi', 'Dự án', 'Dashboard toàn công ty', 'Nhân sự & phòng ban']],
    ['manager', ['Trang của tôi', 'Dự án', 'Công việc của tôi', 'Báo cáo hiệu suất']],
    ['member', ['Trang của tôi', 'Dự án', 'Công việc của tôi']],
  ] as const)('%s thấy đúng các mục', async (role, expected) => {
    setup('/', role)
    expect(await sidebarLinks()).toEqual(expected)
  })

  it('mục đang chọn có aria-current, breadcrumb hiện tên mục', async () => {
    setup('/projects', 'manager')
    const link = await screen.findByRole('link', { name: 'Dự án' })
    expect(link).toHaveAttribute('aria-current', 'page')
    const crumb = screen.getByRole('navigation', { name: 'Vị trí hiện tại' })
    expect(within(crumb).getByText('Dự án')).toHaveAttribute('aria-current', 'page')
  })

  it('trang chủ chào theo tên và vai trò', async () => {
    setup('/', 'member')
    expect(await screen.findByRole('heading', { name: 'Xin chào, Đỗ Thu Hà' })).toBeInTheDocument()
    expect(screen.getByText('Bạn đang đăng nhập với vai trò Thành viên.')).toBeInTheDocument()
  })
})

describe('Đăng nhập [CF-01][UC01]', () => {
  it('câu "Chưa có tài khoản?" chỉ hiện một lần', async () => {
    setup('/login')
    await screen.findByRole('button', { name: 'Đăng nhập' })
    expect(screen.getAllByText(/Chưa có tài khoản\?/)).toHaveLength(1)
  })

  it('bỏ trống → báo lỗi dưới ô, không gọi API', async () => {
    const user = userEvent.setup()
    const login = vi.fn()
    server.use(mock.post(`${API}/api/v1/auth/login`, login))
    setup('/login')
    await user.click(await screen.findByRole('button', { name: 'Đăng nhập' }))
    expect(screen.getByLabelText('Email')).toHaveAccessibleDescription('Vui lòng nhập email.')
    expect(screen.getByLabelText('Mật khẩu')).toHaveAccessibleDescription('Vui lòng nhập mật khẩu.')
    expect(login).not.toHaveBeenCalled()
  })

  it('sai mật khẩu → hiện thông báo của máy chủ, ở lại trang', async () => {
    const user = userEvent.setup()
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () =>
        HttpResponse.json(
          {
            error: {
              code: 'AUTH_INVALID_CREDENTIALS',
              message: 'Email hoặc mật khẩu không đúng',
            },
          },
          { status: 401 },
        ),
      ),
    )
    setup('/login')
    await user.type(await screen.findByLabelText('Email'), 'manager@worktrack.local')
    await user.type(screen.getByLabelText('Mật khẩu'), 'sai-mat-khau1')
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Email hoặc mật khẩu không đúng')
  })

  it('đúng → về trang next và sidebar theo vai trò', async () => {
    const user = userEvent.setup()
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () => HttpResponse.json(makeSession('manager'))),
    )
    const { router } = setup(`/login?next=${encodeURIComponent('/reports')}`)
    await user.type(await screen.findByLabelText('Email'), 'manager@worktrack.local')
    await user.type(screen.getByLabelText('Mật khẩu'), 'matkhau123')
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/reports')
    })
    expect(await sidebarLinks()).toContain('Báo cáo hiệu suất')
  })

  it('next trỏ ra ngoài (//evil) → bị bỏ qua, về trang của tôi', async () => {
    const user = userEvent.setup()
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () => HttpResponse.json(makeSession('member'))),
    )
    const { router } = setup(`/login?next=${encodeURIComponent('//evil.example')}`)
    await user.type(await screen.findByLabelText('Email'), 'member@worktrack.local')
    await user.type(screen.getByLabelText('Mật khẩu'), 'matkhau123')
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/')
    })
  })

  it('đã đăng nhập mở /login → chuyển về trang của tôi', async () => {
    const { router } = setup('/login', 'ceo')
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/')
    })
  })
})

describe('Đăng xuất [CF-01]', () => {
  it('đang đăng xuất → nút bị khóa, bấm lần hai không gọi API lần nữa', async () => {
    const user = userEvent.setup()
    let release: () => void = () => undefined
    const logout = vi.fn(
      () =>
        new Promise<HttpResponse<null>>((resolve) => {
          release = () => {
            resolve(new HttpResponse(null, { status: 204 }))
          }
        }),
    )
    server.use(mock.post(`${API}/api/v1/auth/logout`, logout))
    const { router } = setup('/', 'member')
    const button = await screen.findByRole('button', { name: 'Đăng xuất' })
    await user.click(button)
    await waitFor(() => {
      expect(button).toBeDisabled()
    })
    await user.click(button)
    release()
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/login')
    })
    expect(logout).toHaveBeenCalledTimes(1)
  })

  it('bấm Đăng xuất → gọi API, toast, về /login', async () => {
    const user = userEvent.setup()
    const logout = vi.fn(() => new HttpResponse(null, { status: 204 }))
    server.use(mock.post(`${API}/api/v1/auth/logout`, logout))
    const { router } = setup('/', 'member')
    await user.click(await screen.findByRole('button', { name: 'Đăng xuất' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/login')
    })
    expect(logout).toHaveBeenCalledTimes(1)
    expect(screen.getByText('Đã đăng xuất')).toBeInTheDocument()
  })
})

describe('Kích hoạt tài khoản [CF-01][UC02]', () => {
  it('chỉ có một tiêu đề trang (không lồng khung)', async () => {
    setup('/activate')
    await screen.findByRole('heading', { name: 'Thiếu liên kết kích hoạt' })
    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
  })

  it('lỗi mạng khi kiểm tra liên kết → báo lỗi tải và cho thử lại', async () => {
    const user = userEvent.setup()
    let calls = 0
    server.use(
      mock.post(`${API}/api/v1/auth/activation/verify`, () => {
        calls++
        return calls === 1
          ? HttpResponse.error()
          : HttpResponse.json({ email: 'ha.do@worktrack.local', fullName: 'Đỗ Thu Hà' })
      }),
    )
    setup('/activate?token=tok-abc')
    await user.click(await screen.findByRole('button', { name: 'Thử lại' }))
    expect(await screen.findByText('Đỗ Thu Hà')).toBeInTheDocument()
  })

  it('thiếu token → báo thiếu liên kết, có nút về đăng nhập', async () => {
    setup('/activate')
    expect(
      await screen.findByRole('heading', { name: 'Thiếu liên kết kích hoạt' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Về trang đăng nhập' })).toBeInTheDocument()
  })

  it('token hết hạn → hiện thông báo của máy chủ', async () => {
    server.use(
      mock.post(`${API}/api/v1/auth/activation/verify`, () =>
        HttpResponse.json(
          {
            error: {
              code: 'AUTH_ACTIVATION_INVALID',
              message: 'Liên kết kích hoạt không hợp lệ hoặc đã hết hạn',
            },
          },
          { status: 400 },
        ),
      ),
    )
    setup('/activate?token=cu')
    expect(
      await screen.findByRole('heading', { name: 'Liên kết kích hoạt không dùng được' }),
    ).toBeInTheDocument()
    expect(screen.getByText('Liên kết kích hoạt không hợp lệ hoặc đã hết hạn')).toBeInTheDocument()
  })

  it('token hợp lệ → xóa token khỏi URL, đặt mật khẩu, toast và về /login', async () => {
    const user = userEvent.setup()
    const activate = vi.fn<(body: unknown) => void>()
    server.use(
      mock.post(`${API}/api/v1/auth/activation/verify`, () =>
        HttpResponse.json({ email: 'ha.do@worktrack.local', fullName: 'Đỗ Thu Hà' }),
      ),
      mock.post(`${API}/api/v1/auth/activation`, async ({ request }) => {
        activate(await request.json())
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const { router } = setup('/activate?token=tok-abc')
    expect(await screen.findByText('Đỗ Thu Hà')).toBeInTheDocument()
    expect(screen.getByText(/Tài khoản:/)).toHaveTextContent(
      'Tài khoản: Đỗ Thu Hà · ha.do@worktrack.local',
    )
    expect(router.state.location.search).toBe('')

    await user.type(screen.getByLabelText('Mật khẩu mới'), 'matkhau123')
    await user.type(screen.getByLabelText('Nhập lại mật khẩu'), 'matkhau124')
    await user.click(screen.getByRole('button', { name: 'Kích hoạt tài khoản' }))
    expect(screen.getByLabelText('Nhập lại mật khẩu')).toHaveAccessibleDescription(
      'Hai mật khẩu chưa khớp.',
    )
    expect(activate).not.toHaveBeenCalled()

    await user.clear(screen.getByLabelText('Nhập lại mật khẩu'))
    await user.type(screen.getByLabelText('Nhập lại mật khẩu'), 'matkhau123')
    await user.click(screen.getByRole('button', { name: 'Kích hoạt tài khoản' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/login')
    })
    expect(activate).toHaveBeenCalledWith({ token: 'tok-abc', password: 'matkhau123' })
    expect(screen.getByText('Đã kích hoạt tài khoản. Vui lòng đăng nhập.')).toBeInTheDocument()
  })
})
