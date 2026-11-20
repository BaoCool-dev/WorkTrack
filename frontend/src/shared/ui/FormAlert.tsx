import { CircleAlert } from 'lucide-react'

/** Lỗi từ máy chủ hiển thị ngay trên form (không dùng toast cho lỗi cần đọc kỹ — §5). */
export function FormAlert({ message }: { message: string | null }) {
  if (!message) return null
  return (
    <div
      role="alert"
      className="flex items-start gap-2 rounded-md border border-bad-border bg-bad-bg px-3 py-2 text-sm text-bad"
    >
      <CircleAlert size={16} strokeWidth={2} aria-hidden className="mt-px shrink-0" />
      <span>{message}</span>
    </div>
  )
}
