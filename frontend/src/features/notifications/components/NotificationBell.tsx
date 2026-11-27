import { Bell, BellOff } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { cn } from '@/shared/lib/cn'
import { formatNumber, formatTimestampDate } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState, ErrorState, Skeleton, useToast } from '@/shared/ui'
import { useMarkAllRead, useMarkRead, useNotifications, useUnreadCount } from '../api/queries'
import type { AppNotification } from '../model/types'

const n = messages.notifications

/** Câu hiển thị dựng từ loại + người làm + tên việc; server chỉ gửi dữ liệu, không gửi câu. */
function describe(item: AppNotification): string {
  return n.text[item.type](item.actor?.name ?? n.someone, item.task.name)
}

/**
 * §4: nút chuông 34px ở Topbar, chấm đỏ 6px khi có thông báo chưa đọc (CF-07).
 * Bấm mở bảng thông báo ngay dưới; Escape hoặc bấm ra ngoài để đóng.
 */
export function NotificationBell() {
  const [open, setOpen] = useState(false)
  const unread = useUnreadCount()
  const count = unread.data ?? 0
  const wrapRef = useRef<HTMLDivElement>(null)
  const panelId = useId()

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    const onClick = (e: MouseEvent) => {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    document.addEventListener('mousedown', onClick)
    return () => {
      document.removeEventListener('keydown', onKey)
      document.removeEventListener('mousedown', onClick)
    }
  }, [open])

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        aria-label={count > 0 ? n.bellUnread(formatNumber(count)) : n.bell}
        aria-expanded={open}
        aria-controls={open ? panelId : undefined}
        onClick={() => {
          setOpen((v) => !v)
        }}
        className={cn(
          'relative inline-flex size-[34px] items-center justify-center rounded-md border border-line text-ink-muted',
          'hover:bg-subtle focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent',
        )}
      >
        <Bell size={16} strokeWidth={2} aria-hidden />
        {count > 0 && (
          <span aria-hidden className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-bad" />
        )}
      </button>
      {open && <NotificationPanel id={panelId} hasUnread={count > 0} />}
    </div>
  )
}

function NotificationPanel({ id, hasUnread }: { id: string; hasUnread: boolean }) {
  const query = useNotifications(true)
  const markAll = useMarkAllRead()
  const toast = useToast()
  const titleId = useId()

  let content
  if (query.isError)
    content = (
      <ErrorState
        title={n.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  else if (!query.data)
    content = (
      <div className="flex flex-col gap-2 p-3">
        <Skeleton className="h-11 w-full" />
        <Skeleton className="h-11 w-full" />
      </div>
    )
  else if (query.data.items.length === 0)
    content = (
      <div className="p-3">
        <EmptyState icon={BellOff} title={n.emptyTitle} description={n.emptyHint} />
      </div>
    )
  else
    content = (
      <ul className="max-h-96 overflow-y-auto">
        {query.data.items.map((item) => (
          <NotificationRow key={item.id} item={item} />
        ))}
      </ul>
    )

  return (
    <section
      id={id}
      aria-labelledby={titleId}
      className="absolute top-full right-0 z-30 mt-2 w-[360px] rounded-md border border-line bg-surface shadow-overlay"
    >
      <div className="flex items-center gap-2 border-b border-line px-3 py-2">
        <h2 id={titleId} className="text-base font-semibold text-ink">
          {n.title}
        </h2>
        {hasUnread && (
          <Button
            variant="ghost"
            className="ml-auto"
            loading={markAll.isPending}
            onClick={() => {
              markAll.mutate(undefined, {
                onSuccess: () => {
                  toast.show(n.markedAll)
                },
              })
            }}
          >
            {n.markAll}
          </Button>
        )}
      </div>
      {content}
    </section>
  )
}

function NotificationRow({ item }: { item: AppNotification }) {
  const markRead = useMarkRead()
  const text = describe(item)
  const body = (
    <>
      <span className="flex items-start gap-2">
        <span className={cn('text-base text-ink', !item.read && 'font-semibold')}>{text}</span>
        {!item.read && (
          <span className="ml-auto shrink-0 rounded-sm border border-bad-border bg-bad-bg px-1.5 text-sm font-semibold text-bad">
            {n.unread}
          </span>
        )}
      </span>
      {item.message && (
        <span className="line-clamp-2 text-sm text-ink-muted">“{item.message}”</span>
      )}
      <span className="text-sm text-ink-faint">
        {item.project.name} · <span className="num">{formatTimestampDate(item.createdAt)}</span>
      </span>
    </>
  )
  const cls = 'flex w-full flex-col gap-0.5 px-3 py-2 text-left'
  return (
    <li className="border-b border-line last:border-b-0">
      {item.read ? (
        <div className={cls}>{body}</div>
      ) : (
        <button
          type="button"
          aria-label={n.markRead(text)}
          disabled={markRead.isPending}
          onClick={() => {
            markRead.mutate(item.id)
          }}
          className={cn(
            cls,
            'bg-accent-subtle hover:bg-subtle focus-visible:outline-2 focus-visible:outline-accent',
          )}
        >
          {body}
        </button>
      )}
    </li>
  )
}
