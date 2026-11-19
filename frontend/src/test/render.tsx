import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { http as mock, HttpResponse } from 'msw'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { routes } from '@/app/routes'
import { AuthProvider, createSession, type Role } from '@/features/auth'
import { ToastProvider } from '@/shared/ui'
import { makeSession } from './fixtures'
import { API, server } from './server'

/**
 * Dựng toàn bộ app (providers + routes thật) trên memory router.
 * as: vai trò đã đăng nhập (refresh cookie hợp lệ) hoặc 'anonymous' (refresh → 401).
 */
export function renderApp(path: string, as: Role | 'anonymous' = 'anonymous') {
  server.use(
    mock.post(`${API}/api/v1/auth/refresh`, () =>
      as === 'anonymous'
        ? HttpResponse.json(
            { error: { code: 'UNAUTHENTICATED', message: 'Phiên đăng nhập đã hết hạn' } },
            { status: 401 },
          )
        : HttpResponse.json(makeSession(as)),
    ),
    mock.get(`${API}/healthz`, () => HttpResponse.json({ status: 'ok' })),
  )
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const session = createSession()
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <AuthProvider session={session}>
          <RouterProvider router={router} />
        </AuthProvider>
      </ToastProvider>
    </QueryClientProvider>,
  )
  return { ...utils, router, session }
}
