import { formatDate, formatNumber, formatTimestampDate } from './format'

describe('formatNumber', () => {
  it('dùng dấu chấm phân cách hàng nghìn kiểu vi-VN', () => {
    expect(formatNumber(5480)).toBe('5.480')
    expect(formatNumber(0)).toBe('0')
  })
})

describe('formatDate', () => {
  it('đổi YYYY-MM-DD sang dd/mm/yyyy, không lệch múi giờ', () => {
    expect(formatDate('2026-12-01')).toBe('01/12/2026')
    expect(formatDate('2027-01-31')).toBe('31/01/2027')
  })
  it('trả null khi trống hoặc sai định dạng', () => {
    expect(formatDate(null)).toBeNull()
    expect(formatDate('')).toBeNull()
    expect(formatDate('01/12/2026')).toBeNull()
  })
})

describe('formatTimestampDate', () => {
  it('định dạng mốc thời gian ISO theo kiểu dd/mm/yyyy', () => {
    expect(formatTimestampDate('2026-11-20T05:00:00Z')).toMatch(/^\d{2}\/\d{2}\/\d{4}$/)
  })
})
