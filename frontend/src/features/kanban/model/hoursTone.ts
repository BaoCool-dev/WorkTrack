import type { HoursLevel } from '@/shared/lib/hoursLevel'

/** Màu theo mức giờ (đi kèm hình khối khác nhau, không dùng màu một mình). */
export const toneClass: Record<HoursLevel, string> = {
  ok: 'text-ok',
  warn: 'text-warn',
  bad: 'text-bad',
}

export const barClass: Record<HoursLevel, string> = {
  ok: 'bg-ok',
  warn: 'bg-warn',
  bad: 'bg-bad',
}
