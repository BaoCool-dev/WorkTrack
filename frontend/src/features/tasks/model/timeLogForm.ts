import { messages } from '@/shared/lib/messages'
import { TIME_LOG_MINUTES_MAX, TIME_LOG_NOTE_MAX } from './types'

const t = messages.tasks.timeLog

/** Ngày hôm nay theo giờ máy người dùng, dạng giá trị ô date "YYYY-MM-DD" (không phải để hiển thị). */
export function todayValue(now: Date = new Date()): string {
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  return `${String(now.getFullYear())}-${m}-${d}`
}

export type HoursResult = { minutes: number } | { error: string }

/**
 * Đổi chuỗi số giờ người dùng gõ ("1,5" hoặc "1.5") thành phút nguyên.
 * Vì sao làm tròn tới phút: backend lưu phút (1..1440), so sánh và cộng dồn đều theo phút.
 */
export function parseHours(raw: string): HoursResult {
  const value = raw.trim().replace(',', '.')
  if (value === '') return { error: t.hoursInvalid }
  if (!/^-?\d+(\.\d+)?$/.test(value)) return { error: t.hoursInvalid }
  const minutes = Math.round(Number(value) * 60)
  // design-system §7: 0 hoặc âm → "Không được log số giờ âm."
  if (minutes <= 0) return { error: t.negative }
  if (minutes > TIME_LOG_MINUTES_MAX) return { error: t.tooMany }
  return { minutes }
}

export interface TimeLogFormErrors {
  hours?: string
  note?: string
  workDate?: string
}

export function validateTimeLog(
  hours: string,
  note: string,
  workDate: string,
  today: string = todayValue(),
): { errors: TimeLogFormErrors; minutes?: number } {
  const errors: TimeLogFormErrors = {}
  const parsed = parseHours(hours)
  if ('error' in parsed) errors.hours = parsed.error
  const trimmed = note.trim()
  if (trimmed === '') errors.note = t.noteRequired
  // Đếm theo ký tự Unicode như backend (utf8.RuneCountInString).
  else if (Array.from(trimmed).length > TIME_LOG_NOTE_MAX) errors.note = t.noteTooLong
  if (workDate === '') errors.workDate = t.dateRequired
  // So chuỗi YYYY-MM-DD đúng thứ tự từ điển = thứ tự ngày.
  else if (workDate > today) errors.workDate = t.dateFuture
  return { errors, minutes: 'minutes' in parsed ? parsed.minutes : undefined }
}
