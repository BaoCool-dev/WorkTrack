import type { ReactNode } from 'react'
import type { RouteObject } from 'react-router'
import { RequireAccess, RequireAuth } from '@/features/auth'
import { ActivatePage } from '@/pages/ActivatePage'
import { AdminUsersPage } from '@/pages/AdminUsersPage'
import { HomePage } from '@/pages/HomePage'
import { LoginPage } from '@/pages/LoginPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { PlaceholderPage } from '@/pages/PlaceholderPage'
import { AppLayout } from './layout/AppLayout'
import { MENU } from './layout/menu'

// Route đã có màn thật; phần còn lại của menu vẫn là trang giữ chỗ.
const PAGES: Record<string, ReactNode> = {
  '/admin/users': <AdminUsersPage />,
}

// Tách khỏi router.tsx để test dựng bằng createMemoryRouter(routes).
export const routes: RouteObject[] = [
  { path: '/login', element: <LoginPage /> },
  { path: '/activate', element: <ActivatePage /> },
  {
    element: <RequireAuth />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { index: true, element: <HomePage /> },
          // Guard dùng chung luật với sidebar (menu.ts) — gõ thẳng URL cũng bị chặn ở UI;
          // backend vẫn là nơi chặn thật.
          ...MENU.filter((m) => m.path !== '/').map((m): RouteObject => ({
            element: <RequireAccess rule={m.access} />,
            children: [
              { path: m.path, element: PAGES[m.path] ?? <PlaceholderPage title={m.label} /> },
            ],
          })),
          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
]
