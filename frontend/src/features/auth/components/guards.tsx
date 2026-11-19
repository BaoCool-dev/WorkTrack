import { Navigate, Outlet, useLocation } from 'react-router'
import { useAuth } from '../hooks/useAuth'
import { canAccess, type AccessRule } from '../model/access'
import { ForbiddenState } from './ForbiddenState'
import { SessionLoading } from './SessionLoading'

/** Chặn route khi chưa đăng nhập → /login?next=<trang đang mở>. */
export function RequireAuth() {
  const { status } = useAuth()
  const location = useLocation()
  if (status === 'loading') return <SessionLoading />
  if (status === 'anonymous') {
    const next = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`/login?next=${next}`} replace />
  }
  return <Outlet />
}

/** Chặn route theo vai trò/quyền — chỉ là lớp UI, backend vẫn kiểm tra lại. */
export function RequireAccess({ rule }: { rule: AccessRule }) {
  const { user } = useAuth()
  return canAccess(user, rule) ? <Outlet /> : <ForbiddenState />
}
