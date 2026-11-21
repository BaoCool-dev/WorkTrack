import { CircleAlert, WifiOff, X } from 'lucide-react'

import { messages } from '@/shared/lib/messages'

const t = messages.kanban

/**
 * Dải cảnh báo mất kết nối realtime. Bảng vẫn dùng được (kéo thả đi qua HTTP),
 * chỉ là thay đổi của người khác không hiện ngay — nên là `warn`, không phải `bad`.
 */
export function DisconnectBanner() {
  return (
    <div
      role="status"
      className="flex items-center gap-2 rounded-md border border-warn-border bg-warn-bg px-3 py-2 text-sm text-warn"
    >
      <WifiOff size={16} strokeWidth={2} aria-hidden className="shrink-0" />
      <span>{t.realtime.lost}</span>
    </div>
  )
}

/** Dải lỗi khi lưu vị trí thẻ thất bại: nói chuyện gì xảy ra + làm gì tiếp (design-system §7). */
export function SaveErrorBanner({
  message,
  onDismiss,
}: {
  message: string
  onDismiss: () => void
}) {
  return (
    <div
      role="alert"
      className="flex items-center gap-2 rounded-md border border-bad-border bg-bad-bg px-3 py-2 text-sm text-bad"
    >
      <CircleAlert size={16} strokeWidth={2} aria-hidden className="shrink-0" />
      <span className="flex-1">{message}</span>
      <button
        type="button"
        onClick={onDismiss}
        aria-label={t.move.dismiss}
        className="inline-flex size-5 items-center justify-center rounded-sm hover:bg-bad-border/50"
      >
        <X size={14} strokeWidth={2} aria-hidden />
      </button>
    </div>
  )
}
