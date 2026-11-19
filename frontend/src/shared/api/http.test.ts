import { http as mock, HttpResponse } from 'msw'
import { API, server } from '@/test/server'
import { ApiError, configureHttpAuth, http } from './http'

afterEach(() => {
  configureHttpAuth(null)
})

describe('http', () => {
  it('trả về undefined khi máy chủ trả 204', async () => {
    server.use(
      mock.post(`${API}/api/v1/auth/logout`, () => new HttpResponse(null, { status: 204 })),
    )

    await expect(
      http<undefined>('/api/v1/auth/logout', { method: 'POST', auth: false }),
    ).resolves.toBeUndefined()
  })

  it('gửi body JSON và đọc lỗi theo định dạng {error:{code,message}}', async () => {
    server.use(
      mock.post(`${API}/api/v1/auth/login`, async ({ request }) => {
        expect(request.headers.get('Content-Type')).toBe('application/json')
        expect(await request.json()).toEqual({ email: 'a@b.vn' })
        return HttpResponse.json(
          {
            error: { code: 'AUTH_INVALID_CREDENTIALS', message: 'Email hoặc mật khẩu không đúng' },
          },
          { status: 401 },
        )
      }),
    )

    const err = await http('/api/v1/auth/login', {
      method: 'POST',
      json: { email: 'a@b.vn' },
      auth: false,
    }).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 401, code: 'AUTH_INVALID_CREDENTIALS' })
  })

  it('gắn Bearer token khi đã cấu hình phiên', async () => {
    configureHttpAuth({ getToken: () => 'tok-1', refresh: () => Promise.resolve(null) })
    server.use(
      mock.get(`${API}/api/v1/auth/me`, ({ request }) =>
        HttpResponse.json({ auth: request.headers.get('Authorization') }),
      ),
    )

    await expect(http('/api/v1/auth/me')).resolves.toEqual({ auth: 'Bearer tok-1' })
  })

  it('401 → làm mới token một lần rồi gửi lại request', async () => {
    let token = 'het-han'
    const refresh = vi.fn(() => {
      token = 'moi'
      return Promise.resolve(token)
    })
    configureHttpAuth({ getToken: () => token, refresh })
    server.use(
      mock.get(`${API}/api/v1/users`, ({ request }) =>
        request.headers.get('Authorization') === 'Bearer moi'
          ? HttpResponse.json({ ok: true })
          : HttpResponse.json(
              { error: { code: 'UNAUTHENTICATED', message: 'x' } },
              { status: 401 },
            ),
      ),
    )

    await expect(http('/api/v1/users')).resolves.toEqual({ ok: true })
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('401 và làm mới thất bại → ném ApiError 401, không lặp vô hạn', async () => {
    const refresh = vi.fn(() => Promise.resolve(null))
    configureHttpAuth({ getToken: () => 'cu', refresh })
    server.use(
      mock.get(`${API}/api/v1/users`, () =>
        HttpResponse.json({ error: { code: 'UNAUTHENTICATED', message: 'x' } }, { status: 401 }),
      ),
    )

    await expect(http('/api/v1/users')).rejects.toMatchObject({ status: 401 })
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('request auth:false không kích hoạt làm mới khi 401', async () => {
    const refresh = vi.fn(() => Promise.resolve('x'))
    configureHttpAuth({ getToken: () => null, refresh })
    server.use(
      mock.post(`${API}/api/v1/auth/login`, () =>
        HttpResponse.json(
          { error: { code: 'AUTH_INVALID_CREDENTIALS', message: 'x' } },
          { status: 401 },
        ),
      ),
    )

    await expect(http('/api/v1/auth/login', { method: 'POST', auth: false })).rejects.toMatchObject(
      { status: 401 },
    )
    expect(refresh).not.toHaveBeenCalled()
  })

  it('lỗi mạng → ApiError NETWORK', async () => {
    server.use(mock.get(`${API}/healthz`, () => HttpResponse.error()))

    await expect(http('/healthz')).rejects.toMatchObject({ status: 0, code: 'NETWORK' })
  })
})
