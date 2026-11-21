export type HoursLevel = 'ok' | 'warn' | 'bad'

/** Ngưỡng "vượt nặng": vượt ≥ 1/5 (20%) dự kiến (design-system §10 — tạm `< 20%` vàng, `≥ 20%` đỏ). */
const BAD_DIVISOR = 5

/**
 * Mức so giờ thực tế với dự kiến — hàm DUY NHẤT dùng cho thanh giờ, ký hiệu và chú giải.
 * Chưa có dự kiến (chưa đặt Story Point) mà đã có giờ → không có định mức để so, coi là vượt nặng.
 */
export function getHoursLevel(actual: number, estimated: number): HoursLevel {
  // So bằng phút nguyên: số thực kiểu 19,2 − 16 cho 3,1999… làm lệch đúng ngưỡng 20%.
  const a = Math.round(actual * 60)
  const e = Math.round(estimated * 60)
  if (a <= e) return 'ok'
  if (e <= 0) return 'bad'
  return (a - e) * BAD_DIVISOR < e ? 'warn' : 'bad'
}
