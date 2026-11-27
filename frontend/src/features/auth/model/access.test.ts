import { makeUser } from '@/test/fixtures'
import { canAccess, safeNext } from './access'

describe('canAccess', () => {
  it('chưa đăng nhập → không truy cập được gì', () => {
    expect(canAccess(null)).toBe(false)
  })

  it.each([
    ['admin', { permission: 'users.manage' }, true],
    ['manager', { permission: 'users.manage' }, false],
    ['ceo', { permission: 'dashboard.company.view' }, true],
    ['manager', { permission: 'dashboard.company.view' }, false],
    ['member', { roles: ['manager', 'member'] }, true],
    ['admin', { roles: ['manager', 'member'] }, false],
  ] as const)('%s + %j → %s', (role, rule, want) => {
    expect(canAccess(makeUser(role), rule)).toBe(want)
  })
})

describe('safeNext', () => {
  it.each([
    [null, '/'],
    ['/projects?tab=1', '/projects?tab=1'],
    ['//evil.com', '/'],
    ['/\\evil.com', '/'],
    ['https://evil.com', '/'],
    ['/login', '/'],
  ])('%s → %s', (input, want) => {
    expect(safeNext(input)).toBe(want)
  })
})
