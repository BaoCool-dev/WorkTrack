import {
  Building2,
  ChartColumn,
  FolderKanban,
  House,
  ListChecks,
  Users,
  type LucideIcon,
} from 'lucide-react'
import type { AccessRule } from '@/features/auth'
import { messages } from '@/shared/lib/messages'

export type MenuGroup = keyof typeof messages.shell.groups

export interface MenuItem {
  path: string
  label: string
  icon: LucideIcon
  group: MenuGroup
  /** Dùng chung cho cả sidebar (ẩn mục) và route guard (chặn khi gõ thẳng URL). */
  access: AccessRule
}

const nav = messages.shell.nav

// Một nguồn duy nhất: thêm màn hình mới → thêm một dòng ở đây.
export const MENU: readonly MenuItem[] = [
  { path: '/', label: nav.home, icon: House, group: 'work', access: {} },
  {
    path: '/projects',
    label: nav.projects,
    icon: FolderKanban,
    group: 'work',
    access: { roles: ['manager', 'member', 'ceo'] },
  },
  {
    path: '/my-tasks',
    label: nav.myTasks,
    icon: ListChecks,
    group: 'work',
    access: { roles: ['manager', 'member'] },
  },
  {
    path: '/reports',
    label: nav.teamReports,
    icon: ChartColumn,
    group: 'track',
    access: { permission: 'dashboard.team.view' },
  },
  {
    path: '/company',
    label: nav.companyDashboard,
    icon: Building2,
    group: 'track',
    access: { permission: 'dashboard.company.view' },
  },
  {
    path: '/admin/users',
    label: nav.users,
    icon: Users,
    group: 'admin',
    access: { permission: 'users.manage' },
  },
]

export const MENU_GROUPS: readonly MenuGroup[] = ['work', 'track', 'admin']

/** Mục menu khớp đường dẫn hiện tại (khớp dài nhất) — dùng cho breadcrumb. */
export function findMenuItem(pathname: string): MenuItem | undefined {
  return [...MENU]
    .filter((m) => (m.path === '/' ? pathname === '/' : pathname.startsWith(m.path)))
    .sort((a, b) => b.path.length - a.path.length)[0]
}
