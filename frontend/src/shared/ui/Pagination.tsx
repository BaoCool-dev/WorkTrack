import { ChevronLeft, ChevronRight } from 'lucide-react'
import { formatNumber } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button } from './Button'

const t = messages.pagination

export interface PaginationProps {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
}

/** B9: hiển thị "1–20 / 57" và nút trang trước/sau. Ẩn khi chỉ có một trang. */
export function Pagination({ page, pageSize, total, onChange }: PaginationProps) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  if (total <= pageSize && page === 1) return null
  const from = total === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(total, page * pageSize)
  return (
    <nav
      aria-label={t.label}
      className="flex items-center justify-end gap-3 text-sm text-ink-muted"
    >
      <span className="num">
        {t.info(formatNumber(from), formatNumber(to), formatNumber(total))}
      </span>
      <span className="num">{t.page(formatNumber(page), formatNumber(pages))}</span>
      <Button
        variant="secondary"
        aria-label={t.prev}
        icon={<ChevronLeft size={16} strokeWidth={2} aria-hidden />}
        disabled={page <= 1}
        onClick={() => {
          onChange(page - 1)
        }}
      />
      <Button
        variant="secondary"
        aria-label={t.next}
        icon={<ChevronRight size={16} strokeWidth={2} aria-hidden />}
        disabled={page >= pages}
        onClick={() => {
          onChange(page + 1)
        }}
      />
    </nav>
  )
}
