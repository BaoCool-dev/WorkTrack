import { activateSchema, loginSchema, passwordSchema } from './schemas'

const firstError = (r: { success: boolean; error?: { issues: { message: string }[] } }) =>
  r.error?.issues[0]?.message

describe('loginSchema', () => {
  it.each([
    [{ email: '', password: 'x' }, 'Vui lòng nhập email.'],
    [{ email: 'khong-phai-email', password: 'x' }, 'Email chưa đúng định dạng.'],
    [{ email: 'a@b.vn', password: '' }, 'Vui lòng nhập mật khẩu.'],
  ])('%j → %s', (input, msg) => {
    expect(firstError(loginSchema.safeParse(input))).toBe(msg)
  })

  it('cắt khoảng trắng quanh email', () => {
    expect(loginSchema.parse({ email: '  a@b.vn ', password: 'x' }).email).toBe('a@b.vn')
  })
})

describe('passwordSchema (khớp quy tắc backend)', () => {
  it.each([
    ['abc123', 'Mật khẩu cần tối thiểu 8 ký tự.'],
    ['abcdefgh', 'Mật khẩu cần có cả chữ và số.'],
    ['12345678', 'Mật khẩu cần có cả chữ và số.'],
    ['ạ1'.repeat(25), 'Mật khẩu tối đa 72 byte.'],
  ])('%s → %s', (pw, msg) => {
    expect(firstError(passwordSchema.safeParse(pw))).toBe(msg)
  })

  it('chấp nhận mật khẩu có dấu tiếng Việt: 8 rune dù nhiều byte', () => {
    expect(passwordSchema.safeParse('mậtkhẩu1').success).toBe(true)
  })
})

describe('activateSchema', () => {
  it('hai mật khẩu không khớp → lỗi ở ô nhập lại', () => {
    const r = activateSchema.safeParse({ password: 'Matkhau123', confirm: 'Matkhau124' })
    expect(r.error?.issues[0]).toMatchObject({ path: ['confirm'], message: 'Hai mật khẩu chưa khớp.' })
  })
})
