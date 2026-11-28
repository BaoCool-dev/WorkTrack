import { cn } from '@/shared/lib/cn'
import type { HoursLevel } from '@/shared/lib/hoursLevel'
import { messages } from '@/shared/lib/messages'

import { toneClass } from './hoursTone'

const t = messages.kanban.hoursLevel

const LEVELS: readonly HoursLevel[] = ['ok', 'warn', 'bad']

/**
 * Ký hiệu mức giờ — hình khối khác nhau cho từng mức (vuông/tròn/tam giác) để người mù màu
 * vẫn phân biệt được, không chỉ dựa vào màu (design-system §6).
 */
export function HoursLevelMark({
  level,
  labelled = false,
}: {
  level: HoursLevel
  labelled?: boolean
}) {
  const shape =
    level === 'ok' ? (
      <rect x="0.5" y="0.5" width="8" height="8" />
    ) : level === 'warn' ? (
      <circle cx="4.5" cy="4.5" r="4" />
    ) : (
      <polygon points="4.5,0.5 8.5,8.5 0.5,8.5" />
    )
  return (
    <svg
      viewBox="0 0 9 9"
      className={cn('size-2.25 shrink-0 fill-current', toneClass[level])}
      role={labelled ? 'img' : undefined}
      aria-label={labelled ? t[level] : undefined}
      aria-hidden={labelled ? undefined : true}
    >
      {shape}
    </svg>
  )
}

/** Chú giải ba mức trên thanh công cụ Kanban. */
export function HoursLegend() {
  return (
    <div className="flex items-center gap-3 text-sm text-ink-muted" aria-label={t.legend}>
      {LEVELS.map((l) => (
        <span key={l} className="inline-flex items-center gap-1">
          <HoursLevelMark level={l} />
          {t[l]}
        </span>
      ))}
    </div>
  )
}
