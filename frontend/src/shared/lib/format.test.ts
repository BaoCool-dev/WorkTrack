import { formatDate, formatHours, formatMinutesAsHours, formatNumber, formatTimestampDate } from './format'

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

describe('formatHours', () => {
  it('giờ nguyên và thập phân dấu phẩy, tối đa 1 chữ số', () => {
    expect(formatHours(16)).toBe('16h')
    expect(formatHours(14.5)).toBe('14,5h')
    expect(formatHours(1.25)).toBe('1,3h')
    expect(formatHours(0)).toBe('0h')
  })
  it('đổi phút sang giờ', () => {
    expect(formatMinutesAsHours(960)).toBe('16h')
    expect(formatMinutesAsHours(870)).toBe('14,5h')
  })
})
