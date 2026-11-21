import { describe, expect, it } from 'vitest'

import { makeBoard } from '@/features/projects/test-data'
import { makeTask } from '@/features/tasks/test-data'

import { boardRoleOf, canDragTask, checkDrop, type DropInput } from './dropRules'

const board = makeBoard()
function col(i: number) {
  const c = board.columns[i]
  if (!c) throw new Error('thiếu cột')
  return c
}
const backlog = col(0)
const doing = col(1)
const done = col(2)
const manager = { id: 'u-manager', role: 'manager' as const }
const member = { id: 'u-2', role: 'member' as const }

function input(over: Partial<DropInput> = {}): DropInput {
  return {
    task: makeTask({ assignee: { id: 'u-2', name: 'Đỗ Thu Hà' } }),
    from: backlog,
    to: doing,
    transitions: board.transitions,
    boardRole: 'member',
    viewerId: member.id,
    wipCount: 0,
    wipLimit: 5,
    ...over,
  }
}

describe('boardRoleOf', () => {
  it('Manager phụ trách dùng luật manager, người khác dùng luật member', () => {
    expect(boardRoleOf(manager, 'u-manager')).toBe('manager')
    expect(boardRoleOf({ id: 'u-x', role: 'manager' }, 'u-manager')).toBe('member')
    expect(boardRoleOf(member, 'u-manager')).toBe('member')
  })
  it('CEO và Admin không kéo thả', () => {
    expect(boardRoleOf({ id: 'u-ceo', role: 'ceo' }, 'u-manager')).toBeNull()
    expect(boardRoleOf({ id: 'u-admin', role: 'admin' }, 'u-manager')).toBeNull()
  })
})

describe('canDragTask', () => {
  const mine = makeTask({ assignee: { id: 'u-2', name: 'Đỗ Thu Hà' } })
  const other = makeTask({ assignee: { id: 'u-9', name: 'Khác' } })
  it('Manager phụ trách kéo mọi thẻ; Member chỉ kéo thẻ của mình', () => {
    expect(canDragTask('manager', manager.id, other)).toBe(true)
    expect(canDragTask('member', member.id, mine)).toBe(true)
    expect(canDragTask('member', member.id, other)).toBe(false)
    expect(canDragTask('member', member.id, makeTask({ assignee: null }))).toBe(false)
    expect(canDragTask(null, 'u-ceo', mine)).toBe(false)
  })
})

describe('checkDrop', () => {
  it('cùng cột luôn hợp lệ (đổi thứ tự)', () => {
    expect(checkDrop(input({ to: backlog, wipCount: 5 }))).toEqual({ ok: true })
  })
  it('Member theo luật member: Backlog → In Progress hợp lệ', () => {
    expect(checkDrop(input())).toEqual({ ok: true })
  })
  it('Member không chuyển vào/ra nhóm hoàn thành', () => {
    const r = checkDrop(input({ from: doing, to: done }))
    expect(r).toMatchObject({ ok: false, code: 'done_group' })
    expect(r.ok ? '' : r.reason).toContain('Quản lý dự án')
    expect(checkDrop(input({ from: done, to: doing }))).toMatchObject({
      ok: false,
      code: 'done_group',
    })
  })
  it('không có luật chuyển cột → chặn kèm lý do', () => {
    const r = checkDrop(
      input({ boardRole: 'manager', viewerId: manager.id, from: doing, to: backlog }),
    )
    expect(r).toMatchObject({ ok: false, code: 'no_rule' })
    expect(r.ok ? '' : r.reason).toContain('In Progress')
  })
  it('Manager chuyển sang Done theo luật manager', () => {
    expect(
      checkDrop(input({ boardRole: 'manager', viewerId: manager.id, from: doing, to: done })),
    ).toEqual({
      ok: true,
    })
  })
  it('đủ WIP → chặn và nêu tên người nhận', () => {
    const r = checkDrop(input({ wipCount: 5 }))
    expect(r).toMatchObject({ ok: false, code: 'wip' })
    expect(r.ok ? '' : r.reason).toBe('Đỗ Thu Hà đã đủ 5 việc đang làm')
  })
  it('WIP chỉ kiểm khi đi vào nhóm đang làm và thẻ có người nhận', () => {
    expect(checkDrop(input({ wipCount: 4 }))).toEqual({ ok: true })
    expect(
      checkDrop(
        input({
          boardRole: 'manager',
          viewerId: manager.id,
          task: makeTask({ assignee: null }),
          wipCount: 9,
        }),
      ),
    ).toEqual({ ok: true })
  })
  it('Member kéo thẻ người khác → chặn', () => {
    const r = checkDrop(input({ task: makeTask({ assignee: { id: 'u-9', name: 'Khác' } }) }))
    expect(r).toMatchObject({ ok: false, code: 'not_assignee' })
  })
})
