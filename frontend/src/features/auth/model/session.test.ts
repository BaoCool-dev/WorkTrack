import { http as mock, HttpResponse } from 'msw'
import { makeSession } from '@/test/fixtures'
import { API, server } from '@/test/server'
import { createSession } from './session'

const unauthenticated = () =>
  HttpResponse.json({ error: { code: 'UNAUTHENTICATED', message: 'x' } }, { status: 401 })

describe('session', () => {
  it('start: làm mới thất bại → anonymous, không có token', async () => {
    server.use(mock.post(`${API}/api/v1/auth/refresh`, unauthenticated))
    const s = createSession()
    await s.start()
    expect(s.getSnapshot()).toEqual({ status: 'anonymous', user: null })
    expect(s.getToken()).toBeNull()
    s.dispose()
  })

  it('start: còn cookie hợp lệ → authenticated và giữ token trong bộ nhớ', async () => {
    server.use(
      mock.post(`${API}/api/v1/auth/refresh`, () => HttpResponse.json(makeSession('manager'))),
    )
    const s = createSession()
    await s.start()
    expect(s.getSnapshot().status).toBe('authenticated')
    expect(s.getSnapshot().user?.role).toBe('manager')
    expect(s.getToken()).toBe('tok-1')
    s.dispose()
  })

  it('refresh single-flight: gọi đồng thời chỉ gửi một request (refresh token xoay vòng)', async () => {
    let calls = 0
    server.use(
      mock.post(`${API}/api/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json(makeSession('member', `tok-${String(calls)}`))
      }),
    )
    const s = createSession()
    const [a, b] = await Promise.all([s.refresh(), s.refresh(), s.start()])
    expect(calls).toBe(1)
    expect(a).toBe('tok-1')
    expect(b).toBe('tok-1')
    s.dispose()
  })

  it('start gọi nhiều lần (StrictMode) chỉ làm mới một lần', async () => {
    let calls = 0
    server.use(
      mock.post(`${API}/api/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json(makeSession())
      }),
    )
    const s = createSession()
    await Promise.all([s.start(), s.start()])
    await s.start()
    expect(calls).toBe(1)
    s.dispose()
  })

  it('login thành công → authenticated và báo cho subscriber', async () => {
    server.use(mock.post(`${API}/api/v1/auth/login`, () => HttpResponse.json(makeSession('admin'))))
    const s = createSession()
    const listener = vi.fn()
    s.subscribe(listener)
    await s.login('admin@worktrack.local', 'Matkhau123')
    expect(s.getSnapshot().user?.role).toBe('admin')
    expect(listener).toHaveBeenCalled()
    s.dispose()
  })

  it('login sai → ném ApiError, giữ nguyên anonymous', async () => {
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () =>
        HttpResponse.json(
          {
            error: { code: 'AUTH_INVALID_CREDENTIALS', message: 'Email hoặc mật khẩu không đúng' },
          },
          { status: 401 },
        ),
      ),
      mock.post(`${API}/api/v1/auth/refresh`, unauthenticated),
    )
    const s = createSession()
    await s.start()
    await expect(s.login('a@b.vn', 'sai12345')).rejects.toMatchObject({
      code: 'AUTH_INVALID_CREDENTIALS',
    })
    expect(s.getSnapshot().status).toBe('anonymous')
    s.dispose()
  })

  it('logout: xóa phiên kể cả khi máy chủ lỗi', async () => {
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () => HttpResponse.json(makeSession())),
      mock.post(`${API}/api/v1/auth/logout`, () => HttpResponse.error()),
    )
    const s = createSession()
    await s.login('member@worktrack.local', 'Matkhau123')
    await s.logout()
    expect(s.getSnapshot()).toEqual({ status: 'anonymous', user: null })
    expect(s.getToken()).toBeNull()
    s.dispose()
  })

  it('làm mới thất bại giữa phiên → chuyển anonymous', async () => {
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () => HttpResponse.json(makeSession())),
      mock.post(`${API}/api/v1/auth/refresh`, unauthenticated),
    )
    const s = createSession()
    await s.login('member@worktrack.local', 'Matkhau123')
    await expect(s.refresh()).resolves.toBeNull()
    expect(s.getSnapshot().status).toBe('anonymous')
    s.dispose()
  })

  it('tự làm mới trước khi access token hết hạn 60 giây', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    let calls = 0
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () => HttpResponse.json(makeSession('member', 'dau'))),
      mock.post(`${API}/api/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json(makeSession('member', 'moi'))
      }),
    )
    const s = createSession()
    await s.login('member@worktrack.local', 'Matkhau123')
    await vi.advanceTimersByTimeAsync(14 * 60_000 + 1_000)
    await vi.waitFor(() => {
      expect(s.getToken()).toBe('moi')
    })
    expect(calls).toBe(1)
    s.dispose()
  })
})
