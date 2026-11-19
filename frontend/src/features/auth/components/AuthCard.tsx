import { Clock } from 'lucide-react'
import type { ReactNode } from 'react'
import { messages } from '@/shared/lib/messages'

/** Khung chung cho màn hình đăng nhập / kích hoạt: logo + tiêu đề + nội dung. */
export function AuthCard({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle: string
  children: ReactNode
}) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-page p-6">
      <div className="flex w-full max-w-100 flex-col gap-5 rounded-md border border-line bg-surface p-6">
        <div className="flex items-center gap-2">
          <span className="flex size-7 items-center justify-center rounded-md bg-accent text-white">
            <Clock size={16} strokeWidth={2} aria-hidden />
          </span>
          <span className="text-base font-semibold text-ink">{messages.appName}</span>
        </div>
        <header className="flex flex-col gap-1">
          <h1 className="text-lg font-semibold text-ink">{title}</h1>
          <p className="text-sm text-ink-muted">{subtitle}</p>
        </header>
        {children}
      </div>
    </main>
  )
}
