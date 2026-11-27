import { formatNumber } from './format'

describe('formatNumber', () => {
  it('dùng dấu chấm phân cách hàng nghìn kiểu vi-VN', () => {
    expect(formatNumber(5480)).toBe('5.480')
    expect(formatNumber(0)).toBe('0')
  })
})
