import { Inbox, type LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/cn'

export interface EmptyStateProps {
  title: string
  description?: ReactNode
  icon?: LucideIcon
  actions?: ReactNode
  tone?: 'empty' | 'error'
  role?: 'alert'
}

/** §7: khung nét đứt, nền page, icon 36px, tiêu đề 14/600, câu hướng dẫn 12px. */
export function EmptyState({
  title,
  description,
  icon: Icon = Inbox,
  actions,
  tone = 'empty',
  role,
}: EmptyStateProps) {
  return (
    <section
      role={role}
      className={cn(
        'flex flex-col items-center gap-2 rounded-md border border-dashed bg-page px-6 py-10 text-center',
        tone === 'error' ? 'border-bad-border' : 'border-line-strong',
      )}
    >
      <Icon
        size={36}
        strokeWidth={2}
        aria-hidden
        className={tone === 'error' ? 'text-bad' : 'text-ink-faint'}
      />
      <h2 className="text-base font-semibold text-ink">{title}</h2>
      {description && <div className="max-w-65 text-sm text-ink-muted">{description}</div>}
      {actions && <div className="mt-2 flex gap-2">{actions}</div>}
    </section>
  )
}
