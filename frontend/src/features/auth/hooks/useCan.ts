import type { AccessRule } from '../model/access'
import { canAccess, hasPermission } from '../model/access'
import type { Permission } from '../model/types'
import { useAuth } from './useAuth'

/**
 * Ẩn/hiện theo quyền ở UI. Quyền theo dự án (resource) thêm ở feature/projects
 * khi backend có quyền theo dự án.
 */
export function useCan(permission: Permission): boolean {
  return hasPermission(useAuth().user, permission)
}

export function useCanAccess(rule?: AccessRule): boolean {
  return canAccess(useAuth().user, rule)
}
