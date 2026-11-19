import { Clock, LogOut } from 'lucide-react'
import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router'
import { canAccess, useAuth } from '@/features/auth'
import { cn } from '@/shared/lib/cn'
import { messages } from '@/shared/lib/messages'
import { Avatar, useToast } from '@/shared/ui'
import { MENU, MENU_GROUPS } from './menu'

/** §4: sidebar 236px, logo, nhóm menu theo vai trò, đáy là người dùng hiện tại. */
export function Sidebar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const toast = useToast()
  const [loggingOut, setLoggingOut] = useState(false)
  if (!user) return null
  const role = messages.roles[user.role]
  const items = MENU.filter((m) => canAccess(user, m.access))

  return (
    <aside className="flex w-59 shrink-0 flex-col border-r border-line bg-surface px-2.5 py-3">
      <div className="flex items-center gap-2 px-1.5 pb-4">
        <span className="flex size-7 items-center justify-center rounded-md bg-accent text-white">
          <Clock size={16} strokeWidth={2} aria-hidden />
        </span>
        <div className="flex flex-col leading-tight">
          <span className="text-base font-semibold text-ink">{messages.appName}</span>
          <span className="text-sm text-ink-faint">{role}</span>
        </div>
      </div>

      <nav aria-label={messages.shell.mainNav} className="flex flex-1 flex-col gap-4 overflow-y-auto">
        {MENU_GROUPS.map((group) => {
          const groupItems = items.filter((m) => m.group === group)
          if (groupItems.length === 0) return null
          const headingId = `nav-group-${group}`
          return (
            <section key={group} aria-labelledby={headingId} className="flex flex-col gap-0.5">
              <h2 id={headingId} className="px-2.5 pb-1 text-sm font-semibold text-ink-faint">
                {messages.shell.groups[group]}
              </h2>
              <ul className="flex flex-col gap-0.5">
                {groupItems.map(({ path, label, icon: Icon }) => (
                  <li key={path}>
                    <NavLink
                      to={path}
                      end={path === '/'}
                      className={({ isActive }) =>
                        cn(
                          'flex h-10 items-center gap-2 rounded-md px-2.5 text-base',
                          'focus-visible:outline-2 focus-visible:outline-accent',
                          isActive
                            ? 'bg-accent-subtle font-semibold text-accent-hover'
                            : 'text-ink-muted hover:bg-subtle',
                        )
                      }
                    >
                      <Icon size={16} strokeWidth={2} aria-hidden />
                      {label}
                    </NavLink>
                  </li>
                ))}
              </ul>
            </section>
          )
        })}
      </nav>

      <div className="flex items-center gap-2 border-t border-line px-1.5 pt-3">
        <Avatar name={user.fullName} size={28} />
        <div className="flex min-w-0 flex-1 flex-col leading-tight">
          <span className="truncate text-base font-semibold text-ink">{user.fullName}</span>
          <span className="truncate text-sm text-ink-faint">{role}</span>
        </div>
        <button
          type="button"
          aria-label={messages.auth.logout}
          title={messages.auth.logout}
          disabled={loggingOut}
          aria-busy={loggingOut}
          onClick={() => {
            setLoggingOut(true)
            // session.logout luôn xóa phiên cục bộ kể cả khi API lỗi → không cần nhánh catch.
            void logout().then(() => {
              toast.show(messages.auth.loggedOut)
              void navigate('/login', { replace: true })
            })
          }}
          className="flex size-10 items-center justify-center rounded-md text-ink-muted hover:bg-subtle focus-visible:outline-2 focus-visible:outline-accent disabled:cursor-not-allowed disabled:text-disabled disabled:hover:bg-transparent"
        >
          <LogOut size={16} strokeWidth={2} aria-hidden />
        </button>
      </div>
    </aside>
  )
}
