const numberFormat = new Intl.NumberFormat('vi-VN')

/** 5480 → "5.480" */
export function formatNumber(n: number): string {
  return numberFormat.format(n)
}

const dateFormat = new Intl.DateTimeFormat('vi-VN', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  timeZone: 'UTC',
})
const DAY_RE = /^(\d{4})-(\d{2})-(\d{2})$/

/** Ngày lịch "2026-12-01" → "01/12/2026"; null/sai định dạng → null. Tính theo UTC để không lệch múi giờ. */
export function formatDate(day: string | null): string | null {
  const m = day ? DAY_RE.exec(day) : null
  if (!m) return null
  return dateFormat.format(new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]))))
}

const localDateFormat = new Intl.DateTimeFormat('vi-VN', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
})

/** Mốc thời gian ISO → ngày theo giờ máy người dùng, "dd/mm/yyyy". */
export function formatTimestampDate(iso: string): string {
  return localDateFormat.format(new Date(iso))
}

const hoursFormat = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 1 })

/** Số giờ → "16h", "14,5h" (tối đa 1 chữ số thập phân). */
export function formatHours(h: number): string {
  return `${hoursFormat.format(h)}h`
}

/** Phút → giờ hiển thị: 960 → "16h", 870 → "14,5h". */
export function formatMinutesAsHours(minutes: number): string {
  return formatHours(minutes / 60)
}
