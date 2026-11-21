import { describe, expect, it } from 'vitest'
import { getHoursLevel } from './hoursLevel'

describe('getHoursLevel', () => {
  it('trong định mức khi thực tế ≤ dự kiến', () => {
    expect(getHoursLevel(0, 16)).toBe('ok')
    expect(getHoursLevel(16, 16)).toBe('ok')
  })

  it('vượt dưới 20% là warn, từ 20% trở lên là bad (design-system §10, tạm)', () => {
    expect(getHoursLevel(19, 16)).toBe('warn') // +18,75%
    expect(getHoursLevel(19.2, 16)).toBe('bad') // đúng +20%
    expect(getHoursLevel(30, 16)).toBe('bad')
  })

  it('chưa có dự kiến: có giờ thực tế coi như vượt nặng, chưa có gì là ok', () => {
    expect(getHoursLevel(0, 0)).toBe('ok')
    expect(getHoursLevel(2, 0)).toBe('bad')
  })
})
