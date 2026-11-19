import type { CurrentUser, Permission, Role } from './types'

/** Điều kiện truy cập một màn hình/mục menu. Rỗng = mọi người dùng đã đăng nhập. */
export interface AccessRule {
  roles?: readonly Role[]
  permission?: Permission
}

export function hasPermission(user: CurrentUser | null, permission: Permission): boolean {
  return user?.permissions.includes(permission) ?? false
}

/** Chỉ để ẩn/hiện ở UI — backend mới là nơi chặn thật (CLAUDE.md §5.1). */
export function canAccess(user: CurrentUser | null, rule: AccessRule = {}): boolean {
  if (!user) return false
  if (rule.roles && !rule.roles.includes(user.role)) return false
  if (rule.permission && !hasPermission(user, rule.permission)) return false
  return true
}

/**
 * Chỉ nhận đường dẫn nội bộ cho tham số ?next= — chặn open redirect
 * ("//evil.com", "/\evil.com", "https://…").
 */
export function safeNext(next: string | null | undefined): string {
  if (!next?.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/'
  if (next.startsWith('/login') || next.startsWith('/activate')) return '/'
  return next
}
