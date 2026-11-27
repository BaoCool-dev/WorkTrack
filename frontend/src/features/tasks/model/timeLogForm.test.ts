import { messages } from '@/shared/lib/messages'
import { parseHours, todayValue, validateTimeLog } from './timeLogForm'

const t = messages.tasks.timeLog

describe('parseHours (CF-08)', () => {
  it.each([
    ['1,5', 90],
    ['1.5', 90],
    ['2', 120],
    ['0,02', 1],
    ['24', 1440],
  ])('%s → %i phút', (raw, minutes) => {
    expect(parseHours(raw)).toEqual({ minutes })
  })

  it.each(['0', '-2', '0,001'])('%s → lỗi số giờ âm', (raw) => {
    expect(parseHours(raw)).toEqual({ error: t.negative })
  })

  it('vượt 24 giờ một lần log', () => {
    expect(parseHours('24,1')).toEqual({ error: t.tooMany })
  })

  it.each(['', 'abc', '1,5,5', '1e2'])('"%s" không hợp lệ', (raw) => {
    expect(parseHours(raw)).toEqual({ error: t.hoursInvalid })
  })
})

describe('validateTimeLog', () => {
  it('ghi chú bắt buộc, tối đa 500 ký tự Unicode', () => {
    expect(validateTimeLog('1', '  ', '2026-11-20', '2026-11-28').errors.note).toBe(t.noteRequired)
    expect(validateTimeLog('1', 'ầ'.repeat(500), '2026-11-20', '2026-11-28').errors.note).toBe(
      undefined,
    )
    expect(validateTimeLog('1', 'a'.repeat(501), '2026-11-20', '2026-11-28').errors.note).toBe(
      t.noteTooLong,
    )
  })

  it('ngày làm không được ở tương lai', () => {
    expect(validateTimeLog('1', 'x', '2026-11-29', '2026-11-28').errors.workDate).toBe(t.dateFuture)
    expect(validateTimeLog('1', 'x', '2026-11-28', '2026-11-28').errors).toEqual({})
    expect(validateTimeLog('1', 'x', '', '2026-11-28').errors.workDate).toBe(t.dateRequired)
  })

  it('todayValue theo giờ máy', () => {
    expect(todayValue(new Date(2026, 0, 5))).toBe('2026-01-05')
  })
})
