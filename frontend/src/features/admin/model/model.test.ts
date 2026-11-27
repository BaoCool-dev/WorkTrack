import { descendantsOf, flattenDepartments } from './departmentTree'
import { nameSchema, userFormSchema } from './schemas'
import type { Department } from './types'

const base = {
  email: 'an@congty.vn',
  fullName: 'Nguyễn Văn An',
  role: 'member',
  departmentId: '',
  jobTitleId: '',
}

describe('userFormSchema [CF-02]', () => {
  it('hợp lệ → cắt khoảng trắng, "" thành null', () => {
    const r = userFormSchema.parse({ ...base, email: '  an@congty.vn ', departmentId: 'd1' })
    expect(r).toEqual({
      email: 'an@congty.vn',
      fullName: 'Nguyễn Văn An',
      role: 'member',
      departmentId: 'd1',
      jobTitleId: null,
    })
  })

  it.each([
    [{ email: '' }, 'Vui lòng nhập email.'],
    [{ email: 'khong-phai-email' }, 'Email chưa đúng định dạng.'],
    [{ fullName: '   ' }, 'Vui lòng nhập họ tên.'],
    [{ fullName: 'Ặ'.repeat(101) }, 'Họ tên tối đa 100 ký tự.'],
  ])('%o → %s', (patch, msg) => {
    const r = userFormSchema.safeParse({ ...base, ...patch })
    expect(r.success).toBe(false)
    expect(r.error?.issues[0]?.message).toBe(msg)
  })

  it('họ tên đúng 100 ký tự có dấu vẫn hợp lệ (đếm ký tự, không đếm byte)', () => {
    expect(userFormSchema.safeParse({ ...base, fullName: 'Ặ'.repeat(100) }).success).toBe(true)
  })

  it('vai trò lạ bị từ chối', () => {
    expect(userFormSchema.safeParse({ ...base, role: 'root' }).success).toBe(false)
  })
})

describe('nameSchema', () => {
  it('tên rỗng hoặc quá 200 ký tự bị từ chối', () => {
    expect(nameSchema.safeParse({ name: ' ' }).error?.issues[0]?.message).toBe('Vui lòng nhập tên.')
    expect(nameSchema.safeParse({ name: 'a'.repeat(201) }).error?.issues[0]?.message).toBe(
      'Tên tối đa 200 ký tự.',
    )
    expect(nameSchema.parse({ name: ' Kỹ thuật ', parentId: '' })).toEqual({
      name: 'Kỹ thuật',
      parentId: null,
    })
  })
})

const d = (id: string, name: string, parentId: string | null = null): Department => ({
  id,
  name,
  parentId,
  memberCount: 0,
})

describe('cây phòng ban [CF-02]', () => {
  const items = [
    d('be', 'Backend', 'kt'),
    d('kt', 'Kỹ thuật'),
    d('fe', 'Frontend', 'kt'),
    d('kd', 'Kinh doanh'),
    d('go', 'Nhóm Go', 'be'),
  ]

  it('cha trước con, anh em theo tên, có độ sâu', () => {
    expect(flattenDepartments(items).map((r) => `${'-'.repeat(r.depth)}${r.name}`)).toEqual([
      'Kinh doanh',
      'Kỹ thuật',
      '-Backend',
      '--Nhóm Go',
      '-Frontend',
    ])
  })

  it('cha không tồn tại → hiện ở gốc, không mất', () => {
    expect(flattenDepartments([d('x', 'Mồ côi', 'khong-co')])).toEqual([
      { ...d('x', 'Mồ côi', 'khong-co'), depth: 0 },
    ])
  })

  it('con cháu gồm chính nó', () => {
    expect([...descendantsOf(items, 'kt')].sort()).toEqual(['be', 'fe', 'go', 'kt'])
    expect([...descendantsOf(items, 'fe')]).toEqual(['fe'])
  })
})
