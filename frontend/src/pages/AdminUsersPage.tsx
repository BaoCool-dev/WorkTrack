import { AdminOrgView, type ExtraAdminTab } from '@/features/admin'
import { LabelsPanel, SettingsPanel } from '@/features/tasks'
import { messages } from '@/shared/lib/messages'

// Danh mục nhãn và tham số B1 thuộc feature tasks (CF-06), hiện chung trang Quản trị.
const EXTRA_TABS: readonly ExtraAdminTab[] = [
  { value: 'labels', label: messages.admin.tabs.labels, render: () => <LabelsPanel /> },
  { value: 'settings', label: messages.admin.tabs.settings, render: () => <SettingsPanel /> },
]

export function AdminUsersPage() {
  return (
    <div className="px-5 py-3">
      <AdminOrgView extraTabs={EXTRA_TABS} />
    </div>
  )
}
