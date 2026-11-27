import { useLocation } from 'react-router'
import { NotificationBell } from '@/features/notifications'
import { messages } from '@/shared/lib/messages'
import { findMenuItem } from './menu'

/** §4: topbar nền surface, viền dưới, padding 8/20; trái là breadcrumb, phải là chuông thông báo. */
export function Topbar() {
  const { pathname } = useLocation()
  const current = findMenuItem(pathname)
  return (
    <header className="flex min-h-12 items-center border-b border-line bg-surface px-5 py-2">
      <nav aria-label={messages.shell.breadcrumb}>
        <ol className="flex items-center gap-1.5">
          <li className="text-sm text-ink-faint">{messages.appName}</li>
          {current && (
            <>
              <li aria-hidden className="text-sm text-ink-faint">
                ›
              </li>
              <li aria-current="page" className="text-base font-semibold text-ink">
                {current.label}
              </li>
            </>
          )}
        </ol>
      </nav>
      <div className="ml-auto flex items-center gap-3">
        <NotificationBell />
      </div>
    </header>
  )
}
