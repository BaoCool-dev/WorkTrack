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

// QA bổ sung — BVA quanh phép làm tròn về phút và biên 1/1440 phút, 500 ký tự Unicode [CF-08].
describe('parseHours — biên làm tròn (QA) [CF-08]', () => {
  it.each([
    ['0,009', 1], // 0,54 phút → làm tròn 1 (biên dưới, phía trong) [TC_LOG_201]
    ['0,0084', 1], // 0,504 phút → 1
    ['23,99', 1439], // 1439,4 phút → 1439
    ['24,005', 1440], // 1440,3 phút → 1440 (biên trên, phía trong) [TC_LOG_202]
    ['  2  ', 120], // khoảng trắng hai đầu
    ['\t1,5\n', 90], // tab/xuống dòng
    ['01,50', 90],
  ])('%j → %i phút [CF-08]', (raw, minutes) => {
    expect(parseHours(raw)).toEqual({ minutes })
  })

  it.each([
    ['0,008', t.negative], // 0,48 phút → làm tròn 0 → bị chặn [TC_LOG_203]
    ['-0', t.negative],
    ['24,01', t.tooMany], // 1440,6 → 1441 [TC_LOG_204]
    ['1000000', t.tooMany],
  ])('%j → lỗi [CF-08]', (raw, error) => {
    expect(parseHours(raw)).toEqual({ error })
  })

  it.each(['1,', ',5', '+1', '1 5', '١', 'Infinity', 'NaN', '0x10', '1.5.'])(
    '%j không hợp lệ [CF-08]',
    (raw) => {
      expect(parseHours(raw)).toEqual({ error: t.hoursInvalid })
    },
  )
})

describe('validateTimeLog — biên ghi chú và ngày (QA) [CF-08]', () => {
  const day = '2026-11-20'
  const today = '2026-11-28'
  it('500 emoji (code point) hợp lệ, 501 thì quá dài [TC_LOG_205]', () => {
    expect(validateTimeLog('1', '😀'.repeat(500), day, today).errors.note).toBeUndefined()
    expect(validateTimeLog('1', '😀'.repeat(501), day, today).errors.note).toBe(t.noteTooLong)
  })

  it('500 ký tự kèm khoảng trắng hai đầu vẫn hợp lệ (đếm sau khi trim) [TC_LOG_206]', () => {
    expect(validateTimeLog('1', `  ${'a'.repeat(500)}\n`, day, today).errors.note).toBeUndefined()
  })

  it('chỉ có tab/xuống dòng coi như rỗng [TC_LOG_207]', () => {
    expect(validateTimeLog('1', '\t\n \r\n', day, today).errors.note).toBe(t.noteRequired)
  })

  it('HTML và tiếng Việt có dấu là ghi chú hợp lệ [TC_LOG_208]', () => {
    expect(validateTimeLog('1', '<b>Sửa lỗi</b> đăng nhập 🚀', day, today).errors).toEqual({})
  })

  it('báo đồng thời mọi lỗi, không trả minutes khi số giờ sai [TC_LOG_209]', () => {
    const r = validateTimeLog('0', '', '2026-11-29', today)
    expect(r.errors).toEqual({ hours: t.negative, note: t.noteRequired, workDate: t.dateFuture })
    expect(r.minutes).toBeUndefined()
  })

  it('ngày qua mốc năm/tháng so đúng thứ tự [TC_LOG_210]', () => {
    expect(validateTimeLog('1', 'x', '2027-01-01', '2026-12-31').errors.workDate).toBe(t.dateFuture)
    expect(validateTimeLog('1', 'x', '2026-12-31', '2027-01-01').errors.workDate).toBeUndefined()
  })
})
